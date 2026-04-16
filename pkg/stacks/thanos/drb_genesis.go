package thanos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const (
	//nolint:unused // used by downloadDRBArtifact
	drbNpmPackageName = "@tokamak-network/commit-reveal2-contracts"
	//nolint:unused // used by downloadDRBArtifact
	drbNpmTag         = "1.0.0"
	//nolint:unused // used by downloadDRBArtifact
	drbArtifactFile   = "CommitReveal2L2.json"

	// Predeploy address for DRB (Gaming/Full preset)
	drbPredeployAddress = "0x4200000000000000000000000000000000000060"

	// ERC1967 implementation slot: bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1)
	erc1967ImplementationSlot = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"

	// Gas limit for simulated EVM deployment
	drbDeployGasLimit = 30_000_000
)

// drbArtifact represents the relevant fields from the CommitReveal2L2.json forge artifact.
//nolint:unused // used by buildDRBCreationCode
type drbArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

// DRBGenesisConfig holds the constructor parameters for the CommitReveal2L2 predeploy.
type DRBGenesisConfig struct {
	ActivationThreshold                *big.Int
	FlatFee                            *big.Int
	Name                               string
	Version                            string
	OffChainSubmissionPeriod            *big.Int
	RequestOrSubmitOrFailDecisionPeriod *big.Int
	OnChainSubmissionPeriod             *big.Int
	OffChainSubmissionPeriodPerOperator *big.Int
	OnChainSubmissionPeriodPerOperator  *big.Int
}

// DefaultDRBGenesisConfig returns sensible defaults for a local/testnet DRB deployment.
func DefaultDRBGenesisConfig() *DRBGenesisConfig {
	return &DRBGenesisConfig{
		ActivationThreshold:                big.NewInt(3),
		FlatFee:                            big.NewInt(0),
		Name:                               "Commit-Reveal2",
		Version:                            "1",
		OffChainSubmissionPeriod:            big.NewInt(120),
		RequestOrSubmitOrFailDecisionPeriod: big.NewInt(120),
		OnChainSubmissionPeriod:             big.NewInt(120),
		OffChainSubmissionPeriodPerOperator: big.NewInt(30),
		OnChainSubmissionPeriodPerOperator:  big.NewInt(30),
	}
}

// injectDRBIntoGenesis downloads the CommitReveal2L2 artifact from npm, deploys it
// in a simulated EVM to resolve immutables, and patches the genesis.json alloc section
// with the DRB contract at its predeploy address.
//nolint:unused // called during L2 deployment orchestration
func injectDRBIntoGenesis(ctx context.Context, logger interface{ Info(args ...interface{}) }, genesisPath string, config *DRBGenesisConfig) error {
	// Step 1: Download artifact from npm
	artifactData, err := downloadDRBArtifact(ctx, logger)
	if err != nil {
		return fmt.Errorf("failed to download DRB artifact: %w", err)
	}

	// Step 2: Parse artifact
	var artifact drbArtifact
	if err := json.Unmarshal(artifactData, &artifact); err != nil {
		return fmt.Errorf("failed to parse DRB artifact: %w", err)
	}

	// Step 3: Build constructor input (bytecode + ABI-encoded args)
	creationCode, err := buildDRBCreationCode(&artifact, config)
	if err != nil {
		return fmt.Errorf("failed to build DRB creation code: %w", err)
	}

	// Step 4: Execute constructor in simulated EVM to get runtime bytecode with immutables
	// CommitReveal2 constructor requires msg.value >= activationThreshold
	runtimeBytecode, err := deployDRBSimulated(creationCode, config.ActivationThreshold)
	if err != nil {
		return fmt.Errorf("failed to deploy DRB in simulated EVM: %w", err)
	}

	// Step 5: Patch genesis.json
	if err := patchGenesisWithDRB(genesisPath, runtimeBytecode); err != nil {
		return fmt.Errorf("failed to patch genesis with DRB: %w", err)
	}

	logger.Info("✅ DRB contract (CommitReveal2L2) injected into genesis at " + drbPredeployAddress)
	return nil
}

// downloadDRBArtifact fetches the CommitReveal2L2.json artifact from the npm package.
//nolint:unused // called by injectDRBIntoGenesis
func downloadDRBArtifact(ctx context.Context, logger interface{ Info(args ...interface{}) }) ([]byte, error) {
	logger.Info("Downloading DRB contract artifact from npm...")

	tarballURL, err := resolveNpmTarballURL(ctx, drbNpmPackageName, drbNpmTag)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DRB npm tarball URL: %w", err)
	}

	return downloadAndExtractSingleFile(ctx, tarballURL, "artifacts/"+drbArtifactFile)
}

// buildDRBCreationCode constructs the full creation bytecode: constructor code + ABI-encoded args.
//nolint:unused // called by injectDRBIntoGenesis
func buildDRBCreationCode(artifact *drbArtifact, config *DRBGenesisConfig) ([]byte, error) {
	parsedABI, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	encodedArgs, err := parsedABI.Constructor.Inputs.Pack(
		config.ActivationThreshold,
		config.FlatFee,
		config.Name,
		config.Version,
		config.OffChainSubmissionPeriod,
		config.RequestOrSubmitOrFailDecisionPeriod,
		config.OnChainSubmissionPeriod,
		config.OffChainSubmissionPeriodPerOperator,
		config.OnChainSubmissionPeriodPerOperator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode constructor args: %w", err)
	}

	bytecodeHex := strings.TrimPrefix(artifact.Bytecode.Object, "0x")
	bytecode, err := hex.DecodeString(bytecodeHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode hex: %w", err)
	}

	return append(bytecode, encodedArgs...), nil
}

// deployDRBSimulated executes the creation code in go-ethereum's simulated EVM
// and returns the deployed runtime bytecode (with immutable values filled in).
// Uses Cancun-enabled EVM to support PUSH0 (Solidity ≥ 0.8.20).
// The value parameter is sent as msg.value to satisfy the constructor's
// require(msg.value >= activationThreshold) check.
func deployDRBSimulated(creationCode []byte, value *big.Int) ([]byte, error) {
	cancunTime := uint64(0)
	shanghaiTime := uint64(0)

	// Create in-memory state DB with sender balance to cover msg.value transfer
	sender := common.HexToAddress("0x1000000000000000000000000000000000000000")
	stateDB, err := state.New(common.Hash{}, state.NewDatabaseForTesting())
	if err != nil {
		return nil, fmt.Errorf("failed to create state DB: %w", err)
	}
	// Give sender enough balance for the value transfer
	senderBalance := new(big.Int).Mul(value, big.NewInt(10))
	if senderBalance.Sign() == 0 {
		senderBalance = big.NewInt(1_000_000_000_000_000_000) // 1 ETH default
	}
	senderBalanceU256, _ := uint256.FromBig(senderBalance)
	stateDB.AddBalance(sender, senderBalanceU256, tracing.BalanceChangeUnspecified)

	cfg := &runtime.Config{
		ChainConfig: &params.ChainConfig{
			ChainID:                 big.NewInt(1),
			HomesteadBlock:          new(big.Int),
			DAOForkBlock:            new(big.Int),
			EIP150Block:             new(big.Int),
			EIP155Block:             new(big.Int),
			EIP158Block:             new(big.Int),
			ByzantiumBlock:          new(big.Int),
			ConstantinopleBlock:     new(big.Int),
			PetersburgBlock:         new(big.Int),
			IstanbulBlock:           new(big.Int),
			MuirGlacierBlock:        new(big.Int),
			BerlinBlock:             new(big.Int),
			LondonBlock:             new(big.Int),
			TerminalTotalDifficulty: big.NewInt(0),
			ShanghaiTime:            &shanghaiTime,
			CancunTime:              &cancunTime,
		},
		Origin:   sender,
		GasLimit: drbDeployGasLimit,
		Value:    value,
		State:    stateDB,
	}

	runtimeBytecode, _, _, err := runtime.Create(creationCode, cfg)
	if err != nil {
		return nil, fmt.Errorf("EVM create failed: %w", err)
	}
	if len(runtimeBytecode) == 0 {
		return nil, fmt.Errorf("EVM create returned empty bytecode")
	}
	return runtimeBytecode, nil
}

// patchGenesisWithDRB reads genesis.json, adds the DRB implementation at the
// code-namespace address, sets the proxy's implementation slot, and writes it back.
// Uses json.RawMessage to preserve unknown fields in existing alloc entries.
func patchGenesisWithDRB(genesisPath string, runtimeBytecode []byte) error {
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

	// Compute code-namespace address for the implementation.
	// See: Predeploys.sol predeployToCodeNamespace()
	proxyAddr := common.HexToAddress(drbPredeployAddress)
	codeAddr := predeployToCodeNamespace(proxyAddr)

	// Detect alloc key format (with or without 0x prefix) from existing entries
	has0xPrefix := false
	for key := range alloc {
		if strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X") {
			has0xPrefix = true
		}
		break
	}

	formatAddr := func(addr common.Address) string {
		hex := strings.ToLower(addr.Hex())
		if !has0xPrefix {
			return strings.TrimPrefix(hex, "0x")
		}
		return hex
	}

	codeAddrHex := formatAddr(codeAddr)
	proxyAddrHex := formatAddr(proxyAddr)

	// Add implementation bytecode at code-namespace address
	implJSON, err := json.Marshal(map[string]interface{}{
		"code":    "0x" + hex.EncodeToString(runtimeBytecode),
		"balance": "0x0",
	})
	if err != nil {
		return err
	}
	alloc[codeAddrHex] = implJSON
	if err := patchProxyImplementationSlot(alloc, proxyAddrHex, codeAddr); err != nil {
		return err
	}

	allocJSON, err := json.Marshal(alloc)
	if err != nil {
		return err
	}
	genesis["alloc"] = allocJSON

	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(genesisPath, output, 0644)
}

// patchProxyImplementationSlot updates the ERC1967 implementation slot on an existing
// proxy alloc entry, preserving all other fields (code, balance, nonce, etc.).
func patchProxyImplementationSlot(alloc map[string]json.RawMessage, proxyAddrHex string, implAddr common.Address) error {
	existing, ok := alloc[proxyAddrHex]
	if !ok {
		return fmt.Errorf("proxy alloc entry not found at %s", proxyAddrHex)
	}

	// Parse as generic map to preserve all fields
	var proxyData map[string]json.RawMessage
	if err := json.Unmarshal(existing, &proxyData); err != nil {
		return fmt.Errorf("failed to parse proxy alloc: %w", err)
	}

	// Parse existing storage or create new
	var storage map[string]string
	if raw, ok := proxyData["storage"]; ok {
		if err := json.Unmarshal(raw, &storage); err != nil {
			return fmt.Errorf("failed to parse proxy storage: %w", err)
		}
	}
	if storage == nil {
		storage = make(map[string]string)
	}

	// Set ERC1967 implementation slot
	storage[erc1967ImplementationSlot] = common.BytesToHash(implAddr.Bytes()).Hex()

	storageJSON, err := json.Marshal(storage)
	if err != nil {
		return err
	}
	proxyData["storage"] = storageJSON

	updated, err := json.Marshal(proxyData)
	if err != nil {
		return err
	}
	alloc[proxyAddrHex] = updated
	return nil
}

// predeployToCodeNamespace computes the implementation address for a predeploy proxy.
// Mirrors Predeploys.sol: (addr & 0xffff) | 0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000
func predeployToCodeNamespace(addr common.Address) common.Address {
	prefix := common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000")
	var result common.Address
	copy(result[:], prefix[:])
	result[18] = addr[18]
	result[19] = addr[19]
	return result
}

// downloadAndExtractSingleFile downloads an npm tarball and extracts a specific file.
//nolint:unused // called by downloadDRBArtifact
func downloadAndExtractSingleFile(ctx context.Context, tarballURL, targetFile string) ([]byte, error) {
	req, err := newHTTPRequest(ctx, tarballURL)
	if err != nil {
		return nil, err
	}

	client := newHTTPClient(downloadTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tarball download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("tarball download returned status %d", resp.StatusCode)
	}

	return extractFileFromTarball(resp.Body, "package/"+targetFile)
}
