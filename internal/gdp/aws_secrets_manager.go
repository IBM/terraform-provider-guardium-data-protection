package gdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AWSSecretsManagerConfig represents the configuration for AWS Secrets Manager
type AWSSecretsManagerConfig struct {
	Name              string `json:"name"`
	AuthType          string `json:"auth_type"`
	AccessKeyID       string `json:"access_key_id"`
	SecretAccessKey   string `json:"secret_access_key"`
	SecretKeyUsername string `json:"secret_key_username"`
	SecretKeyPassword string `json:"secret_key_password"`
}

// NewAWSSecretsManagerConfig creates a new AWSSecretsManagerConfig with the provided values
func NewAWSSecretsManagerConfig(name, authType, accessKeyID, secretAccessKey, secretKeyUsername, secretKeyPassword string) *AWSSecretsManagerConfig {
	return &AWSSecretsManagerConfig{
		Name:              name,
		AuthType:          authType,
		AccessKeyID:       accessKeyID,
		SecretAccessKey:   secretAccessKey,
		SecretKeyUsername: secretKeyUsername,
		SecretKeyPassword: secretKeyPassword,
	}
}

// CreateAWSSecretsManager creates a new AWS Secrets Manager configuration
func (c *Client) CreateAWSSecretsManager(ctx context.Context, httpClient *http.Client, accessToken string, config *AWSSecretsManagerConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s://%s:%s/restAPI/aws_secrets_manager", c.protocol, c.Host, c.port)

	// Marshal the request body
	jsonBody, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}

	tflog.Debug(ctx, "AWS Secrets Manager create request URL: "+url)
	tflog.Debug(ctx, "AWS Secrets Manager create request body: "+string(jsonBody))

	// Create the request - use POST for creating new configurations
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error response from server: %s, status code: %d", string(body), resp.StatusCode)
	}

	tflog.Debug(ctx, "AWS Secrets Manager create response: "+string(body))
	return nil
}

// UpdateAWSSecretsManager updates an existing AWS Secrets Manager configuration
func (c *Client) UpdateAWSSecretsManager(ctx context.Context, httpClient *http.Client, accessToken string, config *AWSSecretsManagerConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s://%s:%s/restAPI/aws_secrets_manager", c.protocol, c.Host, c.port)

	// Marshal the request body
	jsonBody, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}

	tflog.Debug(ctx, "AWS Secrets Manager update request URL: "+url)
	tflog.Debug(ctx, "AWS Secrets Manager update request body: "+string(jsonBody))

	// Create the request - use PUT for updating existing configurations
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error response from server: %s, status code: %d", string(body), resp.StatusCode)
	}

	tflog.Debug(ctx, "AWS Secrets Manager update response: "+string(body))
	return nil
}

// GetExistingAWSSecretsManagerNames gets the list of existing AWS Secrets Manager configuration names
func (c *Client) GetExistingAWSSecretsManagerNames(ctx context.Context, httpClient *http.Client, accessToken string) ([]string, error) {
	// Get all configurations
	configs, err := c.GetAllAWSSecretsManagerConfigs(ctx, httpClient, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting all AWS Secrets Manager configurations: %w", err)
	}

	// Extract the names
	var names []string
	for _, config := range configs {
		names = append(names, config.Name)
	}

	return names, nil
}

// GetAllAWSSecretsManagerConfigs gets all AWS Secrets Manager configurations
func (c *Client) GetAllAWSSecretsManagerConfigs(ctx context.Context, httpClient *http.Client, accessToken string) ([]AWSSecretsManagerConfig, error) {
	// Prepare the request URL
	url := fmt.Sprintf("%s://%s:%s/restAPI/aws_secrets_manager", c.protocol, c.Host, c.port)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from server: %s, status code: %d", string(body), resp.StatusCode)
	}

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	tflog.Debug(ctx, "AWS Secrets Manager response body: "+string(body))

	// Try to unmarshal as an array
	var configs []struct {
		ID                          int    `json:"id"`
		Name                        string `json:"name"`
		AccessKeyID                 string `json:"accessKeyId"`
		SecretAccessKey             string `json:"secretAccessKey"`
		AuthType                    string `json:"authType"`
		RoleARN                     string `json:"roleARN"`
		SecretKeyUsernameIdentifier string `json:"secretKeyUsernameIdentifier"`
		SecretKeyPasswordIdentifier string `json:"secretKeyPasswordIdentifier"`
		SecretsManager              bool   `json:"secretsManager"`
	}

	if err := json.Unmarshal(body, &configs); err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	// Convert to AWSSecretsManagerConfig
	var result []AWSSecretsManagerConfig
	for _, c := range configs {
		config := *NewAWSSecretsManagerConfig(
			c.Name,
			c.AuthType,
			c.AccessKeyID,
			c.SecretAccessKey,
			c.SecretKeyUsernameIdentifier,
			c.SecretKeyPasswordIdentifier,
		)
		result = append(result, config)
	}

	return result, nil
}

// GetAWSSecretsManager retrieves an AWS Secrets Manager configuration by name
func (c *Client) GetAWSSecretsManager(ctx context.Context, httpClient *http.Client, accessToken string, name string) (*AWSSecretsManagerConfig, error) {
	// Get all configurations
	configs, err := c.GetAllAWSSecretsManagerConfigs(ctx, httpClient, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting all AWS Secrets Manager configurations: %w", err)
	}

	// Find the configuration with the matching name
	for _, config := range configs {
		if config.Name == name {
			return &config, nil
		}
	}

	// If no matching configuration is found, return nil
	return nil, nil
}

// DeleteAWSSecretsManager deletes an AWS Secrets Manager configuration by name
func (c *Client) DeleteAWSSecretsManager(ctx context.Context, httpClient *http.Client, accessToken string, name string) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s://%s:%s/restAPI/aws_secrets_manager", c.protocol, c.Host, c.port)

	// Create the request body with the name parameter
	type DeleteRequestBody struct {
		Name string `json:"name"`
	}
	requestBody := DeleteRequestBody{
		Name: name,
	}

	// Marshal the request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}

	tflog.Debug(ctx, "AWS Secrets Manager delete request URL: "+url)
	tflog.Debug(ctx, "AWS Secrets Manager delete request body: "+string(jsonBody))

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("error response from server: %s, status code: %d", string(body), resp.StatusCode)
	}

	tflog.Debug(ctx, "AWS Secrets Manager delete response: "+string(body))
	return nil
}
