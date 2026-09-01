// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
)

// NewSecureClient creates a new SecureClient with verified TLS.
//
// caCertPath may be:
//   - a non-empty file path to a PEM-encoded CA certificate — that CA is added
//     to the trust pool (useful for self-signed / private PKI appliances).
//   - an empty string — the server's certificate chain is fetched automatically
//     via a TLS dial to c.Host:c.port and the leaf certificate is trusted.
//
// A non-empty path that does not exist on disk is rejected immediately so
// configuration errors surface before any network call is attempted.
func (c *Client) NewSecureClient(caCertPath string) (*SecureClient, error) {
	var fetchedCACert []byte

	if caCertPath != "" {
		// Verify CA cert file exists up-front so operators get a clear error.
		if _, err := os.Stat(caCertPath); err != nil {
			return nil, fmt.Errorf("CA certificate file not found: %w", err)
		}
	} else {
		// No path provided — fetch the server's certificate chain automatically.
		cert, err := FetchServerCACert(c.Host, c.port)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-fetch server CA certificate from %s:%s: %w", c.Host, c.port, err)
		}
		fetchedCACert = cert
	}

	return &SecureClient{
		Client: Client{
			Host:     c.Host,
			port:     c.port,
			protocol: "https",
		},
		CACertPath:    caCertPath,
		fetchedCACert: fetchedCACert,
	}, nil
}

// FetchServerCACert connects to host:port over TLS (skipping verification for
// the handshake only) and returns the PEM-encoded certificate chain presented
// by the server.  The returned PEM contains every certificate in the chain so
// it can be used as a trusted CA pool for subsequent verified connections.
func FetchServerCACert(host, port string) ([]byte, error) {
	// InsecureSkipVerify is intentional here — we are only fetching the cert,
	// not trusting it yet.  The PEM we return is what the caller will pin.
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // intentional bootstrap fetch
	})
	if err != nil {
		return nil, fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("server presented no certificates")
	}

	var pemData []byte
	for _, cert := range certs {
		pemData = append(pemData, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})...)
	}
	return pemData, nil
}

// createSecureHTTPClient creates an HTTP client with verified TLS.
// It prefers fetchedCACert (auto-fetched PEM bytes) over CACertPath (file).
// When both are empty, RootCAs is left nil so Go uses the OS trust store.
// InsecureSkipVerify is never set.
func (s *SecureClient) createSecureHTTPClient() (*http.Client, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	switch {
	case len(s.fetchedCACert) > 0:
		// Use the PEM bytes fetched at construction time.
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(s.fetchedCACert) {
			return nil, fmt.Errorf("failed to parse auto-fetched CA certificate")
		}
		tlsConfig.RootCAs = caCertPool

	case s.CACertPath != "":
		// Load the operator-supplied CA certificate from disk.
		caCert, err := os.ReadFile(s.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool

	default:
		// tlsConfig.RootCAs remains nil — Go verifies against the OS trust store.
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// ImportProfilesFromFile imports profiles from a local file using secure TLS connection
func (s *SecureClient) ImportProfilesFromFile(ctx context.Context, accessToken, pathToFile string, updateMode, testConnections bool) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.ImportProfilesFromFile(ctx, secureClient, accessToken, pathToFile, updateMode, testConnections)
}

// GenerateAccessToken generates an access token using secure TLS connection
func (s *SecureClient) GenerateAccessToken(ctx context.Context, clientSecret, username, password, clientId string) (string, error) {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return "", fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	otr, err := s.generateAccessToken(ctx, secureClient, clientSecret, username, password, clientId)
	if err != nil {
		return "", err
	}

	return otr.AccessToken, nil
}

// BulkInstallConnector installs connectors in bulk using secure TLS connection
func (s *SecureClient) BulkInstallConnector(ctx context.Context, accessToken, udcName, gdpMuHost string) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.BulkInstallConnector(ctx, secureClient, accessToken, udcName, gdpMuHost)
}

// CreateAWSSecretsManager creates AWS Secrets Manager configuration using secure TLS connection
func (s *SecureClient) CreateAWSSecretsManager(ctx context.Context, accessToken string, config *AWSSecretsManagerConfig) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.CreateAWSSecretsManager(ctx, secureClient, accessToken, config)
}

// GetAWSSecretsManager retrieves AWS Secrets Manager configuration using secure TLS connection
func (s *SecureClient) GetAWSSecretsManager(ctx context.Context, accessToken string, name string) (*AWSSecretsManagerConfig, error) {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.GetAWSSecretsManager(ctx, secureClient, accessToken, name)
}

// GetExistingAWSSecretsManagerNames gets existing AWS Secrets Manager names using secure TLS connection
func (s *SecureClient) GetExistingAWSSecretsManagerNames(ctx context.Context, accessToken string) ([]string, error) {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.GetExistingAWSSecretsManagerNames(ctx, secureClient, accessToken)
}

// UpdateAWSSecretsManager updates AWS Secrets Manager configuration using secure TLS connection
func (s *SecureClient) UpdateAWSSecretsManager(ctx context.Context, accessToken string, config *AWSSecretsManagerConfig) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.UpdateAWSSecretsManager(ctx, secureClient, accessToken, config)
}

// DeleteAWSSecretsManager deletes AWS Secrets Manager configuration using secure TLS connection
func (s *SecureClient) DeleteAWSSecretsManager(ctx context.Context, accessToken string, name string) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.DeleteAWSSecretsManager(ctx, secureClient, accessToken, name)
}

// RegisterVADataSource registers a VA datasource using secure TLS connection
func (s *SecureClient) RegisterVADataSource(ctx context.Context, accessToken string, payload []byte) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.RegisterVADataSource(ctx, secureClient, accessToken, payload)
}

// ConfigureVADataSource configures a VA datasource using secure TLS connection
func (s *SecureClient) ConfigureVADataSource(ctx context.Context, accessToken string, payload []byte) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.ConfigureVADataSource(ctx, secureClient, accessToken, payload)
}

// ConfigureVANotifications configures VA notifications using secure TLS connection
func (s *SecureClient) ConfigureVANotifications(ctx context.Context, accessToken string, payload []byte) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.ConfigureVANotifications(ctx, secureClient, accessToken, payload)
}
