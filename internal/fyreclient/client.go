package fyreclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents the Fyre API client
type Client struct {
	Username       string
	APIKey         string
	ClusterType    string
	ProductGroupID string
	HTTPClient     *http.Client
}

// NewClient creates a new Fyre API client
func NewClient(username, apiKey, clusterType, productGroupID string) *Client {
	return &Client{
		Username:       username,
		APIKey:         apiKey,
		ClusterType:    clusterType,
		ProductGroupID: productGroupID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// GetAPIEndpoint returns the appropriate API endpoint based on cluster type
func (c *Client) GetAPIEndpoint() string {
	if c.ClusterType == "beta-fyre" {
		return "https://ocpapi.svl.ibm.com/v1/vm/"
	}
	return "https://api.fyre.ibm.com/rest/v1/?operation=build"
}

// GetDomainSuffix returns the domain suffix based on cluster type
func (c *Client) GetDomainSuffix() string {
	if c.ClusterType == "beta-fyre" {
		return "dev.fyre.ibm.com"
	}
	return "fyre.ibm.com"
}

// CreateVMRequest represents the request to create a VM
type CreateVMRequest struct {
	ClusterName   string
	MasterNodes   []NodeConfig
	WorkerNodes   []NodeConfig
	ClusterConfig ClusterConfig
	NetworkConfig NetworkConfig
}

// NodeConfig represents a node configuration
type NodeConfig struct {
	Name               string
	Count              int
	CPU                int
	Memory             int
	OS                 string
	AdditionalDiskSize int
}

// ClusterConfig represents cluster configuration
type ClusterConfig struct {
	Platform string
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	PublicVLAN  string
	PrivateVLAN string
	DNS         string
}

// OCPNodeConfig represents an OCP node configuration
type OCPNodeConfig struct {
	Count          int
	CPU            int
	Memory         int
	AdditionalDisk []int
}

// CreateOCPRequest represents the request to create an OCP cluster
type CreateOCPRequest struct {
	Name           string
	Description    string
	Platform       string
	Site           string
	QuotaType      string
	ProductGroupID string
	TimeToLive     string
	OCPVersion     string
	FIPS           string
	SSHKey         string
	Master         []OCPNodeConfig
	Worker         []OCPNodeConfig
}

// OCPStatusResponse represents the response from OCP status check
type OCPStatusResponse struct {
	DeployedStatus string `json:"deployed_status"`
	ClusterName    string `json:"cluster_name"`
	ClusterID      string `json:"cluster_id"`
	VMs            []struct {
		State string `json:"state"`
	} `json:"vms"`
}

// OCPClusterDetails represents the detailed cluster info from GET /v1/ocp/{cluster_name}
type OCPClusterDetails struct {
	ClusterName       string `json:"cluster_name"`
	KubeadminPassword string `json:"kubeadmin_password"`
	AccessURL         string `json:"access_url"`
	DeploymentStatus  string `json:"deployment_status"`
}

// OCPClusterDetailsResponse represents the top-level API response
type OCPClusterDetailsResponse struct {
	Clusters []OCPClusterDetails `json:"clusters"`
}

// CreateVMResponse represents the response from VM creation
type CreateVMResponse struct {
	RequestID string `json:"request_id"`
	Details   string `json:"details"`
	Status    string `json:"status"`
}

// VMRequestInfo represents a single request status entry
// Note: Fyre API may return numeric fields as either strings or numbers
type VMRequestInfo struct {
	LastStatus        string      `json:"last_status"`
	CompletionPercent json.Number `json:"completion_percent"`
	Failed            json.Number `json:"failed"`
	Complete          json.Number `json:"complete"`
	InProgress        json.Number `json:"in_progress"`
	Pending           json.Number `json:"pending"`
	Status            string      `json:"status"`
	Details           string      `json:"details"`
	ErrorDetails      string      `json:"error_details"`
}

// VMStatusResponse represents the response from status check
// Standard fyre returns request as an array, beta-fyre as an object
type VMStatusResponse struct {
	Status  string        `json:"status"`
	Request VMRequestInfo `json:"-"`
}

func (v *VMStatusResponse) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Status  string          `json:"status"`
		Request json.RawMessage `json:"request"`
	}
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	v.Status = alias.Status

	if len(alias.Request) == 0 {
		return nil
	}

	// Try array first (standard fyre)
	var arr []VMRequestInfo
	if err := json.Unmarshal(alias.Request, &arr); err == nil && len(arr) > 0 {
		v.Request = arr[0]
		return nil
	}

	// Try object (beta-fyre)
	return json.Unmarshal(alias.Request, &v.Request)
}

// CreateVM creates a new VM via the Fyre API
func (c *Client) CreateVM(req CreateVMRequest) (*CreateVMResponse, error) {
	var payload interface{}

	if c.ClusterType == "beta-fyre" {
		payload = c.buildBetaFyrePayload(req)
	} else {
		payload = c.buildStandardFyrePayload(req)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.GetAPIEndpoint(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateVMResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createResp, nil
}

// GetVMStatus checks the status of a VM creation request
func (c *Client) GetVMStatus(requestID string) (*VMStatusResponse, error) {
	var url string
	if c.ClusterType == "beta-fyre" {
		url = fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/vm/request/%s", requestID)
	} else {
		// For standard fyre, requestID is the details URL from creation response
		url = requestID
	}

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var statusResp VMStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// GetVMInfo checks if a VM exists by querying the Fyre API
// Returns nil if the VM does not exist (404), or the response body on success
func (c *Client) GetVMInfo(clusterName string) (map[string]interface{}, error) {
	var url string
	if c.ClusterType == "beta-fyre" {
		url = fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/vm/%s", clusterName)
	} else {
		url = fmt.Sprintf("https://api.fyre.ibm.com/rest/v1/?operation=query&request=showvmdetails&cluster_name=%s", clusterName)
	}

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // VM does not exist
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error status in response body
	if status, ok := result["status"].(string); ok && status == "error" {
		return nil, nil // VM does not exist
	}

	return result, nil
}

// GetOCPInfo checks if an OCP cluster exists by querying the Fyre API
// Returns nil if the cluster does not exist (404), or the status response on success
func (c *Client) GetOCPInfo(clusterName string) (*OCPStatusResponse, error) {
	url := fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/ocp/%s/status", clusterName)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Cluster does not exist
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var statusResp OCPStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// DeleteVM deletes a VM
func (c *Client) DeleteVM(clusterName string) error {
	var url string
	var method string
	var body io.Reader
	if c.ClusterType == "beta-fyre" {
		url = fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/vm/%s", clusterName)
		method = "DELETE"
	} else {
		url = "https://api.fyre.ibm.com/rest/v1/?operation=delete"
		method = "POST"
		jsonData, _ := json.Marshal(map[string]string{"cluster_name": clusterName})
		body = bytes.NewBuffer(jsonData)
	}
	httpReq, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	if c.ClusterType != "beta-fyre" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildBetaFyrePayload builds the payload for beta-fyre API
func (c *Client) buildBetaFyrePayload(req CreateVMRequest) map[string]interface{} {
	nodeArray := []map[string]interface{}{}

	// Add master nodes
	for _, master := range req.MasterNodes {
		nodeArray = append(nodeArray, map[string]interface{}{
			"hostname":        []string{fmt.Sprintf("%s-%s", req.ClusterName, master.Name)},
			"platform":        req.ClusterConfig.Platform,
			"cpu":             fmt.Sprintf("%d", master.CPU),
			"memory":          fmt.Sprintf("%d", master.Memory),
			"os":              master.OS,
			"public_network":  req.NetworkConfig.PublicVLAN,
			"dns":             req.NetworkConfig.DNS,
			"additional_disk": []string{fmt.Sprintf("%d", master.AdditionalDiskSize)},
		})
	}

	// Add worker nodes
	for _, worker := range req.WorkerNodes {
		nodeArray = append(nodeArray, map[string]interface{}{
			"hostname":        []string{fmt.Sprintf("%s-%s", req.ClusterName, worker.Name)},
			"platform":        req.ClusterConfig.Platform,
			"cpu":             fmt.Sprintf("%d", worker.CPU),
			"memory":          fmt.Sprintf("%d", worker.Memory),
			"os":              worker.OS,
			"public_network":  req.NetworkConfig.PublicVLAN,
			"dns":             req.NetworkConfig.DNS,
			"additional_disk": []string{fmt.Sprintf("%d", worker.AdditionalDiskSize)},
		})
	}

	return map[string]interface{}{
		"quota_type":       "product_group",
		"product_group_id": c.ProductGroupID,
		"node_array":       nodeArray,
	}
}

// buildStandardFyrePayload builds the payload for standard fyre API
func (c *Client) buildStandardFyrePayload(req CreateVMRequest) map[string]interface{} {
	nodes := []map[string]interface{}{}

	// Add master nodes
	for _, master := range req.MasterNodes {
		nodes = append(nodes, map[string]interface{}{
			"name":        master.Name,
			"count":       master.Count,
			"cpu":         master.CPU,
			"memory":      master.Memory,
			"os":          master.OS,
			"publicvlan":  req.NetworkConfig.PublicVLAN,
			"privatevlan": req.NetworkConfig.PrivateVLAN,
			"additional_disks": []map[string]interface{}{
				{"size": master.AdditionalDiskSize},
			},
		})
	}

	// Add worker nodes
	for _, worker := range req.WorkerNodes {
		nodes = append(nodes, map[string]interface{}{
			"name":        worker.Name,
			"count":       worker.Count,
			"cpu":         worker.CPU,
			"memory":      worker.Memory,
			"os":          worker.OS,
			"publicvlan":  req.NetworkConfig.PublicVLAN,
			"privatevlan": req.NetworkConfig.PrivateVLAN,
			"additional_disks": []map[string]interface{}{
				{"size": worker.AdditionalDiskSize},
			},
		})
	}

	return map[string]interface{}{
		"fyre": map[string]interface{}{
			"creds": map[string]interface{}{
				"username":   c.Username,
				"api_key":    c.APIKey,
				"public_key": "",
			},
		},
		"product_group_id": c.ProductGroupID,
		"cluster_prefix":   req.ClusterName,
		"clusterconfig": map[string]interface{}{
			"instance_type": "virtual_server",
			"platform":      req.ClusterConfig.Platform,
		},
		req.ClusterName: nodes,
	}
}

// WaitForVMReady polls the API until the VM is ready or times out
func (c *Client) WaitForVMReady(requestID string, timeoutMinutes int) error {
	timeout := time.Duration(timeoutMinutes) * time.Minute
	deadline := time.Now().Add(timeout)
	pollInterval := 30 * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetVMStatus(requestID)
		if err != nil {
			return fmt.Errorf("failed to get VM status: %w", err)
		}

		if c.ClusterType == "beta-fyre" {
			// Beta-fyre status check
			if status.Status == "success" &&
				status.Request.CompletionPercent.String() == "100" &&
				status.Request.Failed.String() == "0" &&
				status.Request.InProgress.String() == "0" &&
				status.Request.Pending.String() == "0" {
				return nil
			}
			if status.Request.Failed.String() != "0" && status.Request.Failed.String() != "null" {
				return fmt.Errorf("VM creation failed: %s", status.Request.ErrorDetails)
			}
		} else {
			// Standard fyre status check
			if status.Status == "error" {
				return fmt.Errorf("VM creation failed: %s", status.Request.Details)
			}
			if status.Request.Status == "error" {
				return fmt.Errorf("VM creation failed: %s", status.Request.ErrorDetails)
			}
			if status.Request.Status == "completed" {
				return nil
			}
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for VM to be ready after %d minutes", timeoutMinutes)
}

// CreateOCP creates a new OCP cluster via the Fyre API
func (c *Client) CreateOCP(req CreateOCPRequest) (*CreateVMResponse, error) {
	payload := c.buildOCPPayload(req)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := "https://ocpapi.svl.ibm.com/v1/ocp"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateVMResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createResp, nil
}

// GetOCPStatus checks the status of an OCP cluster
func (c *Client) GetOCPStatus(clusterName string) (*OCPStatusResponse, error) {
	url := fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/ocp/%s/status", clusterName)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var statusResp OCPStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// GetOCPClusterDetails fetches detailed cluster info including kubeadmin_password
func (c *Client) GetOCPClusterDetails(clusterName string) (*OCPClusterDetails, error) {
	url := fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/ocp/%s", clusterName)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var detailsResp OCPClusterDetailsResponse
	if err := json.Unmarshal(body, &detailsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(detailsResp.Clusters) == 0 {
		return nil, fmt.Errorf("no cluster details found for %s", clusterName)
	}

	return &detailsResp.Clusters[0], nil
}

// DeleteOCP deletes an OCP cluster
func (c *Client) DeleteOCP(clusterName string) error {
	url := fmt.Sprintf("https://ocpapi.svl.ibm.com/v1/ocp/%s", clusterName)

	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.Username, c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildOCPPayload builds the payload for OCP API
func (c *Client) buildOCPPayload(req CreateOCPRequest) map[string]interface{} {
	payload := map[string]interface{}{
		"name":             req.Name,
		"description":      req.Description,
		"platform":         req.Platform,
		"site":             req.Site,
		"quota_type":       req.QuotaType,
		"product_group_id": req.ProductGroupID,
		"time_to_live":     req.TimeToLive,
		"ocp_version":      req.OCPVersion,
		"fips":             req.FIPS,
		"ssh-key":          req.SSHKey,
	}

	// Add master nodes
	masterNodes := []map[string]interface{}{}
	for _, master := range req.Master {
		node := map[string]interface{}{
			"count":  fmt.Sprintf("%d", master.Count),
			"cpu":    fmt.Sprintf("%d", master.CPU),
			"memory": fmt.Sprintf("%d", master.Memory),
		}
		if len(master.AdditionalDisk) > 0 {
			disks := []string{}
			for _, disk := range master.AdditionalDisk {
				disks = append(disks, fmt.Sprintf("%d", disk))
			}
			node["additional_disk"] = disks
		}
		masterNodes = append(masterNodes, node)
	}
	payload["master"] = masterNodes

	// Add worker nodes
	workerNodes := []map[string]interface{}{}
	for _, worker := range req.Worker {
		node := map[string]interface{}{
			"count":  fmt.Sprintf("%d", worker.Count),
			"cpu":    fmt.Sprintf("%d", worker.CPU),
			"memory": fmt.Sprintf("%d", worker.Memory),
		}
		if len(worker.AdditionalDisk) > 0 {
			disks := []string{}
			for _, disk := range worker.AdditionalDisk {
				disks = append(disks, fmt.Sprintf("%d", disk))
			}
			node["additional_disk"] = disks
		}
		workerNodes = append(workerNodes, node)
	}
	payload["worker"] = workerNodes

	return payload
}

// WaitForOCPReady polls the API until the OCP cluster is ready or times out
func (c *Client) WaitForOCPReady(clusterName string, timeoutMinutes int, pollIntervalSeconds int) error {
	timeout := time.Duration(timeoutMinutes) * time.Minute
	deadline := time.Now().Add(timeout)
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second

	for time.Now().Before(deadline) {
		status, err := c.GetOCPStatus(clusterName)
		if err != nil {
			return fmt.Errorf("failed to get OCP status: %w", err)
		}

		// Check if cluster is deployed and all VMs are running
		if status.DeployedStatus == "deployed" {
			allRunning := true
			for _, vm := range status.VMs {
				if vm.State != "running" {
					allRunning = false
					break
				}
			}
			if allRunning {
				return nil
			}
		}

		// Check for error states
		if status.DeployedStatus == "error" || status.DeployedStatus == "failed" {
			return fmt.Errorf("OCP cluster deployment failed with status: %s", status.DeployedStatus)
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for OCP cluster to be ready after %d minutes", timeoutMinutes)
}
