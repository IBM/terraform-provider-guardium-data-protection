// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k3sclient

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/edgeclient"
)

// Client manages SSH connections and K3S operations on remote nodes
type Client struct {
	SSHUser     string
	SSHPassword string
	SSHOptions  SSHOptions
}

// UninstallK3S removes k3s from a node
func (c *Client) UninstallK3S(host string, isServer bool) error {
	var script string
	if isServer {
		script = "/usr/local/bin/k3s-uninstall.sh"
	} else {
		script = "/usr/local/bin/k3s-agent-uninstall.sh"
	}

	// Check if uninstall script exists
	checkCmd := fmt.Sprintf("test -f %s && echo exists", script)
	output, err := c.RunSSH(host, checkCmd)
	if err != nil {
		return fmt.Errorf("failed to check uninstall script on %s: %w", host, err)
	}
	if output == "" {
		return nil // Already uninstalled or never installed
	}

	_, err = c.RunSSH(host, script)
	return err
}

// CheckK3SInstalled checks if k3s is installed on a node
func (c *Client) CheckK3SInstalled(host string) (bool, error) {
	output, err := c.RunSSH(host, "which k3s 2>/dev/null || echo notfound")
	if err != nil {
		return false, err
	}
	return output != "notfound\n" && output != "notfound", nil
}

// VerifyCluster checks cluster health and returns node status
func (c *Client) VerifyCluster(config K3SInstallConfig) (string, error) {
	if len(config.MasterNodes) == 0 {
		return "", fmt.Errorf("no master nodes configured")
	}
	primaryMaster := config.MasterNodes[0]
	return c.RunSSH(primaryMaster, "k3s kubectl get nodes -o wide")
}

// WaitForNodes waits until all expected nodes are in Ready state
func (c *Client) WaitForNodes(config K3SInstallConfig) error {
	if len(config.MasterNodes) == 0 {
		return fmt.Errorf("no master nodes configured")
	}

	timeout, err := time.ParseDuration(config.NodeWaitTimeout)
	if err != nil {
		timeout = 600 * time.Second
	}

	primaryMaster := config.MasterNodes[0]
	expectedNodes := len(config.MasterNodes) + len(config.WorkerNodes)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		output, err := c.RunSSH(primaryMaster, "k3s kubectl get nodes --no-headers")
		if err == nil {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			readyCount := 0
			for _, line := range lines {
				if strings.Contains(line, " Ready") && !strings.Contains(line, "NotReady") {
					readyCount++
				}
			}
			if readyCount >= expectedNodes {
				return nil
			}
		}
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timed out waiting for %d nodes to be Ready after %s", expectedNodes, timeout)
}

// prepareAirgapInstall moves files from /tmp to the correct locations for k3s airgap install
func (c *Client) prepareAirgapInstall(host string) error {
	// Move k3s binary to /usr/local/bin
	if _, err := c.RunSSH(host, "cp /tmp/k3s /usr/local/bin/k3s && chmod +x /usr/local/bin/k3s"); err != nil {
		return fmt.Errorf("failed to install k3s binary: %w", err)
	}

	// Create images directory and move images tarball
	setupImagesCmd := "mkdir -p /var/lib/rancher/k3s/agent/images && " +
		"cp /tmp/k3s-airgap-images.tar /var/lib/rancher/k3s/agent/images/"
	if _, err := c.RunSSH(host, setupImagesCmd); err != nil {
		return fmt.Errorf("failed to setup airgap images: %w", err)
	}

	return nil
}

// installEnvVars returns the environment variable prefix for k3s install commands
func (config K3SInstallConfig) installEnvVars() string {
	if config.AirgapInstall {
		return "INSTALL_K3S_SKIP_DOWNLOAD=true"
	}
	if config.Version != "" {
		version := config.Version
		if !strings.Contains(version, "+") {
			version = version + "+k3s1"
		}
		return fmt.Sprintf("INSTALL_K3S_VERSION=%s", version)
	}
	return ""
}

// prepareOnlineInstall downloads the K3S install script from https://get.k3s.io
// and uploads it to /tmp/k3s-install.sh on the remote host via SFTP
func (c *Client) prepareOnlineInstall(host string) error {
	resp, err := http.Get("https://get.k3s.io")
	if err != nil {
		return fmt.Errorf("failed to download k3s install script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download k3s install script: HTTP %d", resp.StatusCode)
	}

	scriptData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read k3s install script: %w", err)
	}

	// Upload to remote host via SFTP
	sshClient, cleanup, err := c.sshDial(host)
	if err != nil {
		return err
	}
	defer cleanup()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Create("/tmp/k3s-install.sh")
	if err != nil {
		return fmt.Errorf("failed to create remote install script: %w", err)
	}
	defer remoteFile.Close()

	if _, err := remoteFile.Write(scriptData); err != nil {
		return fmt.Errorf("failed to write install script to remote: %w", err)
	}

	// Make executable
	if _, err := c.RunSSH(host, "chmod +x /tmp/k3s-install.sh"); err != nil {
		return fmt.Errorf("failed to make install script executable: %w", err)
	}

	return nil
}

// InstallK3SWorker installs k3s agent on a worker node
func (c *Client) InstallK3SWorker(config K3SInstallConfig, masterIP string, worker string, nodeIndex int) error {
	if config.AirgapInstall && config.AirgapInstallationPath != "" {
		if err := c.UploadAirgapFiles(worker, config.AirgapInstallationPath); err != nil {
			return fmt.Errorf("failed to upload airgap files to worker %s: %w", worker, err)
		}
		if err := c.prepareAirgapInstall(worker); err != nil {
			return fmt.Errorf("failed to prepare airgap install on worker %s: %w", worker, err)
		}
	} else {
		if err := c.prepareOnlineInstall(worker); err != nil {
			return fmt.Errorf("failed to prepare online install on worker %s: %w", worker, err)
		}
	}

	// Build install command
	envVars := config.installEnvVars()
	installCmd := fmt.Sprintf(
		"%s K3S_URL=https://%s:6443 K3S_TOKEN=%s /tmp/k3s-install.sh",
		envVars, masterIP, config.Token,
	)

	_, err := c.RunSSH(worker, installCmd)
	return err
}

// InstallK3SAdditionalMaster installs k3s server on additional master node
func (c *Client) InstallK3SAdditionalMaster(config K3SInstallConfig, masterIP string, node string, nodeIndex int) error {
	if config.AirgapInstall && config.AirgapInstallationPath != "" {
		if err := c.UploadAirgapFiles(node, config.AirgapInstallationPath); err != nil {
			return fmt.Errorf("failed to upload airgap files to master %s: %w", node, err)
		}
		if err := c.prepareAirgapInstall(node); err != nil {
			return fmt.Errorf("failed to prepare airgap install on master %s: %w", node, err)
		}
	} else {
		if err := c.prepareOnlineInstall(node); err != nil {
			return fmt.Errorf("failed to prepare online install on master %s: %w", node, err)
		}
	}

	// Build install command for additional server
	installOpts := fmt.Sprintf("--server https://%s:6443", masterIP)
	if config.DisableTraefik {
		installOpts += " --disable traefik"
	}
	if config.TaintMasters {
		installOpts += " --node-taint CriticalAddonsOnly=true:NoExecute"
	}

	envVars := config.installEnvVars()
	installCmd := fmt.Sprintf(
		"%s K3S_TOKEN=%s /tmp/k3s-install.sh server %s",
		envVars, config.Token, installOpts,
	)

	_, err := c.RunSSH(node, installCmd)
	return err
}

// InstallK3SPrimaryMaster installs k3s server on the primary master node
func (c *Client) InstallK3SPrimaryMaster(config K3SInstallConfig) error {
	if len(config.MasterNodes) == 0 {
		return fmt.Errorf("no master nodes configured")
	}
	primaryMaster := config.MasterNodes[0]

	if config.AirgapInstall && config.AirgapInstallationPath != "" {
		if err := c.UploadAirgapFiles(primaryMaster, config.AirgapInstallationPath); err != nil {
			return fmt.Errorf("failed to upload airgap files to primary master: %w", err)
		}
		if err := c.prepareAirgapInstall(primaryMaster); err != nil {
			return fmt.Errorf("failed to prepare airgap install on primary master: %w", err)
		}
	} else {
		if err := c.prepareOnlineInstall(primaryMaster); err != nil {
			return fmt.Errorf("failed to prepare online install on primary master: %w", err)
		}
	}

	// Build install options
	installOpts := "--cluster-init"
	if config.DisableTraefik {
		installOpts += " --disable traefik"
	}
	if config.TaintMasters {
		installOpts += " --node-taint CriticalAddonsOnly=true:NoExecute"
	}

	envVars := config.installEnvVars()
	installCmd := fmt.Sprintf(
		"%s K3S_TOKEN=%s /tmp/k3s-install.sh server %s",
		envVars, config.Token, installOpts,
	)

	_, err := c.RunSSH(primaryMaster, installCmd)
	return err
}

// InstallK3SSingleNode installs k3s on a single node (no HA)
func (c *Client) InstallK3SSingleNode(config K3SInstallConfig) error {
	if len(config.MasterNodes) == 0 {
		return fmt.Errorf("no master nodes configured")
	}
	node := config.MasterNodes[0]

	if config.AirgapInstall && config.AirgapInstallationPath != "" {
		if err := c.UploadAirgapFiles(node, config.AirgapInstallationPath); err != nil {
			return fmt.Errorf("failed to upload airgap files: %w", err)
		}
		if err := c.prepareAirgapInstall(node); err != nil {
			return fmt.Errorf("failed to prepare airgap install: %w", err)
		}
	} else {
		if err := c.prepareOnlineInstall(node); err != nil {
			return fmt.Errorf("failed to prepare online install: %w", err)
		}
	}

	// Build install options
	var installOpts string
	if config.DisableTraefik {
		installOpts += " --disable traefik"
	}

	envVars := config.installEnvVars()
	installCmd := fmt.Sprintf(
		"%s K3S_TOKEN=%s /tmp/k3s-install.sh server %s",
		envVars, config.Token, installOpts,
	)

	_, err := c.RunSSH(node, installCmd)
	return err
}

// SSHOptions holds SSH connection parameters
type SSHOptions struct {
	ConnectTimeout      int
	ServerAliveInterval int
	ServerAliveCount    int
	KnownHostsFile      string // path to known_hosts file for SSH host key verification; leave empty to disable verification
}

// K3SInstallConfig holds K3S installation parameters
type K3SInstallConfig struct {
	ClusterName            string
	Version                string
	Token                  string
	MasterNodes            []string
	WorkerNodes            []string
	DisableTraefik         bool
	TaintMasters           bool
	NodeWaitTimeout        string
	AirgapInstall          bool
	AirgapInstallationPath string
}

// NewClient creates a new SSH client for K3S operations
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

	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}

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

// RunSSH executes a command on a remote host using native Go SSH
func (c *Client) RunSSH(host string, command string) (string, error) {
	client, cleanup, err := c.sshDial(host)
	if err != nil {
		return "", err
	}
	defer cleanup()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session on %s: %w", host, err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	output := stdout.String() + stderr.String()
	if err != nil {
		return output, fmt.Errorf("SSH command failed on %s: %w\nOutput: %s", host, err, output)
	}
	return output, nil
}

// UploadFile uploads a local file to a remote host via SFTP
func (c *Client) UploadFile(host, localPath, remotePath string) error {
	client, cleanup, err := c.sshDial(host)
	if err != nil {
		return err
	}
	defer cleanup()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
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
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("failed to upload file to %s: %w", remotePath, err)
	}

	return nil
}

// UploadAirgapFiles uploads k3s binary, images, and install script to /tmp/ on a remote node
func (c *Client) UploadAirgapFiles(host string, airgapPath string) error {
	remoteDir := "/tmp"

	// Upload k3s binary
	k3sBinary := filepath.Join(airgapPath, "k3s")
	if _, err := os.Stat(k3sBinary); err == nil {
		if err := c.UploadFile(host, k3sBinary, filepath.Join(remoteDir, "k3s")); err != nil {
			return fmt.Errorf("failed to upload k3s binary: %w", err)
		}
		if _, err := c.RunSSH(host, "chmod +x "+remoteDir+"/k3s"); err != nil {
			return fmt.Errorf("failed to make k3s executable: %w", err)
		}
	}

	// Upload airgap images tar if exists
	imagesTar := filepath.Join(airgapPath, "k3s-airgap-images.tar")
	if _, err := os.Stat(imagesTar); err == nil {
		if err := c.UploadFile(host, imagesTar, filepath.Join(remoteDir, "k3s-airgap-images.tar")); err != nil {
			return fmt.Errorf("failed to upload airgap images: %w", err)
		}
	}

	// Upload install script if exists
	installScript := filepath.Join(airgapPath, "install.sh")
	if _, err := os.Stat(installScript); err == nil {
		if err := c.UploadFile(host, installScript, filepath.Join(remoteDir, "k3s-install.sh")); err != nil {
			return fmt.Errorf("failed to upload install script: %w", err)
		}
		if _, err := c.RunSSH(host, "chmod +x "+remoteDir+"/k3s-install.sh"); err != nil {
			return fmt.Errorf("failed to make install script executable: %w", err)
		}
	}

	return nil
}
