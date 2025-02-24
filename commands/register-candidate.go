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
)

// Layer2ManagerABI contains the ABI definition for the Layer2Manager contract
const Layer2ManagerABI = `[
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "_amount",
				"type": "uint256"
			},
			{
				"internalType": "string",
				"name": "_memo",
				"type": "string"
			}
		],
		"name": "registerCandidateAddOn",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

func ActionRegisterCandidates() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get flags
		rpcURL := cmd.String("rpc-url")
		amount := float64(cmd.Float("amount"))
		memo := cmd.String("memo")

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
		contractAbi, err := abi.JSON(strings.NewReader(Layer2ManagerABI))
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
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1)) // Replace 1 with actual chain ID
		if err != nil {
			return fmt.Errorf("failed to create transaction options: %v", err)
		}

		// Call registerCandidateAddOn
		tx, err := contract.Transact(auth, "registerCandidateAddOn", amountBigInt, memo)
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