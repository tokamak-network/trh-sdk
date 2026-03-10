package mock

import (
	"context"
	"sync"
)

// HelmRunner is a thread-safe, hand-written mock for runner.HelmRunner.
// Configure expected results via the On* fields before calling methods.
// Recorded calls are available via Calls / CallCount / GetCalls after the test.
//
// Example:
//
//	m := &mock.HelmRunner{}
//	m.OnInstall = func(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error { return nil }
//	err := m.Install(ctx, "my-release", "my-chart", "default", nil)
type HelmRunner struct {
	mu    sync.Mutex
	Calls []Call

	OnInstall          func(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error
	OnUpgrade          func(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error
	OnUninstall        func(ctx context.Context, release, namespace string) error
	OnList             func(ctx context.Context, namespace string) ([]string, error)
	OnRepoAdd          func(ctx context.Context, name, url string) error
	OnRepoUpdate       func(ctx context.Context) error
	OnDependencyUpdate func(ctx context.Context, chartPath string) error
	OnStatus           func(ctx context.Context, release, namespace string) (string, error)
	OnSearch           func(ctx context.Context, keyword string) (string, error)
	OnUpgradeWithFiles func(ctx context.Context, release, chart, namespace string, valueFiles []string) error
}

func (m *HelmRunner) record(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

// CallCount returns how many times method was called.
func (m *HelmRunner) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, c := range m.Calls {
		if c.Method == method {
			count++
		}
	}
	return count
}

// GetCalls returns a snapshot of all recorded calls, safe for concurrent reads.
func (m *HelmRunner) GetCalls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make([]Call, len(m.Calls))
	copy(snapshot, m.Calls)
	return snapshot
}

func (m *HelmRunner) Install(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error {
	m.record("Install", release, chart, namespace, values)
	if m.OnInstall != nil {
		return m.OnInstall(ctx, release, chart, namespace, values)
	}
	return nil
}

func (m *HelmRunner) Upgrade(ctx context.Context, release, chart, namespace string, values map[string]interface{}) error {
	m.record("Upgrade", release, chart, namespace, values)
	if m.OnUpgrade != nil {
		return m.OnUpgrade(ctx, release, chart, namespace, values)
	}
	return nil
}

func (m *HelmRunner) Uninstall(ctx context.Context, release, namespace string) error {
	m.record("Uninstall", release, namespace)
	if m.OnUninstall != nil {
		return m.OnUninstall(ctx, release, namespace)
	}
	return nil
}

func (m *HelmRunner) List(ctx context.Context, namespace string) ([]string, error) {
	m.record("List", namespace)
	if m.OnList != nil {
		return m.OnList(ctx, namespace)
	}
	return nil, nil
}

func (m *HelmRunner) RepoAdd(ctx context.Context, name, url string) error {
	m.record("RepoAdd", name, url)
	if m.OnRepoAdd != nil {
		return m.OnRepoAdd(ctx, name, url)
	}
	return nil
}

func (m *HelmRunner) RepoUpdate(ctx context.Context) error {
	m.record("RepoUpdate")
	if m.OnRepoUpdate != nil {
		return m.OnRepoUpdate(ctx)
	}
	return nil
}

func (m *HelmRunner) DependencyUpdate(ctx context.Context, chartPath string) error {
	m.record("DependencyUpdate", chartPath)
	if m.OnDependencyUpdate != nil {
		return m.OnDependencyUpdate(ctx, chartPath)
	}
	return nil
}

func (m *HelmRunner) Status(ctx context.Context, release, namespace string) (string, error) {
	m.record("Status", release, namespace)
	if m.OnStatus != nil {
		return m.OnStatus(ctx, release, namespace)
	}
	return "", nil
}

func (m *HelmRunner) Search(ctx context.Context, keyword string) (string, error) {
	m.record("Search", keyword)
	if m.OnSearch != nil {
		return m.OnSearch(ctx, keyword)
	}
	return "", nil
}

func (m *HelmRunner) UpgradeWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error {
	m.record("UpgradeWithFiles", release, chart, namespace, valueFiles)
	if m.OnUpgradeWithFiles != nil {
		return m.OnUpgradeWithFiles(ctx, release, chart, namespace, valueFiles)
	}
	return nil
}
