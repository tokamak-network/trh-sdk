package runner

import "context"

// HelmRunner defines the Helm operations used across TRH SDK.
// It replaces 21 helm subprocess calls.
//
// NativeHelmRunner uses the helm.sh/helm/v3/pkg/action library directly;
// ShellOutHelmRunner shells out to the helm binary as a fallback.
type HelmRunner interface {
	// Install installs a Helm chart (helm install).
	Install(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error

	// Upgrade performs helm upgrade --install for a release.
	Upgrade(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error

	// Uninstall removes a Helm release (helm uninstall).
	Uninstall(ctx context.Context, release, namespace string) error

	// List returns all release names in a namespace (helm list -q).
	List(ctx context.Context, namespace string) ([]string, error)

	// RepoAdd adds a Helm chart repository.
	RepoAdd(ctx context.Context, name, url string) error

	// RepoUpdate updates all Helm chart repositories.
	RepoUpdate(ctx context.Context) error

	// DependencyUpdate updates chart dependencies for the chart at chartPath.
	DependencyUpdate(ctx context.Context, chartPath string) error

	// Status returns the status of a release as a string.
	Status(ctx context.Context, release, namespace string) (string, error)

	// Search searches for charts matching a keyword in configured repositories.
	Search(ctx context.Context, keyword string) (string, error)

	// UpgradeWithFiles performs helm upgrade --install using a values file path
	// instead of an in-memory map. This matches the existing call sites that pass
	// --values <file>.
	UpgradeWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error
}
