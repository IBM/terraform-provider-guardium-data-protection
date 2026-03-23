// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

# Complete K3S Edge Deployment
# Uses unified guardium-data-protection Terraform provider for K3S, Rook-Ceph, and Edge deployment
# Deploys to existing K3S clusters

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
# Locals - K3S cluster configuration
# ============================================================================

locals {
  # K3S nodes for edge deployment
  k3s_nodes       = var.k3s_nodes
  k3s_master_node = var.k3s_master_node

  # Worker count for rook-ceph (total nodes minus master)
  worker_count = length(var.k3s_nodes) - 1
}

# ============================================================================
# Provider Configurations
# ============================================================================

provider "guardium-data-protection" {
  # Edge deployment config
  cm_url       = var.edge_cm_url
  oauth_token  = var.edge_oauth_token
  platform     = "k3s"
  ssh_user     = var.ssh_user
  ssh_password = var.ssh_password
  
  # K3s config (prefixed)
  k3s_ssh_user              = var.ssh_user
  k3s_ssh_password          = var.ssh_password
  k3s_connect_timeout       = var.ssh_options.connect_timeout
  k3s_server_alive_interval = var.ssh_options.server_alive_interval
  k3s_server_alive_count    = var.ssh_options.server_alive_count
  
  # Rook-Ceph config (prefixed)
  rook_ceph_ssh_user              = var.ssh_user
  rook_ceph_ssh_password          = var.ssh_password
  rook_ceph_connect_timeout       = var.ssh_options.connect_timeout
  rook_ceph_server_alive_interval = var.ssh_options.server_alive_interval
  rook_ceph_server_alive_count    = var.ssh_options.server_alive_count
}

# ============================================================================
# Resource 1: Install K3S on Existing Nodes (Optional)
# When install_k3s=false: K3S resource is not created, existing K3S is left untouched
# When install_k3s=true: K3S is installed and managed by Terraform
# ============================================================================

resource "guardium-data-protection_k3s_cluster" "main" {
  provider = guardium-data-protection
  count    = var.install_k3s ? 1 : 0  # count=0 means resource not created, existing K3S unaffected

  cluster_name             = var.cluster_name
  master_nodes             = [var.k3s_master_node]
  worker_nodes             = length(var.k3s_nodes) > 1 ? slice(var.k3s_nodes, 1, length(var.k3s_nodes)) : []
  k3s_version              = var.k3s_version
  k3s_token                = var.k3s_token
  airgap_install           = var.k3s_airgap_install
  airgap_installation_path = var.k3s_airgap_installation_path
  disable_traefik          = var.k3s_install_options.disable_traefik
  taint_masters            = var.k3s_install_options.taint_masters
  node_wait_timeout        = var.k3s_install_options.node_wait_timeout
  delete_timeout           = var.k3s_delete_timeout
}

# ============================================================================
# Resource 2: Install Rook-Ceph Storage (Optional)
# ============================================================================

resource "guardium-data-protection_rook_ceph_cluster" "this" {
  provider = guardium-data-protection
  count    = var.install_rook_ceph ? 1 : 0

  depends_on = [guardium-data-protection_k3s_cluster.main]

  cluster_name                = var.cluster_name
  platform                    = "k3s"
  target_node                 = local.k3s_master_node
  rook_ceph_version           = var.rook_ceph_version
  airgap_rook_ceph_installation_path = var.rook_ceph_airgap_installation_path
  airgap_install              = var.rook_ceph_airgap_install
  worker_count                = local.worker_count
  taint_masters               = var.k3s_install_options.taint_masters
  set_as_default_storage      = var.rook_ceph_config.set_as_default_storage
  disable_local_path          = var.rook_ceph_config.disable_local_path
  pod_wait_timeout            = var.rook_ceph_config.pod_wait_timeout
  sleep_between_steps         = var.rook_ceph_config.sleep_between_steps
  delete_timeout              = var.rook_ceph_delete_timeout
}

# ============================================================================
# Resource 3: Deploy Edge Components (Optional)
# Finalizer cleanup and namespace termination are handled natively by the
# guardium-data-protection provider via WaitForNamespaceDeletion during destroy.
# ============================================================================

resource "guardium-data-protection_deployment" "edge" {
  provider = guardium-data-protection
  count    = var.install_edge ? 1 : 0

  depends_on = [
    guardium-data-protection_k3s_cluster.main,
    guardium-data-protection_rook_ceph_cluster.this,
  ]

  edge_name             = var.edge_name
  edge_bundle_directory = var.edge_bundle_directory
  platform              = "k3s"
  k3s_master_node       = local.k3s_master_node
  k3s_nodes             = local.k3s_nodes

  monitor_max_attempts    = var.edge_monitor_max_attempts
  monitor_sleep_interval  = var.edge_monitor_sleep_interval
  cleanup_bundle          = var.edge_cleanup_bundle
  delete_timeout          = var.edge_delete_timeout
  external_image_registry = var.external_image_registry
}
