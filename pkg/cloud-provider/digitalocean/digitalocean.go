package digitalocean

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Region struct {
	Slug      string `json:"slug"`
	Name      string `json:"Name"`
	Available bool   `json:"available"`
}

// tokenSource implements oauth2.TokenSource for a static DO API token.
type tokenSource struct{ token string }

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: t.token}, nil
}

// newGodoClient creates a godo.Client authenticated with the given token.
func newGodoClient(token string) *godo.Client {
	oauthClient := oauth2.NewClient(context.Background(), &tokenSource{token: token})
	return godo.NewClient(oauthClient)
}

// ValidateToken checks if the given DigitalOcean API token is valid.
func ValidateToken(ctx context.Context, token string) error {
	client := newGodoClient(token)
	_, _, err := client.Account.Get(ctx)
	if err != nil {
		return fmt.Errorf("invalid DigitalOcean token: %w", err)
	}
	return nil
}

// GetRegions returns all DigitalOcean regions. Callers should check Region.Available
// before using a region. Use IsValidRegion for availability validation.
func GetRegions(ctx context.Context, token string) ([]Region, error) {
	client := newGodoClient(token)

	var allRegions []Region
	opt := &godo.ListOptions{PerPage: 200}

	for {
		regions, resp, err := client.Regions.List(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list DigitalOcean regions: %w", err)
		}
		for _, r := range regions {
			features := make([]string, len(r.Features))
			copy(features, r.Features)
			allRegions = append(allRegions, Region{
				Slug:      r.Slug,
				Name:      r.Name,
				Available: r.Available,
			})
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("failed to list DigitalOcean regions: parse pagination: %w", err)
		}
		opt.Page = page + 1
	}

	return allRegions, nil
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

// SaveKubeconfig saves the kubeconfig for a DOKS cluster.
func SaveKubeconfig(ctx context.Context, token, clusterName string) error {
	client := newGodoClient(token)

	clusterID, err := resolveClusterID(ctx, client, clusterName)
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: %w", clusterName, err)
	}

	kubeConfig, _, err := client.Kubernetes.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: %w", clusterName, err)
	}

	config, err := clientcmd.Load(kubeConfig.KubeconfigYAML)
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: parse: %w", clusterName, err)
	}

	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: %w", clusterName, err)
	}

	// Load existing kubeconfig or start with an empty one.
	existing, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		existing = clientcmdapi.NewConfig()
	}

	// Merge clusters, contexts, and auth info from the downloaded config.
	for k, v := range config.Clusters {
		existing.Clusters[k] = v
	}
	for k, v := range config.Contexts {
		existing.Contexts[k] = v
	}
	for k, v := range config.AuthInfos {
		existing.AuthInfos[k] = v
	}
	if config.CurrentContext != "" {
		existing.CurrentContext = config.CurrentContext
	}

	if err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0700); err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: create dir: %w", clusterName, err)
	}

	if err := clientcmd.WriteToFile(*existing, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to save kubeconfig for cluster %s: write: %w", clusterName, err)
	}
	return nil
}

// resolveClusterID finds a DOKS cluster ID by its name.
func resolveClusterID(ctx context.Context, client *godo.Client, name string) (string, error) {
	opt := &godo.ListOptions{PerPage: 200}

	for {
		clusters, resp, err := client.Kubernetes.List(ctx, opt)
		if err != nil {
			return "", fmt.Errorf("resolve cluster: %w", err)
		}
		for _, c := range clusters {
			if c.Name == name {
				return c.ID, nil
			}
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return "", fmt.Errorf("resolve cluster: parse pagination: %w", err)
		}
		opt.Page = page + 1
	}
	return "", fmt.Errorf("cluster %q not found", name)
}

// defaultKubeconfigPath returns the standard kubeconfig file path.
func defaultKubeconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home dir: %w", err)
	}
	return filepath.Join(home, ".kube", "config"), nil
}
