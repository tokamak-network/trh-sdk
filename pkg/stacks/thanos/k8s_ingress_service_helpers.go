package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// k8sFirstIngressName returns the metadata.name of the first Ingress in namespace.
func (t *ThanosStack) k8sFirstIngressName(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "ingress", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sFirstIngressName: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sFirstIngressName: failed to parse ingress list JSON: %w", err)
		}
		if len(list.Items) > 0 {
			return list.Items[0].Metadata.Name, nil
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sGetIngressHostname returns the first loadBalancer hostname for an ingress.
// Returns "" when the ingress is not found or has no hostname yet.
func (t *ThanosStack) k8sGetIngressHostname(ctx context.Context, name, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "ingress", name, namespace)
		if err != nil {
			return "", nil
		}
		var obj struct {
			Status struct {
				LoadBalancer struct {
					Ingress []struct {
						Hostname string `json:"hostname"`
					} `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sGetIngressHostname: failed to parse ingress JSON: %w", err)
		}
		if len(obj.Status.LoadBalancer.Ingress) > 0 {
			return obj.Status.LoadBalancer.Ingress[0].Hostname, nil
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", namespace, name,
		"-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}", "--ignore-not-found=true")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// k8sIngressHasLoadBalancer reports whether an ingress has any loadBalancer entries.
func (t *ThanosStack) k8sIngressHasLoadBalancer(ctx context.Context, name, namespace string) (bool, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "ingress", name, namespace)
		if err != nil {
			return false, nil
		}
		var obj struct {
			Status struct {
				LoadBalancer struct {
					Ingress []struct{} `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return false, fmt.Errorf("k8sIngressHasLoadBalancer: failed to parse ingress JSON: %w", err)
		}
		return len(obj.Status.LoadBalancer.Ingress) > 0, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", namespace, name,
		"-o", "jsonpath={.status.loadBalancer.ingress}", "--ignore-not-found=true")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// pvVolumeHandleEntry holds the name and CSI volumeHandle of a PersistentVolume.
type pvVolumeHandleEntry struct {
	Name         string
	VolumeHandle string
}

// k8sListPVVolumeHandles returns every PV's name and its CSI volumeHandle.
func (t *ThanosStack) k8sListPVVolumeHandles(ctx context.Context) ([]pvVolumeHandleEntry, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pv", "", "")
		if err != nil {
			return nil, fmt.Errorf("k8sListPVVolumeHandles: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Spec struct {
					CSI *struct {
						VolumeHandle string `json:"volumeHandle"`
					} `json:"csi"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListPVVolumeHandles: failed to parse PV list JSON: %w", err)
		}
		entries := make([]pvVolumeHandleEntry, 0, len(list.Items))
		for _, item := range list.Items {
			handle := ""
			if item.Spec.CSI != nil {
				handle = item.Spec.CSI.VolumeHandle
			}
			entries = append(entries, pvVolumeHandleEntry{Name: item.Metadata.Name, VolumeHandle: handle})
		}
		return entries, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv",
		"-o", "custom-columns=NAME:.metadata.name,VOLUMEHANDLE:.spec.csi.volumeHandle", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("k8sListPVVolumeHandles: %w", err)
	}
	var entries []pvVolumeHandleEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			entries = append(entries, pvVolumeHandleEntry{Name: parts[0], VolumeHandle: parts[1]})
		}
	}
	return entries, nil
}

// k8sListServiceNames returns the name of every Service in the namespace.
func (t *ThanosStack) k8sListServiceNames(ctx context.Context, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "services", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sListServiceNames: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListServiceNames: failed to parse service list JSON: %w", err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "services", "-n", namespace,
		"-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("k8sListServiceNames: %w", err)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if n := strings.TrimSpace(line); n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}
