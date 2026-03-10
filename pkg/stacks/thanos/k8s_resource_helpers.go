package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// k8sDeletePodsByLabel deletes all pods in namespace matching labelSelector, ignoring not-found.
// The k8sRunner path issues one Delete call per pod (N calls for N pods). The shellout path uses a single label-select delete.
func (t *ThanosStack) k8sDeletePodsByLabel(ctx context.Context, namespace, labelSelector string) error {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pods", namespace, labelSelector)
		if err != nil {
			return fmt.Errorf("k8sDeletePodsByLabel: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return fmt.Errorf("k8sDeletePodsByLabel: failed to parse pod list JSON: %w", err)
		}
		for _, item := range list.Items {
			if err := t.k8sRunner.Delete(ctx, "pod", item.Metadata.Name, namespace, true); err != nil {
				return fmt.Errorf("k8sDeletePodsByLabel: %w", err)
			}
		}
		return nil
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pod", "-n", namespace, "-l", labelSelector, "--ignore-not-found=true")
	return err
}

// k8sApplyManifest applies a YAML/JSON manifest using k8sRunner when available,
// otherwise falls back to writing a temp file and running kubectl apply -f.
func (t *ThanosStack) k8sApplyManifest(ctx context.Context, manifest string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Apply(ctx, []byte(manifest))
	}
	return t.applyManifestWithTempFile(ctx, manifest, "k8s-manifest-*.yaml")
}

// k8sDeleteResource deletes a namespaced or cluster-scoped resource, ignoring not-found.
// Pass namespace="" for cluster-scoped resources.
func (t *ThanosStack) k8sDeleteResource(ctx context.Context, resource, name, namespace string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, resource, name, namespace, true)
	}
	args := []string{"delete", resource, name, "--ignore-not-found=true"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	return err
}

// k8sGetPodNameByLabel returns the first pod name matching the label selector,
// or "" when no matching pod is found.
func (t *ThanosStack) k8sGetPodNameByLabel(ctx context.Context, namespace, labelSelector string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pods", namespace, labelSelector)
		if err != nil {
			return "", fmt.Errorf("k8sGetPodNameByLabel: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sGetPodNameByLabel: failed to parse pod list JSON: %w", err)
		}
		if len(list.Items) == 0 {
			return "", nil
		}
		return list.Items[0].Metadata.Name, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", "-n", namespace,
		"-l", labelSelector, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return "", fmt.Errorf("k8sGetPodNameByLabel: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", nil
	}
	return strings.TrimSpace(lines[0]), nil
}

// k8sGetPodJSON returns the raw JSON for the named pod in the given namespace.
func (t *ThanosStack) k8sGetPodJSON(ctx context.Context, name, namespace string) ([]byte, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "pod", name, namespace)
		if err != nil {
			return nil, fmt.Errorf("k8sGetPodJSON: %w", err)
		}
		return raw, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", name, "-n", namespace, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("k8sGetPodJSON: %w", err)
	}
	return []byte(out), nil
}

// k8sListResourceNames returns the names of all resources of the given type in the namespace.
func (t *ThanosStack) k8sListResourceNames(ctx context.Context, resource, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, resource, namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sListResourceNames: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListResourceNames: failed to parse %s list JSON: %w", resource, err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", resource, "-n", namespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, fmt.Errorf("k8sListResourceNames: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), " "), nil
}
