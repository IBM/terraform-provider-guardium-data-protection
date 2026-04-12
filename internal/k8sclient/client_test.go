// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// TestAuthConfig tests the AuthConfig function
func TestAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected AuthConfig
	}{
		{
			name: "k3s_with_kubeconfig",
			cfg: Config{
				Platform:       "k3s",
				KubeconfigPath: "/path/to/kubeconfig",
			},
			expected: AuthConfig{
				Platform:       "k3s",
				KubeconfigPath: "/path/to/kubeconfig",
			},
		},
		{
			name: "eks_with_credentials",
			cfg: Config{
				Platform:       "eks",
				AWSRegion:      "us-east-1",
				AWSProfile:     "default",
				AWSAccessKey:   "AKIAIOSFODNN7EXAMPLE",
				AWSSecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				EKSClusterName: "test-cluster",
			},
			expected: AuthConfig{
				Platform:       "eks",
				AWSRegion:      "us-east-1",
				AWSProfile:     "default",
				AWSAccessKey:   "AKIAIOSFODNN7EXAMPLE",
				AWSSecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				EKSClusterName: "test-cluster",
			},
		},
		{
			name: "openshift_with_token",
			cfg: Config{
				Platform:              "openshift",
				OCPServer:             "https://api.cluster.example.com:6443",
				OCPToken:              "sha256~test-token",
				OCPInsecureSkipVerify: true,
			},
			expected: AuthConfig{
				Platform:              "openshift",
				OCPServer:             "https://api.cluster.example.com:6443",
				OCPToken:              "sha256~test-token",
				OCPInsecureSkipVerify: true,
			},
		},
		{
			name: "openshift_with_username_password",
			cfg: Config{
				Platform:              "openshift",
				OCPServer:             "https://api.cluster.example.com:6443",
				OCPUsername:           "admin",
				OCPPassword:           "password",
				OCPInsecureSkipVerify: false,
			},
			expected: AuthConfig{
				Platform:              "openshift",
				OCPServer:             "https://api.cluster.example.com:6443",
				OCPUsername:           "admin",
				OCPPassword:           "password",
				OCPInsecureSkipVerify: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AuthConfig(tt.cfg)

			if result.Platform != tt.expected.Platform {
				t.Errorf("Platform = %v, want %v", result.Platform, tt.expected.Platform)
			}
			if result.KubeconfigPath != tt.expected.KubeconfigPath {
				t.Errorf("KubeconfigPath = %v, want %v", result.KubeconfigPath, tt.expected.KubeconfigPath)
			}
			if result.AWSRegion != tt.expected.AWSRegion {
				t.Errorf("AWSRegion = %v, want %v", result.AWSRegion, tt.expected.AWSRegion)
			}
			if result.OCPServer != tt.expected.OCPServer {
				t.Errorf("OCPServer = %v, want %v", result.OCPServer, tt.expected.OCPServer)
			}
		})
	}
}

// mockClient wraps a fake clientset for testing
type mockClient struct {
	*Client
	fakeClientset *k8sfake.Clientset
}

// createMockClient creates a mock k8s client for testing
func createMockClient(objects ...runtime.Object) *mockClient {
	fakeClientset := k8sfake.NewSimpleClientset(objects...)
	dynamicClient := fake.NewSimpleDynamicClient(scheme.Scheme, objects...)

	client := &Client{
		config:   &rest.Config{},
		dynamic:  dynamicClient,
		platform: "k3s",
	}

	return &mockClient{
		Client:        client,
		fakeClientset: fakeClientset,
	}
}

// Override methods to use fake clientset
func (m *mockClient) Clientset() kubernetes.Interface {
	return m.fakeClientset
}

func (m *mockClient) NamespaceExists(ctx context.Context, name string) (bool, error) {
	_, err := m.fakeClientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *mockClient) CreateNamespace(ctx context.Context, name string) error {
	exists, err := m.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err = m.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (m *mockClient) DeleteNamespace(ctx context.Context, name string) error {
	err := m.fakeClientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (m *mockClient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return m.fakeClientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (m *mockClient) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return m.fakeClientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (m *mockClient) CreateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	_, err := m.fakeClientset.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (m *mockClient) UpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	_, err := m.fakeClientset.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func (m *mockClient) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	err := m.fakeClientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (m *mockClient) GetConfigMapField(ctx context.Context, namespace, name, field string) (string, error) {
	cm, err := m.GetConfigMap(ctx, namespace, name)
	if err != nil {
		return "", err
	}

	value, ok := cm.Data[field]
	if !ok {
		return "", fmt.Errorf("field %s not found in configmap %s/%s", field, namespace, name)
	}
	return value, nil
}

func (m *mockClient) CreateOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing, err := m.fakeClientset.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return m.CreateConfigMap(ctx, cm)
	}
	if err != nil {
		return err
	}

	cm.ResourceVersion = existing.ResourceVersion
	return m.UpdateConfigMap(ctx, cm)
}

func (m *mockClient) ListNodeNames(ctx context.Context) ([]string, error) {
	nodes, err := m.fakeClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		names[i] = node.Name
	}
	return names, nil
}

func (m *mockClient) ListWorkerNodeNames(ctx context.Context) ([]string, error) {
	nodes, err := m.fakeClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workerNames []string
	for _, node := range nodes.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			continue
		}
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			continue
		}
		workerNames = append(workerNames, node.Name)
	}
	return workerNames, nil
}

func (m *mockClient) GetNodeInternalIP(ctx context.Context, nodeName string) (string, error) {
	node, err := m.fakeClientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("internal IP not found for node %s", nodeName)
}

func (m *mockClient) GetNodeExternalIP(ctx context.Context, nodeName string) (string, error) {
	node, err := m.fakeClientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("external IP not found for node %s", nodeName)
}

func (m *mockClient) WaitForConfigMapExists(ctx context.Context, namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := m.GetConfigMap(ctx, namespace, name)
		if err == nil {
			return nil
		}
		if !errors.IsNotFound(err) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for configmap %s/%s", namespace, name)
}

func (m *mockClient) ListNodeNamesWithLabel(ctx context.Context, labelSelector string) ([]string, error) {
	nodes, err := m.fakeClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes with label %s: %w", labelSelector, err)
	}
	names := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		names[i] = node.Name
	}
	return names, nil
}

func (m *mockClient) CreateNamespaceAndWait(ctx context.Context, name string, timeout time.Duration) error {
	return m.CreateNamespace(ctx, name)
}

func (m *mockClient) DeleteNamespaceAndWait(ctx context.Context, name string, timeout time.Duration) error {
	return m.DeleteNamespace(ctx, name)
}

func (m *mockClient) RemoveFinalizersFromNamespace(ctx context.Context, namespace string) error {
	ns, err := m.fakeClientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	ns.Finalizers = nil
	_, err = m.fakeClientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

func (m *mockClient) forceDeleteNamespace(ctx context.Context, namespace string) error {
	ns, err := m.fakeClientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get namespace for force-finalize: %w", err)
	}
	ns.Spec.Finalizers = nil
	_, err = m.fakeClientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

func (m *mockClient) WaitForNamespaceDeletion(ctx context.Context, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := m.fakeClientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error checking namespace status: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for namespace %q to be deleted", namespace)
}

func (m *mockClient) CleanupTerminatingNamespace(ctx context.Context, namespace string) error {
	ns, err := m.fakeClientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get namespace %s: %w", namespace, err)
	}
	if ns.Status.Phase != corev1.NamespaceTerminating {
		return nil
	}
	if err := m.RemoveFinalizersFromNamespace(ctx, namespace); err != nil {
		return err
	}
	return m.forceDeleteNamespace(ctx, namespace)
}

// TestClient_Getters tests the client getter methods
func TestClient_Getters(t *testing.T) {
	client := createMockClient()

	if client.Platform() != "k3s" {
		t.Errorf("Platform() = %v, want k3s", client.Platform())
	}

	if client.Clientset() == nil {
		t.Error("Clientset() returned nil")
	}

	if client.Dynamic() == nil {
		t.Error("Dynamic() returned nil")
	}

	if client.RESTConfig() == nil {
		t.Error("RESTConfig() returned nil")
	}
}

// TestClient_IsOpenShift tests the IsOpenShift method
func TestClient_IsOpenShift(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool
	}{
		{
			name:     "openshift_platform",
			platform: "openshift",
			expected: true,
		},
		{
			name:     "k3s_platform",
			platform: "k3s",
			expected: false,
		},
		{
			name:     "eks_platform",
			platform: "eks",
			expected: false,
		},
		{
			name:     "empty_platform",
			platform: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				platform: tt.platform,
			}

			result := client.IsOpenShift()
			if result != tt.expected {
				t.Errorf("IsOpenShift() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestConfig_Fields tests the Config struct fields
func TestConfig_Fields(t *testing.T) {
	config := Config{
		KubeconfigPath:        "/path/to/kubeconfig",
		Platform:              "k3s",
		AWSRegion:             "us-west-2",
		AWSProfile:            "production",
		AWSAccessKey:          "AKIAIOSFODNN7EXAMPLE",
		AWSSecretKey:          "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		EKSClusterName:        "prod-cluster",
		OCPServer:             "https://api.ocp.example.com:6443",
		OCPUsername:           "admin",
		OCPPassword:           "secret",
		OCPToken:              "sha256~token",
		OCPInsecureSkipVerify: true,
	}

	// Verify all fields are accessible and set correctly
	if config.KubeconfigPath != "/path/to/kubeconfig" {
		t.Error("KubeconfigPath mismatch")
	}
	if config.Platform != "k3s" {
		t.Error("Platform mismatch")
	}
	if config.AWSRegion != "us-west-2" {
		t.Error("AWSRegion mismatch")
	}
	if config.AWSProfile != "production" {
		t.Error("AWSProfile mismatch")
	}
	if config.EKSClusterName != "prod-cluster" {
		t.Error("EKSClusterName mismatch")
	}
	if config.OCPServer != "https://api.ocp.example.com:6443" {
		t.Error("OCPServer mismatch")
	}
	if config.OCPUsername != "admin" {
		t.Error("OCPUsername mismatch")
	}
	if !config.OCPInsecureSkipVerify {
		t.Error("OCPInsecureSkipVerify should be true")
	}
}

// TestNewConfigMap tests the NewConfigMap helper function
func TestNewConfigMap(t *testing.T) {
	namespace := "test-namespace"
	name := "test-configmap"
	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	cm := NewConfigMap(namespace, name, data)

	if cm == nil {
		t.Fatal("NewConfigMap returned nil")
	}

	if cm.Namespace != namespace {
		t.Errorf("Namespace = %v, want %v", cm.Namespace, namespace)
	}

	if cm.Name != name {
		t.Errorf("Name = %v, want %v", cm.Name, name)
	}

	if len(cm.Data) != len(data) {
		t.Errorf("Data length = %v, want %v", len(cm.Data), len(data))
	}

	for k, v := range data {
		if cm.Data[k] != v {
			t.Errorf("Data[%s] = %v, want %v", k, cm.Data[k], v)
		}
	}
}

// TestNewConfigMapWithBinaryData tests the NewConfigMapWithBinaryData helper function
func TestNewConfigMapWithBinaryData(t *testing.T) {
	namespace := "test-namespace"
	name := "test-configmap"
	data := map[string]string{
		"text": "value",
	}
	binaryData := map[string][]byte{
		"binary": []byte{0x01, 0x02, 0x03},
	}

	cm := NewConfigMapWithBinaryData(namespace, name, data, binaryData)

	if cm == nil {
		t.Fatal("NewConfigMapWithBinaryData returned nil")
	}

	if cm.Namespace != namespace {
		t.Errorf("Namespace = %v, want %v", cm.Namespace, namespace)
	}

	if cm.Name != name {
		t.Errorf("Name = %v, want %v", cm.Name, name)
	}

	if len(cm.Data) != len(data) {
		t.Errorf("Data length = %v, want %v", len(cm.Data), len(data))
	}

	if len(cm.BinaryData) != len(binaryData) {
		t.Errorf("BinaryData length = %v, want %v", len(cm.BinaryData), len(binaryData))
	}

	if string(cm.BinaryData["binary"]) != string(binaryData["binary"]) {
		t.Error("BinaryData mismatch")
	}
}

// TestClient_NamespaceExists tests namespace existence check with mock client
func TestClient_NamespaceExists(t *testing.T) {
	ctx := context.Background()

	// Create mock client with existing namespace
	existingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "existing-namespace",
		},
	}
	client := createMockClient(existingNS)

	tests := []struct {
		name      string
		namespace string
		expected  bool
		wantErr   bool
	}{
		{
			name:      "existing_namespace",
			namespace: "existing-namespace",
			expected:  true,
			wantErr:   false,
		},
		{
			name:      "non_existing_namespace",
			namespace: "non-existing",
			expected:  false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := client.NamespaceExists(ctx, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("NamespaceExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.expected {
				t.Errorf("NamespaceExists() = %v, want %v", exists, tt.expected)
			}
		})
	}
}

// TestClient_CreateNamespace tests namespace creation with mock client
func TestClient_CreateNamespace(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{
			name:      "create_new_namespace",
			namespace: "new-namespace",
			wantErr:   false,
		},
		{
			name:      "create_another_namespace",
			namespace: "another-namespace",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockClient()

			err := client.CreateNamespace(ctx, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify namespace was created
			if !tt.wantErr {
				exists, err := client.NamespaceExists(ctx, tt.namespace)
				if err != nil {
					t.Errorf("Failed to check namespace existence: %v", err)
				}
				if !exists {
					t.Errorf("Namespace %s was not created", tt.namespace)
				}
			}
		})
	}
}

// TestClient_GetConfigMap tests ConfigMap retrieval with mock client
func TestClient_GetConfigMap(t *testing.T) {
	ctx := context.Background()

	// Create mock client with existing ConfigMap
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	client := createMockClient(existingCM)

	tests := []struct {
		name      string
		namespace string
		cmName    string
		wantErr   bool
	}{
		{
			name:      "get_existing_configmap",
			namespace: "default",
			cmName:    "test-cm",
			wantErr:   false,
		},
		{
			name:      "get_non_existing_configmap",
			namespace: "default",
			cmName:    "non-existing",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := client.GetConfigMap(ctx, tt.namespace, tt.cmName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cm.Name != tt.cmName {
					t.Errorf("ConfigMap name = %v, want %v", cm.Name, tt.cmName)
				}
				if cm.Namespace != tt.namespace {
					t.Errorf("ConfigMap namespace = %v, want %v", cm.Namespace, tt.namespace)
				}
			}
		})
	}
}

// TestClient_CreateConfigMap tests ConfigMap creation with mock client
func TestClient_CreateConfigMap(t *testing.T) {
	ctx := context.Background()
	client := createMockClient()

	cm := NewConfigMap("default", "new-cm", map[string]string{
		"key": "value",
	})

	err := client.CreateConfigMap(ctx, cm)
	if err != nil {
		t.Errorf("CreateConfigMap() error = %v", err)
		return
	}

	// Verify ConfigMap was created
	retrieved, err := client.GetConfigMap(ctx, "default", "new-cm")
	if err != nil {
		t.Errorf("Failed to retrieve created ConfigMap: %v", err)
		return
	}

	if retrieved.Data["key"] != "value" {
		t.Errorf("ConfigMap data mismatch: got %v, want value", retrieved.Data["key"])
	}
}

// TestClient_GetConfigMapField tests ConfigMap field retrieval with mock client
func TestClient_GetConfigMapField(t *testing.T) {
	ctx := context.Background()

	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"field1": "value1",
			"field2": "value2",
		},
	}
	client := createMockClient(existingCM)

	tests := []struct {
		name      string
		namespace string
		cmName    string
		field     string
		expected  string
		wantErr   bool
	}{
		{
			name:      "get_existing_field",
			namespace: "default",
			cmName:    "test-cm",
			field:     "field1",
			expected:  "value1",
			wantErr:   false,
		},
		{
			name:      "get_non_existing_field",
			namespace: "default",
			cmName:    "test-cm",
			field:     "non-existing",
			expected:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := client.GetConfigMapField(ctx, tt.namespace, tt.cmName, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigMapField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && value != tt.expected {
				t.Errorf("GetConfigMapField() = %v, want %v", value, tt.expected)
			}
		})
	}
}

// TestClient_DeleteConfigMap tests ConfigMap deletion with mock client
func TestClient_DeleteConfigMap(t *testing.T) {
	ctx := context.Background()

	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
	}
	client := createMockClient(existingCM)

	// Delete existing ConfigMap
	err := client.DeleteConfigMap(ctx, "default", "test-cm")
	if err != nil {
		t.Errorf("DeleteConfigMap() error = %v", err)
		return
	}

	// Verify ConfigMap was deleted
	_, err = client.GetConfigMap(ctx, "default", "test-cm")
	if err == nil {
		t.Error("ConfigMap should have been deleted but still exists")
	}

	// Delete non-existing ConfigMap (should not error)
	err = client.DeleteConfigMap(ctx, "default", "non-existing")
	if err != nil {
		t.Errorf("DeleteConfigMap() on non-existing CM should not error, got: %v", err)
	}
}

// TestClient_ListNodeNames tests node listing with mock client
func TestClient_ListNodeNames(t *testing.T) {
	ctx := context.Background()

	// Create mock client with nodes
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node2",
		},
	}
	client := createMockClient(node1, node2)

	names, err := client.ListNodeNames(ctx)
	if err != nil {
		t.Errorf("ListNodeNames() error = %v", err)
		return
	}

	if len(names) != 2 {
		t.Errorf("ListNodeNames() returned %d nodes, want 2", len(names))
	}

	expectedNames := map[string]bool{"node1": true, "node2": true}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("Unexpected node name: %s", name)
		}
	}
}

// TestClient_ListWorkerNodeNames tests worker node listing with mock client
func TestClient_ListWorkerNodeNames(t *testing.T) {
	ctx := context.Background()

	// Create mock client with master and worker nodes
	masterNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-node",
			Labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "",
			},
		},
	}
	workerNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "worker-node",
			Labels: map[string]string{},
		},
	}
	client := createMockClient(masterNode, workerNode)

	names, err := client.ListWorkerNodeNames(ctx)
	if err != nil {
		t.Errorf("ListWorkerNodeNames() error = %v", err)
		return
	}

	if len(names) != 1 {
		t.Errorf("ListWorkerNodeNames() returned %d nodes, want 1", len(names))
	}

	if names[0] != "worker-node" {
		t.Errorf("ListWorkerNodeNames() = %v, want [worker-node]", names)
	}
}

// TestClient_GetNodeInternalIP tests node IP retrieval with mock client
func TestClient_GetNodeInternalIP(t *testing.T) {
	ctx := context.Background()

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.100",
				},
				{
					Type:    corev1.NodeExternalIP,
					Address: "203.0.113.1",
				},
			},
		},
	}
	client := createMockClient(node)

	ip, err := client.GetNodeInternalIP(ctx, "test-node")
	if err != nil {
		t.Errorf("GetNodeInternalIP() error = %v", err)
		return
	}

	if ip != "192.168.1.100" {
		t.Errorf("GetNodeInternalIP() = %v, want 192.168.1.100", ip)
	}
}

// TestClient_GetNodeExternalIP tests external IP retrieval with mock client
func TestClient_GetNodeExternalIP(t *testing.T) {
	ctx := context.Background()

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.100",
				},
				{
					Type:    corev1.NodeExternalIP,
					Address: "203.0.113.1",
				},
			},
		},
	}
	client := createMockClient(node)

	ip, err := client.GetNodeExternalIP(ctx, "test-node")
	if err != nil {
		t.Errorf("GetNodeExternalIP() error = %v", err)
		return
	}

	if ip != "203.0.113.1" {
		t.Errorf("GetNodeExternalIP() = %v, want 203.0.113.1", ip)
	}
}

// TestClient_DeleteNamespace tests namespace deletion with mock client
func TestClient_DeleteNamespace(t *testing.T) {
	ctx := context.Background()

	existingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := createMockClient(existingNS)

	// Delete existing namespace
	err := client.DeleteNamespace(ctx, "test-namespace")
	if err != nil {
		t.Errorf("DeleteNamespace() error = %v", err)
		return
	}

	// Verify namespace was deleted
	exists, err := client.NamespaceExists(ctx, "test-namespace")
	if err != nil {
		t.Errorf("Failed to check namespace existence: %v", err)
	}
	if exists {
		t.Error("Namespace should have been deleted but still exists")
	}

	// Delete non-existing namespace (should not error)
	err = client.DeleteNamespace(ctx, "non-existing")
	if err != nil {
		t.Errorf("DeleteNamespace() on non-existing namespace should not error, got: %v", err)
	}
}

// TestClient_WaitForConfigMapExists tests waiting for ConfigMap with mock client
func TestClient_WaitForConfigMapExists(t *testing.T) {
	ctx := context.Background()
	client := createMockClient()

	// Create ConfigMap in background after a delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cm := NewConfigMap("default", "delayed-cm", map[string]string{"key": "value"})
		_ = client.CreateConfigMap(context.Background(), cm)
	}()

	// Wait for ConfigMap to exist
	err := client.WaitForConfigMapExists(ctx, "default", "delayed-cm", 2*time.Second)
	if err != nil {
		t.Errorf("WaitForConfigMapExists() error = %v", err)
	}
}

// TestClient_UpdateConfigMap tests ConfigMap update with mock client
func TestClient_UpdateConfigMap(t *testing.T) {
	ctx := context.Background()

	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "old-value",
		},
	}
	client := createMockClient(existingCM)

	// Get the ConfigMap to get its ResourceVersion
	cm, err := client.GetConfigMap(ctx, "default", "test-cm")
	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Update the data
	cm.Data["key"] = "new-value"

	err = client.UpdateConfigMap(ctx, cm)
	if err != nil {
		t.Errorf("UpdateConfigMap() error = %v", err)
		return
	}

	// Verify update
	updated, err := client.GetConfigMap(ctx, "default", "test-cm")
	if err != nil {
		t.Errorf("Failed to get updated ConfigMap: %v", err)
		return
	}

	if updated.Data["key"] != "new-value" {
		t.Errorf("ConfigMap data = %v, want new-value", updated.Data["key"])
	}
}

// TestClient_CreateOrUpdateConfigMap tests ConfigMap create or update with mock client
func TestClient_CreateOrUpdateConfigMap(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupFunc  func(*mockClient)
		cmToApply  *corev1.ConfigMap
		wantErr    bool
		finalValue string
	}{
		{
			name:       "create_new_configmap",
			setupFunc:  func(c *mockClient) {},
			cmToApply:  NewConfigMap("default", "new-cm", map[string]string{"key": "value"}),
			wantErr:    false,
			finalValue: "value",
		},
		{
			name: "update_existing_configmap",
			setupFunc: func(c *mockClient) {
				cm := NewConfigMap("default", "existing-cm", map[string]string{"key": "old"})
				_ = c.CreateConfigMap(context.Background(), cm)
			},
			cmToApply:  NewConfigMap("default", "existing-cm", map[string]string{"key": "new"}),
			wantErr:    false,
			finalValue: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockClient()
			tt.setupFunc(client)

			err := client.CreateOrUpdateConfigMap(ctx, tt.cmToApply)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateConfigMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				cm, err := client.GetConfigMap(ctx, tt.cmToApply.Namespace, tt.cmToApply.Name)
				if err != nil {
					t.Errorf("Failed to get ConfigMap: %v", err)
					return
				}
				if cm.Data["key"] != tt.finalValue {
					t.Errorf("ConfigMap data = %v, want %v", cm.Data["key"], tt.finalValue)
				}
			}
		})
	}
}

// TestClient_GetNamespace tests namespace retrieval with mock client
func TestClient_GetNamespace(t *testing.T) {
	ctx := context.Background()

	existingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	client := createMockClient(existingNS)

	ns, err := client.GetNamespace(ctx, "test-namespace")
	if err != nil {
		t.Errorf("GetNamespace() error = %v", err)
		return
	}

	if ns.Name != "test-namespace" {
		t.Errorf("Namespace name = %v, want test-namespace", ns.Name)
	}

	if ns.Status.Phase != corev1.NamespaceActive {
		t.Errorf("Namespace phase = %v, want Active", ns.Status.Phase)
	}
}

// Made with Bob
