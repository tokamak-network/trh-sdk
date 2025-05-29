package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) checkAdminBalance(ctx context.Context, adminAddress ethCommon.Address, amount float64, l1Client *ethclient.Client) error {
	fmt.Printf("Checking admin's TON token balance... \n")

	chainIDFromClient, errChainID := l1Client.ChainID(ctx)
	if errChainID != nil {
		return fmt.Errorf("failed to get L1 chain ID: %w", errChainID)
	}

	chainConfig := constants.L1ChainConfigurations[chainIDFromClient.Uint64()]

	tonAddress := ethCommon.HexToAddress(chainConfig.TON)
	tokenInstance, errToken := abis.NewTON(tonAddress, l1Client)
	if errToken != nil {
		return fmt.Errorf("failed to instantiate TON token contract at %s: %w", tonAddress.Hex(), errToken)
	}

	adminBalance, errBalance := tokenInstance.BalanceOf(&bind.CallOpts{Context: ctx}, adminAddress)
	if errBalance != nil {
		return fmt.Errorf("failed to get TON balance for admin %s from contract %s: %w", adminAddress, tonAddress.Hex(), errBalance)
	}
	amountInWei := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1e18))
	requiredAmountBigInt, _ := amountInWei.Int(nil)

	if adminBalance.Cmp(requiredAmountBigInt) < 0 {
		return fmt.Errorf("insufficient TON token balance for admin %s. Have: %s, Required: %s. Please top up the admin account",
			adminAddress,
			adminBalance.String(),
			requiredAmountBigInt.String())
	}
	return nil
}

// fromDeployContract flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) verifyRegisterCandidates(ctx context.Context, config *types.Config, registerCandidate *RegisterCandidateInput) error {
	l1Client, err := ethclient.DialContext(ctx, config.L1RPCURL)
	if err != nil {
		return err
	}
	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

	chainConfig := constants.L1ChainConfigurations[chainID.Uint64()]

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error determining current directory:", err)
		return err
	}

	file, err := os.Open(fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", cwd, fmt.Sprintf("%d-deploy.json", chainID)))
	if err != nil {
		fmt.Println("Error opening deployment file:", err)
		return err
	}

	// Decode JSON
	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		fmt.Println("Error decoding deployment JSON file:", err)
		return err
	}

	privateKeyString := config.AdminPrivateKey

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyString, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	// Create transaction auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transaction auth: %v", err)
	}

	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	// Get contract address from environment
	contractAddrStr := chainConfig.L1VerificationContractAddress
	if contractAddrStr == "" {
		return fmt.Errorf("L1_VERIFICATION_CONTRACT_ADDRESS not set in constant")
	}
	contractAddr := ethCommon.HexToAddress(contractAddrStr)

	// Create contract instance
	contract, err := abis.NewL1ContractVerification(contractAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	systemConfigProxy := contracts.SystemConfigProxy
	if systemConfigProxy == "" {
		return fmt.Errorf("SystemConfigProxy is not set")
	}

	proxyAdmin := contracts.ProxyAdmin
	if proxyAdmin == "" {
		return fmt.Errorf("ProxyAdmin is not set")
	}

	l2TonAddress := chainConfig.L2TonAddress
	if l2TonAddress == "" {
		return fmt.Errorf("L2TonAddress variable is not set")
	}

	safeWalletAddress := contracts.SystemOwnerSafe
	if safeWalletAddress == "" {
		return fmt.Errorf("SafeWallet addresss is not set")
	}

	isVerificationPossible, err := contract.IsVerificationPossible(callOpts)
	if err != nil {
		return fmt.Errorf("failed to check if verification is possible: %v", err)
	}

	contractAddrStrBridgeRegistry := chainConfig.L1BridgeRegistry
	if contractAddrStrBridgeRegistry == "" {
		return fmt.Errorf("L1BridgeRegistry variable not set in constant")
	}
	contractAddressBridgeRegistry := ethCommon.HexToAddress(contractAddrStrBridgeRegistry)
	// Create contract instance
	bridgeRegistryContract, err := abis.NewL1BridgeRegistry(contractAddressBridgeRegistry, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	rollupType, err := bridgeRegistryContract.RollupType(callOpts, ethCommon.HexToAddress(systemConfigProxy))
	if err != nil {
		return fmt.Errorf("failed to get rollup type: %v", err)
	}

	if rollupType != 0 {
		fmt.Printf("Config already registered \n")
	}

	// Verify and register config
	if isVerificationPossible && rollupType == 0 {
		txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
			auth,
			ethCommon.HexToAddress(systemConfigProxy),
			ethCommon.HexToAddress(proxyAdmin),
			2, //TODO: Need to check and update this using TON
			ethCommon.HexToAddress(l2TonAddress),
			registerCandidate.nameInfo,
			ethCommon.HexToAddress(safeWalletAddress),
		)
		if err != nil {
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		fmt.Printf("Verification and register config transaction submitted: %s\n", txVerifyAndRegisterConfig.Hash().Hex())

		// Wait for transaction confirmation
		receiptVerifyRegisterConfig, err := bind.WaitMined(ctx, l1Client, txVerifyAndRegisterConfig)
		if err != nil {
			return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
		}

		if receiptVerifyRegisterConfig.Status != 1 {
			return fmt.Errorf("transaction failed with status: %d", receiptVerifyRegisterConfig.Status)
		}

		fmt.Printf("Transaction confirmed in block %d\n", receiptVerifyRegisterConfig.BlockNumber.Uint64())
	} else if rollupType == 0 {
		txRegisterConfig, err := bridgeRegistryContract.RegisterRollupConfig(auth, ethCommon.HexToAddress(systemConfigProxy), 2, ethCommon.HexToAddress(l2TonAddress),
			registerCandidate.nameInfo)

		if err != nil {
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		fmt.Printf("Register config transaction submitted: %s\n", txRegisterConfig.Hash().Hex())

		// Wait for transaction confirmation
		receiptRegisterConfig, err := bind.WaitMined(ctx, l1Client, txRegisterConfig)
		if err != nil {
			return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
		}

		if receiptRegisterConfig.Status != 1 {
			return fmt.Errorf("transaction failed with status: %d", receiptRegisterConfig.Status)
		}

		fmt.Printf("Transaction confirmed in block %d\n", receiptRegisterConfig.BlockNumber.Uint64())
	}

	// Convert amount to Wei
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidate.amount), big.NewFloat(1e18))
	amountBigInt, _ := amountInWei.Int(nil)

	// Get contract address from environment
	l2ManagerAddressStr := chainConfig.L2ManagerAddress
	if l2ManagerAddressStr == "" {
		return fmt.Errorf("L2_MANAGER_ADDRESS variable is not set")
	}
	l2ManagerAddress := ethCommon.HexToAddress(l2ManagerAddressStr)

	tonAddressStr := chainConfig.TON
	if tonAddressStr == "" {
		return fmt.Errorf("TON variable is not set")
	}
	tonAddress := ethCommon.HexToAddress(tonAddressStr)

	// Create TON contract instance
	tonContract, err := abis.NewTON(tonAddress, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create TON contract instance: %v", err)
	}

	// Approve transaction
	txApprove, err := tonContract.Approve(auth, l2ManagerAddress, amountBigInt)
	if err != nil {
		return fmt.Errorf("failed to approve TON: %v", err)
	}

	fmt.Printf("Approve TON transaction submitted: %s\n", txApprove.Hash().Hex())

	// Wait for transaction confirmation
	receiptApprove, err := bind.WaitMined(ctx, l1Client, txApprove)
	if err != nil {
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	fmt.Printf("Transaction confirmed in block %d\n", receiptApprove.BlockNumber.Uint64())

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		ethCommon.HexToAddress(systemConfigProxy),
		amountBigInt,
		registerCandidate.useTon,
		registerCandidate.memo,
	)
	if err != nil {
		return fmt.Errorf("failed to register candidate: %v", err)
	}

	fmt.Printf("Register Candidate transaction submitted: %s\n", txRegisterCandidate.Hash().Hex())

	// Wait for transaction confirmation
	receiptRegisterCandidate, err := bind.WaitMined(ctx, l1Client, txRegisterCandidate)
	if err != nil {
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	if receiptRegisterCandidate.Status != 1 {
		return fmt.Errorf("transaction failed with status: %d", receiptRegisterCandidate.Status)
	}

	fmt.Printf("Transaction confirmed in block %d\n", receiptRegisterCandidate.BlockNumber.Uint64())

	return nil
}

func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, cwd string) error {
	var err error
	registerCandidate, err := t.inputRegisterCandidate()
	if err != nil {
		return fmt.Errorf("❌ failed to get register candidate input: %w", err)
	}
	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return err
	}
	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		return err
	}
	err = t.checkAdminBalance(ctx, adminAddress, registerCandidate.amount, l1Client)
	if err != nil {
		return err
	}
	err = t.setupSafeWallet(t.deployConfig, cwd)
	if err != nil {
		return fmt.Errorf("❌ failed to set up Safe Wallet: %w", err)
	}
	err = t.verifyRegisterCandidates(ctx, t.deployConfig, registerCandidate)
	if err != nil {
		return fmt.Errorf("❌ candidate verification failed: %w", err)
	}
	fmt.Println("✅ Candidate registration completed successfully!")
	return nil
}

func (t *ThanosStack) setupSafeWallet(config *types.Config, cwd string) error {
	// 1. Set the safe wallet address
	deployJSONPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fmt.Sprintf("%d-deploy.json", config.L1ChainID))
	deployData, err := os.ReadFile(deployJSONPath)
	if err != nil {
		return fmt.Errorf("failed to read deployment file: %v", err)
	}

	var deployMap map[string]interface{}
	if err := json.Unmarshal(deployData, &deployMap); err != nil {
		return fmt.Errorf("failed to parse deployment file: %v", err)
	}

	safeWalletAddress, ok := deployMap["SystemOwnerSafe"].(string)
	if !ok {
		return fmt.Errorf("failed to get the value of 'SystemOwnerSafe' field in the deployment file")
	}
	fmt.Println("SafeWalletAddess: ", safeWalletAddress)
	// 2. Run hardhat task
	sdkPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "sdk")
	cmdStr := fmt.Sprintf("cd %s && L1_URL=%s PRIVATE_KEY=%s SAFE_WALLET_ADDRESS=%s npx hardhat set-safe-wallet", sdkPath, config.L1RPCURL, config.AdminPrivateKey, safeWalletAddress)
	if err := utils.ExecuteCommandStream("bash", "-c", cmdStr); err != nil {
		fmt.Print("\rfailed to setup the Safe wallet!\n")
		return err
	}

	return nil
}
