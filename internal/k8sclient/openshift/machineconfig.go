// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	mcfgv1 "github.com/openshift/api/machineconfiguration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IgnitionConfig represents a minimal Ignition 3.2.0 config structure
type IgnitionConfig struct {
	Ignition IgnitionVersion `json:"ignition"`
	Storage  *Storage        `json:"storage,omitempty"`
}

type IgnitionVersion struct {
	Version string `json:"version"`
}

type Storage struct {
	Files []File `json:"files,omitempty"`
}

type File struct {
	Path     string       `json:"path"`
	Mode     *int         `json:"mode,omitempty"`
	User     *FileUser    `json:"user,omitempty"`
	Contents FileContents `json:"contents"`
}

type FileUser struct {
	Name string `json:"name,omitempty"`
}

type FileContents struct {
	Source string `json:"source"`
}

// CreateRegistryCertMachineConfig creates a MachineConfig for registry CA certificate
func (c *Client) CreateRegistryCertMachineConfig(ctx context.Context, name, registryHost string, certData []byte) error {
	certBase64 := base64.StdEncoding.EncodeToString(certData)
	certPath := fmt.Sprintf("/etc/containers/certs.d/%s/ca.crt", registryHost)

	mode := 0644
	ignitionConfig := IgnitionConfig{
		Ignition: IgnitionVersion{Version: "3.2.0"},
		Storage: &Storage{
			Files: []File{
				{
					Path: certPath,
					Mode: &mode,
					User: &FileUser{Name: "root"},
					Contents: FileContents{
						Source: fmt.Sprintf("data:text/plain;base64,%s", certBase64),
					},
				},
			},
		},
	}

	configJSON, err := json.Marshal(ignitionConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal ignition config: %w", err)
	}

	mc := &mcfgv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": "worker",
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: runtime.RawExtension{
				Raw: configJSON,
			},
		},
	}

	_, err = c.machineConfig.MachineconfigurationV1().MachineConfigs().Create(ctx, mc, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing MachineConfig
			existing, getErr := c.machineConfig.MachineconfigurationV1().MachineConfigs().Get(ctx, name, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("failed to get existing machineconfig: %w", getErr)
			}
			mc.ResourceVersion = existing.ResourceVersion
			_, err = c.machineConfig.MachineconfigurationV1().MachineConfigs().Update(ctx, mc, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update machineconfig: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create machineconfig: %w", err)
	}

	return nil
}

// DeleteMachineConfig deletes a MachineConfig
func (c *Client) DeleteMachineConfig(ctx context.Context, name string) error {
	err := c.machineConfig.MachineconfigurationV1().MachineConfigs().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete machineconfig: %w", err)
	}
	return nil
}

// GetMachineConfig retrieves a MachineConfig
func (c *Client) GetMachineConfig(ctx context.Context, name string) (*mcfgv1.MachineConfig, error) {
	mc, err := c.machineConfig.MachineconfigurationV1().MachineConfigs().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get machineconfig: %w", err)
	}
	return mc, nil
}
