// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// MachineConfigPoolStatus holds the status of a MachineConfigPool
type MachineConfigPoolStatus struct {
	MachineCount         int32
	UpdatedMachineCount  int32
	DegradedMachineCount int32
	ReadyMachineCount    int32
}

// GetMachineConfigPoolStatus retrieves the status of a MachineConfigPool
func (c *Client) GetMachineConfigPoolStatus(ctx context.Context, poolName string) (*MachineConfigPoolStatus, error) {
	mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
	}

	return &MachineConfigPoolStatus{
		MachineCount:         mcp.Status.MachineCount,
		UpdatedMachineCount:  mcp.Status.UpdatedMachineCount,
		DegradedMachineCount: mcp.Status.DegradedMachineCount,
		ReadyMachineCount:    mcp.Status.ReadyMachineCount,
	}, nil
}

// WaitForMachineConfigPoolUpdate waits for all machines in the pool to be updated.
// It handles three scenarios:
// 1. Rollout is in progress (Updating=True)
// 2. Rollout completed before we started polling (all machines already updated)
// 3. Rollout completes during polling
func (c *Client) WaitForMachineConfigPoolUpdate(ctx context.Context, poolName string, timeout time.Duration) error {
	tflog.Info(ctx, "Waiting for MachineConfigPool update", map[string]interface{}{
		"pool":    poolName,
		"timeout": timeout.String(),
	})

	sawUpdating := false
	initialCheckDone := false
	var initialUpdatedCount int32

	return wait.PollUntilContextTimeout(ctx, 30*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
		}

		// Check if all machines are updated and not degraded
		if mcp.Status.MachineCount == 0 {
			tflog.Info(ctx, "MachineConfigPool has no machines, considering it done", map[string]interface{}{
				"pool": poolName,
			})
			return true, nil
		}

		// On first check, record the initial state
		if !initialCheckDone {
			initialUpdatedCount = mcp.Status.UpdatedMachineCount
			initialCheckDone = true
			tflog.Info(ctx, "Initial MachineConfigPool state", map[string]interface{}{
				"pool":                 poolName,
				"machineCount":         mcp.Status.MachineCount,
				"updatedMachineCount":  mcp.Status.UpdatedMachineCount,
				"degradedMachineCount": mcp.Status.DegradedMachineCount,
				"readyMachineCount":    mcp.Status.ReadyMachineCount,
			})
		}

		// Check if currently updating
		isUpdating := false
		for _, condition := range mcp.Status.Conditions {
			if condition.Type == "Updating" && condition.Status == "True" {
				isUpdating = true
				sawUpdating = true
				break
			}
		}

		allUpdated := mcp.Status.UpdatedMachineCount == mcp.Status.MachineCount
		noneDegraded := mcp.Status.DegradedMachineCount == 0

		tflog.Debug(ctx, "MachineConfigPool status check", map[string]interface{}{
			"pool":                 poolName,
			"machineCount":         mcp.Status.MachineCount,
			"updatedMachineCount":  mcp.Status.UpdatedMachineCount,
			"degradedMachineCount": mcp.Status.DegradedMachineCount,
			"readyMachineCount":    mcp.Status.ReadyMachineCount,
			"isUpdating":           isUpdating,
			"sawUpdating":          sawUpdating,
			"allUpdated":           allUpdated,
			"noneDegraded":         noneDegraded,
		})

		// Success conditions:
		// 1. All machines are updated and none degraded, AND
		// 2. Either we saw the Updating state, OR the update count changed from initial, OR already fully updated on first check
		if allUpdated && noneDegraded {
			if sawUpdating || mcp.Status.UpdatedMachineCount != initialUpdatedCount || initialUpdatedCount == mcp.Status.MachineCount {
				tflog.Info(ctx, "MachineConfigPool update completed successfully", map[string]interface{}{
					"pool":                poolName,
					"machineCount":        mcp.Status.MachineCount,
					"updatedMachineCount": mcp.Status.UpdatedMachineCount,
					"sawUpdating":         sawUpdating,
				})
				return true, nil
			}
		}

		// If we're currently updating, keep waiting
		if isUpdating {
			tflog.Info(ctx, "MachineConfigPool is updating, waiting...", map[string]interface{}{
				"pool":                poolName,
				"updatedMachineCount": mcp.Status.UpdatedMachineCount,
				"machineCount":        mcp.Status.MachineCount,
			})
			return false, nil
		}

		// If not updating and not all updated, keep waiting (rollout hasn't started yet)
		return false, nil
	})
}

// WaitForMachineConfigPoolReady waits for all machines to be ready
func (c *Client) WaitForMachineConfigPoolReady(ctx context.Context, poolName string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 30*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
		}

		if mcp.Status.MachineCount == 0 {
			return true, nil
		}

		allReady := mcp.Status.ReadyMachineCount == mcp.Status.MachineCount
		noneDegraded := mcp.Status.DegradedMachineCount == 0

		return allReady && noneDegraded, nil
	})
}

// IsMachineConfigPoolUpdating checks if the MCP is currently updating
func (c *Client) IsMachineConfigPoolUpdating(ctx context.Context, poolName string) (bool, error) {
	mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
	}

	// Check conditions for updating state
	for _, condition := range mcp.Status.Conditions {
		if condition.Type == "Updating" && condition.Status == "True" {
			return true, nil
		}
	}

	return false, nil
}
