# Copyright (c) IBM Corporation
# SPDX-License-Identifier: Apache-2.0

# Complete OCP Edge Deployment
# Uses unified guardium-data-protection provider for Rook-Ceph and Edge deployment
# Deploys to existing OpenShift clusters

terraform {
  required_providers {
    guardium-data-protection = {
      # For internal testing with IBM Artifactory
      # source  = "registry.terraform.io/ibm/guardium-data-protection"
      # For public release (uncomment when published to HashiCorp registry)
      source  = "hashicorp.com/ibm/guardium-data-protection"
      version = "1.0.0"
    }
  }
}

# ============================================================================
# Locals - OCP cluster configuration
# ============================================================================

locals {
  # OCP API server URL (for edge provider OAuth)
  ocp_api_server = var.ocp_api_server
}

# ============================================================================
# Provider Configuration - Unified Guardium Data Protection Provider
# ============================================================================

provider "guardium-data-protection" {
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
# Resource 1: Install Rook-Ceph Storage for OCP (Optional)
# ============================================================================

resource "guardium-data-protection_rook_ceph_cluster" "this" {
  provider = guardium-data-protection
  count    = var.install_rook_ceph ? 1 : 0

  cluster_name                     = var.cluster_name
  platform                         = "openshift"
  target_node                      = var.ocp_infra_node_hostname
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
# Resource 2: Deploy Edge Components (Optional)
# Finalizer cleanup and namespace termination are handled natively by the
# guardium-data-protection provider via WaitForNamespaceDeletion during destroy.
# ============================================================================

resource "guardium-data-protection_deployment" "edge" {
  count    = var.install_edge ? 1 : 0
  provider = guardium-data-protection

  depends_on = [guardium-data-protection_rook_ceph_cluster.this]

  edge_name             = var.edge_name
  edge_bundle_directory = var.edge_bundle_directory
  platform              = "openshift"

  # OCP authentication
  ocp_server               = local.ocp_api_server
  ocp_username             = var.ocp_admin_user
  ocp_password             = var.ocp_admin_password
  ocp_token                = var.ocp_token
  ocp_insecure_skip_verify = var.ocp_insecure_skip_verify

  monitor_max_attempts       = var.edge_monitor_max_attempts
  monitor_sleep_interval     = var.edge_monitor_sleep_interval
  cleanup_bundle             = var.edge_cleanup_bundle
  delete_timeout             = var.edge_delete_timeout
  ocp_machineconfig_timeout  = var.ocp_machineconfig_timeout
  external_image_registry    = var.external_image_registry
}
