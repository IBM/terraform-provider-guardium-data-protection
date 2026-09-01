// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// TestBuildKubeconfigConfig_DefaultLocations tests default kubeconfig locations
func TestBuildKubeconfigConfig_DefaultLocations(t *testing.T) {
	// Test with empty path - should try default locations
	_, err := buildKubeconfigConfig("")
	// Expected to fail since we're not in a real environment
	if err == nil {
		t.Log("buildKubeconfigConfig() succeeded with default path")
	}
}

// TestBuildKubeconfigConfig_CustomPath tests custom kubeconfig path
func TestBuildKubeconfigConfig_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create a minimal kubeconfig file
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644); err != nil {
		t.Fatalf("Failed to create test kubeconfig: %v", err)
	}

	config, err := buildKubeconfigConfig(kubeconfigPath)
	if err != nil {
		t.Errorf("buildKubeconfigConfig() error = %v", err)
		return
	}

	if config.Host != "https://localhost:6443" {
		t.Errorf("config.Host = %q, want %q", config.Host, "https://localhost:6443")
	}
}

// TestBuildKubeconfigConfig_NonExistent tests non-existent kubeconfig
func TestBuildKubeconfigConfig_NonExistent(t *testing.T) {
	_, err := buildKubeconfigConfig("/nonexistent/kubeconfig")
	if err == nil {
		t.Error("buildKubeconfigConfig() expected error for non-existent file")
	}
}

// TestBuildOpenShiftConfig_WithToken tests OpenShift config with token
func TestBuildOpenShiftConfig_WithToken(t *testing.T) {
	cfg := AuthConfig{
		OCPServer:             "https://api.test.example.com:6443",
		OCPToken:              "test-token",
		OCPInsecureSkipVerify: true,
	}

	config, err := buildOpenShiftConfig(cfg)
	if err != nil {
		t.Errorf("buildOpenShiftConfig() error = %v", err)
		return
	}

	if config.Host != cfg.OCPServer {
		t.Errorf("config.Host = %q, want %q", config.Host, cfg.OCPServer)
	}

	if config.BearerToken != cfg.OCPToken {
		t.Errorf("config.BearerToken = %q, want %q", config.BearerToken, cfg.OCPToken)
	}

	if !config.Insecure {
		t.Error("config.TLSClientConfig.Insecure = false, want true")
	}
}

// TestExtractTokenFromRedirect tests extracting token from OAuth redirect
func TestExtractTokenFromRedirect(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantErr  bool
		want     string
	}{
		{
			name:     "valid token in fragment",
			location: "https://oauth.example.com/callback#access_token=test-token-123&token_type=Bearer&expires_in=3600",
			wantErr:  false,
			want:     "test-token-123",
		},
		{
			name:     "valid token in query",
			location: "https://oauth.example.com/callback?access_token=test-token-456&token_type=Bearer",
			wantErr:  false,
			want:     "test-token-456",
		},
		{
			name:     "missing token",
			location: "https://oauth.example.com/callback#token_type=Bearer",
			wantErr:  true,
		},
		{
			name:     "invalid URL",
			location: "://invalid-url",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractTokenFromRedirect(tt.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractTokenFromRedirect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractTokenFromRedirect() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAuthConfig_Conversion tests AuthConfig type conversion
func TestAuthConfig_Conversion(t *testing.T) {
	cfg := Config{
		KubeconfigPath:        "/path/to/kubeconfig",
		Platform:              "k3s",
		AWSRegion:             "us-west-2",
		AWSProfile:            "default",
		AWSAccessKey:          "AKIAIOSFODNN7EXAMPLE",
		AWSSecretKey:          "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		EKSClusterName:        "test-cluster",
		OCPServer:             "https://api.test.com:6443",
		OCPUsername:           "admin",
		OCPPassword:           "password",
		OCPToken:              "token",
		OCPInsecureSkipVerify: true,
	}

	authCfg := AuthConfig(cfg)

	if authCfg.KubeconfigPath != cfg.KubeconfigPath {
		t.Errorf("KubeconfigPath = %q, want %q", authCfg.KubeconfigPath, cfg.KubeconfigPath)
	}
	if authCfg.Platform != cfg.Platform {
		t.Errorf("Platform = %q, want %q", authCfg.Platform, cfg.Platform)
	}
	if authCfg.AWSRegion != cfg.AWSRegion {
		t.Errorf("AWSRegion = %q, want %q", authCfg.AWSRegion, cfg.AWSRegion)
	}
	if authCfg.EKSClusterName != cfg.EKSClusterName {
		t.Errorf("EKSClusterName = %q, want %q", authCfg.EKSClusterName, cfg.EKSClusterName)
	}
	if authCfg.OCPServer != cfg.OCPServer {
		t.Errorf("OCPServer = %q, want %q", authCfg.OCPServer, cfg.OCPServer)
	}
}

// TestExtractTokenFromRedirect_EdgeCases tests edge cases for token extraction
func TestExtractTokenFromRedirect_EdgeCases(t *testing.T) {
	// Test empty fragment and query
	_, err := extractTokenFromRedirect("https://example.com/callback")
	if err == nil {
		t.Error("extractTokenFromRedirect() expected error for URL without token")
	}

	// Test URL parsing
	validURL := "https://example.com#access_token=abc123"
	token, err := extractTokenFromRedirect(validURL)
	if err != nil {
		t.Errorf("extractTokenFromRedirect() unexpected error: %v", err)
	}
	if token != "abc123" {
		t.Errorf("extractTokenFromRedirect() = %q, want %q", token, "abc123")
	}
}

// TestURLParsing tests URL parsing utilities
func TestURLParsing(t *testing.T) {
	testURL := "https://example.com:6443/path?query=value#fragment=data"
	parsed, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if parsed.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", parsed.Scheme, "https")
	}
	if parsed.Host != "example.com:6443" {
		t.Errorf("Host = %q, want %q", parsed.Host, "example.com:6443")
	}
}

// Made with Bob
