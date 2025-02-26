package commands

import (
	"context"
	"fmt"
	"os"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v3"
	"github.com/tokamak-network/trh-sdk/abis"
)

func ActionRegisterCandidates() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get flags
		rollupConfig := cmd.String("rollup-config")
		amount := float64(cmd.Float("amount"))
		useTon := cmd.Bool("use-ton")
		memo := cmd.String("memo")

		rpcURL := os.Getenv("RPC_URL")
		if rpcURL == "" {
			return fmt.Errorf("RPC_URL environment variable is not set")
		}

		privateKeyString := os.Getenv("PRIVATE_KEY")
		if privateKeyString == "" {
			return fmt.Errorf("PRIVATE_KEY environment variable is not set")
		}

		// Parse private key
		privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyString, "0x"))
		if err != nil {
			return fmt.Errorf("invalid private key: %v", err)
		}

		l2ManagerContractAddress := os.Getenv("L2_MANAGER_ADDRESS")
		if l2ManagerContractAddress == "" {
			return fmt.Errorf("L2_MANAGER_ADDRESS environment variable is not set")
		}

		// Validate minimum amount
		if amount < 1000.1 {
			return fmt.Errorf("minimum staking amount is 1000.1 TON")
		}

		// Connect to Ethereum client
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to connect to the Ethereum client: %v", err)
		}

		// Create contract ABI
		contractAbi, err := abi.JSON(strings.NewReader(abis.Layer2ManagerABI))
		if err != nil {
			return fmt.Errorf("failed to parse contract ABI: %v", err)
		}

		// Create contract instance
		contractAddress := common.HexToAddress(l2ManagerContractAddress)
		contract := bind.NewBoundContract(contractAddress, contractAbi, client, client, client)

		// Convert amount to Wei (18 decimals)
		amountInWei := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1e18))
		amountBigInt := new(big.Int)
		amountInWei.Int(amountBigInt)

		// Create transaction options
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(11155111)) // Replace 1 with actual chain ID
		if err != nil {
			return fmt.Errorf("failed to create transaction options: %v", err)
		}

		// Call registerCandidateAddOn
		tx, err := contract.Transact(auth, "registerCandidateAddOn",common.HexToAddress(rollupConfig), amountBigInt, useTon, memo)
		if err != nil {
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		fmt.Printf("Transaction sent: %s\n", tx.Hash().Hex())
		fmt.Println("Waiting for transaction confirmation...")

		receipt, err := bind.WaitMined(ctx, client, tx)
		if err != nil {
			return fmt.Errorf("failed to get transaction receipt: %v", err)
		}

		if receipt.Status == 1 {
			fmt.Println("Successfully registered as candidate! ✅")
		} else {
			return fmt.Errorf("transaction failed")
		}

		return nil
	}
}