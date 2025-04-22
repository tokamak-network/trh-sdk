package utils

import (
	"strings"
)

// GetHelmReleases fetches the list of Helm releases in the given namespace
func GetHelmReleases(namespace string, logFileName string) ([]string, error) {
	if namespace == "" {
		return nil, nil
	}
	output, err := ExecuteCommand("helm", logFileName, "list", "--namespace", namespace, "-q")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

func FilterHelmReleases(namespace string, releaseName string, logFileName string) ([]string, error) {
	releases, err := GetHelmReleases(namespace, logFileName)
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
