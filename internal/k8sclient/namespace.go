package k8sclient

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NamespaceExists checks if a namespace exists
func (c *Client) NamespaceExists(ctx context.Context, name string) (bool, error) {
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}
	return true, nil
}

// CreateNamespace creates a namespace if it doesn't exist
func (c *Client) CreateNamespace(ctx context.Context, name string) error {
	exists, err := c.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	return nil
}

// CreateNamespaceAndWait creates a namespace and waits for it to be active
func (c *Client) CreateNamespaceAndWait(ctx context.Context, name string, timeout time.Duration) error {
	if err := c.CreateNamespace(ctx, name); err != nil {
		return err
	}

	// Wait for namespace to be active
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return ns.Status.Phase == corev1.NamespaceActive, nil
	})
}

// DeleteNamespace deletes a namespace
func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete namespace %s: %w", name, err)
	}
	return nil
}

// DeleteNamespaceAndWait deletes a namespace and waits for it to be fully removed
func (c *Client) DeleteNamespaceAndWait(ctx context.Context, name string, timeout time.Duration) error {
	if err := c.DeleteNamespace(ctx, name); err != nil {
		return err
	}

	// Wait for namespace to be fully deleted
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		_, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

// GetNamespace returns a namespace
func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}
	return ns, nil
}
