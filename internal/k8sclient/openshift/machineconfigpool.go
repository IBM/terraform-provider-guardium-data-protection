package openshift

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// MachineConfigPoolStatus holds the status of a MachineConfigPool
type MachineConfigPoolStatus struct {
	MachineCount        int32
	UpdatedMachineCount int32
	DegradedMachineCount int32
	ReadyMachineCount   int32
}

// GetMachineConfigPoolStatus retrieves the status of a MachineConfigPool
func (c *Client) GetMachineConfigPoolStatus(ctx context.Context, poolName string) (*MachineConfigPoolStatus, error) {
	mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
	}

	return &MachineConfigPoolStatus{
		MachineCount:        mcp.Status.MachineCount,
		UpdatedMachineCount: mcp.Status.UpdatedMachineCount,
		DegradedMachineCount: mcp.Status.DegradedMachineCount,
		ReadyMachineCount:   mcp.Status.ReadyMachineCount,
	}, nil
}

// WaitForMachineConfigPoolUpdate waits for all machines in the pool to be updated.
// It first waits for the MCP to enter Updating=True (rollout has started), then waits
// for UpdatedMachineCount == MachineCount (rollout has finished). This prevents a false
// positive on the initial poll before the MachineConfig daemon has begun processing the
// newly created MachineConfig.
func (c *Client) WaitForMachineConfigPoolUpdate(ctx context.Context, poolName string, timeout time.Duration) error {
	sawUpdating := false
	return wait.PollUntilContextTimeout(ctx, 30*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		mcp, err := c.machineConfig.MachineconfigurationV1().MachineConfigPools().Get(ctx, poolName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get machineconfigpool %s: %w", poolName, err)
		}

		// Check if all machines are updated and not degraded
		if mcp.Status.MachineCount == 0 {
			// No machines in pool, consider it done
			return true, nil
		}

		// Track whether we've seen the rollout begin
		for _, condition := range mcp.Status.Conditions {
			if condition.Type == "Updating" && condition.Status == "True" {
				sawUpdating = true
				break
			}
		}

		// Don't declare done until we've confirmed the rollout actually started
		if !sawUpdating {
			return false, nil
		}

		allUpdated := mcp.Status.UpdatedMachineCount == mcp.Status.MachineCount
		noneDegraded := mcp.Status.DegradedMachineCount == 0

		return allUpdated && noneDegraded, nil
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
