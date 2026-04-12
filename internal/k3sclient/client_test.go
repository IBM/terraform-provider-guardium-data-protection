// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k3sclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewClient tests the NewClient constructor
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		sshUser     string
		sshPassword string
		opts        SSHOptions
		expectUser  string
		description string
	}{
		{
			name:        "with_user_and_password",
			sshUser:     "testuser",
			sshPassword: "testpass",
			opts: SSHOptions{
				ConnectTimeout:      30,
				ServerAliveInterval: 60,
				ServerAliveCount:    3,
			},
			expectUser:  "testuser",
			description: "Should create client with specified user",
		},
		{
			name:        "with_empty_user",
			sshUser:     "",
			sshPassword: "testpass",
			opts: SSHOptions{
				ConnectTimeout: 30,
			},
			expectUser:  "root",
			description: "Should default to root user",
		},
		{
			name:        "with_custom_options",
			sshUser:     "admin",
			sshPassword: "adminpass",
			opts: SSHOptions{
				ConnectTimeout:      60,
				ServerAliveInterval: 120,
				ServerAliveCount:    5,
			},
			expectUser:  "admin",
			description: "Should create client with custom SSH options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.sshUser, tt.sshPassword, tt.opts)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}

			if client.SSHUser != tt.expectUser {
				t.Errorf("SSHUser = %v, want %v", client.SSHUser, tt.expectUser)
			}

			if client.SSHPassword != tt.sshPassword {
				t.Errorf("SSHPassword = %v, want %v", client.SSHPassword, tt.sshPassword)
			}

			if client.SSHOptions.ConnectTimeout != tt.opts.ConnectTimeout {
				t.Errorf("ConnectTimeout = %v, want %v", client.SSHOptions.ConnectTimeout, tt.opts.ConnectTimeout)
			}
		})
	}
}

// TestK3SInstallConfig_Fields tests the K3SInstallConfig struct
func TestK3SInstallConfig_Fields(t *testing.T) {
	config := K3SInstallConfig{
		ClusterName:            "test-cluster",
		Version:                "v1.28.0",
		Token:                  "test-token",
		MasterNodes:            []string{"master1", "master2"},
		WorkerNodes:            []string{"worker1", "worker2"},
		DisableTraefik:         true,
		TaintMasters:           true,
		NodeWaitTimeout:        "600s",
		AirgapInstall:          false,
		AirgapInstallationPath: "/path/to/airgap",
	}

	// Verify all fields are accessible
	if config.ClusterName != "test-cluster" {
		t.Error("ClusterName mismatch")
	}
	if config.Version != "v1.28.0" {
		t.Error("Version mismatch")
	}
	if len(config.MasterNodes) != 2 {
		t.Error("MasterNodes count mismatch")
	}
	if len(config.WorkerNodes) != 2 {
		t.Error("WorkerNodes count mismatch")
	}
	if !config.DisableTraefik {
		t.Error("DisableTraefik should be true")
	}
	if !config.TaintMasters {
		t.Error("TaintMasters should be true")
	}
}

// TestK3SInstallConfig_InstallEnvVars tests environment variable generation
func TestK3SInstallConfig_InstallEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		config   K3SInstallConfig
		expected string
	}{
		{
			name: "airgap_install",
			config: K3SInstallConfig{
				AirgapInstall: true,
			},
			expected: "INSTALL_K3S_SKIP_DOWNLOAD=true",
		},
		{
			name: "with_version",
			config: K3SInstallConfig{
				Version: "v1.28.0",
			},
			expected: "INSTALL_K3S_VERSION=v1.28.0+k3s1",
		},
		{
			name: "with_version_already_has_suffix",
			config: K3SInstallConfig{
				Version: "v1.28.0+k3s2",
			},
			expected: "INSTALL_K3S_VERSION=v1.28.0+k3s2",
		},
		{
			name:     "no_version",
			config:   K3SInstallConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.installEnvVars()
			if result != tt.expected {
				t.Errorf("installEnvVars() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestClient_CheckK3SInstalled tests K3S installation check
func TestClient_CheckK3SInstalled(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	installed, err := client.CheckK3SInstalled("127.0.0.1")
	if err == nil {
		t.Fatal("Expected error without SSH server")
	}
	if installed {
		t.Error("Expected installed to be false on SSH failure")
	}
}

// TestClient_UninstallK3S tests K3S uninstallation
func TestClient_UninstallK3S(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	t.Run("uninstall_server", func(t *testing.T) {
		err := client.UninstallK3S("127.0.0.1", true)
		if err == nil {
			t.Fatal("Expected error without SSH server")
		}
		if !strings.Contains(err.Error(), "failed to check uninstall script") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("uninstall_agent", func(t *testing.T) {
		err := client.UninstallK3S("127.0.0.1", false)
		if err == nil {
			t.Fatal("Expected error without SSH server")
		}
		if !strings.Contains(err.Error(), "failed to check uninstall script") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestClient_VerifyCluster tests cluster verification
func TestClient_VerifyCluster(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	t.Run("no_master_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes: []string{},
		}
		_, err := client.VerifyCluster(config)
		if err == nil {
			t.Fatal("Expected error with no master nodes")
		}
		if err.Error() != "no master nodes configured" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("with_master_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes: []string{"127.0.0.1"},
		}
		_, err := client.VerifyCluster(config)
		if err == nil {
			t.Fatal("Expected error without SSH server")
		}
	})
}

// TestClient_WaitForNodes tests node readiness waiting
func TestClient_WaitForNodes(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	t.Run("no_master_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes: []string{},
		}
		err := client.WaitForNodes(config)
		if err == nil {
			t.Fatal("Expected error with no master nodes")
		}
		if err.Error() != "no master nodes configured" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("with_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes:     []string{"127.0.0.1"},
			WorkerNodes:     []string{"worker1"},
			NodeWaitTimeout: "0s",
		}
		err := client.WaitForNodes(config)
		if err == nil {
			t.Fatal("Expected timeout waiting for nodes")
		}
		if !strings.Contains(err.Error(), "timed out waiting for 2 nodes to be Ready") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestClient_InstallK3SSingleNode tests single node installation
func TestClient_InstallK3SSingleNode(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	t.Run("no_master_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes: []string{},
		}
		err := client.InstallK3SSingleNode(config)
		if err == nil {
			t.Error("Expected error with no master nodes")
		}
		if err.Error() != "no master nodes configured" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("with_master_node", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes:    []string{"127.0.0.1:19999"},
			Token:          "test-token",
			DisableTraefik: true,
		}
		err := client.InstallK3SSingleNode(config)
		if err == nil {
			t.Error("Expected error without SSH server")
		}
	})
}

// TestClient_InstallK3SPrimaryMaster tests primary master installation
func TestClient_InstallK3SPrimaryMaster(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	t.Run("no_master_nodes", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes: []string{},
		}
		err := client.InstallK3SPrimaryMaster(config)
		if err == nil {
			t.Error("Expected error with no master nodes")
		}
		if err.Error() != "no master nodes configured" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("with_master_node", func(t *testing.T) {
		config := K3SInstallConfig{
			MasterNodes:    []string{"127.0.0.1:19999"},
			Token:          "test-token",
			DisableTraefik: true,
			TaintMasters:   true,
		}
		err := client.InstallK3SPrimaryMaster(config)
		if err == nil {
			t.Error("Expected error without SSH server")
		}
	})
}

// TestClient_InstallK3SAdditionalMaster tests additional master installation
func TestClient_InstallK3SAdditionalMaster(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	config := K3SInstallConfig{
		Token:          "test-token",
		DisableTraefik: true,
		TaintMasters:   true,
	}

	err := client.InstallK3SAdditionalMaster(config, "127.0.0.1:19999", "127.0.0.1:19999", 1)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_InstallK3SWorker tests worker node installation
func TestClient_InstallK3SWorker(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	config := K3SInstallConfig{
		Token: "test-token",
	}

	err := client.InstallK3SWorker(config, "127.0.0.1:19999", "127.0.0.1:19999", 0)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_RunSSH tests SSH command execution fails without a server
func TestClient_RunSSH(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	_, err := client.RunSSH("127.0.0.1:19999", "echo test")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_UploadFile tests file upload fails without a server
func TestClient_UploadFile(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(localFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := client.UploadFile("127.0.0.1:19999", localFile, "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_UploadAirgapFiles tests airgap file upload fails without a server
func TestClient_UploadAirgapFiles(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	tmpDir := t.TempDir()

	// Create mock airgap files
	k3sBinary := filepath.Join(tmpDir, "k3s")
	if err := os.WriteFile(k3sBinary, []byte("mock binary"), 0755); err != nil {
		t.Fatal(err)
	}

	imagesTar := filepath.Join(tmpDir, "k3s-airgap-images.tar")
	if err := os.WriteFile(imagesTar, []byte("mock tar"), 0644); err != nil {
		t.Fatal(err)
	}

	installScript := filepath.Join(tmpDir, "install.sh")
	if err := os.WriteFile(installScript, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	err := client.UploadAirgapFiles("127.0.0.1:19999", tmpDir)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestSSHOptions_Fields tests SSHOptions struct
func TestSSHOptions_Fields(t *testing.T) {
	opts := SSHOptions{
		ConnectTimeout:      60,
		ServerAliveInterval: 120,
		ServerAliveCount:    5,
	}

	if opts.ConnectTimeout != 60 {
		t.Errorf("ConnectTimeout = %v, want 60", opts.ConnectTimeout)
	}
	if opts.ServerAliveInterval != 120 {
		t.Errorf("ServerAliveInterval = %v, want 120", opts.ServerAliveInterval)
	}
	if opts.ServerAliveCount != 5 {
		t.Errorf("ServerAliveCount = %v, want 5", opts.ServerAliveCount)
	}
}

// TestClient_PrepareAirgapInstall tests airgap preparation
func TestClient_PrepareAirgapInstall(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	err := client.prepareAirgapInstall("127.0.0.1:19999")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_PrepareOnlineInstall tests online installation preparation
func TestClient_PrepareOnlineInstall(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	err := client.prepareOnlineInstall("127.0.0.1:19999")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// Made with Bob
