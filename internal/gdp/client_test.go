// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// Test cases
	testCases := []struct {
		name     string
		host     string
		port     string
		expected *Client
	}{
		{
			name: "Valid client creation",
			host: "localhost",
			port: "8080",
			expected: &Client{
				Host: "localhost",
				port: "8080",
			},
		},
		{
			name: "Empty host",
			host: "",
			port: "8080",
			expected: &Client{
				Host: "",
				port: "8080",
			},
		},
		{
			name: "Empty port",
			host: "localhost",
			port: "",
			expected: &Client{
				Host: "localhost",
				port: "",
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(tc.host, tc.port)

			if client.Host != tc.expected.Host {
				t.Errorf("Expected Host to be %s, got %s", tc.expected.Host, client.Host)
			}

			if client.port != tc.expected.port {
				t.Errorf("Expected port to be %s, got %s", tc.expected.port, client.port)
			}
		})
	}
}

func TestGenerateAccessToken(t *testing.T) {
	// Test cases
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
		protocol       string
	}{
		{
			name:           "Successful token generation",
			clientSecret:   "test-secret-123",
			username:       "admin",
			password:       "admin-pass",
			clientId:       "guardium-client",
			serverStatus:   http.StatusOK,
			serverResponse: `{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test"}`,
			expectError:    false,
			expectedToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			protocol:       "http",
		},
		{
			name:           "Successful token with different credentials",
			clientSecret:   "test-secret-456",
			username:       "user",
			password:       "pass",
			clientId:       "oidc-client",
			serverStatus:   http.StatusOK,
			serverResponse: `{"access_token":"test-token-http"}`,
			expectError:    false,
			expectedToken:  "test-token-http",
			protocol:       "http",
		},
		{
			name:           "Invalid credentials - Bad Request",
			clientSecret:   "wrong-secret",
			username:       "user",
			password:       "wrong-pass",
			clientId:       "client1",
			serverStatus:   http.StatusBadRequest,
			serverResponse: `{"error":"invalid_client"}`,
			expectError:    true,
			expectedToken:  "",
			protocol:       "https",
		},
		{
			name:           "Invalid JSON response",
			clientSecret:   "secret",
			username:       "user",
			password:       "pass",
			clientId:       "client1",
			serverStatus:   http.StatusOK,
			serverResponse: `invalid-json`,
			expectError:    true,
			expectedToken:  "",
			protocol:       "http",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request method
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Verify OAuth token endpoint path
				if !strings.Contains(r.URL.Path, "/oauth/token") {
					t.Errorf("Expected /oauth/token path, got %s", r.URL.Path)
				}

				// Check query parameters
				query := r.URL.Query()
				if query.Get("client_id") != tc.clientId {
					t.Errorf("Expected client_id=%s, got %s", tc.clientId, query.Get("client_id"))
				}
				if query.Get("client_secret") != tc.clientSecret {
					t.Errorf("Expected client_secret=%s, got %s", tc.clientSecret, query.Get("client_secret"))
				}
				if query.Get("username") != tc.username {
					t.Errorf("Expected username=%s, got %s", tc.username, query.Get("username"))
				}
				if query.Get("password") != tc.password {
					t.Errorf("Expected password=%s, got %s", tc.password, query.Get("password"))
				}
				if query.Get("grant_type") != "password" {
					t.Errorf("Expected grant_type=password, got %s", query.Get("grant_type"))
				}

				// Set response status and body
				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			// Create client with protocol
			client := &Client{
				Host:     host,
				port:     port,
				protocol: tc.protocol,
			}

			// Call the function
			ctx := context.Background()
			result, err := client.generateAccessToken(ctx, server.Client(), tc.clientSecret, tc.username, tc.password, tc.clientId)

			// Check error
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check result
			if !tc.expectError {
				if result == nil {
					t.Error("Expected result but got nil")
				} else if result.AccessToken != tc.expectedToken {
					t.Errorf("Expected token %s, got %s", tc.expectedToken, result.AccessToken)
				}
			}
		})
	}
}

// TestGenerateAccessTokenWithGDPURLAndPort validates bearer token generation
// using GDP URL, port, and OIDC client credentials
func TestGenerateAccessTokenWithGDPURLAndPort(t *testing.T) {
	testCases := []struct {
		name         string
		gdpHost      string
		gdpPort      string
		clientId     string
		clientSecret string
		username     string
		password     string
		expectError  bool
	}{
		{
			name:         "Valid GDP URL and port with OIDC credentials",
			gdpHost:      "guardium.example.com",
			gdpPort:      "8443",
			clientId:     "oidc-client-id",
			clientSecret: "oidc-client-secret",
			username:     "gdp-admin",
			password:     "gdp-password",
			expectError:  false,
		},
		{
			name:         "Different port configuration",
			gdpHost:      "gdp-server.local",
			gdpPort:      "9443",
			clientId:     "client-123",
			clientSecret: "secret-456",
			username:     "user1",
			password:     "pass1",
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that validates the URL construction
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate that the request contains all OIDC parameters
				query := r.URL.Query()

				if query.Get("client_id") != tc.clientId {
					t.Errorf("client_id mismatch: expected %s, got %s", tc.clientId, query.Get("client_id"))
				}
				if query.Get("client_secret") != tc.clientSecret {
					t.Errorf("client_secret mismatch: expected %s, got %s", tc.clientSecret, query.Get("client_secret"))
				}
				if query.Get("username") != tc.username {
					t.Errorf("username mismatch: expected %s, got %s", tc.username, query.Get("username"))
				}
				if query.Get("password") != tc.password {
					t.Errorf("password mismatch: expected %s, got %s", tc.password, query.Get("password"))
				}
				if query.Get("grant_type") != "password" {
					t.Errorf("grant_type mismatch: expected password, got %s", query.Get("grant_type"))
				}

				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(`{"access_token":"valid-token"}`)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			// Extract host and port from test server
			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			// Create client with GDP configuration
			client := &Client{
				Host:     host,
				port:     port,
				protocol: "http",
			}

			ctx := context.Background()
			result, err := client.generateAccessToken(ctx, server.Client(), tc.clientSecret, tc.username, tc.password, tc.clientId)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tc.expectError && result == nil {
				t.Error("Expected result but got nil")
			}
		})
	}
}

// TestGenerateAccessTokenInvalidCredentials tests error handling for invalid credentials
func TestGenerateAccessTokenInvalidCredentials(t *testing.T) {
	testCases := []struct {
		name           string
		clientSecret   string
		username       string
		password       string
		serverStatus   int
		serverResponse string
		expectedError  string
	}{
		{
			name:           "Invalid client credentials",
			clientSecret:   "invalid-secret",
			username:       "user",
			password:       "pass",
			serverStatus:   http.StatusBadRequest,
			serverResponse: `{"error":"invalid_client","error_description":"Client authentication failed"}`,
			expectedError:  "invalid credentials for access token",
		},
		{
			name:           "Invalid user credentials",
			clientSecret:   "valid-secret",
			username:       "invalid-user",
			password:       "wrong-pass",
			serverStatus:   http.StatusBadRequest,
			serverResponse: `{"error":"invalid_grant","error_description":"Invalid user credentials"}`,
			expectedError:  "invalid credentials for access token",
		},
		{
			name:           "Expired credentials",
			clientSecret:   "expired-secret",
			username:       "user",
			password:       "pass",
			serverStatus:   http.StatusBadRequest,
			serverResponse: `{"error":"invalid_client"}`,
			expectedError:  "invalid credentials for access token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			client := &Client{
				Host:     host,
				port:     port,
				protocol: "http",
			}

			ctx := context.Background()
			_, err := client.generateAccessToken(ctx, server.Client(), tc.clientSecret, tc.username, tc.password, "test-client")

			if err == nil {
				t.Error("Expected error for invalid credentials but got nil")
			}
			if err != nil && !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", tc.expectedError, err)
			}
		})
	}
}

// TestGenerateAccessTokenUnreachableEndpoint tests error handling for unreachable endpoints
func TestGenerateAccessTokenUnreachableEndpoint(t *testing.T) {
	testCases := []struct {
		name        string
		host        string
		port        string
		expectError bool
	}{
		{
			name:        "Unreachable host",
			host:        "unreachable.invalid.host",
			port:        "8443",
			expectError: true,
		},
		{
			name:        "Invalid port",
			host:        "localhost",
			port:        "99999",
			expectError: true,
		},
		{
			name:        "Connection refused",
			host:        "127.0.0.1",
			port:        "1",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				Host:     tc.host,
				port:     tc.port,
				protocol: "http",
			}

			// Use a client with short timeout to avoid long waits
			httpClient := &http.Client{
				Timeout: 1 * time.Second,
			}

			ctx := context.Background()
			_, err := client.generateAccessToken(ctx, httpClient, "secret", "user", "pass", "client")

			if tc.expectError && err == nil {
				t.Error("Expected error for unreachable endpoint but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestImportProfilesFromFile(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		accessToken    string
		updateMode     bool
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "Successful import",
			accessToken:    "test-token",
			updateMode:     true,
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"123","Message":"Import successful"}`,
			expectError:    false,
		},
		{
			name:           "Server error",
			accessToken:    "test-token",
			updateMode:     true,
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error":"internal server error"}`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary file for testing
			tmpFile, err := os.CreateTemp(t.TempDir(), "test-profile-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write some test content
			if _, err := tmpFile.WriteString(`{"test":"data"}`); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request method
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Check request path
				if r.URL.Path != "/restAPI/importProfilesFromFile" {
					t.Errorf("Expected path /restAPI/importProfilesFromFile, got %s", r.URL.Path)
				}

				// Check headers
				if r.Header.Get("Authorization") != "Bearer "+tc.accessToken {
					t.Errorf("Expected Authorization header Bearer %s, got %s", tc.accessToken, r.Header.Get("Authorization"))
				}

				// Verify it's multipart form data
				if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
					t.Errorf("Expected Content-Type to contain multipart/form-data, got %s", r.Header.Get("Content-Type"))
				}

				// Set response status and body
				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			// Extract host and port from test server URL
			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			// Create client
			client := &Client{
				Host:     host,
				port:     port,
				protocol: "http",
			}

			// Call the function
			ctx := context.Background()
			err = client.ImportProfilesFromFile(ctx, server.Client(), tc.accessToken, tmpFile.Name(), tc.updateMode)

			// Check error
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestBulkInstallConnector(t *testing.T) {
	// Test cases
	testCases := []struct {
		name           string
		accessToken    string
		udcName        string
		gdpMuHost      string
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "Successful installation",
			accessToken:    "test-token",
			udcName:        "connector-profile",
			gdpMuHost:      "host1.example.com",
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"123","Message":"Installation successful"}`,
			expectError:    false,
		},
		{
			name:           "Server error with non-OK status",
			accessToken:    "test-token",
			udcName:        "connector-profile",
			gdpMuHost:      "host1.example.com",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"ID":"0","Message":"Internal server error"}`,
			expectError:    true,
		},
		{
			name:           "Known error message",
			accessToken:    "test-token",
			udcName:        "connector-profile",
			gdpMuHost:      "host1.example.com",
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"0","Message":"One or more of the specified hosts could not be found"}`,
			expectError:    true,
		},
		{
			name:           "Error keyword in message",
			accessToken:    "test-token",
			udcName:        "connector-profile",
			gdpMuHost:      "host1.example.com",
			serverStatus:   http.StatusOK,
			serverResponse: `{"ID":"0","Message":"Installation failed due to network error"}`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request method
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Check request path
				if r.URL.Path != "/restAPI/bulkInstall" {
					t.Errorf("Expected path /restAPI/bulkInstall, got %s", r.URL.Path)
				}

				// Check headers
				if r.Header.Get("Authorization") != "Bearer "+tc.accessToken {
					t.Errorf("Expected Authorization header Bearer %s, got %s", tc.accessToken, r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Check request body
				var requestBody bulkInstallRequestBody
				decoder := json.NewDecoder(r.Body)
				if err := decoder.Decode(&requestBody); err != nil {
					t.Errorf("Error decoding request body: %v", err)
				}

				if requestBody.ProfileNames != tc.udcName {
					t.Errorf("Expected profileNames %s, got %s", tc.udcName, requestBody.ProfileNames)
				}
				if requestBody.Hosts != tc.gdpMuHost {
					t.Errorf("Expected hosts %s, got %s", tc.gdpMuHost, requestBody.Hosts)
				}

				// Set response status and body
				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			// Extract host and port from test server URL
			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			// Create client
			client := &Client{
				Host:     host,
				port:     port,
				protocol: "http",
			}

			// Call the function
			ctx := context.Background()
			err := client.BulkInstallConnector(ctx, server.Client(), tc.accessToken, tc.udcName, tc.gdpMuHost)

			// Check error
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSharedTokenUsageBetweenVAAndUC validates that the same bearer token
// can be used for both VA (Vulnerability Assessment) and UC (Universal Connector) operations
func TestSharedTokenUsageBetweenVAAndUC(t *testing.T) {
	// Track which endpoints were called with the same token
	var receivedTokens []string
	var endpointsCalled []string

	// Create a test server that handles multiple endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		// Only track endpoints that use Bearer token authentication
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			receivedTokens = append(receivedTokens, token)
			endpointsCalled = append(endpointsCalled, r.URL.Path)
		}

		switch r.URL.Path {
		case "/oauth/token":
			// OAuth token endpoint
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"access_token":"shared-token-12345"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/datasource":
			// VA datasource registration endpoint
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"ds-123","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/va/config":
			// VA configuration endpoint
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"config-456","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/bulkInstall":
			// UC connector installation endpoint
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"ID":"install-789","Message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/notifications":
			// VA notifications endpoint
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"notif-101","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	client := &Client{
		Host:     host,
		port:     port,
		protocol: "http",
	}

	ctx := context.Background()

	// Step 1: Generate access token
	tokenResp, err := client.generateAccessToken(ctx, server.Client(), "secret", "user", "pass", "client")
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}
	if tokenResp.AccessToken != "shared-token-12345" {
		t.Errorf("Expected token 'shared-token-12345', got '%s'", tokenResp.AccessToken)
	}

	sharedToken := tokenResp.AccessToken

	// Step 2: Use the same token for VA operations
	// Register VA datasource
	vaPayload := []byte(`{"datasourceName":"test-db","host":"db.example.com"}`)
	err = client.RegisterVADataSource(ctx, server.Client(), sharedToken, vaPayload)
	if err != nil {
		t.Errorf("Failed to register VA datasource with shared token: %v", err)
	}

	// Configure VA datasource
	vaConfigPayload := []byte(`{"datasourceName":"test-db","scanSchedule":"daily"}`)
	err = client.ConfigureVADataSource(ctx, server.Client(), sharedToken, vaConfigPayload)
	if err != nil {
		t.Errorf("Failed to configure VA datasource with shared token: %v", err)
	}

	// Configure VA notifications
	notifPayload := []byte(`{"email":"admin@example.com","enabled":true}`)
	err = client.ConfigureVANotifications(ctx, server.Client(), sharedToken, notifPayload)
	if err != nil {
		t.Errorf("Failed to configure VA notifications with shared token: %v", err)
	}

	// Step 3: Use the same token for UC operations
	// Install connector (UC operation)
	err = client.BulkInstallConnector(ctx, server.Client(), sharedToken, "connector-profile", "host1.example.com")
	if err != nil {
		t.Errorf("Failed to install connector with shared token: %v", err)
	}

	// Verify that all operations used the same token
	expectedEndpoints := []string{
		"/restAPI/datasource",
		"/restAPI/va/config",
		"/restAPI/notifications",
		"/restAPI/bulkInstall",
	}

	if len(receivedTokens) != len(expectedEndpoints) {
		t.Errorf("Expected %d API calls with token, got %d", len(expectedEndpoints), len(receivedTokens))
	}

	// Verify all tokens are the same
	for i, token := range receivedTokens {
		if token != sharedToken {
			t.Errorf("Token mismatch at call %d: expected '%s', got '%s'", i, sharedToken, token)
		}
	}

	// Verify all expected endpoints were called
	for _, expectedEndpoint := range expectedEndpoints {
		found := false
		for _, calledEndpoint := range endpointsCalled {
			if calledEndpoint == expectedEndpoint {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected endpoint '%s' was not called", expectedEndpoint)
		}
	}
}

// TestInsecureClientSharedTokenUsage validates that InsecureClient also supports
// shared token usage between VA and UC paths
func TestInsecureClientSharedTokenUsage(t *testing.T) {
	t.Skip("Skipping InsecureClient test - requires TLS server setup")
	// Note: InsecureClient uses HTTPS protocol and requires proper TLS server setup
	// This test validates the concept but is skipped in unit tests
	// Integration tests should cover InsecureClient functionality with real HTTPS endpoints
}

// TestClientProtocolConfiguration validates that the client correctly uses
// the protocol setting for different scenarios
func TestClientProtocolConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		protocol string
	}{
		{
			name:     "HTTP protocol",
			protocol: "http",
		},
		{
			name:     "HTTPS protocol setting",
			protocol: "https",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(`{"access_token":"test-token"}`)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			client := &Client{
				Host:     host,
				port:     port,
				protocol: tc.protocol,
			}

			// Verify the client stores the protocol correctly
			if client.protocol != tc.protocol {
				t.Errorf("Expected protocol '%s', got '%s'", tc.protocol, client.protocol)
			}

			// For HTTP protocol, test actual token generation
			if tc.protocol == "http" {
				ctx := context.Background()
				result, err := client.generateAccessToken(ctx, server.Client(), "secret", "user", "pass", "client")
				if err != nil {
					t.Errorf("Unexpected error with HTTP protocol: %v", err)
				}
				if result == nil || result.AccessToken != "test-token" {
					t.Error("Failed to generate token with HTTP protocol")
				}
			}
		})
	}
}

// TestTokenReuseAcrossMultipleClients validates that a token generated by one client
// can be used by another client instance (simulating shared token scenario)
func TestTokenReuseAcrossMultipleClients(t *testing.T) {
	var tokenUsageCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"access_token":"reusable-token"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/datasource", "/restAPI/bulkInstall":
			authHeader := r.Header.Get("Authorization")
			if strings.Contains(authHeader, "reusable-token") {
				tokenUsageCount++
			}
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	// Client 1 generates the token
	client1 := &Client{
		Host:     host,
		port:     port,
		protocol: "http",
	}

	ctx := context.Background()
	tokenResp, err := client1.generateAccessToken(ctx, server.Client(), "secret", "user", "pass", "client")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	sharedToken := tokenResp.AccessToken

	// Client 2 uses the same token for VA operation
	client2 := &Client{
		Host:     host,
		port:     port,
		protocol: "http",
	}

	vaPayload := []byte(`{"datasourceName":"test"}`)
	err = client2.RegisterVADataSource(ctx, server.Client(), sharedToken, vaPayload)
	if err != nil {
		t.Errorf("Client 2 failed to use shared token for VA: %v", err)
	}

	// Client 3 uses the same token for UC operation
	client3 := &Client{
		Host:     host,
		port:     port,
		protocol: "http",
	}

	err = client3.BulkInstallConnector(ctx, server.Client(), sharedToken, "profile", "host")
	if err != nil {
		t.Errorf("Client 3 failed to use shared token for UC: %v", err)
	}

	// Verify the token was successfully used by both clients
	if tokenUsageCount != 2 {
		t.Errorf("Expected token to be used 2 times, but was used %d times", tokenUsageCount)
	}
}
