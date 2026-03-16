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

// GetConfigMap retrieves a ConfigMap
func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s/%s: %w", namespace, name, err)
	}
	return cm, nil
}

// GetConfigMapField retrieves a specific field from a ConfigMap
func (c *Client) GetConfigMapField(ctx context.Context, namespace, name, field string) (string, error) {
	cm, err := c.GetConfigMap(ctx, namespace, name)
	if err != nil {
		return "", err
	}

	value, ok := cm.Data[field]
	if !ok {
		return "", fmt.Errorf("field %s not found in configmap %s/%s", field, namespace, name)
	}
	return value, nil
}

// CreateConfigMap creates a new ConfigMap
func (c *Client) CreateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	_, err := c.clientset.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap %s/%s: %w", cm.Namespace, cm.Name, err)
	}
	return nil
}

// UpdateConfigMap updates an existing ConfigMap
func (c *Client) UpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	_, err := c.clientset.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update configmap %s/%s: %w", cm.Namespace, cm.Name, err)
	}
	return nil
}

// CreateOrUpdateConfigMap creates or updates a ConfigMap
func (c *Client) CreateOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing, err := c.clientset.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return c.CreateConfigMap(ctx, cm)
	}
	if err != nil {
		return fmt.Errorf("failed to check configmap %s/%s: %w", cm.Namespace, cm.Name, err)
	}

	// Preserve resource version for update
	cm.ResourceVersion = existing.ResourceVersion
	return c.UpdateConfigMap(ctx, cm)
}

// DeleteConfigMap deletes a ConfigMap
func (c *Client) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete configmap %s/%s: %w", namespace, name, err)
	}
	return nil
}

// NewConfigMap creates a new ConfigMap object (doesn't persist)
func NewConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

// NewConfigMapWithBinaryData creates a ConfigMap with binary data
func NewConfigMapWithBinaryData(namespace, name string, data map[string]string, binaryData map[string][]byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data:       data,
		BinaryData: binaryData,
	}
}

// ConfigMapFieldCondition is a function that checks if a ConfigMap field meets a condition
type ConfigMapFieldCondition func(value string) (done bool, err error)

// WaitForConfigMapField polls a ConfigMap field until condition is met
func (c *Client) WaitForConfigMapField(ctx context.Context, namespace, name, field string,
	condition ConfigMapFieldCondition, interval time.Duration, maxAttempts int) (string, error) {

	var lastValue string
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		cm, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				lastErr = err
			}
			// ConfigMap doesn't exist yet, keep waiting
		} else if cm != nil {
			if value, ok := cm.Data[field]; ok {
				lastValue = value
				done, err := condition(value)
				if err != nil {
					return value, err
				}
				if done {
					return value, nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return lastValue, ctx.Err()
		case <-time.After(interval):
			continue
		}
	}

	if lastErr != nil {
		return lastValue, fmt.Errorf("timeout after %d attempts waiting for configmap %s/%s field %s: %w",
			maxAttempts, namespace, name, field, lastErr)
	}
	return lastValue, fmt.Errorf("timeout after %d attempts waiting for configmap %s/%s field %s",
		maxAttempts, namespace, name, field)
}

// WaitForConfigMapExists waits for a ConfigMap to exist
func (c *Client) WaitForConfigMapExists(ctx context.Context, namespace, name string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		_, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	})
}
