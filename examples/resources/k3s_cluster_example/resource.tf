# Install and manage a K3S cluster on remote nodes via SSH
resource "guardium-data-protection_k3s_cluster" "example" {
  # Required parameters
  cluster_name = "my-k3s-cluster"
  master_nodes = ["192.168.1.10"]
  worker_nodes = ["192.168.1.11", "192.168.1.12"] # omit for single-node cluster

  # K3s parameters
  k3s_version = "v1.32.3"  # K3S version to install (default: "v1.32.3")
  k3s_token   = "edge1234" # cluster join token (default: "edge1234")

  # Airgap (offline) installation
  airgap_install           = false                     # set true for air-gapped environments (default: true)
  airgap_installation_path = "/opt/k3s-airgap-install" # local path to K3S installation binaries


  # Optional
  disable_traefik   = true   # disable Traefik ingress controller (default: true)
  taint_masters     = true   # prevent workload scheduling on masters (default: true)
  node_wait_timeout = "600s" # timeout waiting for nodes to become Ready (default: "600s")
  delete_timeout    = "2h"   # timeout for destroy operation (default: "2h")

}
