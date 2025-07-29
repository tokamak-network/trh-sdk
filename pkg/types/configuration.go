package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
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

type StakingInfo struct {
	IsCandidate         bool    `json:"is_candidate"`
	StakingAmount       float64 `json:"staking_amount"`
	RollupConfigAddress string  `json:"rollup_config_address"`
	CandidateName       string  `json:"candidate_name"`
	CandidateMemo       string  `json:"candidate_memo"`
	RegistrationTime    string  `json:"registration_time"`
	RegistrationTxHash  string  `json:"registration_tx_hash"`
	CandidateAddress    string  `json:"candidate_address"`
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

func (c *ChainConfiguration) Validate(l1ChainId uint64) error {
	if c.BatchSubmissionFrequency <= 0 {
		return errors.New("BatchSubmissionFrequency is not set")
	}

	if c.ChallengePeriod <= 0 {
		return errors.New("ChallengePeriod is not set")
	}

	if c.OutputRootFrequency <= 0 {
		return errors.New("OutputRootFrequency is not set")
	}

	if c.L1BlockTime <= 0 {
		return errors.New("L1BlockTime is not set")
	}

	if c.L2BlockTime <= 0 {
		return errors.New("L2BlockTime is not set")
	}

	if c.OutputRootFrequency%c.L2BlockTime != 0 {
		return fmt.Errorf("OutputRootFrequency must be a multiple of %d", c.L2BlockTime)
	}

	if c.BatchSubmissionFrequency%c.L1BlockTime != 0 {
		return fmt.Errorf("BatchSubmissionFrequency must be a multiple of %d", c.L1BlockTime)
	}

	if l1ChainId == constants.EthereumMainnetChainID {
		mainnetChallengePeriod := constants.L1ChainConfigurations[l1ChainId].FinalizationPeriodSeconds
		if c.ChallengePeriod != mainnetChallengePeriod {
			return fmt.Errorf("challengePeriod must be equal by %d", mainnetChallengePeriod)
		}
	}

	return nil
}

type DeployContractStatus int

const (
	DeployContractStatusInProgress = iota + 1
	DeployContractStatusCompleted
)

type DeployContractState struct {
	Status DeployContractStatus `json:"status"`
	Error  string               `json:"error,omitempty"`
}

type Config struct {
	AdminPrivateKey      string `json:"admin_private_key"`
	SequencerPrivateKey  string `json:"sequencer_private_key"`
	BatcherPrivateKey    string `json:"batcher_private_key"`
	ProposerPrivateKey   string `json:"proposer_private_key"`
	ChallengerPrivateKey string `json:"challenger_private_key,omitempty"`

	DeploymentFilePath string `json:"deployment_file_path"`
	L1RPCURL           string `json:"l1_rpc_url"`
	L1BeaconURL        string `json:"l1_beacon_url"`
	L1RPCProvider      string `json:"l1_rpc_provider"`
	L1ChainID          uint64 `json:"l1_chain_id"`
	L2ChainID          uint64 `json:"l2_chain_id"`

	Stack            string `json:"stack"`
	Network          string `json:"network"`
	EnableFraudProof bool   `json:"enable_fraud_proof"`

	// these fields are added after installing the infrastructure successfully
	L2RpcUrl string `json:"l2_rpc_url"`

	// AWS config
	AWS *AWSConfig `json:"aws,omitempty"`

	// K8s config
	K8s *K8sConfig `json:"k8s,omitempty"`

	ChainName string `json:"chain_name,omitempty"`

	ChainConfiguration *ChainConfiguration `json:"chain_configuration"`

	// Deployments
	DeployContractState *DeployContractState `json:"deploy_contract_state"`

	// Staking information
	StakingInfo *StakingInfo `json:"staking_info,omitempty"`
}

const ConfigFileName = "settings.json"

func (c *Config) WriteToJSONFile(deploymentPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	fileName := fmt.Sprintf("%s/%s", deploymentPath, ConfigFileName)
	// Write JSON to a file
	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		fmt.Println("Error writing JSON file:", err)
		return err
	}

	return nil
}

func (c *Config) SupportAWS() bool {
	return c != nil && (c.Network == constants.Testnet || c.Network == constants.Mainnet)
}
