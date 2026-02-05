// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

// NewSecureClient creates a new SecureClient with custom CA certificate
func (c *Client) NewSecureClient(caCertPath string) (*SecureClient, error) {
	if caCertPath == "" {
		return nil, fmt.Errorf("CA certificate path cannot be empty")
	}

	// Verify CA cert file exists
	if _, err := os.Stat(caCertPath); err != nil {
		return nil, fmt.Errorf("CA certificate file not found: %w", err)
	}

	return &SecureClient{
		Client: Client{
			Host:     c.Host,
			port:     c.port,
			protocol: "https",
		},
		CACertPath: caCertPath,
	}, nil
}

// createSecureHTTPClient creates an HTTP client with custom CA certificate
func (s *SecureClient) createSecureHTTPClient() (*http.Client, error) {
	// Load CA cert
	caCert, err := os.ReadFile(s.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Create cert pool and add CA cert
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config with custom CA
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	// Create HTTP client with secure TLS config
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// ImportProfilesFromFile imports profiles from a local file using secure TLS connection
func (s *SecureClient) ImportProfilesFromFile(ctx context.Context, accessToken, pathToFile string, updateMode bool) error {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	return s.Client.ImportProfilesFromFile(ctx, secureClient, accessToken, pathToFile, updateMode)
}

// GenerateAccessToken generates an access token using secure TLS connection
func (s *SecureClient) GenerateAccessToken(ctx context.Context, clientSecret, username, password, clientId string) (string, error) {
	secureClient, err := s.createSecureHTTPClient()
	if err != nil {
		return "", fmt.Errorf("failed to create secure HTTP client: %w", err)
	}

	otr, err := s.Client.generateAccessToken(ctx, secureClient, clientSecret, username, password, clientId)
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

// Made with Bob
