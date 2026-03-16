package k8sclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// Kubeconfig file path (K3S, OpenShift)
	KubeconfigPath string

	// Platform type: k3s, eks, openshift
	Platform string

	// EKS-specific
	AWSRegion      string
	AWSProfile     string
	AWSAccessKey   string
	AWSSecretKey   string
	EKSClusterName string

	// OpenShift-specific (native OAuth)
	OCPServer             string // API server URL (e.g., https://api.cluster.example.com:6443)
	OCPUsername           string // OpenShift username
	OCPPassword           string // OpenShift password
	OCPToken              string // Pre-existing OAuth token (alternative to username/password)
	OCPInsecureSkipVerify bool   // Skip TLS certificate verification
}

// BuildRestConfig creates a Kubernetes REST config based on platform
func BuildRestConfig(ctx context.Context, cfg AuthConfig) (*rest.Config, error) {
	switch cfg.Platform {
	case "eks":
		return buildEKSConfig(ctx, cfg)
	case "openshift":
		// Use native OAuth if OCP credentials provided, otherwise fall back to kubeconfig
		if cfg.OCPServer != "" && (cfg.OCPToken != "" || (cfg.OCPUsername != "" && cfg.OCPPassword != "")) {
			return buildOpenShiftConfig(ctx, cfg)
		}
		return buildKubeconfigConfig(cfg.KubeconfigPath)
	case "k3s":
		return buildKubeconfigConfig(cfg.KubeconfigPath)
	default:
		// Try in-cluster config first, then kubeconfig
		if inCluster, err := rest.InClusterConfig(); err == nil {
			return inCluster, nil
		}
		return buildKubeconfigConfig(cfg.KubeconfigPath)
	}
}

// buildKubeconfigConfig loads config from kubeconfig file
func buildKubeconfigConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		// Try default locations
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
		// Also check KUBECONFIG env var
		if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
			kubeconfigPath = envPath
		}
	}

	if kubeconfigPath == "" {
		return nil, fmt.Errorf("kubeconfig path not specified and no default found")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// buildOpenShiftConfig creates config with OpenShift OAuth authentication
func buildOpenShiftConfig(ctx context.Context, cfg AuthConfig) (*rest.Config, error) {
	var token string
	var err error

	// Use provided token or obtain one via OAuth
	if cfg.OCPToken != "" {
		token = cfg.OCPToken
	} else {
		token, err = getOpenShiftOAuthToken(cfg.OCPServer, cfg.OCPUsername, cfg.OCPPassword, cfg.OCPInsecureSkipVerify)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain OpenShift OAuth token: %w", err)
		}
	}

	// Build REST config with bearer token
	restConfig := &rest.Config{
		Host:        cfg.OCPServer,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: cfg.OCPInsecureSkipVerify,
		},
	}

	return restConfig, nil
}

// getOpenShiftOAuthToken obtains an OAuth token from OpenShift using username/password
// This implements the OAuth Resource Owner Password Credentials flow
func getOpenShiftOAuthToken(server, username, password string, insecureSkipVerify bool) (string, error) {
	// Normalize server URL
	server = strings.TrimSuffix(server, "/")

	// Create HTTP client with optional TLS skip verify
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
		// Don't follow redirects automatically - we need to handle OAuth flow
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: Discover the OAuth server metadata
	oauthServerURL, err := discoverOAuthServer(httpClient, server)
	if err != nil {
		// Fall back to well-known OAuth endpoint
		oauthServerURL = server
	}

	// Step 2: Request authorization using the OAuth authorize endpoint
	// OpenShift uses the "openshift-challenging-client" for CLI-style authentication
	authorizeURL := fmt.Sprintf("%s/oauth/authorize?client_id=openshift-challenging-client&response_type=token", oauthServerURL)

	req, err := http.NewRequest("GET", authorizeURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create authorize request: %w", err)
	}

	// Set Basic Auth header for challenging client flow
	req.SetBasicAuth(username, password)
	req.Header.Set("X-CSRF-Token", "1")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("OAuth authorize request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle different response scenarios
	switch resp.StatusCode {
	case http.StatusFound, http.StatusSeeOther:
		// Success - token is in the redirect URL fragment
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("OAuth redirect missing Location header")
		}
		return extractTokenFromRedirect(location)

	case http.StatusUnauthorized:
		// Check if this is a negotiate/challenge response
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if strings.Contains(wwwAuth, "Basic") {
			return "", fmt.Errorf("authentication failed: invalid username or password")
		}
		return "", fmt.Errorf("authentication failed: %s", wwwAuth)

	default:
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OAuth authorize failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// discoverOAuthServer discovers the OAuth server URL from the API server
func discoverOAuthServer(client *http.Client, apiServer string) (string, error) {
	// OpenShift exposes OAuth metadata at .well-known/oauth-authorization-server
	metadataURL := fmt.Sprintf("%s/.well-known/oauth-authorization-server", apiServer)

	resp, err := client.Get(metadataURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get OAuth metadata: status %d", resp.StatusCode)
	}

	var metadata struct {
		Issuer                string `json:"issuer"`
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", fmt.Errorf("failed to parse OAuth metadata: %w", err)
	}

	if metadata.Issuer != "" {
		return metadata.Issuer, nil
	}

	return "", fmt.Errorf("OAuth issuer not found in metadata")
}

// extractTokenFromRedirect extracts the access token from OAuth redirect URL
func extractTokenFromRedirect(location string) (string, error) {
	// The token is in the URL fragment: https://...#access_token=xxx&token_type=Bearer&...
	parsedURL, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect URL: %w", err)
	}

	// Fragment contains the token parameters
	fragment := parsedURL.Fragment
	if fragment == "" {
		// Some OAuth servers put token in query params instead
		fragment = parsedURL.RawQuery
	}

	values, err := url.ParseQuery(fragment)
	if err != nil {
		return "", fmt.Errorf("failed to parse token fragment: %w", err)
	}

	token := values.Get("access_token")
	if token == "" {
		return "", fmt.Errorf("access_token not found in OAuth redirect")
	}

	return token, nil
}

// eksTokenTransport wraps an http.RoundTripper to inject a fresh EKS bearer token
// on every request. This avoids token expiry issues since a new presigned URL is
// generated for each k8s API call.
type eksTokenTransport struct {
	base        http.RoundTripper
	awsCfg      aws.Config
	clusterName string
}

func (t *eksTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := generateEKSToken(req.Context(), t.awsCfg, t.clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate EKS token: %w", err)
	}
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token)
	return t.base.RoundTrip(req)
}

// buildEKSConfig creates config with EKS authentication.
// Uses WrapTransport to generate a fresh token per request, avoiding expiry issues.
func buildEKSConfig(ctx context.Context, cfg AuthConfig) (*rest.Config, error) {
	// Build AWS config
	var awsOpts []func(*config.LoadOptions) error

	if cfg.AWSRegion != "" {
		awsOpts = append(awsOpts, config.WithRegion(cfg.AWSRegion))
	}

	if cfg.AWSProfile != "" {
		awsOpts = append(awsOpts, config.WithSharedConfigProfile(cfg.AWSProfile))
	}

	if cfg.AWSAccessKey != "" && cfg.AWSSecretKey != "" {
		awsOpts = append(awsOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKey, cfg.AWSSecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get cluster info from EKS
	eksClient := eks.NewFromConfig(awsCfg)
	cluster, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(cfg.EKSClusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EKS cluster: %w", err)
	}

	// Decode CA certificate
	caData, err := base64.StdEncoding.DecodeString(*cluster.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cluster CA: %w", err)
	}

	return &rest.Config{
		Host: *cluster.Cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
		// WrapTransport injects a fresh EKS token on every request.
		// This ensures tokens never expire, even during long deployments.
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &eksTokenTransport{
				base:        rt,
				awsCfg:      awsCfg,
				clusterName: cfg.EKSClusterName,
			}
		},
	}, nil
}

// generateEKSToken generates an EKS authentication token using STS presigned URL.
// Uses the low-level v4 signer to construct a presigned GetCallerIdentity GET request
// with x-k8s-aws-id as a signed header, matching the aws-iam-authenticator token format.
func generateEKSToken(ctx context.Context, awsCfg aws.Config, clusterName string) (string, error) {
	// Retrieve credentials from the AWS config
	creds, err := awsCfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Build STS GetCallerIdentity as a GET request with query parameters
	region := awsCfg.Region
	if region == "" {
		region = "us-east-1"
	}

	// Build the STS GetCallerIdentity presigned URL.
	// X-Amz-Expires=60 is required — the EKS authenticator rejects tokens without it
	// (must be ≤ 900 seconds).
	stsURL := fmt.Sprintf("https://sts.%s.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15&X-Amz-Expires=60", region)
	req, err := http.NewRequest("GET", stsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create STS request: %w", err)
	}

	// x-k8s-aws-id must be a signed header so the EKS authenticator can verify
	// the token was generated for this specific cluster
	req.Header.Set("x-k8s-aws-id", clusterName)

	// Presign with SigV4 — the signer includes x-k8s-aws-id in the signature
	signer := v4.NewSigner()
	// SHA256 of empty payload (GET request has no body)
	const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	presignedURL, _, err := signer.PresignHTTP(ctx, creds, req, emptyPayloadHash, "sts", region, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to presign GetCallerIdentity: %w", err)
	}

	// Format: k8s-aws-v1.<base64url-encoded-presigned-url>
	return "k8s-aws-v1." + base64.RawURLEncoding.EncodeToString([]byte(presignedURL)), nil
}
