package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

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

// k8sPrometheusRuleOutput returns the serialized PrometheusRule list for namespace.
// When k8sRunner is set the output is JSON; the shellout path requests YAML via kubectl.
// JSON is a strict subset of YAML so downstream parsers that accept YAML will handle both.
func (t *ThanosStack) k8sPrometheusRuleYAML(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner != nil {
		raw, err := t.k8sRunner.List(ctx, "prometheusrule", namespace, "")
		if err != nil {
			return "", fmt.Errorf("k8sPrometheusRuleYAML: %w", err)
		}
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

// k8sPatchSecretMerge applies a JSON merge-patch to a named Secret resource.
func (t *ThanosStack) k8sPatchSecretMerge(ctx context.Context, name, namespace string, patch []byte) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Patch(ctx, "secret", name, namespace, patch)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", name, "-n", namespace, "--type", "merge", "-p", string(patch))
	return err
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
