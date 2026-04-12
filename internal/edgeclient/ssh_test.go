// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package edgeclient

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewSSHClient tests SSH client creation
func TestNewSSHClient(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		password    string
		keyPath     string
		expectError bool
		description string
	}{
		{
			name:        "with_password",
			user:        "testuser",
			password:    "testpass",
			keyPath:     "",
			expectError: false,
			description: "Should create client with password auth",
		},
		{
			name:        "with_nonexistent_key",
			user:        "testuser",
			password:    "",
			keyPath:     "/nonexistent/key",
			expectError: true,
			description: "Should fail with nonexistent key file",
		},
		{
			name:        "without_auth",
			user:        "testuser",
			password:    "",
			keyPath:     "",
			expectError: true,
			description: "Should fail without authentication method",
		},
		{
			name:        "with_password_and_key",
			user:        "testuser",
			password:    "testpass",
			keyPath:     "/nonexistent/key",
			expectError: true,
			description: "Should fail if key file doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewSSHClient(tt.user, tt.password, tt.keyPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if client != nil {
					t.Error("Client should be nil on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Client should not be nil")
				}
				if client != nil && client.user != tt.user {
					t.Errorf("user = %v, want %v", client.user, tt.user)
				}
			}
		})
	}
}

// TestNewSSHClient_WithValidKey tests SSH client with a valid key file
func TestNewSSHClient_WithValidKey(t *testing.T) {
	// Create a temporary SSH key file (mock)
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	// Create a mock private key (not a real key, just for file existence test)
	mockKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
-----END OPENSSH PRIVATE KEY-----`

	if err := os.WriteFile(keyPath, []byte(mockKey), 0600); err != nil {
		t.Fatal(err)
	}

	// This will fail because it's not a valid key, but tests the file reading
	_, err := NewSSHClient("testuser", "", keyPath)
	if err == nil {
		t.Error("Expected error with invalid key format")
	}
}

// TestSSHClient_Run tests command execution fails without a server
func TestSSHClient_Run(t *testing.T) {
	client, err := NewSSHClient("testuser", "testpass", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Run("127.0.0.1:19999", "echo test")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestSSHClient_CopyTo tests file upload fails without a server
func TestSSHClient_CopyTo(t *testing.T) {
	client, err := NewSSHClient("testuser", "testpass", "")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(localFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	err = client.CopyTo("127.0.0.1:19999", localFile, "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestSSHClient_CopyFrom tests file download fails without a server
func TestSSHClient_CopyFrom(t *testing.T) {
	client, err := NewSSHClient("testuser", "testpass", "")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "downloaded.txt")

	err = client.CopyFrom("127.0.0.1:19999", "/tmp/test.txt", localFile)
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestSSHClient_RunWithSudo tests sudo command execution fails without a server
func TestSSHClient_RunWithSudo(t *testing.T) {
	client, err := NewSSHClient("testuser", "testpass", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.RunWithSudo("127.0.0.1:19999", "ls /root")
	if err == nil {
		t.Error("Expected error without SSH server")
	}
}

// TestSSHClient_HostPortHandling tests host:port parsing on connection failure
func TestSSHClient_HostPortHandling(t *testing.T) {
	client, err := NewSSHClient("testuser", "testpass", "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		host string
	}{
		{"hostname_only", "127.0.0.1"},
		{"hostname_with_port", "127.0.0.1:19999"},
		{"ip_only", "127.0.0.2"},
		{"ip_with_port", "127.0.0.2:19999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Run(tt.host, "echo test")
			if err == nil {
				t.Error("Expected error without SSH server")
			}
		})
	}
}

// Made with Bob
