package runner

import "context"

// DORunner defines DigitalOcean operations used across TRH SDK.
// It replaces 4 doctl subprocess calls.
type DORunner interface {
	// ValidateToken checks if the provided DO token is valid.
	ValidateToken(ctx context.Context, token string) error

	// ListRegions returns available DO regions as slug strings.
	ListRegions(ctx context.Context, token string) ([]string, error)

	// GetKubeconfig saves the kubeconfig for the given DOKS cluster.
	GetKubeconfig(ctx context.Context, clusterName, token string) error

	// CheckVersion verifies doctl is available (for legacy mode only).
	CheckVersion(ctx context.Context) error
}
