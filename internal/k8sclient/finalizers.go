// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RemoveFinalizersFromNamespace discovers every namespaced resource type via the
// discovery API and removes metadata.finalizers from any live instances that have
// them.  Partial discovery errors (stale API groups) are logged and ignored so
// that as many resources as possible are still processed.
func (c *Client) RemoveFinalizersFromNamespace(ctx context.Context, namespace string) error {
	// ServerPreferredNamespacedResources tolerates partial errors (e.g. stale API groups)
	resourceLists, err := c.clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		fmt.Printf("Warning: partial API discovery error (will continue with discovered resources): %v\n", err)
	}

	for _, rl := range resourceLists {
		gv, parseErr := schema.ParseGroupVersion(rl.GroupVersion)
		if parseErr != nil {
			continue
		}

		for _, r := range rl.APIResources {
			if !r.Namespaced || !containsVerb(r.Verbs, "list") || !containsVerb(r.Verbs, "update") {
				continue
			}

			gvr := gv.WithResource(r.Name)
			list, listErr := c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if listErr != nil {
				// Resource may not exist or we lack permission — skip silently
				continue
			}

			for i := range list.Items {
				obj := &list.Items[i]
				finalizers := obj.GetFinalizers()
				if len(finalizers) == 0 {
					continue
				}
				fmt.Printf("  Removing finalizers from %s/%s %q: %v\n",
					gv.Group, r.Name, obj.GetName(), finalizers)
				obj.SetFinalizers([]string{})
				if _, updateErr := c.dynamic.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{}); updateErr != nil {
					fmt.Printf("  Warning: could not remove finalizers from %s/%s %q: %v\n",
						gv.Group, r.Name, obj.GetName(), updateErr)
				}
			}
		}
	}
	return nil
}

// forceDeleteNamespace clears namespace spec.finalizers via the finalize subresource,
// equivalent to: kubectl replace --raw /api/v1/namespaces/<name>/finalize
func (c *Client) forceDeleteNamespace(ctx context.Context, namespace string) error {
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil // already gone
		}
		return fmt.Errorf("get namespace for force-finalize: %w", err)
	}
	ns.Spec.Finalizers = []corev1.FinalizerName{}
	_, err = c.clientset.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("finalize namespace: %w", err)
	}
	fmt.Printf("Force-finalized namespace %q\n", namespace)
	return nil
}

// WaitForNamespaceDeletion waits for a namespace to be fully deleted.
// While the namespace is Terminating it periodically removes finalizers from any
// resources blocking deletion.  If the namespace has not terminated by the
// deadline it attempts a force-finalize as a last resort.
func (c *Client) WaitForNamespaceDeletion(ctx context.Context, namespace string, timeout time.Duration) error {
	fmt.Printf("Waiting up to %s for namespace %q to terminate\n", timeout, namespace)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Printf("Namespace %q terminated successfully\n", namespace)
				return nil
			}
			return fmt.Errorf("error checking namespace status: %w", err)
		}

		fmt.Printf("  Namespace %q still terminating, removing stuck finalizers... (%ds remaining)\n",
			namespace, int(time.Until(deadline).Seconds()))
		if err := c.RemoveFinalizersFromNamespace(ctx, namespace); err != nil {
			fmt.Printf("Warning: failed to remove some finalizers: %v\n", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	// Last resort: patch namespace spec.finalizers via the finalize subresource
	fmt.Printf("Timeout reached, force-finalizing namespace %q\n", namespace)
	return c.forceDeleteNamespace(ctx, namespace)
}

// CleanupTerminatingNamespace removes all resource finalizers inside a namespace
// that is stuck in Terminating and then force-clears spec.finalizers so
// Kubernetes can complete the deletion.  It is a no-op if the namespace does
// not exist or is not in the Terminating phase, so it is safe to call as a
// best-effort fallback (e.g. after an interrupted rook-ceph uninstall).
func (c *Client) CleanupTerminatingNamespace(ctx context.Context, namespace string) error {
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil // already gone
		}
		return fmt.Errorf("get namespace %s: %w", namespace, err)
	}
	if ns.Status.Phase != corev1.NamespaceTerminating {
		return nil // not our business
	}

	fmt.Printf("Cleaning up stuck Terminating namespace %q\n", namespace)

	// Remove finalizers from all resources (tolerates partial discovery errors).
	if err := c.RemoveFinalizersFromNamespace(ctx, namespace); err != nil {
		fmt.Printf("Warning: failed to remove some finalizers from %q: %v\n", namespace, err)
	}

	// Force-clear spec.finalizers so the namespace controller releases it.
	return c.forceDeleteNamespace(ctx, namespace)
}

func containsVerb(verbs metav1.Verbs, target string) bool {
	for _, v := range verbs {
		if v == target {
			return true
		}
	}
	return false
}
