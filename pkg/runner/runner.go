// Package runner provides a ToolRunner abstraction that replaces external binary
// subprocess calls (kubectl, helm, aws, doctl, terraform) with native Go library
// implementations. The ShellOutRunner preserves existing behaviour; NativeRunner
// calls Go libraries directly and requires no external binaries.
package runner

import (
	"os"
)

// ToolRunner is the top-level interface that aggregates all sub-runners.
// Use RunnerFactory to obtain an implementation.
type ToolRunner interface {
	K8s() K8sRunner
}

// RunnerConfig controls which implementation is selected.
type RunnerConfig struct {
	// UseNative selects NativeRunner when true. When false (default) or when the
	// TRHS_LEGACY environment variable is set, ShellOutRunner is used.
	UseNative      bool
	KubeconfigPath string // path to kubeconfig; empty = in-cluster or default ~/.kube/config
}

// New returns the appropriate ToolRunner based on cfg.
// Pass cfg.UseNative = true to get NativeRunner (Phase 1: K8s only).
// Set TRHS_LEGACY=1 to force ShellOutRunner regardless of cfg.
func New(cfg RunnerConfig) (ToolRunner, error) {
	if os.Getenv("TRHS_LEGACY") == "1" || !cfg.UseNative {
		return &ShellOutRunner{}, nil
	}
	return newNativeRunner(cfg)
}

// newNativeRunner builds a NativeRunner wired to the specified kubeconfig.
func newNativeRunner(cfg RunnerConfig) (*NativeRunner, error) {
	k8s, err := newNativeK8sRunner(cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	return &NativeRunner{k8s: k8s}, nil
}

// NativeRunner is the ToolRunner implementation that calls Go libraries directly.
type NativeRunner struct {
	k8s K8sRunner
}

func (r *NativeRunner) K8s() K8sRunner { return r.k8s }

// ShellOutRunner is the legacy ToolRunner that delegates to external binaries
// via ExecuteCommand. It is always available as a fallback.
type ShellOutRunner struct{}

func (r *ShellOutRunner) K8s() K8sRunner { return &ShellOutK8sRunner{} }

