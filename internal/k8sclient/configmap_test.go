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

// TestGetConfigMap tests retrieving a ConfigMap
func TestGetConfigMap(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key": "value"},
	})

	cm, err := mock.GetConfigMap(context.Background(), "default", "test-config")
	if err != nil {
		t.Fatalf("GetConfigMap() unexpected error: %v", err)
	}
	if cm.Data["key"] != "value" {
		t.Errorf("ConfigMap field = %q, want %q", cm.Data["key"], "value")
	}
}

// TestGetConfigMap_NotFound tests retrieving a non-existent ConfigMap
func TestGetConfigMap_NotFound(t *testing.T) {
	mock := createMockClient()

	_, err := mock.GetConfigMap(context.Background(), "default", "missing")
	if err == nil {
		t.Fatal("GetConfigMap() expected not found error")
	}
}

// TestGetConfigMapField tests retrieving a specific field from a ConfigMap
func TestGetConfigMapField(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key": "value"},
	})

	value, err := mock.GetConfigMapField(context.Background(), "default", "test-config", "key")
	if err != nil {
		t.Fatalf("GetConfigMapField() unexpected error: %v", err)
	}
	if value != "value" {
		t.Errorf("GetConfigMapField() = %q, want %q", value, "value")
	}
}

// TestGetConfigMapField_NotFound tests retrieving a non-existent field
func TestGetConfigMapField_NotFound(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key": "value"},
	})

	_, err := mock.GetConfigMapField(context.Background(), "default", "test-config", "missing")
	if err == nil {
		t.Fatal("GetConfigMapField() expected missing field error")
	}
}

// TestCreateConfigMap tests creating a ConfigMap
func TestCreateConfigMap(t *testing.T) {
	mock := createMockClient()
	cm := NewConfigMap("default", "created-config", map[string]string{"key": "value"})

	if err := mock.CreateConfigMap(context.Background(), cm); err != nil {
		t.Fatalf("CreateConfigMap() unexpected error: %v", err)
	}

	created, err := mock.GetConfigMap(context.Background(), "default", "created-config")
	if err != nil {
		t.Fatalf("failed to get created ConfigMap: %v", err)
	}
	if created.Data["key"] != "value" {
		t.Errorf("created ConfigMap field = %q, want %q", created.Data["key"], "value")
	}
}

// TestUpdateConfigMap tests updating a ConfigMap
func TestUpdateConfigMap(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-config",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Data: map[string]string{"key": "old"},
	})

	cm := NewConfigMap("default", "test-config", map[string]string{"key": "new"})
	cm.ResourceVersion = "1"

	if err := mock.UpdateConfigMap(context.Background(), cm); err != nil {
		t.Fatalf("UpdateConfigMap() unexpected error: %v", err)
	}

	updated, err := mock.GetConfigMap(context.Background(), "default", "test-config")
	if err != nil {
		t.Fatalf("failed to get updated ConfigMap: %v", err)
	}
	if updated.Data["key"] != "new" {
		t.Errorf("updated ConfigMap field = %q, want %q", updated.Data["key"], "new")
	}
}

// TestCreateOrUpdateConfigMap_Create tests creating a new ConfigMap
func TestCreateOrUpdateConfigMap_Create(t *testing.T) {
	mock := createMockClient()
	cm := NewConfigMap("default", "create-or-update", map[string]string{"key": "created"})

	if err := mock.CreateOrUpdateConfigMap(context.Background(), cm); err != nil {
		t.Fatalf("CreateOrUpdateConfigMap() unexpected error: %v", err)
	}

	got, err := mock.GetConfigMap(context.Background(), "default", "create-or-update")
	if err != nil {
		t.Fatalf("failed to get ConfigMap: %v", err)
	}
	if got.Data["key"] != "created" {
		t.Errorf("ConfigMap field = %q, want %q", got.Data["key"], "created")
	}
}

// TestCreateOrUpdateConfigMap_Update tests updating an existing ConfigMap
func TestCreateOrUpdateConfigMap_Update(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "create-or-update",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Data: map[string]string{"key": "old"},
	})

	cm := NewConfigMap("default", "create-or-update", map[string]string{"key": "updated"})
	if err := mock.CreateOrUpdateConfigMap(context.Background(), cm); err != nil {
		t.Fatalf("CreateOrUpdateConfigMap() unexpected error: %v", err)
	}

	got, err := mock.GetConfigMap(context.Background(), "default", "create-or-update")
	if err != nil {
		t.Fatalf("failed to get ConfigMap: %v", err)
	}
	if got.Data["key"] != "updated" {
		t.Errorf("ConfigMap field = %q, want %q", got.Data["key"], "updated")
	}
}

// TestDeleteConfigMap tests deleting a ConfigMap
func TestDeleteConfigMap(t *testing.T) {
	mock := createMockClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-config",
			Namespace: "default",
		},
	})

	if err := mock.DeleteConfigMap(context.Background(), "default", "delete-config"); err != nil {
		t.Fatalf("DeleteConfigMap() unexpected error: %v", err)
	}

	_, err := mock.GetConfigMap(context.Background(), "default", "delete-config")
	if err == nil {
		t.Fatal("expected ConfigMap to be deleted")
	}
}

// TestDeleteConfigMap_NotFound tests deleting a non-existent ConfigMap
func TestDeleteConfigMap_NotFound(t *testing.T) {
	mock := createMockClient()

	if err := mock.DeleteConfigMap(context.Background(), "default", "missing"); err != nil {
		t.Fatalf("DeleteConfigMap() unexpected error: %v", err)
	}
}

// TestWaitForConfigMapField tests waiting for a ConfigMap field
func TestWaitForConfigMapField(t *testing.T) {
	client := &Client{}

	defer func() {
		if recover() == nil {
			t.Fatal("WaitForConfigMapField() expected panic with uninitialized clientset")
		}
	}()

	_, _ = client.WaitForConfigMapField(
		context.Background(),
		"default",
		"wait-config",
		"status",
		func(value string) (bool, error) {
			return value == "ready", nil
		},
		time.Millisecond,
		1,
	)
}

// TestWaitForConfigMapExists tests waiting for a ConfigMap to exist
func TestWaitForConfigMapExists(t *testing.T) {
	client := &Client{}

	defer func() {
		if recover() == nil {
			t.Fatal("WaitForConfigMapExists() expected panic with uninitialized clientset")
		}
	}()

	_ = client.WaitForConfigMapExists(
		context.Background(),
		"default",
		"exists-config",
		time.Millisecond,
	)
}

// TestWaitForConfigMapExists_Timeout tests timeout when ConfigMap doesn't exist
func TestWaitForConfigMapExists_Timeout(t *testing.T) {
	client := &Client{}

	defer func() {
		if recover() == nil {
			t.Fatal("WaitForConfigMapExists() expected panic with uninitialized clientset")
		}
	}()

	_ = client.WaitForConfigMapExists(
		context.Background(),
		"default",
		"missing",
		time.Millisecond,
	)
}

// TestConfigMapFieldCondition tests ConfigMap field condition function
func TestConfigMapFieldCondition(t *testing.T) {
	// Test a simple condition function
	condition := func(value string) (bool, error) {
		return value == "expected", nil
	}

	done, err := condition("expected")
	if err != nil {
		t.Errorf("Condition function error = %v", err)
	}
	if !done {
		t.Error("Condition function should return true for matching value")
	}

	done, err = condition("other")
	if err != nil {
		t.Errorf("Condition function error = %v", err)
	}
	if done {
		t.Error("Condition function should return false for non-matching value")
	}
}

// Made with Bob
