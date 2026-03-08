package digitalocean

import (
	"context"
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

// ValidateToken checks if the given DigitalOcean API token is valid.
// Uses "doctl account get" which is read-only and does NOT modify the local doctl config.
func ValidateToken(ctx context.Context, token string) error {
	cmd := exec.CommandContext(ctx, "doctl", "account", "get", "--access-token", token, "--no-header")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("invalid DigitalOcean token: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GetRegions returns all DigitalOcean regions. Callers should check Region.Available
// before using a region. Use IsValidRegion for availability validation.
func GetRegions(ctx context.Context, token string) ([]Region, error) {
	cmd := exec.CommandContext(ctx, "doctl", "compute", "region", "list", "--output", "json", "--access-token", token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list DigitalOcean regions: %s", strings.TrimSpace(string(output)))
	}

	var regions []Region
	if err := json.Unmarshal(output, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse region list: %w", err)
	}

	return regions, nil
}

// IsValidRegion checks whether the given region slug is available.
// Pass a pre-fetched region list to avoid redundant API calls.
func IsValidRegion(regions []Region, region string) bool {
	for _, r := range regions {
		if r.Slug == region && r.Available {
			return true
		}
	}
	return false
}

// SaveKubeconfig saves the kubeconfig for a DOKS cluster using doctl.
func SaveKubeconfig(ctx context.Context, token, clusterName string) error {
	cmd := exec.CommandContext(ctx, "doctl", "kubernetes", "cluster", "kubeconfig", "save", clusterName, "--access-token", token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: %s", clusterName, strings.TrimSpace(string(output)))
	}
	return nil
}
