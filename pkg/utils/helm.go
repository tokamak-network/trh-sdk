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
			if i == maxRetries-1 {
				return false, err
			}
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
	pvcStatus, err := ExecuteCommand("bash", "-c", fmt.Sprintf("if kubectl get pvc -n %s -o jsonpath='{range .items[*]}{.status.phase}{\"\\n\"}{end}' | grep -qv Bound; then echo 'There are unbound PVCs'; else echo 'Bound'; fi", namespace))
	fmt.Println("PVC status:", pvcStatus)
	if err != nil {
		return false, err
	}
	return pvcStatus == "Bound", nil
}
