# Complete K3S Deployment Variables - Custom Provider Edition
# Uses custom Terraform providers (fyre, k3s, rook-ceph, gdp-edge)

# ============================================================================
# Deployment Control
# ============================================================================

variable "create_fyre_vm" {
  description = "Whether to create Fyre VMs (set to false to skip VM creation and use existing VMs)"
  type        = bool
  default     = false
}

variable "install_k3s" {
  description = "Whether to deploy K3S cluster (set to false to use existing cluster)"
  type        = bool
  default     = true
}

variable "manage_k3s" {
  description = "Whether to manage K3S with Terraform (set to false to keep existing cluster unmanaged when install_k3s is set to false)"
  type        = bool
  default     = true
}

variable "external_k3s_nodes" {
  description = "List of external K3S node hostnames (required if install_k3s=false)"
  type        = list(string)
  default     = []
}

variable "external_k3s_master_node" {
  description = "External K3S master node hostname (required if install_k3s=false)"
  type        = string
  default     = ""
}

variable "external_cluster_name" {
  description = "External cluster name (required if install_k3s=false and install_rook_ceph=true)"
  type        = string
  default     = ""
}

variable "external_worker_count" {
  description = "Number of worker nodes in external cluster (required if install_k3s=false and install_rook_ceph=true)"
  type        = number
  default     = 0
}

# ============================================================================
# Fyre Credentials
# ============================================================================

variable "fyre_user_name" {
  description = "Fyre username for authentication"
  type        = string
  default     = ""
  sensitive   = true
}

variable "fyre_user_apikey" {
  description = "Fyre API key for authentication"
  type        = string
  default     = ""
  sensitive   = true
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
  description = "SSH password for root user on Fyre VMs"
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
  description = "Unique name for the Fyre cluster"
  type        = string
  default     = "qb-test"

  validation {
    condition     = length(var.cluster_name) > 0
    error_message = "Cluster name must not be empty."
  }
}

variable "fyre_cluster_type" {
  description = "Fyre platform type (fyre or beta-fyre)"
  type        = string
  default     = "fyre"

  validation {
    condition     = contains(["fyre", "beta-fyre"], var.fyre_cluster_type)
    error_message = "Cluster type must be either 'fyre' or 'beta-fyre'."
  }
}

variable "fyre_product_group_id" {
  description = "Fyre product group ID - ensure you have quota in this group"
  type        = string
  default     = "180"

  validation {
    condition     = contains(["180", "413", "676", "310", "746", "691", "455"], var.fyre_product_group_id)
    error_message = "Product group ID must be one of: 180, 413, 676, 310, 746, 691, 455."
  }
}

# ============================================================================
# Node Configuration
# ============================================================================

variable "master_nodes" {
  description = "Configuration for master nodes"
  type = list(object({
    name                 = string
    count                = number
    cpu                  = number
    memory               = number
    os                   = string
    additional_disk_size = number
  }))
  default = [
    {
      name                 = "master1"
      count                = 1
      cpu                  = 16
      memory               = 64
      os                   = "rhel9"
      additional_disk_size = 1000
    }
  ]
}

variable "worker_nodes" {
  description = "Configuration for worker nodes (empty list for single-node cluster)"
  type = list(object({
    name                 = string
    count                = number
    cpu                  = number
    memory               = number
    os                   = string
    additional_disk_size = number
  }))
  default = []
}

variable "cluster_config" {
  description = "General cluster configuration"
  type = object({
    platform = string
  })
  default = {
    platform = "x"
  }
}

variable "network_config" {
  description = "Network configuration for nodes"
  type = object({
    public_vlan  = string
    private_vlan = string
    dns          = string
  })
  default = {
    public_vlan  = "y"
    private_vlan = "y"
    dns          = ""
  }
}

variable "polling_timeout_minutes" {
  description = "Maximum time in minutes to wait for VM creation"
  type        = number
  default     = 60
}

variable "polling_interval_seconds" {
  description = "Interval in seconds between polling attempts for VM creation status"
  type        = number
  default     = 30
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
variable "manage_rook_ceph" {
  description = "Whether to manage Rook-Ceph with Terraform (set to false to keep existing installation unmanaged)"
  type        = bool
  default     = true
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

variable "manage_edge" {
  description = "Whether to manage Edge deployment with Terraform (set to false to keep existing installation unmanaged during destroy)"
  type        = bool
  default     = true
}

variable "edge_name" {
  description = "Name of the edge to deploy (required if downloading bundle from CM)"
  type        = string
  default     = ""
}

variable "edge_bundle_directory" {
  description = "Path to local edge bundle directory. If empty, bundle will be downloaded from CM."
  type        = string
  default     = ""
}

variable "platform" {
  description = "Edge deploy target platform"
  type        = string
  default     = ""
}

variable "edge_cm_url" {
  description = "Guardium Insights Central Manager URL"
  type        = string
  default     = ""
}

variable "edge_oauth_token" {
  description = "OAuth token for CM authentication"
  type        = string
  default     = ""
  sensitive   = true
}

variable "edge_monitor_max_attempts" {
  description = "Maximum polling attempts for edge deployment monitoring (default: 180 = ~30 min with 10s interval)"
  type        = number
  default     = 180
}

variable "edge_monitor_sleep_interval" {
  description = "Sleep interval in seconds between edge monitoring polls"
  type        = number
  default     = 10
}

variable "edge_cleanup_bundle" {
  description = "Whether to cleanup downloaded edge bundle on destroy"
  type        = bool
  default     = true
}

variable "external_image_registry" {
  description = "Set to true when using an external image registry (e.g. Docker Hub, Quay) instead of the CM private registry. Skips registry certificate installation on cluster nodes."
  type        = bool
  default     = false
}

# ============================================================================
# Destroy Timeouts
# ============================================================================

variable "fyre_vm_delete_timeout" {
  description = "Timeout for destroying the Fyre VM cluster (e.g. '2h', '90m')"
  type        = string
  default     = "2h"
}

variable "k3s_delete_timeout" {
  description = "Timeout for uninstalling the K3S cluster (e.g. '2h', '90m')"
  type        = string
  default     = "2h"
}

variable "rook_ceph_delete_timeout" {
  description = "Timeout for uninstalling Rook-Ceph (e.g. '2h', '90m')"
  type        = string
  default     = "2h"
}

variable "edge_delete_timeout" {
  description = "Timeout for deleting the Edge deployment (e.g. '2h', '90m')"
  type        = string
  default     = "2h"
}
