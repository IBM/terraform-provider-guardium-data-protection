// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package edgeclient

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"log"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHClient wraps an SSH connection
type SSHClient struct {
	config *ssh.ClientConfig
	user   string
}

// HostKeyCallback builds an ssh.HostKeyCallback that verifies server host
// keys against knownHostsFile (standard OpenSSH known_hosts format). If
// knownHostsFile is empty, it falls back to accepting any host key —
// callers should warn when doing so, since this permits MITM attacks.
func HostKeyCallback(knownHostsFile string) (ssh.HostKeyCallback, error) {
	if knownHostsFile == "" {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec // explicit opt-out when no known_hosts file is configured
	}

	cb, err := knownhosts.New(knownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts file %q: %w", knownHostsFile, err)
	}
	return cb, nil
}

// NewSSHClient creates a new SSH client with password or key authentication.
// For passphrase-protected keys, pass the passphrase as the password parameter
// when keyPath is also set. If knownHostsFile is empty, host key verification
// is skipped (StrictHostKeyChecking=no) and a warning is logged.
func NewSSHClient(user, password, keyPath, knownHostsFile string) (*SSHClient, error) {
	var authMethods []ssh.AuthMethod

	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			// Key might be passphrase-protected — try with password as passphrase
			if password != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(password))
				if err != nil {
					return nil, fmt.Errorf("failed to parse SSH key with passphrase: %w", err)
				}
				// Password was used as passphrase, don't also add it as password auth
				password = ""
			} else {
				return nil, fmt.Errorf("failed to parse SSH key (passphrase required?): %w", err)
			}
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method provided")
	}

	hostKeyCallback, err := HostKeyCallback(knownHostsFile)
	if err != nil {
		return nil, err
	}
	if knownHostsFile == "" {
		log.Printf("[WARN] SSH host key verification is disabled (no known_hosts file configured) — connections are vulnerable to MITM attacks")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	return &SSHClient{
		config: config,
		user:   user,
	}, nil
}

// Run executes a command on a remote host
func (c *SSHClient) Run(host string, command string) (string, error) {
	// Add default SSH port if not specified
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "22")
	}

	client, err := ssh.Dial("tcp", host, c.config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// CopyTo copies a local file to a remote host (SCP upload).
// Implements the SCP sink-mode protocol including acknowledgment handshake.
func (c *SSHClient) CopyTo(host, localPath, remotePath string) error {
	log.Printf("[DEBUG] CopyTo: localPath=%s host=%s", localPath, host)
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "22")
	}

	client, err := ssh.Dial("tcp", host, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", host, err)
	}
	defer client.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	stat, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	remoteDir := filepath.Dir(remotePath)
	remoteFile := filepath.Base(remotePath)

	// Start the remote SCP receiver FIRST, then drive the protocol.
	// Using Start+Wait instead of Run so we control the write order.
	if err := session.Start(fmt.Sprintf("scp -t %s", remoteDir)); err != nil {
		return fmt.Errorf("failed to start SCP: %w", err)
	}

	// readAck reads the one-byte SCP acknowledgment sent by the remote after
	// each protocol step (0x00 = OK, 0x01 = warning, 0x02 = fatal error).
	readAck := func() error {
		buf := make([]byte, 1)
		if _, err := io.ReadFull(stdout, buf); err != nil {
			return fmt.Errorf("failed to read SCP ack: %w", err)
		}
		if buf[0] != 0 {
			return fmt.Errorf("SCP error response (code %d)", buf[0])
		}
		return nil
	}

	// Step 1: remote scp signals it is ready
	if err := readAck(); err != nil {
		return fmt.Errorf("SCP ready signal: %w", err)
	}

	// Step 2: send file header
	if _, err := fmt.Fprintf(stdin, "C0644 %d %s\n", stat.Size(), remoteFile); err != nil {
		return fmt.Errorf("failed to write SCP header: %w", err)
	}
	if err := readAck(); err != nil {
		return fmt.Errorf("SCP header ack: %w", err)
	}

	// Step 3: send file content
	if _, err := io.Copy(stdin, localFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Step 4: send end-of-transfer marker and read final ack
	if _, err := fmt.Fprint(stdin, "\x00"); err != nil {
		return fmt.Errorf("failed to write SCP EOT: %w", err)
	}
	if err := readAck(); err != nil {
		return fmt.Errorf("SCP final ack: %w", err)
	}

	stdin.Close()
	return session.Wait()
}

// CopyFrom copies a file from a remote host to local (SCP download)
func (c *SSHClient) CopyFrom(host, remotePath, localPath string) error {
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "22")
	}

	client, err := ssh.Dial("tcp", host, c.config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use cat to read the file content (simpler than SCP protocol)
	output, err := session.Output(fmt.Sprintf("cat '%s'", remotePath))
	if err != nil {
		return fmt.Errorf("failed to read remote file: %w", err)
	}

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Write to local file
	if err := os.WriteFile(localPath, output, 0600); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// RunWithSudo executes a command with sudo on a remote host
func (c *SSHClient) RunWithSudo(host string, command string) (string, error) {
	return c.Run(host, fmt.Sprintf("sudo %s", command))
}
