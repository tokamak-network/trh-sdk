package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// HelmLister abstracts the Helm list operation so that callers can pass either
// a HelmLister or any other implementation without importing the runner
// package (which would cause an import cycle).
type HelmLister interface {
	List(ctx context.Context, namespace string) ([]string, error)
}

// HelmInstaller abstracts Helm install/upgrade/uninstall operations.
type HelmInstaller interface {
	HelmLister
	Uninstall(ctx context.Context, release, namespace string) error
	UpgradeWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error
}

// GetHelmReleases fetches the list of Helm releases in the given namespace.
// Deprecated: Use GetHelmReleasesWithRunner for new code.
func GetHelmReleases(ctx context.Context, namespace string) ([]string, error) {
	if namespace == "" {
		return nil, nil
	}
	output, err := ExecuteCommand(ctx, "helm", "list", "--namespace", namespace, "-q")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetHelmReleasesWithRunner fetches Helm releases using the provided HelmRunner.
func GetHelmReleasesWithRunner(ctx context.Context, hr HelmLister, namespace string) ([]string, error) {
	if namespace == "" {
		return nil, nil
	}
	return hr.List(ctx, namespace)
}

// FilterHelmReleases filters releases by name substring.
// Deprecated: Use FilterHelmReleasesWithRunner for new code.
func FilterHelmReleases(ctx context.Context, namespace string, releaseName string) ([]string, error) {
	releases, err := GetHelmReleases(ctx, namespace)
	if err != nil {
		return nil, err
	}

	helmReleases := make([]string, 0)
	for _, r := range releases {
		if strings.Contains(r, releaseName) {
			helmReleases = append(helmReleases, r)
		}
	}
	return helmReleases, nil
}

// FilterHelmReleasesWithRunner filters releases by name substring using the provided HelmRunner.
func FilterHelmReleasesWithRunner(ctx context.Context, hr HelmLister, namespace string, releaseName string) ([]string, error) {
	releases, err := hr.List(ctx, namespace)
	if err != nil {
		return nil, err
	}

	helmReleases := make([]string, 0)
	for _, r := range releases {
		if strings.Contains(r, releaseName) {
			helmReleases = append(helmReleases, r)
		}
	}
	return helmReleases, nil
}

func CheckK8sReady(ctx context.Context, namespace string) (bool, error) {
	// TODO: these values can be adjusted if it is not enough
	maxRetries := 10
	retryInterval := 20 * time.Second

	var isAPiReady bool
	var err error

	for i := 0; i < maxRetries; i++ {

		isAPiReady, err = CheckK8sApiHealth(ctx, namespace)
		if err != nil {
			fmt.Printf("K8s API health check failed (attempt %d/%d): %v\n", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		if isAPiReady {
			return true, nil
		}

		fmt.Printf("K8s not ready yet. Retrying in %v... (attempt %d/%d)\n", retryInterval, i+1, maxRetries)
		time.Sleep(retryInterval)
	}

	return false, fmt.Errorf("K8s not ready after %d attempts", maxRetries)
}

func CheckK8sApiHealth(ctx context.Context, namespace string) (bool, error) {
	apiHealth, err := ExecuteCommand(ctx, "kubectl", "get", "--raw=/readyz")
	if err != nil {
		return false, err
	}
	return apiHealth == "ok", nil
}

// InstallHelmRelease installs or upgrades a Helm release via shellout.
// Deprecated: Use InstallHelmReleaseWithRunner for new code.
func InstallHelmRelease(ctx context.Context, releaseName string, chartPath string, filePath string, namespace string) error {
	_, err := ExecuteCommand(ctx, "helm", []string{
		"upgrade",
		"--install",
		releaseName,
		chartPath,
		"--values", filePath,
		"--namespace", namespace,
	}...)
	if err != nil {
		return fmt.Errorf("failed to install helm release: %v", err)
	}
	return nil
}

// InstallHelmReleaseWithRunner installs or upgrades a Helm release using the provided HelmInstaller.
func InstallHelmReleaseWithRunner(ctx context.Context, hr HelmInstaller, releaseName string, chartPath string, filePath string, namespace string) error {
	if err := hr.UpgradeWithFiles(ctx, releaseName, chartPath, namespace, []string{filePath}); err != nil {
		return fmt.Errorf("failed to install helm release: %w", err)
	}
	return nil
}

// UninstallHelmRelease uninstalls a Helm release via shellout.
// Deprecated: Use UninstallHelmReleaseWithRunner for new code.
func UninstallHelmRelease(ctx context.Context, namespace string, releaseName string) error {
	_, err := ExecuteCommand(ctx, "helm", "uninstall", releaseName, "--namespace", namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall helm release: %v", err)
	}
	return nil
}

// UninstallHelmReleaseWithRunner uninstalls a Helm release using the provided HelmInstaller.
func UninstallHelmReleaseWithRunner(ctx context.Context, hr HelmInstaller, releaseName string, namespace string) error {
	if err := hr.Uninstall(ctx, releaseName, namespace); err != nil {
		return fmt.Errorf("failed to uninstall helm release: %w", err)
	}
	return nil
}

// CheckMonitoringPluginInstalled checks if monitoring plugin is installed
func CheckMonitoringPluginInstalled(ctx context.Context) error {
	// Check if monitoring namespace exists
	exists, err := CheckNamespaceExists(ctx, constants.MonitoringNamespace)
	if err != nil {
		return fmt.Errorf("failed to check monitoring namespace: %w", err)
	}
	if !exists {
		return fmt.Errorf("monitoring plugin is not installed. Please install it first with 'trh-sdk install monitoring'")
	}

	// Check if monitoring release exists
	releases, err := GetHelmReleases(ctx, constants.MonitoringNamespace)
	if err != nil {
		return fmt.Errorf("failed to check monitoring releases: %w", err)
	}
	if len(releases) == 0 {
		return fmt.Errorf("monitoring plugin is not installed. Please install it first with 'trh-sdk install monitoring'")
	}

	return nil
}
