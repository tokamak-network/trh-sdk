package runner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	genericclioptions "k8s.io/cli-runtime/pkg/genericclioptions"
)

// NativeHelmRunner implements HelmRunner using the helm.sh/helm/v3 library.
// No helm binary is required.
type NativeHelmRunner struct {
	settings       *cli.EnvSettings
	kubeconfigPath string // explicit kubeconfig path; overrides default ~/.kube/config
	repoMu         sync.Mutex // guards concurrent RepoAdd calls (TOCTOU prevention)
}

// newNativeHelmRunner creates a NativeHelmRunner backed by the Helm SDK.
// kubeconfigPath overrides the default ~/.kube/config when non-empty.
func newNativeHelmRunner(kubeconfigPath string) (*NativeHelmRunner, error) {
	settings := cli.New()
	return &NativeHelmRunner{settings: settings, kubeconfigPath: kubeconfigPath}, nil
}

// actionConfig builds a per-namespace action.Configuration.
// When kubeconfigPath is set, kube.GetConfig is used directly so the path
// takes effect immediately (cli.EnvSettings.KubeConfig is only read at New()
// time and does not retroactively update the internal RESTClientGetter).
func (r *NativeHelmRunner) actionConfig(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	var getter genericclioptions.RESTClientGetter
	if r.kubeconfigPath != "" {
		getter = kube.GetConfig(r.kubeconfigPath, "", namespace)
	} else {
		getter = r.settings.RESTClientGetter()
	}
	if err := cfg.Init(getter, namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return nil, fmt.Errorf("helm action config: %w", err)
	}
	return cfg, nil
}

// loadChart locates and loads a chart from a local path or repository reference.
func (r *NativeHelmRunner) loadChart(chartRef string) (*chart.Chart, error) {
	// Try loading directly if it's a local path.
	absPath, err := filepath.Abs(chartRef)
	if err == nil {
		if _, statErr := os.Stat(absPath); statErr == nil {
			return loader.Load(absPath)
		}
	}
	// Fall back to LocateChart for repo-based references.
	pathOpts := action.ChartPathOptions{}
	cp, err := pathOpts.LocateChart(chartRef, r.settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart %s: %w", chartRef, err)
	}
	return loader.Load(cp)
}

// Install installs a Helm chart into the given namespace.
func (r *NativeHelmRunner) Install(ctx context.Context, release, chartRef, namespace string, vals map[string]interface{}) error {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return fmt.Errorf("helm install: %w", err)
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = release
	client.Namespace = namespace
	client.CreateNamespace = true

	chartObj, err := r.loadChart(chartRef)
	if err != nil {
		return fmt.Errorf("helm install: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, runErr := client.RunWithContext(ctx, chartObj, vals)
		errCh <- runErr
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("helm install %s: %w", release, ctx.Err())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("helm install %s: %w", release, err)
		}
		return nil
	}
}

// Upgrade performs helm upgrade --install for a release.
// In Helm SDK v3, Upgrade.Install is purely informational; the SDK does NOT
// automatically fall back to Install when the release does not exist.
// We must detect driver.ErrNoDeployedReleases and retry with action.Install.
func (r *NativeHelmRunner) Upgrade(ctx context.Context, release, chartRef, namespace string, vals map[string]interface{}) error {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return fmt.Errorf("helm upgrade: %w", err)
	}

	chartObj, err := r.loadChart(chartRef)
	if err != nil {
		return fmt.Errorf("helm upgrade: %w", err)
	}

	upClient := action.NewUpgrade(cfg)
	upClient.Namespace = namespace

	type upgradeResult struct{ err error }
	resCh := make(chan upgradeResult, 1)
	go func() {
		_, runErr := upClient.RunWithContext(ctx, release, chartObj, vals)
		resCh <- upgradeResult{runErr}
	}()

	var upgradeErr error
	select {
	case <-ctx.Done():
		return fmt.Errorf("helm upgrade %s: %w", release, ctx.Err())
	case res := <-resCh:
		upgradeErr = res.err
	}

	// Helm SDK does not implement --install fallback; detect missing release and install.
	if upgradeErr != nil && errors.Is(upgradeErr, driver.ErrNoDeployedReleases) {
		return r.Install(ctx, release, chartRef, namespace, vals)
	}
	if upgradeErr != nil {
		return fmt.Errorf("helm upgrade %s: %w", release, upgradeErr)
	}
	return nil
}

// UpgradeWithFiles performs helm upgrade --install using values file paths.
func (r *NativeHelmRunner) UpgradeWithFiles(ctx context.Context, release, chartRef, namespace string, valueFiles []string) error {
	valOpts := &values.Options{ValueFiles: valueFiles}
	providers := getter.All(r.settings)
	vals, err := valOpts.MergeValues(providers)
	if err != nil {
		return fmt.Errorf("helm upgrade-with-files: merge values: %w", err)
	}
	return r.Upgrade(ctx, release, chartRef, namespace, vals)
}

// Uninstall removes a Helm release from the given namespace.
func (r *NativeHelmRunner) Uninstall(ctx context.Context, release, namespace string) error {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return fmt.Errorf("helm uninstall: %w", err)
	}

	client := action.NewUninstall(cfg)

	errCh := make(chan error, 1)
	go func() {
		_, runErr := client.Run(release)
		errCh <- runErr
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("helm uninstall %s: %w", release, ctx.Err())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("helm uninstall %s: %w", release, err)
		}
		return nil
	}
}

// listResult carries the output of a helm list operation.
type listResult struct {
	names []string
	err   error
}

// List returns the names of all releases in a namespace.
// The Helm SDK's List.Run does not accept a context, so we wrap it in a goroutine
// and honour ctx cancellation while the SDK call runs independently.
func (r *NativeHelmRunner) List(ctx context.Context, namespace string) ([]string, error) {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("helm list: %w", err)
	}

	client := action.NewList(cfg)
	client.SetStateMask()

	resCh := make(chan listResult, 1)
	go func() {
		releases, runErr := client.Run()
		if runErr != nil {
			resCh <- listResult{err: fmt.Errorf("helm list namespace %s: %w", namespace, runErr)}
			return
		}
		names := make([]string, 0, len(releases))
		for _, rel := range releases {
			names = append(names, rel.Name)
		}
		resCh <- listResult{names: names}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("helm list %s: %w", namespace, ctx.Err())
	case res := <-resCh:
		return res.names, res.err
	}
}

// RepoAdd adds a Helm chart repository.
func (r *NativeHelmRunner) RepoAdd(ctx context.Context, name, url string) error {
	r.repoMu.Lock()
	defer r.repoMu.Unlock()

	repoFile := r.settings.RepositoryConfig

	repoFileObj, err := repo.LoadFile(repoFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("helm repo add %s: load repo file: %w", name, err)
		}
		repoFileObj = repo.NewFile()
	}

	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	chartRepo, err := repo.NewChartRepository(entry, getter.All(r.settings))
	if err != nil {
		return fmt.Errorf("helm repo add %s: %w", name, err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("helm repo add %s: download index: %w", name, err)
	}

	repoFileObj.Update(entry)

	if err := repoFileObj.WriteFile(repoFile, 0600); err != nil {
		return fmt.Errorf("helm repo add %s: write repo file: %w", name, err)
	}

	return nil
}

// RepoUpdate updates all configured Helm repositories.
func (r *NativeHelmRunner) RepoUpdate(ctx context.Context) error {
	repoFile := r.settings.RepositoryConfig
	repoFileObj, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("helm repo update: load repo file: %w", err)
	}

	var errs []string
	for _, entry := range repoFileObj.Repositories {
		chartRepo, repoErr := repo.NewChartRepository(entry, getter.All(r.settings))
		if repoErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name, repoErr))
			continue
		}
		if _, repoErr := chartRepo.DownloadIndexFile(); repoErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name, repoErr))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("helm repo update: %s", strings.Join(errs, "; "))
	}
	return nil
}

// DependencyUpdate updates chart dependencies for the chart at chartPath.
// Note: the Helm SDK's downloader.Manager.Update does not accept a context; ctx is not forwarded.
func (r *NativeHelmRunner) DependencyUpdate(ctx context.Context, chartPath string) error {
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		return fmt.Errorf("helm dependency update: resolve path: %w", err)
	}

	man := &downloader.Manager{
		Out:              os.Stdout,
		ChartPath:        absPath,
		Getters:          getter.All(r.settings),
		RepositoryConfig: r.settings.RepositoryConfig,
		RepositoryCache:  r.settings.RepositoryCache,
	}

	if err := man.Update(); err != nil {
		return fmt.Errorf("helm dependency update %s: %w", chartPath, err)
	}
	return nil
}

// Status returns the status string for a release.
// Note: the Helm SDK's Status.Run does not accept a context; ctx is not forwarded.
func (r *NativeHelmRunner) Status(ctx context.Context, release, namespace string) (string, error) {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return "", fmt.Errorf("helm status: %w", err)
	}

	client := action.NewStatus(cfg)
	rel, err := client.Run(release)
	if err != nil {
		return "", fmt.Errorf("helm status %s: %w", release, err)
	}

	return rel.Info.Status.String(), nil
}

// Search searches configured repositories for charts matching the keyword.
// Note: ctx is accepted for interface compatibility but is not used in this implementation.
func (r *NativeHelmRunner) Search(ctx context.Context, keyword string) (string, error) {
	repoFile := r.settings.RepositoryConfig
	repoFileObj, err := repo.LoadFile(repoFile)
	if err != nil {
		return "", fmt.Errorf("helm search: load repo file: %w", err)
	}

	var results []string
	for _, entry := range repoFileObj.Repositories {
		idxPath := filepath.Join(r.settings.RepositoryCache, fmt.Sprintf("%s-index.yaml", entry.Name))
		idx, idxErr := repo.LoadIndexFile(idxPath)
		if idxErr != nil {
			continue
		}
		for chartName, versions := range idx.Entries {
			if strings.Contains(chartName, keyword) && len(versions) > 0 {
				latest := versions[0]
				results = append(results, fmt.Sprintf("%s/%s\t%s\t%s",
					entry.Name, chartName, latest.Version, latest.Description))
			}
		}
	}
	return strings.Join(results, "\n"), nil
}
