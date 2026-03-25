# Provider configuration for edge installation
provider "guardium-data-protection" {
  # Central Manager URL and OAuth token for bundle download
  cm_url      = "https://cm.example.com"
  oauth_token = "your_oauth_token"

  # Target platform: k3s | openshift | eks
  platform = "target_platform"

  # Edge SSH credentials — used for certificate installation and kubeconfig fetch
  ssh_user     = "root"
  ssh_password = "your_ssh_password"
}



# Deploy GDP Edge components to a K3S Kubernetes cluster
resource "guardium-data-protection_deployment" "edge_k3s_example" {
  # Required: bundle source — one of edge_name or edge_bundle_directory must be set
  edge_name             = "my-edge"                    # downloads bundle from Central Manager (requires cm_url + oauth_token in provider)
  edge_bundle_directory = "/path/to/edge-bundle/my-edge" # alternative: use a pre-extracted local bundle directory

  # Required: target platform — k3s | openshift | eks
  platform = "target_platform"

  # K3S: master node hostname/IP used to fetch kubeconfig
  k3s_master_node = "192.168.1.10"

  # K3S: all nodes (master + workers) for certificate installation
  k3s_nodes = ["192.168.1.10", "192.168.1.11", "192.168.1.12"]


  # OpenShift auth — resource-level values take precedence over provider config
  ocp_server               = "https://api.my-cluster.example.com:6443"
  ocp_username             = "admin"
  ocp_password             = "your_ocp_password" # sensitive
  ocp_token                = "your_ocp_token"    # alternative to username/password; sensitive
  ocp_insecure_skip_verify = false               # set true to skip TLS verification

  # Timeout for MachineConfig rollout during certificate installation (default: "30m")
  # Increase for large clusters or slow node updates
  ocp_machineconfig_timeout = "30m"


  # EKS: AWS EKS cluster name
  eks_cluster_name = "my-eks-cluster"

  # EKS: install Kubernetes Metrics Server before deploying Edge (default: false)
  k8s_metrics_server_install = true

  # EKS: airgap installation for Metrics Server (default: false)
  k8s_metrics_server_airgap_install      = false
  k8s_metrics_server_airgap_install_path = "/opt/metrics-server-yaml" # required when airgap install is true


  # External image registry configuration
  external_image_registry = false # set true to skip CM registry cert install (default: false)

  # Optional parameters
  monitor_max_attempts    = 180   # max polling attempts waiting for pods (default: 180)
  monitor_sleep_interval  = 10    # seconds between polls (default: 10)
  cleanup_bundle          = true  # remove downloaded bundle on destroy (default: true)
  delete_timeout          = "2h"  # timeout for destroy operation (default: "2h")

}
