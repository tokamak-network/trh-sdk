package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// VerifyRegisterCandidates verifies the register candidates.
func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, config *types.Config) error {
	registerCandidateInputs, err := t.inputRegisterCandidate("")
	if err != nil {
		return err
	}

	err = t.verifyRegisterCandidates(ctx, config, registerCandidateInputs)
	if err != nil {
		return err
	}
	return nil
}

// isFromDeployContractStep flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) verifyRegisterCandidates(ctx context.Context, config *types.Config, registerCandidateInputs *RegisterCandidateInput) error {
	var (
		l1ChainID = new(big.Int).SetUint64(config.L1ChainID)
	)

	l1Client, err := ethclient.DialContext(ctx, config.L1RPCURL)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error determining current directory:", err)
		return err
	}

	file, err := os.Open(fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", cwd, fmt.Sprintf("%d-deploy.json", l1ChainID)))
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

	fmt.Println("Retrieving accounts...")
	accounts, err := types.GetAccountMap(ctx, l1Client, registerCandidateInputs.seed)
	if err != nil {
		return err
	}
	displayAccounts(accounts)

	fmt.Print("Please enter the private key of the account you want to use to register candidates: ")
	var selectedAccount types.Account
	for {
		input, err := scanner.ScanString()
		if err != nil {
			fmt.Printf("Failed to read input: %s", err)
			return err
		}

		selectingIndex, err := strconv.Atoi(input)
		if err != nil || selectingIndex < 0 || selectingIndex >= len(accounts) {
			fmt.Println("Invalid selection. Please try again.")
			continue
		}

		selectedAccount = accounts[selectingIndex]
		break
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(selectedAccount.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	fmt.Println("Private key selected:", privateKey)

	// Create transaction auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, l1ChainID)
	if err != nil {
		return fmt.Errorf("failed to create transaction auth: %v", err)
	}

	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	// Get contract address from environment
	contractAddrStr := constants.L1ChainConfigurations[l1ChainID.Uint64()].L1VerificationContractAddress
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

	l2TonAddress := constants.L1ChainConfigurations[l1ChainID.Uint64()].L2TonAddress
	if l2TonAddress == "" {
		return fmt.Errorf("L2TonAddress variable is not set")
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
			registerCandidateInputs.nameInfo,
			common.HexToAddress(registerCandidateInputs.safeWalletAddress),
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
		contractAddrStrBridgeRegistry := constants.L1ChainConfigurations[l1ChainID.Uint64()].L1BridgeRegistry
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
			registerCandidateInputs.nameInfo)

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
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidateInputs.amount), big.NewFloat(1e18))
	amountBigInt, _ := amountInWei.Int(nil)

	// Get contract address from environment
	l2ManagerAddressStr := constants.L1ChainConfigurations[l1ChainID.Uint64()].L2ManagerAddress
	if l2ManagerAddressStr == "" {
		return fmt.Errorf("L2_MANAGER_ADDRESS variable is not set")
	}
	l2ManagerAddress := common.HexToAddress(l2ManagerAddressStr)

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		common.HexToAddress(registerCandidateInputs.rollupConfig),
		amountBigInt,
		registerCandidateInputs.useTon,
		registerCandidateInputs.memo,
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
