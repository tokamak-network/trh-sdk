package thanos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const (
	l1BlockProxyAddress = "0x4200000000000000000000000000000000000015"
)

// injectL1BlockBytecode patches the L1Block implementation entry (code namespace address)
// in genesis.json with the Isthmus-capable bytecode from the forge artifact.
//
// Root cause this fixes:
//
//	When isthmus_time == genesis block timestamp, IsIsthmusActivationBlock in op-node is
//	never true for any block it builds (activation would be block 0, which op-node skips).
//	As a result, the Isthmus upgrade transactions that would normally update the L1Block
//	proxy implementation are never injected. The genesis therefore ends up with the
//	old Ecotone-only L1Block implementation (no setL1BlockValuesIsthmus selector).
//	op-node always sends Isthmus-format calldata → unknown selector → every L1 attributes
//	deposit transaction reverts (status=0x0).
//
// This function overwrites the `code` field at the L1Block code namespace address with
// the bytecode freshly compiled from tokamak-thanos (which includes setL1BlockValuesIsthmus).
// All other fields (balance, storage, nonce) in the existing alloc entry are preserved.
//
// Prerequisite: forge build must have been run so that forge-artifacts/L1Block.sol/ exists.
func injectL1BlockBytecode(genesisPath, deploymentPath string) error {
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(data, &genesis); err != nil {
		return fmt.Errorf("failed to parse genesis JSON: %w", err)
	}

	var alloc map[string]json.RawMessage
	if err := json.Unmarshal(genesis["alloc"], &alloc); err != nil {
		return fmt.Errorf("failed to parse alloc section: %w", err)
	}

	// Detect alloc key format (with or without 0x prefix) from existing entries
	has0xPrefix := false
	for key := range alloc {
		if strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X") {
			has0xPrefix = true
		}
		break
	}

	formatAddr := func(addr string) string {
		lower := strings.ToLower(addr)
		if !has0xPrefix {
			return strings.TrimPrefix(lower, "0x")
		}
		return lower
	}

	// Load deployedBytecode from forge-artifacts (reuses loadForgeArtifactBytecode from aa_paymaster_genesis.go)
	artifactPath := filepath.Join(
		deploymentPath,
		"tokamak-thanos", "packages", "tokamak", "contracts-bedrock",
		"forge-artifacts", "L1Block.sol", "L1Block.json",
	)
	implBytecode, err := loadForgeArtifactBytecode(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to load L1Block bytecode: %w", err)
	}

	// Compute code namespace address for the L1Block implementation.
	// See: Predeploys.sol predeployToCodeNamespace()
	proxyAddr := common.HexToAddress(l1BlockProxyAddress)
	codeAddr := predeployToCodeNamespace(proxyAddr)
	codeAddrKey := formatAddr(codeAddr.Hex())

	// Preserve existing alloc entry fields (balance, storage, nonce); only replace code.
	// L1Block is a core predeploy — its storage slots hold initialised state in genesis.
	var entry map[string]json.RawMessage
	if existing, ok := alloc[codeAddrKey]; ok {
		if err := json.Unmarshal(existing, &entry); err != nil {
			entry = make(map[string]json.RawMessage)
		}
	} else {
		entry = make(map[string]json.RawMessage)
	}

	codeJSON, err := json.Marshal(implBytecode)
	if err != nil {
		return fmt.Errorf("failed to marshal L1Block bytecode: %w", err)
	}
	entry["code"] = codeJSON

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal L1Block alloc entry: %w", err)
	}
	alloc[codeAddrKey] = entryJSON

	// Write back genesis.json
	allocJSON, err := json.Marshal(alloc)
	if err != nil {
		return fmt.Errorf("failed to marshal alloc: %w", err)
	}
	genesis["alloc"] = allocJSON

	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}

	fmt.Println("Injected L1Block (Isthmus-capable) implementation into genesis code namespace at", codeAddr.Hex())
	return os.WriteFile(genesisPath, output, 0644)
}
