// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Client struct {
	protocol string
	Host     string
	port     string
}

type SecureClient struct {
	Client
	CACertPath string
}

func NewClient(host, port string) *Client {
	return &Client{
		Host: host,
		port: port,
	}
}

type OauthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c *Client) generateAccessToken(ctx context.Context, httpClient *http.Client, clientSecret, username, password, clientId string) (*OauthTokenResponse, error) {
	parsedUrl, err := url.Parse(fmt.Sprintf("%s://%s:%s/oauth/token", c.protocol, c.Host, c.port))
	if err != nil {
		tflog.Error(ctx, "failed to parse url "+err.Error())
		return nil, err
	}

	queryParams := parsedUrl.Query()
	queryParams.Set("client_id", clientId)
	queryParams.Set("client_secret", clientSecret)
	queryParams.Set("password", password)
	queryParams.Set("username", username)
	queryParams.Set("grant_type", "password")

	parsedUrl.RawQuery = queryParams.Encode()
	tflog.Info(ctx, "parsed url "+parsedUrl.String())
	req, err := http.NewRequest("POST", parsedUrl.String(), nil)
	if err != nil {
		tflog.Error(ctx, "failed to create new request "+err.Error())
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		tflog.Error(ctx, "failed to preform request "+err.Error())
		return nil, err
	}

	if res.StatusCode == http.StatusBadRequest {
		tflog.Error(ctx, "invalid credentials for access token. Please review your client_id and client_secret values")
		return nil, fmt.Errorf("invalid credentials for access token. Please review your client_id and client_secret values")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		tflog.Error(ctx, "failed to read body "+err.Error())
		return nil, err
	}

	otr := new(OauthTokenResponse)
	if err := json.Unmarshal(body, otr); err != nil {
		tflog.Error(ctx, "failed to parse body "+err.Error())
		return nil, err
	}

	return otr, nil
}

type ImportProfilesFromFileRequest struct {
	UpdateMode bool   `json:"updateMode"`
	Path       string `json:"path"`
}

type ImportProfilesFromFileResponse struct {
	ID      string `json:"ID"`
	Message string `json:"Message"`
}

// ImportProfilesFromFile imports profiles from a file
// Supports two methods:
// 1. Multipart upload: If pathToFile exists locally, uploads file content via multipart/form-data
// 2. Legacy SFTP: If pathToFile doesn't exist locally (server path), sends JSON with server path
func (c *Client) ImportProfilesFromFile(ctx context.Context, httpClient *http.Client, accessToken, pathToFile string, updateMode bool) error {
	// Prepare the request URL
	importProfilesFromFileUrl := fmt.Sprintf("%s://%s:%s/restAPI/importProfilesFromFile", c.protocol, c.Host, c.port)

	// Detect if this is a local file path or server path by checking if file exists locally
	_, err := os.Stat(pathToFile)
	isLocalFile := err == nil

	var req *http.Request

	if isLocalFile {
		// NEW METHOD: Multipart upload for local files
		tflog.Info(ctx, "Detected local file - using multipart upload", map[string]any{"pathToFile": pathToFile})

		file, err := os.Open(pathToFile)
		if err != nil {
			tflog.Error(ctx, "Error opening local file", map[string]any{"pathToFile": pathToFile, "error": err.Error()})
			return fmt.Errorf("error opening file %s: %w", pathToFile, err)
		}
		defer file.Close()

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add the file to the multipart form
		part, err := writer.CreateFormFile("path", filepath.Base(pathToFile))
		if err != nil {
			return fmt.Errorf("error creating form file: %w", err)
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return fmt.Errorf("error copying file content: %w", err)
		}

		// Add updateMode parameter
		updateModeStr := "false"
		if updateMode {
			updateModeStr = "true"
		}
		err = writer.WriteField("updateMode", updateModeStr)
		if err != nil {
			return fmt.Errorf("error writing updateMode field: %w", err)
		}

		// Add TestConnections parameter
		err = writer.WriteField("TestConnections", "false")
		if err != nil {
			return fmt.Errorf("error writing TestConnections field: %w", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("error closing multipart writer: %w", err)
		}

		// Create the request
		req, err = http.NewRequestWithContext(ctx, "POST", importProfilesFromFileUrl, body)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		tflog.Info(ctx, "Sending multipart upload request", map[string]any{
			"url":         importProfilesFromFileUrl,
			"contentType": writer.FormDataContentType(),
		})
	} else {
		// LEGACY METHOD: JSON API with server path (for SFTP)
		tflog.Info(ctx, "File not found locally - using legacy SFTP method with server path", map[string]any{"pathToFile": pathToFile})

		requestBody := ImportProfilesFromFileRequest{
			UpdateMode: updateMode,
			Path:       pathToFile,
		}

		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}

		req, err = http.NewRequestWithContext(ctx, "POST", importProfilesFromFileUrl, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req.Header.Set("Content-Type", "application/json")

		tflog.Info(ctx, "Sending JSON request with server path", map[string]any{
			"url":        importProfilesFromFileUrl,
			"serverPath": pathToFile,
		})
	}

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	tflog.Info(ctx, "Received response", map[string]any{
		"statusCode":   resp.StatusCode,
		"responseBody": string(responseBody),
	})

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from server: %s, status code: %d", string(responseBody), resp.StatusCode)
	}

	// Parse the response body to check for errors
	var apiResponse ImportProfilesFromFileResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		tflog.Warn(ctx, "failed to parse import profiles response, continuing anyway: "+err.Error())
		tflog.Debug(ctx, "sent request to import profiles from file response "+string(responseBody))
		return nil
	}

	tflog.Debug(ctx, "sent request to import profiles from file response "+string(responseBody))

	// Check if the Message field contains an error
	if apiResponse.Message != "" && containsErrorKeywords(apiResponse.Message) {
		return fmt.Errorf("import profiles failed: %s", apiResponse.Message)
	}

	return nil
}

// containsErrorKeywords checks if a message contains error-indicating keywords
func containsErrorKeywords(message string) bool {
	errorKeywords := []string{
		"not found",
		"not supported",
		"failed",
		"error",
		"invalid",
		"could not",
		"unable to",
		"does not exist",
	}

	messageLower := bytes.ToLower([]byte(message))
	for _, keyword := range errorKeywords {
		if bytes.Contains(messageLower, []byte(keyword)) {
			return true
		}
	}
	return false
}

type bulkInstallRequestBody struct {
	ProfileNames string `json:"profileNames"`
	Hosts        string `json:"hosts"`
}

var (
	bulkInstallErrors = map[string]struct{}{
		"One or more of the specified hosts could not be found": {},
	}
)

type bulkInstallConnectorResponse struct {
	ID      string `json:"ID"`
	Message string `json:"Message"`
}

// BulkInstallConnector installs connectors in bulk
func (c *Client) BulkInstallConnector(ctx context.Context, httpClient *http.Client, accessToken, udcName, gdpMuHost string) error {
	// Create the request URL
	bulkInstallUrl := fmt.Sprintf("%s://%s:%s/restAPI/bulkInstall", c.protocol, c.Host, c.port)
	// Create the request body
	requestBody := &bulkInstallRequestBody{
		ProfileNames: udcName,
		Hosts:        gdpMuHost,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}
	tflog.Debug(ctx, "parsed install connector url "+bulkInstallUrl)
	tflog.Debug(ctx, "parsed install connector body "+string(jsonBody))

	// Create the HTTP request
	req, err := http.NewRequest("POST", bulkInstallUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	parsedBody := new(bulkInstallConnectorResponse)
	if err = json.Unmarshal(body, parsedBody); err != nil {
		tflog.Warn(ctx, "failed to parse bulk install response, continuing anyway: "+err.Error())
		tflog.Debug(ctx, "install connector response "+string(body))
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from server: %s, status code: %d", string(body), resp.StatusCode)
	}

	// Check for known error messages in the predefined map
	if _, k := bulkInstallErrors[parsedBody.Message]; k {
		return fmt.Errorf("bulk install failed: %s", parsedBody.Message)
	}

	// Also check if the Message field contains error keywords
	// The API may return ID="0" but still have an error in the Message field
	if parsedBody.Message != "" && containsErrorKeywords(parsedBody.Message) {
		return fmt.Errorf("bulk install failed: %s", parsedBody.Message)
	}

	tflog.Debug(ctx, "install connector response "+string(body))
	return nil
}

// RegisterDatasourceResponse represents the response from the API
type RegisterDatasourceResponse struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (c *Client) RegisterVADataSource(ctx context.Context, httpClient *http.Client, accessToken string, payload []byte) error {
	// Create the request URL
	registerURL := fmt.Sprintf("%s://%s:%s/restAPI/datasource", c.protocol, c.Host, c.port)
	tflog.Debug(ctx, "register data source url "+registerURL)
	tflog.Debug(ctx, fmt.Sprintf("register data source payload %s", string(payload)))
	tflog.Debug(ctx, fmt.Sprintf("register data source token  %s", accessToken))

	test := make(map[string]interface{})
	err := json.Unmarshal(payload, &test)
	if err != nil {
		panic(err)
	}

	payloadJson, err := json.Marshal(test)
	if err != nil {
		panic(err)
	}

	tflog.Debug(ctx, "string(payloadJson)")
	tflog.Debug(ctx, string(payloadJson))

	tflog.Debug(ctx, fmt.Sprintf("json output "+string(payloadJson)))
	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", registerURL, bytes.NewReader(payloadJson))
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not create request: %s", err))
		return err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send the request
	res, err := httpClient.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not send request: %s", err))
		return err
	}
	defer res.Body.Close()

	// Parse the response
	var apiResp RegisterDatasourceResponse
	body, err := io.ReadAll(res.Body)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not parse response: %s. Body %s", err, string(body)))
		return err
	}
	tflog.Debug(ctx, "register data source response "+string(body))

	// Check for errors
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		tflog.Error(ctx, fmt.Sprintf("Status code: %d, Error: %s, Message: %s", res.StatusCode, apiResp.Error, apiResp.Message))
		return err
	}
	return nil
}

// VAConfigResponse represents the response from the API
type VAConfigResponse struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ConfigureVADataSource configures the va datasource
func (c *Client) ConfigureVADataSource(ctx context.Context, httpClient *http.Client, accessToken string, payload []byte) error {
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("Could not marshal payload: %s", err))
		return err
	}

	// Create the request URL
	configURL := fmt.Sprintf("%s://%s:%s/restAPI/va/config", c.protocol, c.Host, c.port)

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", configURL, bytes.NewReader(payloadBytes))
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("Could not create request: %s", err))
		return err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send the request
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("Could not send request: %s", err))
		return err
	}
	defer httpResp.Body.Close()

	// Parse the response
	var apiResp VAConfigResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not parse response: %s", err))
		return err
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		tflog.Debug(ctx, fmt.Sprintf("Status code: %d, Error: %s, Message: %s",
			httpResp.StatusCode, apiResp.Error, apiResp.Message))
		return err
	}
	return nil
}

// NotificationsResponse represents the response from the API
type NotificationsResponse struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ConfigureVANotifications configure va notifications
func (c *Client) ConfigureVANotifications(ctx context.Context, httpClient *http.Client, accessToken string, payload []byte) error {
	// Create the request URL
	notificationsURL := fmt.Sprintf("%s://%s:%s/restAPI/notifications", c.protocol, c.Host, c.port)

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", notificationsURL, bytes.NewReader(payload))
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not create request: %s", err))
		return err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send the request
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not send request: %s", err))
		return err
	}
	defer httpResp.Body.Close()

	// Parse the response
	var apiResp NotificationsResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Could not parse response: %s", err))
		return err
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		tflog.Error(ctx, fmt.Sprintf("Status code: %d, Error: %s, Message: %s", httpResp.StatusCode, apiResp.Error, apiResp.Message))
		return err
	}
	return nil
}
