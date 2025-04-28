package utils

import (
	"fmt"
	"strings"
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
	pvcStatus, err := CheckPVCStatus(namespace)
	if err != nil {
		return false, err
	}
	apiHealth, err := CheckK8sApiHealth(namespace)
	if err != nil {
		return false, err
	}
	return pvcStatus && apiHealth, nil
}

func CheckK8sApiHealth(namespace string) (bool, error) {
	apiHealth, err := ExecuteCommand("kubectl", "get", "--raw=/readyz?verbose")
	if err != nil {
		return false, err
	}
	return apiHealth != "", nil
}

func CheckPVCStatus(namespace string) (bool, error) {
	pvcStatus, err := ExecuteCommand("bash", "-c", fmt.Sprintf("if kubectl get pvc -n %s -o jsonpath='{range .items[*]}{.status.phase}{\"\\n\"}{end}' | grep -qv Bound; then echo 'There are unbound PVCs'; else echo 'Bound'; fi", namespace))
	fmt.Println("PVC status:", pvcStatus)
	if err != nil {
		return false, err
	}
	return pvcStatus == "Bound", nil
}
