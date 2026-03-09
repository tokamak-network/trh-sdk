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

// k8sAlertManagerPodConfigSecretName returns the name of the secret mounted as
// "config-volume" in the first alertmanager pod in the given namespace.
func (t *ThanosStack) k8sAlertManagerPodConfigSecretName(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "pods", namespace, "app.kubernetes.io/name=alertmanager")
		if err != nil {
			return "", fmt.Errorf("k8sAlertManagerPodConfigSecretName: %w", err)
		}
		var list struct {
			Items []struct {
				Spec struct {
					Volumes []struct {
						Name   string `json:"name"`
						Secret *struct {
							SecretName string `json:"secretName"`
						} `json:"secret"`
					} `json:"volumes"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sAlertManagerPodConfigSecretName: failed to parse pod list JSON: %w", err)
		}
		if len(list.Items) > 0 {
			for _, vol := range list.Items[0].Spec.Volumes {
				if vol.Name == "config-volume" && vol.Secret != nil {
					return vol.Secret.SecretName, nil
				}
			}
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", namespace, "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sAlertManagerResourceConfigSecret returns the spec.configSecret field from
// the first AlertManager resource in the given namespace.
func (t *ThanosStack) k8sAlertManagerResourceConfigSecret(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "alertmanager", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sAlertManagerResourceConfigSecret: %w", err)
		}
		var list struct {
			Items []struct {
				Spec struct {
					ConfigSecret string `json:"configSecret"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sAlertManagerResourceConfigSecret: failed to parse alertmanager list JSON: %w", err)
		}
		if len(list.Items) > 0 {
			return list.Items[0].Spec.ConfigSecret, nil
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "alertmanager", "-n", namespace, "-o", "jsonpath={.items[0].spec.configSecret}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sAlertManagerGeneratedSecretName returns the name of the prometheus-operator
// generated secret that contains "alertmanager" and "generated" in its name.
func (t *ThanosStack) k8sAlertManagerGeneratedSecretName(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "secret", namespace, "managed-by=prometheus-operator")
		if err != nil {
			return "", fmt.Errorf("k8sAlertManagerGeneratedSecretName: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sAlertManagerGeneratedSecretName: failed to parse secret list JSON: %w", err)
		}
		for _, item := range list.Items {
			name := item.Metadata.Name
			if strings.Contains(name, "alertmanager") && strings.Contains(name, "generated") {
				return name, nil
			}
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", namespace, "-l", "managed-by=prometheus-operator", "-o", "jsonpath={.items[?(@.metadata.name contains 'alertmanager' && @.metadata.name contains 'generated')].metadata.name}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sSecretDataField returns the value of a single data key from a named secret.
// The field parameter is the actual map key (e.g. "alertmanager.yaml.gz").
// In the shellout path the dots are escaped for jsonpath automatically.
func (t *ThanosStack) k8sSecretDataField(ctx context.Context, namespace, secretName, field string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.Get(ctx, "secret", secretName, namespace)
		if err != nil {
			return "", fmt.Errorf("k8sSecretDataField: %w", err)
		}
		var obj struct {
			Data map[string]string `json:"data"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sSecretDataField: failed to parse secret JSON: %w", err)
		}
		return obj.Data[field], nil
	}
	// Escape dots in the field name for jsonpath syntax.
	escapedField := strings.ReplaceAll(field, ".", "\\.")
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", namespace, secretName, "-o", fmt.Sprintf("jsonpath={.data.%s}", escapedField))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sPrometheusRuleYAML returns the full YAML of all PrometheusRule resources in namespace.
func (t *ThanosStack) k8sPrometheusRuleYAML(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "prometheusrule", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleYAML: %w", err)
		}
		// Return JSON; callers using YAML must accept JSON-compatible YAML.
		return string(raw), nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", namespace, "-o", "yaml")
	if err != nil {
		return "", fmt.Errorf("k8sPrometheusRuleYAML: %w", err)
	}
	return out, nil
}

// k8sPrometheusRuleName returns the metadata.name of the first PrometheusRule in namespace.
func (t *ThanosStack) k8sPrometheusRuleName(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "prometheusrule", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleName: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleName: failed to parse prometheusrule list JSON: %w", err)
		}
		if len(list.Items) > 0 {
			return list.Items[0].Metadata.Name, nil
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sPrometheusRuleAlertNames returns the alert names from the first group of
// the first PrometheusRule in namespace (space-separated in shellout path, parsed in k8s path).
func (t *ThanosStack) k8sPrometheusRuleAlertNames(ctx context.Context, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "prometheusrule", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sPrometheusRuleAlertNames: %w", err)
		}
		var list struct {
			Items []struct {
				Spec struct {
					Groups []struct {
						Rules []struct {
							Alert string `json:"alert"`
						} `json:"rules"`
					} `json:"groups"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sPrometheusRuleAlertNames: failed to parse prometheusrule list JSON: %w", err)
		}
		var names []string
		if len(list.Items) > 0 && len(list.Items[0].Spec.Groups) > 0 {
			for _, rule := range list.Items[0].Spec.Groups[0].Rules {
				names = append(names, rule.Alert)
			}
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", namespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

// k8sPrometheusRuleAlertExpr returns the expr field of the rule at the given index
// in the first group of the first PrometheusRule.
func (t *ThanosStack) k8sPrometheusRuleAlertExpr(ctx context.Context, namespace string, index int) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "prometheusrule", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleAlertExpr: %w", err)
		}
		var list struct {
			Items []struct {
				Spec struct {
					Groups []struct {
						Rules []struct {
							Expr string `json:"expr"`
						} `json:"rules"`
					} `json:"groups"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleAlertExpr: failed to parse prometheusrule list JSON: %w", err)
		}
		if len(list.Items) > 0 && len(list.Items[0].Spec.Groups) > 0 {
			rules := list.Items[0].Spec.Groups[0].Rules
			if index >= 0 && index < len(rules) {
				return rules[index].Expr, nil
			}
		}
		return "", nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", namespace, "-o", fmt.Sprintf("jsonpath={.items[0].spec.groups[0].rules[%d].expr}", index))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sPatchPrometheusRuleJSON applies a JSON patch to the named PrometheusRule resource.
func (t *ThanosStack) k8sPatchPrometheusRuleJSON(ctx context.Context, name, namespace string, patch []byte) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Patch(ctx, "prometheusrule", name, namespace, patch)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "prometheusrule", name, "-n", namespace, "--type=json", "-p", string(patch))
	return err
}

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

// k8sPatchSecretMerge applies a JSON merge-patch to a named Secret resource.
func (t *ThanosStack) k8sPatchSecretMerge(ctx context.Context, name, namespace string, patch []byte) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Patch(ctx, "secret", name, namespace, patch)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", name, "-n", namespace, "--type", "merge", "-p", string(patch))
	return err
}

// k8sDeletePodsByLabel deletes all pods in namespace matching labelSelector, ignoring not-found.
// The k8sRunner path deletes pods one by one (list then delete each).
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
// The k8sRunner path falls back to shellout because the K8sRunner interface does
// not expose raw API calls.
func (t *ThanosStack) k8sReplaceNamespaceFinalize(ctx context.Context, namespace, filePath string) error {
	_, err := utils.ExecuteCommand(ctx, "kubectl", "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", filePath)
	return err
}

// k8sDeleteNamespace deletes the named namespace.
// The k8sRunner path falls back to shellout because the K8sRunner Delete method
// requires a resource name but namespace deletion is namespace-scoped itself.
func (t *ThanosStack) k8sDeleteNamespace(ctx context.Context, namespace string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, "namespace", namespace, "", false)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace)
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

// k8sDeletePV deletes a cluster-scoped PersistentVolume, ignoring not-found errors.
func (t *ThanosStack) k8sDeletePV(ctx context.Context, name string) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Delete(ctx, "pv", name, "", true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", name, "--ignore-not-found=true")
	return err
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
		return "", nil
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

// k8sListConfigMapNamesByLabel returns the names of ConfigMaps in the namespace
// matching the given label selector.
func (t *ThanosStack) k8sListConfigMapNamesByLabel(ctx context.Context, namespace, labelSelector string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "configmaps", namespace, labelSelector)
		if err != nil {
			return nil, nil
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListConfigMapNamesByLabel: failed to parse configmap list JSON: %w", err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "configmap", "-n", namespace,
		"-l", labelSelector, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return nil, nil
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if n := strings.TrimSpace(line); n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

// k8sListResourceNames returns the names of all resources of the given type in the namespace.
func (t *ThanosStack) k8sListResourceNames(ctx context.Context, resource, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, resource, namespace, "")
		if err != nil {
			return nil, nil
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
		return nil, nil
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), " "), nil
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
