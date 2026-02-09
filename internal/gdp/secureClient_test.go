// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// createTestCACert generates ephemeral test certificates for unit testing only.
// These keys are generated dynamically at test runtime and never stored or used
// for real authentication. They exist only in memory during test execution.
// GitGuardian: This is a test helper that generates temporary keys, not a secret leak.
func createTestCACert(t *testing.T) (string, *x509.Certificate, *rsa.PrivateKey) {
	// Generate ephemeral private key for testing only (not a real secret)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Write certificate to temp file
	tmpFile, err := os.CreateTemp(t.TempDir(), "ca-cert-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	if _, err := tmpFile.Write(certPEM); err != nil {
		t.Fatalf("Failed to write certificate: %v", err)
	}
	tmpFile.Close()

	return tmpFile.Name(), cert, privateKey
}

// TestNewSecureClient validates the creation and initialization of SecureClient
func TestNewSecureClient(t *testing.T) {
	testCases := []struct {
		name             string
		host             string
		port             string
		caCertPath       string
		createCert       bool
		expectError      bool
		expectedProtocol string
	}{
		{
			name:             "Valid secure client creation",
			host:             "guardium.example.com",
			port:             "8443",
			createCert:       true,
			expectError:      false,
			expectedProtocol: "https",
		},
		{
			name:        "Empty CA cert path",
			host:        "localhost",
			port:        "8443",
			caCertPath:  "",
			createCert:  false,
			expectError: true,
		},
		{
			name:        "Non-existent CA cert file",
			host:        "localhost",
			port:        "8443",
			caCertPath:  "/nonexistent/ca-cert.pem",
			createCert:  false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseClient := NewClient(tc.host, tc.port)

			var caCertPath string
			if tc.createCert {
				caCertPath, _, _ = createTestCACert(t)
			} else {
				caCertPath = tc.caCertPath
			}

			secureClient, err := baseClient.NewSecureClient(caCertPath)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if secureClient == nil {
				t.Fatal("Expected SecureClient to be created, got nil")
				return
			}

			// Verify Host is preserved
			if secureClient.Host != tc.host {
				t.Errorf("Expected Host to be %s, got %s", tc.host, secureClient.Host)
			}

			// Verify port is preserved
			if secureClient.port != tc.port {
				t.Errorf("Expected port to be %s, got %s", tc.port, secureClient.port)
			}

			// Verify protocol is always https
			if secureClient.protocol != tc.expectedProtocol {
				t.Errorf("Expected protocol to be %s, got %s", tc.expectedProtocol, secureClient.protocol)
			}

			// Verify CA cert path is set
			if secureClient.CACertPath != caCertPath {
				t.Errorf("Expected CACertPath to be %s, got %s", caCertPath, secureClient.CACertPath)
			}
		})
	}
}

// TestSecureClient_ProtocolEnforcement ensures protocol is always HTTPS
func TestSecureClient_ProtocolEnforcement(t *testing.T) {
	caCertPath, _, _ := createTestCACert(t)

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

			secureClient, err := baseClient.NewSecureClient(caCertPath)
			if err != nil {
				t.Fatalf("Failed to create secure client: %v", err)
			}

			if secureClient.protocol != tc.expectedProtocol {
				t.Errorf("Expected protocol to be %s, got %s", tc.expectedProtocol, secureClient.protocol)
			}
		})
	}
}

// TestSecureClient_CertificateValidation validates proper certificate validation
func TestSecureClient_CertificateValidation(t *testing.T) {
	// Create test CA and certificate
	caCertPath, caCert, caKey := createTestCACert(t)

	// Create a test server with certificate signed by our CA
	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverCert, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create server certificate: %v", err)
	}

	// Create TLS certificate for server
	serverTLSCert := tls.Certificate{
		Certificate: [][]byte{serverCertDER},
		PrivateKey:  serverKey,
	}

	// Create test server with our certificate
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"access_token":"test-token"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))

	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
	}
	server.StartTLS()
	defer server.Close()

	// Extract host and port
	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	t.Run("SecureClient accepts valid certificate", func(t *testing.T) {
		baseClient := &Client{
			Host: host,
			port: port,
		}
		secureClient, err := baseClient.NewSecureClient(caCertPath)
		if err != nil {
			t.Fatalf("Failed to create secure client: %v", err)
		}

		ctx := context.Background()
		token, err := secureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err != nil {
			t.Errorf("SecureClient should accept valid certificate, got error: %v", err)
		}
		if token != "test-token" {
			t.Errorf("Expected token 'test-token', got '%s'", token)
		}
	})
}

// TestSecureClient_InvalidCACertificate tests handling of invalid CA certificates
func TestSecureClient_InvalidCACertificate(t *testing.T) {
	testCases := []struct {
		name        string
		certContent string
		expectError bool
	}{
		{
			name:        "Invalid PEM format",
			certContent: "This is not a valid certificate",
			expectError: true,
		},
		{
			name:        "Empty certificate file",
			certContent: "",
			expectError: true,
		},
		{
			name: "Valid PEM but not a certificate",
			certContent: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF0K3j8v8rR0Jx8nQJc8pjZJ0Z3VS
-----END RSA PRIVATE KEY-----`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file with test content
			tmpFile, err := os.CreateTemp(t.TempDir(), "invalid-cert-*.pem")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			if _, err := tmpFile.WriteString(tc.certContent); err != nil {
				t.Errorf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			baseClient := NewClient("localhost", "8443")
			secureClient, err := baseClient.NewSecureClient(tmpFile.Name())

			if err != nil {
				// Error during creation is acceptable
				return
			}

			// Try to use the client
			ctx := context.Background()
			_, err = secureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

			if tc.expectError && err == nil {
				t.Error("Expected error for invalid certificate but got nil")
			}
		})
	}
}

// TestSecureClient_GenerateAccessToken tests token generation with secure client
func TestSecureClient_GenerateAccessToken(t *testing.T) {
	caCertPath, caCert, caKey := createTestCACert(t)

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
			serverResponse: `{"access_token":"secure-token-123"}`,
			expectError:    false,
			expectedToken:  "secure-token-123",
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
			// Create server certificate
			serverCert := createServerCert(t, caCert, caKey)

			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/oauth/token") {
					t.Errorf("Expected /oauth/token path, got %s", r.URL.Path)
				}

				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverResponse)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))

			server.TLS = &tls.Config{
				Certificates: []tls.Certificate{*serverCert},
			}
			server.StartTLS()
			defer server.Close()

			serverURL := strings.TrimPrefix(server.URL, "https://")
			urlSplit := strings.Split(serverURL, ":")
			host := urlSplit[0]
			port := urlSplit[1]

			baseClient := &Client{
				Host: host,
				port: port,
			}
			secureClient, err := baseClient.NewSecureClient(caCertPath)
			if err != nil {
				t.Fatalf("Failed to create secure client: %v", err)
			}

			ctx := context.Background()
			token, err := secureClient.GenerateAccessToken(ctx, tc.clientSecret, tc.username, tc.password, tc.clientId)

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

// createServerCert generates ephemeral server certificates for unit testing only.
// These keys are generated dynamically at test runtime and never stored or used
// for real authentication. They exist only in memory during test execution.
// GitGuardian: This is a test helper that generates temporary keys, not a secret leak.
func createServerCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) *tls.Certificate {
	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Generate ephemeral server key for testing only (not a real secret)
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverCert, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create server certificate: %v", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{serverCertDER},
		PrivateKey:  serverKey,
	}

	return tlsCert
}

// TestSecureClient_AllOperations tests all SecureClient operations
func TestSecureClient_AllOperations(t *testing.T) {
	caCertPath, caCert, caKey := createTestCACert(t)
	serverCert := createServerCert(t, caCert, caKey)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/restAPI/datasource":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"ds-123","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/va/config":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"config-456","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/notifications":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"notif-789","message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/bulkInstall":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"ID":"install-123","Message":"success"}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		case "/restAPI/aws_secrets_manager":
			if r.Method == "GET" {
				response := []map[string]interface{}{
					{
						"id":                          1,
						"name":                        "test-secret",
						"authType":                    "access_key",
						"accessKeyId":                 "AKIAIOSFODNN7EXAMPLE",
						"secretAccessKey":             "secret",
						"secretKeyUsernameIdentifier": "username-key",
						"secretKeyPasswordIdentifier": "password-key",
						"roleARN":                     "",
						"secretsManager":              true,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			} else {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(`{"message":"success"}`)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))

	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{*serverCert},
	}
	server.StartTLS()
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "https://")
	urlSplit := strings.Split(serverURL, ":")
	host := urlSplit[0]
	port := urlSplit[1]

	baseClient := &Client{
		Host: host,
		port: port,
	}
	secureClient, err := baseClient.NewSecureClient(caCertPath)
	if err != nil {
		t.Fatalf("Failed to create secure client: %v", err)
	}

	ctx := context.Background()
	token := "test-token"

	// Test VA operations
	t.Run("RegisterVADataSource", func(t *testing.T) {
		payload := []byte(`{"datasourceName":"test-db"}`)
		err := secureClient.RegisterVADataSource(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("ConfigureVADataSource", func(t *testing.T) {
		payload := []byte(`{"datasourceName":"test-db"}`)
		err := secureClient.ConfigureVADataSource(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("ConfigureVANotifications", func(t *testing.T) {
		payload := []byte(`{"email":"admin@example.com"}`)
		err := secureClient.ConfigureVANotifications(ctx, token, payload)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test UC operations
	t.Run("BulkInstallConnector", func(t *testing.T) {
		err := secureClient.BulkInstallConnector(ctx, token, "connector", "host")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test AWS Secrets Manager operations
	t.Run("GetAWSSecretsManager", func(t *testing.T) {
		result, err := secureClient.GetAWSSecretsManager(ctx, token, "test-secret")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})
}

// TestSecureClient_ErrorHandling tests error scenarios
func TestSecureClient_ErrorHandling(t *testing.T) {
	caCertPath, _, _ := createTestCACert(t)

	t.Run("Unreachable server", func(t *testing.T) {
		baseClient := &Client{
			Host: "unreachable.invalid.host",
			port: "8443",
		}
		secureClient, err := baseClient.NewSecureClient(caCertPath)
		if err != nil {
			t.Fatalf("Failed to create secure client: %v", err)
		}

		ctx := context.Background()
		_, err = secureClient.GenerateAccessToken(ctx, "secret", "user", "pass", "client")

		if err == nil {
			t.Error("Expected error for unreachable server, got nil")
		}
	})
}

