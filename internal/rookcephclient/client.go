// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package rookcephclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/edgeclient"
)

// Client manages SSH connections and Rook-Ceph operations on remote nodes
type Client struct {
	SSHUser     string
	SSHPassword string
	SSHOptions  SSHOptions
}

// SSHOptions holds SSH connection parameters
type SSHOptions struct {
	ConnectTimeout      int
	ServerAliveInterval int
	ServerAliveCount    int
	KnownHostsFile      string // path to known_hosts file for SSH host key verification; leave empty to disable verification
}

// RookCephConfig holds Rook-Ceph installation parameters
type RookCephConfig struct {
	ClusterName              string
	Platform                 string // "k3s" or "openshift"
	TargetNode               string // primary master or OCP API node
	RookCephVersion          string
	RookCephInstallationPath string
	AirgapInstall            bool
	WorkerCount              int
	TaintMasters             bool
	SetAsDefaultStorage      bool
	DisableLocalPath         bool // K3S only
	PodWaitTimeout           string
	SleepBetweenSteps        int
}

// NewClient creates a new SSH client for Rook-Ceph operations
func NewClient(sshUser string, sshPassword string, opts SSHOptions) *Client {
	if sshUser == "" {
		sshUser = "root"
	}
	return &Client{
		SSHUser:     sshUser,
		SSHPassword: sshPassword,
		SSHOptions:  opts,
	}
}

// sshDial establishes an SSH connection with keepalive support
func (c *Client) sshDial(host string) (*ssh.Client, func(), error) {
	hostKeyCallback, err := edgeclient.HostKeyCallback(c.SSHOptions.KnownHostsFile)
	if err != nil {
		return nil, nil, err
	}
	if c.SSHOptions.KnownHostsFile == "" {
		log.Printf("[WARN] SSH host key verification is disabled (no known_hosts file configured) — connections are vulnerable to MITM attacks")
	}

	config := &ssh.ClientConfig{
		User: c.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.SSHPassword),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(c.SSHOptions.ConnectTimeout) * time.Second,
	}

	addr := net.JoinHostPort(host, "22")
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to %s: %w", host, err)
	}

	// Start keepalive goroutine
	done := make(chan struct{})
	if c.SSHOptions.ServerAliveInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(c.SSHOptions.ServerAliveInterval) * time.Second)
			defer ticker.Stop()
			missedCount := 0
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
					if err != nil {
						missedCount++
						if missedCount >= c.SSHOptions.ServerAliveCount {
							client.Close()
							return
						}
					} else {
						missedCount = 0
					}
				}
			}
		}()
	}

	cleanup := func() {
		close(done)
		client.Close()
	}

	return client, cleanup, nil
}

// RunSSH executes a command on a remote host via native Go SSH.
// The context is honoured: if ctx is cancelled or times out the SSH session
// is closed and an error is returned immediately.
func (c *Client) RunSSH(ctx context.Context, host string, command string) (string, error) {
	client, cleanup, err := c.sshDial(host)
	if err != nil {
		return "", err
	}
	defer cleanup()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session on %s: %w", host, err)
	}
	defer session.Close()

	// Capture combined output (stdout + stderr)
	var output bytes.Buffer
	session.Stdout = &output
	session.Stderr = &output

	if err := session.Start(command); err != nil {
		return "", fmt.Errorf("SSH command failed to start on %s: %w", host, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return output.String(), fmt.Errorf("SSH command timed out on %s: %w", host, ctx.Err())
	case err := <-done:
		if err != nil {
			return output.String(), fmt.Errorf("SSH command failed on %s: %w\nOutput: %s", host, err, output.String())
		}
		return output.String(), nil
	}
}

// CloneRookRepo clones the rook repository locally to a temporary directory
func (c *Client) CloneRookRepo(version string) (string, error) {
	// Create a temporary directory for the clone
	tmpDir, err := os.MkdirTemp("", "rook-clone-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	repoURL := "https://github.com/rook/rook.git"

	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewTagReferenceName(version),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone rook repository: %w", err)
	}

	return tmpDir, nil
}

// UploadDirectory uploads a local directory to a remote host via SFTP
func (c *Client) UploadDirectory(host string, localPath string, remotePath string) error {
	client, cleanup, err := c.sshDial(host)
	if err != nil {
		return err
	}
	defer cleanup()

	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Remove existing remote directory if it exists (best-effort)
	if err := sftpClient.RemoveAll(remotePath); err != nil {
		// Ignore "not exist" errors, warn on others
		if !os.IsNotExist(err) {
			fmt.Printf("[WARN] Failed to remove existing remote directory %s: %v\n", remotePath, err)
		}
	}

	// Walk the local directory and upload files
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// Skip .git directory to reduce upload size
		if strings.HasPrefix(relPath, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		remoteDest := filepath.Join(remotePath, relPath)
		// Convert to Unix-style path for remote
		remoteDest = strings.ReplaceAll(remoteDest, "\\", "/")

		// Handle symlinks by resolving them
		if info.Mode()&os.ModeSymlink != 0 {
			// Resolve symlink to get the actual target info
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				// Skip broken symlinks
				return err
			}
			realInfo, err := os.Stat(realPath)
			if err != nil {
				return err
			}
			if realInfo.IsDir() {
				// Symlink to directory - create directory on remote
				return sftpClient.MkdirAll(remoteDest)
			}
			// Symlink to file - upload the actual file
			return c.uploadFile(sftpClient, realPath, remoteDest)
		}

		if info.IsDir() {
			// Create remote directory
			return sftpClient.MkdirAll(remoteDest)
		}

		// Upload file
		return c.uploadFile(sftpClient, path, remoteDest)
	})
}

// uploadFile uploads a single file via SFTP
func (c *Client) uploadFile(sftpClient *sftp.Client, localPath string, remotePath string) error {
	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Ensure remote directory exists
	remoteDir := filepath.Dir(remotePath)
	remoteDir = strings.ReplaceAll(remoteDir, "\\", "/")
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	// Copy content
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content to %s: %w", remotePath, err)
	}

	return nil
}

// CleanupLocalRepo removes the locally cloned repository
func (c *Client) CleanupLocalRepo(localPath string) error {
	return os.RemoveAll(localPath)
}

// parseRookVersion extracts major.minor from a rook version string like "v1.19.0" or "v1.18".
func parseRookVersion(v string) (int, int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	major, minor := 0, 0
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

// rookVersionGTE returns true if version >= threshold (major.minor comparison).
func rookVersionGTE(version, threshold string) bool {
	vMaj, vMin := parseRookVersion(version)
	tMaj, tMin := parseRookVersion(threshold)
	if vMaj != tMaj {
		return vMaj > tMaj
	}
	return vMin >= tMin
}

// kubeCmd returns "oc" for openshift, "kubectl" for k3s
func kubeCmd(platform string) string {
	if platform == "openshift" {
		return "oc"
	}
	return "kubectl"
}

// kubeconfigExport returns the KUBECONFIG export line based on platform
func kubeconfigExport(platform string) string {
	if platform == "openshift" {
		return "source ~/.bash_profile 2>/dev/null || source ~/.bashrc 2>/dev/null || true\nexport KUBECONFIG=${KUBECONFIG:-~/.kube/config}"
	}
	return "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml"
}

// InstallRookCeph performs the full Rook-Ceph installation
func (c *Client) InstallRookCeph(ctx context.Context, config RookCephConfig) error {
	kc := kubeCmd(config.Platform)
	kcExport := kubeconfigExport(config.Platform)
	isTest := config.WorkerCount <= 1 && config.Platform == "k3s"

	// Determine which operator/cluster YAML to use
	operatorYaml := "operator.yaml"
	if config.Platform == "openshift" {
		operatorYaml = "operator-openshift.yaml"
	}

	clusterYaml := "cluster.yaml"
	if isTest {
		clusterYaml = "cluster-test.yaml"
	}

	filesystemYaml := "filesystem.yaml"
	if isTest {
		filesystemYaml = "filesystem-test.yaml"
	}

	rbdStorageYaml := "csi/rbd/storageclass.yaml"
	if isTest {
		rbdStorageYaml = "csi/rbd/storageclass-test.yaml"
	}

	// Single-node: reduce CSI provisioner replicas to 1.
	// Rook v1.15+ defaults CSI_PROVISIONER_REPLICAS to 2 in operator.yaml, but with taint_master = true for 1 master and 1 worker cluster, only 1 schedulable node and required pod anti-affinity, the 2nd replica can never schedule.
	// Patch the ConfigMap after operator.yaml (which creates it).
	csiProvisionerReplicaStep := ""
	if isTest && config.TaintMasters {
		csiProvisionerReplicaStep = fmt.Sprintf(`
echo "[INFO] Single-node cluster: setting CSI_PROVISIONER_REPLICAS to 1"
%s -n rook-ceph patch configmap rook-ceph-operator-config --type merge -p '{"data": {"CSI_PROVISIONER_REPLICAS": "1"}}'`, kc)
	}

	// Extra CSI operator required for rook ceph v1.18+ (introduces csi.ceph.io/v1 CRDs: CephConnection, OperatorConfig)
	csiOperatorStep := ""
	if rookVersionGTE(config.RookCephVersion, "v1.18") {
		csiOperatorStep = fmt.Sprintf(`
echo "[INFO] Installing CSI operator"
%s apply -f csi-operator.yaml
echo "[INFO] Waiting for CSI operator CRDs to be established"
%s wait --for=condition=Established crd/cephconnections.csi.ceph.io --timeout=60s
%s wait --for=condition=Established crd/operatorconfigs.csi.ceph.io --timeout=60s
echo "[INFO] Waiting for CSI operator pod to be ready"
%s -n rook-ceph rollout status deployment/ceph-csi-controller-manager --timeout=120s
echo "[SUCCESS] CSI operator is ready"`, kc, kc, kc, kc)
	}

	// In rook v1.18+, CSI plugin pods are managed by the CSI operator and use new names
	// (e.g. rook-ceph.cephfs.csi.ceph.com-ctrlplugin) instead of the old app= labels
	// (csi-cephfsplugin, csi-rbdplugin, etc.). Use rollout status on the new resources.
	csiDriverWaitStep := ""
	csiSelectorList := "csi-cephfsplugin csi-rbdplugin csi-cephfsplugin-provisioner csi-rbdplugin-provisioner rook-ceph-mon rook-ceph-mgr rook-ceph-osd"
	if rookVersionGTE(config.RookCephVersion, "v1.18") {
		csiSelectorList = "rook-ceph-mon rook-ceph-mgr rook-ceph-osd"
		csiDriverWaitStep = fmt.Sprintf(`
echo "[INFO] Waiting for CSI driver components (v1.18+)"
for CSI_RESOURCE in deployment/rook-ceph.cephfs.csi.ceph.com-ctrlplugin deployment/rook-ceph.rbd.csi.ceph.com-ctrlplugin daemonset/rook-ceph.cephfs.csi.ceph.com-nodeplugin daemonset/rook-ceph.rbd.csi.ceph.com-nodeplugin; do
  WAIT_COUNT=0
  while ! %s -n rook-ceph get "$CSI_RESOURCE" >/dev/null 2>&1; do
    WAIT_COUNT=$((WAIT_COUNT + 1))
    if [ $WAIT_COUNT -ge 30 ]; then
      echo "[ERROR] Timed out waiting for $CSI_RESOURCE to be created"
      exit 1
    fi
    sleep %d
  done
  %s -n rook-ceph rollout status "$CSI_RESOURCE" --timeout=%s
  echo "[SUCCESS] $CSI_RESOURCE is ready"
done
echo "[SUCCESS] All CSI driver components are ready"`, kc, config.SleepBetweenSteps, kc, config.PodWaitTimeout)
	}

	var localRepoPath string
	var err error
	if !config.AirgapInstall {
		// Step 1: Clone rook repository locally
		localRepoPath, err = c.CloneRookRepo(config.RookCephVersion)
		if err != nil {
			return fmt.Errorf("failed to clone rook repository locally: %w", err)
		}
		defer func() {
			if cleanupErr := c.CleanupLocalRepo(localRepoPath); cleanupErr != nil {
				// Log cleanup error but don't override the main error
				_ = cleanupErr
			}
		}()

	} else {
		localRepoPath = config.RookCephInstallationPath
	}

	// Step 2: Upload to remote node
	remotePath := "/tmp/rook"
	err = c.UploadDirectory(config.TargetNode, localRepoPath, remotePath)
	if err != nil {
		return fmt.Errorf("failed to upload rook repository to %s: %w", config.TargetNode, err)
	}

	// Step 3: Run installation script (no git required on remote)
	script := fmt.Sprintf(`
set -e
%s

echo "-------------------------------------------------------"
echo "        Installing Rook-Ceph %s                       "
echo "        Platform: %s                                   "
echo "-------------------------------------------------------"

if [ ! -d /tmp/rook/deploy/examples ]; then
  echo "[ERROR] Rook repository not found at /tmp/rook"
  exit 1
fi

cd /tmp/rook/deploy/examples/

# Install CRDs
echo "[INFO] Installing Rook-Ceph CRDs"
%s -n rook-ceph create namespace rook-ceph --dry-run -o yaml 2>/dev/null | %s apply -f - 2>/dev/null || true
%s -n rook-ceph apply -f crds.yaml

# Install common resources
echo "[INFO] Installing common resources"
%s -n rook-ceph apply -f common.yaml

# Install operator
echo "[INFO] Installing Rook-Ceph operator"
%s -n rook-ceph apply -f %s
%s
%s

# Wait for operator
echo "[INFO] Waiting for operator to be ready (sleeping %ds)"
sleep %d
echo "[INFO] Waiting for rook-ceph-operator pod to be ready"
%s -n rook-ceph wait --for=condition=Ready --selector=app=rook-ceph-operator pod --timeout=%s
echo "[SUCCESS] Rook-Ceph operator is ready"

# Create cluster
echo "[INFO] Creating Rook-Ceph cluster"
%s -n rook-ceph apply -f %s
echo "[INFO] Waiting for cluster components (sleeping %ds)"
sleep %d

# Wait for Ceph components
echo "[INFO] Waiting for Ceph components to be ready"

for SELECTOR in %s; do
  echo "[INFO] Waiting for $SELECTOR pods to exist"
  WAIT_COUNT=0
  while ! %s -n rook-ceph get pod --selector=app=$SELECTOR --no-headers 2>/dev/null | grep -q .; do
    WAIT_COUNT=$((WAIT_COUNT + 1))
    if [ $WAIT_COUNT -ge 30 ]; then
      echo "[ERROR] Timed out waiting for $SELECTOR pods to be created"
      exit 1
    fi
    echo "[INFO] No $SELECTOR pods found yet, retrying in %ds..."
    sleep %d
  done
  echo "[INFO] $SELECTOR pods found, waiting for readiness"
  %s -n rook-ceph wait --for=condition=Ready --selector=app=$SELECTOR pod --timeout=%s
  echo "[SUCCESS] $SELECTOR pods are ready"
done
%s

# Create filesystem
echo "[INFO] Creating Ceph filesystem"
%s -n rook-ceph apply -f %s
echo "[INFO] Waiting for filesystem to initialize (sleeping %ds)"
sleep %d

# Create storage classes (before toolbox, since toolbox is optional)
echo "[INFO] Creating storage classes"
%s -n rook-ceph apply -f csi/cephfs/storageclass.yaml
%s -n rook-ceph apply -f %s

%s get storageclass
echo "[SUCCESS] Storage classes created"

# Create toolbox (non-fatal - toolbox is a debug tool, not required for storage)
echo "[INFO] Creating Ceph toolbox"
%s -n rook-ceph apply -f toolbox.yaml
echo "[INFO] Waiting for toolbox to be ready"
%s -n rook-ceph wait --for=condition=Ready --selector=app=rook-ceph-tools pod --timeout=%s || echo "[WARN] Toolbox not ready yet, will continue (toolbox is optional)"
echo "[SUCCESS] Ceph toolbox setup completed"

echo "[SUCCESS] Rook-Ceph installation completed"
`,
		kcExport,
		config.RookCephVersion, config.Platform,
		kc, kc,
		kc,
		kc,
		kc, operatorYaml,
		csiOperatorStep,
		csiProvisionerReplicaStep,
		config.SleepBetweenSteps, config.SleepBetweenSteps,
		kc, config.PodWaitTimeout,
		kc, clusterYaml,
		config.SleepBetweenSteps, config.SleepBetweenSteps,
		csiSelectorList,
		kc,
		config.SleepBetweenSteps, config.SleepBetweenSteps,
		kc, config.PodWaitTimeout,
		csiDriverWaitStep,
		kc, filesystemYaml,
		config.SleepBetweenSteps, config.SleepBetweenSteps,
		kc,
		kc, rbdStorageYaml,
		kc,
		kc,
		kc, config.PodWaitTimeout,
	)

	_, err = c.RunSSH(ctx, config.TargetNode, script)
	return err
}

// ConfigureDefaultStorage sets rook-cephfs as default storage class
func (c *Client) ConfigureDefaultStorage(ctx context.Context, config RookCephConfig) error {
	kc := kubeCmd(config.Platform)
	kcExport := kubeconfigExport(config.Platform)

	disableLocalPathStep := ""
	if config.Platform == "k3s" && config.DisableLocalPath {
		disableLocalPathStep = `
echo "[INFO] Disabling local-path as default storage class"

if [ ! -f /etc/rancher/k3s/config.yaml ]; then
  echo "disable:" > /etc/rancher/k3s/config.yaml
  echo "  - local-storage" >> /etc/rancher/k3s/config.yaml
elif ! grep -q "disable:" /etc/rancher/k3s/config.yaml; then
  echo -e "\ndisable:\n  - local-storage" >> /etc/rancher/k3s/config.yaml
elif ! grep -q "local-storage" /etc/rancher/k3s/config.yaml; then
  sed -i "/disable:/a \  - local-storage" /etc/rancher/k3s/config.yaml
fi

echo "[INFO] Restarting K3S to apply configuration"
systemctl restart k3s
sleep 30
kubectl wait --for=condition=Ready nodes --all --timeout=300s
`
	}

	script := fmt.Sprintf(`
set -e
%s

%s

echo "[INFO] Setting rook-cephfs as default storage class"
%s patch storageclass rook-cephfs -p '{"metadata": {"annotations": {"storageclass.kubernetes.io/is-default-class": "true"}}}'

# Add discard mount option to rook-ceph-block (K3S)
%s patch storageclass rook-ceph-block --type json -p '[{"op": "add", "path": "/mountOptions", "value": ["discard"]}]' 2>/dev/null || true

echo "[INFO] Storage class configuration completed"
%s get storageclass
`, kcExport, disableLocalPathStep, kc, kc, kc)

	_, err := c.RunSSH(ctx, config.TargetNode, script)
	return err
}

// VerifyInstallation verifies the Rook-Ceph installation and waits for HEALTH_OK
func (c *Client) VerifyInstallation(ctx context.Context, config RookCephConfig) (string, error) {
	kc := kubeCmd(config.Platform)
	kcExport := kubeconfigExport(config.Platform)

	// Derive health-check loop parameters from pod_wait_timeout (10s poll interval)
	healthCheckInterval := 10
	healthCheckMaxWait := 30 // default: 30 * 10s = 5 min
	if config.PodWaitTimeout != "" {
		if d, err := time.ParseDuration(config.PodWaitTimeout); err == nil {
			if secs := int(d.Seconds()); secs > 0 {
				healthCheckMaxWait = secs / healthCheckInterval
			}
		}
	}
	totalHealthCheckSecs := healthCheckMaxWait * healthCheckInterval

	script := fmt.Sprintf(`
set -e
%s

echo "-------------------------------------------------------"
echo "         Rook-Ceph Pods Status                         "
echo "-------------------------------------------------------"
%s get pods -n rook-ceph
echo ""

echo "-------------------------------------------------------"
echo "         Storage Classes                               "
echo "-------------------------------------------------------"
%s get storageclass
echo ""

echo "-------------------------------------------------------"
echo "         Ceph Cluster Status                           "
echo "-------------------------------------------------------"
%s get cephcluster -n rook-ceph 2>/dev/null || echo "[INFO] CephCluster CRD not available"
echo ""

echo "-------------------------------------------------------"
echo "         Waiting for Ceph HEALTH_OK                    "
echo "-------------------------------------------------------"
%s -n rook-ceph wait --for=condition=Ready --selector=app=rook-ceph-tools pod --timeout=%s 2>/dev/null || echo "[WARN] Toolbox not ready yet, health check may fail"
CEPH_HEALTHY=0
WAIT_COUNT=0
MAX_WAIT=%d
INTERVAL=%d
HEALTH="UNAVAILABLE"
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
  HEALTH=$(%s -n rook-ceph exec deploy/rook-ceph-tools -- ceph health 2>/dev/null || echo "UNAVAILABLE")
  if echo "$HEALTH" | grep -q "HEALTH_OK"; then
    echo "[SUCCESS] Ceph cluster is HEALTH_OK"
    CEPH_HEALTHY=1
    break
  fi
  WAIT_COUNT=$((WAIT_COUNT + 1))
  echo "[INFO] Ceph health: $HEALTH — waiting ${INTERVAL}s... ($WAIT_COUNT/$MAX_WAIT)"
  sleep $INTERVAL
done
if [ $CEPH_HEALTHY -eq 0 ]; then
  echo "[ERROR] Ceph cluster did not reach HEALTH_OK after %ds. Last health: $HEALTH"
  exit 1
fi

echo "-------------------------------------------------------"
echo "         Ceph Status                                   "
echo "-------------------------------------------------------"
%s -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph status 2>/dev/null || echo "[WARN] Unable to get Ceph status"
echo ""

echo "-------------------------------------------------------"
echo "         Ceph OSD Status                               "
echo "-------------------------------------------------------"
%s -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph osd status 2>/dev/null || echo "[WARN] Unable to get OSD status"
echo ""

echo "[INFO] Rook-Ceph verification completed"
`,
		kcExport,
		kc,
		kc,
		kc,
		kc, config.PodWaitTimeout,
		healthCheckMaxWait,
		healthCheckInterval,
		kc,
		totalHealthCheckSecs,
		kc,
		kc,
	)

	return c.RunSSH(ctx, config.TargetNode, script)
}

// UninstallRookCeph removes Rook-Ceph from the cluster following the official teardown guide:
// https://rook.io/docs/rook/latest-release/Getting-Started/ceph-teardown/
func (c *Client) UninstallRookCeph(ctx context.Context, config RookCephConfig) error {
	kc := kubeCmd(config.Platform)
	kcExport := kubeconfigExport(config.Platform)

	// OpenShift installs an extra CSI operator that must be removed during uninstall.
	csiOperatorUninstallStep := ""
	if config.Platform == "openshift" {
		csiOperatorUninstallStep = fmt.Sprintf(
			"%s -n rook-ceph delete -f /tmp/rook/deploy/examples/csi-operator.yaml --ignore-not-found=true 2>/dev/null || true",
			kc)
	}

	script := fmt.Sprintf(`
set -e
%s

echo "[INFO] Uninstalling Rook-Ceph"

# Step 1: Delete storage artifacts before removing the cluster.
# Volumes may hang if the operator is removed before applications are cleaned up.
%s -n rook-ceph delete cephblockpool --all --wait=false --ignore-not-found=true 2>/dev/null || true
%s -n rook-ceph delete cephfilesystem --all --wait=false --ignore-not-found=true 2>/dev/null || true
%s delete storageclass rook-cephfs --ignore-not-found=true 2>/dev/null || true
%s delete storageclass rook-ceph-block --ignore-not-found=true 2>/dev/null || true
%s delete storageclass rook-ceph-block-test --ignore-not-found=true 2>/dev/null || true
%s -n rook-ceph delete -f /tmp/rook/deploy/examples/toolbox.yaml --ignore-not-found=true 2>/dev/null || true

# Step 2: Enable cleanup policy so the operator wipes data and OSD drives on deletion.
%s -n rook-ceph patch cephcluster rook-ceph --type merge \
  -p '{"spec":{"cleanupPolicy":{"confirmation":"yes-really-destroy-data"}}}' 2>/dev/null || true

# Step 3: Delete the named CephCluster and wait for cleanup jobs to finish (up to 10 min).
%s -n rook-ceph delete cephcluster rook-ceph --wait=false --ignore-not-found=true 2>/dev/null || true
echo "[INFO] Waiting for CephCluster deletion and cleanup jobs..."
for i in $(seq 1 60); do
  if ! %s -n rook-ceph get cephcluster rook-ceph >/dev/null 2>&1; then
    echo "[INFO] CephCluster deleted"
    break
  fi
  echo "[INFO] Waiting... ($i/60)"
  sleep 10
done

# Step 4: Remove finalizers from any stuck CRDs to unblock namespace termination.
# Match both ceph.rook.io and csi.ceph.io CRDs (e.g. clientprofiles.csi.ceph.io).
for CRD in $(%s get crd 2>/dev/null | awk '/ceph\.rook\.io|csi\.ceph\.io/ {print $1}'); do
  %s get -n rook-ceph "$CRD" -o name 2>/dev/null | \
    xargs -I {} %s patch -n rook-ceph {} --type merge -p '{"metadata":{"finalizers": []}}' 2>/dev/null || true
done
%s -n rook-ceph patch configmap rook-ceph-mon-endpoints --type merge -p '{"metadata":{"finalizers": []}}' 2>/dev/null || true
%s -n rook-ceph patch secrets rook-ceph-mon --type merge -p '{"metadata":{"finalizers": []}}' 2>/dev/null || true

# Step 5: Delete operator, common resources, and CRDs.
%s
%s -n rook-ceph delete -f /tmp/rook/deploy/examples/operator.yaml --wait=false --ignore-not-found=true 2>/dev/null || true
%s -n rook-ceph delete -f /tmp/rook/deploy/examples/operator-openshift.yaml --wait=false --ignore-not-found=true 2>/dev/null || true
%s -n rook-ceph delete -f /tmp/rook/deploy/examples/common.yaml --wait=false --ignore-not-found=true 2>/dev/null || true
%s -n rook-ceph delete -f /tmp/rook/deploy/examples/crds.yaml --wait=false --ignore-not-found=true 2>/dev/null || true

# Step 6: Delete namespace.
%s delete namespace rook-ceph --wait=false --ignore-not-found=true 2>/dev/null || true

# Step 6b: Remove any remaining CRD finalizers and the namespace spec.finalizers so
# the namespace can terminate without waiting on the Kubernetes namespace controller.
# Match both ceph.rook.io and csi.ceph.io CRDs (e.g. clientprofiles.csi.ceph.io).
for CRD in $(%s get crd 2>/dev/null | awk '/ceph\.rook\.io|csi\.ceph\.io/ {print $1}'); do
  %s get -n rook-ceph "$CRD" -o name 2>/dev/null | \
    xargs -I {} %s patch -n rook-ceph {} --type merge -p '{"metadata":{"finalizers": []}}' 2>/dev/null || true
done
%s patch namespace rook-ceph -p '{"spec":{"finalizers":[]}}' --type=merge 2>/dev/null || true

# Step 7: Host cleanup - remove dataDirHostPath (/var/lib/rook) and device mapper entries.
rm -rf /var/lib/rook
ls /dev/mapper/ceph-* 2>/dev/null | xargs -I%% -- dmsetup remove %% 2>/dev/null || true
rm -rf /dev/ceph-*
rm -rf /dev/mapper/ceph--*

# Cleanup cloned repo
rm -rf /tmp/rook

echo "[INFO] Rook-Ceph uninstalled"
`, kcExport,
		kc, kc, kc, kc, kc, kc, // delete blockpool, filesystem, 3 storageclasses, toolbox
		kc,         // patch cleanup policy
		kc,         // delete cephcluster
		kc,         // wait loop check
		kc, kc, kc, // CRD finalizer loop (get crd, get objects, patch)
		kc, kc, // patch configmap, patch secrets
		csiOperatorUninstallStep, // delete csi-operator.yaml (OpenShift only, empty string otherwise)
		kc, kc, kc, kc,           // delete operator, operator-openshift, common, crds
		kc,         // delete namespace
		kc, kc, kc, // step 6b: CRD finalizer loop (get crd, get objects, patch)
		kc) // step 6b: patch namespace spec.finalizers

	_, err := c.RunSSH(ctx, config.TargetNode, script)
	return err
}

// CheckRookCephInstalled checks if Rook-Ceph is running
func (c *Client) CheckRookCephInstalled(ctx context.Context, config RookCephConfig) (bool, error) {
	kc := kubeCmd(config.Platform)
	kcExport := kubeconfigExport(config.Platform)

	script := fmt.Sprintf(`%s
%s get namespace rook-ceph -o jsonpath='{.metadata.name}' 2>/dev/null || echo ""`, kcExport, kc)

	output, err := c.RunSSH(ctx, config.TargetNode, script)
	if err != nil {
		return false, err
	}
	return strings.Contains(output, "rook-ceph"), nil
}
