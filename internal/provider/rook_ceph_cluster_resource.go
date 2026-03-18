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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/rookcephclient"
)

var _ resource.Resource = &RookCephClusterResource{}
var _ resource.ResourceWithImportState = &RookCephClusterResource{}

func NewRookCephClusterResource() resource.Resource {
	return &RookCephClusterResource{}
}

type RookCephClusterResource struct {
	client *rookcephclient.Client
}

type RookCephClusterResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	ClusterName              types.String `tfsdk:"cluster_name"`
	Platform                 types.String `tfsdk:"platform"`
	TargetNode               types.String `tfsdk:"target_node"`
	RookCephVersion          types.String `tfsdk:"rook_ceph_version"`
	RookCephInstallationPath types.String `tfsdk:"airgap_rook_ceph_installation_path"`
	AirgapInstall            types.Bool   `tfsdk:"airgap_install"`
	WorkerCount              types.Int64  `tfsdk:"worker_count"`
	TaintMasters             types.Bool   `tfsdk:"taint_masters"`
	SetAsDefaultStorage      types.Bool   `tfsdk:"set_as_default_storage"`
	DisableLocalPath         types.Bool   `tfsdk:"disable_local_path"`
	PodWaitTimeout           types.String `tfsdk:"pod_wait_timeout"`
	SleepBetweenSteps        types.Int64  `tfsdk:"sleep_between_steps"`
	DeleteTimeout            types.String `tfsdk:"delete_timeout"`

	// Computed
	ClusterType        types.String `tfsdk:"cluster_type"`
	Namespace          types.String `tfsdk:"namespace"`
	CephfsStorageClass types.String `tfsdk:"cephfs_storage_class"`
	BlockStorageClass  types.String `tfsdk:"block_storage_class"`
	LastUpdated        types.String `tfsdk:"last_updated"`
}

func (r *RookCephClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rook_ceph_cluster"
}

func (r *RookCephClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Installs and manages Rook-Ceph distributed storage on Kubernetes clusters. Supports both K3S and OpenShift platforms.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Target platform: 'k3s' or 'openshift'",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_node": schema.StringAttribute{
				MarkdownDescription: "Target node hostname for SSH operations (primary master for K3S, API node for OpenShift)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rook_ceph_version": schema.StringAttribute{
				MarkdownDescription: "Rook-Ceph version to install (e.g., v1.15.4)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("v1.15.4"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"airgap_rook_ceph_installation_path": schema.StringAttribute{
				MarkdownDescription: "Rook-Ceph airgap installation yaml root directory",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"airgap_install": schema.BoolAttribute{
				MarkdownDescription: "Use airgap installation",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes. For K3S: 0-1 = test cluster, 2+ = production. Not used for OpenShift.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"taint_masters": schema.BoolAttribute{
				MarkdownDescription: "Whether master nodes are tainted (not schedulable for regular workloads). When true with worker_count=1, sets CSI provisioner replicas to 1 to avoid anti-affinity scheduling failures.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"set_as_default_storage": schema.BoolAttribute{
				MarkdownDescription: "Set rook-cephfs as the default storage class",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"disable_local_path": schema.BoolAttribute{
				MarkdownDescription: "Disable local-path storage in K3S (only applies to K3S platform)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"pod_wait_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for waiting on pod readiness (e.g., 600s)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("600s"),
			},
			"sleep_between_steps": schema.Int64Attribute{
				MarkdownDescription: "Sleep time in seconds between installation steps",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},
			"delete_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for the delete operation (e.g. '2h', '90m'). Default: 2h.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("2h"),
			},

			// Computed outputs
			"cluster_type": schema.StringAttribute{
				MarkdownDescription: "Cluster type: 'test' or 'production' (K3S only)",
				Computed:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace where Rook-Ceph is installed",
				Computed:            true,
			},
			"cephfs_storage_class": schema.StringAttribute{
				MarkdownDescription: "Name of the CephFS storage class",
				Computed:            true,
			},
			"block_storage_class": schema.StringAttribute{
				MarkdownDescription: "Name of the RBD block storage class",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update",
				Computed:            true,
			},
		},
	}
}

func (r *RookCephClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if unifiedClient.RookCephClient == nil {
		resp.Diagnostics.AddError(
			"RookCeph Client Not Configured",
			fmt.Sprintf("Expected unifiedClient.RookCephClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = unifiedClient.RookCephClient
}

func (r *RookCephClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RookCephClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	platform := data.Platform.ValueString()
	if platform != "k3s" && platform != "openshift" {
		resp.Diagnostics.AddError("Invalid Platform", "Platform must be 'k3s' or 'openshift'.")
		return
	}

	workerCount := int(data.WorkerCount.ValueInt64())
	taintMasters := data.TaintMasters.ValueBool()
	isTest := workerCount <= 1 && platform == "k3s"

	if data.AirgapInstall.ValueBool() && data.RookCephInstallationPath.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid Configuration",
			"rook_ceph_installation_path is required when airgap_install is true.")
		return
	}

	config := rookcephclient.RookCephConfig{
		ClusterName:              data.ClusterName.ValueString(),
		Platform:                 platform,
		TargetNode:               data.TargetNode.ValueString(),
		RookCephVersion:          data.RookCephVersion.ValueString(),
		RookCephInstallationPath: data.RookCephInstallationPath.ValueString(),
		AirgapInstall:            data.AirgapInstall.ValueBool(),
		WorkerCount:              workerCount,
		TaintMasters:             taintMasters,
		SetAsDefaultStorage:      data.SetAsDefaultStorage.ValueBool(),
		DisableLocalPath:         data.DisableLocalPath.ValueBool(),
		PodWaitTimeout:           data.PodWaitTimeout.ValueString(),
		SleepBetweenSteps:        int(data.SleepBetweenSteps.ValueInt64()),
	}

	tflog.Info(ctx, "Installing Rook-Ceph", map[string]interface{}{
		"cluster_name": config.ClusterName,
		"platform":     platform,
		"version":      config.RookCephVersion,
		"is_test":      isTest,
	})

	// Step 1: Install Rook-Ceph
	err := r.client.InstallRookCeph(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Rook-Ceph Installation Failed", err.Error())
		return
	}

	// Step 2: Configure default storage (optional)
	if config.SetAsDefaultStorage {
		tflog.Info(ctx, "Configuring default storage class")
		err = r.client.ConfigureDefaultStorage(ctx, config)
		if err != nil {
			resp.Diagnostics.AddWarning("Default Storage Configuration Warning", err.Error())
		}
	}

	// Step 3: Verify (blocks until Ceph reaches HEALTH_OK or times out)
	tflog.Info(ctx, "Verifying Rook-Ceph installation")
	output, err := r.client.VerifyInstallation(ctx, config)
	tflog.Info(ctx, "Verification output", map[string]interface{}{"output": output})
	if err != nil {
		resp.Diagnostics.AddError("Rook-Ceph Verification Failed", err.Error())
		return
	}

	// Set computed values
	clusterType := "production"
	if isTest {
		clusterType = "test"
	}

	blockSC := "rook-ceph-block"

	data.ID = types.StringValue(fmt.Sprintf("%s-%s", platform, data.ClusterName.ValueString()))
	data.ClusterType = types.StringValue(clusterType)
	data.Namespace = types.StringValue("rook-ceph")
	data.CephfsStorageClass = types.StringValue("rook-cephfs")
	data.BlockStorageClass = types.StringValue(blockSC)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Rook-Ceph installed successfully", map[string]interface{}{
		"cluster_name": config.ClusterName,
		"cluster_type": clusterType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RookCephClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RookCephClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := rookcephclient.RookCephConfig{
		Platform:   data.Platform.ValueString(),
		TargetNode: data.TargetNode.ValueString(),
	}

	installed, err := r.client.CheckRookCephInstalled(ctx, config)
	if err != nil {
		resp.Diagnostics.AddWarning("Could not check Rook-Ceph status", err.Error())
	} else if !installed {
		tflog.Info(ctx, "Rook-Ceph no longer installed, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RookCephClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RookCephClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RookCephClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RookCephClusterResourceModel

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

	config := rookcephclient.RookCephConfig{
		ClusterName: data.ClusterName.ValueString(),
		Platform:    data.Platform.ValueString(),
		TargetNode:  data.TargetNode.ValueString(),
	}

	tflog.Info(ctx, "Uninstalling Rook-Ceph", map[string]interface{}{
		"cluster_name": config.ClusterName,
	})

	err = r.client.UninstallRookCeph(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Rook-Ceph Uninstall Failed",
			fmt.Sprintf("Could not fully uninstall Rook-Ceph: %s", err.Error()))
		return
	}

	tflog.Info(ctx, "Rook-Ceph uninstalled")
}

func (r *RookCephClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
