package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShellOutHelmRunner implements HelmRunner by shelling out to the helm binary.
// It preserves the exact existing behaviour and serves as a fallback when
// NativeHelmRunner is not available or when --legacy mode is requested.
type ShellOutHelmRunner struct{}

// Install installs a Helm chart via helm install.
func (r *ShellOutHelmRunner) Install(ctx context.Context, release, chart, namespace string, vals map[string]interface{}) error {
	if release == "" {
		return fmt.Errorf("shellout helm install: release name cannot be empty")
	}
	if namespace == "" {
		return fmt.Errorf("shellout helm install: namespace cannot be empty")
	}
	args := []string{"install", release, chart, "--namespace", namespace}
	args = append(args, flattenValues(vals)...)

	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	if err != nil {
		return fmt.Errorf("shellout helm install %s: %w", release, err)
	}
	return nil
}

// Upgrade performs helm upgrade --install via the helm binary.
func (r *ShellOutHelmRunner) Upgrade(ctx context.Context, release, chart, namespace string, vals map[string]interface{}) error {
	args := []string{"upgrade", "--install", release, chart, "--namespace", namespace}
	args = append(args, flattenValues(vals)...)

	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	if err != nil {
		return fmt.Errorf("shellout helm upgrade %s: %w", release, err)
	}
	return nil
}

// UpgradeWithFiles performs helm upgrade --install using values file paths.
func (r *ShellOutHelmRunner) UpgradeWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error {
	args := []string{"upgrade", "--install", release, chart, "--namespace", namespace}
	for _, f := range valueFiles {
		args = append(args, "--values", f)
	}

	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	if err != nil {
		return fmt.Errorf("shellout helm upgrade-with-files %s: %w", release, err)
	}
	return nil
}

// Uninstall removes a Helm release via helm uninstall.
func (r *ShellOutHelmRunner) Uninstall(ctx context.Context, release, namespace string) error {
	if release == "" {
		return fmt.Errorf("shellout helm uninstall: release name cannot be empty")
	}
	if namespace == "" {
		return fmt.Errorf("shellout helm uninstall: namespace cannot be empty")
	}
	_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
	if err != nil {
		return fmt.Errorf("shellout helm uninstall %s: %w", release, err)
	}
	return nil
}

// List returns the names of all releases in a namespace via helm list -q.
func (r *ShellOutHelmRunner) List(ctx context.Context, namespace string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "helm", "list", "--namespace", namespace, "-q")
	if err != nil {
		return nil, fmt.Errorf("shellout helm list namespace %s: %w", namespace, err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return []string{}, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// RepoAdd adds a Helm chart repository via helm repo add.
func (r *ShellOutHelmRunner) RepoAdd(ctx context.Context, name, url string) error {
	_, err := utils.ExecuteCommand(ctx, "helm", "repo", "add", name, url)
	if err != nil {
		return fmt.Errorf("shellout helm repo add %s: %w", name, err)
	}
	return nil
}

// RepoUpdate updates all Helm chart repositories via helm repo update.
func (r *ShellOutHelmRunner) RepoUpdate(ctx context.Context) error {
	_, err := utils.ExecuteCommand(ctx, "helm", "repo", "update")
	if err != nil {
		return fmt.Errorf("shellout helm repo update: %w", err)
	}
	return nil
}

// DependencyUpdate updates chart dependencies via helm dependency update.
func (r *ShellOutHelmRunner) DependencyUpdate(ctx context.Context, chartPath string) error {
	_, err := utils.ExecuteCommand(ctx, "helm", "dependency", "update", chartPath)
	if err != nil {
		return fmt.Errorf("shellout helm dependency update %s: %w", chartPath, err)
	}
	return nil
}

// Status returns the status of a release via helm status.
func (r *ShellOutHelmRunner) Status(ctx context.Context, release, namespace string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "helm", "status", release, "--namespace", namespace)
	if err != nil {
		return "", fmt.Errorf("shellout helm status %s: %w", release, err)
	}
	return strings.TrimSpace(out), nil
}

// Search searches for charts matching a keyword via helm search repo.
func (r *ShellOutHelmRunner) Search(ctx context.Context, keyword string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "helm", "search", "repo", keyword)
	if err != nil {
		return "", fmt.Errorf("shellout helm search %s: %w", keyword, err)
	}
	return strings.TrimSpace(out), nil
}

// flattenValues converts a map of Helm values into --set key=value arguments.
// Keys are sorted to produce a deterministic argument order.
func flattenValues(vals map[string]interface{}) []string {
	if len(vals) == 0 {
		return nil
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	args := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, "--set", fmt.Sprintf("%s=%v", k, vals[k]))
	}
	return args
}
