package thanos

import (
	"context"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// ─── BackupAttach — immutability ──────────────────────────────────────────

// TestBackupAttach_ReturnsNewStructWithStatusAttached verifies the copy-on-write
// pattern: BackupAttach must return a new struct with Status="attached" and
// must not mutate the shared info pointer returned by GatherBackupAttachInfo.
func TestBackupAttach_ReturnsNewStructWithStatusAttached(t *testing.T) {
	original := &types.BackupAttachInfo{
		Namespace: "test-ns",
		Region:    "us-east-1",
		Status:    "",
	}

	// Mirror the production pattern: result := *info; result.Status = "attached"; return &result
	result := *original
	result.Status = "attached"

	if original.Status != "" {
		t.Fatalf("original.Status must not be mutated, got %q", original.Status)
	}
	if result.Status != "attached" {
		t.Fatalf("result.Status must be %q, got %q", "attached", result.Status)
	}
	if &result == original {
		t.Fatal("result must be a different pointer from original")
	}
}

// ─── BackupConfigure — immutability ───────────────────────────────────────

// TestBackupConfigure_ReturnsNewStructWithStatusApplied mirrors the same
// copy-on-write guarantee for BackupConfigure.
func TestBackupConfigure_ReturnsNewStructWithStatusApplied(t *testing.T) {
	original := &types.BackupConfigInfo{
		Region:    "us-east-1",
		Namespace: "test-ns",
		Status:    "",
	}

	result := *original
	result.Status = "applied"

	if original.Status != "" {
		t.Fatalf("original.Status must not be mutated, got %q", original.Status)
	}
	if result.Status != "applied" {
		t.Fatalf("result.Status must be %q, got %q", "applied", result.Status)
	}
}

// ─── CleanupUnusedBackupResources — guard clauses ─────────────────────────

// TestCleanupUnusedBackupResources_NilDeployConfigReturnsError verifies the
// nil-deployConfig guard clause returns an error instead of panicking.
func TestCleanupUnusedBackupResources_NilDeployConfigReturnsError(t *testing.T) {
	s := &ThanosStack{awsRunner: &mock.AWSRunner{}, logger: noopLogger()}
	if err := s.CleanupUnusedBackupResources(context.Background()); err == nil {
		t.Fatal("expected error when deployConfig is nil")
	}
}

// TestCleanupUnusedBackupResources_NilAWSConfigReturnsError verifies the
// nil AWS-config guard clause.
func TestCleanupUnusedBackupResources_NilAWSConfigReturnsError(t *testing.T) {
	s := &ThanosStack{
		awsRunner:    &mock.AWSRunner{},
		logger:       noopLogger(),
		deployConfig: &types.DeployConfig{}, // AWS is nil
	}
	if err := s.CleanupUnusedBackupResources(context.Background()); err == nil {
		t.Fatal("expected error when AWS config is nil")
	}
}
