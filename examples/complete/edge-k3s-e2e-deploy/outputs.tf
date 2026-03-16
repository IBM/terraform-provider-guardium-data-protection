# Complete K3S Deployment Outputs - Custom Provider Edition

# ============================================================================
# Fyre VM Outputs (if deployed)
# ============================================================================

output "fyre_cluster_id" {
  description = "The ID of the created Fyre cluster"
  value       = var.create_fyre_vm ? guardium-data-protection_fyre_vm.cluster[0].id : null
}

output "fyre_cluster_name" {
  description = "Name of the Fyre cluster"
  value       = var.create_fyre_vm ? guardium-data-protection_fyre_vm.cluster[0].cluster_name : var.external_cluster_name
}

output "fyre_cluster_hostname" {
  description = "The hostname of the cluster master"
  value       = var.create_fyre_vm ? guardium-data-protection_fyre_vm.cluster[0].cluster_host_name : null
}

output "domain_suffix" {
  description = "Domain suffix for the cluster"
  value       = local.domain_suffix
}

output "master_node_fqdns" {
  description = "List of master node FQDNs"
  value       = local.master_node_fqdns
}

output "worker_node_fqdns" {
  description = "List of worker node FQDNs"
  value       = local.worker_node_fqdns
}

output "all_node_fqdns" {
  description = "List of all node FQDNs"
  value       = local.all_node_fqdns
}

# ============================================================================
# K3S Installation Outputs (if deployed)
# ============================================================================

output "k3s_version" {
  description = "Installed K3S version"
  value       = var.install_k3s? guardium-data-protection_k3s_cluster.main[0].k3s_version : null
}

output "k3s_cluster_type" {
  description = "K3S cluster type (single-node or multi-node)"
  value       = var.install_k3s? guardium-data-protection_k3s_cluster.main[0].cluster_type : null
}

output "k3s_primary_master" {
  description = "Primary K3S master node"
  value       = var.install_k3s? guardium-data-protection_k3s_cluster.main[0].primary_master : var.external_k3s_master_node
}

output "kubeconfig_location" {
  description = "Location of kubeconfig on master node"
  value       = var.install_k3s? guardium-data-protection_k3s_cluster.main[0].kubeconfig_path : "/etc/rancher/k3s/k3s.yaml"
}

# ============================================================================
# Combined Summary
# ============================================================================

output "deployment_summary" {
  description = "Complete deployment summary"
  value = {
    cluster_name   = var.create_fyre_vm ? guardium-data-protection_fyre_vm.cluster[0].cluster_name : var.external_cluster_name
    domain_suffix  = local.domain_suffix
    k3s_version    = var.install_k3s ? guardium-data-protection_k3s_cluster.main[0].k3s_version : null
    master_count   = length(local.master_node_fqdns)
    worker_count   = length(local.worker_node_fqdns)
    total_nodes    = length(local.all_node_fqdns)
    primary_master = local.primary_master
  }
}

output "access_instructions" {
  description = "Instructions to access the K3S cluster"
  value       = <<-EOT
    To access the K3S cluster:

    1. SSH to the primary master node:
       ssh ${var.ssh_user}@${local.primary_master}

    2. Set KUBECONFIG environment variable:
       export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

    3. Verify cluster status:
       kubectl get nodes
       kubectl cluster-info
  EOT
}

output "quick_access" {
  description = "Quick access command"
  value       = "ssh ${var.ssh_user}@${local.primary_master}"
}

# ============================================================================
# Rook-Ceph Outputs (if installed)
# ============================================================================

output "rook_ceph_installed" {
  description = "Whether Rook-Ceph is installed"
  value       = var.manage_rook_ceph && var.install_rook_ceph
}

output "rook_ceph_version" {
  description = "Installed Rook-Ceph version (if installed)"
  value       = var.manage_rook_ceph && var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].rook_ceph_version : null
}

output "rook_ceph_namespace" {
  description = "Kubernetes namespace where Rook-Ceph is installed"
  value       = var.manage_rook_ceph && var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].namespace : null
}

output "rook_ceph_cluster_type" {
  description = "Rook-Ceph cluster type: test or production"
  value       = var.manage_rook_ceph && var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].cluster_type : null
}

output "rook_ceph_cephfs_storage_class" {
  description = "CephFS storage class name (if installed)"
  value       = var.manage_rook_ceph && var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].cephfs_storage_class : null
}

output "rook_ceph_block_storage_class" {
  description = "RBD block storage class name (if installed)"
  value       = var.manage_rook_ceph && var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].block_storage_class : null
}

output "rook_ceph_summary" {
  description = "Rook-Ceph installation summary (if installed)"
  value = var.manage_rook_ceph && var.install_rook_ceph ? join("\n", [
    "Rook-Ceph Installation Summary:",
    "  Cluster:      ${guardium-data-protection_rook_ceph_cluster.this[0].cluster_name}",
    "  Platform:     ${guardium-data-protection_rook_ceph_cluster.this[0].platform}",
    "  Cluster Type: ${guardium-data-protection_rook_ceph_cluster.this[0].cluster_type}",
    "  Version:      ${guardium-data-protection_rook_ceph_cluster.this[0].rook_ceph_version}",
    "  Namespace:    ${guardium-data-protection_rook_ceph_cluster.this[0].namespace}",
    "  CephFS SC:    ${guardium-data-protection_rook_ceph_cluster.this[0].cephfs_storage_class}",
    "  Block SC:     ${guardium-data-protection_rook_ceph_cluster.this[0].block_storage_class}",
  ]) : null
}

# ============================================================================
# Edge Deployment Outputs (if installed)
# ============================================================================

output "edge_installed" {
  description = "Whether Edge is installed"
  value       = var.install_edge
}

output "edge_namespace" {
  description = "Kubernetes namespace where Edge components are deployed (if installed)"
  value       = var.install_edge ? guardium-data-protection_deployment.edge[0].edge_namespace : null
}

output "edge_registry_url" {
  description = "Container registry URL used by the Edge deployment (if installed)"
  value       = var.install_edge ? guardium-data-protection_deployment.edge[0].registry_url : null
}

output "edge_deployment_status" {
  description = "Edge deployment status message (if installed)"
  value       = var.install_edge ? guardium-data-protection_deployment.edge[0].deployment_status : null
}

output "edge_work_dir" {
  description = "Working directory for the edge bundle (if installed)"
  value       = var.install_edge ? guardium-data-protection_deployment.edge[0].work_dir : null
}

output "edge_summary" {
  description = "Edge deployment summary (if installed)"
  value = var.install_edge ? join("\n", [
    "Edge Deployment Summary:",
    "  Namespace: ${guardium-data-protection_deployment.edge[0].edge_namespace}",
    "  Platform:  ${guardium-data-protection_deployment.edge[0].platform}",
    "  Status:    ${guardium-data-protection_deployment.edge[0].deployment_status}",
    "",
    "To check status:",
    "  kubectl get configmap edge-controller-client-cm -n ${guardium-data-protection_deployment.edge[0].edge_namespace} -o yaml",
    "  kubectl get pods -n ${guardium-data-protection_deployment.edge[0].edge_namespace}",
  ]) : null
}
