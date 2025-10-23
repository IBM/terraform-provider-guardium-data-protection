package gdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
		serverStatus   int
		serverResponse string
		expectError    bool
		expectedToken  string
	}{
		{
			name:           "Successful token generation",
			clientSecret:   "secret",
			username:       "user",
			password:       "pass",
			serverStatus:   http.StatusOK,
			serverResponse: `{"access_token":"test-token"}`,
			expectError:    false,
			expectedToken:  "test-token",
		},
		{
			name:           "Server error",
			clientSecret:   "secret",
			username:       "user",
			password:       "pass",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error":"internal server error"}`,
			expectError:    true,
			expectedToken:  "",
		},
		{
			name:           "Invalid JSON response",
			clientSecret:   "secret",
			username:       "user",
			password:       "pass",
			serverStatus:   http.StatusOK,
			serverResponse: `invalid-json`,
			expectError:    true,
			expectedToken:  "",
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

				// Check query parameters
				query := r.URL.Query()
				if query.Get("client_id") != "client1" {
					t.Errorf("Expected client_id=client1, got %s", query.Get("client_id"))
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

				// Set response status and body
				w.WriteHeader(tc.serverStatus)
				_, err := w.Write([]byte(tc.serverResponse))

				// Check error
				if err == nil {
					t.Error("Expected error but got nil")
				}
			}))
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "http://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			// Create client
			client := &Client{
				Host: host,
				port: port,
			}

			// Call the function
			ctx := context.Background()
			result, err := client.generateAccessToken(ctx, server.Client(), tc.clientSecret, tc.username, tc.password, "test_client_id")

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

func TestImportProfilesFromFile(t *testing.T) {
	// Test cases
	testCases := []struct {
		name         string
		accessToken  string
		pathToFile   string
		updateMode   bool
		serverStatus int
		expectError  bool
	}{
		{
			name:         "Successful import",
			accessToken:  "test-token",
			pathToFile:   "/path/to/file.json",
			updateMode:   true,
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:         "Server error",
			accessToken:  "test-token",
			pathToFile:   "/path/to/file.json",
			updateMode:   true,
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
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
				if r.URL.Path != "/restAPI/importProfilesFromFile" {
					t.Errorf("Expected path /restAPI/importProfilesFromFile, got %s", r.URL.Path)
				}

				// Check headers
				if r.Header.Get("Authorization") != "Bearer "+tc.accessToken {
					t.Errorf("Expected Authorization header Bearer %s, got %s", tc.accessToken, r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Check request body
				var requestBody ImportProfilesFromFileRequest
				decoder := json.NewDecoder(r.Body)
				if err := decoder.Decode(&requestBody); err != nil {
					t.Errorf("Error decoding request body: %v", err)
				}

				if requestBody.UpdateMode != tc.updateMode {
					t.Errorf("Expected updateMode %v, got %v", tc.updateMode, requestBody.UpdateMode)
				}
				if requestBody.Path != tc.pathToFile {
					t.Errorf("Expected path %s, got %s", tc.pathToFile, requestBody.Path)
				}

				// Set response status
				w.WriteHeader(tc.serverStatus)
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
			err := client.ImportProfilesFromFile(ctx, server.Client(), tc.accessToken, tc.pathToFile, tc.updateMode)

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
		name         string
		accessToken  string
		udcName      string
		gdpMuHost    string
		serverStatus int
		expectError  bool
	}{
		{
			name:         "Successful installation",
			accessToken:  "test-token",
			udcName:      "connector-profile",
			gdpMuHost:    "host1.example.com",
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:         "Server error",
			accessToken:  "test-token",
			udcName:      "connector-profile",
			gdpMuHost:    "host1.example.com",
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
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

				// Set response status
				w.WriteHeader(tc.serverStatus)
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
