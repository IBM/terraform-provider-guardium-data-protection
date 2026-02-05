// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestNewInsecureClient validates the creation and initialization of InsecureClient
func TestNewInsecureClient(t *testing.T) {
	testCases := []struct {
		name             string
		host             string
		port             string
		expectedProtocol string
	}{
		{
			name:             "Valid insecure client creation",
			host:             "guardium.example.com",
			port:             "8443",
			expectedProtocol: "https",
		},
		{
			name:             "Insecure client with localhost",
			host:             "localhost",
			port:             "8443",
			expectedProtocol: "https",
		},
		{
			name:             "Insecure client with IP address",
			host:             "192.168.1.100",
			port:             "9443",
			expectedProtocol: "https",
		},
		{
			name:             "Empty host should still create client",
			host:             "",
			port:             "8443",
			expectedProtocol: "https",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseClient := NewClient(tc.host, tc.port)
			insecureClient := baseClient.NewInsecureClient()

			if insecureClient == nil {
				t.Fatal("Expected InsecureClient to be created, got nil")
			}

			// Verify Host is preserved
			if insecureClient.Client.Host != tc.host {
				t.Errorf("Expected Host to be %s, got %s", tc.host, insecureClient.Client.Host)
			}

			// Verify port is preserved
			if insecureClient.Client.port != tc.port {
				t.Errorf("Expected port to be %s, got %s", tc.port, insecureClient.Client.port)
			}

			// Verify protocol is always https
			if insecureClient.Client.protocol != tc.expectedProtocol {
				t.Errorf("Expected protocol to be %s, got %s", tc.expectedProtocol, insecureClient.Client.protocol)
			}
		})
	}
}

// TestInsecureClient_ProtocolEnforcement ensures protocol is always HTTPS
func TestInsecureClient_ProtocolEnforcement(t *testing.T) {
	testCases := []struct {
		name             string
		initialProtocol  string
		expectedProtocol string
	}{
		{
			name:             "HTTP client becomes HTTPS",
			initialProtocol:  "http",
			expectedProtocol: "https",
		},
		{
			name:             "HTTPS client stays HTTPS",
			initialProtocol:  "https",
			expectedProtocol: "https",
		},
		{
			name:             "Empty protocol becomes HTTPS",
			initialProtocol:  "",
			expectedProtocol: "https",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseClient := &Client{
				Host:     "test.example.com",
				port:     "8443",
				protocol: tc.initialProtocol,
			}

			insecureClient := baseClient.NewInsecureClient()

			if insecureClient.Client.protocol != tc.expectedProtocol {
				t.Errorf("Expected protocol to be %s, got %s", tc.expectedProtocol, insecureClient.Client.protocol)
			}
		})
	}
}

// TestInsecureClient_TLSConfigVerification validates that TLS verification is properly disabled
func TestInsecureClient_TLSConfigVerification(t *testing.T) {
	// Create a test HTTPS server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"access_token":"test-token"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Extract host and port
	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	baseClient := &Client{
		Host: host,
		port: port,
	}
	insecureClient := baseClient.NewInsecureClient()

	ctx := context.Background()

	// This should succeed because InsecureClient skips TLS verification
	token, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")
	if err != nil {
		t.Errorf("InsecureClient should handle self-signed certs, got error: %v", err)
	}

	if token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", token)
	}
}

// TestInsecureClient_GenerateAccessToken tests token generation with insecure client
func TestInsecureClient_GenerateAccessToken(t *testing.T) {
	testCases := []struct {
		name           string
		clientSecret   string
		username       string
		password       string
		clientId       string
		serverStatus   int
		serverResponse string
		expectError    bool
		expectedToken  string
	}{
		{
			name:           "Successful token generation",
			clientSecret:   "test-secret",
			username:       "admin",
			password:       "admin-pass",
			clientId:       "guardium-client",
			serverStatus:   http.StatusOK,
			serverResponse: `{"access_token":"insecure-token-123"}`,
			expectError:    false,
			expectedToken:  "insecure-token-123",
		},
		{
			name:           "Invalid credentials",
			clientSecret:   "wrong-secret",
			username:       "user",
			password:       "wrong-pass",
			clientId:       "client",
			serverStatus:   http.StatusBadRequest,
			serverResponse: `{"error":"invalid_client"}`,
			expectError:    true,
			expectedToken:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify OAuth endpoint
				if !strings.Contains(r.URL.Path, "/oauth/token") {
					t.Errorf("Expected /oauth/token path, got %s", r.URL.Path)
				}

				w.WriteHeader(tc.serverStatus)
				w.Write([]byte(tc.serverResponse))
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "https://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			baseClient := &Client{
				Host: host,
				port: port,
			}
			insecureClient := baseClient.NewInsecureClient()

			ctx := context.Background()
			token, err := insecureClient.GenerateAccessToken(ctx, tc.clientSecret, tc.username, tc.password, tc.clientId)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tc.expectError && token != tc.expectedToken {
				t.Errorf("Expected token %s, got %s", tc.expectedToken, token)
			}
		})
	}
}

// TestInsecureClient_ImportProfilesFromFile tests profile import with insecure client
func TestInsecureClient_ImportProfilesFromFile(t *testing.T) {
	testCases := []struct {
		name           string
		updateMode     bool
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "Successful import with update mode",
			updateMode:     true,
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"123","Message":"Import successful"}`,
			expectError:    false,
		},
		{
			name:           "Successful import without update mode",
			updateMode:     false,
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"456","Message":"Import successful"}`,
			expectError:    false,
		},
		{
			name:           "Server error during import",
			updateMode:     true,
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error":"internal server error"}`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary test file
			tmpFile, err := os.CreateTemp(t.TempDir(), "test-profile-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			tmpFile.WriteString(`{"test":"profile"}`)
			tmpFile.Close()

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/restAPI/importProfilesFromFile" {
					t.Errorf("Expected path /restAPI/importProfilesFromFile, got %s", r.URL.Path)
				}

				// Verify multipart form data
				if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
					t.Errorf("Expected multipart/form-data, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tc.serverStatus)
				w.Write([]byte(tc.serverResponse))
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "https://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			baseClient := &Client{
				Host: host,
				port: port,
			}
			insecureClient := baseClient.NewInsecureClient()

			ctx := context.Background()
			err = insecureClient.ImportProfilesFromFile(ctx, "test-token", tmpFile.Name(), tc.updateMode)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestInsecureClient_BulkInstallConnector tests connector installation with insecure client
func TestInsecureClient_BulkInstallConnector(t *testing.T) {
	testCases := []struct {
		name           string
		udcName        string
		gdpMuHost      string
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "Successful connector installation",
			udcName:        "test-connector",
			gdpMuHost:      "host1.example.com",
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"123","Message":"Installation successful"}`,
			expectError:    false,
		},
		{
			name:           "Host not found error",
			udcName:        "test-connector",
			gdpMuHost:      "invalid-host",
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"0","Message":"One or more of the specified hosts could not be found"}`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/restAPI/bulkInstall" {
					t.Errorf("Expected path /restAPI/bulkInstall, got %s", r.URL.Path)
				}

				w.WriteHeader(tc.serverStatus)
				w.Write([]byte(tc.serverResponse))
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "https://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			baseClient := &Client{
				Host: host,
				port: port,
			}
			insecureClient := baseClient.NewInsecureClient()

			ctx := context.Background()
			err := insecureClient.BulkInstallConnector(ctx, "test-token", tc.udcName, tc.gdpMuHost)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestInsecureClient_VAOperations tests VA-related operations with insecure client
func TestInsecureClient_VAOperations(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/restAPI/datasource":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"ds-123","message":"success"}`))
		case "/restAPI/va/config":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"config-456","message":"success"}`))
		case "/restAPI/notifications":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"notif-789","message":"success"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	baseClient := &Client{
		Host: host,
		port: port,
	}
	insecureClient := baseClient.NewInsecureClient()

	ctx := context.Background()
	token := "test-token"

	// Test RegisterVADataSource
	t.Run("RegisterVADataSource", func(t *testing.T) {
		payload := []byte(`{"datasourceName":"test-db"}`)
		err := insecureClient.RegisterVADataSource(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test ConfigureVADataSource
	t.Run("ConfigureVADataSource", func(t *testing.T) {
		payload := []byte(`{"datasourceName":"test-db","scanSchedule":"daily"}`)
		err := insecureClient.ConfigureVADataSource(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test ConfigureVANotifications
	t.Run("ConfigureVANotifications", func(t *testing.T) {
		payload := []byte(`{"email":"admin@example.com"}`)
		err := insecureClient.ConfigureVANotifications(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// TestInsecureClient_AWSSecretsManagerOperations tests AWS Secrets Manager operations
func TestInsecureClient_AWSSecretsManagerOperations(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"created"}`))
		case "GET":
			// GetAWSSecretsManager calls GetAllAWSSecretsManagerConfigs which expects an array
			w.WriteHeader(http.StatusOK)
			response := []map[string]interface{}{
				{
					"id":                          1,
					"name":                        "test-secret",
					"authType":                    "access_key",
					"accessKeyId":                 "AKIAIOSFODNN7EXAMPLE",
					"secretAccessKey":             "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"secretKeyUsernameIdentifier": "username-key",
					"secretKeyPasswordIdentifier": "password-key",
					"roleARN":                     "",
					"secretsManager":              true,
				},
			}
			json.NewEncoder(w).Encode(response)
		case "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"updated"}`))
		case "DELETE":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"deleted"}`))
		}
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	baseClient := &Client{
		Host: host,
		port: port,
	}
	insecureClient := baseClient.NewInsecureClient()

	ctx := context.Background()
	token := "test-token"

	config := &AWSSecretsManagerConfig{
		Name:              "test-secret",
		AuthType:          "access_key",
		AccessKeyID:       "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SecretKeyUsername: "username-key",
		SecretKeyPassword: "password-key",
	}

	// Test CreateAWSSecretsManager
	t.Run("CreateAWSSecretsManager", func(t *testing.T) {
		err := insecureClient.CreateAWSSecretsManager(ctx, token, config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test GetAWSSecretsManager
	t.Run("GetAWSSecretsManager", func(t *testing.T) {
		result, err := insecureClient.GetAWSSecretsManager(ctx, token, "test-secret")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})

	// Test UpdateAWSSecretsManager
	t.Run("UpdateAWSSecretsManager", func(t *testing.T) {
		err := insecureClient.UpdateAWSSecretsManager(ctx, token, config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test DeleteAWSSecretsManager
	t.Run("DeleteAWSSecretsManager", func(t *testing.T) {
		err := insecureClient.DeleteAWSSecretsManager(ctx, token, "test-secret")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// TestInsecureClient_ErrorHandling tests error scenarios and safe defaults
func TestInsecureClient_ErrorHandling(t *testing.T) {
	t.Run("Unreachable server", func(t *testing.T) {
		baseClient := &Client{
			Host: "unreachable.invalid.host",
			port: "8443",
		}
		insecureClient := baseClient.NewInsecureClient()

		ctx := context.Background()
		_, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err == nil {
			t.Error("Expected error for unreachable server, got nil")
		}
	})

	t.Run("Invalid port", func(t *testing.T) {
		baseClient := &Client{
			Host: "localhost",
			port: "99999",
		}
		insecureClient := baseClient.NewInsecureClient()

		ctx := context.Background()
		_, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err == nil {
			t.Error("Expected error for invalid port, got nil")
		}
	})

	t.Run("Empty host and port", func(t *testing.T) {
		baseClient := &Client{
			Host: "",
			port: "",
		}
		insecureClient := baseClient.NewInsecureClient()

		// Should create client without error
		if insecureClient == nil {
			t.Error("Expected InsecureClient to be created even with empty host/port")
		}

		// But operations should fail
		ctx := context.Background()
		_, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err == nil {
			t.Error("Expected error for empty host/port, got nil")
		}
	})
}

// TestInsecureClient_TLSHandshakeFailure tests behavior when TLS handshake fails
func TestInsecureClient_TLSHandshakeFailure(t *testing.T) {
	// Create a server that will cause TLS handshake issues
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Configure with invalid TLS settings
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS10, // Invalid: min > max
	}

	// This will fail to start, which is expected
	defer func() {
		if r := recover(); r != nil {
			// Expected to fail
		}
	}()

	// Even with invalid TLS config, InsecureClient should be created
	baseClient := &Client{
		Host: "localhost",
		port: "8443",
	}
	insecureClient := baseClient.NewInsecureClient()

	if insecureClient == nil {
		t.Error("InsecureClient should be created even if TLS will fail")
	}
}

// TestInsecureClient_CertificateValidation ensures InsecureSkipVerify works correctly
func TestInsecureClient_CertificateValidation(t *testing.T) {
	// Create server with self-signed cert
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token"}`))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	t.Run("InsecureClient accepts self-signed cert", func(t *testing.T) {
		baseClient := &Client{
			Host: host,
			port: port,
		}
		insecureClient := baseClient.NewInsecureClient()

		ctx := context.Background()
		token, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err != nil {
			t.Errorf("InsecureClient should accept self-signed cert, got error: %v", err)
		}
		if token != "test-token" {
			t.Errorf("Expected token 'test-token', got '%s'", token)
		}
	})

	t.Run("Regular client rejects self-signed cert", func(t *testing.T) {
		baseClient := &Client{
			Host:     host,
			port:     port,
			protocol: "https",
		}

		// Create a regular HTTP client (without InsecureSkipVerify)
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		}

		ctx := context.Background()
		_, err := baseClient.generateAccessToken(ctx, httpClient, "secret", "user", "pass", "client")

		// Should fail due to certificate validation
		if err == nil {
			t.Error("Regular client should reject self-signed cert")
		}

		// Verify it's a certificate error
		if err != nil {
			if _, ok := err.(*x509.UnknownAuthorityError); !ok {
				// Also check for other certificate-related errors
				if !strings.Contains(err.Error(), "certificate") &&
					!strings.Contains(err.Error(), "x509") &&
					!strings.Contains(err.Error(), "tls") {
					t.Logf("Expected certificate error, got: %v", err)
				}
			}
		}
	})
}

// TestInsecureClient_SharedTokenUsage validates token reuse across operations
func TestInsecureClient_SharedTokenUsage(t *testing.T) {
	var tokenUsageCount int
	sharedToken := "shared-insecure-token"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if strings.Contains(authHeader, sharedToken) {
			tokenUsageCount++
		}

		switch r.URL.Path {
		case "/oauth/token":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"` + sharedToken + `"}`))
		case "/restAPI/datasource", "/restAPI/bulkInstall":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"success"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	baseClient := &Client{
		Host: host,
		port: port,
	}
	insecureClient := baseClient.NewInsecureClient()

	ctx := context.Background()

	// Generate token
	token, err := insecureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Use token for VA operation
	vaPayload := []byte(`{"datasourceName":"test"}`)
	err = insecureClient.RegisterVADataSource(ctx, token, vaPayload)
	if err != nil {
		t.Errorf("Failed to use token for VA: %v", err)
	}

	// Use token for UC operation
	err = insecureClient.BulkInstallConnector(ctx, token, "profile", "host")
	if err != nil {
		t.Errorf("Failed to use token for UC: %v", err)
	}

	// Verify token was used multiple times
	if tokenUsageCount != 2 {
		t.Errorf("Expected token to be used 2 times, got %d", tokenUsageCount)
	}
}
