package thanos

import (
	"math/big"
	"strings"
	"testing"
)

// TestBuildDeployContractsArgs_FaultProofOn verifies that EnableFaultProof=true
// causes --fault-proof to be appended to the deploy-contracts argv.
// Regression test for Bug #8 (fault-proof contracts silently skipped because
// trh-sdk never passed the CLI flag).
func TestBuildDeployContractsArgs_FaultProofOn(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:         "https://example",
		PrivateKey:       "0xabc",
		L2ChainID:        42,
		OutPath:          "/tmp/out.json",
		EnableFaultProof: true,
	})

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--fault-proof") {
		t.Errorf("expected --fault-proof in args, got: %v", args)
	}
}

// TestBuildDeployContractsArgs_FaultProofOff verifies that EnableFaultProof=false
// omits the flag (no-op default on the CLI side). We pass it by presence/absence
// rather than --fault-proof=false so older tokamak-deployer binaries that don't
// recognize the flag only fail when the feature is actually requested.
func TestBuildDeployContractsArgs_FaultProofOff(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:         "https://example",
		PrivateKey:       "0xabc",
		L2ChainID:        42,
		OutPath:          "/tmp/out.json",
		EnableFaultProof: false,
	})

	for _, a := range args {
		if a == "--fault-proof" {
			t.Errorf("expected no --fault-proof when disabled, got: %v", args)
		}
	}
}

// TestBuildDeployContractsArgs_GasPricePreserved verifies the existing gas-price
// wiring is untouched by the new flag.
func TestBuildDeployContractsArgs_GasPricePreserved(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:    "https://example",
		PrivateKey:  "0xabc",
		L2ChainID:   42,
		OutPath:     "/tmp/out.json",
		GasPriceWei: big.NewInt(3_000_000_000),
	})

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--gas-price 3000000000") {
		t.Errorf("expected --gas-price 3000000000 in args, got: %v", args)
	}
}
