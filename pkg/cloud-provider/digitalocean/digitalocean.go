package digitalocean

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Region struct {
	Slug      string `json:"slug"`
	Name      string `json:"Name"`
	Available bool   `json:"available"`
}

type regionListResponse struct {
	Regions []Region `json:"regions"`
}

// ValidateToken checks if the given DigitalOcean API token is valid by calling doctl.
func ValidateToken(token string) error {
	cmd := exec.Command("doctl", "auth", "init", "--access-token", token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("invalid DigitalOcean token: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GetAvailableRegions returns a list of available DigitalOcean region slugs.
func GetAvailableRegions(token string) ([]string, error) {
	cmd := exec.Command("doctl", "compute", "region", "list", "--output", "json", "--access-token", token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list DigitalOcean regions: %s", strings.TrimSpace(string(output)))
	}

	var regions []Region
	if err := json.Unmarshal(output, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse region list: %w", err)
	}

	slugs := make([]string, 0, len(regions))
	for _, r := range regions {
		if r.Available {
			slugs = append(slugs, r.Slug)
		}
	}
	return slugs, nil
}

// IsValidRegion checks whether the given region slug is available on DigitalOcean.
func IsValidRegion(token, region string) (bool, error) {
	regions, err := GetAvailableRegions(token)
	if err != nil {
		return false, err
	}
	for _, r := range regions {
		if r == region {
			return true, nil
		}
	}
	return false, nil
}

// SaveKubeconfig saves the kubeconfig for a DOKS cluster using doctl.
func SaveKubeconfig(token, clusterName string) error {
	cmd := exec.Command("doctl", "kubernetes", "cluster", "kubeconfig", "save", clusterName, "--access-token", token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: %s", clusterName, strings.TrimSpace(string(output)))
	}
	return nil
}
