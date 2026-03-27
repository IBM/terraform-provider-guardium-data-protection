// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PatchImageConfig patches image.config.openshift.io/cluster to add additionalTrustedCA
func (c *Client) PatchImageConfig(ctx context.Context, additionalTrustedCAName string) error {
	patch := []byte(fmt.Sprintf(`{
		"spec": {
			"additionalTrustedCA": {
				"name": "%s"
			}
		}
	}`, additionalTrustedCAName))

	_, err := c.config.ConfigV1().Images().Patch(ctx, "cluster",
		types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch image config: %w", err)
	}

	return nil
}

// GetImageConfig retrieves the cluster image config
func (c *Client) GetImageConfig(ctx context.Context) (string, error) {
	image, err := c.config.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get image config: %w", err)
	}

	return image.Spec.AdditionalTrustedCA.Name, nil
}
