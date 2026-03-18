package edgeclient

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k8sclient"
)

// Config holds client configuration
type Config struct {
	// Central Manager
	CMUrl      string
	OAuthToken string

	// Platform
	Platform string // k3s, eks, openshift

	// SSH configuration
	SSHUser     string
	SSHPassword string
	SSHKeyPath  string

	// AWS EKS
	AWSRegion           string
	AWSProfile          string
	AWSAccessKey        string
	AWSSecretKey        string
	EKSClusterName      string
	EKSSSHUser          string
	EKSSSHKeyPath       string
	EKSSSHKeyPassphrase string
	EKSHostnameType     string

	// OpenShift native OAuth authentication
	OCPServer             string // API server URL (e.g., https://api.cluster.example.com:6443)
	OCPUsername           string // OpenShift username
	OCPPassword           string // OpenShift password
	OCPToken              string // Pre-existing OAuth token (alternative to username/password)
	OCPInsecureSkipVerify bool   // Skip TLS certificate verification
}

// Client handles Edge deployment operations
type Client struct {
	Config    Config
	k8sClient *k8sclient.Client
	sshClient *SSHClient
}

// NewClient creates a new Client
func NewClient(cfg Config) *Client {
	c := &Client{
		Config: cfg,
	}

	// Initialize SSH client if credentials provided
	if cfg.SSHUser != "" && (cfg.SSHPassword != "" || cfg.SSHKeyPath != "") {
		sshClient, err := NewSSHClient(cfg.SSHUser, cfg.SSHPassword, cfg.SSHKeyPath)
		if err == nil {
			c.sshClient = sshClient
		}
	}

	return c
}

// InitK8sClient initializes the Kubernetes client
func (c *Client) InitK8sClient(ctx context.Context, kubeconfigPath string) error {
	k8sCfg := k8sclient.Config{
		KubeconfigPath:        kubeconfigPath,
		Platform:              c.Config.Platform,
		AWSRegion:             c.Config.AWSRegion,
		AWSProfile:            c.Config.AWSProfile,
		AWSAccessKey:          c.Config.AWSAccessKey,
		AWSSecretKey:          c.Config.AWSSecretKey,
		EKSClusterName:        c.Config.EKSClusterName,
		OCPServer:             c.Config.OCPServer,
		OCPUsername:           c.Config.OCPUsername,
		OCPPassword:           c.Config.OCPPassword,
		OCPToken:              c.Config.OCPToken,
		OCPInsecureSkipVerify: c.Config.OCPInsecureSkipVerify,
	}

	client, err := k8sclient.NewClient(ctx, k8sCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize k8s client: %w", err)
	}

	c.k8sClient = client
	return nil
}

// K8sClient returns the Kubernetes client
func (c *Client) K8sClient() *k8sclient.Client {
	return c.k8sClient
}

// RunSSH executes a command on a remote host via SSH (native Go implementation)
func (c *Client) RunSSH(host string, command string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("SSH client not initialized")
	}
	return c.sshClient.Run(host, command)
}

// SCPFile copies a file to a remote host via SCP (native Go implementation)
func (c *Client) SCPFile(localPath, remotePath, host string) error {
	if c.sshClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}
	return c.sshClient.CopyTo(host, localPath, remotePath)
}

// SCPFileFrom copies a file from a remote host via SCP (native Go implementation)
func (c *Client) SCPFileFrom(remotePath, localPath, host string) error {
	if c.sshClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}
	return c.sshClient.CopyFrom(host, remotePath, localPath)
}

// FetchKubeconfig fetches kubeconfig from K3S master node
func (c *Client) FetchKubeconfig(masterNode string, kubeconfigPath string) error {
	// Create directory for kubeconfig if needed
	if err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create kubeconfig directory: %w", err)
	}

	// Copy kubeconfig from master
	if err := c.SCPFileFrom("/etc/rancher/k3s/k3s.yaml", kubeconfigPath, masterNode); err != nil {
		return fmt.Errorf("failed to copy kubeconfig: %w", err)
	}

	// Update server address in kubeconfig
	content, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Replace 127.0.0.1 with actual master IP/hostname
	updated := strings.ReplaceAll(string(content), "127.0.0.1", masterNode)
	updated = strings.ReplaceAll(updated, "localhost", masterNode)

	if err := os.WriteFile(kubeconfigPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	return nil
}

// DownloadBundle downloads an edge bundle from Central Manager (native Go implementation)
// example: curl -k -H "Authorization:Bearer WVC2i8Shl5s6LCpWhZhfn8-sdbY" -O -J "https://gat-snif-40.svl.ibm.com:8443/restAPI/get_bundle?name=edge-ns2-2"
func (c *Client) DownloadBundle(edgeName, destDir string) error {
	if c.Config.CMUrl == "" || c.Config.OAuthToken == "" {
		return fmt.Errorf("CM URL and OAuth token are required for bundle download")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// URL matches: /restAPI/get_bundle?name={edgeName}
	bundleURL := fmt.Sprintf("%s/restAPI/get_bundle?name=%s", strings.TrimSuffix(c.Config.CMUrl, "/"), edgeName)

	// Create HTTP client with TLS skip verify (equivalent to curl -k)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Minute,
	}

	req, err := http.NewRequest("GET", bundleURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Config.OAuthToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download bundle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download bundle: HTTP %d", resp.StatusCode)
	}

	// Determine filename from Content-Disposition header (equivalent to curl -J)
	filename := bundleFilenameFromResponse(resp, edgeName)
	bundleFile := filepath.Join(destDir, filename)

	// Save response body to file (equivalent to curl -O)
	f, err := os.Create(bundleFile)
	if err != nil {
		return fmt.Errorf("failed to create bundle file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("failed to save bundle: %w", err)
	}
	f.Close()

	// Extract based on file extension
	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		f, err := os.Open(bundleFile)
		if err != nil {
			return fmt.Errorf("failed to open bundle file: %w", err)
		}
		defer f.Close()
		if err := extractTarGz(f, destDir); err != nil {
			return fmt.Errorf("failed to extract bundle: %w", err)
		}
	} else {
		// Default: treat as zip (e.g. if server returns an unexpected format)
		if err := extractZip(bundleFile, destDir); err != nil {
			return fmt.Errorf("failed to extract bundle: %w", err)
		}
	}

	os.Remove(bundleFile)
	return nil
}

// bundleFilenameFromResponse extracts filename from Content-Disposition header (curl -J behavior).
// Falls back to "{edgeName}.zip" if the header is absent or cannot be parsed.
func bundleFilenameFromResponse(resp *http.Response, edgeName string) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if name := params["filename"]; name != "" {
				return filepath.Base(name)
			}
		}
	}
	return edgeName + ".tar.gz"
}

// extractZip extracts a zip archive to destDir
func extractZip(zipFile, destDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		// Prevent path traversal attacks
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// extractTarGz extracts a tar.gz stream to a directory
func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		// Prevent path traversal attacks
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			f.Close()
		}
	}

	return nil
}

// ExtractCertInfo extracts certificate info from the edge bundle.
// When externalImageRegistry is true, reads EXTERNAL_IMAGE_REGISTRY from the configmap
// and skips CM registry certificate extraction from the secret.
// When false, reads CM_PRIVATE_REGISTRY and extracts the CM registry cert.
func (c *Client) ExtractCertInfo(workDir string, externalImageRegistry bool) (registry, namespace string, err error) {
	// Read edge-controller-client configmap YAML
	cmPath := filepath.Join(workDir, "01-edge-controller-client-configmap.yaml")
	cmContent, err := os.ReadFile(cmPath)
	if err != nil {
		return "", "", fmt.Errorf("could not read %s: %w", cmPath, err)
	}

	// Parse YAML to extract registry and namespace.
	// Field name depends on registry mode:
	//   external_image_registry=true  -> EXTERNAL_IMAGE_REGISTRY
	//   external_image_registry=false -> CM_PRIVATE_REGISTRY
	registryField := "CM_PRIVATE_REGISTRY:"
	if externalImageRegistry {
		registryField = "EXTERNAL_IMAGE_REGISTRY:"
	}
	lines := strings.Split(string(cmContent), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, registryField) {
			registry = strings.TrimSpace(strings.TrimPrefix(line, registryField))
			registry = strings.Trim(registry, "\"'")
		}
		if strings.HasPrefix(line, "NAMESPACE:") {
			namespace = strings.TrimSpace(strings.TrimPrefix(line, "NAMESPACE:"))
			namespace = strings.Trim(namespace, "\"'")
		}
	}

	if registry == "" {
		return "", "", fmt.Errorf("could not extract registry URL from configmap (%s not found)", registryField)
	}
	if namespace == "" {
		return "", "", fmt.Errorf("could not extract namespace from configmap (NAMESPACE not found)")
	}

	// Save extracted info
	os.WriteFile(filepath.Join(workDir, ".registry_info"), []byte(registry), 0644)
	os.WriteFile(filepath.Join(workDir, ".namespace_info"), []byte(namespace), 0644)

	// Extract CM registry certificate from the secret.
	// Skip when using an external image registry — no CM registry cert is present in the bundle.
	if !externalImageRegistry {
		secretPath := filepath.Join(workDir, "02-edge-imagepull-secret.yaml")
		if secretContent, readErr := os.ReadFile(secretPath); readErr == nil {
			for _, line := range strings.Split(string(secretContent), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "cm-registry.crt:") {
					certB64 := strings.TrimSpace(strings.TrimPrefix(line, "cm-registry.crt:"))
					certB64 = strings.Trim(certB64, "\"'")
					if certB64 != "" {
						certData, decodeErr := base64.StdEncoding.DecodeString(certB64)
						if decodeErr == nil {
							os.WriteFile(filepath.Join(workDir, ".registry_cert.crt"), certData, 0644)
						}
					}
					break
				}
			}
		}
	}

	return registry, namespace, nil
}

// InstallCertsK3S installs registry certificates on K3S nodes
func (c *Client) InstallCertsK3S(ctx context.Context, workDir string, nodes []string, registryHost string) error {
	certPath := filepath.Join(workDir, ".registry_cert.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		// No certificate to install
		return nil
	}

	// If no nodes specified, get from K8s API
	if len(nodes) == 0 && c.k8sClient != nil {
		var err error
		nodes, err = c.k8sClient.ListNodeNames(ctx)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}
	}

	// Install cert on each node
	for _, node := range nodes {
		remoteCertDir := fmt.Sprintf("/var/lib/rancher/k3s/agent/etc/containerd/certs.d/%s", registryHost)

		// Create directory on remote node
		if _, err := c.RunSSH(node, fmt.Sprintf("sudo mkdir -p '%s'", remoteCertDir)); err != nil {
			return fmt.Errorf("failed to create cert dir on %s: %w", node, err)
		}

		// Copy certificate
		remoteCertPath := filepath.Join(remoteCertDir, "ca.crt")
		if err := c.SCPFile(certPath, "/tmp/ca.crt", node); err != nil {
			return fmt.Errorf("failed to copy cert to %s: %w", node, err)
		}

		// Move cert to final location
		if _, err := c.RunSSH(node, fmt.Sprintf("sudo mv /tmp/ca.crt '%s'", remoteCertPath)); err != nil {
			return fmt.Errorf("failed to install cert on %s: %w", node, err)
		}
	}

	return nil
}

// InstallCertsOpenShift installs registry certificates on OpenShift cluster
func (c *Client) InstallCertsOpenShift(ctx context.Context, workDir string, registryHost string, mcTimeout time.Duration) error {
	certPath := filepath.Join(workDir, ".registry_cert.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil
	}

	certData, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	if c.k8sClient == nil || c.k8sClient.OpenShift() == nil {
		return fmt.Errorf("OpenShift client not initialized")
	}

	osClient := c.k8sClient.OpenShift()

	// Create/update registry-config ConfigMap in openshift-config namespace
	registryKey := strings.ReplaceAll(registryHost, ":", "..")
	cm := k8sclient.NewConfigMap("openshift-config", "registry-config", map[string]string{
		registryKey: string(certData),
	})

	if err := c.k8sClient.CreateOrUpdateConfigMap(ctx, cm); err != nil {
		return fmt.Errorf("failed to create registry-config configmap: %w", err)
	}

	// Patch image.config.openshift.io/cluster
	if err := osClient.PatchImageConfig(ctx, "registry-config"); err != nil {
		return fmt.Errorf("failed to patch image config: %w", err)
	}

	// Create MachineConfig for worker nodes
	mcName := fmt.Sprintf("99-registry-ca-%s", strings.ReplaceAll(registryHost, ":", "-"))
	if err := osClient.CreateRegistryCertMachineConfig(ctx, mcName, registryHost, certData); err != nil {
		return fmt.Errorf("failed to create machineconfig: %w", err)
	}

	// Wait for MachineConfigPool to update
	if err := osClient.WaitForMachineConfigPoolUpdate(ctx, "worker", mcTimeout); err != nil {
		return fmt.Errorf("timeout waiting for machineconfig rollout: %w", err)
	}

	return nil
}

// InstallCertsEKS installs registry certificates on EKS worker nodes via SSH.
// Adds the cert to the system trust store and restarts containerd.
func (c *Client) InstallCertsEKS(ctx context.Context, workDir string, registryHost string) error {
	certPath := filepath.Join(workDir, ".registry_cert.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		// No certificate to install
		return nil
	}

	// Require EKS SSH credentials
	if c.Config.EKSSSHUser == "" || c.Config.EKSSSHKeyPath == "" {
		return fmt.Errorf("EKS SSH credentials (eks_ssh_user, eks_ssh_key_path) are required for certificate installation")
	}

	// Create SSH client with EKS-specific credentials (passphrase passed as password)
	eksSSH, err := NewSSHClient(c.Config.EKSSSHUser, c.Config.EKSSSHKeyPassphrase, c.Config.EKSSSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create EKS SSH client: %w", err)
	}

	// Get worker node names from k8s API
	if c.k8sClient == nil {
		return fmt.Errorf("k8s client not initialized")
	}
	nodeNames, err := c.k8sClient.ListWorkerNodeNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list worker nodes: %w", err)
	}

	if len(nodeNames) == 0 {
		return fmt.Errorf("no worker nodes found in cluster")
	}

	// Resolve node addresses based on hostname type and install cert on each
	for _, nodeName := range nodeNames {
		var nodeAddr string
		switch c.Config.EKSHostnameType {
		case "public":
			nodeAddr, err = c.k8sClient.GetNodeExternalIP(ctx, nodeName)
			if err != nil {
				// Fall back to internal IP if no external IP
				nodeAddr, err = c.k8sClient.GetNodeInternalIP(ctx, nodeName)
				if err != nil {
					return fmt.Errorf("failed to get IP for node %s: %w", nodeName, err)
				}
			}
		default: // "private" or unset
			nodeAddr, err = c.k8sClient.GetNodeInternalIP(ctx, nodeName)
			if err != nil {
				return fmt.Errorf("failed to get internal IP for node %s: %w", nodeName, err)
			}
		}

		// Copy certificate to the node
		if err := eksSSH.CopyTo(nodeAddr, certPath, "/tmp/registry_ca.crt"); err != nil {
			return fmt.Errorf("failed to copy cert to node %s (%s): %w", nodeName, nodeAddr, err)
		}

		// Install cert in /etc/docker/certs.d/<registryHost>/ca.crt
		// and in system trust store (Amazon Linux 2/2023), then restart containerd
		certDir := fmt.Sprintf("/etc/docker/certs.d/%s", registryHost)
		certFileName := strings.ReplaceAll(registryHost, ":", "_")
		installCmd := fmt.Sprintf(
			"sudo mkdir -p '%s' && "+
				"sudo cp /tmp/registry_ca.crt '%s/ca.crt' && "+
				"sudo cp /tmp/registry_ca.crt '/etc/pki/ca-trust/source/anchors/%s.crt' && "+
				"sudo update-ca-trust extract && "+
				"sudo systemctl restart containerd && "+
				"rm -f /tmp/registry_ca.crt",
			certDir, certDir, certFileName,
		)
		if _, err := eksSSH.Run(nodeAddr, installCmd); err != nil {
			return fmt.Errorf("failed to install cert on node %s (%s): %w", nodeName, nodeAddr, err)
		}
	}

	return nil
}

// DeployEdge deploys edge components to the cluster
func (c *Client) DeployEdge(ctx context.Context, workDir, namespace, platform string) error {
	if c.k8sClient == nil {
		return fmt.Errorf("k8s client not initialized")
	}

	// Create namespace and wait for it to be active
	if err := c.k8sClient.CreateNamespaceAndWait(ctx, namespace, 60*time.Second); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// For OpenShift, apply SCC and RBAC from openshift subdirectory first
	if platform == "openshift" {
		openshiftDir := filepath.Join(workDir, "openshift")
		if _, err := os.Stat(openshiftDir); err == nil {
			// Create gatekeeper-system namespace and wait for it to be active
			if err := c.k8sClient.CreateNamespaceAndWait(ctx, "gatekeeper-system", 60*time.Second); err != nil {
				return fmt.Errorf("failed to create gatekeeper-system namespace: %w", err)
			}

			// Apply all YAML files in openshift subdirectory
			if err := c.k8sClient.ApplyDirectory(ctx, openshiftDir, namespace); err != nil {
				return fmt.Errorf("failed to apply openshift manifests: %w", err)
			}
		}
	}

	// Apply all manifests in the work directory
	if err := c.k8sClient.ApplyDirectory(ctx, workDir, namespace); err != nil {
		return fmt.Errorf("failed to apply manifests: %w", err)
	}

	return nil
}

// MonitorDeployment monitors the edge deployment status
func (c *Client) MonitorDeployment(ctx context.Context, namespace string, maxAttempts, sleepInterval int) (string, error) {
	if c.k8sClient == nil {
		return "", fmt.Errorf("k8s client not initialized")
	}

	cmName := "edge-controller-client-cm"
	fieldName := "EDGE_SERVICES_INSTALLATION_STATUS"

	status, err := c.k8sClient.WaitForConfigMapField(ctx, namespace, cmName, fieldName,
		func(value string) (bool, error) {
			switch value {
			case "Completed":
				return true, nil
			case "Failed":
				return false, fmt.Errorf("deployment failed")
			default:
				return false, nil
			}
		},
		time.Duration(sleepInterval)*time.Second,
		maxAttempts,
	)

	return status, err
}

// DeleteEdge removes edge components from the cluster
func (c *Client) DeleteEdge(ctx context.Context, workDir, namespace string) error {
	if c.k8sClient == nil {
		return fmt.Errorf("k8s client not initialized")
	}

	// Delete all resources from manifests
	if err := c.k8sClient.DeleteDirectory(ctx, workDir, namespace); err != nil {
		// Log but don't fail on delete errors
		fmt.Printf("Warning: failed to delete some resources: %v\n", err)
	}

	// Clean up the rook-ceph namespace if it is stuck in Terminating (e.g.
	// after an interrupted or failed rook-ceph provider uninstall).  This is a
	// best-effort fallback: errors are logged but do not fail DeleteEdge.
	if err := c.k8sClient.CleanupTerminatingNamespace(ctx, "rook-ceph"); err != nil {
		fmt.Printf("Warning: failed to clean up rook-ceph namespace: %v\n", err)
	}

	// Delete namespace
	if err := c.k8sClient.DeleteNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	// Wait for namespace deletion with finalizer cleanup
	// This will automatically remove finalizers from resources blocking deletion
	timeout := 10 * time.Minute
	if err := c.k8sClient.WaitForNamespaceDeletion(ctx, namespace, timeout); err != nil {
		return fmt.Errorf("failed to wait for namespace deletion: %w", err)
	}

	return nil
}

// InstallMetricsServer installs the Kubernetes Metrics Server using the native K8s client.
// In airgap mode, YAML manifests are read from airgapPath.
// In online mode, the manifest is fetched from the official GitHub release URL.
func (c *Client) InstallMetricsServer(ctx context.Context, airgap bool, airgapPath string) error {
	if c.k8sClient == nil {
		return fmt.Errorf("k8s client not initialized")
	}

	if airgap {
		if airgapPath == "" {
			return fmt.Errorf("k8s_metrics_server_airgap_install_path is required when k8s_metrics_server_airgap_install is true")
		}
		return c.k8sClient.ApplyDirectory(ctx, airgapPath, "kube-system")
	}

	// Online mode: fetch manifest from GitHub
	const metricsServerURL = "https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"
	httpClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := httpClient.Get(metricsServerURL)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics-server manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch metrics-server manifest: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read metrics-server manifest: %w", err)
	}

	return c.k8sClient.ApplyManifestContent(ctx, content, "kube-system")
}

// CleanupBundle removes the downloaded bundle directory
func (c *Client) CleanupBundle(bundleDir string) error {
	return os.RemoveAll(bundleDir)
}
