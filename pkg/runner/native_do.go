package runner

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

// NativeDORunner implements DORunner using github.com/digitalocean/godo.
// No doctl binary is required.
type NativeDORunner struct{}

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

// ValidateToken checks if the provided DO token is valid by calling the Account API.
func (r *NativeDORunner) ValidateToken(ctx context.Context, token string) error {
	client := newGodoClient(token)
	_, _, err := client.Account.Get(ctx)
	if err != nil {
		return fmt.Errorf("do validate token: %w", err)
	}
	return nil
}

// ListRegions returns slugs for all available DO regions.
func (r *NativeDORunner) ListRegions(ctx context.Context, token string) ([]string, error) {
	client := newGodoClient(token)

	var allRegions []string
	opt := &godo.ListOptions{PerPage: 200}

	for {
		regions, resp, err := client.Regions.List(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("do list regions: %w", err)
		}
		for _, region := range regions {
			if region.Available {
				allRegions = append(allRegions, region.Slug)
			}
		}
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("do list regions: parse pagination: %w", err)
		}
		opt.Page = page + 1
	}

	return allRegions, nil
}

// GetKubeconfig fetches and saves the kubeconfig for a DOKS cluster.
// It finds the cluster by name, downloads the kubeconfig, and merges it
// into the default kubeconfig path (~/.kube/config).
func (r *NativeDORunner) GetKubeconfig(ctx context.Context, clusterName, token string) error {
	client := newGodoClient(token)

	clusterID, err := resolveClusterID(ctx, client, clusterName)
	if err != nil {
		return err
	}

	kubeConfig, _, err := client.Kubernetes.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("do get kubeconfig: %w", err)
	}

	config, err := clientcmd.Load(kubeConfig.KubeconfigYAML)
	if err != nil {
		return fmt.Errorf("do get kubeconfig: parse kubeconfig: %w", err)
	}

	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return err
	}

	// Load existing kubeconfig or start with an empty one.
	// Only ignore the error when the file does not yet exist.
	existing, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("do get kubeconfig: load existing kubeconfig: %w", err)
		}
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
		return fmt.Errorf("do get kubeconfig: create kube dir: %w", err)
	}

	if err := clientcmd.WriteToFile(*existing, kubeconfigPath); err != nil {
		return fmt.Errorf("do get kubeconfig: write kubeconfig: %w", err)
	}
	return nil
}

// CheckVersion is a no-op for NativeDORunner since no external binary is needed.
func (r *NativeDORunner) CheckVersion(ctx context.Context) error {
	return nil
}

// resolveClusterID finds a DOKS cluster ID by its name.
func resolveClusterID(ctx context.Context, client *godo.Client, name string) (string, error) {
	opt := &godo.ListOptions{PerPage: 200}

	for {
		clusters, resp, err := client.Kubernetes.List(ctx, opt)
		if err != nil {
			return "", fmt.Errorf("do resolve cluster: %w", err)
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
			return "", fmt.Errorf("do resolve cluster: parse pagination: %w", err)
		}
		opt.Page = page + 1
	}
	return "", fmt.Errorf("do resolve cluster: cluster %q not found", name)
}

// defaultKubeconfigPath returns the standard kubeconfig file path.
func defaultKubeconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("do get kubeconfig: determine home dir: %w", err)
	}
	return filepath.Join(home, ".kube", "config"), nil
}
