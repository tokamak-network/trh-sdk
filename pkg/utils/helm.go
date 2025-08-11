package utils

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GetHelmReleases fetches the list of Helm releases in the given namespace
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

func UninstallHelmRelease(ctx context.Context, namespace string, releaseName string) error {
	_, err := ExecuteCommand(ctx, "helm", "uninstall", releaseName, "--namespace", namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall helm release: %v", err)
	}
	return nil
}

// CheckMonitoringPluginInstalled checks if monitoring plugin is installed
func CheckMonitoringPluginInstalled(ctx context.Context) error {
	// Check if monitoring namespace exists
	exists, err := CheckNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check monitoring namespace: %w", err)
	}
	if !exists {
		return fmt.Errorf("monitoring plugin is not installed. Please install it first with 'trh-sdk install monitoring'")
	}

	// Check if monitoring release exists
	releases, err := GetHelmReleases(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check monitoring releases: %w", err)
	}
	if len(releases) == 0 {
		return fmt.Errorf("monitoring plugin is not installed. Please install it first with 'trh-sdk install monitoring'")
	}

	return nil
}
