# Complete OCP Fyre Edge Deployment - Consolidated Provider Edition
# Uses unified guardium-data-protection provider for all resources (Fyre OCP, Rook-Ceph, Edge deployment)
# Set manage_openshift=false and deploy_openshift=false to use an existing cluster without Terraform management
# Set manage_openshift=true and deploy_openshift=false to skip creation but allow Terraform to manage existing cluster

terraform {
  required_providers {
    guardium-data-protection = {
      # For internal testing with IBM Artifactory
      source  = "registry.terraform.io/ibm/guardium-data-protection"
      # For public release (uncomment when published to HashiCorp registry)
      # source  = "hashicorp.com/ibm/guardium-data-protection"
      version = "1.0.0"
    }
  }
}

# ============================================================================
# Locals - Construct OCP infrastructure hostname and API server URL
# ============================================================================

locals {
  # OCP infrastructure node hostname (for SSH access and API)
  # Priority: 1) Use ocp_master_node_hostname if provided, 2) Construct from cluster_name
  ocp_inf_hostname = var.ocp_infra_node_hostname != "" ? (
    var.ocp_infra_node_hostname
  ) : "${var.cluster_name}-inf.fyre.ibm.com"

  # OCP API server URL (for edge provider OAuth)
  ocp_api_server = "https://${local.ocp_inf_hostname}:6443"
}

# ============================================================================
# Provider Configuration - Unified Guardium Data Protection Provider
# ============================================================================

provider "guardium-data-protection" {
  # Fyre configuration (prefixed)
  fyre_username         = var.fyre_username
  fyre_api_key          = var.fyre_api_key
  fyre_product_group_id = var.product_group_id

  # Rook-Ceph configuration (prefixed)
  rook_ceph_ssh_user             = var.ocp_ssh_user
  rook_ceph_ssh_password         = var.ocp_ssh_password
  rook_ceph_connect_timeout      = var.ssh_options.connect_timeout
  rook_ceph_server_alive_interval = var.ssh_options.server_alive_interval
  rook_ceph_server_alive_count    = var.ssh_options.server_alive_count

  # Edge deployment configuration
  cm_url       = var.edge_cm_url
  oauth_token  = var.edge_oauth_token
  platform     = "openshift"
  ssh_user     = var.ocp_ssh_user
  ssh_password = var.ocp_ssh_password

  # OpenShift native OAuth
  ocp_server               = local.ocp_api_server
  ocp_username             = var.ocp_admin_user
  ocp_password             = var.ocp_admin_password
  ocp_token                = var.ocp_token
  ocp_insecure_skip_verify = var.ocp_insecure_skip_verify
}

# ============================================================================
# Resource 1: Create OCP Cluster on Fyre (Optional)
# ============================================================================

resource "guardium-data-protection_fyre_ocp" "cluster" {
  provider = guardium-data-protection
  count    = var.manage_openshift && var.deploy_openshift ? 1 : 0

  name        = var.cluster_name
  description = var.cluster_description

  # Cluster Configuration
  platform    = var.ocp_platform
  site        = var.site
  ocp_version = var.ocp_version

  # Quota Configuration
  quota_type       = var.quota_type
  product_group_id = var.product_group_id
  time_to_live     = var.time_to_live

  # Security
  fips    = var.fips_enabled ? "yes" : "no"
  ssh_key = var.ssh_key

  # Master Nodes
  master = [{
    count           = var.master_node_count
    cpu             = var.master_node_cpu
    memory          = var.master_node_memory
    additional_disk = var.master_additional_disks
  }]

  # Worker Nodes
  worker = [{
    count           = var.worker_node_count
    cpu             = var.worker_node_cpu
    memory          = var.worker_node_memory
    additional_disk = var.worker_additional_disks
  }]

  # Polling Configuration
  wait_for_cluster         = var.wait_for_cluster
  polling_timeout_minutes  = var.polling_timeout_minutes
  polling_interval_seconds = var.polling_interval_seconds
  delete_timeout           = var.fyre_ocp_delete_timeout
}

# ============================================================================
# Resource 2: Install Rook-Ceph Storage for OCP (Optional)
# ============================================================================

resource "guardium-data-protection_rook_ceph_cluster" "this" {
  provider = guardium-data-protection
  count    = var.manage_rook_ceph && var.install_rook_ceph ? 1 : 0

  depends_on = [guardium-data-protection_fyre_ocp.cluster]

  cluster_name                     = var.cluster_name
  platform                         = "openshift"
  target_node                      = local.ocp_inf_hostname
  rook_ceph_version                = var.rook_ceph_version
  airgap_rook_ceph_installation_path = var.rook_ceph_airgap_installation_path
  airgap_install                   = var.rook_ceph_airgap_install
  worker_count                     = var.worker_node_count
  set_as_default_storage           = var.rook_ceph_config.set_as_default_storage
  pod_wait_timeout                 = var.rook_ceph_config.pod_wait_timeout
  sleep_between_steps              = var.rook_ceph_config.sleep_between_steps
  delete_timeout                   = var.rook_ceph_delete_timeout
}

# ============================================================================
# Resource 3: Deploy Edge Components (Optional)
# Finalizer cleanup and namespace termination are handled natively by the
# guardium-data-protection provider via WaitForNamespaceDeletion during destroy.
# ============================================================================

resource "guardium-data-protection_deployment" "edge" {
  count    = var.manage_edge && var.install_edge ? 1 : 0
  provider = guardium-data-protection

  depends_on = [
    guardium-data-protection_fyre_ocp.cluster,
    guardium-data-protection_rook_ceph_cluster.this,
  ]

  edge_name             = var.edge_name
  edge_bundle_directory = var.edge_bundle_directory
  platform              = "openshift"

  # OCP auth: use kubeadmin_password from fyre_ocp if cluster was created by Terraform,
  # otherwise fall back to user-provided variables
  ocp_server               = local.ocp_api_server
  ocp_username             = var.ocp_admin_user
  ocp_password             = var.manage_openshift && var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].kubeadmin_password : var.ocp_admin_password
  ocp_token                = var.ocp_token
  ocp_insecure_skip_verify = var.ocp_insecure_skip_verify

  monitor_max_attempts    = var.edge_monitor_max_attempts
  monitor_sleep_interval  = var.edge_monitor_sleep_interval
  cleanup_bundle          = var.edge_cleanup_bundle
  delete_timeout          = var.edge_delete_timeout
  external_image_registry = var.external_image_registry
}
