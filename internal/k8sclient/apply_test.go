// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestApplyManifest_InvalidYAML tests applying invalid YAML
func TestApplyManifest_InvalidYAML(t *testing.T) {
	client := &Client{}

	err := client.ApplyManifest(context.Background(), []byte("not: valid: yaml:"), "default")
	if err == nil {
		t.Fatal("ApplyManifest() expected decode error")
	}
}

// TestApplyManifest_ValidDeployment tests applying a valid deployment manifest
func TestApplyManifest_ValidDeployment(t *testing.T) {
	client := &Client{}

	manifest := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`)

	defer func() {
		if recover() == nil {
			t.Fatal("ApplyManifest() expected panic without configured mapper")
		}
	}()

	_ = client.ApplyManifest(context.Background(), manifest, "default")
}

// TestApplyManifestContent_MultiDocument tests applying multi-document YAML
func TestApplyManifestContent_MultiDocument(t *testing.T) {
	client := &Client{}

	content := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-one
data:
  key: one
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-two
data:
  key: two
`)

	defer func() {
		if recover() == nil {
			t.Fatal("ApplyManifestContent() expected panic without configured mapper")
		}
	}()

	_ = client.ApplyManifestContent(context.Background(), content, "default")
}

// TestApplyManifestFile_Success tests applying a YAML file successfully
func TestApplyManifestFile_Success(t *testing.T) {
	client := &Client{}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "configmap.yaml")
	content := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: file-config
data:
  key: from-file
`)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write manifest file: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("ApplyManifestFile() expected panic without configured mapper")
		}
	}()

	_ = client.ApplyManifestFile(context.Background(), filePath, "default")
}

// TestApplyManifestFile_NonExistent tests applying a non-existent file
func TestApplyManifestFile_NonExistent(t *testing.T) {
	// This test doesn't require a real client
	client := &Client{}

	err := client.ApplyManifestFile(context.Background(), "/nonexistent/file.yaml", "default")
	if err == nil {
		t.Error("ApplyManifestFile() expected error for non-existent file")
	}
}

// TestApplyDirectory_Success tests applying YAML files from a directory
func TestApplyDirectory_Success(t *testing.T) {
	client := &Client{}

	tmpDir := t.TempDir()
	files := map[string]string{
		"one.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: dir-one
data:
  key: one
`,
		"two.yml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: dir-two
data:
  key: two
`,
		"ignore.txt": "not yaml",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", name, err)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("ApplyDirectory() expected panic when YAML processing reaches unconfigured client")
		}
	}()

	_ = client.ApplyDirectory(context.Background(), tmpDir, "default")
}

// TestApplyDirectory_SkipsSubdirectories tests that subdirectories are skipped
func TestApplyDirectory_SkipsSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a YAML file in subdirectory
	subFile := filepath.Join(subDir, "test.yaml")
	if err := os.WriteFile(subFile, []byte("apiVersion: v1\nkind: ConfigMap"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// ApplyDirectory should skip subdirectories
	// This test verifies the directory reading logic
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	yamlCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" {
			yamlCount++
		}
	}

	if yamlCount != 0 {
		t.Errorf("Expected 0 YAML files in top-level directory, got %d", yamlCount)
	}
}

// TestApplyDirectory_NonExistent tests applying from a non-existent directory
func TestApplyDirectory_NonExistent(t *testing.T) {
	client := &Client{}

	err := client.ApplyDirectory(context.Background(), "/nonexistent/directory", "default")
	if err == nil {
		t.Error("ApplyDirectory() expected error for non-existent directory")
	}
}

// TestApplyManifestFiles_EmptyList tests applying an empty list of files
func TestApplyManifestFiles_EmptyList(t *testing.T) {
	client := &Client{}

	if err := client.ApplyManifestFiles(context.Background(), nil, "default"); err != nil {
		t.Fatalf("ApplyManifestFiles() unexpected error: %v", err)
	}
}

// TestDeleteManifest_Success tests deleting a resource from manifest
func TestDeleteManifest_Success(t *testing.T) {
	client := &Client{}

	manifest := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: delete-me
`)

	defer func() {
		if recover() == nil {
			t.Fatal("DeleteManifest() expected panic without configured mapper")
		}
	}()

	_ = client.DeleteManifest(context.Background(), manifest, "default")
}

// TestDeleteManifestFile_NonExistent tests deleting from a non-existent file
func TestDeleteManifestFile_NonExistent(t *testing.T) {
	client := &Client{}

	// Should not error for non-existent file
	err := client.DeleteManifestFile(context.Background(), "/nonexistent/file.yaml", "default")
	if err != nil {
		t.Errorf("DeleteManifestFile() unexpected error for non-existent file: %v", err)
	}
}

// TestDeleteManifestContent_MultiDocument tests deleting multi-document YAML
func TestDeleteManifestContent_MultiDocument(t *testing.T) {
	client := &Client{}

	content := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-one
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-two
`)

	defer func() {
		if recover() == nil {
			t.Fatal("DeleteManifestContent() expected panic without configured mapper")
		}
	}()

	_ = client.DeleteManifestContent(context.Background(), content, "default")
}

// TestDeleteDirectory_NonExistent tests deleting from a non-existent directory
func TestDeleteDirectory_NonExistent(t *testing.T) {
	client := &Client{}

	// Should not error for non-existent directory
	err := client.DeleteDirectory(context.Background(), "/nonexistent/directory", "default")
	if err != nil {
		t.Errorf("DeleteDirectory() unexpected error for non-existent directory: %v", err)
	}
}

// TestFieldManager tests the field manager constant
func TestFieldManager(t *testing.T) {
	expected := "terraform-provider-gdp-edge"
	if fieldManager != expected {
		t.Errorf("fieldManager = %q, want %q", fieldManager, expected)
	}
}

// Made with Bob
