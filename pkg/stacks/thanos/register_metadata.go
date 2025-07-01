package thanos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const (
	MetadataRepoURL  = "https://github.com/tokamak-network/tokamak-rollup-metadata-repository.git"
	MetadataRepoName = "tokamak-rollup-metadata-repository"
)

type RollupMetadata struct {
	L1ChainId   uint64 `json:"l1ChainId"`
	L2ChainId   uint64 `json:"l2ChainId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Website     string `json:"website"`

	RollupType string `json:"rollupType"`
	Stack      Stack  `json:"stack"`

	RpcUrl string `json:"rpcUrl"`
	WsUrl  string `json:"wsUrl"`

	NativeToken NativeToken `json:"nativeToken"`

	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	LastUpdated string `json:"lastUpdated"`

	L1Contracts      L1Contracts      `json:"l1Contracts"`
	L2Contracts      L2Contracts      `json:"l2Contracts"`
	Bridges          []Bridge         `json:"bridges"`
	Explorers        []Explorer       `json:"explorers"`
	Sequencer        Sequencer        `json:"sequencer"`
	Staking          Staking          `json:"staking"`
	NetworkConfig    NetworkConfig    `json:"networkConfig"`
	WithdrawalConfig WithdrawalConfig `json:"withdrawalConfig"`
	SupportResources SupportResources `json:"supportResources"`
	Metadata         Metadata         `json:"metadata"`
}

type Stack struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type NativeToken struct {
	Type      string `json:"type"`
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Decimals  int    `json:"decimals"`
	L1Address string `json:"l1Address"`
}

type L1Contracts struct {
	L1CrossDomainMessenger string `json:"l1CrossDomainMessenger"`
	L1StandardBridge       string `json:"l1StandardBridge"`
	OptimismPortal         string `json:"optimismPortal"`
	SystemConfig           string `json:"systemConfig"`
}

type L2Contracts struct {
	NativeToken            string `json:"nativeToken"`
	L2CrossDomainMessenger string `json:"l2CrossDomainMessenger"`
	L2StandardBridge       string `json:"l2StandardBridge"`
	GasPriceOracle         string `json:"gasPriceOracle"`
	L1Block                string `json:"l1Block"`
	L2ToL1MessagePasser    string `json:"l2ToL1MessagePasser"`
	WrappedETH             string `json:"wrappedETH"`
}

type Bridge struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	URL             string  `json:"url"`
	Status          string  `json:"status"`
	SupportedTokens []Token `json:"supportedTokens"`
}

type Token struct {
	Symbol        string `json:"symbol"`
	L1Address     string `json:"l1Address"`
	L2Address     string `json:"l2Address"`
	Decimals      int    `json:"decimals"`
	IsNativeToken bool   `json:"isNativeToken,omitempty"`
	IsWrappedETH  bool   `json:"isWrappedETH,omitempty"`
}

type Explorer struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type Sequencer struct {
	Address         string `json:"address"`
	BatcherAddress  string `json:"batcherAddress"`
	ProposerAddress string `json:"proposerAddress"`
}

type Staking struct {
	IsCandidate bool `json:"isCandidate"`
}

type NetworkConfig struct {
	BlockTime                int    `json:"blockTime"`
	GasLimit                 string `json:"gasLimit"`
	BaseFeePerGas            string `json:"baseFeePerGas"`
	PriorityFeePerGas        string `json:"priorityFeePerGas"`
	BatchSubmissionFrequency int    `json:"batchSubmissionFrequency"`
	OutputRootFrequency      int    `json:"outputRootFrequency"`
}

type WithdrawalConfig struct {
	ChallengePeriod         int            `json:"challengePeriod"`
	ExpectedWithdrawalDelay int            `json:"expectedWithdrawalDelay"`
	MonitoringInfo          MonitoringInfo `json:"monitoringInfo"`
}

type MonitoringInfo struct {
	L2OutputOracleAddress    string `json:"l2OutputOracleAddress"`
	OutputProposedEventTopic string `json:"outputProposedEventTopic"`
}

type SupportResources struct {
	StatusPageUrl     string `json:"statusPageUrl"`
	SupportContactUrl string `json:"supportContactUrl"`
	DocumentationUrl  string `json:"documentationUrl"`
	CommunityUrl      string `json:"communityUrl"`
	HelpCenterUrl     string `json:"helpCenterUrl"`
	AnnouncementUrl   string `json:"announcementUrl"`
}

type Metadata struct {
	Version   string `json:"version"`
	Signature string `json:"signature"`
	SignedBy  string `json:"signedBy"`
}

func (t *ThanosStack) RegisterMetadata(ctx context.Context) error {
	fmt.Println("🔄 Generating rollup metadata and submitting PR...")

	var err error
	l1Client, err := ethclient.DialContext(ctx, t.deployConfig.L1RPCURL)
	if err != nil {
		fmt.Printf("Failed to connect to L1 RPC: %s", err)
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

	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		fmt.Println("Error decoding deployment JSON file:", err)
		return err
	}

	systemConfigAddress := strings.ToLower(contracts.SystemConfigProxy)
	if systemConfigAddress == "" {
		return fmt.Errorf("SystemConfigProxy address not found in deployment contracts")
	}

	branchName := fmt.Sprintf("feat/add-rollup-%s", systemConfigAddress)
	repoPath := "tokamak-rollup-metadata-repository"

	// STEP 1. Clone the repository
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		err = t.cloneSourcecode("tokamak-rollup-metadata-repository", "https://github.com/tokamak-network/tokamak-rollup-metadata-repository.git")
		if err != nil {
			return err
		}
		fmt.Println("✅ Repository cloned successfully!")
	} else {
		fmt.Println("✅ Repository already cloned!")
	}

	// STEP 2. Checkout branch and install dependencies
	currentBranch, err := utils.ExecuteCommand("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	fmt.Printf("Current branch: %s\n", currentBranch)

	if branchName == currentBranch {
		fmt.Println("✅ Branch already exists!")
	} else {
		fmt.Printf("Creating and checking out branch: %s\n", branchName)
		err = utils.ExecuteCommandStream("git", "-C", "tokamak-rollup-metadata-repository", "checkout", "-b", branchName)
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch: %w", err)
		}
	}

	fmt.Println("Installing dependencies...")
	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-rollup-metadata-repository && npm install")
	if err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// STEP 3. Create metadata file
	if chainID.Uint64() != 11155111 { // Sepolia chain ID
		return fmt.Errorf("unsupported network. Currently only Sepolia (chain ID: 11155111) is supported, got chain ID: %d", chainID.Uint64())
	}
	networkDir := fmt.Sprintf("%s/data/sepolia", repoPath)

	metadataFileName := fmt.Sprintf("%s.json", systemConfigAddress)
	sourceFile := fmt.Sprintf("%s/schemas/example-rollup-metadata.json", repoPath)
	targetFile := fmt.Sprintf("%s/%s", networkDir, metadataFileName)

	fmt.Printf("Copying example metadata to %s...\n", targetFile)
	err = utils.ExecuteCommandStream("cp", sourceFile, targetFile)
	if err != nil {
		return fmt.Errorf("failed to copy example metadata file: %w", err)
	}

	fmt.Println("Updating metadata file with deployment information...")

	metadataBytes, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata RollupMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	metadata.L1ChainId = chainID.Uint64()
	metadata.L2ChainId = t.deployConfig.L2ChainID

	timestamp := time.Now().UTC()
	metadata.CreatedAt = timestamp.Format(time.RFC3339)
	metadata.LastUpdated = timestamp.Format(time.RFC3339)

	metadata.L1Contracts = L1Contracts{
		L1CrossDomainMessenger: contracts.L1CrossDomainMessengerProxy,
		L1StandardBridge:       contracts.L1StandardBridgeProxy,
		OptimismPortal:         contracts.OptimismPortalProxy,
		SystemConfig:           contracts.SystemConfigProxy,
	}

	sequencerAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.SequencerPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to get sequencer address: %w", err)
	}

	batcherAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.BatcherPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to get batcher address: %w", err)
	}

	proposerAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.ProposerPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to get proposer address: %w", err)
	}

	metadata.Sequencer = Sequencer{
		Address:         sequencerAddress.String(),
		BatcherAddress:  batcherAddress.String(),
		ProposerAddress: proposerAddress.String(),
	}

	message := fmt.Sprintf("Tokamak Rollup Registry\n"+
		"L1 Chain ID: %d\n"+
		"L2 Chain ID: %d\n"+
		"Operation: register\n"+
		"SystemConfig: %s\n"+
		"Timestamp: %d",
		metadata.L1ChainId,
		metadata.L2ChainId,
		strings.ToLower(metadata.L1Contracts.SystemConfig),
		timestamp.Unix())

	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)

	hash := crypto.Keccak256Hash([]byte(prefixedMessage))

	privateKeyHex := strings.TrimPrefix(t.deployConfig.SequencerPrivateKey, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to parse sequencer private key: %w", err)
	}

	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign metadata: %w", err)
	}

	if signature[64] < 27 {
		signature[64] += 27
	}

	signatureHex := "0x" + hex.EncodeToString(signature)

	metadata.Metadata = Metadata{
		Version:   "1.0.0",
		Signature: signatureHex,
		SignedBy:  sequencerAddress.String(),
	}

	updatedMetadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	if err := os.WriteFile(targetFile, updatedMetadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write updated metadata file: %w", err)
	}
	fmt.Println("✅ Metadata file updated successfully!")

	// STEP 4. Validate metadata
	fmt.Println("Validating metadata...")
	validationPath := fmt.Sprintf("data/sepolia/%s.json", systemConfigAddress)
	validationCmd := fmt.Sprintf("cd tokamak-rollup-metadata-repository && npm run validate %s", validationPath)

	fmt.Printf("Running validation command: %s\n", validationCmd)
	err = utils.ExecuteCommandStream("bash", "-c", validationCmd)
	if err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}
	fmt.Println("✅ Metadata file validated successfully!")

	// STEP 5. Commit changes, push to remote and create PR
	fmt.Println("Committing changes...")
	err = utils.ExecuteCommandStream("git", "-C", repoPath, "add", ".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}
	commitMessage := fmt.Sprintf("[Rollup] sepolia %s - %s", systemConfigAddress, t.deployConfig.ChainName)
	err = utils.ExecuteCommandStream("git", "-C", repoPath, "commit", "-m", commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	fmt.Println("✅ Changes committed successfully!")

	return nil
}
