package openshift

import (
	"fmt"

	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	mcfgv1client "github.com/openshift/client-go/machineconfiguration/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps OpenShift-specific clients
type Client struct {
	config        *configv1client.Clientset
	machineConfig *mcfgv1client.Clientset
	clientset     *kubernetes.Clientset
}

// NewClient creates OpenShift-specific clients
func NewClient(restConfig *rest.Config) (*Client, error) {
	configClient, err := configv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config client: %w", err)
	}

	mcfgClient, err := mcfgv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create machineconfiguration client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		config:        configClient,
		machineConfig: mcfgClient,
		clientset:     clientset,
	}, nil
}

// ConfigClient returns the OpenShift config client
func (c *Client) ConfigClient() *configv1client.Clientset {
	return c.config
}

// MachineConfigClient returns the MachineConfiguration client
func (c *Client) MachineConfigClient() *mcfgv1client.Clientset {
	return c.machineConfig
}

// Clientset returns the standard Kubernetes clientset
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}
