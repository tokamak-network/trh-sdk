package types

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"os"
	"regexp"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type K8sConfig struct {
	Namespace string `json:"namespace"`
}

type Config struct {
	AdminPrivateKey      string `json:"admin_private_key"`
	SequencerPrivateKey  string `json:"sequencer_private_key"`
	BatcherPrivateKey    string `json:"batcher_private_key"`
	ProposerPrivateKey   string `json:"proposer_private_key"`
	ChallengerPrivateKey string `json:"challenger_private_key,omitempty"`

	DeploymentPath string `json:"deployment_path"`
	L1RPCURL       string `json:"l1_rpc_url"`
	L1BeaconURL    string `json:"l1_beacon_url"`
	L1RPCProvider  string `json:"l1_rpc_provider"`
	L1ChainID      uint64 `json:"l1_chain_id"`
	L2ChainID      uint64 `json:"l2_chain_id"`

	Stack            string `json:"stack"`
	Network          string `json:"network"`
	EnableFraudProof bool   `json:"enable_fraud_proof"`

	// these fields are added after installing the infrastructure successfully
	L2RpcUrl string `json:"l2_rpc_url"`

	// AWS config
	AWS *AWSConfig `json:"aws"`

	// K8s config
	K8s *K8sConfig `json:"k8s"`

	ChainName string `json:"chain_name"`
}

const configFileName = "settings.json"

func (c *Config) WriteToJSONFile() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Write JSON to a file
	err = os.WriteFile(configFileName, data, 0644)
	if err != nil {
		fmt.Println("Error writing JSON file:", err)
		return err
	}

	return nil
}

func ReadConfigFromJSONFile() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	fileExist := utils.CheckFileExists(fmt.Sprintf("%s/%s", cwd, configFileName))
	if !fileExist {
		return nil, nil
	}

	data, err := os.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// If L2ChainId doesn't exist, fetch it from the L2 RPC
	if config.L2ChainID == 0 {
		l2Provider, err := ethclient.Dial(config.L2RpcUrl)
		if err != nil {
			fmt.Println("Error connecting to L2 blockchain:", err)
			return nil, err
		}

		chainId, err := l2Provider.ChainID(context.Background())
		if err != nil || chainId == nil {
			fmt.Println("Error getting L2 chain id:", err)
			return nil, err
		}
		config.L2ChainID = chainId.Uint64()
	}

	return &config, nil
}

func ConvertChainNameToNamespace(chainName string) string {
	processed := strings.ToLower(chainName)
	processed = strings.ReplaceAll(processed, " ", "-")
	processed = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(processed, "")
	processed = strings.Trim(processed, "-")
	if len(processed) > 63 {
		processed = processed[:63]
	}
	return processed
}
