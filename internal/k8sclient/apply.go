// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

const fieldManager = "terraform-provider-gdp-edge"

// ApplyManifest applies a single YAML manifest using Server-Side Apply
func (c *Client) ApplyManifest(ctx context.Context, manifest []byte, defaultNamespace string) error {
	// Decode YAML to unstructured
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(manifest, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Get REST mapping for GVK -> GVR conversion
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to get REST mapping for %v: %w", gvk, err)
	}

	// Determine if namespaced
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = defaultNamespace
		}
		if ns == "" {
			return fmt.Errorf("namespace required for %s %s but not specified", gvk.Kind, obj.GetName())
		}
		obj.SetNamespace(ns)
		dr = c.dynamic.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = c.dynamic.Resource(mapping.Resource)
	}

	// Convert to JSON for Server-Side Apply
	data, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	// Server-Side Apply
	_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: fieldManager,
	})
	if err != nil {
		return fmt.Errorf("failed to apply %s %s: %w", gvk.Kind, obj.GetName(), err)
	}

	return nil
}

// ApplyManifestFile applies a YAML file (supports multi-document)
func (c *Client) ApplyManifestFile(ctx context.Context, filePath string, defaultNamespace string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return c.ApplyManifestContent(ctx, content, defaultNamespace)
}

// ApplyManifestContent applies YAML content (supports multi-document)
func (c *Client) ApplyManifestContent(ctx context.Context, content []byte, defaultNamespace string) error {
	// Use YAML decoder that handles multi-document YAML
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)

	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode YAML document: %w", err)
		}

		// Skip empty documents
		if len(rawObj.Object) == 0 {
			continue
		}

		// Re-encode to YAML for ApplyManifest
		data, err := rawObj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}

		if err := c.ApplyManifest(ctx, data, defaultNamespace); err != nil {
			return fmt.Errorf("failed to apply from content: %w", err)
		}
	}

	return nil
}

// ApplyDirectory applies YAML files in a directory (top-level only, no subdirectories)
func (c *Client) ApplyDirectory(ctx context.Context, dirPath string, defaultNamespace string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		// Skip all subdirectories (handled separately in DeployEdge based on platform)
		if entry.IsDir() {
			continue
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())
		if err := c.ApplyManifestFile(ctx, path, defaultNamespace); err != nil {
			return err
		}
	}

	return nil
}

// ApplyManifestFiles applies multiple YAML files
func (c *Client) ApplyManifestFiles(ctx context.Context, filePaths []string, defaultNamespace string) error {
	for _, path := range filePaths {
		if err := c.ApplyManifestFile(ctx, path, defaultNamespace); err != nil {
			return err
		}
	}
	return nil
}

// DeleteManifest deletes resources defined in a YAML manifest
func (c *Client) DeleteManifest(ctx context.Context, manifest []byte, defaultNamespace string) error {
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(manifest, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}

	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to get REST mapping for %v: %w", gvk, err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = defaultNamespace
		}
		dr = c.dynamic.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = c.dynamic.Resource(mapping.Resource)
	}

	err = dr.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete %s %s: %w", gvk.Kind, obj.GetName(), err)
	}

	return nil
}

// DeleteManifestFile deletes resources defined in a YAML file
func (c *Client) DeleteManifestFile(ctx context.Context, filePath string, defaultNamespace string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return c.DeleteManifestContent(ctx, content, defaultNamespace)
}

// DeleteManifestContent deletes resources defined in YAML content
func (c *Client) DeleteManifestContent(ctx context.Context, content []byte, defaultNamespace string) error {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)

	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode YAML document: %w", err)
		}

		if len(rawObj.Object) == 0 {
			continue
		}

		data, err := rawObj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}

		if err := c.DeleteManifest(ctx, data, defaultNamespace); err != nil {
			return err
		}
	}

	return nil
}

// DeleteDirectory deletes resources from YAML files in a directory (top-level only, no subdirectories)
func (c *Client) DeleteDirectory(ctx context.Context, dirPath string, defaultNamespace string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		// Skip all subdirectories (handled separately based on platform)
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())
		if err := c.DeleteManifestFile(ctx, path, defaultNamespace); err != nil {
			return err
		}
	}

	return nil
}
