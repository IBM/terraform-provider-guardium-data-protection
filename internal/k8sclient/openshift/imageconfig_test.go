// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"strings"
	"testing"
)

// TestPatchImageConfig_ConnectionError tests that PatchImageConfig wraps errors
func TestPatchImageConfig_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	err := client.PatchImageConfig(context.Background(), "registry-ca")
	if err == nil {
		t.Fatal("PatchImageConfig() expected error without accessible server")
	}
	if !strings.Contains(err.Error(), "failed to patch image config") {
		t.Fatalf("PatchImageConfig() error = %v, want 'failed to patch image config'", err)
	}
}

// TestGetImageConfig_ConnectionError tests that GetImageConfig wraps errors
func TestGetImageConfig_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	name, err := client.GetImageConfig(context.Background())
	if err == nil {
		t.Fatal("GetImageConfig() expected error without accessible server")
	}
	if name != "" {
		t.Fatalf("GetImageConfig() name = %q, want empty string on error", name)
	}
	if !strings.Contains(err.Error(), "failed to get image config") {
		t.Fatalf("GetImageConfig() error = %v, want 'failed to get image config'", err)
	}
}

// Made with Bob
