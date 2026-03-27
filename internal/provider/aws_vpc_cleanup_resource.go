// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AWSVPCCleanupResource{}

// AWSVPCCleanupResource cleans up AWS resources orphaned inside a VPC by
// Kubernetes (load balancers, NAT Gateways, security groups, ENIs) that are
// not tracked by Terraform and would otherwise block VPC deletion.
//
// Create/Read are no-ops that simply persist the VPC ID in state.
// Delete performs the actual cleanup so that `terraform destroy` can proceed.
type AWSVPCCleanupResource struct{}

// AWSVPCCleanupResourceModel is the state schema for this resource.
type AWSVPCCleanupResourceModel struct {
	ID              types.String `tfsdk:"id"`
	VPCID           types.String `tfsdk:"vpc_id"`
	Region          types.String `tfsdk:"region"`
	Profile         types.String `tfsdk:"profile"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func NewAWSVPCCleanupResource() resource.Resource {
	return &AWSVPCCleanupResource{}
}

func (r *AWSVPCCleanupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_vpc_cleanup"
}

func (r *AWSVPCCleanupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Cleans up AWS resources orphaned in a VPC by Kubernetes (ALB/NLB load balancers, Classic ELBs, NAT Gateways, security groups, ENIs) that are not tracked by Terraform. On `terraform destroy`, removes these resources so that VPC/subnet/IGW deletion succeeds without manual intervention.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the VPC to clean up orphaned resources in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "AWS region where the VPC resides.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "AWS named profile. Used when `access_key_id`/`secret_access_key` are not set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key_id": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "AWS access key ID. Takes precedence over `profile` and the standard SDK credential chain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "AWS secret access key. Required when `access_key_id` is set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create stores the VPC ID in state; no AWS resources are created.
func (r *AWSVPCCleanupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AWSVPCCleanupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = data.VPCID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read is a no-op; the state is the source of truth for cleanup parameters.
func (r *AWSVPCCleanupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AWSVPCCleanupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is never called because all attributes have RequiresReplace.
func (r *AWSVPCCleanupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AWSVPCCleanupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete performs the VPC cleanup: removes load balancers, NAT Gateways,
// security groups, and ENIs that Kubernetes created outside of Terraform state.
func (r *AWSVPCCleanupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AWSVPCCleanupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcID := data.VPCID.ValueString()
	region := data.Region.ValueString()

	tflog.Info(ctx, "Starting VPC cleanup", map[string]interface{}{
		"vpc_id": vpcID,
		"region": region,
	})

	awsCfg, err := buildVPCCleanupAWSConfig(
		ctx, region,
		data.Profile.ValueString(),
		data.AccessKeyID.ValueString(),
		data.SecretAccessKey.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("AWS Config Error",
			fmt.Sprintf("Failed to build AWS config for VPC cleanup: %v", err))
		return
	}

	var errs []string

	if err := vpcCleanupELBv2(ctx, awsCfg, vpcID); err != nil {
		errs = append(errs, fmt.Sprintf("ELBv2 (ALB/NLB): %v", err))
	}
	if err := vpcCleanupClassicELB(ctx, awsCfg, vpcID); err != nil {
		errs = append(errs, fmt.Sprintf("Classic ELB: %v", err))
	}
	if err := vpcCleanupNATGateways(ctx, awsCfg, vpcID); err != nil {
		errs = append(errs, fmt.Sprintf("NAT Gateways: %v", err))
	}
	if err := vpcCleanupENIs(ctx, awsCfg, vpcID); err != nil {
		errs = append(errs, fmt.Sprintf("ENIs: %v", err))
	}
	if err := vpcCleanupSecurityGroups(ctx, awsCfg, vpcID); err != nil {
		errs = append(errs, fmt.Sprintf("Security Groups: %v", err))
	}

	if len(errs) > 0 {
		resp.Diagnostics.AddError(
			"VPC Cleanup Failed",
			fmt.Sprintf("One or more cleanup steps failed for VPC %s:\n  - %s",
				vpcID, strings.Join(errs, "\n  - ")),
		)
		return
	}

	tflog.Info(ctx, "VPC cleanup complete", map[string]interface{}{"vpc_id": vpcID})
}

// buildVPCCleanupAWSConfig builds an AWS config using the same credential
// priority order as the vpc-cleanup standalone binary:
//  1. Explicit access key + secret (static credentials)
//  2. Named profile
//  3. Standard SDK chain (env vars → shared credentials file → instance profile)
func buildVPCCleanupAWSConfig(ctx context.Context, region, profile, accessKeyID, secretAccessKey string) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(region))

	switch {
	case accessKeyID != "" && secretAccessKey != "":
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	case profile != "":
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}

// vpcCleanupELBv2 deletes all ALB/NLB load balancers in the VPC and waits
// until they are fully removed (up to ~4 minutes) before returning.
func vpcCleanupELBv2(ctx context.Context, cfg aws.Config, vpcID string) error {
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	arns, err := vpcListELBv2(ctx, client, vpcID)
	if err != nil {
		return err
	}
	if len(arns) == 0 {
		return nil
	}

	for _, arn := range arns {
		if _, err := client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
			LoadBalancerArn: aws.String(arn),
		}); err != nil {
			// Log and continue — partial deletion is still progress.
			fmt.Printf("  [ELBv2] WARNING: failed to delete %s: %v\n", arn, err)
		}
	}

	// Poll until all LBs in the VPC are gone (max 24 × 10 s ≈ 4 minutes).
	for attempt := 1; attempt <= 24; attempt++ {
		time.Sleep(10 * time.Second)
		remaining, err := vpcListELBv2(ctx, client, vpcID)
		if err != nil {
			continue
		}
		if len(remaining) == 0 {
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for ALB/NLB deletion in VPC %s", vpcID)
}

func vpcListELBv2(ctx context.Context, client *elasticloadbalancingv2.Client, vpcID string) ([]string, error) {
	var arns []string
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client,
		&elasticloadbalancingv2.DescribeLoadBalancersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			var notFound *elbv2types.LoadBalancerNotFoundException
			if errors.As(err, &notFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("describe ELBv2: %w", err)
		}
		for _, lb := range page.LoadBalancers {
			if aws.ToString(lb.VpcId) == vpcID {
				arns = append(arns, aws.ToString(lb.LoadBalancerArn))
			}
		}
	}
	return arns, nil
}

// vpcCleanupClassicELB deletes all Classic (ELBv1) load balancers in the VPC.
func vpcCleanupClassicELB(ctx context.Context, cfg aws.Config, vpcID string) error {
	client := elasticloadbalancing.NewFromConfig(cfg)

	var names []string
	paginator := elasticloadbalancing.NewDescribeLoadBalancersPaginator(client,
		&elasticloadbalancing.DescribeLoadBalancersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("describe Classic ELB: %w", err)
		}
		for _, lb := range page.LoadBalancerDescriptions {
			if aws.ToString(lb.VPCId) == vpcID {
				names = append(names, aws.ToString(lb.LoadBalancerName))
			}
		}
	}

	if len(names) == 0 {
		return nil
	}

	for _, name := range names {
		if _, err := client.DeleteLoadBalancer(ctx, &elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(name),
		}); err != nil {
			fmt.Printf("  [ELBv1] WARNING: failed to delete %s: %v\n", name, err)
		}
	}

	// Classic ELBs delete quickly; a short pause is sufficient.
	time.Sleep(15 * time.Second)
	return nil
}

// vpcCleanupNATGateways deletes all non-deleted NAT Gateways in the VPC,
// waits for them to reach 'deleted' state, then releases their Elastic IPs.
func vpcCleanupNATGateways(ctx context.Context, cfg aws.Config, vpcID string) error {
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return fmt.Errorf("describe NAT gateways: %w", err)
	}

	var gwIDs []string
	var eipAllocIDs []string
	for _, gw := range out.NatGateways {
		if gw.State == ec2types.NatGatewayStateDeleted || gw.State == ec2types.NatGatewayStateFailed {
			continue
		}
		gwIDs = append(gwIDs, aws.ToString(gw.NatGatewayId))
		for _, addr := range gw.NatGatewayAddresses {
			if addr.AllocationId != nil {
				eipAllocIDs = append(eipAllocIDs, aws.ToString(addr.AllocationId))
			}
		}
	}

	if len(gwIDs) == 0 {
		return nil
	}

	for _, id := range gwIDs {
		if _, err := client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
			NatGatewayId: aws.String(id),
		}); err != nil {
			fmt.Printf("  [NAT] WARNING: failed to delete %s: %v\n", id, err)
		}
	}

	// Poll until all reach 'deleted' state (max 30 × 10 s ≈ 5 minutes).
	for attempt := 1; attempt <= 30; attempt++ {
		time.Sleep(10 * time.Second)
		poll, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: gwIDs,
		})
		if err != nil {
			continue
		}
		allDone := true
		for _, gw := range poll.NatGateways {
			if gw.State != ec2types.NatGatewayStateDeleted && gw.State != ec2types.NatGatewayStateFailed {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		if attempt == 30 {
			return fmt.Errorf("timed out waiting for NAT Gateway deletion in VPC %s", vpcID)
		}
	}

	for _, allocID := range eipAllocIDs {
		if _, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: aws.String(allocID),
		}); err != nil {
			fmt.Printf("  [NAT] WARNING: failed to release EIP %s: %v\n", allocID, err)
		}
	}
	return nil
}

// vpcCleanupENIs deletes orphaned ENIs (status=available) in the VPC.
// These are typically left behind by deleted load balancers.
func vpcCleanupENIs(ctx context.Context, cfg aws.Config, vpcID string) error {
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("status"), Values: []string{"available"}},
		},
	})
	if err != nil {
		return fmt.Errorf("describe ENIs: %w", err)
	}

	var deleteErrs []string
	for _, eni := range out.NetworkInterfaces {
		eniID := aws.ToString(eni.NetworkInterfaceId)
		if _, err := client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: aws.String(eniID),
		}); err != nil {
			deleteErrs = append(deleteErrs, fmt.Sprintf("%s: %v", eniID, err))
		}
	}

	if len(deleteErrs) > 0 {
		return fmt.Errorf("%d ENI(s) could not be deleted: %s", len(deleteErrs), strings.Join(deleteErrs, "; "))
	}
	return nil
}

// vpcCleanupSecurityGroups deletes all non-default security groups in the VPC.
// It first revokes all rules (to break cross-references) then deletes them.
// The default security group is skipped — AWS does not allow deleting it.
func vpcCleanupSecurityGroups(ctx context.Context, cfg aws.Config, vpcID string) error {
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return fmt.Errorf("describe security groups: %w", err)
	}

	var groups []ec2types.SecurityGroup
	for _, sg := range out.SecurityGroups {
		if aws.ToString(sg.GroupName) != "default" {
			groups = append(groups, sg)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	// First pass: revoke all ingress/egress rules to clear cross-references.
	for _, sg := range groups {
		sgID := aws.ToString(sg.GroupId)
		if len(sg.IpPermissions) > 0 {
			if _, err := client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
				GroupId:       aws.String(sgID),
				IpPermissions: sg.IpPermissions,
			}); err != nil {
				fmt.Printf("  [SG] WARNING: failed to revoke ingress for %s: %v\n", sgID, err)
			}
		}
		if len(sg.IpPermissionsEgress) > 0 {
			if _, err := client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
				GroupId:       aws.String(sgID),
				IpPermissions: sg.IpPermissionsEgress,
			}); err != nil {
				fmt.Printf("  [SG] WARNING: failed to revoke egress for %s: %v\n", sgID, err)
			}
		}
	}

	// Second pass: delete the groups.
	var deleteErrs []string
	for _, sg := range groups {
		sgID := aws.ToString(sg.GroupId)
		if _, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		}); err != nil {
			deleteErrs = append(deleteErrs, fmt.Sprintf("%s: %v", sgID, err))
		}
	}

	if len(deleteErrs) > 0 {
		return fmt.Errorf("%d security group(s) could not be deleted: %s", len(deleteErrs), strings.Join(deleteErrs, "; "))
	}
	return nil
}
