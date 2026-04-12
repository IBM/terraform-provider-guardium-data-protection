// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

// newTestClient creates a real Client pointing at a non-existent server for unit tests.
// The client is structurally valid but all API calls will fail with a connection error.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(&rest.Config{Host: "https://127.0.0.1:6443"})
	if err != nil {
		t.Fatalf("newTestClient() unexpected error: %v", err)
	}
	return client
}

// TestNewClient tests creating a new OpenShift client
func TestNewClient(t *testing.T) {
	client, err := NewClient(&rest.Config{Host: "https://127.0.0.1:6443"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
	if client.ConfigClient() == nil {
		t.Fatal("ConfigClient() returned nil")
	}
	if client.MachineConfigClient() == nil {
		t.Fatal("MachineConfigClient() returned nil")
	}
	if client.Clientset() == nil {
		t.Fatal("Clientset() returned nil")
	}
}

// TestNewClient_InvalidConfig tests creating client with invalid config
func TestNewClient_InvalidConfig(t *testing.T) {
	client, err := NewClient(&rest.Config{Host: "://bad-url"})
	if err == nil {
		t.Fatal("NewClient() expected error for invalid config")
	}
	if client != nil {
		t.Fatalf("NewClient() client = %#v, want nil", client)
	}
	if !strings.Contains(err.Error(), "failed to create config client") {
		t.Fatalf("NewClient() error = %v, want config client creation failure", err)
	}
}

// TestConfigClient tests getting the config client
func TestConfigClient(t *testing.T) {
	client := newTestClient(t)
	if client.ConfigClient() == nil {
		t.Fatal("ConfigClient() returned nil")
	}
}

// TestMachineConfigClient tests getting the machine config client
func TestMachineConfigClient(t *testing.T) {
	client := newTestClient(t)
	if client.MachineConfigClient() == nil {
		t.Fatal("MachineConfigClient() returned nil")
	}
}

// TestClientset tests getting the Kubernetes clientset
func TestClientset(t *testing.T) {
	client := newTestClient(t)
	if client.Clientset() == nil {
		t.Fatal("Clientset() returned nil")
	}
}

// Made with Bob
