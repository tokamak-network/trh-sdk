package utils

import (
	"strings"
)

// GetHelmReleases fetches the list of Helm releases in the given namespace
func GetHelmReleases(namespace string) ([]string, error) {
	output, err := ExecuteCommand("helm", "list", "--namespace", namespace, "-q")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// HelmReleaseExists checks if a release name exists in the list
func HelmReleaseExists(namespace string, releaseName string) (bool, error) {
	releases, err := GetHelmReleases(namespace)
	if err != nil {
		return false, err
	}
	for _, r := range releases {
		if r == releaseName {
			return true, nil
		}
	}
	return false, nil
}
