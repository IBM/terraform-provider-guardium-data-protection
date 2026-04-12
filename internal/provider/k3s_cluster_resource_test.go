// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k3sclient"
)

// mockK3sClient is a mock implementation of the K3S client for testing
type mockK3sClient struct {
	installSingleNodeFunc       func(config k3sclient.K3SInstallConfig) error
	installPrimaryMasterFunc    func(config k3sclient.K3SInstallConfig) error
	installAdditionalMasterFunc func(config k3sclient.K3SInstallConfig, masterIP string, node string, nodeIndex int) error
	installWorkerFunc           func(config k3sclient.K3SInstallConfig, masterIP string, worker string, nodeIndex int) error
	waitForNodesFunc            func(config k3sclient.K3SInstallConfig) error
	verifyClusterFunc           func(config k3sclient.K3SInstallConfig) (string, error)
	uninstallK3SFunc            func(host string, isServer bool) error
	checkK3SInstalledFunc       func(host string) (bool, error)
}

func (m *mockK3sClient) InstallK3SSingleNode(config k3sclient.K3SInstallConfig) error {
	if m.installSingleNodeFunc != nil {
		return m.installSingleNodeFunc(config)
	}
	return nil
}

func (m *mockK3sClient) InstallK3SPrimaryMaster(config k3sclient.K3SInstallConfig) error {
	if m.installPrimaryMasterFunc != nil {
		return m.installPrimaryMasterFunc(config)
	}
	return nil
}

func (m *mockK3sClient) InstallK3SAdditionalMaster(config k3sclient.K3SInstallConfig, masterIP string, node string, nodeIndex int) error {
	if m.installAdditionalMasterFunc != nil {
		return m.installAdditionalMasterFunc(config, masterIP, node, nodeIndex)
	}
	return nil
}

func (m *mockK3sClient) InstallK3SWorker(config k3sclient.K3SInstallConfig, masterIP string, worker string, nodeIndex int) error {
	if m.installWorkerFunc != nil {
		return m.installWorkerFunc(config, masterIP, worker, nodeIndex)
	}
	return nil
}

func (m *mockK3sClient) WaitForNodes(config k3sclient.K3SInstallConfig) error {
	if m.waitForNodesFunc != nil {
		return m.waitForNodesFunc(config)
	}
	return nil
}

func (m *mockK3sClient) VerifyCluster(config k3sclient.K3SInstallConfig) (string, error) {
	if m.verifyClusterFunc != nil {
		return m.verifyClusterFunc(config)
	}
	return "cluster verified", nil
}

func (m *mockK3sClient) UninstallK3S(host string, isServer bool) error {
	if m.uninstallK3SFunc != nil {
		return m.uninstallK3SFunc(host, isServer)
	}
	return nil
}

func (m *mockK3sClient) CheckK3SInstalled(host string) (bool, error) {
	if m.checkK3SInstalledFunc != nil {
		return m.checkK3SInstalledFunc(host)
	}
	return true, nil
}

func TestK3SClusterResource_Metadata(t *testing.T) {
	r := NewK3SClusterResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "guardium-data-protection",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	expected := "guardium-data-protection_k3s_cluster"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}
}

func TestK3SClusterResource_Schema(t *testing.T) {
	r := NewK3SClusterResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Expected schema attributes to be defined")
	}

	// Verify required attributes exist
	requiredAttrs := []string{
		"id", "cluster_name", "master_nodes", "worker_nodes",
		"k3s_version", "k3s_token", "disable_traefik", "taint_masters",
		"node_wait_timeout", "primary_master", "cluster_type",
		"kubeconfig_path", "last_updated", "airgap_install",
		"airgap_installation_path", "delete_timeout",
	}

	for _, attrName := range requiredAttrs {
		if _, exists := resp.Schema.Attributes[attrName]; !exists {
			t.Errorf("Expected attribute %s to exist in schema", attrName)
		}
	}

	// Verify cluster_name is required
	clusterNameAttr := resp.Schema.Attributes["cluster_name"].(schema.StringAttribute)
	if !clusterNameAttr.Required {
		t.Error("Expected cluster_name to be required")
	}

	// Verify master_nodes is required
	masterNodesAttr := resp.Schema.Attributes["master_nodes"].(schema.ListAttribute)
	if !masterNodesAttr.Required {
		t.Error("Expected master_nodes to be required")
	}

	// Verify k3s_token is sensitive
	k3sTokenAttr := resp.Schema.Attributes["k3s_token"].(schema.StringAttribute)
	if !k3sTokenAttr.Sensitive {
		t.Error("Expected k3s_token to be sensitive")
	}
}

func TestK3SClusterResource_Configure(t *testing.T) {
	tests := []struct {
		name          string
		providerData  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
		{
			name: "valid unified client",
			providerData: &UnifiedClient{
				K3sClient: &k3sclient.Client{},
			},
			expectError: false,
		},
		{
			name:          "invalid provider data type",
			providerData:  "invalid",
			expectError:   true,
			errorContains: "Unexpected Resource Configure Type",
		},
		{
			name:          "unified client without k3s client",
			providerData:  &UnifiedClient{},
			expectError:   true,
			errorContains: "K3s Client Not Configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &K3SClusterResource{}
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), req, resp)

			if tt.expectError {
				if !resp.Diagnostics.HasError() {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" {
					found := false
					for _, diag := range resp.Diagnostics.Errors() {
						if strings.Contains(diag.Summary(), tt.errorContains) || strings.Contains(diag.Detail(), tt.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing %q, but didn't find it", tt.errorContains)
					}
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Errorf("Unexpected error: %v", resp.Diagnostics.Errors())
				}
			}
		})
	}
}

// TestK3SClusterResource_NewK3SClusterResource tests the constructor
func TestK3SClusterResource_NewK3SClusterResource(t *testing.T) {
	r := NewK3SClusterResource()
	if r == nil {
		t.Fatal("Expected NewK3SClusterResource to return a non-nil resource")
	}

	_, ok := r.(*K3SClusterResource)
	if !ok {
		t.Error("Expected NewK3SClusterResource to return a *K3SClusterResource")
	}
}

// TestMockK3sClient_Methods tests the mock client methods
func TestMockK3sClient_Methods(t *testing.T) {
	t.Run("InstallK3SSingleNode", func(t *testing.T) {
		called := false
		mock := &mockK3sClient{
			installSingleNodeFunc: func(config k3sclient.K3SInstallConfig) error {
				called = true
				return nil
			},
		}
		err := mock.InstallK3SSingleNode(k3sclient.K3SInstallConfig{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected installSingleNodeFunc to be called")
		}
	})

	t.Run("CheckK3SInstalled", func(t *testing.T) {
		mock := &mockK3sClient{
			checkK3SInstalledFunc: func(host string) (bool, error) {
				if host != "test-host" {
					t.Errorf("Expected host 'test-host', got %s", host)
				}
				return true, nil
			},
		}
		installed, err := mock.CheckK3SInstalled("test-host")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Error("Expected installed to be true")
		}
	})

	t.Run("UninstallK3S", func(t *testing.T) {
		mock := &mockK3sClient{
			uninstallK3SFunc: func(host string, isServer bool) error {
				if host != "test-host" {
					t.Errorf("Expected host 'test-host', got %s", host)
				}
				if !isServer {
					t.Error("Expected isServer to be true")
				}
				return nil
			},
		}
		err := mock.UninstallK3S("test-host", true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("VerifyCluster", func(t *testing.T) {
		mock := &mockK3sClient{
			verifyClusterFunc: func(config k3sclient.K3SInstallConfig) (string, error) {
				return "test output", nil
			},
		}
		output, err := mock.VerifyCluster(k3sclient.K3SInstallConfig{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if output != "test output" {
			t.Errorf("Expected output 'test output', got %s", output)
		}
	})
}

// Made with Bob
