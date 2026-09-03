# Complete End-to-End OCP Edge Deployment

This Terraform configuration provides a complete end-to-end deployment solution for OpenShift Container Platform (OCP), with optional Rook-Ceph storage and Edge components. It uses the unified [`terraform-provider-guardium-data-protection`](../../../) provider for all resources.

## What This Does

1. **Creates OCP cluster** via consolidated provider - **OPTIONAL**
2. **Monitors cluster readiness** with configurable polling
3. **Installs Rook-Ceph storage** via consolidated provider - **OPTIONAL**
4. **Deploys Edge components** via consolidated provider with native OCP OAuth - **OPTIONAL**
5. **Verifies deployment** (all components ready)

All in **one `terraform apply` command**!

## Unified Provider

| Provider | Source | Resources | Purpose |
|----------|--------|-----------|---------|
| `ibm/guardium-data-protection` | `hashicorp.com/ibm/guardium-data-protection` | `guardium-data-protection_rook_ceph_cluster`<br>`guardium-data-protection_edge_deploy` | Unified provider for:<br>- Rook-Ceph storage<br>- Edge deployment |

## Deployment Modes

### Mode 1: Full Deployment (Default)
Deploy everything from scratch - OCP cluster and optionally Rook-Ceph and Edge.

```hcl
manage_openshift = true   # Default - manage cluster with Terraform
deploy_openshift = true   # Default - create new cluster
```

### Mode 2: Components-Only Deployment
Skip OpenShift creation and deploy Edge/Rook-Ceph to an existing cluster without Terraform managing it.

```hcl
manage_openshift        = false  # Don't manage existing cluster
deploy_openshift        = false  # Don't create new cluster
ocp_infra_node_hostname = "existing-ocp-inf.example.com"
install_edge            = true
```

## OpenShift Cluster Management

This configuration uses a two-variable pattern (similar to Rook-Ceph) to control OpenShift cluster lifecycle:

| manage_openshift | deploy_openshift | Behavior |
|------------------|------------------|----------|
| `true` (default) | `true` (default) | ✅ Create and manage new cluster |
| `true` | `false` | ❌ No cluster created (count = 0) |
| `false` | `true` | ❌ No cluster created (count = 0) |
| `false` | `false` | ✅ Use existing cluster, unmanaged by Terraform |

**Key Benefits:**
- **Prevents accidental destruction**: Setting `deploy_openshift = false` alone won't destroy an existing managed cluster
- **Safe transitions**: Change both variables together to transition from managed to unmanaged state
- **Backward compatible**: Default values maintain existing behavior

## Quick Start

### 1. Configure

```bash
cd core/container/test-suits/terraform/complete-e2e-ocp-deployment
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
# Cluster
cluster_name = "my-ocp-cluster"
ocp_version  = "4.18.28"

# Optional components
install_rook_ceph = false
install_edge      = false
```

**SSH host key verification (recommended):** by default, connections to nodes
skip SSH host key verification and are vulnerable to MITM attacks. Set
`ssh_options.known_hosts_file` to a `known_hosts` file (standard OpenSSH
format, covering the infra/worker nodes) to enable verification:

```hcl
ssh_options = {
  known_hosts_file = "/path/to/known_hosts"
}
```

### 2. Deploy

```bash
terraform init
terraform plan
terraform apply
```

### 3. Access

```bash
terraform output next_steps
terraform output access_instructions
```

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│  complete-e2e-ocp-deployment (Consolidated Provider)       │
└────────────────────────────────────────────────────────────┘
                          │
                          │ gdpedge provider
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌─────────────────┐ ┌──────────────┐ ┌──────────────┐
│  gdp_edge_      │─▶│  gdp_edge_   │
│  rook_ceph      │ │  deployment  │
│  _cluster       │ │  (Optional)  │
│  (Optional)     │ └──────────────┘
└─────────────────┘        │
        │                  ▼
        ▼            Edge Components
  Storage Layer      (Native OAuth)
                   (Rook-Ceph)
```

## Configuration Examples

### OCP with Rook-Ceph Storage

```hcl
# ... basic configuration ...

install_rook_ceph           = true
rook_ceph_version           = "v1.15.4"
rook_ceph_installation_path = "/path/to/rook-ceph-files"
rook_ceph_airgap_install    = true

ocp_ssh_user     = "core"
ocp_ssh_password = "your-password"

rook_ceph_config = {
  set_as_default_storage = false
  pod_wait_timeout       = "600s"
  sleep_between_steps    = 60
}
```

### Complete E2E Deployment (OCP + Rook-Ceph + Edge)

```hcl
# When deploy_openshift=true, the kubeadmin password is automatically
# fetched - no need to set ocp_admin_password!
cluster_name      = "ocp-edge-complete"
worker_node_count = 6
wait_for_cluster  = true

# Rook-Ceph
install_rook_ceph = true
ocp_ssh_user      = "core"
ocp_ssh_password  = "your-password"

# Edge (uses native OCP OAuth - no oc login needed)
# The url may need port for rest api call to get edge bundle
install_edge       = true
edge_name          = "my-edge"
edge_cm_url        = "https://guardium-cm.example.com"
edge_oauth_token   = "your-oauth-token"

# Set to true when using an external image registry (e.g. Docker Hub, Quay)
# instead of the CM private registry. Skips registry certificate installation.
# external_image_registry = true
```

### Deploy Edge to Existing OpenShift Cluster

```hcl
manage_openshift         = false  # Keep existing cluster unmanaged
deploy_openshift         = false  # Don't create new cluster
cluster_name             = "my-existing-ocp"
ocp_master_node_hostname = "existing-ocp-inf.example.com"

# OCP authentication for Edge provider (native OAuth)
# Required when deploy_openshift=false since there's no OCP resource
# to auto-fetch the kubeadmin password from
ocp_admin_user     = "kubeadmin"
ocp_admin_password = "your-admin-password"
# Or use token: ocp_token = "sha256~your-token"

install_edge     = true
edge_name        = "my-edge"
edge_cm_url      = "https://gi-cm.example.com"
edge_oauth_token = "your-oauth-token"
```

### Install Rook-Ceph on Existing OpenShift Cluster

```hcl
manage_openshift            = false  # Keep existing cluster unmanaged
deploy_openshift            = false  # Don't create new cluster
cluster_name                = "my-existing-ocp"
ocp_master_node_hostname    = "existing-ocp-inf.example.com"
ocp_ssh_user                = "core"
ocp_ssh_password            = "your-password"

install_rook_ceph           = true
rook_ceph_version           = "v1.15.4"
rook_ceph_installation_path = "/path/to/rook-ceph-files"
```

## Outputs

```bash
terraform output
```

Key outputs:
- `cluster_summary` - Complete cluster info
- `access_instructions` - How to access cluster
- `deployment_summary` - All components status
- `next_steps` - Post-deployment instructions
- `rook_ceph_summary` - Rook-Ceph status (if installed)
- `edge_summary` - Edge deployment status (if installed)

## Two-Stage Deployment

For Rook-Ceph or Edge, you may need a two-stage approach:

**Stage 1: Create OCP Cluster**
```hcl
wait_for_cluster  = true
install_rook_ceph = false
install_edge      = false
```

```bash
terraform apply
```

**Stage 2: Install Components** (after cluster is ready, update terraform.tfvars)
```hcl
ocp_ssh_user      = "core"
ocp_ssh_password  = "your-password"
install_rook_ceph = true
install_edge      = true
# Note: ocp_admin_password is auto-fetched when deploy_openshift=true
```

```bash
terraform apply
```

## Troubleshooting

### Cluster Creation Timeout

Increase polling timeout:
```hcl
polling_timeout_minutes = 180  # Up to 240 minutes
```

### Rook-Ceph Installation Fails

```bash
kubectl get pods -n rook-ceph
kubectl get storageclass
kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph status
```

### Edge Deployment Fails

```bash
kubectl get pods -n <edge-namespace>
kubectl logs -n <edge-namespace> <pod-name>
```

## Cleanup

### Destroy All Resources

```bash
terraform destroy
```

This will remove all resources in reverse order: Edge -> Rook-Ceph -> OCP cluster.

### Transition from Managed to Unmanaged Cluster

If you want to keep the cluster but stop managing it with Terraform:

**Option 1: Remove from state first (recommended)**
```bash
# Remove cluster from Terraform state
terraform state rm guardium-data-protection_ocp.cluster[0]

# Update terraform.tfvars
manage_openshift = false
deploy_openshift = false

# Apply changes (no cluster destruction)
terraform apply
```

**Option 2: Update variables together**
```hcl
# In terraform.tfvars, change both at once:
manage_openshift = false
deploy_openshift = false
```

Then run `terraform apply`. The cluster resource count becomes 0, but since both variables are false, this indicates intentional transition to unmanaged state.

**⚠️ Warning**: Changing only `deploy_openshift` from `true` to `false` while keeping `manage_openshift = true` will result in cluster destruction. Always set both variables together when transitioning.

## File Structure

```
complete-e2e-ocp-deployment/
├── main.tf                    # Provider configs + resource definitions
├── variables.tf               # All variables
├── outputs.tf                 # Combined outputs
├── terraform.tfvars.example   # Example config
├── .gitignore                 # Security patterns
└── README.md                  # This file
```

## Provider Dependencies

- [`terraform-provider-gdp-edge`](../provider/terraform-provider-gdp-edge-native/) - Consolidated provider for all resources

See the provider directory for setup instructions (Artifactory mirror configuration).

