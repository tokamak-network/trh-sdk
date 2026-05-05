package thanos

import (
	"context"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

// TestUninstallFeatures_CollectsAllErrors verifies that uninstallFeatures continues
// after individual failures and collects all errors rather than returning on the first.
// With K8s=nil, Bridge, BlockExplorer, and UptimeService return immediate errors
// (no external calls), guaranteeing at least 3 errors regardless of environment.
func TestUninstallFeatures_CollectsAllErrors(t *testing.T) {
	stack := &ThanosStack{
		deployConfig:   &types.Config{L2ChainID: 12345},
		logger:         zap.NewNop().Sugar(),
		deploymentPath: t.TempDir(),
	}

	err := stack.uninstallFeatures(context.Background())
	if err == nil {
		t.Fatal("expected error from failed uninstallers, got nil")
	}

	errStr := err.Error()
	// Bridge and UptimeService return K8s-nil errors without external calls.
	// Both must appear — confirming the pipeline did NOT stop at the first error.
	if !strings.Contains(errStr, "Bridge") {
		t.Errorf("expected Bridge error in output, got: %s", errStr)
	}
	if !strings.Contains(errStr, "UptimeService") {
		t.Errorf("expected UptimeService error in output, got: %s", errStr)
	}
}
