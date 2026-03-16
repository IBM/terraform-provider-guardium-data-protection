package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/fyreclient"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OCPResource{}
var _ resource.ResourceWithImportState = &OCPResource{}

func NewOCPResource() resource.Resource {
	return &OCPResource{}
}

// OCPResource defines the resource implementation.
type OCPResource struct {
	client *fyreclient.Client
}

// OCPResourceModel describes the resource data model.
type OCPResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Description       types.String   `tfsdk:"description"`
	Platform          types.String   `tfsdk:"platform"`
	Site              types.String   `tfsdk:"site"`
	QuotaType         types.String   `tfsdk:"quota_type"`
	ProductGroupID    types.String   `tfsdk:"product_group_id"`
	TimeToLive        types.String   `tfsdk:"time_to_live"`
	OCPVersion        types.String   `tfsdk:"ocp_version"`
	FIPS              types.String   `tfsdk:"fips"`
	SSHKey            types.String   `tfsdk:"ssh_key"`
	MasterNodes       []OCPNodeModel `tfsdk:"master"`
	WorkerNodes       []OCPNodeModel `tfsdk:"worker"`
	WaitForCluster    types.Bool     `tfsdk:"wait_for_cluster"`
	PollingTimeout    types.Int64    `tfsdk:"polling_timeout_minutes"`
	PollingInterval   types.Int64    `tfsdk:"polling_interval_seconds"`
	RequestID         types.String   `tfsdk:"request_id"`
	ClusterURL        types.String   `tfsdk:"cluster_url"`
	KubeadminPassword types.String   `tfsdk:"kubeadmin_password"`
	LastUpdated       types.String   `tfsdk:"last_updated"`
	DeleteTimeout     types.String   `tfsdk:"delete_timeout"`
}

// OCPNodeModel describes an OCP node configuration
type OCPNodeModel struct {
	Count          types.Int64   `tfsdk:"count"`
	CPU            types.Int64   `tfsdk:"cpu"`
	Memory         types.Int64   `tfsdk:"memory"`
	AdditionalDisk []types.Int64 `tfsdk:"additional_disk"`
}

func (r *OCPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fyre_ocp"
}

func (r *OCPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an OpenShift Container Platform (OCP) cluster on Fyre.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier (cluster name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the OCP cluster",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the OCP cluster",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Platform type (e.g., 'x', 'p', 'z')",
				Required:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "Site location (e.g., 'svl', 'rtp', 'pok')",
				Required:            true,
			},
			"quota_type": schema.StringAttribute{
				MarkdownDescription: "Quota type (e.g., 'product_group', 'quick_burn')",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("product_group"),
			},
			"product_group_id": schema.StringAttribute{
				MarkdownDescription: "Product group ID for quota management. If not specified, uses provider configuration.",
				Optional:            true,
			},
			"time_to_live": schema.StringAttribute{
				MarkdownDescription: "Time to live in hours for the cluster",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("36"),
			},
			"ocp_version": schema.StringAttribute{
				MarkdownDescription: "OpenShift version (e.g., '4.16.11', '4.18.28')",
				Required:            true,
			},
			"fips": schema.StringAttribute{
				MarkdownDescription: "Enable FIPS mode ('yes' or 'no')",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("yes"),
			},
			"ssh_key": schema.StringAttribute{
				MarkdownDescription: "SSH public key for cluster access",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"master": schema.ListNestedAttribute{
				MarkdownDescription: "List of master node configurations",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"count": schema.Int64Attribute{
							MarkdownDescription: "Number of master nodes",
							Required:            true,
						},
						"cpu": schema.Int64Attribute{
							MarkdownDescription: "Number of CPUs per master node",
							Required:            true,
						},
						"memory": schema.Int64Attribute{
							MarkdownDescription: "Memory in GB per master node",
							Required:            true,
						},
						"additional_disk": schema.ListAttribute{
							MarkdownDescription: "Additional disk sizes in GB",
							Optional:            true,
							ElementType:         types.Int64Type,
						},
					},
				},
			},
			"worker": schema.ListNestedAttribute{
				MarkdownDescription: "List of worker node configurations",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"count": schema.Int64Attribute{
							MarkdownDescription: "Number of worker nodes",
							Required:            true,
						},
						"cpu": schema.Int64Attribute{
							MarkdownDescription: "Number of CPUs per worker node",
							Required:            true,
						},
						"memory": schema.Int64Attribute{
							MarkdownDescription: "Memory in GB per worker node",
							Required:            true,
						},
						"additional_disk": schema.ListAttribute{
							MarkdownDescription: "Additional disk sizes in GB",
							Optional:            true,
							ElementType:         types.Int64Type,
						},
					},
				},
			},
			"wait_for_cluster": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the OCP cluster to be fully ready before completing. Set to false to skip polling.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"polling_timeout_minutes": schema.Int64Attribute{
				MarkdownDescription: "Timeout in minutes for OCP cluster creation polling",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(120),
			},
			"polling_interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "Interval in seconds between polling attempts",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},
			"request_id": schema.StringAttribute{
				MarkdownDescription: "Request ID from Fyre API",
				Computed:            true,
			},
			"cluster_url": schema.StringAttribute{
				MarkdownDescription: "URL to access the OCP cluster",
				Computed:            true,
			},
			"kubeadmin_password": schema.StringAttribute{
				MarkdownDescription: "Kubeadmin password for the OCP cluster (fetched from Fyre API)",
				Computed:            true,
				Sensitive:           true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update",
				Computed:            true,
			},
			"delete_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for the delete operation (e.g. '2h', '90m'). Default: 2h.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("2h"),
			},
		},
	}
}

func (r *OCPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	unifiedClient, ok := req.ProviderData.(*UnifiedClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *UnifiedClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if unifiedClient.FyreClient == nil {
		resp.Diagnostics.AddError(
			"Fyre Client Not Configured.",
			fmt.Sprintf("Expected unifiedClient.FyreClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = unifiedClient.FyreClient
}

func (r *OCPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OCPResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating OCP cluster", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Build master nodes
	masterNodes := []fyreclient.OCPNodeConfig{}
	for _, node := range data.MasterNodes {
		additionalDisks := []int{}
		for _, disk := range node.AdditionalDisk {
			additionalDisks = append(additionalDisks, int(disk.ValueInt64()))
		}
		masterNodes = append(masterNodes, fyreclient.OCPNodeConfig{
			Count:          int(node.Count.ValueInt64()),
			CPU:            int(node.CPU.ValueInt64()),
			Memory:         int(node.Memory.ValueInt64()),
			AdditionalDisk: additionalDisks,
		})
	}

	// Build worker nodes
	workerNodes := []fyreclient.OCPNodeConfig{}
	for _, node := range data.WorkerNodes {
		additionalDisks := []int{}
		for _, disk := range node.AdditionalDisk {
			additionalDisks = append(additionalDisks, int(disk.ValueInt64()))
		}
		workerNodes = append(workerNodes, fyreclient.OCPNodeConfig{
			Count:          int(node.Count.ValueInt64()),
			CPU:            int(node.CPU.ValueInt64()),
			Memory:         int(node.Memory.ValueInt64()),
			AdditionalDisk: additionalDisks,
		})
	}

	// Use product_group_id from resource or fall back to provider config
	productGroupID := data.ProductGroupID.ValueString()
	if productGroupID == "" {
		productGroupID = r.client.ProductGroupID
	}

	// Create OCP request
	createReq := fyreclient.CreateOCPRequest{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		Platform:       data.Platform.ValueString(),
		Site:           data.Site.ValueString(),
		QuotaType:      data.QuotaType.ValueString(),
		ProductGroupID: productGroupID,
		TimeToLive:     data.TimeToLive.ValueString(),
		OCPVersion:     data.OCPVersion.ValueString(),
		FIPS:           data.FIPS.ValueString(),
		SSHKey:         data.SSHKey.ValueString(),
		Master:         masterNodes,
		Worker:         workerNodes,
	}

	// Create the OCP cluster
	createResp, err := r.client.CreateOCP(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create OCP cluster", err.Error())
		return
	}

	tflog.Info(ctx, "OCP cluster creation initiated", map[string]interface{}{
		"request_id": createResp.RequestID,
		"status":     createResp.Status,
	})

	// Wait for OCP cluster to be ready (unless wait_for_cluster is false)
	if data.WaitForCluster.ValueBool() {
		timeoutMinutes := int(data.PollingTimeout.ValueInt64())
		pollIntervalSeconds := int(data.PollingInterval.ValueInt64())
		err = r.client.WaitForOCPReady(data.Name.ValueString(), timeoutMinutes, pollIntervalSeconds)
		if err != nil {
			resp.Diagnostics.AddError("OCP cluster creation failed or timed out", err.Error())
			return
		}
	} else {
		tflog.Info(ctx, "Skipping cluster readiness polling (wait_for_cluster = false)")
	}

	// Set computed values
	data.ID = data.Name
	data.RequestID = types.StringValue(createResp.RequestID)
	data.ClusterURL = types.StringValue(fmt.Sprintf("https://fyre.ibm.com/clusters/%s", data.Name.ValueString()))
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	// Fetch kubeadmin password from cluster details
	details, err := r.client.GetOCPClusterDetails(data.Name.ValueString())
	if err != nil {
		tflog.Warn(ctx, "Failed to fetch kubeadmin password", map[string]interface{}{"error": err.Error()})
		data.KubeadminPassword = types.StringValue("")
	} else if details != nil {
		data.KubeadminPassword = types.StringValue(details.KubeadminPassword)
	} else {
		data.KubeadminPassword = types.StringValue("")
	}

	tflog.Info(ctx, "OCP cluster created successfully", map[string]interface{}{
		"name":       data.Name.ValueString(),
		"request_id": data.RequestID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OCPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OCPResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading OCP cluster", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Check if the OCP cluster still exists
	info, err := r.client.GetOCPInfo(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OCP cluster", err.Error())
		return
	}
	if info == nil {
		// Cluster no longer exists, remove from state
		tflog.Info(ctx, "OCP cluster no longer exists, removing from state", map[string]interface{}{
			"name": data.Name.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	// Refresh kubeadmin password from cluster details
	details, err := r.client.GetOCPClusterDetails(data.Name.ValueString())
	if err != nil {
		tflog.Warn(ctx, "Failed to fetch kubeadmin password during read", map[string]interface{}{"error": err.Error()})
	} else if details != nil {
		data.KubeadminPassword = types.StringValue(details.KubeadminPassword)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OCPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OCPResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Most OCP cluster attributes require replacement (handled by RequiresReplace plan modifiers)
	// Update is mainly for metadata changes
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Updating OCP cluster metadata", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OCPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OCPResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, err := time.ParseDuration(data.DeleteTimeout.ValueString())
	if err != nil {
		timeout = 2 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tflog.Info(ctx, "Deleting OCP cluster", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	err = r.client.DeleteOCP(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete OCP cluster", err.Error())
		return
	}

	// Verify the cluster has been deleted
	info, err := r.client.GetOCPInfo(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning("Could not verify OCP cluster deletion", err.Error())
	} else if info != nil {
		resp.Diagnostics.AddWarning("OCP cluster may still exist after delete request",
			fmt.Sprintf("Cluster %s still returned status: %s. It may take time to fully delete.", data.Name.ValueString(), info.DeployedStatus))
	}

	tflog.Info(ctx, "OCP cluster deleted successfully", map[string]interface{}{
		"name": data.Name.ValueString(),
	})
}

func (r *OCPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
