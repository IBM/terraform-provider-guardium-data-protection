# Complete End-to-End K3S Deployment

This is a **complete end-to-end solution** that combines VM creation, K3S installation, optional Rook-Ceph storage, and optional Edge deployment in a single Terraform deployment. It uses the unified [`terraform-provider-guardium-data-protection`](../../../) provider for all resources.

## What This Does

1. **Creates VMs** (masters + workers) - **OPTIONAL**
2. **Installs K3S** on all nodes via custom provider - **OPTIONAL**
3. **Configures cluster** (HA, taints, airgap, etc.)
4. **Installs Rook-Ceph storage** via custom provider - **OPTIONAL**
5. **Deploys Edge components** via custom provider - **OPTIONAL**
6. **Verifies deployment** (all nodes ready)

All in **one `terraform apply` command**!

## Unified Provider

This deployment uses the unified Guardium Data Protection Terraform provider from IBM Artifactory:

| Provider | Source | Resources | Purpose |
|----------|--------|-----------|---------|
| `ibm/guardium-data-protection` | `hashicorp.com/ibm/guardium-data-protection` | `guardium-data-protection_k3s_cluster`<br>`guardium-data-protection_rook_ceph_cluster`<br>`guardium-data-protection_edge_deploy` | Unified provider for:<br>- K3S installation<br>- Rook-Ceph storage<br>- Edge deployment |

## Deployment Modes

### Mode 1: Full Deployment (Default)
Deploy everything from scratch - VMs, K3S cluster, and optionally Rook-Ceph + Edge.

```hcl
install_k3s              = true  # Default
install_rook_ceph        = true
install_edge             = true
```

### Mode 2: K3S on Existing VMs
Install K3S on existing VMs (identified by cluster name).

```hcl
install_k3s              = true
cluster_name             = "my-existing-cluster"  # Must match existing cluster
install_rook_ceph        = true
install_edge             = true
```

### Mode 3: Edge/Storage-Only Deployment
Deploy Edge/Rook-Ceph to an existing K3S cluster.

```hcl
install_k3s              = false
external_k3s_nodes       = ["node1.example.com", "node2.example.com"]
external_k3s_master_node = "node1.example.com"
install_rook_ceph        = true
install_edge             = true
```

## Quick Start

### 1. Configure

```bash
cd core/container/test-suits/terraform/complete-e2e-k3s-deployment
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
# Credentials
ssh_password     = "your-ssh-password"

# Cluster
cluster_name = "my-k3s-cluster"


# K3S
k3s_version  = "v1.32.3"
k3s_airgap_install = true
```

### 2. Deploy

```bash
terraform init
terraform plan
terraform apply
```

### 3. Access

```bash
# Get access command
terraform output quick_access

# SSH to cluster
ssh root@<cluster-name>-master1.example.com

# Use kubectl
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get nodes
```

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│  complete-e2e-k3s-deployment (Custom Providers)          │
└──────────────────────────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┬──────────────────┐
        │                │                │                  │
        ▼                ▼                ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ k3s_cluster  │─▶│  rook_ceph   │─▶│  gdp_edge    │
│  (Provider)  │  │  (Optional)  │  │  (Optional)  │
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                    │
        ▼                ▼                    ▼
  Install K3S      Storage Layer      Edge Components
```

## Configuration Examples


### Enable Rook-Ceph Storage

```hcl
install_rook_ceph           = true   # Install Rook-Ceph
rook_ceph_version           = "v1.15.4"
rook_ceph_installation_path = "/path/to/rook-ceph-files"
rook_ceph_airgap_install    = true

rook_ceph_config = {
  set_as_default_storage = true
  disable_local_path     = true
  pod_wait_timeout       = "600s"
  sleep_between_steps    = 60
}
```

### Enable Edge Deployment
```hcl
install_edge = true

# Download from CM
# The url may need port for rest api call to get edge bundle
edge_name        = "my-edge-cluster"
edge_cm_url      = "https://guardium-cm.example.com"
edge_oauth_token = "your-oauth-token"

# Or use local bundle
# edge_bundle_directory = "/path/to/edge-bundle"

# Set to true when using an external image registry (e.g. Docker Hub, Quay)
# instead of the CM private registry. Skips registry certificate installation.
# external_image_registry = true
```

### Deploy Edge to Existing K3S Cluster

```hcl
install_k3s              = false
external_k3s_nodes       = [
  "existing-master.example.com",
  "existing-worker1.example.com",
  "existing-worker2.example.com"
]
external_k3s_master_node = "existing-master.example.com"
ssh_password             = "your-ssh-password"

install_edge         = true
edge_name            = "my-edge"
edge_cm_url          = "https://guardium-insights.example.com"
edge_oauth_token     = "your-oauth-token"
edge_cleanup_bundle  = true
```

### Install Rook-Ceph on Existing K3S Cluster

```hcl
install_k3s                 = false
external_cluster_name       = "my-existing-cluster"
external_k3s_master_node    = "existing-master.example.com"
external_worker_count       = 3
ssh_password                = "your-ssh-password"

install_rook_ceph           = true
rook_ceph_version           = "v1.15.4"
rook_ceph_installation_path = "/path/to/rook-ceph-files"
```

### Retry Edge Installation (Keep Existing K3S + Rook-Ceph)

When K3S and Rook-Ceph are already installed, but Edge deployment failed:

```hcl
install_k3s              = false
install_rook_ceph        = false  # Don't install new Rook-Ceph
external_k3s_nodes       = ["cluster-master1.example.com"]
external_k3s_master_node = "cluster-master1.example.com"
install_edge             = true
edge_name                = "my-edge"
edge_bundle_directory    = "/path/to/edge/bundle"
```

**Result**: Terraform will only deploy Edge, leaving K3S and Rook-Ceph untouched.

### Complete Stack (VMs + K3S + Rook-Ceph + Edge)

```hcl
cluster_name = "full-stack"

master_nodes = [
  {
    name                 = "master1"
    count                = 1
    cpu                  = 16
    memory               = 64
    os                   = "rhel9"
    additional_disk_size = 1000
  }
]

worker_nodes = [
  {
    name                 = "worker1"
    count                = 2
    cpu                  = 16
    memory               = 64
    os                   = "rhel9"
    additional_disk_size = 1000
  }
]

# Enable storage
install_rook_ceph           = true
rook_ceph_installation_path = "/path/to/rook-ceph-files"

# Enable Edge
install_edge     = true
edge_name        = "my-edge"
edge_cm_url      = "https://guardium-insights.example.com"
edge_oauth_token = "your-oauth-token"
```

## Rook-Ceph Management

### Understanding `install_rook_ceph`

The deployment uses the `install_rook_ceph` variable to control Rook-Ceph installation:

```hcl
resource "rook_ceph_cluster" "this" {
  count = var.install_rook_ceph ? 1 : 0
  ...
}
```

| install_rook_ceph | Result |
|-------------------|--------|
| `true` | ✅ Rook-Ceph installed and managed by Terraform |
| `false` | ❌ No Rook-Ceph installation (existing installations are left untouched) |

### Troubleshooting Rook-Ceph

**Q: I set `install_rook_ceph = false` but Terraform wants to destroy it**

A: If Rook-Ceph was previously managed by Terraform, setting `install_rook_ceph = false` will cause Terraform to destroy it. To keep an existing installation, remove it from Terraform state first:
```bash
terraform state rm guardium-data-protection_rook_ceph_cluster.this[0]
```

**Q: Can I use `terraform destroy` with `install_rook_ceph = false`?**

A: Yes. Since the resource isn't in Terraform's plan (count = 0), `terraform destroy` will ignore it and only destroy resources Terraform is managing.

**Q: What if I want to re-enable Terraform management later?**

A: You'll need to import the resource back into state:
```bash
terraform import guardium-data-protection_rook_ceph_cluster.this[0] <cluster-name>
```
Then set `install_rook_ceph = true`.

## Outputs

After deployment:

```bash
terraform output
```

Key outputs:
- `deployment_summary` - Complete cluster info
- `access_instructions` - How to access cluster
- `quick_access` - SSH command
- `master_node_fqdns` - All master hostnames
- `worker_node_fqdns` - All worker hostnames
- `rook_ceph_summary` - Rook-Ceph status (if installed)
- `edge_summary` - Edge deployment status (if installed)

## Advanced Usage

### Custom K3S Options

```hcl
k3s_install_options = {
  disable_traefik   = false   # Enable Traefik
  taint_masters     = false   # Allow workloads on masters
  node_wait_timeout = "1200s" # Longer timeout
}
```

### Airgap Installation

```hcl
k3s_airgap_install       = true
airgap_installation_path = "/path/to/k3s-airgap-binaries"
```

## Troubleshooting

### VM Creation Failed

Increase polling timeout if needed:
```hcl
polling_timeout_minutes = 90
```

### K3S Installation Failed

```bash
ssh root@<node-hostname>
systemctl status k3s
journalctl -u k3s -f
```

### Rook-Ceph Issues

```bash
kubectl get pods -n rook-ceph
kubectl get storageclass
kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph status
```

### Edge Deployment Failed

```bash
ssh root@<master-hostname>
kubectl get pods -n <edge-namespace>
kubectl logs -n <edge-namespace> <pod-name>
```

## Cleanup

```bash
terraform destroy
```

This will remove all resources in reverse order: Edge -> Rook-Ceph -> K3S.

**Note**: Resources with `manage_* = false` will not be destroyed.

## File Structure

```
complete-e2e-k3s-deployment/
├── main.tf                    # Provider configs + resource definitions
├── variables.tf               # All variables
├── outputs.tf                 # Combined outputs
├── terraform.tfvars.example   # Example config
├── .gitignore                 # Security patterns
└── README.md                  # This file
```

## Provider Dependencies

This deployment uses the unified Guardium Data Protection Terraform provider which includes:
- K3S cluster installation and management
- Rook-Ceph storage provisioning
- Edge component deployment

See the main provider documentation for setup instructions.
