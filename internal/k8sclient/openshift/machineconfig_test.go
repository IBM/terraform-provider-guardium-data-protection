// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestIgnitionConfig_Marshal tests marshaling Ignition config
func TestIgnitionConfig_Marshal(t *testing.T) {
	mode := 0644
	certData := []byte("test-certificate-data")
	certBase64 := base64.StdEncoding.EncodeToString(certData)
	registryHost := "registry.example.com"
	certPath := fmt.Sprintf("/etc/containers/certs.d/%s/ca.crt", registryHost)

	config := IgnitionConfig{
		Ignition: IgnitionVersion{Version: "3.2.0"},
		Storage: &Storage{
			Files: []File{
				{
					Path: certPath,
					Mode: &mode,
					User: &FileUser{Name: "root"},
					Contents: FileContents{
						Source: fmt.Sprintf("data:text/plain;base64,%s", certBase64),
					},
				},
			},
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal Ignition config: %v", err)
		return
	}

	var decoded IgnitionConfig
	if err := json.Unmarshal(configJSON, &decoded); err != nil {
		t.Errorf("Failed to unmarshal Ignition config: %v", err)
		return
	}

	if decoded.Ignition.Version != "3.2.0" {
		t.Errorf("Ignition version = %q, want %q", decoded.Ignition.Version, "3.2.0")
	}

	if decoded.Storage == nil {
		t.Error("Storage is nil")
		return
	}

	if len(decoded.Storage.Files) != 1 {
		t.Errorf("Files length = %d, want %d", len(decoded.Storage.Files), 1)
		return
	}

	file := decoded.Storage.Files[0]
	if file.Path != certPath {
		t.Errorf("File path = %q, want %q", file.Path, certPath)
	}

	if file.User == nil || file.User.Name != "root" {
		t.Error("File user is not root")
	}

	expectedSource := fmt.Sprintf("data:text/plain;base64,%s", certBase64)
	if file.Contents.Source != expectedSource {
		t.Errorf("File contents source mismatch")
	}
}

// TestIgnitionConfig_EmptyStorage tests Ignition config with no storage
func TestIgnitionConfig_EmptyStorage(t *testing.T) {
	config := IgnitionConfig{
		Ignition: IgnitionVersion{Version: "3.2.0"},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal Ignition config: %v", err)
		return
	}

	var decoded IgnitionConfig
	if err := json.Unmarshal(configJSON, &decoded); err != nil {
		t.Errorf("Failed to unmarshal Ignition config: %v", err)
	}

	if decoded.Ignition.Version != "3.2.0" {
		t.Errorf("Ignition version = %q, want %q", decoded.Ignition.Version, "3.2.0")
	}
}

// TestIgnitionConfig_MultipleFiles tests Ignition config with multiple files
func TestIgnitionConfig_MultipleFiles(t *testing.T) {
	mode := 0644
	config := IgnitionConfig{
		Ignition: IgnitionVersion{Version: "3.2.0"},
		Storage: &Storage{
			Files: []File{
				{
					Path: "/etc/file1.txt",
					Mode: &mode,
					User: &FileUser{Name: "root"},
					Contents: FileContents{
						Source: "data:text/plain;base64,dGVzdDE=",
					},
				},
				{
					Path: "/etc/file2.txt",
					Mode: &mode,
					User: &FileUser{Name: "root"},
					Contents: FileContents{
						Source: "data:text/plain;base64,dGVzdDI=",
					},
				},
			},
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal Ignition config: %v", err)
		return
	}

	var decoded IgnitionConfig
	if err := json.Unmarshal(configJSON, &decoded); err != nil {
		t.Errorf("Failed to unmarshal Ignition config: %v", err)
		return
	}

	if len(decoded.Storage.Files) != 2 {
		t.Errorf("Files length = %d, want %d", len(decoded.Storage.Files), 2)
	}
}

// TestFileContents_Base64Encoding tests base64 encoding in file contents
func TestFileContents_Base64Encoding(t *testing.T) {
	testData := []byte("This is test certificate data")
	encoded := base64.StdEncoding.EncodeToString(testData)

	source := fmt.Sprintf("data:text/plain;base64,%s", encoded)
	contents := FileContents{Source: source}

	if contents.Source != source {
		t.Errorf("Source mismatch")
	}

	prefix := "data:text/plain;base64,"
	if len(source) <= len(prefix) {
		t.Error("Source too short")
		return
	}

	base64Part := source[len(prefix):]
	decoded, err := base64.StdEncoding.DecodeString(base64Part)
	if err != nil {
		t.Errorf("Failed to decode base64: %v", err)
		return
	}

	if string(decoded) != string(testData) {
		t.Errorf("Decoded data = %q, want %q", string(decoded), string(testData))
	}
}

// TestIgnitionVersion tests Ignition version struct
func TestIgnitionVersion(t *testing.T) {
	version := IgnitionVersion{Version: "3.2.0"}
	if version.Version != "3.2.0" {
		t.Errorf("Version = %q, want %q", version.Version, "3.2.0")
	}
}

// TestFileUser tests FileUser struct
func TestFileUser(t *testing.T) {
	user := FileUser{Name: "root"}
	if user.Name != "root" {
		t.Errorf("User name = %q, want %q", user.Name, "root")
	}
}

// TestFile_WithoutMode tests File struct without mode
func TestFile_WithoutMode(t *testing.T) {
	file := File{
		Path: "/etc/test.txt",
		Contents: FileContents{
			Source: "data:text/plain;base64,dGVzdA==",
		},
	}

	if file.Mode != nil {
		t.Error("Mode should be nil")
	}

	if file.Path != "/etc/test.txt" {
		t.Errorf("Path = %q, want %q", file.Path, "/etc/test.txt")
	}
}

// TestFile_WithMode tests File struct with mode
func TestFile_WithMode(t *testing.T) {
	mode := 0644
	file := File{
		Path: "/etc/test.txt",
		Mode: &mode,
		Contents: FileContents{
			Source: "data:text/plain;base64,dGVzdA==",
		},
	}

	if file.Mode == nil {
		t.Error("Mode should not be nil")
		return
	}

	if *file.Mode != 0644 {
		t.Errorf("Mode = %o, want %o", *file.Mode, 0644)
	}
}

// TestCreateRegistryCertMachineConfig_ConnectionError tests error wrapping
func TestCreateRegistryCertMachineConfig_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	err := client.CreateRegistryCertMachineConfig(context.Background(), "registry-ca", "registry.example.com", []byte("cert"))
	if err == nil {
		t.Fatal("CreateRegistryCertMachineConfig() expected error without accessible server")
	}
	if !strings.Contains(err.Error(), "failed to create machineconfig") {
		t.Fatalf("CreateRegistryCertMachineConfig() error = %v, want 'failed to create machineconfig'", err)
	}
}

// TestGetMachineConfig_ConnectionError tests error wrapping
func TestGetMachineConfig_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	mc, err := client.GetMachineConfig(context.Background(), "registry-ca")
	if err == nil {
		t.Fatal("GetMachineConfig() expected error without accessible server")
	}
	if mc != nil {
		t.Fatalf("GetMachineConfig() = %#v, want nil on error", mc)
	}
	if !strings.Contains(err.Error(), "failed to get machineconfig") {
		t.Fatalf("GetMachineConfig() error = %v, want 'failed to get machineconfig'", err)
	}
}

// TestDeleteMachineConfig_ConnectionError tests error wrapping
func TestDeleteMachineConfig_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	err := client.DeleteMachineConfig(context.Background(), "registry-ca")
	if err == nil {
		t.Fatal("DeleteMachineConfig() expected error without accessible server")
	}
	if !strings.Contains(err.Error(), "failed to delete machineconfig") {
		t.Fatalf("DeleteMachineConfig() error = %v, want 'failed to delete machineconfig'", err)
	}
}

// Made with Bob
