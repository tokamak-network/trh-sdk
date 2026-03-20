package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShellOutDORunner implements DORunner by shelling out to the doctl binary.
// It preserves the exact existing behaviour and serves as a fallback when
// NativeDORunner is not available or when --legacy mode is requested.
type ShellOutDORunner struct{}

// doRegion matches the JSON structure returned by "doctl compute region list --output json".
type doRegion struct {
	Slug      string `json:"slug"`
	Available bool   `json:"available"`
}

// ValidateToken checks if the provided DO token is valid using doctl.
func (r *ShellOutDORunner) ValidateToken(ctx context.Context, token string) error {
	_, err := utils.ExecuteCommand(ctx, "doctl", "account", "get", "--access-token", token, "--no-header")
	if err != nil {
		return fmt.Errorf("do validate token: %w", err)
	}
	return nil
}

// ListRegions returns slugs for all available DO regions using doctl.
func (r *ShellOutDORunner) ListRegions(ctx context.Context, token string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "doctl", "compute", "region", "list", "--output", "json", "--access-token", token)
	if err != nil {
		return nil, fmt.Errorf("do list regions: %w", err)
	}

	var regions []doRegion
	if err := json.Unmarshal([]byte(out), &regions); err != nil {
		return nil, fmt.Errorf("do list regions: parse output: %w", err)
	}

	var slugs []string
	for _, region := range regions {
		if region.Available {
			slugs = append(slugs, region.Slug)
		}
	}
	return slugs, nil
}

// GetKubeconfig saves the kubeconfig for a DOKS cluster using doctl.
func (r *ShellOutDORunner) GetKubeconfig(ctx context.Context, clusterName, token string) error {
	out, err := utils.ExecuteCommand(ctx, "doctl", "kubernetes", "cluster", "kubeconfig", "save", clusterName, "--access-token", token)
	if err != nil {
		return fmt.Errorf("do get kubeconfig: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}

// CheckVersion verifies that doctl is installed and available.
func (r *ShellOutDORunner) CheckVersion(ctx context.Context) error {
	_, err := utils.ExecuteCommand(ctx, "doctl", "version")
	if err != nil {
		return fmt.Errorf("do check version: %w", err)
	}
	return nil
}
