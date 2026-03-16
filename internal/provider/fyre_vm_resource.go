package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/fyreclient"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VMResource{}
var _ resource.ResourceWithImportState = &VMResource{}

func NewVMResource() resource.Resource {
	return &VMResource{}
}

// VMResource defines the resource implementation.
type VMResource struct {
	client *fyreclient.Client
}

// VMResourceModel describes the resource data model.
type VMResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ClusterName     types.String `tfsdk:"cluster_name"`
	ProductGroupID  types.String `tfsdk:"product_group_id"`
	MasterNodes     []NodeModel  `tfsdk:"master_nodes"`
	WorkerNodes     []NodeModel  `tfsdk:"worker_nodes"`
	ClusterConfig   types.Object `tfsdk:"cluster_config"`
	NetworkConfig   types.Object `tfsdk:"network_config"`
	PollingTimeout  types.Int64  `tfsdk:"polling_timeout_minutes"`
	PollingInterval types.Int64  `tfsdk:"polling_interval_seconds"`
	RequestID       types.String `tfsdk:"request_id"`
	ClusterHostName types.String `tfsdk:"cluster_host_name"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	DeleteTimeout   types.String `tfsdk:"delete_timeout"`
}

// NodeModel describes a node configuration
type NodeModel struct {
	Name               types.String `tfsdk:"name"`
	Count              types.Int64  `tfsdk:"count"`
	CPU                types.Int64  `tfsdk:"cpu"`
	Memory             types.Int64  `tfsdk:"memory"`
	OS                 types.String `tfsdk:"os"`
	AdditionalDiskSize types.Int64  `tfsdk:"additional_disk_size"`
}

// ClusterConfigModel describes cluster configuration
type ClusterConfigModel struct {
	Platform types.String `tfsdk:"platform"`
}

// NetworkConfigModel describes network configuration
type NetworkConfigModel struct {
	PublicVLAN  types.String `tfsdk:"public_vlan"`
	PrivateVLAN types.String `tfsdk:"private_vlan"`
	DNS         types.String `tfsdk:"dns"`
}

func (r *VMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fyre_vm"
}

func (r *VMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Fyre VM cluster with master and worker nodes.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier (cluster name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the Fyre cluster",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product_group_id": schema.StringAttribute{
				MarkdownDescription: "Product group ID for quota management. If not specified, uses provider configuration.",
				Optional:            true,
			},
			"master_nodes": schema.ListNestedAttribute{
				MarkdownDescription: "List of master node configurations",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the master node",
							Required:            true,
						},
						"count": schema.Int64Attribute{
							MarkdownDescription: "Number of master nodes",
							Required:            true,
						},
						"cpu": schema.Int64Attribute{
							MarkdownDescription: "Number of CPUs",
							Required:            true,
						},
						"memory": schema.Int64Attribute{
							MarkdownDescription: "Memory in GB",
							Required:            true,
						},
						"os": schema.StringAttribute{
							MarkdownDescription: "Operating system",
							Required:            true,
						},
						"additional_disk_size": schema.Int64Attribute{
							MarkdownDescription: "Additional disk size in GB",
							Required:            true,
						},
					},
				},
			},
			"worker_nodes": schema.ListNestedAttribute{
				MarkdownDescription: "List of worker node configurations",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the worker node",
							Required:            true,
						},
						"count": schema.Int64Attribute{
							MarkdownDescription: "Number of worker nodes",
							Required:            true,
						},
						"cpu": schema.Int64Attribute{
							MarkdownDescription: "Number of CPUs",
							Required:            true,
						},
						"memory": schema.Int64Attribute{
							MarkdownDescription: "Memory in GB",
							Required:            true,
						},
						"os": schema.StringAttribute{
							MarkdownDescription: "Operating system",
							Required:            true,
						},
						"additional_disk_size": schema.Int64Attribute{
							MarkdownDescription: "Additional disk size in GB",
							Required:            true,
						},
					},
				},
			},
			"cluster_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Cluster configuration",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"platform": schema.StringAttribute{
						MarkdownDescription: "Platform type (e.g., 'x', 'z')",
						Required:            true,
					},
				},
			},
			"network_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Network configuration",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"public_vlan": schema.StringAttribute{
						MarkdownDescription: "Public VLAN",
						Required:            true,
					},
					"private_vlan": schema.StringAttribute{
						MarkdownDescription: "Private VLAN (for standard fyre)",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
					"dns": schema.StringAttribute{
						MarkdownDescription: "DNS server (for beta-fyre)",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
				},
			},
			"polling_timeout_minutes": schema.Int64Attribute{
				MarkdownDescription: "Timeout in minutes for VM creation polling",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},
			"polling_interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "Interval in seconds between polling attempts",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
			},
			"request_id": schema.StringAttribute{
				MarkdownDescription: "Request ID from Fyre API",
				Computed:            true,
			},
			"cluster_host_name": schema.StringAttribute{
				MarkdownDescription: "Fully qualified hostname of the cluster master",
				Computed:            true,
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

func (r *VMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating Fyre VM", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})

	// Extract cluster config
	var clusterConfig ClusterConfigModel
	resp.Diagnostics.Append(data.ClusterConfig.As(ctx, &clusterConfig, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract network config
	var networkConfig NetworkConfigModel
	resp.Diagnostics.Append(data.NetworkConfig.As(ctx, &networkConfig, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build master nodes
	masterNodes := []fyreclient.NodeConfig{}
	for _, node := range data.MasterNodes {
		masterNodes = append(masterNodes, fyreclient.NodeConfig{
			Name:               node.Name.ValueString(),
			Count:              int(node.Count.ValueInt64()),
			CPU:                int(node.CPU.ValueInt64()),
			Memory:             int(node.Memory.ValueInt64()),
			OS:                 node.OS.ValueString(),
			AdditionalDiskSize: int(node.AdditionalDiskSize.ValueInt64()),
		})
	}

	// Build worker nodes
	workerNodes := []fyreclient.NodeConfig{}
	for _, node := range data.WorkerNodes {
		workerNodes = append(workerNodes, fyreclient.NodeConfig{
			Name:               node.Name.ValueString(),
			Count:              int(node.Count.ValueInt64()),
			CPU:                int(node.CPU.ValueInt64()),
			Memory:             int(node.Memory.ValueInt64()),
			OS:                 node.OS.ValueString(),
			AdditionalDiskSize: int(node.AdditionalDiskSize.ValueInt64()),
		})
	}

	// Use product_group_id from resource or fall back to provider config
	productGroupID := data.ProductGroupID.ValueString()
	if productGroupID == "" {
		productGroupID = r.client.ProductGroupID
	}

	// Temporarily update client's product group ID for this request
	originalPGID := r.client.ProductGroupID
	r.client.ProductGroupID = productGroupID
	defer func() { r.client.ProductGroupID = originalPGID }()

	// Create VM request
	createReq := fyreclient.CreateVMRequest{
		ClusterName: data.ClusterName.ValueString(),
		MasterNodes: masterNodes,
		WorkerNodes: workerNodes,
		ClusterConfig: fyreclient.ClusterConfig{
			Platform: clusterConfig.Platform.ValueString(),
		},
		NetworkConfig: fyreclient.NetworkConfig{
			PublicVLAN:  networkConfig.PublicVLAN.ValueString(),
			PrivateVLAN: networkConfig.PrivateVLAN.ValueString(),
			DNS:         networkConfig.DNS.ValueString(),
		},
	}

	// Create the VM
	createResp, err := r.client.CreateVM(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create VM", err.Error())
		return
	}

	tflog.Info(ctx, "VM creation initiated", map[string]interface{}{
		"request_id": createResp.RequestID,
		"status":     createResp.Status,
	})

	// Wait for VM to be ready
	timeoutMinutes := int(data.PollingTimeout.ValueInt64())
	var requestIDOrURL string
	if r.client.ClusterType == "beta-fyre" {
		requestIDOrURL = createResp.RequestID
	} else {
		requestIDOrURL = createResp.Details
	}

	err = r.client.WaitForVMReady(requestIDOrURL, timeoutMinutes)
	if err != nil {
		resp.Diagnostics.AddError("VM creation failed or timed out", err.Error())
		return
	}

	// Set computed values
	data.ID = data.ClusterName
	data.RequestID = types.StringValue(createResp.RequestID)
	data.ClusterHostName = types.StringValue(fmt.Sprintf("%s-master1.%s",
		data.ClusterName.ValueString(),
		r.client.GetDomainSuffix()))
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "VM created successfully", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
		"request_id":   data.RequestID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Fyre VM", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})

	// Check if the VM still exists
	// beta-fyre API (ocpapi.svl.ibm.com) requires a valid id, IP, or FQDN — not a cluster name
	lookupIdentifier := data.ClusterName.ValueString()
	if r.client.ClusterType == "beta-fyre" && !data.ClusterHostName.IsNull() && data.ClusterHostName.ValueString() != "" {
		lookupIdentifier = data.ClusterHostName.ValueString()
	}
	info, err := r.client.GetVMInfo(lookupIdentifier)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Fyre VM", err.Error())
		return
	}
	if info == nil {
		// VM no longer exists, remove from state
		tflog.Info(ctx, "Fyre VM no longer exists, removing from state", map[string]interface{}{
			"cluster_name": data.ClusterName.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Most Fyre VM attributes require replacement (handled by RequiresReplace plan modifiers)
	// Update is mainly for metadata changes
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Updating Fyre VM metadata", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMResourceModel

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

	tflog.Info(ctx, "Deleting Fyre VM", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})

	deleteIdentifier := data.ClusterName.ValueString()
	if r.client.ClusterType == "beta-fyre" {
		deleteIdentifier = data.ClusterHostName.ValueString()
	}
	err = r.client.DeleteVM(deleteIdentifier)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete VM", err.Error())
		return
	}

	// Verify the VM has been deleted
	info, err := r.client.GetVMInfo(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning("Could not verify VM deletion", err.Error())
	} else if info != nil {
		resp.Diagnostics.AddWarning("VM may still exist after delete request",
			fmt.Sprintf("VM %s still returned data. It may take time to fully delete.", data.ClusterName.ValueString()))
	}

	tflog.Info(ctx, "VM deleted successfully", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})
}

func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
