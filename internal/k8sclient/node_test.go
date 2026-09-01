// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testNode(name string, labels map[string]string, addresses ...corev1.NodeAddress) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Addresses: addresses,
		},
	}
}

// TestListNodeNames tests listing all node names
func TestListNodeNames(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", nil),
		testNode("node-2", nil),
	)

	names, err := mock.ListNodeNames(context.Background())
	if err != nil {
		t.Fatalf("ListNodeNames() unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("ListNodeNames() length = %d, want %d", len(names), 2)
	}
}

// TestListNodeNames_Empty tests listing nodes when none exist
func TestListNodeNames_Empty(t *testing.T) {
	mock := createMockClient()

	names, err := mock.ListNodeNames(context.Background())
	if err != nil {
		t.Fatalf("ListNodeNames() unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("ListNodeNames() length = %d, want %d", len(names), 0)
	}
}

// TestListNodeNamesWithLabel tests listing nodes with a specific label
func TestListNodeNamesWithLabel(t *testing.T) {
	mock := createMockClient(
		testNode("labeled-node", map[string]string{"env": "test"}),
		testNode("other-node", map[string]string{"env": "prod"}),
	)

	names, err := mock.ListNodeNamesWithLabel(context.Background(), "env=test")
	if err != nil {
		t.Fatalf("ListNodeNamesWithLabel() unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "labeled-node" {
		t.Fatalf("ListNodeNamesWithLabel() = %v, want [labeled-node]", names)
	}
}

// TestListNodeNamesWithLabel_NoMatch tests listing nodes with non-matching label
func TestListNodeNamesWithLabel_NoMatch(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", map[string]string{"env": "prod"}),
	)

	names, err := mock.ListNodeNamesWithLabel(context.Background(), "env=test")
	if err != nil {
		t.Fatalf("ListNodeNamesWithLabel() unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("ListNodeNamesWithLabel() = %v, want []", names)
	}
}

// TestListWorkerNodeNames tests listing worker node names
func TestListWorkerNodeNames(t *testing.T) {
	mock := createMockClient(
		testNode("worker-1", map[string]string{}),
		testNode("control-plane", map[string]string{"node-role.kubernetes.io/control-plane": ""}),
		testNode("master", map[string]string{"node-role.kubernetes.io/master": ""}),
	)

	names, err := mock.ListWorkerNodeNames(context.Background())
	if err != nil {
		t.Fatalf("ListWorkerNodeNames() unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "worker-1" {
		t.Fatalf("ListWorkerNodeNames() = %v, want [worker-1]", names)
	}
}

// TestListWorkerNodeNames_OnlyControlPlane tests when only control plane nodes exist
func TestListWorkerNodeNames_OnlyControlPlane(t *testing.T) {
	mock := createMockClient(
		testNode("control-plane", map[string]string{"node-role.kubernetes.io/control-plane": ""}),
		testNode("master", map[string]string{"node-role.kubernetes.io/master": ""}),
	)

	names, err := mock.ListWorkerNodeNames(context.Background())
	if err != nil {
		t.Fatalf("ListWorkerNodeNames() unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("ListWorkerNodeNames() = %v, want []", names)
	}
}

// TestGetNodeInternalIP tests getting a node's internal IP
func TestGetNodeInternalIP(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", nil, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: "10.0.0.10"}),
	)

	ip, err := mock.GetNodeInternalIP(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("GetNodeInternalIP() unexpected error: %v", err)
	}
	if ip != "10.0.0.10" {
		t.Errorf("GetNodeInternalIP() = %q, want %q", ip, "10.0.0.10")
	}
}

// TestGetNodeInternalIP_NotFound tests getting IP for non-existent node
func TestGetNodeInternalIP_NotFound(t *testing.T) {
	mock := createMockClient()

	_, err := mock.GetNodeInternalIP(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetNodeInternalIP() expected not found error")
	}
}

// TestGetNodeInternalIP_NoIP tests when node has no internal IP
func TestGetNodeInternalIP_NoIP(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", nil, corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: "1.2.3.4"}),
	)

	_, err := mock.GetNodeInternalIP(context.Background(), "node-1")
	if err == nil {
		t.Fatal("GetNodeInternalIP() expected missing internal IP error")
	}
}

// TestGetNodeExternalIP tests getting a node's external IP
func TestGetNodeExternalIP(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", nil, corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: "1.2.3.4"}),
	)

	ip, err := mock.GetNodeExternalIP(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("GetNodeExternalIP() unexpected error: %v", err)
	}
	if ip != "1.2.3.4" {
		t.Errorf("GetNodeExternalIP() = %q, want %q", ip, "1.2.3.4")
	}
}

// TestGetNodeExternalIP_NotFound tests getting IP for non-existent node
func TestGetNodeExternalIP_NotFound(t *testing.T) {
	mock := createMockClient()

	_, err := mock.GetNodeExternalIP(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetNodeExternalIP() expected not found error")
	}
}

// TestGetNodeExternalIP_NoIP tests when node has no external IP
func TestGetNodeExternalIP_NoIP(t *testing.T) {
	mock := createMockClient(
		testNode("node-1", nil, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: "10.0.0.10"}),
	)

	_, err := mock.GetNodeExternalIP(context.Background(), "node-1")
	if err == nil {
		t.Fatal("GetNodeExternalIP() expected missing external IP error")
	}
}

// Made with Bob
