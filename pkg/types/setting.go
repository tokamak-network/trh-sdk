package types

import (
	"encoding/json"
	"fmt"
	"os"
)

type K8sConfig struct {
	Namespace string `json:"namespace"`
}

type ChainConfiguration struct {
	BatchSubmissionFrequency uint64 `json:"batch_submission_frequency"` // = l1BlockTime * maxChannelDuration
	ChallengePeriod          uint64 `json:"challenge_period"`           // = finalizationPeriodSeconds
	OutputRootFrequency      uint64 `json:"output_root_frequency"`      // = l2BlockTime * l2OutputOracleSubmissionInterval
	L2BlockTime              uint64 `json:"l2_block_time"`
	L1BlockTime              uint64 `json:"l1_block_time"`
}

func (c *ChainConfiguration) GetL2OutputOracleSubmissionInterval() uint64 {
	if c.L2BlockTime == 0 {
		panic("L2BlockTime is not set")
	}
	return c.OutputRootFrequency / c.L2BlockTime
}

func (c *ChainConfiguration) GetMaxChannelDuration() uint64 {
	if c.L1BlockTime == 0 {
		panic("L1BlockTime is not set")
	}
	return c.BatchSubmissionFrequency / c.L1BlockTime
}

func (c *ChainConfiguration) GetFinalizationPeriodSeconds() uint64 {
	if c.ChallengePeriod == 0 {
		panic("ChallengePeriod is not set")
	}
	return c.ChallengePeriod
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

	ChainConfiguration *ChainConfiguration `json:"chain_configuration"`
}

const ConfigFileName = "settings.json"

func (c *Config) WriteToJSONFile() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Write JSON to a file
	err = os.WriteFile(ConfigFileName, data, 0644)
	if err != nil {
		fmt.Println("Error writing JSON file:", err)
		return err
	}

	return nil
}
