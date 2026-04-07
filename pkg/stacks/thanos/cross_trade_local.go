package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/abis"
	"go.uber.org/zap"
)

// DeployCrossTradeLocalInput defines the parameters required to deploy CrossTrade contracts
// via L1 OptimismPortal depositTransaction calls. Matches PRD v2.1 interface definition.
type DeployCrossTradeLocalInput struct {
	L1RPCUrl             string      `json:"l1_rpc_url"`
	L1ChainID            uint64      `json:"l1_chain_id"`
	DeployerPrivateKey   string      `json:"deployer_private_key"`
	L2RPCUrl             string      `json:"l2_rpc_url"`
	L2ChainID            uint64      `json:"l2_chain_id"`
	OptimismPortalProxy  string      `json:"optimism_portal_proxy"`
	CrossDomainMessenger string      `json:"cross_domain_messenger"`
	L1CrossTradeProxy    string      `json:"l1_cross_trade_proxy"`
	L2toL2CrossTradeL1   string      `json:"l2_to_l2_cross_trade_l1"`
	SupportedTokens      []TokenPair `json:"supported_tokens"`
}

// TokenPair represents an L1/L2 token pair to be registered in CrossTrade.
type TokenPair struct {
	L1Token string `json:"l1_token"`
	L2Token string `json:"l2_token"`
	Symbol  string `json:"symbol"`
}

// DeployCrossTradeLocalOutput contains the deployed contract addresses and registration tx hashes.
// Matches PRD v2.1 interface definition. Bytecode fields are intentionally absent (stored in
// cross_trade_local_bytecodes.go constants to keep this struct clean).
type DeployCrossTradeLocalOutput struct {
	L2CrossTradeProxy     string `json:"l2_cross_trade_proxy"`
	L2CrossTrade          string `json:"l2_cross_trade"`
	L2toL2CrossTradeProxy string `json:"l2_to_l2_cross_trade_proxy"`
	L2toL2CrossTradeL2    string `json:"l2_to_l2_cross_trade_l2"`
	L1RegistrationTxHash  string `json:"l1_registration_tx_hash"`
	L1RegistrationL2L2Tx  string `json:"l1_registration_l2_l2_tx"`
}

// DeployCrossTradeLocal deploys the CrossTrade L2 contracts via L1 OptimismPortal
// depositTransaction calls. Two pairs are deployed sequentially:
//   - L2CrossTrade impl+proxy (L2→L1 trades)
//   - L2toL2CrossTradeL2 impl+proxy (L2→L2 trades)
//
// Bytecode is loaded from package-level constants in cross_trade_local_bytecodes.go.
// This is the local (Docker Compose) deployment path, distinct from the AWS/Foundry path
// in cross_trade.go.
func (t *ThanosStack) DeployCrossTradeLocal(
	ctx context.Context,
	input *DeployCrossTradeLocalInput,
) (*DeployCrossTradeLocalOutput, error) {
	// ---------------------------------------------------------------------------
	// 1. Client initialization
	// ---------------------------------------------------------------------------
	l1Client, err := ethclient.Dial(input.L1RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC %s: %w", input.L1RPCUrl, err)
	}
	defer l1Client.Close()

	l2Client, err := ethclient.Dial(input.L2RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L2 RPC %s: %w", input.L2RPCUrl, err)
	}
	defer l2Client.Close()

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(input.DeployerPrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployer private key: %w", err)
	}
	deployerAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	l1ChainID := new(big.Int).SetUint64(input.L1ChainID)
	l2ChainID := new(big.Int).SetUint64(input.L2ChainID)

	t.logger.Infof("DeployCrossTradeLocal: deployer=%s l1ChainID=%s l2ChainID=%s",
		deployerAddr.Hex(), l1ChainID.String(), l2ChainID.String())

	// ---------------------------------------------------------------------------
	// 2. OptimismPortal binding instance
	// ---------------------------------------------------------------------------
	portal, err := abis.NewOptimismPortal(common.HexToAddress(input.OptimismPortalProxy), l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind OptimismPortal at %s: %w", input.OptimismPortalProxy, err)
	}

	// ---------------------------------------------------------------------------
	// 3. bind.TransactOpts (EIP-1559-compatible signer via NewKeyedTransactorWithChainID)
	// ---------------------------------------------------------------------------
	opts, err := bind.NewKeyedTransactorWithChainID(privKey, l1ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	// GasLimit is set per-call in sendDepositCreation/sendDepositCall to account for
	// worst-case ResourceMetering gas burn (see those functions for details).
	opts.Context = ctx

	// ---------------------------------------------------------------------------
	// 4. Obtain L2 deployer nonce at deployment start (creation txs consume nonces)
	// ---------------------------------------------------------------------------
	l2Nonce, err := l2Client.PendingNonceAt(ctx, deployerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 pending nonce for %s: %w", deployerAddr.Hex(), err)
	}
	t.logger.Infof("L2 deployer nonce at start: %d", l2Nonce)

	// ---------------------------------------------------------------------------
	// 5. ABI JSON strings from abis package constants
	// ---------------------------------------------------------------------------
	l2CrossTradeImplABI := abis.L2CrossTradeABI
	l2CrossTradeProxyABI := abis.L2CrossTradeProxyABI
	l2toL2CrossTradeL2ImplABI := abis.L2toL2CrossTradeL2ABI
	l2toL2CrossTradeProxyABI := abis.L2toL2CrossTradeProxyABI

	// Parse ABI objects for building registerToken callbacks
	parsedL2CrossTradeImplABI, err := abi.JSON(strings.NewReader(l2CrossTradeImplABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse L2CrossTrade ABI: %w", err)
	}
	parsedL2toL2ImplABI, err := abi.JSON(strings.NewReader(l2toL2CrossTradeL2ImplABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse L2toL2CrossTradeL2 ABI: %w", err)
	}

	// ---------------------------------------------------------------------------
	// 6. registerToken callbacks — L2CrossTrade uses 3-param, L2toL2 uses 6-param
	// ---------------------------------------------------------------------------
	// L2CrossTrade: registerToken(address l1token, address l2token, uint256 l1chainId)
	l2CrossTradeRegisterTokenFn := func(_ common.Address, token TokenPair) ([]byte, error) {
		return parsedL2CrossTradeImplABI.Pack("registerToken",
			common.HexToAddress(token.L1Token),
			common.HexToAddress(token.L2Token),
			l1ChainID,
		)
	}

	// L2toL2CrossTradeL2: registerToken(address l1token, address l2SourceToken, address l2DestinationToken,
	//   uint256 l1ChainId, uint256 l2SourceChainId, uint256 l2DestinationChainId)
	// Phase 1: single L2 — source and destination chain IDs are both l2ChainID.
	l2toL2RegisterTokenFn := func(_ common.Address, token TokenPair) ([]byte, error) {
		return parsedL2toL2ImplABI.Pack("registerToken",
			common.HexToAddress(token.L1Token),
			common.HexToAddress(token.L2Token),
			common.HexToAddress(token.L2Token),
			l1ChainID,
			l2ChainID,
			l2ChainID,
		)
	}

	// ---------------------------------------------------------------------------
	// Phase 1/2: Deploy L2CrossTrade impl+proxy (L2→L1 pair)
	// Bytecode from cross_trade_local_bytecodes.go package constants.
	// ---------------------------------------------------------------------------
	t.logger.Infof("=== Phase 1/2: Deploying L2CrossTrade pair ===")
	l2CrossTradeResult, err := deployL2CrossTradePair(
		ctx, &portal.OptimismPortalTransactor, opts, l1Client, l2Client,
		deployerAddr, l2Nonce,
		L2CrossTradeBytecode, L2CrossTradeProxyBytecode,
		l2CrossTradeProxyABI, l2CrossTradeImplABI,
		common.HexToAddress(input.CrossDomainMessenger),
		common.HexToAddress(input.L1CrossTradeProxy),
		l1ChainID,
		input.SupportedTokens,
		l2CrossTradeRegisterTokenFn,
		t.logger,
	)
	if err != nil {
		return nil, fmt.Errorf("L2CrossTrade pair deployment failed: %w", err)
	}
	t.logger.Infof("L2CrossTrade pair deployed: impl=%s proxy=%s",
		l2CrossTradeResult.ImplAddr.Hex(), l2CrossTradeResult.ProxyAddr.Hex())

	// ---------------------------------------------------------------------------
	// Phase 2/2: Deploy L2toL2CrossTradeL2 impl+proxy (L2→L2 pair)
	// L2 nonce: L2CrossTrade pair consumed 2 creation txs (nonce+0, nonce+1).
	// Function calls do not consume L2 deployer nonce.
	// ---------------------------------------------------------------------------
	l2toL2Nonce := l2Nonce + 2
	t.logger.Infof("=== Phase 2/2: Deploying L2toL2CrossTradeL2 pair (l2Nonce=%d) ===", l2toL2Nonce)
	l2toL2Result, err := deployL2CrossTradePair(
		ctx, &portal.OptimismPortalTransactor, opts, l1Client, l2Client,
		deployerAddr, l2toL2Nonce,
		L2toL2CrossTradeL2Bytecode, L2toL2CrossTradeProxyBytecode,
		l2toL2CrossTradeProxyABI, l2toL2CrossTradeL2ImplABI,
		common.HexToAddress(input.CrossDomainMessenger),
		common.HexToAddress(input.L2toL2CrossTradeL1),
		l1ChainID,
		input.SupportedTokens,
		l2toL2RegisterTokenFn,
		t.logger,
	)
	if err != nil {
		return nil, fmt.Errorf("L2toL2CrossTradeL2 pair deployment failed: %w", err)
	}
	t.logger.Infof("L2toL2CrossTradeL2 pair deployed: impl=%s proxy=%s",
		l2toL2Result.ImplAddr.Hex(), l2toL2Result.ProxyAddr.Hex())

	// ---------------------------------------------------------------------------
	// Return all 4 deployed contract addresses
	// ---------------------------------------------------------------------------
	output := &DeployCrossTradeLocalOutput{
		L2CrossTrade:          l2CrossTradeResult.ImplAddr.Hex(),
		L2CrossTradeProxy:     l2CrossTradeResult.ProxyAddr.Hex(),
		L2toL2CrossTradeL2:    l2toL2Result.ImplAddr.Hex(),
		L2toL2CrossTradeProxy: l2toL2Result.ProxyAddr.Hex(),
	}
	t.logger.Infof("CrossTrade local deployment complete. Addresses: %+v", output)
	return output, nil
}

// waitForContractCode polls L2 for contract deployment confirmation via eth_getCode.
// Returns nil when the contract is deployed (code length > 0) or error on timeout.
// Per D-04: used for creation deposit tx verification.
func waitForContractCode(ctx context.Context, l2Client *ethclient.Client, addr common.Address, logger *zap.SugaredLogger) error {
	for attempt := 1; attempt <= 60; attempt++ {
		if attempt%10 == 0 {
			logger.Infof("waiting for contract at %s (attempt %d/60)", addr.Hex(), attempt)
		}

		code, err := l2Client.CodeAt(ctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to call CodeAt for %s: %w", addr.Hex(), err)
		}
		if len(code) > 0 {
			logger.Infof("contract deployed at %s (attempt %d/60)", addr.Hex(), attempt)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("contract at %s not deployed after 120s", addr.Hex())
}

// verifyDepositCallEffect checks that a function-call deposit tx actually executed on L2
// by calling a view function on the target contract to verify state change.
// Per D-04: used for non-creation deposit tx verification.
func verifyDepositCallEffect(ctx context.Context, l2Client *ethclient.Client, contractAddr common.Address, checkCalldata []byte, logger *zap.SugaredLogger) error {
	for attempt := 1; attempt <= 60; attempt++ {
		if attempt%10 == 0 {
			logger.Infof("verifying deposit call effect at %s (attempt %d/60)", contractAddr.Hex(), attempt)
		}

		result, err := l2Client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddr,
			Data: checkCalldata,
		}, nil)
		if err == nil && len(result) > 0 {
			logger.Infof("deposit call effect verified at %s (attempt %d/60)", contractAddr.Hex(), attempt)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("deposit call effect not verified at %s after 120s", contractAddr.Hex())
}

// sendDepositCreation sends an L1 OptimismPortal.depositTransaction for contract creation.
// The _to field is address(0) and _isCreation is true. Waits for L1 receipt.
// Per D-09: fails fast if the L1 tx reverts.
//
// L1 gas limit: ResourceMetering in OptimismPortal burns L1 gas proportional to:
//   gasCost = gasLimit * prevBaseFee / max(block.basefee, 1 gwei)
// When block.basefee == prevBaseFee == 1 gwei (Sepolia minimum), gasCost == gasLimit.
// Using eth_estimateGas (opts.GasLimit=0) is unsafe because the estimate is computed at
// the current basefee, but the tx may land in a block with much lower basefee, causing
// Burn.gas() to exceed the gas limit. We set a fixed L1 gas limit = L2 gas + overhead.
func sendDepositCreation(
	ctx context.Context,
	portal *abis.OptimismPortalTransactor,
	opts *bind.TransactOpts,
	l1Client *ethclient.Client,
	bytecode []byte,
	gasLimit uint64,
	logger *zap.SugaredLogger,
) (*types.Receipt, error) {
	// Set L1 gas limit to cover worst-case ResourceMetering burn (1 gwei basefee).
	// 300_000 overhead covers depositTransaction execution + SSTORE + event emit.
	opts.GasLimit = gasLimit + 300_000
	tx, err := portal.DepositTransaction(opts, common.Address{}, big.NewInt(0), big.NewInt(0), gasLimit, true, bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to send deposit creation tx: %w", err)
	}
	logger.Infof("L1 deposit creation tx sent: %s", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for deposit creation tx receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deposit creation tx reverted (tx: %s, gas used: %d)", tx.Hash().Hex(), receipt.GasUsed)
	}
	return receipt, nil
}

// sendDepositCall sends an L1 OptimismPortal.depositTransaction for a function call.
// The _isCreation flag is false. Waits for L1 receipt.
// Per D-09: fails fast if the L1 tx reverts.
func sendDepositCall(
	ctx context.Context,
	portal *abis.OptimismPortalTransactor,
	opts *bind.TransactOpts,
	l1Client *ethclient.Client,
	to common.Address,
	calldata []byte,
	gasLimit uint64,
	logger *zap.SugaredLogger,
) (*types.Receipt, error) {
	// Set L1 gas limit to cover worst-case ResourceMetering burn (see sendDepositCreation).
	opts.GasLimit = gasLimit + 300_000
	tx, err := portal.DepositTransaction(opts, to, big.NewInt(0), big.NewInt(0), gasLimit, false, calldata)
	if err != nil {
		return nil, fmt.Errorf("failed to send deposit call tx to %s: %w", to.Hex(), err)
	}
	logger.Infof("L1 deposit call tx sent to %s: %s", to.Hex(), tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for deposit call tx receipt (to: %s): %w", to.Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deposit call tx reverted (to: %s, tx: %s, gas used: %d)", to.Hex(), tx.Hash().Hex(), receipt.GasUsed)
	}
	return receipt, nil
}

// crossTradePairResult holds the deployed impl and proxy addresses for a CrossTrade pair.
type crossTradePairResult struct {
	ImplAddr  common.Address
	ProxyAddr common.Address
}

// registerTokenFunc is a callback that produces the ABI-encoded calldata for registerToken.
// Allows deployL2CrossTradePair to be reused for both L2CrossTrade (3-param) and
// L2toL2CrossTradeL2 (6-param) contracts without hardcoding the parameter list.
type registerTokenFunc func(proxyAddr common.Address, token TokenPair) ([]byte, error)

// deployL2CrossTradePair deploys a CrossTrade impl+proxy pair on L2 via 7 Deposit Tx steps.
// Each step includes L2 execution verification (per D-04, SDK-06):
//   - Creation txs: getCode polling (waitForContractCode)
//   - Function call txs: view function call (verifyDepositCallEffect)
func deployL2CrossTradePair(
	ctx context.Context,
	portal *abis.OptimismPortalTransactor,
	opts *bind.TransactOpts,
	l1Client *ethclient.Client,
	l2Client *ethclient.Client,
	deployerAddr common.Address,
	l2Nonce uint64,
	implBytecode []byte,
	proxyBytecode []byte,
	proxyABIJSON string,
	implABIJSON string,
	crossDomainMessenger common.Address,
	l1CrossTradeAddr common.Address,
	l1ChainID *big.Int,
	tokens []TokenPair,
	registerTokenFn registerTokenFunc,
	logger *zap.SugaredLogger,
) (*crossTradePairResult, error) {
	proxyABI, err := abi.JSON(strings.NewReader(proxyABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy ABI JSON: %w", err)
	}
	implABI, err := abi.JSON(strings.NewReader(implABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse impl ABI JSON: %w", err)
	}

	// ---------------------------------------------------------------------------
	// Step 1: Deploy L2CrossTrade impl (creation Deposit Tx)
	// ---------------------------------------------------------------------------
	implAddr := crypto.CreateAddress(deployerAddr, l2Nonce)
	logger.Infof("Step 1: deploying L2CrossTrade impl, predicted addr=%s", implAddr.Hex())
	if _, err := sendDepositCreation(ctx, portal, opts, l1Client, implBytecode, 3_000_000, logger); err != nil {
		return nil, fmt.Errorf("step 1 failed: %w", err)
	}
	if err := waitForContractCode(ctx, l2Client, implAddr, logger); err != nil {
		return nil, fmt.Errorf("step 1 L2 verification failed: %w", err)
	}
	logger.Infof("Step 1: L2CrossTrade impl deployed at %s", implAddr.Hex())

	// ---------------------------------------------------------------------------
	// Step 2: Deploy L2CrossTradeProxy (creation Deposit Tx)
	// ---------------------------------------------------------------------------
	proxyAddr := crypto.CreateAddress(deployerAddr, l2Nonce+1)
	logger.Infof("Step 2: deploying L2CrossTradeProxy, predicted addr=%s", proxyAddr.Hex())
	if _, err := sendDepositCreation(ctx, portal, opts, l1Client, proxyBytecode, 3_000_000, logger); err != nil {
		return nil, fmt.Errorf("step 2 failed: %w", err)
	}
	if err := waitForContractCode(ctx, l2Client, proxyAddr, logger); err != nil {
		return nil, fmt.Errorf("step 2 L2 verification failed: %w", err)
	}
	logger.Infof("Step 2: L2CrossTradeProxy deployed at %s", proxyAddr.Hex())

	// ---------------------------------------------------------------------------
	// Step 3: setAliveImplementation2(implAddr, true) — Pitfall 2 prevention
	// Must be called before setSelectorImplementations2.
	// ---------------------------------------------------------------------------
	calldata, err := proxyABI.Pack("setAliveImplementation2", implAddr, true)
	if err != nil {
		return nil, fmt.Errorf("step 3: failed to pack setAliveImplementation2 calldata: %w", err)
	}
	if _, err := sendDepositCall(ctx, portal, opts, l1Client, proxyAddr, calldata, 500_000, logger); err != nil {
		return nil, fmt.Errorf("step 3 failed: %w", err)
	}
	// L2 verification: aliveImplementation(implAddr) should return true
	checkCalldata, err := proxyABI.Pack("aliveImplementation", implAddr)
	if err != nil {
		return nil, fmt.Errorf("step 3: failed to pack aliveImplementation check calldata: %w", err)
	}
	if err := verifyDepositCallEffect(ctx, l2Client, proxyAddr, checkCalldata, logger); err != nil {
		return nil, fmt.Errorf("step 3 L2 verification failed: %w", err)
	}
	logger.Infof("Step 3: setAliveImplementation2 for impl %s — L2 verified", implAddr.Hex())

	// ---------------------------------------------------------------------------
	// Step 4: setSelectorImplementations2(selectors, implAddr)
	// Extract all function selectors from impl ABI (per D-08).
	// ---------------------------------------------------------------------------
	var selectors [][4]byte
	for _, method := range implABI.Methods {
		var sel [4]byte
		copy(sel[:], method.ID[:4])
		selectors = append(selectors, sel)
	}
	calldata, err = proxyABI.Pack("setSelectorImplementations2", selectors, implAddr)
	if err != nil {
		return nil, fmt.Errorf("step 4: failed to pack setSelectorImplementations2 calldata: %w", err)
	}
	if _, err := sendDepositCall(ctx, portal, opts, l1Client, proxyAddr, calldata, 500_000, logger); err != nil {
		return nil, fmt.Errorf("step 4 failed: %w", err)
	}
	// L2 verification: selectorImplementation(selectors[0]) should return implAddr
	if len(selectors) > 0 {
		checkCalldata, err = proxyABI.Pack("selectorImplementation", selectors[0])
		if err != nil {
			return nil, fmt.Errorf("step 4: failed to pack selectorImplementation check calldata: %w", err)
		}
		if err := verifyDepositCallEffect(ctx, l2Client, proxyAddr, checkCalldata, logger); err != nil {
			return nil, fmt.Errorf("step 4 L2 verification failed: %w", err)
		}
	}
	logger.Infof("Step 4: setSelectorImplementations2 with %d selectors — L2 verified", len(selectors))

	// ---------------------------------------------------------------------------
	// Step 5: initialize(crossDomainMessenger)
	// initialize is in proxy ABI; proxy stores the messenger directly.
	// ---------------------------------------------------------------------------
	calldata, err = proxyABI.Pack("initialize", crossDomainMessenger)
	if err != nil {
		return nil, fmt.Errorf("step 5: failed to pack initialize calldata: %w", err)
	}
	if _, err := sendDepositCall(ctx, portal, opts, l1Client, proxyAddr, calldata, 500_000, logger); err != nil {
		return nil, fmt.Errorf("step 5 failed: %w", err)
	}
	// L2 verification: crossDomainMessenger() view function should return the set address.
	checkCalldata, err = proxyABI.Pack("crossDomainMessenger")
	if err != nil {
		return nil, fmt.Errorf("step 5: failed to pack crossDomainMessenger check calldata: %w", err)
	}
	if err := verifyDepositCallEffect(ctx, l2Client, proxyAddr, checkCalldata, logger); err != nil {
		return nil, fmt.Errorf("step 5 L2 verification failed: %w", err)
	}
	logger.Infof("Step 5: initialize with messenger %s — L2 verified", crossDomainMessenger.Hex())

	// ---------------------------------------------------------------------------
	// Step 6: setChainInfo(l1CrossTradeAddr, l1ChainID)
	// ---------------------------------------------------------------------------
	calldata, err = proxyABI.Pack("setChainInfo", l1CrossTradeAddr, l1ChainID)
	if err != nil {
		return nil, fmt.Errorf("step 6: failed to pack setChainInfo calldata: %w", err)
	}
	if _, err := sendDepositCall(ctx, portal, opts, l1Client, proxyAddr, calldata, 500_000, logger); err != nil {
		return nil, fmt.Errorf("step 6 failed: %w", err)
	}
	// L2 verification: chainData(l1ChainID) should return l1CrossTradeAddr (non-zero address).
	// Falls back to sleep if chainData view function is not available in proxy ABI.
	if _, hasChainData := proxyABI.Methods["chainData"]; hasChainData {
		checkCalldata, err = proxyABI.Pack("chainData", l1ChainID)
		if err != nil {
			return nil, fmt.Errorf("step 6: failed to pack chainData check calldata: %w", err)
		}
		if err := verifyDepositCallEffect(ctx, l2Client, proxyAddr, checkCalldata, logger); err != nil {
			return nil, fmt.Errorf("step 6 L2 verification failed: %w", err)
		}
	} else {
		logger.Infof("no view function for chainInfo verification, waited 10s")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	logger.Infof("Step 6: setChainInfo l1=%s chainId=%s — L2 verified", l1CrossTradeAddr.Hex(), l1ChainID.String())

	// ---------------------------------------------------------------------------
	// Step 7: registerToken — for each token pair.
	// registerToken is an impl function called through the proxy.
	// The calldata is produced by registerTokenFn to support both 3-param (L2CrossTrade)
	// and 6-param (L2toL2CrossTradeL2) variants.
	// ---------------------------------------------------------------------------
	for i, token := range tokens {
		l1Token := common.HexToAddress(token.L1Token)
		l2Token := common.HexToAddress(token.L2Token)

		calldata, err = registerTokenFn(proxyAddr, token)
		if err != nil {
			return nil, fmt.Errorf("step 7 (token %d): failed to build registerToken calldata: %w", i, err)
		}
		if _, err := sendDepositCall(ctx, portal, opts, l1Client, proxyAddr, calldata, 500_000, logger); err != nil {
			return nil, fmt.Errorf("step 7 (token %d) failed: %w", i, err)
		}
		// L2 verification: registerCheck(l1ChainId, l1Token, l2Token) should return true.
		// Falls back to sleep if registerCheck view function is not available in proxy ABI.
		if _, hasRegCheck := proxyABI.Methods["registerCheck"]; hasRegCheck {
			checkCalldata, err = proxyABI.Pack("registerCheck", l1ChainID, l1Token, l2Token)
			if err != nil {
				return nil, fmt.Errorf("step 7 (token %d): failed to pack registerCheck calldata: %w", i, err)
			}
			if err := verifyDepositCallEffect(ctx, l2Client, proxyAddr, checkCalldata, logger); err != nil {
				return nil, fmt.Errorf("step 7 (token %d) L2 verification failed: %w", i, err)
			}
		} else {
			logger.Infof("no view function for registerToken verification (token %d), waited 10s", i)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
			}
		}
		logger.Infof("Step 7: registerToken %s -> %s — L2 verified", token.L1Token, token.L2Token)
	}

	return &crossTradePairResult{ImplAddr: implAddr, ProxyAddr: proxyAddr}, nil
}
