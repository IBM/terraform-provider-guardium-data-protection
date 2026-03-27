// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/edgeclient"
)

// Ensure EdgeDeploymentResource implements Resource interface
var _ resource.Resource = &EdgeDeploymentResource{}
var _ resource.ResourceWithConfigure = &EdgeDeploymentResource{}

// EdgeDeploymentResource defines the resource implementation
type EdgeDeploymentResource struct {
	client *edgeclient.Client
}

// EdgeDeploymentResourceModel describes the resource data model
type EdgeDeploymentResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	EdgeName                types.String `tfsdk:"edge_name"`
	BundleDirectory         types.String `tfsdk:"edge_bundle_directory"`
	Platform                types.String `tfsdk:"platform"`
	K3SMasterNode           types.String `tfsdk:"k3s_master_node"`
	K3SNodes                types.List   `tfsdk:"k3s_nodes"`
	EKSClusterName          types.String `tfsdk:"eks_cluster_name"`
	MonitorMaxAttempts      types.Int64  `tfsdk:"monitor_max_attempts"`
	MonitorSleepInterval    types.Int64  `tfsdk:"monitor_sleep_interval"`
	CleanupBundle           types.Bool   `tfsdk:"cleanup_bundle"`
	DeleteTimeout           types.String `tfsdk:"delete_timeout"`
	OCPMachineConfigTimeout types.String `tfsdk:"ocp_machineconfig_timeout"`

	// Optional OCP auth overrides (resource-level, takes precedence over provider config)
	OCPServer             types.String `tfsdk:"ocp_server"`
	OCPUsername           types.String `tfsdk:"ocp_username"`
	OCPPassword           types.String `tfsdk:"ocp_password"`
	OCPToken              types.String `tfsdk:"ocp_token"`
	OCPInsecureSkipVerify types.Bool   `tfsdk:"ocp_insecure_skip_verify"`

	// Registry configuration
	ExternalImageRegistry types.Bool `tfsdk:"external_image_registry"`

	// Metrics Server configuration
	K8SMetricsServerInstall           types.Bool   `tfsdk:"k8s_metrics_server_install"`
	K8SMetricsServerAirgapInstall     types.Bool   `tfsdk:"k8s_metrics_server_airgap_install"`
	K8SMetricsServerAirgapInstallPath types.String `tfsdk:"k8s_metrics_server_airgap_install_path"`

	// Computed outputs
	EdgeNamespace    types.String `tfsdk:"edge_namespace"`
	RegistryURL      types.String `tfsdk:"registry_url"`
	DeploymentStatus types.String `tfsdk:"deployment_status"`
	WorkDir          types.String `tfsdk:"work_dir"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}

func NewEdgeDeploymentResource() resource.Resource {
	return &EdgeDeploymentResource{}
}

func (r *EdgeDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *EdgeDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deploys GDP Edge components to a Kubernetes cluster using native Kubernetes API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for this edge deployment",
			},
			"edge_name": schema.StringAttribute{
				MarkdownDescription: "Name of the edge to download from Central Manager (requires cm_url and oauth_token in provider config)",
				Optional:            true,
			},
			"edge_bundle_directory": schema.StringAttribute{
				MarkdownDescription: "Local path to pre-extracted edge bundle directory",
				Optional:            true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Target platform: k3s, eks, or openshift (overrides provider setting)",
				Optional:            true,
			},
			"k3s_master_node": schema.StringAttribute{
				MarkdownDescription: "K3S master node hostname/IP for kubeconfig fetch",
				Optional:            true,
			},
			"k3s_nodes": schema.ListAttribute{
				MarkdownDescription: "List of K3S node hostnames/IPs for certificate installation",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"eks_cluster_name": schema.StringAttribute{
				MarkdownDescription: "AWS EKS cluster name",
				Optional:            true,
			},
			"monitor_max_attempts": schema.Int64Attribute{
				MarkdownDescription: "Maximum polling attempts for deployment monitoring (default: 180)",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(180),
			},
			"monitor_sleep_interval": schema.Int64Attribute{
				MarkdownDescription: "Sleep interval in seconds between monitoring polls (default: 10)",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(10),
			},
			"cleanup_bundle": schema.BoolAttribute{
				MarkdownDescription: "Whether to cleanup downloaded bundle on destroy (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"delete_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for the delete operation (e.g. '2h', '90m'). Default: 2h.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("2h"),
			},
			"ocp_machineconfig_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for OpenShift MachineConfig rollout during certificate installation (e.g. '10m', '30m', '1h'). Default: 60m. Increase this for large clusters or slow node updates.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("30m"),
			},
			"external_image_registry": schema.BoolAttribute{
				MarkdownDescription: "Set to true when using an external image registry (e.g. Docker Hub, Quay) instead of the CM private registry. Skips registry certificate installation on cluster nodes (default: false).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},

			// Metrics Server configuration
			"k8s_metrics_server_install": schema.BoolAttribute{
				MarkdownDescription: "Whether to install Kubernetes Metrics Server before deploying Edge. Uses AWS SDK credentials — no static Kubernetes token required (default: false).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"k8s_metrics_server_airgap_install": schema.BoolAttribute{
				MarkdownDescription: "Use airgap (offline) installation for Kubernetes Metrics Server. Reads YAML manifests from k8s_metrics_server_airgap_install_path (default: false).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"k8s_metrics_server_airgap_install_path": schema.StringAttribute{
				MarkdownDescription: "Local directory path containing Metrics Server YAML manifests for airgap installation. Required when k8s_metrics_server_airgap_install is true.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},

			// Optional OCP auth overrides (resource-level, takes precedence over provider config)
			// Use these when OCP credentials are only available at apply time
			"ocp_server": schema.StringAttribute{
				MarkdownDescription: "OpenShift API server URL (overrides provider setting)",
				Optional:            true,
			},
			"ocp_username": schema.StringAttribute{
				MarkdownDescription: "OpenShift username (overrides provider setting)",
				Optional:            true,
			},
			"ocp_password": schema.StringAttribute{
				MarkdownDescription: "OpenShift password (overrides provider setting)",
				Optional:            true,
				Sensitive:           true,
			},
			"ocp_token": schema.StringAttribute{
				MarkdownDescription: "OpenShift OAuth token (overrides provider setting)",
				Optional:            true,
				Sensitive:           true,
			},
			"ocp_insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS verification for OpenShift API server (overrides provider setting)",
				Optional:            true,
			},

			// Computed outputs
			"edge_namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace where Edge components are deployed",
				Computed:            true,
			},
			"registry_url": schema.StringAttribute{
				MarkdownDescription: "Container registry URL extracted from bundle",
				Computed:            true,
			},
			"deployment_status": schema.StringAttribute{
				MarkdownDescription: "Final deployment status",
				Computed:            true,
			},
			"work_dir": schema.StringAttribute{
				MarkdownDescription: "Working directory for the edge bundle",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update",
				Computed:            true,
			},
		},
	}
}

func (r *EdgeDeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if unifiedClient.EdgeClient == nil {
		resp.Diagnostics.AddError(
			"Edge Client Not Configured",
			fmt.Sprintf("Expected unifiedClient.EdgeClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = unifiedClient.EdgeClient
}

func (r *EdgeDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EdgeDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine platform
	platform := r.client.Config.Platform
	if !data.Platform.IsNull() && data.Platform.ValueString() != "" {
		platform = data.Platform.ValueString()
	}

	// Determine work directory
	useLocalBundle := !data.BundleDirectory.IsNull() && data.BundleDirectory.ValueString() != ""
	var workDir string
	if useLocalBundle {
		workDir = data.BundleDirectory.ValueString()
	} else {
		edgeName := data.EdgeName.ValueString()
		if edgeName == "" {
			resp.Diagnostics.AddError("Missing Configuration", "Either edge_bundle_directory or edge_name must be provided.")
			return
		}
		if r.client.Config.CMUrl == "" || r.client.Config.OAuthToken == "" {
			resp.Diagnostics.AddError("Missing Provider Configuration", "cm_url and oauth_token must be set in the provider config to download a bundle.")
			return
		}

		workDir = filepath.Join("/tmp", fmt.Sprintf("edge-bundle-%s", edgeName))
		tflog.Info(ctx, "Downloading edge bundle from CM", map[string]interface{}{"edge_name": edgeName})

		if err := r.client.DownloadBundle(edgeName, workDir); err != nil {
			resp.Diagnostics.AddError("Bundle Download Failed", err.Error())
			return
		}
		// The CM archive stores files under an edgeName/ subdirectory; point workDir there.
		workDir = filepath.Join(workDir, edgeName)
	}

	tflog.Info(ctx, "Starting Edge deployment", map[string]interface{}{
		"platform": platform,
		"work_dir": workDir,
	})

	// Fetch kubeconfig for K3S (internal, not stored in state)
	var kubeconfigPath string
	if platform == "k3s" {
		masterNode := data.K3SMasterNode.ValueString()
		if masterNode != "" {
			kubeconfigPath = filepath.Join(workDir, ".kubeconfig")
			tflog.Info(ctx, "Fetching kubeconfig from K3S master", map[string]interface{}{"master": masterNode})

			if err := r.client.FetchKubeconfig(masterNode, kubeconfigPath); err != nil {
				resp.Diagnostics.AddError("Kubeconfig Fetch Failed", err.Error())
				return
			}
		}
	}

	// Apply resource-level EKS cluster name override
	if !data.EKSClusterName.IsNull() && data.EKSClusterName.ValueString() != "" {
		r.client.Config.EKSClusterName = data.EKSClusterName.ValueString()
	}

	// Apply resource-level OCP auth overrides (takes precedence over provider config)
	if !data.OCPServer.IsNull() && data.OCPServer.ValueString() != "" {
		r.client.Config.OCPServer = data.OCPServer.ValueString()
	}
	if !data.OCPUsername.IsNull() && data.OCPUsername.ValueString() != "" {
		r.client.Config.OCPUsername = data.OCPUsername.ValueString()
	}
	if !data.OCPPassword.IsNull() && data.OCPPassword.ValueString() != "" {
		r.client.Config.OCPPassword = data.OCPPassword.ValueString()
	}
	if !data.OCPToken.IsNull() && data.OCPToken.ValueString() != "" {
		r.client.Config.OCPToken = data.OCPToken.ValueString()
	}
	if !data.OCPInsecureSkipVerify.IsNull() {
		r.client.Config.OCPInsecureSkipVerify = data.OCPInsecureSkipVerify.ValueBool()
	}

	// Initialize K8s client
	tflog.Info(ctx, "Initializing Kubernetes client", map[string]interface{}{
		"platform":         r.client.Config.Platform,
		"eks_cluster_name": r.client.Config.EKSClusterName,
		"auth_method":      "WrapTransport-v2",
	})
	if err := r.client.InitK8sClient(ctx, kubeconfigPath); err != nil {
		resp.Diagnostics.AddError("K8s Client Initialization Failed", err.Error())
		return
	}

	// Install Kubernetes Metrics Server if requested (before Edge deployment).
	// Only supported on EKS — k3s and OpenShift have their own metrics infrastructure.
	// Uses AWS SDK credentials via the initialized K8s client — no static token needed.
	if data.K8SMetricsServerInstall.ValueBool() && platform == "eks" {
		tflog.Info(ctx, "Installing Kubernetes Metrics Server")
		airgap := data.K8SMetricsServerAirgapInstall.ValueBool()
		airgapPath := data.K8SMetricsServerAirgapInstallPath.ValueString()
		if err := r.client.InstallMetricsServer(ctx, airgap, airgapPath); err != nil {
			resp.Diagnostics.AddError("Metrics Server Installation Failed", err.Error())
			return
		}
	} else if data.K8SMetricsServerInstall.ValueBool() && platform != "eks" {
		tflog.Warn(ctx, "k8s_metrics_server_install is only supported on EKS; skipping for platform: "+platform)
	}

	// Extract certificate info
	tflog.Info(ctx, "Extracting certificate information")
	registry, namespace, err := r.client.ExtractCertInfo(workDir, data.ExternalImageRegistry.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Certificate Extraction Failed", err.Error())
		return
	}

	tflog.Info(ctx, "Certificate info extracted", map[string]interface{}{
		"registry":  registry,
		"namespace": namespace,
	})

	// Install certificates (skip when using an external image registry)
	if data.ExternalImageRegistry.ValueBool() {
		tflog.Info(ctx, "Skipping certificate installation: external_image_registry is enabled")
	} else {
		tflog.Info(ctx, "Installing certificates", map[string]interface{}{"platform": platform})

		switch platform {
		case "k3s":
			var k3sNodes []string
			if !data.K3SNodes.IsNull() {
				resp.Diagnostics.Append(data.K3SNodes.ElementsAs(ctx, &k3sNodes, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			if err := r.client.InstallCertsK3S(ctx, workDir, k3sNodes, registry); err != nil {
				resp.Diagnostics.AddError("Certificate Installation Failed", err.Error())
				return
			}
		case "openshift":
			// Parse MachineConfig timeout
			mcTimeout, err := time.ParseDuration(data.OCPMachineConfigTimeout.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid OCP MachineConfig Timeout",
					fmt.Sprintf("Failed to parse ocp_machineconfig_timeout: %s", err.Error()))
				return
			}

			tflog.Info(ctx, "Installing certificates on OpenShift", map[string]interface{}{
				"machineconfig_timeout": mcTimeout.String(),
			})

			if err := r.client.InstallCertsOpenShift(ctx, workDir, registry, mcTimeout); err != nil {
				resp.Diagnostics.AddError("Certificate Installation Failed", err.Error())
				return
			}
		case "eks":
			tflog.Info(ctx, "Installing certificates on EKS worker nodes")
			if err := r.client.InstallCertsEKS(ctx, workDir, registry); err != nil {
				resp.Diagnostics.AddError("Certificate Installation Failed", err.Error())
				return
			}
		default:
			resp.Diagnostics.AddError("Invalid Platform", fmt.Sprintf("Unsupported platform: %s", platform))
			return
		}
	}

	// Deploy Edge components
	tflog.Info(ctx, "Deploying Edge components")
	if err := r.client.DeployEdge(ctx, workDir, namespace, platform); err != nil {
		resp.Diagnostics.AddError("Edge Deployment Failed", err.Error())
		return
	}

	// Monitor deployment status
	tflog.Info(ctx, "Monitoring Edge deployment status")
	maxAttempts := int(data.MonitorMaxAttempts.ValueInt64())
	sleepInterval := int(data.MonitorSleepInterval.ValueInt64())

	status, err := r.client.MonitorDeployment(ctx, namespace, maxAttempts, sleepInterval)
	if err != nil {
		// Monitoring failed, but deployment succeeded - save state with warning
		tflog.Warn(ctx, "Edge deployment monitoring failed, but resources were deployed", map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
		resp.Diagnostics.AddWarning(
			"Edge Deployment Monitoring Failed",
			fmt.Sprintf("Edge resources were deployed successfully, but monitoring failed: %s\nLast known status: %s\n\nThe deployment is in state and will be managed by Terraform. Verify deployment manually with: kubectl get pods -n %s",
				err.Error(), status, namespace),
		)
		// Use last known status or "Unknown" if monitoring completely failed
		if status == "" {
			status = "Unknown - Monitoring Failed"
		}
	}

	// Set computed values - ALWAYS save state after successful deployment
	data.ID = types.StringValue(fmt.Sprintf("%s-%s", platform, namespace))
	data.Platform = types.StringValue(platform)
	data.EdgeNamespace = types.StringValue(namespace)
	data.RegistryURL = types.StringValue(registry)
	data.DeploymentStatus = types.StringValue(status)
	data.WorkDir = types.StringValue(workDir)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Info(ctx, "Edge deployment completed", map[string]interface{}{
		"namespace": namespace,
		"platform":  platform,
		"status":    status,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EdgeDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EdgeDeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For now, just return the stored state
	// In a production implementation, you might want to query the cluster
	// to verify the deployment still exists

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EdgeDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeDeploymentResourceModel
	var state EdgeDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state to preserve computed-only values that Terraform cannot
	// know at plan time (they are set only by the provider during Create).
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.EdgeNamespace = state.EdgeNamespace
	plan.RegistryURL = state.RegistryURL
	plan.DeploymentStatus = state.DeploymentStatus
	plan.WorkDir = state.WorkDir

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EdgeDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EdgeDeploymentResourceModel

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

	namespace := data.EdgeNamespace.ValueString()
	workDir := data.WorkDir.ValueString()

	tflog.Info(ctx, "Deleting Edge deployment", map[string]interface{}{
		"namespace": namespace,
		"work_dir":  workDir,
	})

	// Initialize K8s client if needed (derive kubeconfig path from work_dir)
	kubeconfigPath := filepath.Join(workDir, ".kubeconfig")
	if r.client.K8sClient() == nil {
		if err := r.client.InitK8sClient(ctx, kubeconfigPath); err != nil {
			resp.Diagnostics.AddWarning("K8s Client Initialization Warning",
				fmt.Sprintf("Could not initialize K8s client for cleanup: %v", err))
		}
	}

	// Delete Edge resources
	if r.client.K8sClient() != nil && namespace != "" {
		if err := r.client.DeleteEdge(ctx, workDir, namespace); err != nil {
			resp.Diagnostics.AddWarning("Edge Deletion Warning",
				fmt.Sprintf("Could not delete all edge resources: %v", err))
		}
	}

	// Cleanup bundle if requested
	if data.CleanupBundle.ValueBool() && workDir != "" {
		if !data.BundleDirectory.IsNull() && data.BundleDirectory.ValueString() != "" {
			// Don't delete user-provided bundle directories
			tflog.Info(ctx, "Skipping cleanup of user-provided bundle directory")
		} else {
			if err := r.client.CleanupBundle(workDir); err != nil {
				resp.Diagnostics.AddWarning("Bundle Cleanup Warning", err.Error())
			}
		}
	}

	tflog.Info(ctx, "Edge deployment deleted")
}
