# Complete K3S Deployment on Fyre - Custom Provider Edition
# Uses unified guardium-data-protection Terraform provider for all resources
# Set install_k3s=false to skip VM/K3S creation and use an existing cluster

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
# Locals - Construct node FQDNs from cluster name and node definitions
# ============================================================================

locals {
  domain_suffix = var.fyre_cluster_type == "beta-fyre" ? "dev.fyre.ibm.com" : "fyre.ibm.com"

  # Generate FQDNs only if creating Fyre VMs
  master_node_fqdns = var.create_fyre_vm ? [
    for master in var.master_nodes : "${var.cluster_name}-${master.name}.${local.domain_suffix}"
  ] : []

  worker_node_fqdns = var.create_fyre_vm ? [
    for worker in var.worker_nodes : "${var.cluster_name}-${worker.name}.${local.domain_suffix}"
  ] : []

  all_node_fqdns = concat(local.master_node_fqdns, local.worker_node_fqdns)

  # Primary master node (from deployed cluster or external)
  primary_master = var.create_fyre_vm ? (
    length(local.master_node_fqdns) > 0 ? local.master_node_fqdns[0] : ""
  ) : var.external_k3s_master_node

  # K3S nodes for edge deployment
  k3s_nodes       = var.create_fyre_vm ? local.all_node_fqdns : var.external_k3s_nodes
  k3s_master_node = local.primary_master

  # Worker count for rook-ceph
  worker_count = var.create_fyre_vm ? length(local.worker_node_fqdns) : var.external_worker_count

  # Cluster name for rook-ceph
  rook_cluster_name = var.create_fyre_vm ? var.cluster_name : var.external_cluster_name
}

# ============================================================================
# Provider Configurations
# ============================================================================

provider "guardium-data-protection" {
  # Edge deployment config
  cm_url      = var.edge_cm_url
  oauth_token = var.edge_oauth_token
  platform    = "k3s"
  ssh_user    = var.ssh_user
  ssh_password = var.ssh_password
  
  # Fyre config (prefixed)
  fyre_username         = var.fyre_user_name
  fyre_api_key          = var.fyre_user_apikey
  fyre_cluster_type     = var.fyre_cluster_type
  fyre_product_group_id = var.fyre_product_group_id
  
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
# Resource 1: Create Fyre VMs (Optional)
# ============================================================================

resource "guardium-data-protection_fyre_vm" "cluster" {
  provider = guardium-data-protection
  count = var.create_fyre_vm ? 1 : 0

  cluster_name     = var.cluster_name
  product_group_id = var.fyre_product_group_id

  master_nodes = [
    for master in var.master_nodes : {
      name                 = master.name
      count                = master.count
      cpu                  = master.cpu
      memory               = master.memory
      os                   = master.os
      additional_disk_size = master.additional_disk_size
    }
  ]

  worker_nodes = [
    for worker in var.worker_nodes : {
      name                 = worker.name
      count                = worker.count
      cpu                  = worker.cpu
      memory               = worker.memory
      os                   = worker.os
      additional_disk_size = worker.additional_disk_size
    }
  ]

  cluster_config = {
    platform = var.cluster_config.platform
  }

  network_config = {
    public_vlan  = var.network_config.public_vlan
    private_vlan = var.network_config.private_vlan
    dns          = var.network_config.dns
  }

  polling_timeout_minutes  = var.polling_timeout_minutes
  polling_interval_seconds = var.polling_interval_seconds
  delete_timeout           = var.fyre_vm_delete_timeout
}

# ============================================================================
# Resource 2: Install K3S on Created VMs (Optional)
# When manage_k3s=false: K3S resource is not created, existing K3S is left untouched
# When manage_k3s=true && install_k3s=true: K3S is installed and managed by Terraform
# ============================================================================

resource "guardium-data-protection_k3s_cluster" "main" {
  provider = guardium-data-protection
  count = var.manage_k3s && var.install_k3s ? 1 : 0  # count=0 means resource not created, existing K3S unaffected

  depends_on = [guardium-data-protection_fyre_vm.cluster]

  cluster_name             = var.create_fyre_vm ? var.cluster_name : var.external_cluster_name
  master_nodes             = var.create_fyre_vm ? local.master_node_fqdns : [var.external_k3s_master_node]
  worker_nodes             = var.create_fyre_vm ? local.worker_node_fqdns : (length(var.external_k3s_nodes) > 1 ? slice(var.external_k3s_nodes, 1, length(var.external_k3s_nodes)) : [])
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
# Resource 3: Install Rook-Ceph Storage (Optional)
# ============================================================================

resource "guardium-data-protection_rook_ceph_cluster" "this" {
  provider = guardium-data-protection
  count    = var.manage_rook_ceph && var.install_rook_ceph ? 1 : 0

  depends_on = [guardium-data-protection_k3s_cluster.main]

  cluster_name                = local.rook_cluster_name
  platform                    = "k3s"
  target_node                 = local.primary_master
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
# Resource 4: Deploy Edge Components (Optional)
# Finalizer cleanup and namespace termination are handled natively by the
# guardium-data-protection provider via WaitForNamespaceDeletion during destroy.
# ============================================================================

resource "guardium-data-protection_deployment" "edge" {
  provider = guardium-data-protection
  count    = var.manage_edge && var.install_edge ? 1 : 0

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
