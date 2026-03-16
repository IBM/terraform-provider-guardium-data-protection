# Complete OCP Fyre Edge Deployment Variables - Consolidated Provider Edition
# Uses consolidated gdp-edge provider for all resources (Fyre OCP, Rook-Ceph, Edge deployment)

# ============================================================================
# Deployment Control
# ============================================================================

variable "deploy_openshift" {
  description = "Whether to deploy OpenShift cluster (set to true to create new cluster via Fyre)"
  type        = bool
  default     = false
}

variable "manage_openshift" {
  description = "Whether to manage OpenShift with Terraform (set to true only when deploy_openshift is true)"
  type        = bool
  default     = false
}

# ============================================================================
# Fyre API Credentials
# ============================================================================

variable "fyre_username" {
  description = "IBM Fyre API username"
  type        = string
  default     = ""
  sensitive   = true
}

variable "fyre_api_key" {
  description = "IBM Fyre API key"
  type        = string
  default     = ""
  sensitive   = true
}

# ============================================================================
# OCP Cluster Configuration
# ============================================================================

variable "cluster_name" {
  description = "Name of the OpenShift cluster"
  type        = string
  default     = "edge-ocp"
}

variable "cluster_description" {
  description = "Description of the OpenShift cluster"
  type        = string
  default     = "OCP Fyre Edge Cluster"
}

variable "ocp_version" {
  description = "OpenShift Container Platform version"
  type        = string
  default     = "4.18.28"
}

variable "ocp_platform" {
  description = "Platform type for the cluster"
  type        = string
  default     = "x"

  validation {
    condition     = contains(["x", "p", "z"], var.ocp_platform)
    error_message = "Platform must be one of: x (x86), p (Power), z (Z Systems)"
  }
}

variable "site" {
  description = "Fyre site location"
  type        = string
  default     = "svl"

  validation {
    condition     = contains(["svl", "rtp", "pok"], var.site)
    error_message = "Site must be one of: svl, rtp, pok"
  }
}

variable "fips_enabled" {
  description = "Enable FIPS mode for the cluster"
  type        = bool
  default     = true
}

variable "ssh_key" {
  description = "SSH public key for cluster access (optional)"
  type        = string
  default     = ""
}

# ============================================================================
# Quota Configuration
# ============================================================================

variable "quota_type" {
  description = "Quota type for the cluster"
  type        = string
  default     = "product_group"

  validation {
    condition     = contains(["product_group", "quick_burn"], var.quota_type)
    error_message = "Quota type must be either 'product_group' or 'quick_burn'"
  }
}

variable "product_group_id" {
  description = "Product group ID for quota allocation"
  type        = string
  default     = "180"
}

variable "time_to_live" {
  description = "Time to live for the cluster in hours"
  type        = string
  default     = "36"
}

# ============================================================================
# Master Node Configuration
# ============================================================================

variable "master_node_count" {
  description = "Number of master nodes"
  type        = number
  default     = 3

  validation {
    condition     = var.master_node_count >= 1 && var.master_node_count <= 5
    error_message = "Master node count must be between 1 and 5"
  }
}

variable "master_node_cpu" {
  description = "Number of CPUs per master node"
  type        = number
  default     = 8

  validation {
    condition     = var.master_node_cpu >= 4 && var.master_node_cpu <= 32
    error_message = "Master node CPU must be between 4 and 32"
  }
}

variable "master_node_memory" {
  description = "Memory per master node in GB"
  type        = number
  default     = 16

  validation {
    condition     = var.master_node_memory >= 8 && var.master_node_memory <= 256
    error_message = "Master node memory must be between 8 and 256 GB"
  }
}

variable "master_additional_disks" {
  description = "List of additional disk sizes in GB for master nodes"
  type        = list(number)
  default     = []
}

# ============================================================================
# Worker Node Configuration
# ============================================================================

variable "worker_node_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3

  validation {
    condition     = var.worker_node_count >= 0 && var.worker_node_count <= 20
    error_message = "Worker node count must be between 0 and 20"
  }
}

variable "worker_node_cpu" {
  description = "Number of CPUs per worker node"
  type        = number
  default     = 16

  validation {
    condition     = var.worker_node_cpu >= 4 && var.worker_node_cpu <= 64
    error_message = "Worker node CPU must be between 4 and 64"
  }
}

variable "worker_node_memory" {
  description = "Memory per worker node in GB"
  type        = number
  default     = 64

  validation {
    condition     = var.worker_node_memory >= 8 && var.worker_node_memory <= 512
    error_message = "Worker node memory must be between 8 and 512 GB"
  }
}

variable "worker_additional_disks" {
  description = "List of additional disk sizes in GB for worker nodes"
  type        = list(number)
  default     = [1000]
}

# ============================================================================
# Cluster Provisioning Options
# ============================================================================

variable "wait_for_cluster" {
  description = "Whether to wait for the OCP cluster to be fully ready"
  type        = bool
  default     = true
}

variable "polling_timeout_minutes" {
  description = "Maximum time in minutes to wait for cluster to be ready"
  type        = number
  default     = 120

  validation {
    condition     = var.polling_timeout_minutes >= 30 && var.polling_timeout_minutes <= 240
    error_message = "Polling timeout must be between 30 and 240 minutes"
  }
}

variable "polling_interval_seconds" {
  description = "Time to wait between status checks in seconds"
  type        = number
  default     = 60

  validation {
    condition     = var.polling_interval_seconds >= 30 && var.polling_interval_seconds <= 300
    error_message = "Poll interval must be between 30 and 300 seconds"
  }
}

# ============================================================================
# OCP Access Configuration (for Rook-Ceph and Edge deployment)
# ============================================================================

variable "ocp_infra_node_hostname" {
  description = "OCP infrastructure node hostname for SSH access (e.g., api.cluster-name.cp.fyre.ibm.com or cluster-name-inf.fyre.ibm.com)"
  type        = string
  default     = ""
}

variable "ocp_ssh_user" {
  description = "SSH username for OCP node access (default: core for OCP)"
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
  description = "OCP admin username for authentication (default: kubeadmin)"
  type        = string
  default     = "kubeadmin"
}

variable "ocp_admin_password" {
  description = "OCP admin password for authentication"
  type        = string
  default     = ""
  sensitive   = true
}

variable "ocp_token" {
  description = "OpenShift OAuth token (alternative to username/password, can be obtained via 'oc whoami -t')"
  type        = string
  default     = ""
  sensitive   = true
}

variable "ocp_insecure_skip_verify" {
  description = "Skip TLS certificate verification for OpenShift API server"
  type        = bool
  default     = true
}

# ============================================================================
# SSH Connection Options
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

variable "install_rook_ceph" {
  description = "Whether to install Rook-Ceph storage on the OCP cluster"
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
  description = "Rook-Ceph configuration options"
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

variable "install_edge" {
  description = "Whether to install Edge components on the OCP cluster"
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

variable "fyre_ocp_delete_timeout" {
  description = "Timeout for destroying the Fyre OCP cluster (e.g. '2h', '90m')"
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
