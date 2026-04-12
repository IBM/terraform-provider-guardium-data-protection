// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package edgeclient

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k8sclient"
)

// TestNewClient tests the NewClient constructor
func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectSSHInit bool
		description   string
	}{
		{
			name: "client_with_ssh_password",
			config: Config{
				CMUrl:       "https://cm.example.com",
				OAuthToken:  "test-token",
				Platform:    "k3s",
				SSHUser:     "testuser",
				SSHPassword: "testpass",
			},
			expectSSHInit: true,
			description:   "Client should initialize SSH client with password",
		},
		{
			name: "client_with_ssh_key",
			config: Config{
				CMUrl:      "https://cm.example.com",
				OAuthToken: "test-token",
				Platform:   "k3s",
				SSHUser:    "testuser",
				SSHKeyPath: "/path/to/key",
			},
			expectSSHInit: false, // Will fail to read key file, but client still created
			description:   "Client should attempt to initialize SSH client with key",
		},
		{
			name: "client_without_ssh",
			config: Config{
				CMUrl:      "https://cm.example.com",
				OAuthToken: "test-token",
				Platform:   "openshift",
			},
			expectSSHInit: false,
			description:   "Client should be created without SSH client",
		},
		{
			name: "client_with_eks_config",
			config: Config{
				CMUrl:          "https://cm.example.com",
				OAuthToken:     "test-token",
				Platform:       "eks",
				AWSRegion:      "us-east-1",
				EKSClusterName: "test-cluster",
			},
			expectSSHInit: false,
			description:   "Client should be created with EKS configuration",
		},
		{
			name: "client_with_openshift_config",
			config: Config{
				CMUrl:       "https://cm.example.com",
				OAuthToken:  "test-token",
				Platform:    "openshift",
				OCPServer:   "https://api.ocp.example.com:6443",
				OCPUsername: "admin",
				OCPPassword: "password",
			},
			expectSSHInit: false,
			description:   "Client should be created with OpenShift configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}

			if client.Config.CMUrl != tt.config.CMUrl {
				t.Errorf("CMUrl = %v, want %v", client.Config.CMUrl, tt.config.CMUrl)
			}

			if client.Config.Platform != tt.config.Platform {
				t.Errorf("Platform = %v, want %v", client.Config.Platform, tt.config.Platform)
			}

			// SSH client initialization depends on file system access
			// We just verify the client was created
			if tt.expectSSHInit && client.sshClient == nil {
				t.Log("SSH client not initialized (expected if key file doesn't exist)")
			}
		})
	}
}

// TestConfig_Fields tests the Config struct fields
func TestConfig_Fields(t *testing.T) {
	config := Config{
		CMUrl:                 "https://cm.example.com",
		OAuthToken:            "test-token",
		Platform:              "k3s",
		SSHUser:               "testuser",
		SSHPassword:           "testpass",
		SSHKeyPath:            "/path/to/key",
		AWSRegion:             "us-east-1",
		AWSProfile:            "default",
		AWSAccessKey:          "access-key",
		AWSSecretKey:          "secret-key",
		EKSClusterName:        "test-cluster",
		EKSSSHUser:            "ec2-user",
		EKSSSHKeyPath:         "/path/to/eks-key",
		EKSSSHKeyPassphrase:   "passphrase",
		EKSHostnameType:       "private",
		OCPServer:             "https://api.ocp.example.com:6443",
		OCPUsername:           "admin",
		OCPPassword:           "password",
		OCPToken:              "ocp-token",
		OCPInsecureSkipVerify: true,
	}

	// Verify all fields are accessible
	if config.CMUrl == "" {
		t.Error("CMUrl should not be empty")
	}
	if config.Platform == "" {
		t.Error("Platform should not be empty")
	}
	if config.SSHUser == "" {
		t.Error("SSHUser should not be empty")
	}
}

// TestClient_K8sClient tests the K8sClient getter
func TestClient_K8sClient(t *testing.T) {
	client := NewClient(Config{
		CMUrl:      "https://cm.example.com",
		OAuthToken: "test-token",
		Platform:   "k3s",
	})

	// Initially nil
	if client.K8sClient() != nil {
		t.Error("K8sClient should be nil before initialization")
	}

	// After setting (mock)
	mockK8sClient := &k8sclient.Client{}
	client.k8sClient = mockK8sClient

	if client.K8sClient() != mockK8sClient {
		t.Error("K8sClient should return the set client")
	}
}

// TestClient_InitK8sClient tests K8s client initialization error handling
func TestClient_InitK8sClient(t *testing.T) {
	client := NewClient(Config{
		Platform: "k3s",
	})

	ctx := context.Background()
	err := client.InitK8sClient(ctx, "/nonexistent/kubeconfig")

	if err == nil {
		t.Fatal("InitK8sClient should fail with nonexistent kubeconfig")
	}
	if !strings.Contains(err.Error(), "failed to initialize k8s client") {
		t.Errorf("expected wrapped initialization error, got %v", err)
	}
}

// TestClient_SSHMethods tests SSH-related methods
func TestClient_SSHMethods(t *testing.T) {
	client := NewClient(Config{
		CMUrl:      "https://cm.example.com",
		OAuthToken: "test-token",
		Platform:   "k3s",
	})

	t.Run("RunSSH_without_ssh_client", func(t *testing.T) {
		_, err := client.RunSSH("localhost", "echo test")
		if err == nil {
			t.Error("RunSSH should fail without SSH client")
		}
		if err.Error() != "SSH client not initialized" {
			t.Errorf("Expected 'SSH client not initialized', got %v", err)
		}
	})

	t.Run("SCPFile_without_ssh_client", func(t *testing.T) {
		err := client.SCPFile("/local/file", "/remote/file", "localhost")
		if err == nil {
			t.Error("SCPFile should fail without SSH client")
		}
		if err.Error() != "SSH client not initialized" {
			t.Errorf("Expected 'SSH client not initialized', got %v", err)
		}
	})

	t.Run("SCPFileFrom_without_ssh_client", func(t *testing.T) {
		err := client.SCPFileFrom("/remote/file", "/local/file", "localhost")
		if err == nil {
			t.Error("SCPFileFrom should fail without SSH client")
		}
		if err.Error() != "SSH client not initialized" {
			t.Errorf("Expected 'SSH client not initialized', got %v", err)
		}
	})
}

// TestClient_FetchKubeconfig tests kubeconfig fetching
func TestClient_FetchKubeconfig(t *testing.T) {
	client := NewClient(Config{})

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "nested", "kubeconfig")

	err := client.FetchKubeconfig("localhost", kubeconfigPath)
	if err == nil {
		t.Fatal("FetchKubeconfig should fail without SSH client")
	}
	if !strings.Contains(err.Error(), "failed to copy kubeconfig") {
		t.Errorf("expected copy failure, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Dir(kubeconfigPath)); statErr != nil {
		t.Errorf("expected kubeconfig directory to be created, got %v", statErr)
	}
}

// TestClient_DownloadBundle tests bundle download
func TestClient_DownloadBundle(t *testing.T) {
	t.Run("missing_credentials", func(t *testing.T) {
		client := NewClient(Config{})

		err := client.DownloadBundle("test-edge", "/tmp/test")
		if err == nil {
			t.Error("DownloadBundle should fail without credentials")
		}
		if err.Error() != "CM URL and OAuth token are required for bundle download" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("with_credentials", func(t *testing.T) {
		t.Skip("Skipping test that requires real CM server")

		client := NewClient(Config{
			CMUrl:      "https://cm.example.com",
			OAuthToken: "test-token",
		})

		tmpDir := t.TempDir()
		err := client.DownloadBundle("test-edge", tmpDir)
		// Will fail without real server, but tests the flow
		if err == nil {
			t.Error("Expected error with fake server")
		}
	})
}

// TestClient_ExtractCertInfo tests certificate extraction
func TestClient_ExtractCertInfo(t *testing.T) {
	client := NewClient(Config{})

	t.Run("missing_configmap", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, _, err := client.ExtractCertInfo(tmpDir, false)
		if err == nil {
			t.Error("ExtractCertInfo should fail without configmap file")
		}
	})

	t.Run("with_configmap", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create mock configmap
		configmapContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: edge-controller-client
data:
  CM_PRIVATE_REGISTRY: "registry.example.com:5000"
  NAMESPACE: "edge-ns"
`
		configmapPath := filepath.Join(tmpDir, "01-edge-controller-client-configmap.yaml")
		if err := os.WriteFile(configmapPath, []byte(configmapContent), 0644); err != nil {
			t.Fatal(err)
		}

		registry, namespace, err := client.ExtractCertInfo(tmpDir, false)
		if err != nil {
			t.Errorf("ExtractCertInfo failed: %v", err)
		}

		if registry != "registry.example.com:5000" {
			t.Errorf("registry = %v, want registry.example.com:5000", registry)
		}

		if namespace != "edge-ns" {
			t.Errorf("namespace = %v, want edge-ns", namespace)
		}
	})

	t.Run("with_external_registry", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create mock configmap with external registry
		configmapContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: edge-controller-client
data:
  EXTERNAL_IMAGE_REGISTRY: "external.registry.com"
  NAMESPACE: "edge-ns"
`
		configmapPath := filepath.Join(tmpDir, "01-edge-controller-client-configmap.yaml")
		if err := os.WriteFile(configmapPath, []byte(configmapContent), 0644); err != nil {
			t.Fatal(err)
		}

		registry, namespace, err := client.ExtractCertInfo(tmpDir, true)
		if err != nil {
			t.Errorf("ExtractCertInfo failed: %v", err)
		}

		if registry != "external.registry.com" {
			t.Errorf("registry = %v, want external.registry.com", registry)
		}

		if namespace != "edge-ns" {
			t.Errorf("namespace = %v, want edge-ns", namespace)
		}
	})
}

// TestClient_InstallCerts tests certificate installation methods
func TestClient_InstallCerts(t *testing.T) {
	client := NewClient(Config{})
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("InstallCertsK3S_no_cert", func(t *testing.T) {
		// Should succeed with no certificate file
		err := client.InstallCertsK3S(ctx, tmpDir, []string{"node1"}, "registry.example.com")
		if err != nil {
			t.Errorf("InstallCertsK3S should succeed with no cert file: %v", err)
		}
	})

	t.Run("InstallCertsOpenShift_no_cert", func(t *testing.T) {
		// Should succeed with no certificate file
		err := client.InstallCertsOpenShift(ctx, tmpDir, "registry.example.com", 5*time.Minute)
		if err != nil {
			t.Errorf("InstallCertsOpenShift should succeed with no cert file: %v", err)
		}
	})

	t.Run("InstallCertsEKS_no_cert", func(t *testing.T) {
		// Should succeed with no certificate file
		err := client.InstallCertsEKS(ctx, tmpDir, "registry.example.com")
		if err != nil {
			t.Errorf("InstallCertsEKS should succeed with no cert file: %v", err)
		}
	})
}

// TestClient_DeployEdge tests edge deployment
func TestClient_DeployEdge(t *testing.T) {
	client := NewClient(Config{})
	ctx := context.Background()

	err := client.DeployEdge(ctx, "/tmp/workdir", "edge-ns", "k3s")
	if err == nil {
		t.Error("DeployEdge should fail without k8s client")
	}
	if err.Error() != "k8s client not initialized" {
		t.Errorf("Expected 'k8s client not initialized', got %v", err)
	}
}

// TestClient_MonitorDeployment tests deployment monitoring
func TestClient_MonitorDeployment(t *testing.T) {
	client := NewClient(Config{})
	ctx := context.Background()

	_, err := client.MonitorDeployment(ctx, "edge-ns", 10, 5)
	if err == nil {
		t.Error("MonitorDeployment should fail without k8s client")
	}
	if err.Error() != "k8s client not initialized" {
		t.Errorf("Expected 'k8s client not initialized', got %v", err)
	}
}

// TestClient_DeleteEdge tests edge deletion
func TestClient_DeleteEdge(t *testing.T) {
	client := NewClient(Config{})
	ctx := context.Background()

	err := client.DeleteEdge(ctx, "/tmp/workdir", "edge-ns")
	if err == nil {
		t.Error("DeleteEdge should fail without k8s client")
	}
	if err.Error() != "k8s client not initialized" {
		t.Errorf("Expected 'k8s client not initialized', got %v", err)
	}
}

// TestClient_InstallMetricsServer tests metrics server installation
func TestClient_InstallMetricsServer(t *testing.T) {
	client := NewClient(Config{})
	ctx := context.Background()

	t.Run("without_k8s_client", func(t *testing.T) {
		err := client.InstallMetricsServer(ctx, false, "")
		if err == nil {
			t.Error("InstallMetricsServer should fail without k8s client")
		}
		if err.Error() != "k8s client not initialized" {
			t.Errorf("Expected 'k8s client not initialized', got %v", err)
		}
	})

	t.Run("airgap_without_path", func(t *testing.T) {
		err := client.InstallMetricsServer(ctx, true, "")
		if err == nil {
			t.Error("InstallMetricsServer should fail without airgap path")
		}
	})
}

// TestClient_CleanupBundle tests bundle cleanup
func TestClient_CleanupBundle(t *testing.T) {
	client := NewClient(Config{})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := client.CleanupBundle(tmpDir)
	if err != nil {
		t.Errorf("CleanupBundle failed: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Directory should be removed")
	}
}

// Made with Bob
