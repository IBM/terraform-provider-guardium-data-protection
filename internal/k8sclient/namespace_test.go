// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNamespaceExists tests checking if a namespace exists
func TestNamespaceExists(t *testing.T) {
	mock := createMockClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "existing"},
	})

	exists, err := mock.NamespaceExists(context.Background(), "existing")
	if err != nil {
		t.Fatalf("NamespaceExists() unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("NamespaceExists() = false, want true")
	}
}

// TestNamespaceExists_NotFound tests checking for a non-existent namespace
func TestNamespaceExists_NotFound(t *testing.T) {
	mock := createMockClient()

	exists, err := mock.NamespaceExists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("NamespaceExists() unexpected error: %v", err)
	}
	if exists {
		t.Fatal("NamespaceExists() = true, want false")
	}
}

// TestCreateNamespace tests creating a namespace
func TestCreateNamespace(t *testing.T) {
	mock := createMockClient()

	if err := mock.CreateNamespace(context.Background(), "created"); err != nil {
		t.Fatalf("CreateNamespace() unexpected error: %v", err)
	}

	ns, err := mock.GetNamespace(context.Background(), "created")
	if err != nil {
		t.Fatalf("failed to get created namespace: %v", err)
	}
	if ns.Name != "created" {
		t.Errorf("namespace name = %q, want %q", ns.Name, "created")
	}
}

// TestCreateNamespace_AlreadyExists tests creating an existing namespace
func TestCreateNamespace_AlreadyExists(t *testing.T) {
	mock := createMockClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "existing"},
	})

	if err := mock.CreateNamespace(context.Background(), "existing"); err != nil {
		t.Fatalf("CreateNamespace() unexpected error: %v", err)
	}
}

// TestCreateNamespaceAndWait tests creating a namespace and waiting for it
func TestCreateNamespaceAndWait(t *testing.T) {
	mock := createMockClient()

	if err := mock.CreateNamespaceAndWait(context.Background(), "waited-ns", 0); err != nil {
		t.Fatalf("CreateNamespaceAndWait() unexpected error: %v", err)
	}

	exists, err := mock.NamespaceExists(context.Background(), "waited-ns")
	if err != nil {
		t.Fatalf("NamespaceExists() unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("namespace should exist after CreateNamespaceAndWait")
	}
}

// TestDeleteNamespace tests deleting a namespace
func TestDeleteNamespace(t *testing.T) {
	mock := createMockClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "delete-me"},
	})

	if err := mock.DeleteNamespace(context.Background(), "delete-me"); err != nil {
		t.Fatalf("DeleteNamespace() unexpected error: %v", err)
	}

	exists, err := mock.NamespaceExists(context.Background(), "delete-me")
	if err != nil {
		t.Fatalf("NamespaceExists() unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected namespace to be deleted")
	}
}

// TestDeleteNamespace_NotFound tests deleting a non-existent namespace
func TestDeleteNamespace_NotFound(t *testing.T) {
	mock := createMockClient()

	if err := mock.DeleteNamespace(context.Background(), "missing"); err != nil {
		t.Fatalf("DeleteNamespace() unexpected error: %v", err)
	}
}

// TestDeleteNamespaceAndWait tests deleting a namespace and waiting
func TestDeleteNamespaceAndWait(t *testing.T) {
	mock := createMockClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "delete-wait-ns"},
	})

	if err := mock.DeleteNamespaceAndWait(context.Background(), "delete-wait-ns", 0); err != nil {
		t.Fatalf("DeleteNamespaceAndWait() unexpected error: %v", err)
	}

	exists, err := mock.NamespaceExists(context.Background(), "delete-wait-ns")
	if err != nil {
		t.Fatalf("NamespaceExists() unexpected error: %v", err)
	}
	if exists {
		t.Fatal("namespace should be gone after DeleteNamespaceAndWait")
	}
}

// TestGetNamespace tests retrieving a namespace
func TestGetNamespace(t *testing.T) {
	mock := createMockClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "existing"},
	})

	ns, err := mock.GetNamespace(context.Background(), "existing")
	if err != nil {
		t.Fatalf("GetNamespace() unexpected error: %v", err)
	}
	if ns.Name != "existing" {
		t.Errorf("namespace name = %q, want %q", ns.Name, "existing")
	}
}

// TestGetNamespace_NotFound tests retrieving a non-existent namespace
func TestGetNamespace_NotFound(t *testing.T) {
	mock := createMockClient()

	_, err := mock.GetNamespace(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetNamespace() expected not found error")
	}
}

// Made with Bob
