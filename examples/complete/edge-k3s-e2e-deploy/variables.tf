# Complete K3S Edge Deployment Variables
# Deploys K3S, Rook-Ceph, and Edge components to existing clusters

# ============================================================================
# Deployment Control
# ============================================================================

variable "install_k3s" {
  description = "Whether to deploy K3S cluster (set to false to use existing cluster)"
  type        = bool
  default     = true
}

# ============================================================================
# SSH Configuration
# ============================================================================

variable "ssh_user" {
  description = "SSH username for connecting to nodes"
  type        = string
  default     = "root"
}

variable "ssh_password" {
  description = "SSH password for connecting to nodes"
  type        = string
  default     = ""
  sensitive   = true
}

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
# Cluster Configuration
# ============================================================================

variable "cluster_name" {
  description = "Name for the K3S cluster"
  type        = string
  default     = "k3s-cluster"

  validation {
    condition     = length(var.cluster_name) > 0
    error_message = "Cluster name must not be empty."
  }
}

variable "k3s_nodes" {
  description = "List of K3S node hostnames (first node is treated as master)"
  type        = list(string)
  default     = []

  validation {
    condition     = length(var.k3s_nodes) > 0
    error_message = "At least one K3S node must be specified."
  }
}

variable "k3s_master_node" {
  description = "K3S master node hostname (typically the first node in k3s_nodes)"
  type        = string
  default     = ""

  validation {
    condition     = length(var.k3s_master_node) > 0
    error_message = "K3S master node must be specified."
  }
}

# ============================================================================
# K3S Configuration
# ============================================================================

variable "k3s_version" {
  description = "K3S version to install (e.g., v1.32.3, v1.33.1)"
  type        = string
  default     = "v1.32.3"

  validation {
    condition     = can(regex("^v[0-9]+\\.[0-9]+\\.[0-9]+$", var.k3s_version))
    error_message = "K3S version must be in format vX.Y.Z (e.g., v1.32.3)."
  }
}

variable "k3s_token" {
  description = "Token for K3S cluster authentication"
  type        = string
  default     = "edge1234"
  sensitive   = true
}

variable "k3s_install_options" {
  description = "K3S installation options"
  type = object({
    disable_traefik   = bool
    taint_masters     = bool
    node_wait_timeout = string
  })
  default = {
    disable_traefik   = true
    taint_masters     = true
    node_wait_timeout = "600s"
  }
}

variable "k3s_airgap_install" {
  description = "Enable airgap installation for K3S"
  type        = bool
  default     = true
}

variable "k3s_airgap_installation_path" {
  description = "Local path to K3S airgap installation binary files"
  type        = string
  default     = ""
}

# ============================================================================
# Rook-Ceph Storage Configuration (Optional)
# ============================================================================

variable "install_rook_ceph" {
  description = "Whether to install Rook-Ceph storage"
  type        = bool
  default     = false
}

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
    set_as_default_storage = optional(bool, true)
    disable_local_path     = optional(bool, true)
    pod_wait_timeout       = optional(string, "600s")
    sleep_between_steps    = optional(number, 60)
  })
  default = {}
}

# ============================================================================
# Edge Deployment Configuration (Optional)
# ============================================================================

variable "install_edge" {
  description = "Whether to install Edge components"
  type        = bool
  default     = false
}

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

variable "k3s_delete_timeout" {
  description = "Timeout for K3S cluster deletion"
  type        = string
  default     = "30m"
}

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
