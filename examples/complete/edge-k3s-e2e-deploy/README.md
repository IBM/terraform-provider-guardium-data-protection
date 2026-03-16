# Complete End-to-End K3S Deployment on Fyre

This is a **complete end-to-end solution** that combines VM creation, K3S installation, optional Rook-Ceph storage, and optional Edge deployment in a single Terraform deployment. It uses the unified [`terraform-provider-guardium-data-protection`](../../../) provider for all resources.

## What This Does

1. **Creates Fyre VMs** (masters + workers) - **OPTIONAL**
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
| `ibm/guardium-data-protection` | `registry.terraform.io/ibm/guardium-data-protection` | `guardium-data-protection_fyre_vm`<br>`guardium-data-protection_k3s_cluster`<br>`guardium-data-protection_rook_ceph_cluster`<br>`guardium-data-protection_deployment` | Unified provider for:<br>- Fyre VM creation<br>- K3S installation<br>- Rook-Ceph storage<br>- Edge deployment |

## Deployment Modes

### Mode 1: Full Deployment (Default)
Deploy everything from scratch - VMs, K3S cluster, and optionally Rook-Ceph + Edge.

```hcl
create_fyre_vm           = true  # Default - create new Fyre VMs
install_k3s              = true  # Default
install_rook_ceph        = true
install_edge             = true
```

### Mode 2: K3S on Existing VMs (Skip VM Creation)
Skip VM creation but still install K3S on existing Fyre VMs (identified by cluster name).

```hcl
create_fyre_vm           = false  # Skip VM creation, use existing VMs
install_k3s              = true
cluster_name             = "my-existing-cluster"  # Must match existing Fyre cluster
install_rook_ceph        = true
install_edge             = true
```

### Mode 3: Edge/Storage-Only Deployment
Skip VM + K3S deployment and deploy Edge/Rook-Ceph to an existing K3S cluster.

```hcl
create_fyre_vm           = false
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
fyre_user_name   = "your-fyre-username"
fyre_user_apikey = "your-fyre-apikey"
ssh_password     = "your-fyre-root-password"

# Cluster
cluster_name = "my-k3s-cluster"

# Single node (default)
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

worker_nodes = []

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
ssh root@<cluster-name>-master1.fyre.ibm.com

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
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   fyre_vm    │─▶│ k3s_cluster  │─▶│  rook_ceph   │─▶│  gdp_edge    │
│  (Provider)  │  │  (Provider)  │  │  (Optional)  │  │  (Optional)  │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
        │                │                │                    │
        ▼                ▼                ▼                    ▼
  Create VMs      Install K3S      Storage Layer      Edge Components
```

## Configuration Examples

### Single Node Cluster

```hcl
cluster_name = "single-node"

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

worker_nodes = []
```

### 3 Worker Cluster

```hcl
cluster_name = "multi-node"

master_nodes = [
  {
    name                 = "master1"
    count                = 1
    cpu                  = 8
    memory               = 16
    os                   = "rhel9"
    additional_disk_size = 500
  }
]

worker_nodes = [
  {
    name                 = "worker1"
    count                = 1
    cpu                  = 16
    memory               = 64
    os                   = "rhel9"
    additional_disk_size = 1000
  },
  {
    name                 = "worker2"
    count                = 1
    cpu                  = 16
    memory               = 64
    os                   = "rhel9"
    additional_disk_size = 1000
  },
  {
    name                 = "worker3"
    count                = 1
    cpu                  = 16
    memory               = 64
    os                   = "rhel9"
    additional_disk_size = 1000
  }
]
```

### Enable Rook-Ceph Storage

```hcl
manage_rook_ceph            = true   # Terraform manages Rook-Ceph
install_rook_ceph           = true   # Install it
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
  "existing-master.fyre.ibm.com",
  "existing-worker1.fyre.ibm.com",
  "existing-worker2.fyre.ibm.com"
]
external_k3s_master_node = "existing-master.fyre.ibm.com"
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
external_k3s_master_node    = "existing-master.fyre.ibm.com"
external_worker_count       = 3
ssh_password                = "your-ssh-password"

manage_rook_ceph            = true
install_rook_ceph           = true
rook_ceph_version           = "v1.15.4"
rook_ceph_installation_path = "/path/to/rook-ceph-files"
```

### Retry Edge Installation (Keep Existing K3S + Rook-Ceph)

When K3S and Rook-Ceph are already installed, but Edge deployment failed:

```hcl
install_k3s              = false
manage_rook_ceph         = false  # Don't manage existing Rook-Ceph
install_rook_ceph        = false  # Not creating new Rook-Ceph
external_k3s_nodes       = ["cluster-master1.fyre.ibm.com"]
external_k3s_master_node = "cluster-master1.fyre.ibm.com"
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
manage_rook_ceph            = true
install_rook_ceph           = true
rook_ceph_installation_path = "/path/to/rook-ceph-files"

# Enable Edge
install_edge     = true
edge_name        = "my-edge"
edge_cm_url      = "https://guardium-insights.example.com"
edge_oauth_token = "your-oauth-token"
```

## Rook-Ceph Management

### Understanding `manage_rook_ceph` vs `install_rook_ceph`

The deployment uses two variables to control Rook-Ceph behavior:

```hcl
resource "rook_ceph_cluster" "this" {
  count = var.manage_rook_ceph && var.install_rook_ceph ? 1 : 0
  ...
}
```

| manage_rook_ceph | install_rook_ceph | Result |
|------------------|-------------------|--------|
| `true` | `true` | ✅ Rook-Ceph installed/managed by Terraform |
| `true` | `false` | ❌ Rook-Ceph destroyed (if exists) |
| `false` | `true` | ❌ No action (count=0, Terraform ignores it) |
| `false` | `false` | ❌ No action (count=0, Terraform ignores it) |

### Troubleshooting Rook-Ceph

**Q: I set `install_rook_ceph = false` but Terraform wants to destroy it**

A: You need to also set `manage_rook_ceph = false` to tell Terraform to stop managing it.

**Q: Can I use `terraform destroy` with `manage_rook_ceph = false`?**

A: Yes. Since the resource isn't in Terraform's plan (count = 0), `terraform destroy` will ignore it and only destroy resources Terraform is managing.

**Q: What if I want to re-enable Terraform management later?**

A: You'll need to import the resource back into state:
```bash
terraform import rook_ceph_cluster.this[0] <cluster-name>
```
Then set `manage_rook_ceph = true` and `install_rook_ceph = true`.

**Q: Do I need both variables set to true to install Rook-Ceph?**

A: Yes. Both `manage_rook_ceph = true` AND `install_rook_ceph = true` are required because the resource uses AND logic (`&&`).

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

### Beta-Fyre Platform

```hcl
fyre_cluster_type = "beta-fyre"
# Domain suffix will automatically be dev.fyre.ibm.com
```

### Different Product Group

```hcl
fyre_product_group_id = "413"  # Use different quota
```

## Troubleshooting

### VM Creation Failed

Check Fyre quota and increase polling timeout:
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

This will remove all resources in reverse order: Edge -> Rook-Ceph -> K3S -> Fyre VMs.

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

This deployment uses custom providers from IBM Artifactory:
- [`terraform-provider-fyre`](../terraform-provider-fyre-native/) - VM creation
- [`terraform-provider-k3s`](../terraform-provider-k3s-native/) - K3S installation
- [`terraform-provider-rook-ceph`](../terraform-provider-rook-ceph-native/) - Rook-Ceph storage
- [`terraform-provider-gdp-edge`](../terraform-provider-gdp-edge-native/) - Edge deployment

See the individual provider directories for setup instructions (Artifactory mirror configuration).
