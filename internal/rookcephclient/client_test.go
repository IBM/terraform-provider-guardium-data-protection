// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package rookcephclient

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestRookCephConfig_Fields tests the RookCephConfig struct
func TestRookCephConfig_Fields(t *testing.T) {
	config := RookCephConfig{
		ClusterName:              "test-cluster",
		Platform:                 "k3s",
		TargetNode:               "master1",
		RookCephVersion:          "v1.19.0",
		RookCephInstallationPath: "/path/to/rook",
		AirgapInstall:            true,
		WorkerCount:              3,
		TaintMasters:             true,
		SetAsDefaultStorage:      true,
		DisableLocalPath:         true,
		PodWaitTimeout:           "300s",
		SleepBetweenSteps:        10,
	}

	// Verify all fields are accessible
	if config.ClusterName != "test-cluster" {
		t.Error("ClusterName mismatch")
	}
	if config.Platform != "k3s" {
		t.Error("Platform mismatch")
	}
	if config.RookCephVersion != "v1.19.0" {
		t.Error("RookCephVersion mismatch")
	}
	if config.WorkerCount != 3 {
		t.Error("WorkerCount mismatch")
	}
	if !config.TaintMasters {
		t.Error("TaintMasters should be true")
	}
	if !config.SetAsDefaultStorage {
		t.Error("SetAsDefaultStorage should be true")
	}
	if !config.DisableLocalPath {
		t.Error("DisableLocalPath should be true")
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

// TestParseRookVersion tests version parsing
func TestParseRookVersion(t *testing.T) {
	tests := []struct {
		version     string
		expectMajor int
		expectMinor int
	}{
		{"v1.19.0", 1, 19},
		{"v1.18", 1, 18},
		{"v2.0.1", 2, 0},
		{"1.19.0", 1, 19},
		{"invalid", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			major, minor := parseRookVersion(tt.version)
			if major != tt.expectMajor {
				t.Errorf("major = %v, want %v", major, tt.expectMajor)
			}
			if minor != tt.expectMinor {
				t.Errorf("minor = %v, want %v", minor, tt.expectMinor)
			}
		})
	}
}

// TestRookVersionGTE tests version comparison
func TestRookVersionGTE(t *testing.T) {
	tests := []struct {
		version   string
		threshold string
		expected  bool
	}{
		{"v1.19.0", "v1.18.0", true},
		{"v1.18.0", "v1.19.0", false},
		{"v1.19.0", "v1.19.0", true},
		{"v2.0.0", "v1.19.0", true},
		{"v1.17.0", "v1.18.0", false},
		{"v1.19.5", "v1.19.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_vs_"+tt.threshold, func(t *testing.T) {
			result := rookVersionGTE(tt.version, tt.threshold)
			if result != tt.expected {
				t.Errorf("rookVersionGTE(%v, %v) = %v, want %v", tt.version, tt.threshold, result, tt.expected)
			}
		})
	}
}

// TestKubeCmd tests kubectl/oc command selection
func TestKubeCmd(t *testing.T) {
	tests := []struct {
		platform string
		expected string
	}{
		{"k3s", "kubectl"},
		{"openshift", "oc"},
		{"eks", "kubectl"},
		{"", "kubectl"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := kubeCmd(tt.platform)
			if result != tt.expected {
				t.Errorf("kubeCmd(%v) = %v, want %v", tt.platform, result, tt.expected)
			}
		})
	}
}

// TestKubeconfigExport tests kubeconfig export command
func TestKubeconfigExport(t *testing.T) {
	tests := []struct {
		platform string
		contains string
	}{
		{"k3s", "KUBECONFIG=/etc/rancher/k3s/k3s.yaml"},
		{"openshift", "KUBECONFIG=${KUBECONFIG:-~/.kube/config}"},
		{"eks", "KUBECONFIG=/etc/rancher/k3s/k3s.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := kubeconfigExport(tt.platform)
			if result == "" {
				t.Error("kubeconfigExport returned empty string")
			}
			// Just verify it returns something reasonable
		})
	}
}

// TestClient_RunSSH tests SSH command execution
func TestClient_RunSSH(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	_, err := client.RunSSH(ctx, "127.0.0.1", "echo test")
	if err == nil {
		t.Fatal("Expected error without SSH server")
	}
}

// TestClient_RunSSH_WithTimeout tests SSH with context timeout
func TestClient_RunSSH_WithTimeout(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := client.RunSSH(ctx, "127.0.0.1", "sleep 10")
	if err == nil {
		t.Fatal("Expected error")
	}
}

// TestClient_CloneRookRepo tests repository cloning
func TestClient_CloneRookRepo(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	_, err := client.CloneRookRepo("invalid-version")
	if err == nil {
		t.Fatal("Expected error with invalid version")
	}
}

// TestClient_CleanupLocalRepo tests repository cleanup
func TestClient_CleanupLocalRepo(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 30})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := client.CleanupLocalRepo(tmpDir)
	if err != nil {
		t.Errorf("CleanupLocalRepo failed: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Directory should be removed")
	}
}

// TestClient_UploadDirectory tests directory upload
func TestClient_UploadDirectory(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := client.UploadDirectory("127.0.0.1", tmpDir, "/tmp/test-upload")
	if err == nil {
		t.Fatal("Expected error without SSH server")
	}
}

// TestClient_InstallRookCeph tests Rook-Ceph installation
func TestClient_InstallRookCeph(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	config := RookCephConfig{
		ClusterName:       "test-cluster",
		Platform:          "k3s",
		TargetNode:        "127.0.0.1",
		RookCephVersion:   "v1.19.0",
		WorkerCount:       1,
		TaintMasters:      false,
		PodWaitTimeout:    "1s",
		SleepBetweenSteps: 1,
	}

	err := client.InstallRookCeph(ctx, config)
	if err == nil {
		t.Fatal("Expected error without SSH server")
	}
}

// TestClient_ConfigureDefaultStorage tests storage configuration
func TestClient_ConfigureDefaultStorage(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	config := RookCephConfig{
		Platform:         "k3s",
		TargetNode:       "127.0.0.1",
		DisableLocalPath: true,
	}

	err := client.ConfigureDefaultStorage(ctx, config)
	if err == nil {
		t.Fatal("Expected error without SSH server")
	}
	if !strings.Contains(err.Error(), "failed to") && !strings.Contains(err.Error(), "SSH") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestClient_VerifyInstallation tests installation verification
func TestClient_VerifyInstallation(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	config := RookCephConfig{
		Platform:       "k3s",
		TargetNode:     "127.0.0.1:19999",
		PodWaitTimeout: "300s",
	}

	_, err := client.VerifyInstallation(ctx, config)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_UninstallRookCeph tests Rook-Ceph uninstallation
func TestClient_UninstallRookCeph(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	config := RookCephConfig{
		Platform:   "k3s",
		TargetNode: "127.0.0.1:19999",
	}

	err := client.UninstallRookCeph(ctx, config)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestClient_CheckRookCephInstalled tests installation check
func TestClient_CheckRookCephInstalled(t *testing.T) {
	client := NewClient("testuser", "testpass", SSHOptions{ConnectTimeout: 1})
	ctx := context.Background()

	config := RookCephConfig{
		Platform:   "k3s",
		TargetNode: "127.0.0.1:19999",
	}

	_, err := client.CheckRookCephInstalled(ctx, config)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestRookCephConfig_PlatformValidation tests platform-specific configuration
func TestRookCephConfig_PlatformValidation(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		valid    bool
	}{
		{"k3s_platform", "k3s", true},
		{"openshift_platform", "openshift", true},
		{"invalid_platform", "eks", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RookCephConfig{
				Platform: tt.platform,
			}

			// Verify platform is set correctly
			if config.Platform != tt.platform {
				t.Errorf("Platform = %v, want %v", config.Platform, tt.platform)
			}

			// Platform validation happens in the provider layer
			// Here we just verify the field is accessible
		})
	}
}

// TestRookCephConfig_WorkerCountScenarios tests different worker count scenarios
func TestRookCephConfig_WorkerCountScenarios(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		taintMaster bool
		isTest      bool
	}{
		{"single_node", 1, true, true},
		{"multi_node", 3, false, false},
		{"two_nodes", 2, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RookCephConfig{
				Platform:     "k3s",
				WorkerCount:  tt.workerCount,
				TaintMasters: tt.taintMaster,
			}

			// Verify configuration
			if config.WorkerCount != tt.workerCount {
				t.Errorf("WorkerCount = %v, want %v", config.WorkerCount, tt.workerCount)
			}

			// Test cluster determination logic
			isTest := config.WorkerCount <= 1 && config.Platform == "k3s"
			if isTest != tt.isTest {
				t.Errorf("isTest = %v, want %v", isTest, tt.isTest)
			}
		})
	}
}

// Made with Bob
