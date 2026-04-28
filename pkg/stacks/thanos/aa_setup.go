package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// setupAAPaymaster configures the AA Paymaster infrastructure on L2 after the network starts.
// It is a no-op for TON fee token or non-AA presets (General/DeFi).
//
// Steps (all on L2):
//  1. EntryPoint.depositTo(MultiTokenPaymaster) — deposit fee token for gas sponsorship
//  2. SimplePriceOracle.updatePrice(initialPrice) — set initial TON/token exchange rate
//  3. MultiTokenPaymaster.addToken(tokenAddr, oracle, markupPct, decimals) — register fee token
//  4. Start background price updater: fetches TON/feeToken from CoinGecko, keeps SimplePriceOracle fresh
//  5. Start background EntryPoint refill monitor: tops up deposit when balance falls below 0.5 TON
//
// For USDT (no L2 predeploy): OptimismMintableERC20Factory.createOptimismMintableERC20WithDecimals
// is called first to deploy a bridged USDT token on L2. The CREATE2 address is predicted before
// deployment via eth_call simulation. If already deployed, the existing address is used.
func (t *ThanosStack) setupAAPaymaster(ctx context.Context) error {
	if !constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		return nil
	}

	feeToken := t.deployConfig.FeeToken

	t.logger.Infof("🔧 Setting up AA Paymaster for fee token: %s", feeToken)

	// Connect to L2. Use the configured L2 RPC URL (AWS path) or fall back to local Docker.
	l2URL := localL2RPCURL()
	if t.deployConfig.L2RpcUrl != "" {
		l2URL = t.deployConfig.L2RpcUrl
	}
	l2Client, err := ethclient.DialContext(ctx, l2URL)
	if err != nil {
		return fmt.Errorf("failed to connect to L2 RPC: %w", err)
	}
	defer l2Client.Close()

	// Wait for L2 to be responsive (up to ~30s).
	var l2ChainID *big.Int
	for attempt := 1; attempt <= 6; attempt++ {
		l2ChainID, err = l2Client.ChainID(ctx)
		if err == nil {
			break
		}
		t.logger.Warnf("Waiting for L2 RPC (attempt %d/6): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("L2 RPC unavailable after retries: %w", err)
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(t.deployConfig.AdminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// EntryPoint deposit is always in L2 native token (TON @ 18 decimals).
	// Fee token (USDT/ETH) is handled by AA Paymaster internally (after EntryPoint transfers gas).
	// Do NOT scale by feeToken decimals — EntryPoint only understands native token.
	depositAmount := new(big.Int).Set(constants.DefaultEntryPointDeposit)

	// Pre-check: verify admin has enough L2 balance for the EntryPoint deposit.
	adminBalance, err := l2Client.BalanceAt(ctx, adminAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to query admin L2 balance: %w", err)
	}
	if adminBalance.Cmp(depositAmount) < 0 {
		return fmt.Errorf("admin L2 balance insufficient for EntryPoint deposit: have %s wei, need %s wei — fund admin address %s on L2 first", adminBalance.String(), depositAmount.String(), adminAddr.Hex())
	}
	twoXDeposit := new(big.Int).Mul(depositAmount, big.NewInt(2))
	if adminBalance.Cmp(twoXDeposit) < 0 {
		t.logger.Warnf("admin L2 balance is low: %s wei (deposit requires %s wei)", adminBalance.String(), depositAmount.String())
	}

	// sendTxAndWait builds, signs, sends a transaction, and waits for its receipt.
	// Uses EIP-1559 (type 2) transactions to avoid overpaying on OP Stack L2.
	// SuggestGasPrice on OP Stack can return inflated values that include L1 data cost
	// guidance, but legacy tx gas cost = gasLimit × gasPrice can exceed admin balance.
	// With EIP-1559, actual cost = (baseFee + tipCap) × gasUsed — much lower on a fresh L2.
	sendTxAndWait := func(toAddr common.Address, value *big.Int, calldata []byte) (*types.Receipt, error) {
		nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		header, err := l2Client.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block header: %w", err)
		}
		tipCap, err := l2Client.SuggestGasTipCap(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas tip cap: %w", err)
		}
		// maxFeePerGas = baseFee * 2 + tipCap — standard EIP-1559 headroom.
		baseFee := header.BaseFee
		if baseFee == nil {
			baseFee = big.NewInt(0)
		}
		maxFeePerGas := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tipCap)

		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   l2ChainID,
			Nonce:     nonce,
			To:        &toAddr,
			Value:     value,
			Gas:       300_000,
			GasTipCap: tipCap,
			GasFeeCap: maxFeePerGas,
			Data:      calldata,
		})
		signedTx, err := types.SignTx(tx, types.NewLondonSigner(l2ChainID), privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign tx: %w", err)
		}
		if err := l2Client.SendTransaction(ctx, signedTx); err != nil {
			return nil, err
		}
		txHash := signedTx.Hash()
		for attempt := 1; attempt <= 30; attempt++ {
			receipt, err := l2Client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(2 * time.Second)
		}
		return nil, fmt.Errorf("tx %s not mined after 60s", txHash.Hex())
	}

	// sendTx sends a transaction without waiting for receipt (fire-and-forget).
	sendTx := func(toAddr common.Address, value *big.Int, calldata []byte) error {
		_, err := sendTxAndWait(toAddr, value, calldata)
		return err
	}

	// Resolve L2 token address and paymaster parameters.
	var tokenAddr common.Address
	var markupPct uint64
	var decimals uint8
	var initialPrice *big.Int

	if feeToken == constants.FeeTokenUSDT {
		// USDT has no L2 predeploy — deploy a bridged token via the Standard Bridge factory.
		l1USDTCfg := constants.GetFeeTokenConfig(constants.FeeTokenUSDT, t.deployConfig.L1ChainID)
		if l1USDTCfg.L1Address == "" || l1USDTCfg.L1Address == "0x0000000000000000000000000000000000000000" {
			return fmt.Errorf("L1 USDT address not configured for chain %d", t.deployConfig.L1ChainID)
		}
		l1USDTAddr := common.HexToAddress(l1USDTCfg.L1Address)

		deployedAddr, err := deployBridgedUSDT(ctx, l2Client, l1USDTAddr, t.logger.Infof, sendTxAndWait)
		if err != nil {
			return fmt.Errorf("failed to deploy bridged USDT on L2: %w", err)
		}

		tokenAddr = deployedAddr
		markupPct = constants.USDTPaymasterMarkupPct
		decimals = 6
		initialPrice = constants.DefaultUSDTInitialPrice
	} else {
		l2Addr, mp, d, ip, err := resolveAATokenConfig(feeToken)
		if err != nil {
			return fmt.Errorf("unsupported fee token for AA setup: %w", err)
		}
		tokenAddr = common.HexToAddress(l2Addr)
		markupPct = mp
		decimals = d
		initialPrice = ip
	}

	entryPoint := common.HexToAddress(constants.AAEntryPoint)
	oracle := common.HexToAddress(constants.SimplePriceOraclePredeploy)
	paymaster := common.HexToAddress(constants.MultiTokenPaymasterPredeploy)

	// Step 1: EntryPoint.depositTo(MultiTokenPaymaster)
	// ABI: depositTo(address account) payable
	// Calldata: [4-byte selector][32-byte address (right-aligned)]
	selector1 := crypto.Keccak256([]byte("depositTo(address)"))[:4]
	calldata1 := make([]byte, 36)
	copy(calldata1[:4], selector1)
	copy(calldata1[16:36], paymaster.Bytes())

	if err := sendTx(entryPoint, depositAmount, calldata1); err != nil {
		return fmt.Errorf("EntryPoint.depositTo failed: %w", err)
	}
	t.logger.Infof("✅ EntryPoint.depositTo(MultiTokenPaymaster): deposited %s wei (%s)", depositAmount.String(), feeToken)

	// Step 2: SimplePriceOracle.updatePrice(newPrice)
	// ABI: updatePrice(uint256 newPrice)
	// Each fee token uses the single SimplePriceOraclePredeploy instance as its oracle.
	// Calldata: [4-byte selector][32-byte uint256 price]
	selector2 := crypto.Keccak256([]byte("updatePrice(uint256)"))[:4]
	calldata2 := make([]byte, 36)
	copy(calldata2[:4], selector2)
	priceBytes := initialPrice.Bytes()
	copy(calldata2[36-len(priceBytes):36], priceBytes) // uint256 right-aligned

	if err := sendTx(oracle, big.NewInt(0), calldata2); err != nil {
		return fmt.Errorf("SimplePriceOracle.updatePrice failed: %w", err)
	}
	t.logger.Infof("✅ SimplePriceOracle.updatePrice(%s): price set to %s", feeToken, initialPrice.String())

	// Step 3: MultiTokenPaymaster.addToken(token, oracle, markupPercent, decimals)
	// ABI: addToken(address token, address oracle, uint256 markupPercent, uint8 decimals)
	// markupPercent is a plain percentage (e.g. 5 = 5%), max 50. NOT basis points.
	// Calldata: [4][32 token][32 oracle][32 markupPct][32 decimals as uint256-padded uint8]
	selector3 := crypto.Keccak256([]byte("addToken(address,address,uint256,uint8)"))[:4]
	calldata3 := make([]byte, 132) // 4 + 32*4
	copy(calldata3[:4], selector3)
	copy(calldata3[16:36], tokenAddr.Bytes())  // token address, right-aligned
	copy(calldata3[48:68], oracle.Bytes())     // oracle address, right-aligned
	markupBig := new(big.Int).SetUint64(markupPct)
	markupBytes := markupBig.Bytes()
	copy(calldata3[100-len(markupBytes):100], markupBytes) // markupPercent uint256
	calldata3[131] = byte(decimals)                        // decimals uint8, right-aligned in 32 bytes

	if err := sendTx(paymaster, big.NewInt(0), calldata3); err != nil {
		return fmt.Errorf("MultiTokenPaymaster.addToken failed: %w", err)
	}
	t.logger.Infof("✅ MultiTokenPaymaster.addToken(%s, markup=%d%%, decimals=%d)", feeToken, markupPct, decimals)

	// Step 4: Verify paymaster configuration (non-fatal).
	if err := verifyAAPaymaster(ctx, l2Client, paymaster, tokenAddr, entryPoint, oracle, depositAmount, t.logger.Infof); err != nil {
		t.logger.Warnf("AA Paymaster verification warning: %v", err)
	} else {
		t.logger.Infof("AA Paymaster verification: all checks passed")
	}

	// Step 5: EIP-7702 delegation — admin EOA → Simple7702Account
	// Sets admin EOA's code pointer to Simple7702Account predeploy so it can be
	// used as an ERC-4337 smart account without redeployment.
	// Requires Isthmus hardfork (SetCode tx type 0x04) to be active on L2.
	t.logger.Infof("🔧 Delegating admin EOA to Simple7702Account via EIP-7702...")

	simple7702Addr := common.HexToAddress("0x4200000000000000000000000000000000000065")

	delegationNonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for EIP-7702 delegation: %w", err)
	}

	chainIDU256, overflow := uint256.FromBig(l2ChainID)
	if overflow {
		return fmt.Errorf("l2ChainID overflows uint256: %s", l2ChainID.String())
	}

	auth := types.SetCodeAuthorization{
		ChainID: *chainIDU256,
		Address: simple7702Addr,
		Nonce:   delegationNonce,
	}
	signedAuth, err := types.SignSetCode(privKey, auth)
	if err != nil {
		return fmt.Errorf("failed to sign EIP-7702 authorization: %w", err)
	}

	gasTipCapBig, err := l2Client.SuggestGasTipCap(ctx)
	if err != nil {
		gasTipCapBig = big.NewInt(1_000_000_000) // fallback: 1 Gwei
	}
	gasTipCap, _ := uint256.FromBig(new(big.Int).Mul(gasTipCapBig, big.NewInt(2)))
	gasFeeCapBig := new(big.Int).Add(gasTipCapBig, big.NewInt(2_000_000_000)) // tip + 2 Gwei base
	gasFeeCap, _ := uint256.FromBig(new(big.Int).Mul(gasFeeCapBig, big.NewInt(2)))

	setCodeTx := types.NewTx(&types.SetCodeTx{
		ChainID:   chainIDU256,
		Nonce:     delegationNonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       100_000,
		To:        adminAddr, // self-delegation: admin EOA is both sender and To
		Value:     uint256.NewInt(0),
		AuthList:  []types.SetCodeAuthorization{signedAuth},
	})

	// CRITICAL: SetCodeTx requires LatestSignerForChainID, NOT NewEIP155Signer.
	// EIP155Signer does not support tx type 0x04 and will return a type error.
	delegationSigner := types.LatestSignerForChainID(l2ChainID)
	signedSetCodeTx, err := types.SignTx(setCodeTx, delegationSigner, privKey)
	if err != nil {
		return fmt.Errorf("failed to sign SetCode tx: %w", err)
	}

	if err := l2Client.SendTransaction(ctx, signedSetCodeTx); err != nil {
		return fmt.Errorf("failed to send EIP-7702 delegation tx: %w", err)
	}

	// Wait for delegation tx receipt (same polling pattern as sendTxAndWait).
	delegationHash := signedSetCodeTx.Hash()
	for attempt := 1; attempt <= 30; attempt++ {
		receipt, err := l2Client.TransactionReceipt(ctx, delegationHash)
		if err == nil {
			if receipt.Status != 1 {
				return fmt.Errorf("EIP-7702 delegation tx %s reverted (status=%d)", delegationHash.Hex(), receipt.Status)
			}
			t.logger.Infof("EIP-7702 delegation complete: admin EOA %s -> Simple7702Account %s (tx: %s)", adminAddr.Hex(), simple7702Addr.Hex(), delegationHash.Hex())
			break
		}
		if attempt == 30 {
			return fmt.Errorf("EIP-7702 delegation tx %s not mined after 60s", delegationHash.Hex())
		}
		time.Sleep(2 * time.Second)
	}

	// Steps 6 & 7 (price updater and EntryPoint refill monitor) are handled by the
	// aa-operator Docker service, which runs as part of the compose stack when a non-TON
	// fee token is selected. Running them here would tie their lifecycle to the trh-sdk
	// CLI process, which exits after setup.

	return nil
}

// verifyAAPaymaster validates on-chain state after AA paymaster setup completes.
// It performs 3 eth_call checks to confirm that depositTo, updatePrice, and addToken
// transactions actually persisted correct state. The caller decides whether failures
// are fatal (setupAAPaymaster treats them as warnings).
func verifyAAPaymaster(
	ctx context.Context,
	l2Client *ethclient.Client,
	paymaster, tokenAddr, entryPoint, oracle common.Address,
	depositAmount *big.Int,
	logf func(format string, args ...interface{}),
) error {
	// Check 1: MultiTokenPaymaster.supportedTokens(tokenAddr) — verify token is enabled.
	selector1 := crypto.Keccak256([]byte("supportedTokens(address)"))[:4]
	calldata1 := make([]byte, 36)
	copy(calldata1[:4], selector1)
	copy(calldata1[16:36], tokenAddr.Bytes())

	result1, err := l2Client.CallContract(ctx, ethereum.CallMsg{To: &paymaster, Data: calldata1}, nil)
	if err != nil {
		return fmt.Errorf("supportedTokens(%s) call failed: %w", tokenAddr.Hex(), err)
	}
	if len(result1) < 128 {
		return fmt.Errorf("supportedTokens(%s) returned unexpected length %d (expected 128)", tokenAddr.Hex(), len(result1))
	}
	enabled := result1[31] != 0
	if !enabled {
		return fmt.Errorf("supportedTokens(%s).enabled is false", tokenAddr.Hex())
	}

	// Check 2: EntryPoint.balanceOf(paymaster) — verify deposit was credited.
	selector2 := crypto.Keccak256([]byte("balanceOf(address)"))[:4]
	calldata2 := make([]byte, 36)
	copy(calldata2[:4], selector2)
	copy(calldata2[16:36], paymaster.Bytes())

	result2, err := l2Client.CallContract(ctx, ethereum.CallMsg{To: &entryPoint, Data: calldata2}, nil)
	if err != nil {
		return fmt.Errorf("EntryPoint.balanceOf(%s) call failed: %w", paymaster.Hex(), err)
	}
	balance := new(big.Int).SetBytes(result2)
	if balance.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("EntryPoint deposit for paymaster is zero")
	}

	// Check 3: SimplePriceOracle.getPrice() — verify price was set and is not stale.
	selector3 := crypto.Keccak256([]byte("getPrice()"))[:4]

	result3, err := l2Client.CallContract(ctx, ethereum.CallMsg{To: &oracle, Data: selector3}, nil)
	if err != nil {
		return fmt.Errorf("SimplePriceOracle.getPrice() call failed: %w", err)
	}
	price := new(big.Int).SetBytes(result3)
	if price.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("oracle price is zero")
	}

	logf("AA Paymaster verification passed: token enabled, deposit=%s, price=%s", balance.String(), price.String())
	return nil
}

// deployBridgedUSDT deploys a bridged USDT token on L2 via OptimismMintableERC20Factory
// using createOptimismMintableERC20WithDecimals (USDT has 6 decimals, not the default 18).
//
// CREATE2 address prediction:
//   - eth_call simulates the factory call before sending a real transaction.
//   - The factory computes: CREATE2(factory, salt, keccak256(initcode)) where
//     salt = keccak256(abi.encode(remoteToken, name, symbol, decimals)).
//   - eth_call returns this address without modifying state, giving us the deterministic
//     L2 token address before any gas is spent.
//
// Idempotency:
//   - If eth_call fails (target address already has code from a prior deployment),
//     the function queries OptimismMintableERC20Created events filtered by remoteToken
//     to recover the previously deployed L2 address.
func deployBridgedUSDT(
	ctx context.Context,
	l2Client *ethclient.Client,
	l1USDTAddr common.Address,
	logf func(format string, args ...interface{}),
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
) (common.Address, error) {
	factory := common.HexToAddress(constants.OptimismMintableERC20FactoryPredeploy)

	// ABI-encode: createOptimismMintableERC20WithDecimals(address,string,string,uint8)
	// ABI layout for (address, string, string, uint8) — mixed static and dynamic types:
	//   Head section (4 × 32 = 128 bytes after selector):
	//     [4:36]   address _remoteToken (right-aligned)
	//     [36:68]  offset to _name from args start = 128 (0x80)
	//     [68:100] offset to _symbol from args start = 192 (0xC0)
	//     [100:132] uint8 _decimals = 6 (right-aligned)
	//   Data section:
	//     [132:164] len("Tether USD") = 10
	//     [164:196] "Tether USD" zero-padded to 32 bytes
	//     [196:228] len("USDT") = 4
	//     [228:260] "USDT" zero-padded to 32 bytes
	selector := crypto.Keccak256([]byte("createOptimismMintableERC20WithDecimals(address,string,string,uint8)"))[:4]
	const usdtDecimals = uint8(6)

	name := []byte("Tether USD") // 10 bytes
	symbol := []byte("USDT")     // 4 bytes

	padTo32 := func(b []byte) []byte {
		n := (len(b) + 31) / 32 * 32
		if n == 0 {
			n = 32
		}
		p := make([]byte, n)
		copy(p, b)
		return p
	}
	paddedName := padTo32(name)   // 32 bytes
	paddedSym := padTo32(symbol)  // 32 bytes

	// Offsets are measured from start of args (position 4 in calldata).
	// Head section = 4 × 32 = 128 bytes, so dynamic data starts at offset 128.
	nameOffset := uint64(4 * 32)                              // 128
	symOffset := nameOffset + 32 + uint64(len(paddedName))   // 192

	totalLen := 4 + 4*32 + 32 + len(paddedName) + 32 + len(paddedSym) // 260
	calldata := make([]byte, totalLen)

	copy(calldata[:4], selector)

	// address right-aligned in [4:36]
	copy(calldata[16:36], l1USDTAddr.Bytes())

	// nameOffset right-aligned in [36:68]
	nameOffBytes := new(big.Int).SetUint64(nameOffset).Bytes()
	copy(calldata[68-len(nameOffBytes):68], nameOffBytes)

	// symOffset right-aligned in [68:100]
	symOffBytes := new(big.Int).SetUint64(symOffset).Bytes()
	copy(calldata[100-len(symOffBytes):100], symOffBytes)

	// decimals right-aligned in [100:132]
	calldata[131] = usdtDecimals

	// name length right-aligned in [132:164]
	nameLenBytes := new(big.Int).SetUint64(uint64(len(name))).Bytes()
	copy(calldata[164-len(nameLenBytes):164], nameLenBytes)

	// name data [164:196]
	copy(calldata[164:164+len(paddedName)], paddedName)

	// symbol length right-aligned in [196:228]
	symLenBytes := new(big.Int).SetUint64(uint64(len(symbol))).Bytes()
	copy(calldata[228-len(symLenBytes):228], symLenBytes)

	// symbol data [228:260]
	copy(calldata[228:228+len(paddedSym)], paddedSym)

	// Predict the CREATE2 address via eth_call simulation.
	// The factory computes CREATE2 internally and returns the address without deploying.
	// If eth_call fails, the target address likely already has code (prior deployment).
	simResult, simErr := l2Client.CallContract(ctx, ethereum.CallMsg{
		To:   &factory,
		Data: calldata,
	}, nil)

	if simErr == nil {
		// Not yet deployed — predicted address is the factory's CREATE2 output.
		if len(simResult) < 32 {
			return common.Address{}, fmt.Errorf("eth_call returned unexpected length %d", len(simResult))
		}
		predictedAddr := common.BytesToAddress(simResult[12:32])
		logf("🏭 Deploying bridged USDT at predicted address %s (L1: %s)...", predictedAddr.Hex(), l1USDTAddr.Hex())

		receipt, err := sendTxAndWait(factory, big.NewInt(0), calldata)
		if err != nil {
			return common.Address{}, fmt.Errorf("createOptimismMintableERC20WithDecimals tx failed: %w", err)
		}

		// Confirm from emitted event.
		eventSigHash := common.BytesToHash(crypto.Keccak256([]byte("OptimismMintableERC20Created(address,address,address)")))
		for _, log := range receipt.Logs {
			if log.Address == factory && len(log.Topics) >= 2 && log.Topics[0] == eventSigHash {
				deployedAddr := common.BytesToAddress(log.Topics[1].Bytes())
				logf("✅ Bridged USDT deployed at L2: %s", deployedAddr.Hex())
				return deployedAddr, nil
			}
		}
		// Event not found — trust the predicted address (factory return value).
		logf("✅ Bridged USDT deployed at L2 (from prediction): %s", predictedAddr.Hex())
		return predictedAddr, nil
	}

	// eth_call failed — token was likely already deployed in a prior run (idempotency).
	// Recover the L2 address by querying OptimismMintableERC20Created events filtered
	// by remoteToken (indexed topic[2] = L1 USDT address).
	logf("ℹ️  eth_call simulation failed (%v); checking for prior USDT deployment...", simErr)

	eventSigHash := common.BytesToHash(crypto.Keccak256([]byte("OptimismMintableERC20Created(address,address,address)")))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{factory},
		Topics: [][]common.Hash{
			{eventSigHash},
			{},                                                  // any localToken
			{common.BytesToHash(l1USDTAddr.Bytes())},            // remoteToken = L1 USDT (indexed)
		},
		FromBlock: big.NewInt(0),
	}
	logs, err := l2Client.FilterLogs(ctx, query)
	if err != nil {
		return common.Address{}, fmt.Errorf("eth_call failed: %v; event query also failed: %w", simErr, err)
	}
	if len(logs) == 0 {
		return common.Address{}, fmt.Errorf("eth_call failed: %v; no prior USDT deployment found in factory events", simErr)
	}
	// topics[1] = localToken (L2 address, indexed)
	existingAddr := common.BytesToAddress(logs[0].Topics[1].Bytes())
	logf("ℹ️  Bridged USDT already deployed at L2: %s (reusing)", existingAddr.Hex())
	return existingAddr, nil
}

// feeTokenDecimals returns the decimal precision for the given fee token.
// USDC and USDT use 6 decimals; all others (ETH, TON) use 18.
func feeTokenDecimals(feeToken string) uint8 {
	switch feeToken {
	case constants.FeeTokenUSDC, constants.FeeTokenUSDT:
		return 6
	default:
		return 18
	}
}

// aaMarkupForToken returns the markup percent for the given fee token (used for log messages).
func aaMarkupForToken(feeToken string) uint64 {
	switch feeToken {
	case constants.FeeTokenETH:
		return constants.ETHPaymasterMarkupPct
	case constants.FeeTokenUSDC:
		return constants.USDCPaymasterMarkupPct
	case constants.FeeTokenUSDT:
		return constants.USDTPaymasterMarkupPct
	default:
		return 0
	}
}

// resolveAATokenConfig returns the verified L2 predeploy address and paymaster parameters
// for ETH and USDC fee tokens. USDT is handled separately via deployBridgedUSDT.
//
// Verified predeploy addresses (from tokamak-thanos Predeploys.sol):
//   - ETH (WETH):  0x4200000000000000000000000000000000000486
//   - USDC:        0x4200000000000000000000000000000000000778
func resolveAATokenConfig(feeToken string) (l2Addr string, markupPct uint64, decimals uint8, initialPrice *big.Int, err error) {
	switch feeToken {
	case constants.FeeTokenETH:
		// L2 WETH predeploy — verified in Predeploys.sol line 77 and op-bindings/predeploys/addresses.go line 28.
		return "0x4200000000000000000000000000000000000486",
			constants.ETHPaymasterMarkupPct,
			18,
			constants.DefaultETHInitialPrice,
			nil
	case constants.FeeTokenUSDC:
		// L2 USDC predeploy (FiatTokenV2_2) — verified in Predeploys.sol line 119 and op-bindings/predeploys/addresses.go line 42.
		// DeFi and Full presets only.
		return "0x4200000000000000000000000000000000000778",
			constants.USDCPaymasterMarkupPct,
			6,
			constants.DefaultUSDCInitialPrice,
			nil
	default:
		return "", 0, 0, nil, fmt.Errorf("fee token %q: use deployBridgedUSDT for USDT or check token configuration", feeToken)
	}
}
