package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k3sclient"
)

var _ resource.Resource = &K3SClusterResource{}

// var _ resource.ResourceWithModifyPlan = &K3SClusterResource{}
var _ resource.ResourceWithImportState = &K3SClusterResource{}

func NewK3SClusterResource() resource.Resource {
	return &K3SClusterResource{}
}

type K3SClusterResource struct {
	client *k3sclient.Client
}

type K3SClusterResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ClusterName            types.String `tfsdk:"cluster_name"`
	MasterNodes            types.List   `tfsdk:"master_nodes"`
	WorkerNodes            types.List   `tfsdk:"worker_nodes"`
	K3SVersion             types.String `tfsdk:"k3s_version"`
	K3SToken               types.String `tfsdk:"k3s_token"`
	DisableTraefik         types.Bool   `tfsdk:"disable_traefik"`
	TaintMasters           types.Bool   `tfsdk:"taint_masters"`
	NodeWaitTimeout        types.String `tfsdk:"node_wait_timeout"`
	PrimaryMaster          types.String `tfsdk:"primary_master"`
	ClusterType            types.String `tfsdk:"cluster_type"`
	KubeconfigPath         types.String `tfsdk:"kubeconfig_path"`
	LastUpdated            types.String `tfsdk:"last_updated"`
	AirgapInstall          types.Bool   `tfsdk:"airgap_install"`
	AirgapInstallationPath types.String `tfsdk:"airgap_installation_path"`
	DeleteTimeout          types.String `tfsdk:"delete_timeout"`
}

// Metadata implements resource.Resource.
func (k *K3SClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k3s_cluster"

}

// Schema implements resource.Resource.
func (k *K3SClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Installs and manages a K3S cluster on remote nodes via SSH. Supports single-node and multi-node deployments.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier (cluster name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the K3S cluster",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"master_nodes": schema.ListAttribute{
				MarkdownDescription: "List of master node hostnames (FQDNs)",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"worker_nodes": schema.ListAttribute{
				MarkdownDescription: "List of worker node hostnames (FQDNs). Empty for single-node cluster.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"k3s_version": schema.StringAttribute{
				MarkdownDescription: "K3S version to install (e.g., v1.32.3)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("v1.32.3"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"k3s_token": schema.StringAttribute{
				MarkdownDescription: "Token for K3S cluster authentication",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Default:             stringdefault.StaticString("edge1234"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disable_traefik": schema.BoolAttribute{
				MarkdownDescription: "Disable Traefik ingress controller",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"taint_masters": schema.BoolAttribute{
				MarkdownDescription: "Taint master nodes to prevent workload scheduling (multi-node only)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"node_wait_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for node readiness check (e.g., 600s)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("600s"),
			},
			"primary_master": schema.StringAttribute{
				MarkdownDescription: "Primary master node hostname (first master node)",
				Computed:            true,
			},
			"cluster_type": schema.StringAttribute{
				MarkdownDescription: "Cluster deployment type: single-node or multi-node",
				Computed:            true,
			},
			"kubeconfig_path": schema.StringAttribute{
				MarkdownDescription: "Path to kubeconfig on the primary master node",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update",
				Computed:            true,
			},
			"airgap_install": schema.BoolAttribute{
				MarkdownDescription: "Airgap installation",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"airgap_installation_path": schema.StringAttribute{
				MarkdownDescription: "The Path of K3s installation binary saved",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

// Create implements resource.Resource.
func (k *K3SClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data K3SClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract node lists
	var masterNodes []string
	resp.Diagnostics.Append(data.MasterNodes.ElementsAs(ctx, &masterNodes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var workerNodes []string
	if !data.WorkerNodes.IsNull() {
		resp.Diagnostics.Append(data.WorkerNodes.ElementsAs(ctx, &workerNodes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(masterNodes) == 0 {
		resp.Diagnostics.AddError("Invalid Configuration", "At least one master node is required.")
		return
	}

	config := k3sclient.K3SInstallConfig{
		ClusterName:            data.ClusterName.ValueString(),
		Version:                data.K3SVersion.ValueString(),
		Token:                  data.K3SToken.ValueString(),
		MasterNodes:            masterNodes,
		WorkerNodes:            workerNodes,
		DisableTraefik:         data.DisableTraefik.ValueBool(),
		TaintMasters:           data.TaintMasters.ValueBool(),
		NodeWaitTimeout:        data.NodeWaitTimeout.ValueString(),
		AirgapInstall:          data.AirgapInstall.ValueBool(),
		AirgapInstallationPath: data.AirgapInstallationPath.ValueString(),
	}

	isSingleNode := len(workerNodes) == 0 && len(masterNodes) == 1

	tflog.Info(ctx, "Installing K3S cluster", map[string]interface{}{
		"cluster_name": config.ClusterName,
		"version":      config.Version,
		"single_node":  isSingleNode,
		"masters":      len(masterNodes),
		"workers":      len(workerNodes),
	})

	if isSingleNode {
		// Single-node installation
		err := k.client.InstallK3SSingleNode(config)
		if err != nil {
			resp.Diagnostics.AddError("K3S Single Node Installation Failed", err.Error())
			return
		}
	} else {
		// Multi-node: install primary master
		err := k.client.InstallK3SPrimaryMaster(config)
		if err != nil {
			resp.Diagnostics.AddError("K3S Primary Master Installation Failed", err.Error())
			return
		}

		// Install additional masters
		for i := 1; i < len(masterNodes); i++ {
			err := k.client.InstallK3SAdditionalMaster(config, masterNodes[0], masterNodes[i], i+1)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("K3S Additional Master %d Installation Failed", i+1),
					err.Error(),
				)
				return
			}
		}

		// Install workers
		for i, worker := range workerNodes {
			err := k.client.InstallK3SWorker(config, masterNodes[0], worker, i+1)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("K3S Worker %d Installation Failed", i+1),
					err.Error(),
				)
				return
			}
		}
	}

	// Wait for all nodes to be Ready
	if err := k.client.WaitForNodes(config); err != nil {
		resp.Diagnostics.AddWarning("Node Readiness Warning", err.Error())
	}

	// Verify cluster
	output, err := k.client.VerifyCluster(config)
	if err != nil {
		resp.Diagnostics.AddWarning("K3S Cluster Verification Warning", err.Error())
	}
	tflog.Info(ctx, "Cluster verification output", map[string]interface{}{
		"output": output,
	})

	// Set computed values
	clusterType := "single-node"
	if !isSingleNode {
		clusterType = "multi-node"
	}

	data.ID = data.ClusterName
	data.PrimaryMaster = types.StringValue(masterNodes[0])
	data.ClusterType = types.StringValue(clusterType)
	data.KubeconfigPath = types.StringValue("/etc/rancher/k3s/k3s.yaml")
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "K3S cluster installed successfully", map[string]interface{}{
		"cluster_name": config.ClusterName,
		"cluster_type": clusterType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

// Delete implements resource.Resource.
func (k *K3SClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data K3SClusterResourceModel

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

	var masterNodes []string
	resp.Diagnostics.Append(data.MasterNodes.ElementsAs(ctx, &masterNodes, false)...)

	var workerNodes []string
	if !data.WorkerNodes.IsNull() {
		resp.Diagnostics.Append(data.WorkerNodes.ElementsAs(ctx, &workerNodes, false)...)
	}

	tflog.Info(ctx, "Uninstalling K3S cluster", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})

	// Uninstall workers first
	for _, worker := range workerNodes {
		err := k.client.UninstallK3S(worker, false)
		if err != nil {
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("Failed to uninstall K3S agent from %s", worker),
				err.Error(),
			)
		}
	}

	// Uninstall masters (reverse order, primary last)
	for i := len(masterNodes) - 1; i >= 0; i-- {
		err := k.client.UninstallK3S(masterNodes[i], true)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to uninstall K3S server from %s", masterNodes[i]),
				err.Error(),
			)
			return
		}
	}

	tflog.Info(ctx, "K3S cluster uninstalled", map[string]interface{}{
		"cluster_name": data.ClusterName.ValueString(),
	})
}

// Read implements resource.Resource.
func (k *K3SClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data K3SClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if K3S is still running on the primary master
	primaryMaster := data.PrimaryMaster.ValueString()
	if primaryMaster != "" {
		installed, err := k.client.CheckK3SInstalled(primaryMaster)
		if err != nil {
			resp.Diagnostics.AddWarning("Could not check K3S status", err.Error())
		} else if !installed {
			tflog.Info(ctx, "K3S no longer running on primary master, removing from state", map[string]interface{}{
				"primary_master": primaryMaster,
			})
			resp.State.RemoveResource(ctx)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update implements resource.Resource.
func (k *K3SClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data K3SClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (k *K3SClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if unifiedClient.K3sClient == nil {
		resp.Diagnostics.AddError(
			"K3s Client Not Configured",
			fmt.Sprintf("Expected unifiedClient.K3sClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	k.client = unifiedClient.K3sClient
}

// ImportState implements resource.Resource
func (k *K3SClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// // Modify implements resource.Resource.
// func (k *K3SClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
// 	panic("unimplemented")
// }
