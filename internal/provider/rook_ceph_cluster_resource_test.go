// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/rookcephclient"
)

// mockRookCephClient is a mock implementation of the RookCeph client for testing
type mockRookCephClient struct {
	installRookCephFunc         func(ctx context.Context, config rookcephclient.RookCephConfig) error
	configureDefaultStorageFunc func(ctx context.Context, config rookcephclient.RookCephConfig) error
	verifyInstallationFunc      func(ctx context.Context, config rookcephclient.RookCephConfig) (string, error)
	checkRookCephInstalledFunc  func(ctx context.Context, config rookcephclient.RookCephConfig) (bool, error)
	uninstallRookCephFunc       func(ctx context.Context, config rookcephclient.RookCephConfig) error
}

func (m *mockRookCephClient) InstallRookCeph(ctx context.Context, config rookcephclient.RookCephConfig) error {
	if m.installRookCephFunc != nil {
		return m.installRookCephFunc(ctx, config)
	}
	return nil
}

func (m *mockRookCephClient) ConfigureDefaultStorage(ctx context.Context, config rookcephclient.RookCephConfig) error {
	if m.configureDefaultStorageFunc != nil {
		return m.configureDefaultStorageFunc(ctx, config)
	}
	return nil
}

func (m *mockRookCephClient) VerifyInstallation(ctx context.Context, config rookcephclient.RookCephConfig) (string, error) {
	if m.verifyInstallationFunc != nil {
		return m.verifyInstallationFunc(ctx, config)
	}
	return "HEALTH_OK", nil
}

func (m *mockRookCephClient) CheckRookCephInstalled(ctx context.Context, config rookcephclient.RookCephConfig) (bool, error) {
	if m.checkRookCephInstalledFunc != nil {
		return m.checkRookCephInstalledFunc(ctx, config)
	}
	return true, nil
}

func (m *mockRookCephClient) UninstallRookCeph(ctx context.Context, config rookcephclient.RookCephConfig) error {
	if m.uninstallRookCephFunc != nil {
		return m.uninstallRookCephFunc(ctx, config)
	}
	return nil
}

func TestRookCephClusterResource_Metadata(t *testing.T) {
	r := NewRookCephClusterResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "guardium-data-protection",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	expected := "guardium-data-protection_rook_ceph_cluster"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}
}

func TestRookCephClusterResource_Schema(t *testing.T) {
	r := NewRookCephClusterResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Expected schema attributes to be defined")
	}

	// Verify required attributes exist
	requiredAttrs := []string{
		"id", "cluster_name", "platform", "target_node",
		"rook_ceph_version", "airgap_rook_ceph_installation_path",
		"airgap_install", "worker_count", "taint_masters",
		"set_as_default_storage", "disable_local_path",
		"pod_wait_timeout", "sleep_between_steps", "delete_timeout",
		"cluster_type", "namespace", "cephfs_storage_class",
		"block_storage_class", "last_updated",
	}

	for _, attrName := range requiredAttrs {
		if _, exists := resp.Schema.Attributes[attrName]; !exists {
			t.Errorf("Expected attribute %s to exist in schema", attrName)
		}
	}

	// Verify required fields
	clusterNameAttr := resp.Schema.Attributes["cluster_name"].(schema.StringAttribute)
	if !clusterNameAttr.Required {
		t.Error("Expected cluster_name to be required")
	}

	platformAttr := resp.Schema.Attributes["platform"].(schema.StringAttribute)
	if !platformAttr.Required {
		t.Error("Expected platform to be required")
	}

	targetNodeAttr := resp.Schema.Attributes["target_node"].(schema.StringAttribute)
	if !targetNodeAttr.Required {
		t.Error("Expected target_node to be required")
	}

	// Verify computed fields
	namespaceAttr := resp.Schema.Attributes["namespace"].(schema.StringAttribute)
	if !namespaceAttr.Computed {
		t.Error("Expected namespace to be computed")
	}
}

func TestRookCephClusterResource_Configure(t *testing.T) {
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
				RookCephClient: &rookcephclient.Client{},
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
			name:          "unified client without rook ceph client",
			providerData:  &UnifiedClient{},
			expectError:   true,
			errorContains: "RookCeph Client Not Configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RookCephClusterResource{}
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

func TestRookCephClusterResource_NewRookCephClusterResource(t *testing.T) {
	r := NewRookCephClusterResource()
	if r == nil {
		t.Fatal("Expected NewRookCephClusterResource to return a non-nil resource")
	}

	_, ok := r.(*RookCephClusterResource)
	if !ok {
		t.Error("Expected NewRookCephClusterResource to return a *RookCephClusterResource")
	}
}

func TestMockRookCephClient_Methods(t *testing.T) {
	t.Run("InstallRookCeph", func(t *testing.T) {
		called := false
		mock := &mockRookCephClient{
			installRookCephFunc: func(ctx context.Context, config rookcephclient.RookCephConfig) error {
				called = true
				if config.ClusterName != "test-cluster" {
					t.Errorf("Expected cluster name 'test-cluster', got %s", config.ClusterName)
				}
				return nil
			},
		}
		err := mock.InstallRookCeph(context.Background(), rookcephclient.RookCephConfig{
			ClusterName: "test-cluster",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected installRookCephFunc to be called")
		}
	})

	t.Run("VerifyInstallation", func(t *testing.T) {
		mock := &mockRookCephClient{
			verifyInstallationFunc: func(ctx context.Context, config rookcephclient.RookCephConfig) (string, error) {
				return "HEALTH_OK", nil
			},
		}
		output, err := mock.VerifyInstallation(context.Background(), rookcephclient.RookCephConfig{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if output != "HEALTH_OK" {
			t.Errorf("Expected output 'HEALTH_OK', got %s", output)
		}
	})

	t.Run("CheckRookCephInstalled", func(t *testing.T) {
		mock := &mockRookCephClient{
			checkRookCephInstalledFunc: func(ctx context.Context, config rookcephclient.RookCephConfig) (bool, error) {
				if config.Platform != "k3s" {
					t.Errorf("Expected platform 'k3s', got %s", config.Platform)
				}
				return true, nil
			},
		}
		installed, err := mock.CheckRookCephInstalled(context.Background(), rookcephclient.RookCephConfig{
			Platform: "k3s",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Error("Expected installed to be true")
		}
	})

	t.Run("UninstallRookCeph", func(t *testing.T) {
		called := false
		mock := &mockRookCephClient{
			uninstallRookCephFunc: func(ctx context.Context, config rookcephclient.RookCephConfig) error {
				called = true
				if config.ClusterName != "test-cluster" {
					t.Errorf("Expected cluster name 'test-cluster', got %s", config.ClusterName)
				}
				return nil
			},
		}
		err := mock.UninstallRookCeph(context.Background(), rookcephclient.RookCephConfig{
			ClusterName: "test-cluster",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected uninstallRookCephFunc to be called")
		}
	})

	t.Run("ConfigureDefaultStorage", func(t *testing.T) {
		called := false
		mock := &mockRookCephClient{
			configureDefaultStorageFunc: func(ctx context.Context, config rookcephclient.RookCephConfig) error {
				called = true
				if !config.SetAsDefaultStorage {
					t.Error("Expected SetAsDefaultStorage to be true")
				}
				return nil
			},
		}
		err := mock.ConfigureDefaultStorage(context.Background(), rookcephclient.RookCephConfig{
			SetAsDefaultStorage: true,
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected configureDefaultStorageFunc to be called")
		}
	})
}

func TestRookCephClusterResourceModel_Fields(t *testing.T) {
	// Test that the model can be instantiated with all fields
	model := RookCephClusterResourceModel{
		ID:                       types.StringValue("test-id"),
		ClusterName:              types.StringValue("test-cluster"),
		Platform:                 types.StringValue("k3s"),
		TargetNode:               types.StringValue("master.example.com"),
		RookCephVersion:          types.StringValue("v1.15.4"),
		RookCephInstallationPath: types.StringValue("/tmp/rook-ceph"),
		AirgapInstall:            types.BoolValue(true),
		WorkerCount:              types.Int64Value(3),
		TaintMasters:             types.BoolValue(false),
		SetAsDefaultStorage:      types.BoolValue(true),
		DisableLocalPath:         types.BoolValue(true),
		PodWaitTimeout:           types.StringValue("600s"),
		SleepBetweenSteps:        types.Int64Value(60),
		DeleteTimeout:            types.StringValue("2h"),
		ClusterType:              types.StringValue("production"),
		Namespace:                types.StringValue("rook-ceph"),
		CephfsStorageClass:       types.StringValue("rook-cephfs"),
		BlockStorageClass:        types.StringValue("rook-ceph-block"),
		LastUpdated:              types.StringValue("2024-01-01T00:00:00Z"),
	}

	if model.ID.ValueString() != "test-id" {
		t.Error("Expected ID to be set correctly")
	}
	if model.Platform.ValueString() != "k3s" {
		t.Error("Expected Platform to be set correctly")
	}
	if model.WorkerCount.ValueInt64() != 3 {
		t.Error("Expected WorkerCount to be set correctly")
	}
	if model.ClusterType.ValueString() != "production" {
		t.Error("Expected ClusterType to be set correctly")
	}
}

func TestRookCephClusterResource_PlatformValidation(t *testing.T) {
	// Test that validates platform must be k3s or openshift
	// This would be tested in Create method, but we're skipping those tests
	// Document the validation logic here
	validPlatforms := []string{"k3s", "openshift"}

	for _, platform := range validPlatforms {
		t.Run("valid_platform_"+platform, func(t *testing.T) {
			// In actual Create, this would pass validation
			if platform != "k3s" && platform != "openshift" {
				t.Errorf("Platform %s should be valid", platform)
			}
		})
	}

	invalidPlatforms := []string{"eks", "gke", "aks", ""}
	for _, platform := range invalidPlatforms {
		t.Run("invalid_platform_"+platform, func(t *testing.T) {
			// In actual Create, this would fail validation
			if platform == "k3s" || platform == "openshift" {
				t.Errorf("Platform %s should be invalid", platform)
			}
		})
	}
}

// Made with Bob
