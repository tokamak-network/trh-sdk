package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type DesignatedOwners struct {
	TokamakDAO ethCommon.Address
	Foundation ethCommon.Address
}

type RegisterCandidateInput struct {
	Amount   float64
	UseTon   bool
	Memo     string
	NameInfo string
}

func (r *RegisterCandidateInput) Validate(ctx context.Context) error {
	if r.Amount < 1000.1 {
		return fmt.Errorf("amount must be at least 1000.1")
	}
	if r.Memo == "" {
		return fmt.Errorf("memo cannot be empty")
	}

	useWTON := !r.UseTon
	if useWTON {
		return fmt.Errorf("currently only TON is accepted")
	}

	return nil
}

func (t *ThanosStack) SetRegisterCandidate(value bool) *ThanosStack {
	t.registerCandidate = value
	return t
}

func (t *ThanosStack) checkAdminBalance(ctx context.Context, adminAddress ethCommon.Address, amount float64, l1Client *ethclient.Client) error {
	t.logger.Info("Checking admin's TON token balance...")

	chainIDFromClient, errChainID := l1Client.ChainID(ctx)
	if errChainID != nil {
		t.logger.Error("failed to get L1 chain ID", "err", errChainID)
		return fmt.Errorf("failed to get L1 chain ID: %w", errChainID)
	}

	chainConfig := constants.L1ChainConfigurations[chainIDFromClient.Uint64()]

	tonAddress := ethCommon.HexToAddress(chainConfig.TON)
	tokenInstance, errToken := abis.NewTON(tonAddress, l1Client)
	if errToken != nil {
		t.logger.Error("failed to instantiate TON token contract at %s", "err", errToken, "tonAddress", tonAddress.Hex())
		return fmt.Errorf("failed to instantiate TON token contract at %s: %w", tonAddress.Hex(), errToken)
	}

	adminBalance, errBalance := tokenInstance.BalanceOf(&bind.CallOpts{Context: ctx}, adminAddress)
	if errBalance != nil {
		t.logger.Error("failed to get TON balance for admin %s from contract %s", "err", errBalance, "adminAddress", adminAddress, "tonAddress", tonAddress.Hex())
		return fmt.Errorf("failed to get TON balance for admin %s from contract %s: %w", adminAddress, tonAddress.Hex(), errBalance)
	}
	amountInWei := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1e18))
	requiredAmountBigInt, _ := amountInWei.Int(nil)

	if adminBalance.Cmp(requiredAmountBigInt) < 0 {
		t.logger.Error("insufficient TON token balance for admin %s. Have: %s, Required: %s. Please top up the admin account", "adminAddress", adminAddress, "adminBalance", adminBalance.String(), "requiredAmountBigInt", requiredAmountBigInt.String())
		return fmt.Errorf("insufficient TON token balance for admin %s. Have: %s, Required: %s. Please top up the admin account",
			adminAddress,
			adminBalance.String(),
			requiredAmountBigInt.String())
	}
	return nil
}

// waitForAllowanceWithTimeout waits for allowance to be set with a context timeout
func waitForAllowanceWithTimeout(ctx context.Context, tonContract *abis.TON, adminAddress, l2ManagerAddress ethCommon.Address, requiredAmount *big.Int, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	currentDelay := 2 * time.Second
	const maxDelay = 60 * time.Second

	fmt.Printf("Waiting for allowance to propagate with incremental backoff (timeout: %v)...\n", timeout)

	for {
		allowance, err := tonContract.Allowance(&bind.CallOpts{Context: timeoutCtx}, adminAddress, l2ManagerAddress)
		if err != nil {
			if timeoutCtx.Err() == nil {
				fmt.Printf("Error checking allowance: %v\n", err)
			}
		} else {
			if allowance.Cmp(requiredAmount) >= 0 {
				fmt.Printf("✅ Allowance verified: %s TON\n", new(big.Float).Quo(new(big.Float).SetInt(allowance), big.NewFloat(1e18)).String())
				return nil
			}
			fmt.Printf("Allowance: %s/%s (retrying in %v...)\n", allowance.String(), requiredAmount.String(), currentDelay)
		}

		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for allowance propagation after %v", timeout)
		case <-time.After(currentDelay):
			currentDelay *= 2
			if currentDelay > maxDelay {
				currentDelay = maxDelay
			}
		}
	}
}

// fromDeployContract flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) verifyRegisterCandidates(ctx context.Context, registerCandidate *RegisterCandidateInput) error {
	if err := registerCandidate.Validate(ctx); err != nil {
		return err
	}

	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return err
	}
	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		t.logger.Error("Failed to get chain id", "err", err)
		return err
	}

	chainConfig := constants.L1ChainConfigurations[chainID.Uint64()]

	var contracts *types.Contracts

	contracts, err = utils.ReadDeployementConfigFromJSONFile(t.deploymentPath, chainID.Uint64())
	if err != nil {
		return fmt.Errorf("failed to read deployment config: %w", err)
	}

	privateKeyString := t.deployConfig.AdminPrivateKey

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyString, "0x"))
	if err != nil {
		t.logger.Error("invalid private key", "err", err)
		return fmt.Errorf("invalid private key: %v", err)
	}

	// Create transaction auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		t.logger.Error("failed to create transaction auth", "err", err)
		return fmt.Errorf("failed to create transaction auth: %v", err)
	}

	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	// Get contract address from environment
	contractAddrStr := chainConfig.L1VerificationContractAddress
	if contractAddrStr == "" {
		t.logger.Error("L1_VERIFICATION_CONTRACT_ADDRESS not set in constant")
		return fmt.Errorf("L1_VERIFICATION_CONTRACT_ADDRESS not set in constant")
	}
	contractAddr := ethCommon.HexToAddress(contractAddrStr)

	// Create contract instance
	contract, err := abis.NewL1ContractVerification(contractAddr, l1Client)
	if err != nil {
		t.logger.Error("failed to create contract instance", "err", err)
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	systemConfigProxy := contracts.SystemConfigProxy
	if systemConfigProxy == "" {
		t.logger.Error("SystemConfigProxy is not set")
		return fmt.Errorf("SystemConfigProxy is not set")
	}

	proxyAdmin := contracts.ProxyAdmin
	if proxyAdmin == "" {
		t.logger.Error("ProxyAdmin is not set")
		return fmt.Errorf("ProxyAdmin is not set")
	}

	safeWalletAddress := contracts.SystemOwnerSafe
	if safeWalletAddress == "" {
		t.logger.Error("SafeWallet addresss is not set")
		return fmt.Errorf("SafeWallet addresss is not set")
	}

	isVerificationPossible, err := contract.IsVerificationPossible(callOpts)
	if err != nil {
		t.logger.Error("failed to check if verification is possible", "err", err)
		return fmt.Errorf("failed to check if verification is possible: %v", err)
	}

	contractAddrStrBridgeRegistry := chainConfig.L1BridgeRegistry
	if contractAddrStrBridgeRegistry == "" {
		t.logger.Error("L1BridgeRegistry variable not set in constant")
		return fmt.Errorf("L1BridgeRegistry variable not set in constant")
	}
	contractAddressBridgeRegistry := ethCommon.HexToAddress(contractAddrStrBridgeRegistry)
	// Create contract instance
	bridgeRegistryContract, err := abis.NewL1BridgeRegistry(contractAddressBridgeRegistry, l1Client)
	if err != nil {
		t.logger.Error("failed to create contract instance", "err", err)
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	rollupType, err := bridgeRegistryContract.RollupType(callOpts, ethCommon.HexToAddress(systemConfigProxy))
	if err != nil {
		t.logger.Error("failed to get rollup type", "err", err)
		return fmt.Errorf("failed to get rollup type: %v", err)
	}

	if rollupType != 0 {
		t.logger.Info("✅ Rollup config is already registered.")
		return nil
	}

	// Verify and register config
	if isVerificationPossible && rollupType == 0 {
		txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
			auth,
			ethCommon.HexToAddress(systemConfigProxy),
			ethCommon.HexToAddress(proxyAdmin),
			ethCommon.HexToAddress(safeWalletAddress),
			registerCandidate.NameInfo,
		)
		if err != nil {
			t.logger.Error("failed to register candidate", "err", err)
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		t.logger.Info("Verification and register config transaction submitted ", "txHash ", txVerifyAndRegisterConfig.Hash().Hex())

		// Wait for transaction confirmation
		receiptVerifyRegisterConfig, err := bind.WaitMined(ctx, l1Client, txVerifyAndRegisterConfig)
		if err != nil {
			t.logger.Error("failed waiting for transaction confirmation", "err", err)
			return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
		}

		if receiptVerifyRegisterConfig.Status != 1 {
			t.logger.Error("transaction failed with status", "status", receiptVerifyRegisterConfig.Status)
			return fmt.Errorf("transaction failed with status: %d", receiptVerifyRegisterConfig.Status)
		}

		t.logger.Info("Transaction confirmed in block ", "blockNumber ", receiptVerifyRegisterConfig.BlockNumber.Uint64())
	} else if rollupType == 0 {
		t.logger.Error("❌ Verification is not possible. Verification contract not registered as registrant")
		return fmt.Errorf("verification is not possible. Verification contract not registered as registrant")
	}

	// Convert amount to Wei
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidate.Amount), big.NewFloat(1e18))
	amountBigInt, _ := amountInWei.Int(nil)

	// Get contract address from environment
	l2ManagerAddressStr := chainConfig.L2ManagerAddress
	if l2ManagerAddressStr == "" {
		t.logger.Error("L2_MANAGER_ADDRESS variable is not set")
		return fmt.Errorf("L2_MANAGER_ADDRESS variable is not set")
	}
	l2ManagerAddress := ethCommon.HexToAddress(l2ManagerAddressStr)

	tonAddressStr := chainConfig.TON
	if tonAddressStr == "" {
		t.logger.Error("TON variable is not set")
		return fmt.Errorf("TON variable is not set")
	}
	tonAddress := ethCommon.HexToAddress(tonAddressStr)

	// Create TON contract instance
	tonContract, err := abis.NewTON(tonAddress, l1Client)
	if err != nil {
		t.logger.Error("failed to create TON contract instance", "err", err)
		return fmt.Errorf("failed to create TON contract instance: %v", err)
	}

	// Approve transaction
	txApprove, err := tonContract.Approve(auth, l2ManagerAddress, amountBigInt)
	if err != nil {
		t.logger.Error("failed to approve TON", "err", err)
		return fmt.Errorf("failed to approve TON: %v", err)
	}

	t.logger.Info("Approve TON transaction submitted ", "txHash ", txApprove.Hash().Hex())

	// Wait for transaction confirmation
	receiptApprove, err := bind.WaitMined(ctx, l1Client, txApprove)
	if err != nil {
		t.logger.Error("failed waiting for transaction confirmation", "err", err)
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	t.logger.Info("Transaction confirmed in block ", "blockNumber ", receiptApprove.BlockNumber.Uint64())

	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		t.logger.Error("failed to get admin address", "err", err)
		return err
	}

	// Verify the allowance was set correctly
	waitForAllowanceWithTimeout(ctx, tonContract, adminAddress, l2ManagerAddress, amountBigInt, 60*time.Second)

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, l1Client)
	if err != nil {
		t.logger.Error("failed to create contract instance", "err", err)
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	t.logger.Info("Initiating transaction to register DAO candidate...")

	// Final balance check before transfer
	currentBalance, err := tonContract.BalanceOf(&bind.CallOpts{Context: ctx}, adminAddress)
	if err != nil {
		t.logger.Error("failed to get current balance", "err", err)
		return fmt.Errorf("failed to get current balance: %v", err)
	}

	if currentBalance.Cmp(amountBigInt) < 0 {
		t.logger.Error("insufficient balance at transfer time", "currentBalance", currentBalance.String(), "amountBigInt", amountBigInt.String())
		return fmt.Errorf("insufficient balance at transfer time: have %s, required %s",
			currentBalance.String(), amountBigInt.String())
	}

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		ethCommon.HexToAddress(systemConfigProxy),
		amountBigInt,
		registerCandidate.UseTon,
		registerCandidate.Memo,
	)
	if err != nil {
		t.logger.Error("failed to register candidate", "err", err)
		return fmt.Errorf("failed to register candidate: %v", err)
	}

	t.logger.Info("Register Candidate transaction submitted ", "txHash ", txRegisterCandidate.Hash().Hex())

	// Wait for transaction confirmation
	receiptRegisterCandidate, err := bind.WaitMined(ctx, l1Client, txRegisterCandidate)
	if err != nil {
		t.logger.Error("failed waiting for transaction confirmation", "err", err)
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	if receiptRegisterCandidate.Status != 1 {
		t.logger.Error("transaction failed with status", "status", receiptRegisterCandidate.Status)
		return fmt.Errorf("transaction failed with status: %d", receiptRegisterCandidate.Status)
	}

	t.logger.Info("Transaction confirmed in block ", "blockNumber ", receiptRegisterCandidate.BlockNumber.Uint64())

	// Get block details to extract timestamp
	blockNumber := receiptRegisterCandidate.BlockNumber
	block, err := l1Client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get block details: %v", err)
	}

	// Convert block timestamp to the required format
	blockTime := time.Unix(int64(block.Time()), 0)
	formattedTime := blockTime.Format("2006-01-02 15:04:05 MST")

	// Extract candidateAddOn address from event logs
	var candidateAddress string

	for _, vLog := range receiptRegisterCandidate.Logs {
		// Check if this log is from the L2Manager contract
		if vLog.Address == l2ManagerAddress {
			// Try to parse as RegisteredCandidateAddOn event
			event, err := l2ManagerContract.ParseRegisteredCandidateAddOn(*vLog)
			if err == nil {
				// Successfully parsed the event
				candidateAddress = event.CandidateAddOn.Hex()
				fmt.Printf("✅ Successfully extracted Candidate address: %s\n", candidateAddress)
				break
			}
		}
	}

	t.deployConfig.StakingInfo = &types.StakingInfo{
		IsCandidate:         true,
		StakingAmount:       registerCandidate.Amount,
		RollupConfigAddress: contracts.SystemConfigProxy,
		CandidateName:       registerCandidate.NameInfo,
		CandidateMemo:       registerCandidate.Memo,
		RegistrationTime:    formattedTime,
		RegistrationTxHash:  txRegisterCandidate.Hash().Hex(),
		CandidateAddress:    candidateAddress,
	}

	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		fmt.Println("Failed to write settings file:", err)
		return err
	}
	fmt.Println("✅ Settings file updated successfully with StakingInfo")

	return nil
}

func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, registerCandidate *RegisterCandidateInput) error {
	var err error
	t.logger.Info("Starting candidate registration process...")
	t.logger.Info("💲 Admin account will be used to register the candidate. Please ensure it has sufficient TON token balance.")

	if err := registerCandidate.Validate(ctx); err != nil {
		t.logger.Error("failed to validate register candidate", "err", err)
		return err
	}

	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		t.logger.Error("failed to get L1 client", "err", err)
		return err
	}
	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		t.logger.Error("failed to get admin address", "err", err)
		return err
	}
	err = t.checkAdminBalance(ctx, adminAddress, registerCandidate.Amount, l1Client)
	if err != nil {
		t.logger.Error("failed to check admin balance", "err", err)
		return err
	}
	err = t.setupSafeWallet(ctx, t.deploymentPath)
	if err != nil {
		t.logger.Error("failed to set up Safe Wallet", "err", err)
		return fmt.Errorf("❌ failed to set up Safe Wallet: %w", err)
	}
	err = t.verifyRegisterCandidates(ctx, registerCandidate)
	if err != nil {
		t.logger.Error("❌ candidate verification failed", "err", err)
		return fmt.Errorf("❌ candidate verification failed: %w", err)
	}
	t.logger.Info("✅ Candidate registration completed successfully!")
	return nil
}

func (t *ThanosStack) setupSafeWallet(ctx context.Context, cwd string) error {
	// Set the safe wallet address
	deployJSONPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fmt.Sprintf("%d-deploy.json", t.deployConfig.L1ChainID))
	deployData, err := os.ReadFile(deployJSONPath)
	if err != nil {
		t.logger.Error("failed to read deployment file", "err", err)
		return fmt.Errorf("failed to read deployment file: %v", err)
	}

	var deployMap map[string]interface{}
	if err := json.Unmarshal(deployData, &deployMap); err != nil {
		t.logger.Error("failed to parse deployment file", "err", err)
		return fmt.Errorf("failed to parse deployment file: %v", err)
	}

	safeWalletAddress, ok := deployMap["SystemOwnerSafe"].(string)
	if !ok {
		t.logger.Error("failed to get the value of 'SystemOwnerSafe' field in the deployment file")
		return fmt.Errorf("failed to get the value of 'SystemOwnerSafe' field in the deployment file")
	}

	t.logger.Info("Checking if safe wallet is set up properly...")

	// Connect to the L1
	l1Client, err := ethclient.Dial(t.deployConfig.L1RPCURL)
	if err != nil {
		t.logger.Error("failed to connect to Ethereum client", "err", err)
		return fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	// Create the signer from the private key
	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		t.logger.Error("failed to get admin address", "err", err)
		return fmt.Errorf("failed to get admin address: %v", err)
	}

	// Retrieve the owners and threshold
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	contract, err := abis.NewSafeExtender(ethCommon.HexToAddress(safeWalletAddress), l1Client)
	if err != nil {
		t.logger.Error("failed to create contract instance ", "err ", err)
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call getThreshold() function
	threshold, err := contract.GetThreshold(callOpts)
	if err != nil {
		t.logger.Error("failed to call getThreshold ", "err ", err)
		return fmt.Errorf("failed to call getThreshold: %v", err)
	}

	owners, err := contract.GetOwners(callOpts)
	if err != nil {
		t.logger.Error("failed to call getOwners", "err", err)
		return fmt.Errorf("failed to call getOwners: %v", err)
	}

	ownersInfo, err := GetDesignatedOwnersByChainID(t.deployConfig.L1ChainID)
	if err != nil {
		t.logger.Error("failed to get designated owners", "err", err)
		return fmt.Errorf("failed to get designated owners: %v", err)
	}

	requiredOwners := []ethCommon.Address{
		adminAddress, // admin address
		ownersInfo.TokamakDAO,
		ownersInfo.Foundation,
	}

	// Check if the owners match the required ones
	ownersMatch := true
	for _, requiredOwner := range requiredOwners {
		ownerFound := false
		for _, owner := range owners {
			if owner == requiredOwner {
				ownerFound = true
				break
			}
		}
		if !ownerFound {
			ownersMatch = false
			break
		}
	}
	if ownersMatch {
		t.logger.Info("✅ All required owners are present in the Safe wallet.")
	} else {
		fmt.Println("Required owners do not match the Safe wallet.")
	}

	// Check if the threshold is 3
	thresholdMatch := threshold.Cmp(big.NewInt(3)) == 0

	if thresholdMatch {
		t.logger.Info("✅ All required threshold are present in the Safe wallet.")
	} else {
		fmt.Println("Required threshold do not match the Safe wallet.")
	}

	// Skip execution if owners and threshold match
	if ownersMatch && thresholdMatch {
		t.logger.Info("Owners and threshold are already correct. Skipping hardhat task.")
		return nil
	}

	// Run hardhat task
	sdkPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "sdk")
	cmdStr := fmt.Sprintf("cd %s && L1_URL=%s PRIVATE_KEY=%s SAFE_WALLET_ADDRESS=%s npx hardhat set-safe-wallet", sdkPath, t.deployConfig.L1RPCURL, t.deployConfig.AdminPrivateKey, safeWalletAddress)
	if err := utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", cmdStr); err != nil {
		t.logger.Error("failed to setup the Safe wallet!")
		return err
	}

	return nil
}

func GetDesignatedOwnersByChainID(chainID uint64) (DesignatedOwners, error) {
	switch chainID {
	case constants.EthereumSepoliaChainID: // Sepolia
		return DesignatedOwners{
			TokamakDAO: ethCommon.HexToAddress("0xA2101482b28E3D99ff6ced517bA41EFf4971a386"),
			Foundation: ethCommon.HexToAddress("0xeEfb04f3A4406363b399A1c3d194d75ca81A2d1B"),
		}, nil
	case constants.EthereumMainnetChainID: // Ethereum (TODO: need to update)
		return DesignatedOwners{
			TokamakDAO: ethCommon.HexToAddress("0xYourMainnetTokamakDAOAddress"),
			Foundation: ethCommon.HexToAddress("0xYourMainnetFoundationAddress"),
		}, nil
	default:
		return DesignatedOwners{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

// GetRegistrationAdditionalInfo returns additional information after candidate registration
func (t *ThanosStack) GetRegistrationAdditionalInfo(ctx context.Context, registerCandidate *RegisterCandidateInput) (*types.RegistrationAdditionalInfo, error) {
	// Connect to L1 to get contract information
	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1Client.Close()

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	var contracts *types.Contracts

	contracts, err = utils.ReadDeployementConfigFromJSONFile(t.deploymentPath, chainID.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to read deployment config: %w", err)
	}

	result := &types.RegistrationAdditionalInfo{
		URL: constants.L1ChainConfigurations[chainID.Uint64()].StakingURL,
	}

	// 1. Safe wallet information
	if contracts.SystemOwnerSafe != "" {
		safeAddress := ethCommon.HexToAddress(contracts.SystemOwnerSafe)
		safeContract, err := abis.NewSafeExtender(safeAddress, l1Client)
		if err == nil {
			callOpts := &bind.CallOpts{Context: ctx}

			// Get owners
			owners, err := safeContract.GetOwners(callOpts)
			if err == nil {
				ownerStrings := make([]string, len(owners))
				for i, owner := range owners {
					ownerStrings[i] = owner.Hex()
				}

				// Get threshold
				threshold, err := safeContract.GetThreshold(callOpts)
				if err == nil {
					result.SafeWallet = &types.SafeWalletInfo{
						Address:   contracts.SystemOwnerSafe,
						Owners:    ownerStrings,
						Threshold: threshold.Uint64(),
					}
				}
			}
		}
	}

	// 2. Candidate registration information
	result.CandidateRegistration = &types.CandidateRegistrationInfo{
		StakingAmount:       registerCandidate.Amount,
		RollupConfigAddress: contracts.SystemConfigProxy,
		CandidateName:       registerCandidate.NameInfo,
		CandidateMemo:       registerCandidate.Memo,
		RegistrationTime:    time.Now().Format("2006-01-02 15:04:05 MST"),
	}

	return result, nil
}

// DisplayRegistrationAdditionalInfo retrieves and displays additional registration information
func (t *ThanosStack) DisplayRegistrationAdditionalInfo(ctx context.Context, registerCandidate *RegisterCandidateInput) {
	// Get and display additional registration information
	additionalInfo, err := t.GetRegistrationAdditionalInfo(ctx, registerCandidate)
	if err != nil {
		t.logger.Error("⚠️  Warning: Failed to retrieve additional information", "err", err)
		return
	}

	// Pretty print the additional information
	t.logger.Info("📋 Registration Summary:")
	prettyJSON, err := json.MarshalIndent(additionalInfo, "", "  ")
	if err != nil {
		t.logger.Error("Failed to format additional info", "err", err)
		t.logger.Error("Raw data", "additionalInfo", additionalInfo)
	} else {
		t.logger.Info("Registration Summary", "additionalInfo", string(prettyJSON))
	}
}
