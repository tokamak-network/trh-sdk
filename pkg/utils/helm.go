package utils

import (
	"fmt"
	"strings"
	"time"
)

// GetHelmReleases fetches the list of Helm releases in the given namespace
func GetHelmReleases(namespace string) ([]string, error) {
	if namespace == "" {
		return nil, nil
	}
	output, err := ExecuteCommand("helm", "list", "--namespace", namespace, "-q")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

func FilterHelmReleases(namespace string, releaseName string) ([]string, error) {
	releases, err := GetHelmReleases(namespace)
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

func CheckK8sReady(namespace string) (bool, error) {
	// TODO: these values can be adjusted if it is not enough
	maxRetries := 10
	retryInterval := 20 * time.Second

	var isPVCReady, isAPiReady bool
	var err error

	for i := 0; i < maxRetries; i++ {
		isPVCReady, err = CheckPVCStatus(namespace)
		if err != nil {
			fmt.Printf("PVC status check failed (attempt %d/%d): %v\n", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		isAPiReady, err = CheckK8sApiHealth(namespace)
		if err != nil {
			fmt.Printf("K8s API health check failed (attempt %d/%d): %v\n", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		if isPVCReady && isAPiReady {
			return true, nil
		}

		fmt.Printf("K8s not ready yet. Retrying in %v... (attempt %d/%d)\n", retryInterval, i+1, maxRetries)
		time.Sleep(retryInterval)
	}

	return false, fmt.Errorf("K8s not ready after %d attempts", maxRetries)
}

func CheckK8sApiHealth(namespace string) (bool, error) {
	apiHealth, err := ExecuteCommand("kubectl", "get", "--raw=/readyz")
	if err != nil {
		return false, err
	}
	return apiHealth == "ok", nil
}

func CheckPVCStatus(namespace string) (bool, error) {
	cmd := []string{
		"get", "pvc",
		"-n", namespace,
		"-o", "jsonpath={range .items[*]}{.status.phase}{\"\\n\"}{end}",
	}

	output, err := ExecuteCommand("kubectl", cmd...)
	if err != nil {
		return false, fmt.Errorf("failed to get PVC status: %w", err)
	}
	fmt.Println("PVC status:", output)

	// Split output into lines and check each PVC status
	pvcStatuses := strings.Split(strings.TrimSpace(output), "\n")

	if len(pvcStatuses) == 0 {
		return false, fmt.Errorf("no PVCs found in namespace %s", namespace)
	}

	for _, status := range pvcStatuses {
		if status != "Bound" {
			fmt.Printf("❌ Found PVC with status: %s\n", status)
			return false, nil
		}
	}

	fmt.Println("✅ All PVCs are bound")

	return true, nil
}
