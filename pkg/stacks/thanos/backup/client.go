package backup

import "github.com/tokamak-network/trh-sdk/pkg/runner"

// BackupClient wraps optional Kubernetes runner for backup operations.
// When k8sRunner is nil all operations fall back to shelling out to kubectl.
type BackupClient struct {
	k8sRunner runner.K8sRunner // optional; nil → shellout
}

// NewBackupClient creates a BackupClient backed by the given K8sRunner.
// Pass nil to get the shellout-only path.
func NewBackupClient(k runner.K8sRunner) *BackupClient {
	return &BackupClient{k8sRunner: k}
}
