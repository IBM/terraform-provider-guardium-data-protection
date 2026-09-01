// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestRemoveFinalizersFromNamespace tests removing finalizers from a namespace
func TestRemoveFinalizersFromNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "finalized-ns",
			Finalizers: []string{"test-finalizer"},
		},
	}
	mock := createMockClient(ns)

	if err := mock.RemoveFinalizersFromNamespace(context.Background(), "finalized-ns"); err != nil {
		t.Fatalf("RemoveFinalizersFromNamespace() unexpected error: %v", err)
	}

	updated, err := mock.GetNamespace(context.Background(), "finalized-ns")
	if err != nil {
		t.Fatalf("GetNamespace() unexpected error: %v", err)
	}
	if len(updated.Finalizers) != 0 {
		t.Fatalf("Finalizers = %v, want empty", updated.Finalizers)
	}
}

// TestRemoveFinalizersFromNamespace_PartialErrors tests graceful handling when namespace not found
func TestRemoveFinalizersFromNamespace_PartialErrors(t *testing.T) {
	mock := createMockClient()

	// Should return nil (not found is tolerated)
	if err := mock.RemoveFinalizersFromNamespace(context.Background(), "missing"); err != nil {
		t.Fatalf("RemoveFinalizersFromNamespace() on missing namespace error = %v, want nil", err)
	}
}

// TestForceDeleteNamespace tests clearing spec.finalizers on a namespace
func TestForceDeleteNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "force-ns"},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{"kubernetes"},
		},
	}
	mock := createMockClient(ns)

	if err := mock.forceDeleteNamespace(context.Background(), "force-ns"); err != nil {
		t.Fatalf("forceDeleteNamespace() unexpected error: %v", err)
	}

	updated, err := mock.fakeClientset.CoreV1().Namespaces().Get(context.Background(), "force-ns", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Namespace should still exist: %v", err)
	}
	if len(updated.Spec.Finalizers) != 0 {
		t.Fatalf("Spec.Finalizers = %v, want empty", updated.Spec.Finalizers)
	}
}

// TestForceDeleteNamespace_NotFound tests force-deleting a non-existent namespace
func TestForceDeleteNamespace_NotFound(t *testing.T) {
	mock := createMockClient()

	if err := mock.forceDeleteNamespace(context.Background(), "missing"); err != nil {
		t.Fatalf("forceDeleteNamespace() on missing namespace error = %v, want nil", err)
	}
}

// TestWaitForNamespaceDeletion tests waiting until a namespace is gone
func TestWaitForNamespaceDeletion(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "delete-me"},
	}
	mock := createMockClient(ns)

	// Delete after a short delay to exercise the polling loop
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = mock.fakeClientset.CoreV1().Namespaces().Delete(context.Background(), "delete-me", metav1.DeleteOptions{})
	}()

	if err := mock.WaitForNamespaceDeletion(context.Background(), "delete-me", 2*time.Second); err != nil {
		t.Fatalf("WaitForNamespaceDeletion() unexpected error: %v", err)
	}
}

// TestWaitForNamespaceDeletion_Timeout tests that timeout error is returned
func TestWaitForNamespaceDeletion_Timeout(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "sticky-ns"},
	}
	mock := createMockClient(ns)

	err := mock.WaitForNamespaceDeletion(context.Background(), "sticky-ns", 50*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForNamespaceDeletion() expected timeout error, got nil")
	}
}

// TestCleanupTerminatingNamespace tests cleaning up a stuck terminating namespace
func TestCleanupTerminatingNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "terminating-ns",
			Finalizers: []string{"test-finalizer"},
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
	}
	mock := createMockClient(ns)

	if err := mock.CleanupTerminatingNamespace(context.Background(), "terminating-ns"); err != nil {
		t.Fatalf("CleanupTerminatingNamespace() unexpected error: %v", err)
	}
}

// TestCleanupTerminatingNamespace_NotTerminating tests that active namespaces are left alone
func TestCleanupTerminatingNamespace_NotTerminating(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "active-ns"},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
	mock := createMockClient(ns)

	if err := mock.CleanupTerminatingNamespace(context.Background(), "active-ns"); err != nil {
		t.Fatalf("CleanupTerminatingNamespace() unexpected error: %v", err)
	}
}

// TestCleanupTerminatingNamespace_NotFound tests that missing namespaces are silently ignored
func TestCleanupTerminatingNamespace_NotFound(t *testing.T) {
	mock := createMockClient()

	if err := mock.CleanupTerminatingNamespace(context.Background(), "missing"); err != nil {
		t.Fatalf("CleanupTerminatingNamespace() on missing namespace error = %v, want nil", err)
	}
}

// TestContainsVerb tests the containsVerb helper function
func TestContainsVerb(t *testing.T) {
	tests := []struct {
		name   string
		verbs  metav1.Verbs
		target string
		want   bool
	}{
		{
			name:   "verb exists",
			verbs:  metav1.Verbs{"get", "list", "watch"},
			target: "list",
			want:   true,
		},
		{
			name:   "verb does not exist",
			verbs:  metav1.Verbs{"get", "list", "watch"},
			target: "delete",
			want:   false,
		},
		{
			name:   "empty verbs",
			verbs:  metav1.Verbs{},
			target: "get",
			want:   false,
		},
		{
			name:   "first verb",
			verbs:  metav1.Verbs{"create", "update", "delete"},
			target: "create",
			want:   true,
		},
		{
			name:   "last verb",
			verbs:  metav1.Verbs{"create", "update", "delete"},
			target: "delete",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsVerb(tt.verbs, tt.target)
			if got != tt.want {
				t.Errorf("containsVerb() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Made with Bob
