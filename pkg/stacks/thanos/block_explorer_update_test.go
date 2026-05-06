package thanos

import (
	"context"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

// TestUpdateBlockExplorer_K8sNil verifies that UpdateBlockExplorer rejects
// requests when no Kubernetes config is present (i.e., chain not deployed yet
// or local-infra deployment).
func TestUpdateBlockExplorer_K8sNil(t *testing.T) {
	stack := &ThanosStack{
		deployConfig: &types.Config{},
		logger:       zap.NewNop().Sugar(),
	}
	_, err := stack.UpdateBlockExplorer(context.Background(), &InstallBlockExplorerInput{
		DatabaseUsername: "blockscout",
		DatabasePassword: "validpass123",
	})
	if err == nil {
		t.Fatal("expected error when K8s configuration is not set")
	}
	if !strings.Contains(err.Error(), "K8s") {
		t.Errorf("error message should mention K8s configuration: %v", err)
	}
}

// TestUpdateBlockExplorer_NilInput verifies that nil inputs are rejected.
func TestUpdateBlockExplorer_NilInput(t *testing.T) {
	stack := &ThanosStack{
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
		},
		logger: zap.NewNop().Sugar(),
	}
	_, err := stack.UpdateBlockExplorer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when inputs is nil")
	}
	if !strings.Contains(err.Error(), "inputs") {
		t.Errorf("error message should mention inputs: %v", err)
	}
}

// TestUpdateBlockExplorer_InvalidInput verifies that input validation runs
// (empty username/password should fail Validate()).
func TestUpdateBlockExplorer_InvalidInput(t *testing.T) {
	stack := &ThanosStack{
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
		},
		logger: zap.NewNop().Sugar(),
	}
	// Empty input fails Validate() (DatabaseUsername/Password are required).
	_, err := stack.UpdateBlockExplorer(context.Background(), &InstallBlockExplorerInput{})
	if err == nil {
		t.Fatal("expected validation error for empty input")
	}
}
