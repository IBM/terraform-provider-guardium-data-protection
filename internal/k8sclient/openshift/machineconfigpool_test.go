// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestMachineConfigPoolStatus tests the MachineConfigPoolStatus struct
func TestMachineConfigPoolStatus(t *testing.T) {
	status := MachineConfigPoolStatus{
		MachineCount:         10,
		UpdatedMachineCount:  8,
		DegradedMachineCount: 0,
		ReadyMachineCount:    8,
	}

	if status.MachineCount != 10 {
		t.Errorf("MachineCount = %d, want %d", status.MachineCount, 10)
	}

	if status.UpdatedMachineCount != 8 {
		t.Errorf("UpdatedMachineCount = %d, want %d", status.UpdatedMachineCount, 8)
	}

	if status.DegradedMachineCount != 0 {
		t.Errorf("DegradedMachineCount = %d, want %d", status.DegradedMachineCount, 0)
	}

	if status.ReadyMachineCount != 8 {
		t.Errorf("ReadyMachineCount = %d, want %d", status.ReadyMachineCount, 8)
	}
}

// TestMachineConfigPoolStatus_AllUpdated tests fully updated status
func TestMachineConfigPoolStatus_AllUpdated(t *testing.T) {
	status := MachineConfigPoolStatus{
		MachineCount:         5,
		UpdatedMachineCount:  5,
		DegradedMachineCount: 0,
		ReadyMachineCount:    5,
	}

	if status.UpdatedMachineCount != status.MachineCount {
		t.Error("Expected all machines to be updated")
	}
	if status.DegradedMachineCount != 0 {
		t.Error("Expected no degraded machines")
	}
	if status.ReadyMachineCount != status.MachineCount {
		t.Error("Expected all machines to be ready")
	}
}

// TestMachineConfigPoolStatus_PartialUpdate tests partially updated status
func TestMachineConfigPoolStatus_PartialUpdate(t *testing.T) {
	status := MachineConfigPoolStatus{
		MachineCount:         10,
		UpdatedMachineCount:  6,
		DegradedMachineCount: 1,
		ReadyMachineCount:    5,
	}

	if status.UpdatedMachineCount == status.MachineCount {
		t.Error("Expected not all machines to be updated")
	}
	if status.DegradedMachineCount == 0 {
		t.Error("Expected some degraded machines")
	}
}

// TestMachineConfigPoolStatus_ZeroMachines tests status with no machines
func TestMachineConfigPoolStatus_ZeroMachines(t *testing.T) {
	status := MachineConfigPoolStatus{}
	if status.MachineCount != 0 {
		t.Errorf("MachineCount = %d, want 0", status.MachineCount)
	}
}

// TestGetMachineConfigPoolStatus_ConnectionError tests error wrapping
func TestGetMachineConfigPoolStatus_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	status, err := client.GetMachineConfigPoolStatus(context.Background(), "worker")
	if err == nil {
		t.Fatal("GetMachineConfigPoolStatus() expected error without accessible server")
	}
	if status != nil {
		t.Fatalf("GetMachineConfigPoolStatus() = %#v, want nil on error", status)
	}
	if !strings.Contains(err.Error(), "failed to get machineconfigpool") {
		t.Fatalf("GetMachineConfigPoolStatus() error = %v, want 'failed to get machineconfigpool'", err)
	}
}

// TestIsMachineConfigPoolUpdating_ConnectionError tests error wrapping
func TestIsMachineConfigPoolUpdating_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	updating, err := client.IsMachineConfigPoolUpdating(context.Background(), "worker")
	if err == nil {
		t.Fatal("IsMachineConfigPoolUpdating() expected error without accessible server")
	}
	if updating {
		t.Fatal("IsMachineConfigPoolUpdating() = true, want false on error")
	}
	if !strings.Contains(err.Error(), "failed to get machineconfigpool") {
		t.Fatalf("IsMachineConfigPoolUpdating() error = %v, want 'failed to get machineconfigpool'", err)
	}
}

// TestWaitForMachineConfigPoolUpdate_ConnectionError tests that connection errors propagate
func TestWaitForMachineConfigPoolUpdate_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	err := client.WaitForMachineConfigPoolUpdate(context.Background(), "worker", time.Millisecond)
	if err == nil {
		t.Fatal("WaitForMachineConfigPoolUpdate() expected error without accessible server")
	}
}

// TestWaitForMachineConfigPoolReady_ConnectionError tests that connection errors propagate
func TestWaitForMachineConfigPoolReady_ConnectionError(t *testing.T) {
	client := newTestClient(t)

	err := client.WaitForMachineConfigPoolReady(context.Background(), "worker", time.Millisecond)
	if err == nil {
		t.Fatal("WaitForMachineConfigPoolReady() expected error without accessible server")
	}
}

// Made with Bob
