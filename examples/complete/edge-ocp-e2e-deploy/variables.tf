# Complete OCP Edge Deployment Variables
# Deploys Rook-Ceph and Edge components to existing OpenShift clusters

# ============================================================================
# Deployment Control
# ============================================================================

variable "install_rook_ceph" {
  description = "Whether to install Rook-Ceph storage"
  type        = bool
  default     = false
}

variable "install_edge" {
  description = "Whether to install Edge components"
  type        = bool
  default     = false
}

# ============================================================================
# OCP Cluster Configuration
# ============================================================================

variable "cluster_name" {
  description = "Name of the OpenShift cluster"
  type        = string
  default     = "edge-ocp"
}

variable "ocp_infra_node_hostname" {
  description = "OCP infrastructure node hostname (for SSH access)"
  type        = string
  default     = ""

  validation {
    condition     = length(var.ocp_infra_node_hostname) > 0
    error_message = "OCP infrastructure node hostname must be specified."
  }
}

variable "ocp_api_server" {
  description = "OCP API server URL (e.g., https://api.cluster.example.com:6443)"
  type        = string
  default     = ""

  validation {
    condition     = length(var.ocp_api_server) > 0
    error_message = "OCP API server URL must be specified."
  }
}

variable "worker_node_count" {
  description = "Number of worker nodes in the cluster (for Rook-Ceph)"
  type        = number
  default     = 3
}

# ============================================================================
# OCP Authentication
# ============================================================================

variable "ocp_ssh_user" {
  description = "SSH username for OCP nodes (typically 'core' for CoreOS)"
  type        = string
  default     = "core"
}

variable "ocp_ssh_password" {
  description = "SSH password for OCP nodes"
  type        = string
  default     = ""
  sensitive   = true
}

variable "ocp_admin_user" {
  description = "OCP admin username (typically 'kubeadmin')"
  type        = string
  default     = "kubeadmin"
}

variable "ocp_admin_password" {
  description = "OCP admin password"
  type        = string
  default     = ""
  sensitive   = true
}

variable "ocp_token" {
  description = "OCP authentication token (alternative to username/password)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "ocp_insecure_skip_verify" {
  description = "Skip TLS certificate verification for OCP API"
  type        = bool
  default     = true
}

# ============================================================================
# SSH Configuration
# ============================================================================

variable "ssh_options" {
  description = "SSH connection options"
  type = object({
    connect_timeout       = optional(number, 30)
    server_alive_interval = optional(number, 10)
    server_alive_count    = optional(number, 3)
  })
  default = {}
}

# ============================================================================
# Rook-Ceph Storage Configuration (Optional)
# ============================================================================

variable "rook_ceph_version" {
  description = "Rook-Ceph version to install"
  type        = string
  default     = "v1.15.4"
}

variable "rook_ceph_airgap_installation_path" {
  description = "Rook-Ceph installation local directory"
  type        = string
  default     = ""
}

variable "rook_ceph_airgap_install" {
  description = "Enable airgap installation for Rook-Ceph"
  type        = bool
  default     = true
}

variable "rook_ceph_config" {
  description = "Rook-Ceph installation configuration"
  type = object({
    set_as_default_storage = optional(bool, false)
    pod_wait_timeout       = optional(string, "600s")
    sleep_between_steps    = optional(number, 60)
  })
  default = {}
}

# ============================================================================
# Edge Deployment Configuration (Optional)
# ============================================================================

variable "edge_name" {
  description = "Name for the Edge deployment"
  type        = string
  default     = ""
}

variable "edge_cm_url" {
  description = "URL of the Guardium Insights Central Manager (for downloading edge bundle)"
  type        = string
  default     = ""
}

variable "edge_oauth_token" {
  description = "OAuth token for authenticating with Central Manager"
  type        = string
  default     = ""
  sensitive   = true
}

variable "edge_bundle_directory" {
  description = "Local path to edge bundle directory (alternative to downloading from CM)"
  type        = string
  default     = ""
}

variable "edge_monitor_max_attempts" {
  description = "Maximum number of attempts to monitor edge deployment status"
  type        = number
  default     = 60
}

variable "edge_monitor_sleep_interval" {
  description = "Sleep interval in seconds between edge deployment status checks"
  type        = number
  default     = 30
}

variable "edge_cleanup_bundle" {
  description = "Whether to cleanup edge bundle after deployment"
  type        = bool
  default     = true
}

variable "external_image_registry" {
  description = "Set to true when using an external image registry (e.g. Docker Hub, Quay) instead of the CM private registry. Skips registry certificate installation."
  type        = bool
  default     = false
}

# ============================================================================
# Timeout Configuration
# ============================================================================

variable "rook_ceph_delete_timeout" {
  description = "Timeout for Rook-Ceph deletion"
  type        = string
  default     = "30m"
}

variable "edge_delete_timeout" {
  description = "Timeout for Edge deployment deletion"
  type        = string
  default     = "30m"
}

variable "ocp_machineconfig_timeout" {
  description = "Timeout for OpenShift MachineConfig rollout during certificate installation. Increase for large clusters or slow node updates."
  type        = string
  default     = "30m"
}
