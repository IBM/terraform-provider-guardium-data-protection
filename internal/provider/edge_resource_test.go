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
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/edgeclient"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k8sclient"
)

// mockEdgeClient is a mock implementation of the Edge client for testing
type mockEdgeClient struct {
	downloadBundleFunc        func(edgeName, workDir string) error
	fetchKubeconfigFunc       func(masterNode, kubeconfigPath string) error
	initK8sClientFunc         func(ctx context.Context, kubeconfigPath string) error
	extractCertInfoFunc       func(workDir string, externalRegistry bool) (string, string, error)
	installCertsK3SFunc       func(ctx context.Context, workDir string, nodes []string, registry string) error
	installCertsOpenShiftFunc func(ctx context.Context, workDir, registry string, timeout interface{}) error
	installCertsEKSFunc       func(ctx context.Context, workDir, registry string) error
	deployEdgeFunc            func(ctx context.Context, workDir, namespace, platform string) error
	monitorDeploymentFunc     func(ctx context.Context, namespace string, maxAttempts, sleepInterval int) (string, error)
	deleteEdgeFunc            func(ctx context.Context, workDir, namespace string) error
	cleanupBundleFunc         func(workDir string) error
	k8sClientFunc             func() *k8sclient.Client
	installMetricsServerFunc  func(ctx context.Context, airgap bool, airgapPath string) error
}

func (m *mockEdgeClient) DownloadBundle(edgeName, workDir string) error {
	if m.downloadBundleFunc != nil {
		return m.downloadBundleFunc(edgeName, workDir)
	}
	return nil
}

func (m *mockEdgeClient) FetchKubeconfig(masterNode, kubeconfigPath string) error {
	if m.fetchKubeconfigFunc != nil {
		return m.fetchKubeconfigFunc(masterNode, kubeconfigPath)
	}
	return nil
}

func (m *mockEdgeClient) InitK8sClient(ctx context.Context, kubeconfigPath string) error {
	if m.initK8sClientFunc != nil {
		return m.initK8sClientFunc(ctx, kubeconfigPath)
	}
	return nil
}

func (m *mockEdgeClient) ExtractCertInfo(workDir string, externalRegistry bool) (string, string, error) {
	if m.extractCertInfoFunc != nil {
		return m.extractCertInfoFunc(workDir, externalRegistry)
	}
	return "registry.example.com", "edge-namespace", nil
}

func (m *mockEdgeClient) InstallCertsK3S(ctx context.Context, workDir string, nodes []string, registry string) error {
	if m.installCertsK3SFunc != nil {
		return m.installCertsK3SFunc(ctx, workDir, nodes, registry)
	}
	return nil
}

func (m *mockEdgeClient) InstallCertsOpenShift(ctx context.Context, workDir, registry string, timeout interface{}) error {
	if m.installCertsOpenShiftFunc != nil {
		return m.installCertsOpenShiftFunc(ctx, workDir, registry, timeout)
	}
	return nil
}

func (m *mockEdgeClient) InstallCertsEKS(ctx context.Context, workDir, registry string) error {
	if m.installCertsEKSFunc != nil {
		return m.installCertsEKSFunc(ctx, workDir, registry)
	}
	return nil
}

func (m *mockEdgeClient) DeployEdge(ctx context.Context, workDir, namespace, platform string) error {
	if m.deployEdgeFunc != nil {
		return m.deployEdgeFunc(ctx, workDir, namespace, platform)
	}
	return nil
}

func (m *mockEdgeClient) MonitorDeployment(ctx context.Context, namespace string, maxAttempts, sleepInterval int) (string, error) {
	if m.monitorDeploymentFunc != nil {
		return m.monitorDeploymentFunc(ctx, namespace, maxAttempts, sleepInterval)
	}
	return "Running", nil
}

func (m *mockEdgeClient) DeleteEdge(ctx context.Context, workDir, namespace string) error {
	if m.deleteEdgeFunc != nil {
		return m.deleteEdgeFunc(ctx, workDir, namespace)
	}
	return nil
}

func (m *mockEdgeClient) CleanupBundle(workDir string) error {
	if m.cleanupBundleFunc != nil {
		return m.cleanupBundleFunc(workDir)
	}
	return nil
}

func (m *mockEdgeClient) K8sClient() *k8sclient.Client {
	if m.k8sClientFunc != nil {
		return m.k8sClientFunc()
	}
	return &k8sclient.Client{}
}

func (m *mockEdgeClient) InstallMetricsServer(ctx context.Context, airgap bool, airgapPath string) error {
	if m.installMetricsServerFunc != nil {
		return m.installMetricsServerFunc(ctx, airgap, airgapPath)
	}
	return nil
}

func TestEdgeDeploymentResource_Metadata(t *testing.T) {
	r := NewEdgeDeploymentResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "guardium-data-protection",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	expected := "guardium-data-protection_edge_deploy"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}
}

func TestEdgeDeploymentResource_Schema(t *testing.T) {
	r := NewEdgeDeploymentResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Expected schema attributes to be defined")
	}

	// Verify key attributes exist
	requiredAttrs := []string{
		"id", "edge_name", "edge_bundle_directory", "platform",
		"k3s_master_node", "k3s_nodes", "eks_cluster_name",
		"monitor_max_attempts", "monitor_sleep_interval", "cleanup_bundle",
		"delete_timeout", "ocp_machineconfig_timeout",
		"external_image_registry", "k8s_metrics_server_install",
		"edge_namespace", "registry_url", "deployment_status",
		"work_dir", "last_updated",
	}

	for _, attrName := range requiredAttrs {
		if _, exists := resp.Schema.Attributes[attrName]; !exists {
			t.Errorf("Expected attribute %s to exist in schema", attrName)
		}
	}

	// Verify sensitive attributes
	ocpPasswordAttr, ok := resp.Schema.Attributes["ocp_password"].(schema.StringAttribute)
	if !ok {
		t.Fatal("ocp_password attribute is not a StringAttribute")
	}
	if !ocpPasswordAttr.Sensitive {
		t.Error("Expected ocp_password to be sensitive")
	}

	ocpTokenAttr, ok := resp.Schema.Attributes["ocp_token"].(schema.StringAttribute)
	if !ok {
		t.Fatal("ocp_token attribute is not a StringAttribute")
	}
	if !ocpTokenAttr.Sensitive {
		t.Error("Expected ocp_token to be sensitive")
	}
}

func TestEdgeDeploymentResource_Configure(t *testing.T) {
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
				EdgeClient: &edgeclient.Client{},
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
			name:          "unified client without edge client",
			providerData:  &UnifiedClient{},
			expectError:   true,
			errorContains: "Edge Client Not Configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EdgeDeploymentResource{}
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

func TestEdgeDeploymentResource_NewEdgeDeploymentResource(t *testing.T) {
	r := NewEdgeDeploymentResource()
	if r == nil {
		t.Fatal("Expected NewEdgeDeploymentResource to return a non-nil resource")
	}

	_, ok := r.(*EdgeDeploymentResource)
	if !ok {
		t.Error("Expected NewEdgeDeploymentResource to return a *EdgeDeploymentResource")
	}
}

func TestMockEdgeClient_Methods(t *testing.T) {
	t.Run("DownloadBundle", func(t *testing.T) {
		called := false
		mock := &mockEdgeClient{
			downloadBundleFunc: func(edgeName, workDir string) error {
				called = true
				if edgeName != "test-edge" {
					t.Errorf("Expected edgeName 'test-edge', got %s", edgeName)
				}
				return nil
			},
		}
		err := mock.DownloadBundle("test-edge", "/tmp/test")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Expected downloadBundleFunc to be called")
		}
	})

	t.Run("ExtractCertInfo", func(t *testing.T) {
		mock := &mockEdgeClient{
			extractCertInfoFunc: func(workDir string, externalRegistry bool) (string, string, error) {
				return "test-registry.com", "test-namespace", nil
			},
		}
		registry, namespace, err := mock.ExtractCertInfo("/tmp/test", false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if registry != "test-registry.com" {
			t.Errorf("Expected registry 'test-registry.com', got %s", registry)
		}
		if namespace != "test-namespace" {
			t.Errorf("Expected namespace 'test-namespace', got %s", namespace)
		}
	})

	t.Run("MonitorDeployment", func(t *testing.T) {
		mock := &mockEdgeClient{
			monitorDeploymentFunc: func(ctx context.Context, namespace string, maxAttempts, sleepInterval int) (string, error) {
				if namespace != "test-namespace" {
					t.Errorf("Expected namespace 'test-namespace', got %s", namespace)
				}
				if maxAttempts != 180 {
					t.Errorf("Expected maxAttempts 180, got %d", maxAttempts)
				}
				return "Running", nil
			},
		}
		status, err := mock.MonitorDeployment(context.Background(), "test-namespace", 180, 10)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if status != "Running" {
			t.Errorf("Expected status 'Running', got %s", status)
		}
	})

	t.Run("K8sClient", func(t *testing.T) {
		expectedClient := &k8sclient.Client{}
		mock := &mockEdgeClient{
			k8sClientFunc: func() *k8sclient.Client {
				return expectedClient
			},
		}
		client := mock.K8sClient()
		if client != expectedClient {
			t.Error("Expected K8sClient to return the expected client")
		}
	})
}

func TestEdgeDeploymentResourceModel_Fields(t *testing.T) {
	// Test that the model can be instantiated with all fields
	model := EdgeDeploymentResourceModel{
		ID:                      types.StringValue("test-id"),
		EdgeName:                types.StringValue("test-edge"),
		BundleDirectory:         types.StringValue("/tmp/bundle"),
		Platform:                types.StringValue("k3s"),
		K3SMasterNode:           types.StringValue("master.example.com"),
		MonitorMaxAttempts:      types.Int64Value(180),
		MonitorSleepInterval:    types.Int64Value(10),
		CleanupBundle:           types.BoolValue(true),
		DeleteTimeout:           types.StringValue("2h"),
		OCPMachineConfigTimeout: types.StringValue("30m"),
		ExternalImageRegistry:   types.BoolValue(false),
		EdgeNamespace:           types.StringValue("edge-ns"),
		RegistryURL:             types.StringValue("registry.example.com"),
		DeploymentStatus:        types.StringValue("Running"),
		WorkDir:                 types.StringValue("/tmp/work"),
		LastUpdated:             types.StringValue("2024-01-01T00:00:00Z"),
	}

	if model.ID.ValueString() != "test-id" {
		t.Error("Expected ID to be set correctly")
	}
	if model.Platform.ValueString() != "k3s" {
		t.Error("Expected Platform to be set correctly")
	}
	if model.MonitorMaxAttempts.ValueInt64() != 180 {
		t.Error("Expected MonitorMaxAttempts to be set correctly")
	}
}

// Made with Bob
