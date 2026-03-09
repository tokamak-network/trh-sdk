package backup

import "github.com/tokamak-network/trh-sdk/pkg/runner"

// BackupClient wraps optional Kubernetes runner for backup operations.
// When k8sRunner is nil all operations fall back to shelling out to kubectl.
type BackupClient struct {
	k8sRunner runner.K8sRunner // optional; nil → shellout
}

// defaultClient is the package-level BackupClient used by public API functions in attach.go.
// It starts as shellout-only; call SetDefaultK8sRunner to enable the native library path.
var defaultClient = &BackupClient{}

// SetDefaultK8sRunner replaces the package-level BackupClient with one backed by kr.
// Call this once at startup (e.g., from ThanosStack.SetK8sRunner) to enable the
// native k8s.io/client-go path for all backup operations.
func SetDefaultK8sRunner(kr runner.K8sRunner) {
	defaultClient = NewBackupClient(kr)
}

// NewBackupClient creates a BackupClient backed by the given K8sRunner.
// Pass nil to get the shellout-only path.
func NewBackupClient(k runner.K8sRunner) *BackupClient {
	return &BackupClient{k8sRunner: k}
}
