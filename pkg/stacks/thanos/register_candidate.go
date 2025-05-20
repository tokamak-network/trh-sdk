package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// fromDeployContract flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) verifyRegisterCandidates(ctx context.Context, registerCandidate *RegisterCandidateInput) error {
	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		return err
	}
	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

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
	contractAddrStr := constants.L1ChainConfigurations[chainID.Uint64()].L1VerificationContractAddress
	if contractAddrStr == "" {
		return fmt.Errorf("L1_VERIFICATION_CONTRACT_ADDRESS not set in constant")
	}
	contractAddr := common.HexToAddress(contractAddrStr)

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

	l2TonAddress := constants.L1ChainConfigurations[chainID.Uint64()].L2TonAddress
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
	// Verify and register config
	if isVerificationPossible {
		txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
			auth,
			common.HexToAddress(systemConfigProxy),
			common.HexToAddress(proxyAdmin),
			2, //TODO: Need to check and update this using TON
			common.HexToAddress(l2TonAddress),
			registerCandidate.nameInfo,
			common.HexToAddress(safeWalletAddress),
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
	} else {
		contractAddrStrBridgeRegistry := constants.L1ChainConfigurations[chainID.Uint64()].L1BridgeRegistry
		if contractAddrStrBridgeRegistry == "" {
			return fmt.Errorf("L1BridgeRegistry variable not set in constant")
		}
		contractAddressBridgeRegistry := common.HexToAddress(contractAddrStrBridgeRegistry)

		// Create contract instance
		bridgeRegistryContract, err := abis.NewL1BridgeRegistry(contractAddressBridgeRegistry, l1Client)
		if err != nil {
			return fmt.Errorf("failed to create contract instance: %v", err)
		}

		txRegisterConfig, err := bridgeRegistryContract.RegisterRollupConfig(auth, common.HexToAddress(systemConfigProxy), 2, common.HexToAddress(l2TonAddress),
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
	l2ManagerAddressStr := constants.L1ChainConfigurations[chainID.Uint64()].L2ManagerAddress
	if l2ManagerAddressStr == "" {
		return fmt.Errorf("L2_MANAGER_ADDRESS variable is not set")
	}
	l2ManagerAddress := common.HexToAddress(l2ManagerAddressStr)

	tonAddressStr := constants.L1ChainConfigurations[chainID.Uint64()].TON
	if tonAddressStr == "" {
		return fmt.Errorf("TON variable is not set")
	}
	tonAddress := common.HexToAddress(tonAddressStr)

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
		common.HexToAddress(systemConfigProxy),
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
	err = t.setupSafeWallet(cwd)
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

func (t *ThanosStack) setupSafeWallet(cwd string) error {
	// 1. Set the safe wallet address
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
	fmt.Println("SafeWalletAddess: ", safeWalletAddress)
	// 2. Run hardhat task
	sdkPath := filepath.Join(cwd, "tokamak-thanos", "packages", "tokamak", "sdk")
	cmdStr := fmt.Sprintf("cd %s && L1_URL=%s PRIVATE_KEY=%s SAFE_WALLET_ADDRESS=%s npx hardhat set-safe-wallet", sdkPath, t.deployConfig.L1RPCURL, t.deployConfig.AdminPrivateKey, safeWalletAddress)
	if err := utils.ExecuteCommandStream("bash", "-c", cmdStr); err != nil {
		fmt.Print("\rfailed to setup the Safe wallet!\n")
		return err
	}

	return nil
}
