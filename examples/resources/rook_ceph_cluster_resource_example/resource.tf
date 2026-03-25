# Provider configuration for Rook-Ceph cluster installation
provider "guardium-data-protection" {
  # Rook-Ceph SSH credentials — used for Rook-Ceph installation over SSH
  rook_ceph_ssh_user              = "root"
  rook_ceph_ssh_password          = "your_ssh_password"
  rook_ceph_connect_timeout       = 30
  rook_ceph_server_alive_interval = 10
  rook_ceph_server_alive_count    = 3
}

# Install and manage Rook-Ceph distributed storage on a Kubernetes cluster
resource "guardium-data-protection_rook_ceph_cluster" "example" {
  # Required parameters
  cluster_name = "my-cluster"
  platform     = "target_platform"          # target platform: k3s | openshift
  target_node  = "192.168.1.10" # primary master (K3S) or API node (OpenShift) for SSH operations

  # Optional parameters
  rook_ceph_version      = "v1.15.4" # Rook-Ceph version to install (default: "v1.15.4")
  worker_count           = 2         # number of worker nodes; K3S: 0-1 = test cluster, 2+ = production (default: 0)
  taint_masters          = true      # master nodes are tainted (sets CSI provisioner replicas=1 when worker_count=1) (default: false)
  set_as_default_storage = true      # set rook-cephfs as the default storage class (default: true)
  disable_local_path     = true      # disable K3S local-path storage class (K3S only) (default: true)
  pod_wait_timeout       = "600s"    # timeout waiting for pods to become ready (default: "600s")
  sleep_between_steps    = 60        # seconds to sleep between installation steps (default: 60)
  delete_timeout         = "2h"      # timeout for destroy operation (default: "2h")

  # Airgap (offline) installation
  airgap_install                     = false                          # set true for air-gapped environments (default: true)
  airgap_rook_ceph_installation_path = "/opt/rook-ceph-airgap-yaml"  # root directory of Rook-Ceph YAML manifests for airgap install
}
