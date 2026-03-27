# Copyright (c) IBM Corporation
# SPDX-License-Identifier: Apache-2.0

# Complete OCP Edge Deployment Outputs

# ============================================================================
# Cluster Outputs
# ============================================================================

output "cluster_name" {
  description = "Name of the OpenShift cluster"
  value       = var.cluster_name
}

output "ocp_infra_node_hostname" {
  description = "OCP infrastructure node hostname"
  value       = var.ocp_infra_node_hostname
}

output "ocp_api_server" {
  description = "OCP API server URL"
  value       = var.ocp_api_server
}

output "worker_node_count" {
  description = "Number of worker nodes"
  value       = var.worker_node_count
}

# ============================================================================
# Rook-Ceph Outputs (if installed)
# ============================================================================

output "rook_ceph_installed" {
  description = "Whether Rook-Ceph is installed"
  value       = var.install_rook_ceph
}

output "rook_ceph_version" {
  description = "Installed Rook-Ceph version (if installed)"
  value       = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].rook_ceph_version : null
}

output "rook_ceph_namespace" {
  description = "Kubernetes namespace where Rook-Ceph is installed"
  value       = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].namespace : null
}

output "rook_ceph_cluster_type" {
  description = "Rook-Ceph cluster type: test or production"
  value       = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].cluster_type : null
}

output "rook_ceph_cephfs_storage_class" {
  description = "CephFS storage class name (if installed)"
  value       = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].cephfs_storage_class : null
}

output "rook_ceph_block_storage_class" {
  description = "RBD block storage class name (if installed)"
  value       = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].block_storage_class : null
}

output "rook_ceph_summary" {
  description = "Rook-Ceph installation summary (if installed)"
  value = var.install_rook_ceph ? join("\n", [
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
  value       = var.install_edge ? guardium-data-protection_edge_deploy.edge[0].edge_namespace : null
}

output "edge_registry_url" {
  description = "Container registry URL used by the Edge deployment (if installed)"
  value       = var.install_edge ? guardium-data-protection_edge_deploy.edge[0].registry_url : null
}

output "edge_deployment_status" {
  description = "Edge deployment status message (if installed)"
  value       = var.install_edge ? guardium-data-protection_edge_deploy.edge[0].deployment_status : null
}

output "edge_work_dir" {
  description = "Working directory for the edge bundle (if installed)"
  value       = var.install_edge ? guardium-data-protection_edge_deploy.edge[0].work_dir : null
}

output "edge_summary" {
  description = "Edge deployment summary (if installed)"
  value = var.install_edge ? join("\n", [
    "Edge Deployment Summary:",
    "  Namespace: ${guardium-data-protection_edge_deploy.edge[0].edge_namespace}",
    "  Platform:  ${guardium-data-protection_edge_deploy.edge[0].platform}",
    "  Status:    ${guardium-data-protection_edge_deploy.edge[0].deployment_status}",
    "",
    "To check status:",
    "  kubectl get configmap edge-controller-client-cm -n ${guardium-data-protection_edge_deploy.edge[0].edge_namespace} -o yaml",
    "  kubectl get pods -n ${guardium-data-protection_edge_deploy.edge[0].edge_namespace}",
  ]) : null
}

# ============================================================================
# Combined Summary
# ============================================================================

output "deployment_summary" {
  description = "Complete deployment summary"
  value = {
    cluster_name   = var.cluster_name
    inf_node       = var.ocp_infra_node_hostname
    api_server     = var.ocp_api_server
    worker_count   = var.worker_node_count
    rook_ceph      = var.install_rook_ceph
    edge           = var.install_edge
  }
}

output "access_instructions" {
  description = "Instructions for accessing the cluster"
  value = <<-EOT
    OpenShift Cluster Access:

    1. API Server: ${var.ocp_api_server}
    2. Infrastructure Node: ${var.ocp_infra_node_hostname}
    
    To access the cluster:
      oc login ${var.ocp_api_server} -u ${var.ocp_admin_user}
      
    Or set KUBECONFIG:
      export KUBECONFIG=/path/to/kubeconfig
      kubectl get nodes
  EOT
}

output "next_steps" {
  description = "Next steps after deployment"
  value = join("\n", concat(
    [
      "Deployment Complete!",
      "",
      "Cluster: ${var.cluster_name}",
      "API Server: ${var.ocp_api_server}",
      "Inf Node: ${var.ocp_infra_node_hostname}",
      ""
    ],
    var.install_rook_ceph ? [
      "✓ Rook-Ceph installed",
      "  Check storage: kubectl get storageclass",
      ""
    ] : [],
    var.install_edge ? [
      "✓ Edge deployed",
      "  Check status: kubectl get pods -n ${guardium-data-protection_edge_deploy.edge[0].edge_namespace}",
      ""
    ] : []
  ))
}
