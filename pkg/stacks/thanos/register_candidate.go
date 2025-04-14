package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/abis"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// fromDeployContract flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) verifyRegisterCandidates(ctx context.Context, fromDeployContract bool, config *types.Config, registerCandidate *RegisterCandidateInput) error {
	var privateKeyString string

	l1Client, err := ethclient.DialContext(ctx, config.L1RPCURL)
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

	file, err := os.Open(fmt.Sprintf("%s/%s", cwd, fmt.Sprintf("%d-deploy.json", chainID)))
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

	if fromDeployContract {
		privateKeyString = config.AdminPrivateKey
	} else {
		operatorsSelected, err := selectAccounts(ctx, l1Client, config.EnableFraudProof, registerCandidate.seed)
		if err != nil {
			return err
		}
		privateKeyString = operatorsSelected[0].PrivateKey
	}

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
			common.HexToAddress(registerCandidate.safeWalletAddress),
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

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		common.HexToAddress(registerCandidate.rollupConfig),
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

func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, fromDeployContract bool, config *types.Config) (*RegisterCandidateInput, error) {
	registerCandidate, err := t.inputRegisterCandidate(fromDeployContract)
	if err != nil {
		return nil, err
	}
	if fromDeployContract {
		return registerCandidate, nil
	} else {
		err := t.verifyRegisterCandidates(ctx, fromDeployContract, config, registerCandidate)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
