package thanos

import (
	"context"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

func TestDeployLocalInfrastructure_NilInputs(t *testing.T) {
	s := &ThanosStack{
		logger: zap.NewNop().Sugar(),
		deployConfig: &types.Config{
			DeployContractState: &types.DeployContractState{
				Status: types.DeployContractStatusCompleted,
			},
		},
	}
	err := s.DeployLocalInfrastructure(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil inputs")
	}
}

func TestDeployLocalInfrastructure_NilDeployConfig(t *testing.T) {
	s := &ThanosStack{
		logger: zap.NewNop().Sugar(),
	}
	err := s.DeployLocalInfrastructure(context.Background(), &DeployLocalInfraInput{
		ChainName: "test",
	})
	if err == nil {
		t.Fatal("expected error for nil deployConfig")
	}
}

func TestDeployLocalInfrastructure_ContractNotCompleted(t *testing.T) {
	s := &ThanosStack{
		logger: zap.NewNop().Sugar(),
		deployConfig: &types.Config{
			DeployContractState: &types.DeployContractState{
				Status: types.DeployContractStatusInProgress,
			},
		},
	}
	err := s.DeployLocalInfrastructure(context.Background(), &DeployLocalInfraInput{
		ChainName: "test",
	})
	if err == nil {
		t.Fatal("expected error for incomplete contracts")
	}
}

func TestAllComponentsRunning(t *testing.T) {
	tests := []struct {
		name     string
		pods     []string
		expected bool
	}{
		{"all running", []string{"rel-thanos-stack-op-geth-0", "rel-thanos-stack-op-node-0", "rel-op-batcher-abc", "rel-op-proposer-xyz"}, true},
		{"missing op-node", []string{"rel-op-geth-0", "rel-op-batcher-abc", "rel-op-proposer-xyz"}, false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allComponentsRunning(tt.pods); got != tt.expected {
				t.Errorf("allComponentsRunning(%v) = %v, want %v", tt.pods, got, tt.expected)
			}
		})
	}
}
