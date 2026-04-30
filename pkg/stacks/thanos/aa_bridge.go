package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// bridgeAdminTONForAASetup bridges the L2 native token from L1 to the admin address on L2
// if the admin's L2 balance is below the EntryPoint deposit requirement.
//
// The native token is read from SystemConfig.nativeTokenAddress() on-chain, which may
// differ from TON for deployments created before the "L2 native = always TON" fix
// (e.g. feeToken=USDT deployments have USDT as the on-chain native token).
//
// Steps:
//  1. Check admin L2 balance — skip if already >= DefaultEntryPointDeposit
//  2. Read on-chain native token from SystemConfig
//  3. Check admin L1 native-token balance — error if < DefaultAABridgeAmount
//  4. NativeToken.approve(L1StandardBridgeProxy, DefaultAABridgeAmount) on L1
//  5. L1StandardBridge.bridgeNativeTokenTo(admin, DefaultAABridgeAmount, 200_000, "") on L1
//  6. Poll L2 admin balance until >= DefaultEntryPointDeposit (5-min timeout, 3s interval)
func (t *ThanosStack) bridgeAdminTONForAASetup(ctx context.Context) error {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(t.deployConfig.AdminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Step 1: Check admin L2 balance. Skip bridge if already sufficient.
	l2Client, err := ethclient.DialContext(ctx, localL2RPCURL())
	if err != nil {
		return fmt.Errorf("failed to connect to L2 RPC: %w", err)
	}
	defer l2Client.Close()

	l2Balance, err := l2Client.BalanceAt(ctx, adminAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to query admin L2 balance: %w", err)
	}
	if l2Balance.Cmp(constants.DefaultEntryPointDeposit) >= 0 {
		t.logger.Infof("✅ Admin L2 balance sufficient (%s wei); skipping TON bridge", l2Balance.String())
		return nil
	}
	t.logger.Infof("Admin L2 balance %s wei < required %s wei; bridging TON from L1...",
		l2Balance.String(), constants.DefaultEntryPointDeposit.String())

	// Step 2: Connect to L1 and read deployment contracts.
	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1Client.Close()

	l1ChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get L1 chain ID: %w", err)
	}

	// Read L1StandardBridgeProxy and SystemConfigProxy from deployment JSON.
	contracts, err := t.readDeploymentContracts()
	if err != nil {
		return fmt.Errorf("failed to read deployment contracts: %w", err)
	}
	if contracts.L1StandardBridgeProxy == "" {
		return fmt.Errorf("L1StandardBridgeProxy address not found in deployment contracts")
	}
	if contracts.SystemConfigProxy == "" {
		return fmt.Errorf("SystemConfigProxy address not found in deployment contracts")
	}
	bridgeProxy := common.HexToAddress(contracts.L1StandardBridgeProxy)

	// Read the actual native token from SystemConfig on-chain.
	// This is critical: for deployments created before the "L2 native = always TON" fix,
	// SystemConfig.nativeTokenAddress may return a non-TON token (e.g. USDT).
	// The L1StandardBridge.bridgeNativeTokenTo() does transferFrom on this token.
	sysConfigAddr := common.HexToAddress(contracts.SystemConfigProxy)
	// nativeTokenAddress() selector = keccak256("nativeTokenAddress()")[:4]
	nativeTokenSelector := crypto.Keccak256([]byte("nativeTokenAddress()"))[:4]
	nativeTokenResult, err := l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &sysConfigAddr,
		Data: nativeTokenSelector,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to read nativeTokenAddress from SystemConfig(%s): %w",
			sysConfigAddr.Hex(), err)
	}
	if len(nativeTokenResult) < 32 {
		return fmt.Errorf("unexpected response length from SystemConfig.nativeTokenAddress(): %d bytes", len(nativeTokenResult))
	}
	nativeTokenAddr := common.BytesToAddress(nativeTokenResult[12:32])
	if nativeTokenAddr == (common.Address{}) {
		return fmt.Errorf("SystemConfig.nativeTokenAddress() returned zero address; fund admin L2 address %s manually",
			adminAddr.Hex())
	}

	// Log whether the on-chain native token matches the expected TON address.
	chainConfig := constants.L1ChainConfigurations[l1ChainID.Uint64()]
	expectedTON := common.HexToAddress(chainConfig.TON)
	if nativeTokenAddr != expectedTON {
		t.logger.Warnf("On-chain native token (%s) differs from expected TON (%s); bridging the on-chain native token",
			nativeTokenAddr.Hex(), expectedTON.Hex())
	}

	// Step 3: Check admin L1 native-token balance.
	// We use the TON ABI binding since it's a standard ERC20 interface (approve, balanceOf).
	nativeContract, err := abis.NewTON(nativeTokenAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to instantiate L1 native token contract (%s): %w", nativeTokenAddr.Hex(), err)
	}

	l1Balance, err := nativeContract.BalanceOf(&bind.CallOpts{Context: ctx}, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to query admin L1 native token balance: %w", err)
	}
	if l1Balance.Cmp(constants.DefaultAABridgeAmount) < 0 {
		return fmt.Errorf("insufficient L1 native token (%s) balance: have %s wei, need %s wei — top up admin %s on L1 first",
			nativeTokenAddr.Hex(), l1Balance.String(), constants.DefaultAABridgeAmount.String(), adminAddr.Hex())
	}

	// Step 4: NativeToken.approve(L1StandardBridgeProxy, DefaultAABridgeAmount) on L1.
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, l1ChainID)
	if err != nil {
		return fmt.Errorf("failed to create L1 transaction auth: %w", err)
	}
	t.logger.Infof("Approving L1 native token (%s) bridge transfer (%s wei) to %s...",
		nativeTokenAddr.Hex(), constants.DefaultAABridgeAmount.String(), bridgeProxy.Hex())
	var approveTx *types.Transaction
	if err := rpcCallWithRetry(ctx, func() error {
		var txErr error
		approveTx, txErr = nativeContract.Approve(auth, bridgeProxy, constants.DefaultAABridgeAmount)
		return txErr
	}); err != nil {
		return fmt.Errorf("native token approve failed: %w", err)
	}
	if _, err := bind.WaitMined(ctx, l1Client, approveTx); err != nil {
		return fmt.Errorf("waiting for native token approve receipt failed: %w", err)
	}
	t.logger.Infof("Approve confirmed (tx: %s)", approveTx.Hash().Hex())

	// Step 5: L1StandardBridge.bridgeNativeTokenTo(admin, amount, minGasLimit, "").
	// ABI: bridgeNativeTokenTo(address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData)
	// ABI encoding (mixed static + dynamic):
	//   Head:  [4 selector][32 _to][32 _amount][32 _minGasLimit][32 offset=128]
	//   Data:  [32 _extraData.length=0]
	selector := crypto.Keccak256([]byte("bridgeNativeTokenTo(address,uint256,uint32,bytes)"))[:4]
	calldata := make([]byte, 164) // 4 + 5×32
	copy(calldata[:4], selector)

	// _to: address right-aligned in [4:36]
	copy(calldata[16:36], adminAddr.Bytes())

	// _amount: uint256 right-aligned in [36:68]
	amountBytes := constants.DefaultAABridgeAmount.Bytes()
	copy(calldata[68-len(amountBytes):68], amountBytes)

	// _minGasLimit: uint32 right-aligned in [68:100]
	minGasLimitBytes := big.NewInt(200_000).Bytes()
	copy(calldata[100-len(minGasLimitBytes):100], minGasLimitBytes)

	// _extraData offset right-aligned in [100:132]: points to byte 128 from args start
	offsetBytes := big.NewInt(128).Bytes()
	copy(calldata[132-len(offsetBytes):132], offsetBytes)

	// _extraData length right-aligned in [132:164]: 0 (empty bytes) — already zero-initialized

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get L1 nonce: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get L1 gas price: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2)) // 2× for reliable inclusion

	// Estimate gas dynamically. The bridge call chains through
	// L1StandardBridge -> L1CrossDomainMessenger -> OptimismPortal,
	// which requires ~770k+ gas (far more than a simple ERC20 transfer).
	estimatedGas, err := l1Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  adminAddr,
		To:    &bridgeProxy,
		Value: big.NewInt(0),
		Data:  calldata,
	})
	if err != nil {
		return fmt.Errorf("failed to estimate gas for bridgeNativeTokenTo: %w", err)
	}
	gasLimit := estimatedGas * 120 / 100 // 20% safety margin
	t.logger.Infof("Bridge tx estimated gas: %d, using limit: %d", estimatedGas, gasLimit)

	bridgeTx := types.NewTransaction(nonce, bridgeProxy, big.NewInt(0), gasLimit, gasPrice, calldata)
	signedBridgeTx, err := types.SignTx(bridgeTx, types.NewEIP155Signer(l1ChainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign bridge tx: %w", err)
	}
	if err := sendTxWithRetry(ctx, l1Client, signedBridgeTx); err != nil {
		return fmt.Errorf("bridgeNativeTokenTo tx failed: %w", err)
	}
	bridgeTxHash := signedBridgeTx.Hash()
	t.logger.Infof("🌉 Bridge tx submitted (tx: %s); waiting for L1 confirmation...", bridgeTxHash.Hex())

	// Wait for bridge tx L1 receipt.
	for attempt := 1; attempt <= 30; attempt++ {
		receipt, err := l1Client.TransactionReceipt(ctx, bridgeTxHash)
		if err == nil {
			if receipt.Status != 1 {
				return fmt.Errorf("bridgeNativeTokenTo tx %s reverted", bridgeTxHash.Hex())
			}
			t.logger.Infof("✅ Bridge tx confirmed on L1 (block %d)", receipt.BlockNumber.Uint64())
			break
		}
		if attempt == 30 {
			return fmt.Errorf("bridge tx %s not mined on L1 after 60s", bridgeTxHash.Hex())
		}
		time.Sleep(2 * time.Second)
	}

	// Step 6: Poll L2 for admin balance >= DefaultEntryPointDeposit (5-min timeout, 3s interval).
	t.logger.Infof("⏳ Waiting for TON deposit to arrive on L2 (admin: %s)...", adminAddr.Hex())
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		bal, err := l2Client.BalanceAt(ctx, adminAddr, nil)
		if err == nil && bal.Cmp(constants.DefaultEntryPointDeposit) >= 0 {
			t.logger.Infof("✅ Admin L2 balance funded: %s wei", bal.String())
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timeout: admin %s L2 balance still below %s wei after 5 minutes — L1→L2 deposit may still be in flight",
		adminAddr.Hex(), constants.DefaultEntryPointDeposit.String())
}
