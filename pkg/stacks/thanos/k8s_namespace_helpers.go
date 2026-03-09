package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// k8sGetNamespaceJSON returns the raw JSON of a namespace resource.
func (t *ThanosStack) k8sGetNamespaceJSON(ctx context.Context, namespace string) ([]byte, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "namespace", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sGetNamespaceJSON: %w", err)
		}
		return raw, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "-o", "json")
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// k8sReplaceNamespaceFinalize sends a raw PUT to /api/v1/namespaces/<ns>/finalize.
// Always shells out — the K8sRunner interface does not expose raw API calls.
func (t *ThanosStack) k8sReplaceNamespaceFinalize(ctx context.Context, namespace, filePath string) error {
	_, err := utils.ExecuteCommand(ctx, "kubectl", "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", filePath)
	return err
}

// k8sDeleteNamespace deletes the named namespace.
func (t *ThanosStack) k8sDeleteNamespace(ctx context.Context, namespace string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, "namespace", namespace, "", false)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace)
	return err
}

// k8sEnsureNamespace creates the namespace if it does not already exist.
func (t *ThanosStack) k8sEnsureNamespace(ctx context.Context, namespace string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.EnsureNamespace(ctx, namespace)
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return fmt.Errorf("k8sEnsureNamespace: failed to check namespace existence: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace); err != nil {
			return fmt.Errorf("k8sEnsureNamespace: failed to create namespace: %w", err)
		}
	}
	return nil
}
