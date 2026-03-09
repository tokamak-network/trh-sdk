package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// pvEntry holds the name and status phase of a PersistentVolume.
type pvEntry struct {
	Name  string
	Phase string
}

// k8sPVCPhase returns the status phase of the named PVC, or "" when not found.
// When k8sRunner is set it parses the JSON response; otherwise it falls back
// to kubectl with --ignore-not-found.
func (t *ThanosStack) k8sPVCPhase(ctx context.Context, name, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "pvc", name, namespace)
		if err != nil {
			// treat not-found as empty phase
			return "", nil
		}
		var obj struct {
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sPVCPhase: failed to parse PVC JSON: %w", err)
		}
		return obj.Status.Phase, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", name, "-n", namespace, "-o", "jsonpath={.status.phase}", "--ignore-not-found=true")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sPVCNames returns the metadata.name of every PVC in the namespace.
func (t *ThanosStack) k8sPVCNames(ctx context.Context, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pvc", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sPVCNames: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sPVCNames: failed to parse PVC list JSON: %w", err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return nil, fmt.Errorf("k8sPVCNames: %w", err)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if n := strings.TrimSpace(line); n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

// k8sPodPVCClaims returns all PVC claim names referenced by pods in the namespace.
func (t *ThanosStack) k8sPodPVCClaims(ctx context.Context, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pods", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sPodPVCClaims: %w", err)
		}
		var list struct {
			Items []struct {
				Spec struct {
					Volumes []struct {
						PVC *struct {
							ClaimName string `json:"claimName"`
						} `json:"persistentVolumeClaim"`
					} `json:"volumes"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sPodPVCClaims: failed to parse pod list JSON: %w", err)
		}
		var claims []string
		for _, pod := range list.Items {
			for _, vol := range pod.Spec.Volumes {
				if vol.PVC != nil && vol.PVC.ClaimName != "" {
					claims = append(claims, vol.PVC.ClaimName)
				}
			}
		}
		return claims, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", namespace, "-o", "jsonpath={.items[*].spec.volumes[*].persistentVolumeClaim.claimName}")
	if err != nil {
		return nil, fmt.Errorf("k8sPodPVCClaims: %w", err)
	}
	var claims []string
	for _, c := range strings.Fields(out) {
		if c != "" {
			claims = append(claims, c)
		}
	}
	return claims, nil
}

// k8sPVList returns all PersistentVolumes with their name and status phase.
// PVs are cluster-scoped so no namespace is required.
func (t *ThanosStack) k8sPVList(ctx context.Context) ([]pvEntry, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pv", "", "")
		if err != nil {
			return nil, fmt.Errorf("k8sPVList: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Status struct {
					Phase string `json:"phase"`
				} `json:"status"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sPVList: failed to parse PV list JSON: %w", err)
		}
		entries := make([]pvEntry, 0, len(list.Items))
		for _, item := range list.Items {
			entries = append(entries, pvEntry{Name: item.Metadata.Name, Phase: item.Status.Phase})
		}
		return entries, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
	if err != nil {
		return nil, fmt.Errorf("k8sPVList: %w", err)
	}
	var entries []pvEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			entries = append(entries, pvEntry{Name: parts[0], Phase: parts[1]})
		}
	}
	return entries, nil
}

// k8sDeletePVC deletes the named PVC, ignoring not-found errors.
func (t *ThanosStack) k8sDeletePVC(ctx context.Context, name, namespace string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, "pvc", name, namespace, true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", name, "-n", namespace, "--ignore-not-found=true")
	return err
}

// k8sPatchPV applies a JSON merge-patch to a cluster-scoped PersistentVolume.
func (t *ThanosStack) k8sPatchPV(ctx context.Context, name string, patch []byte) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Patch(ctx, "pv", name, "", patch)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", name, "-p", string(patch), "--type=merge")
	return err
}

// k8sDeletePV deletes a cluster-scoped PersistentVolume, ignoring not-found errors.
func (t *ThanosStack) k8sDeletePV(ctx context.Context, name string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, "pv", name, "", true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", name, "--ignore-not-found=true")
	return err
}
