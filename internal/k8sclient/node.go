package k8sclient

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNodeNames returns all node names in the cluster
func (c *Client) ListNodeNames(ctx context.Context) ([]string, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	names := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		names[i] = node.Name
	}
	return names, nil
}

// ListNodeNamesWithLabel returns node names with a specific label
func (c *Client) ListNodeNamesWithLabel(ctx context.Context, labelSelector string) ([]string, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes with label %s: %w", labelSelector, err)
	}

	names := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		names[i] = node.Name
	}
	return names, nil
}

// ListWorkerNodeNames returns worker node names (nodes without control-plane/master role)
func (c *Client) ListWorkerNodeNames(ctx context.Context) ([]string, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var workerNames []string
	for _, node := range nodes.Items {
		// Check for control plane labels
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			continue
		}
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			continue
		}
		workerNames = append(workerNames, node.Name)
	}
	return workerNames, nil
}

// GetNodeInternalIP returns the internal IP of a node
func (c *Client) GetNodeInternalIP(ctx context.Context, nodeName string) (string, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("internal IP not found for node %s", nodeName)
}

// GetNodeExternalIP returns the external IP of a node
func (c *Client) GetNodeExternalIP(ctx context.Context, nodeName string) (string, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == "ExternalIP" {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("external IP not found for node %s", nodeName)
}
