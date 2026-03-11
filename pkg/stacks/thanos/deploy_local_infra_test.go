package thanos

import (
	"context"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// ─── guard-condition unit tests ──────────────────────────────────────────────
//
// These tests exercise the input-validation guards at the top of
// DeployLocalInfrastructure and do not require a real cluster or network access.

// TestDeployLocalInfrastructure_NilInputs verifies that a nil inputs pointer
// is rejected immediately with a descriptive error before any I/O is performed.
func TestDeployLocalInfrastructure_NilInputs(t *testing.T) {
	s := &ThanosStack{
		logger: noopLogger(),
		deployConfig: &types.Config{
			DeployContractState: &types.DeployContractState{
				Status: types.DeployContractStatusCompleted,
			},
		},
	}

	err := s.DeployLocalInfrastructure(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil inputs, got nil")
	}
}

// TestDeployLocalInfrastructure_NilDeployConfig verifies that the function
// returns an error when settings.json has not been loaded (deployConfig == nil).
func TestDeployLocalInfrastructure_NilDeployConfig(t *testing.T) {
	s := &ThanosStack{
		logger:       noopLogger(),
		deployConfig: nil,
	}

	err := s.DeployLocalInfrastructure(context.Background(), &DeployLocalInfraInput{
		ChainName:   "test-chain",
		L1BeaconURL: "http://localhost:5052",
	})
	if err == nil {
		t.Fatal("expected error when deployConfig is nil, got nil")
	}
}

// TestDeployLocalInfrastructure_ContractStateNil verifies that the function
// returns an error when DeployContractState is nil (contracts were never deployed).
func TestDeployLocalInfrastructure_ContractStateNil(t *testing.T) {
	s := &ThanosStack{
		logger: noopLogger(),
		deployConfig: &types.Config{
			DeployContractState: nil,
		},
	}

	err := s.DeployLocalInfrastructure(context.Background(), &DeployLocalInfraInput{
		ChainName:   "test-chain",
		L1BeaconURL: "http://localhost:5052",
	})
	if err == nil {
		t.Fatal("expected error when DeployContractState is nil, got nil")
	}
}

// TestDeployLocalInfrastructure_ContractNotCompleted verifies that the function
// returns an error when the contract deployment is still InProgress.
func TestDeployLocalInfrastructure_ContractNotCompleted(t *testing.T) {
	s := &ThanosStack{
		logger: noopLogger(),
		deployConfig: &types.Config{
			DeployContractState: &types.DeployContractState{
				Status: types.DeployContractStatusInProgress,
			},
		},
	}

	err := s.DeployLocalInfrastructure(context.Background(), &DeployLocalInfraInput{
		ChainName:   "test-chain",
		L1BeaconURL: "http://localhost:5052",
	})
	if err == nil {
		t.Fatal("expected error when contract status is InProgress, got nil")
	}
}
