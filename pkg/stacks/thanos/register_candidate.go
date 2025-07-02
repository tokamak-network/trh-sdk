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
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

	chainConfig := constants.L1ChainConfigurations[chainID.Uint64()]

	file, err := os.Open(fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", t.deploymentPath, fmt.Sprintf("%d-deploy.json", chainID)))
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

	privateKeyString := t.deployConfig.AdminPrivateKey

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
		fmt.Println("✅ Rollup config is already registered.")
		return nil
	}

	// Verify and register config
	if isVerificationPossible && rollupType == 0 {
		txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
			auth,
			ethCommon.HexToAddress(systemConfigProxy),
			ethCommon.HexToAddress(proxyAdmin),
			registerCandidate.NameInfo,
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
		fmt.Println("❌ Verification is not possible. Verification contract not registered as registrant")
		return fmt.Errorf("verification is not possible. Verification contract not registered as registrant")
	}

	// Convert amount to Wei
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidate.Amount), big.NewFloat(1e18))
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

	fmt.Println("Initiating transaction to register DAO candidate...")

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		ethCommon.HexToAddress(systemConfigProxy),
		amountBigInt,
		registerCandidate.UseTon,
		registerCandidate.Memo,
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

func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, registerCandidate *RegisterCandidateInput) error {
	if t.deployConfig.Network != constants.Testnet {
		return fmt.Errorf("register candidates verification is supported only on Testnet")
	}

	var err error
	fmt.Println("Starting candidate registration process...")
	fmt.Println("💲 Admin account will be used to register the candidate. Please ensure it has sufficient TON token balance.")

	if err := registerCandidate.Validate(ctx); err != nil {
		return err
	}

	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return err
	}
	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		return err
	}
	err = t.checkAdminBalance(ctx, adminAddress, registerCandidate.Amount, l1Client)
	if err != nil {
		return err
	}
	err = t.setupSafeWallet(ctx, t.deploymentPath)
	if err != nil {
		return fmt.Errorf("❌ failed to set up Safe Wallet: %w", err)
	}
	err = t.verifyRegisterCandidates(ctx, registerCandidate)
	if err != nil {
		return fmt.Errorf("❌ candidate verification failed: %w", err)
	}
	fmt.Println("✅ Candidate registration completed successfully!")
	return nil
}

func (t *ThanosStack) setupSafeWallet(ctx context.Context, cwd string) error {
	// Set the safe wallet address
	deployJSONPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fmt.Sprintf("%d-deploy.json", t.deployConfig.L1ChainID))
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

	fmt.Println("Checking if safe wallet is set up properly...")

	// Connect to the L1
	l1Client, err := ethclient.Dial(t.deployConfig.L1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	// Create the signer from the private key
	adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to get admin address: %v", err)
	}

	// Retrieve the owners and threshold
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	contract, err := abis.NewSafeExtender(ethCommon.HexToAddress(safeWalletAddress), l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call getThreshold() function
	threshold, err := contract.GetThreshold(callOpts)
	if err != nil {
		return fmt.Errorf("failed to call getThreshold: %v", err)
	}

	owners, err := contract.GetOwners(callOpts)
	if err != nil {
		return fmt.Errorf("failed to call getOwners: %v", err)
	}

	ownersInfo, err := GetDesignatedOwnersByChainID(t.deployConfig.L1ChainID)
	if err != nil {
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
		fmt.Println("✅ All required owners are present in the Safe wallet.")
	} else {
		fmt.Println("❌ Required owners do not match the Safe wallet.")
	}

	// Check if the threshold is 3
	thresholdMatch := threshold.Cmp(big.NewInt(3)) == 0

	if thresholdMatch {
		fmt.Println("✅ All required threshold are present in the Safe wallet.")
	} else {
		fmt.Println("❌ Required threshold do not match the Safe wallet.")
	}

	// Skip execution if owners and threshold match
	if ownersMatch && thresholdMatch {
		fmt.Println("Owners and threshold are already correct. Skipping hardhat task.")
		return nil
	}

	// Run hardhat task
	sdkPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "sdk")
	cmdStr := fmt.Sprintf("cd %s && L1_URL=%s PRIVATE_KEY=%s SAFE_WALLET_ADDRESS=%s npx hardhat set-safe-wallet", sdkPath, t.deployConfig.L1RPCURL, t.deployConfig.AdminPrivateKey, safeWalletAddress)
	if err := utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", cmdStr); err != nil {
		fmt.Print("\rfailed to setup the Safe wallet!\n")
		return err
	}

	return nil
}

func GetDesignatedOwnersByChainID(chainID uint64) (DesignatedOwners, error) {
	switch chainID {
	case constants.EthereumSepoliaChainID: // Sepolia
		return DesignatedOwners{
			TokamakDAO: ethCommon.HexToAddress("0x0Fd5632f3b52458C31A2C3eE1F4b447001872Be9"),
			Foundation: ethCommon.HexToAddress("0x61dc95E5f27266b94805ED23D95B4C9553A3D049"),
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
func (t *ThanosStack) GetRegistrationAdditionalInfo(ctx context.Context, registerCandidate *RegisterCandidateInput) (map[string]interface{}, error) {
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

	// Read deployment file to get contract addresses
	deploymentFile := fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%d-deploy.json", t.deploymentPath, chainID)
	file, err := os.Open(deploymentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open deployment file: %w", err)
	}
	defer file.Close()

	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		return nil, fmt.Errorf("failed to decode deployment JSON: %w", err)
	}

	result := make(map[string]interface{})

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
					result["safe_wallet"] = map[string]interface{}{
						"address":   contracts.SystemOwnerSafe,
						"owners":    ownerStrings,
						"threshold": threshold.Uint64(),
					}
				}
			}
		}
	}

	// 2. Candidate registration information
	result["candidate_registration"] = map[string]interface{}{
		"staking_amount":        registerCandidate.Amount,
		"rollup_config_address": contracts.SystemConfigProxy,
		"candidate_name":        registerCandidate.NameInfo,
		"candidate_memo":        registerCandidate.Memo,
	}

	return result, nil
}
