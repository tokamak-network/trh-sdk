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

// TestBuildDeployContractsArgs_DelayedWETHDelay verifies that a non-zero
// DelayedWETHDelay emits --delayed-weth-delay only when fault-proof is enabled,
// and that zero (default) omits the flag.
func TestBuildDeployContractsArgs_DelayedWETHDelay(t *testing.T) {
	// Non-zero delay with fault-proof on → flag must appear
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:         "https://example",
		PrivateKey:       "0xabc",
		L2ChainID:        42,
		OutPath:          "/tmp/out.json",
		EnableFaultProof: true,
		DelayedWETHDelay: 604800,
	})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--delayed-weth-delay 604800") {
		t.Errorf("expected --delayed-weth-delay 604800 in args, got: %v", args)
	}

	// Zero delay (default) → flag must be absent
	argsZero := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:         "https://example",
		PrivateKey:       "0xabc",
		L2ChainID:        42,
		OutPath:          "/tmp/out.json",
		EnableFaultProof: true,
		DelayedWETHDelay: 0,
	})
	for _, a := range argsZero {
		if a == "--delayed-weth-delay" {
			t.Errorf("expected no --delayed-weth-delay when delay is 0, got: %v", argsZero)
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

// TestBuildDeployContractsArgs_ReuseDeploymentOn verifies that ReuseDeployment=true
// causes --reuse-deployment to be appended to the deploy-contracts argv.
// Available since tokamak-deployer v0.0.9.
func TestBuildDeployContractsArgs_ReuseDeploymentOn(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:        "https://example",
		PrivateKey:      "0xabc",
		L2ChainID:       42,
		OutPath:         "/tmp/out.json",
		ReuseDeployment: true,
	})

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--reuse-deployment") {
		t.Errorf("expected --reuse-deployment in args, got: %v", args)
	}
}

// TestBuildDeployContractsArgs_ReuseDeploymentOff verifies that the default
// (ReuseDeployment=false) omits --reuse-deployment so older deployers ignore the
// feature entirely.
func TestBuildDeployContractsArgs_ReuseDeploymentOff(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:        "https://example",
		PrivateKey:      "0xabc",
		L2ChainID:       42,
		OutPath:         "/tmp/out.json",
		ReuseDeployment: false,
	})

	for _, a := range args {
		if strings.HasPrefix(a, "--reuse") {
			t.Errorf("expected no --reuse-* flag when disabled, got: %v", args)
		}
	}
}

// TestBuildDeployContractsArgs_RegistryPathRequiresReuse verifies that
// RegistryPath alone (without ReuseDeployment=true) does not pass --reuse-impls.
// Mirrors the deployer-side warn-and-ignore behavior so trh-sdk does not
// generate an inconsistent argv.
func TestBuildDeployContractsArgs_RegistryPathRequiresReuse(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:     "https://example",
		PrivateKey:   "0xabc",
		L2ChainID:    42,
		OutPath:      "/tmp/out.json",
		RegistryPath: "/tmp/registry.json",
		// ReuseDeployment intentionally false.
	})

	for _, a := range args {
		if strings.HasPrefix(a, "--reuse") {
			t.Errorf("RegistryPath without ReuseDeployment must not emit --reuse-* flags, got: %v", args)
		}
	}
}

// TestBuildDeployContractsArgs_ReuseWithRegistryPath verifies that both flags
// are passed together when ReuseDeployment=true and RegistryPath is set.
func TestBuildDeployContractsArgs_ReuseWithRegistryPath(t *testing.T) {
	args := buildDeployContractsArgs(deployContractsOpts{
		L1RPCURL:        "https://example",
		PrivateKey:      "0xabc",
		L2ChainID:       42,
		OutPath:         "/tmp/out.json",
		ReuseDeployment: true,
		RegistryPath:    "/tmp/registry.json",
	})

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--reuse-deployment") {
		t.Errorf("expected --reuse-deployment in args, got: %v", args)
	}
	if !strings.Contains(joined, "--reuse-impls /tmp/registry.json") {
		t.Errorf("expected --reuse-impls /tmp/registry.json in args, got: %v", args)
	}
}
