package thanos

import (
	"context"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func TestDeployDRBInputHasExistingFields(t *testing.T) {
	input := types.DeployDRBInput{
		ExistingContractAddress: "0x4200000000000000000000000000000000000060",
		ExistingClusterName:     "my-l2-cluster",
	}
	if input.ExistingContractAddress != "0x4200000000000000000000000000000000000060" {
		t.Error("ExistingContractAddress field missing or not assignable")
	}
	if input.ExistingClusterName != "my-l2-cluster" {
		t.Error("ExistingClusterName field missing or not assignable")
	}
}

func TestDeployDRBInputEmptyFieldsKeepOriginalFlow(t *testing.T) {
	input := types.DeployDRBInput{}
	if input.ExistingContractAddress != "" {
		t.Error("ExistingContractAddress should default to empty string")
	}
	if input.ExistingClusterName != "" {
		t.Error("ExistingClusterName should default to empty string")
	}
}

func TestBuildContractsOutputFromExisting(t *testing.T) {
	out := buildContractsOutputFromExisting("0x4200000000000000000000000000000000000060", 17000)
	if out == nil {
		t.Fatal("buildContractsOutputFromExisting returned nil")
	}
	if out.ContractAddress != "0x4200000000000000000000000000000000000060" {
		t.Errorf("ContractAddress = %q, want predeploy address", out.ContractAddress)
	}
	if out.ChainID != 17000 {
		t.Errorf("ChainID = %d, want 17000", out.ChainID)
	}
}

func TestBuildInfraFromExisting(t *testing.T) {
	cfg := buildInfraFromExisting("my-l2-cluster", "vpc-0abc123", "ap-northeast-2")
	if cfg.ClusterName != "my-l2-cluster" {
		t.Errorf("ClusterName = %q, want %q", cfg.ClusterName, "my-l2-cluster")
	}
	if cfg.VpcID != "vpc-0abc123" {
		t.Errorf("VpcID = %q, want %q", cfg.VpcID, "vpc-0abc123")
	}
	if cfg.Namespace != "my-l2-cluster" {
		t.Errorf("Namespace = %q, want same as ClusterName", cfg.Namespace)
	}
	if cfg.Region != "ap-northeast-2" {
		t.Errorf("Region = %q, want %q", cfg.Region, "ap-northeast-2")
	}
}

func TestActivateRegularOperatorsInputValidation(t *testing.T) {
	stack := &ThanosStack{}

	// nil keys → error
	err := stack.ActivateRegularOperators(context.Background(), "http://rpc", "0x4200000000000000000000000000000000000060", nil)
	if err == nil {
		t.Error("expected error for nil keys")
	}

	// empty keys slice → no error (no-op)
	err = stack.ActivateRegularOperators(context.Background(), "http://rpc", "0x4200000000000000000000000000000000000060", []string{})
	if err != nil {
		t.Errorf("unexpected error for empty keys: %v", err)
	}
}
