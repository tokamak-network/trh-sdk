package types

import (
	"encoding/json"
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"os"
)

type Config struct {
	AdminPrivateKey      string `json:"admin_private_key"`
	SequencerPrivateKey  string `json:"sequencer_private_key"`
	BatcherPrivateKey    string `json:"batcher_private_key"`
	ProposerPrivateKey   string `json:"proposer_private_key"`
	ChallengerPrivateKey string `json:"challenger_private_key,omitempty"`

	DeploymentPath string `json:"deployment_path"`
	L1RPCURL       string `json:"l1_rpc_url"`
	L1RPCProvider  string `json:"l1_rpc_provider"`

	Stack            string `json:"stack"`
	Network          string `json:"network"`
	EnableFraudProof bool   `json:"enable_fraud_proof"`
}

func (c *Config) WriteToJSONFile(jsonFileName string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Write JSON to a file
	err = os.WriteFile(jsonFileName, data, 0644)
	if err != nil {
		fmt.Println("Error writing JSON file:", err)
		return err
	}

	return nil
}

func ReadConfigFromJSONFile(filename string) (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	fileExist := utils.CheckFileExists(fmt.Sprintf("%s/%s", cwd, filename))
	if !fileExist {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
