package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/tokamak-network/trh-sdk/abis"
)

type RegisterRollupRequest struct {
	Rollup string `json:"rollup" binding:"required"`
	Type   uint8  `json:"type" binding:"required"`
	L2TON  string `json:"l2ton" binding:"required"`
	Name   string `json:"name" binding:"required"`
}

// RegisterRollupConfig handles the registration of rollup configuration
func RegisterRollupConfig(c *gin.Context) {
	var request RegisterRollupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Bad request: missing required fields")
		return
	}

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		c.String(http.StatusInternalServerError, "RPC_URL environment variable is not set")
		return
	}

	contractAddress := os.Getenv("L1_BRIDGE_REGISTRY_ADDRESS")
	if contractAddress == "" {
		c.String(http.StatusInternalServerError, "L1_BRIDGE_REGISTRY_ADDRESS environment variable is not set")
		return
	}

	// Validate addresses
	if !common.IsHexAddress(request.Rollup) || !common.IsHexAddress(request.L2TON) {
		c.String(http.StatusBadRequest, "Invalid address format")
		return
	}

	privateKeyString := os.Getenv("PRIVATE_KEY")
	if privateKeyString == "" {
		c.String(http.StatusBadRequest, "PRIVATE_KEY environment variable is not set")
		return
	}

	// Decrypt private key (implement your decryption logic here)
	decryptedKey, err := decryptPrivateKey(privateKeyString)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid private key")
		return
	}

	// Connect to Ethereum client (replace with your RPC URL)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to connect to Ethereum network")
		return
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(decryptedKey)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid private key format")
		return
	}

	// Get chain ID
	chainID, err := client.ChainID(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get chain ID")
		return
	}

	// Create transaction options
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create transaction options")
		return
	}

	// Create contract instance using the generated bindings
	contract, err := abis.NewL1BridgeRegistry(common.HexToAddress(contractAddress), client)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create contract instance")
		return
	}

	// Call registerRollupConfig using the generated method
	tx, err := contract.RegisterRollupConfig(
		auth,
		common.HexToAddress(request.Rollup),
		request.Type,
		common.HexToAddress(request.L2TON),
		request.Name,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to register rollup config: %v", err))
		return
	}

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(c.Request.Context(), client, tx)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get transaction receipt")
		return
	}

	if receipt.Status == 1 {
		c.String(http.StatusOK, "Rollup config registration successful via remote API")
	} else {
		c.String(http.StatusInternalServerError, "Transaction failed")
	}
}

func decryptPrivateKey(encryptedKey string) (string, error) {
	// TODO: Implement your decryption logic
	return encryptedKey, nil
}

// SetupRoutes sets up the API routes
func SetupRoutes(router *gin.Engine) {
	router.POST("/registerRollupConfig", RegisterRollupConfig)
}
