# Complete OCP Fyre Edge Deployment Outputs - Consolidated Provider Edition

# ============================================================================
# OCP Cluster Outputs (if deployed)
# ============================================================================

output "cluster_id" {
  description = "The ID of the OCP cluster"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].id : null
}

output "cluster_name" {
  description = "Name of the deployed OCP cluster"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].name : var.cluster_name
}

output "ocp_version" {
  description = "OpenShift version deployed"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].ocp_version : null
}

output "platform" {
  description = "Platform type (x86, Power, or Z)"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].platform : null
}

output "site" {
  description = "Fyre site location"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].site : null
}

output "cluster_url" {
  description = "URL to access the cluster in Fyre console"
  value       = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].cluster_url : null
}

output "ocp_inf_hostname" {
  description = "OCP infrastructure node hostname"
  value       = local.ocp_inf_hostname
}

output "cluster_summary" {
  description = "Summary of cluster configuration"
  value = var.deploy_openshift ? {
    name        = guardium-data-protection_fyre_ocp.cluster[0].name
    description = guardium-data-protection_fyre_ocp.cluster[0].description
    ocp_version = guardium-data-protection_fyre_ocp.cluster[0].ocp_version
    platform    = guardium-data-protection_fyre_ocp.cluster[0].platform
    site        = guardium-data-protection_fyre_ocp.cluster[0].site
    fips        = guardium-data-protection_fyre_ocp.cluster[0].fips
    cluster_url = guardium-data-protection_fyre_ocp.cluster[0].cluster_url
    master = {
      count  = var.master_node_count
      cpu    = var.master_node_cpu
      memory = var.master_node_memory
    }
    worker = {
      count  = var.worker_node_count
      cpu    = var.worker_node_cpu
      memory = var.worker_node_memory
    }
  } : null
}

output "master_nodes" {
  description = "Master node configuration"
  value = {
    count  = var.master_node_count
    cpu    = var.master_node_cpu
    memory = var.master_node_memory
  }
}

output "worker_nodes" {
  description = "Worker node configuration"
  value = {
    count  = var.worker_node_count
    cpu    = var.worker_node_cpu
    memory = var.worker_node_memory
  }
}

output "access_instructions" {
  description = "Instructions for accessing the cluster"
  value = var.deploy_openshift ? join("\n", [
    "OCP Cluster Created Successfully!",
    "",
    "  Cluster Name: ${guardium-data-protection_fyre_ocp.cluster[0].name}",
    "  OCP Version:  ${guardium-data-protection_fyre_ocp.cluster[0].ocp_version}",
    "  Platform:     ${guardium-data-protection_fyre_ocp.cluster[0].platform}",
    "  Site:         ${guardium-data-protection_fyre_ocp.cluster[0].site}",
    "  Inf Node:     ${local.ocp_inf_hostname}",
    "",
    "  Access your cluster:",
    "  1. Visit: ${guardium-data-protection_fyre_ocp.cluster[0].cluster_url}",
    "  2. Download kubeconfig from Fyre console",
    "  3. Set KUBECONFIG environment variable",
    "  4. Run: kubectl get nodes",
    "",
    "  Master Nodes: ${var.master_node_count} x ${var.master_node_cpu} CPU, ${var.master_node_memory}GB RAM",
    "  Worker Nodes: ${var.worker_node_count} x ${var.worker_node_cpu} CPU, ${var.worker_node_memory}GB RAM",
  ]) : "Using existing cluster: ${var.cluster_name}"
}

# ============================================================================
# Rook-Ceph Outputs (if installed)
# ============================================================================

output "rook_ceph_installed" {
  description = "Whether Rook-Ceph was installed"
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
  description = "Rook-Ceph cluster type"
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
  description = "Whether Edge components were installed"
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

# ============================================================================
# Deployment Summary
# ============================================================================

output "deployment_summary" {
  description = "Complete deployment summary"
  value = {
    cluster = {
      name        = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].name : var.cluster_name
      ocp_version = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].ocp_version : null
      platform    = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].platform : null
      site        = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].site : null
      fips        = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].fips : null
      inf_node    = local.ocp_inf_hostname
    }
    components = {
      rook_ceph_installed = var.install_rook_ceph
      rook_ceph_version   = var.install_rook_ceph ? guardium-data-protection_rook_ceph_cluster.this[0].rook_ceph_version : null
      edge_installed      = var.install_edge
      edge_name           = var.install_edge ? var.edge_name : null
      edge_status         = var.install_edge ? guardium-data-protection_deployment.edge[0].deployment_status : null
    }
    access = {
      cluster_url = var.deploy_openshift ? guardium-data-protection_fyre_ocp.cluster[0].cluster_url : null
      inf_node    = local.ocp_inf_hostname
      api_server  = local.ocp_api_server
    }
  }
}

# ============================================================================
# Next Steps
# ============================================================================

output "next_steps" {
  description = "Next steps after deployment"
  value = var.deploy_openshift ? join("\n", [
    "============================================================",
    "Deployment Complete!",
    "============================================================",
    "",
    "Cluster: ${guardium-data-protection_fyre_ocp.cluster[0].name}",
    "OCP Version: ${guardium-data-protection_fyre_ocp.cluster[0].ocp_version}",
    "Platform: ${guardium-data-protection_fyre_ocp.cluster[0].platform}",
    "Site: ${guardium-data-protection_fyre_ocp.cluster[0].site}",
    "Inf Node: ${local.ocp_inf_hostname}",
    "",
    "Components Installed:",
    "- Rook-Ceph Storage: ${var.install_rook_ceph ? "Yes (${var.rook_ceph_version})" : "No"}",
    "- Edge Components: ${var.install_edge ? "Yes (${var.edge_name})" : "No"}",
    "",
    "Next Steps:",
    "1. Access Fyre Console: ${guardium-data-protection_fyre_ocp.cluster[0].cluster_url}",
    "2. Download kubeconfig from Fyre interface",
    "3. Configure kubectl:",
    "   export KUBECONFIG=/path/to/kubeconfig",
    "   kubectl get nodes",
    "",
    var.install_rook_ceph ? "4. Verify Rook-Ceph:\n   kubectl get pods -n rook-ceph\n   kubectl get storageclass\n" : "",
    var.install_edge ? "5. Verify Edge Deployment:\n   kubectl get pods -n <edge-namespace>\n   kubectl get all -n <edge-namespace>\n" : "",
    "============================================================",
  ]) : join("\n", [
    "============================================================",
    "Deployment Complete!",
    "============================================================",
    "",
    "Using Existing Cluster: ${var.cluster_name}",
    "",
    "Components Installed:",
    "- Rook-Ceph Storage: ${var.install_rook_ceph ? "Yes (${var.rook_ceph_version})" : "No"}",
    "- Edge Components: ${var.install_edge ? "Yes (${var.edge_name})" : "No"}",
    "",
    var.install_rook_ceph ? "Verify Rook-Ceph:\n   kubectl get pods -n rook-ceph\n   kubectl get storageclass\n" : "",
    var.install_edge ? "Verify Edge Deployment:\n   kubectl get pods -n <edge-namespace>\n   kubectl get all -n <edge-namespace>\n" : "",
    "============================================================",
  ])
}
