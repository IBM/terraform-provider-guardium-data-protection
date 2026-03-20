package k8sclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k8sclient/openshift"
)

// Client wraps Kubernetes client operations
type Client struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	mapper    *restmapper.DeferredDiscoveryRESTMapper
	platform  string
	openshift *openshift.Client
}

// Config holds client configuration
type Config struct {
	// Authentication
	KubeconfigPath string
	Platform       string // k3s, eks, openshift

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

// NewClient creates a new Kubernetes client based on configuration
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	authCfg := AuthConfig(cfg)

	restConfig, err := BuildRestConfig(ctx, authCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build REST config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	client := &Client{
		config:    restConfig,
		clientset: clientset,
		dynamic:   dynamicClient,
		discovery: discoveryClient,
		mapper:    mapper,
		platform:  cfg.Platform,
	}

	// Initialize OpenShift client if needed
	if cfg.Platform == "openshift" {
		client.openshift, err = openshift.NewClient(restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenShift client: %w", err)
		}
	}

	return client, nil
}

// Clientset returns the typed Kubernetes clientset
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

// Dynamic returns the dynamic client for unstructured resources
func (c *Client) Dynamic() dynamic.Interface {
	return c.dynamic
}

// RESTConfig returns the REST configuration
func (c *Client) RESTConfig() *rest.Config {
	return c.config
}

// Mapper returns the REST mapper for GVK to GVR conversion
func (c *Client) Mapper() *restmapper.DeferredDiscoveryRESTMapper {
	return c.mapper
}

// Platform returns the platform type (k3s, eks, openshift)
func (c *Client) Platform() string {
	return c.platform
}

// IsOpenShift returns true if connected to OpenShift
func (c *Client) IsOpenShift() bool {
	return c.platform == "openshift"
}

// OpenShift returns the OpenShift-specific client (nil if not OpenShift)
func (c *Client) OpenShift() *openshift.Client {
	return c.openshift
}
