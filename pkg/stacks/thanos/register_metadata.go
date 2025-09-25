package thanos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const (
	MetadataRepoURL  = "https://github.com/tokamak-network/tokamak-rollup-metadata-repository.git"
	MetadataRepoName = "tokamak-rollup-metadata-repository"
)

// getGitHubCredentials prompts user for GitHub username and personal access token
func GetGitHubCredentials() (*types.GitHubCredentials, error) {
	fmt.Println("\n🔑 GitHub Authentication Required")
	fmt.Println("   You'll need a Personal Access Token to push changes")
	fmt.Println("   Create one at: https://github.com/settings/tokens/new")
	fmt.Println("   Required scopes: repo (or public_repo for public repos)")

	var username, token, email string
	var err error

	// Get username
	for {
		fmt.Print("\nEnter your GitHub username: ")
		username, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error reading username: %s\n", err)
			continue
		}
		if strings.TrimSpace(username) == "" {
			fmt.Println("Username cannot be empty. Please try again.")
			continue
		}
		break
	}

	// Get personal access token
	for {
		fmt.Print("Enter your GitHub Personal Access Token: ")
		token, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error reading token: %s\n", err)
			continue
		}
		if strings.TrimSpace(token) == "" {
			fmt.Println("Token cannot be empty. Please try again.")
			continue
		}
		// Basic token validation (GitHub tokens start with specific prefixes)
		trimmedToken := strings.TrimSpace(token)
		if !strings.HasPrefix(trimmedToken, "ghp_") && !strings.HasPrefix(trimmedToken, "github_pat_") {
			fmt.Println("⚠️  Warning: Token doesn't look like a valid GitHub token, but continuing...")
		}
		break
	}

	// Get email address
	for {
		fmt.Print("Enter your git email: ")
		email, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error reading email: %s\n", err)
			continue
		}
		if strings.TrimSpace(email) == "" {
			fmt.Println("Email cannot be empty. Please try again.")
			continue
		}
		// Basic email validation
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
			fmt.Println("Please enter a valid email address.")
			continue
		}
		break
	}

	return &types.GitHubCredentials{
		Username: strings.TrimSpace(username),
		Token:    strings.TrimSpace(token),
		Email:    strings.TrimSpace(email),
	}, nil
}

func (t *ThanosStack) handleBranchCheckout(ctx context.Context, branchName string) error {
	t.logger.Info("Fetching latest changes from remote...")
	if err := utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "fetch", "origin"); err != nil {
		t.logger.Warn("Failed to fetch from remote, continuing with local branches", "error", err)
	}

	if err := utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "rev-parse", "--verify", "--quiet", branchName); err == nil {
		t.logger.Info("Branch exists locally, switching to branch", "branch", branchName)

		if err := utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "checkout", branchName); err != nil {
			return fmt.Errorf("failed to checkout existing branch %s: %w", branchName, err)
		}

		t.logger.Info("Pulling latest changes from remote", "branch", branchName)
		if err := utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "pull", "origin", branchName); err != nil {
			t.logger.Warn("Failed to pull latest changes, continuing with local version", "error", err)
		}
	} else {
		t.logger.Info("Creating new branch from current HEAD", "branch", branchName)
		if err := utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "checkout", "-b", branchName); err != nil {
			return fmt.Errorf("failed to create new branch %s: %w", branchName, err)
		}
	}

	return nil
}

func (t *ThanosStack) RegisterMetadata(ctx context.Context, creds *types.GitHubCredentials, metadataInfo *types.MetadataInfo) (*types.RegisterMetadataDaoResult, error) {
	if creds == nil {
		t.logger.Error("Credentials are required")
		return nil, errors.New("credentials are required")
	}

	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	t.logger.Info("🔄 Generating rollup metadata and submitting PR...")

	// Change directory to the deployment path
	err := os.Chdir(t.deploymentPath)
	if err != nil {
		t.logger.Error("Failed to change directory", "err", err)
		return nil, fmt.Errorf("failed to change directory: %w", err)
	}

	stackInfo, err := t.ShowInformation(ctx)
	if err != nil {
		t.logger.Error("Failed to show stack information", "err", err)
		return nil, fmt.Errorf("failed to show stack information: %w", err)
	}

	var contracts *types.Contracts
	var newMetadataEntry bool
	var isOpenPR bool
	var branchName string
	networkName := constants.ChainIDToForgeChainName[t.deployConfig.L1ChainID]

	contracts, err = utils.ReadDeployementConfigFromJSONFile(t.deploymentPath, t.deployConfig.L1ChainID)
	if err != nil {
		t.logger.Error("Failed to read deployment config", "err", err)
		return nil, fmt.Errorf("failed to read deployment config: %w", err)
	}

	systemConfigAddress := strings.ToLower(contracts.SystemConfigProxy)
	if systemConfigAddress == "" {
		t.logger.Error("SystemConfigProxy address not found in deployment contracts")
		return nil, fmt.Errorf("SystemConfigProxy address not found in deployment contracts")
	}

	networkDir := fmt.Sprintf("%s/data/%s", MetadataRepoName, networkName)

	metadataFileName := fmt.Sprintf("%s.json", systemConfigAddress)
	sourceFile := fmt.Sprintf("%s/schemas/example-rollup-metadata.json", MetadataRepoName)
	targetFile := fmt.Sprintf("%s/%s", networkDir, metadataFileName)
	// STEP 1. Fork the repository
	t.logger.Info("📋 STEP 1: Forking repository...")
	forkExists, err := utils.CheckIfForkExists(creds.Username, creds.Token, MetadataRepoName)
	if err != nil {
		t.logger.Warn("⚠️ Warning: Could not check if fork exists", "err", err)
		forkExists = false
	}

	if forkExists {
		t.logger.Info("✅ Fork already exists ", "user ", creds.Username, "repo ", MetadataRepoName)
	} else {
		err = utils.ForkRepository(creds.Username, creds.Token, MetadataRepoName)
		if err != nil {
			return nil, fmt.Errorf("failed to fork repository: %w", err)
		}
	}

	// STEP 2. Clone the user's forked repository locally
	t.logger.Info("📋 STEP 2: Cloning your forked repository...")
	forkURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", creds.Username, creds.Token, creds.Username, MetadataRepoName)
	err = t.cloneSourcecode(ctx, MetadataRepoName, forkURL)
	if err != nil {
		t.logger.Error("Failed to clone fork ", "err ", err)
		return nil, fmt.Errorf("failed to clone fork: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "remote", "add", "upstream", MetadataRepoURL)
	if err != nil {
		t.logger.Warn("⚠️ Warning: Could not add upstream remote ", "err ", err)
	}

	// Fetch latest changes from upstream
	t.logger.Info("📥 Fetching latest changes from upstream...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "fetch", "upstream")
	if err != nil {
		t.logger.Warn("⚠️ Warning: Failed to fetch upstream ", "err ", err)
	}

	t.logger.Info("🔄 Updating main branch...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "checkout", "main")
	if err != nil {
		t.logger.Warn("⚠️ Warning: Failed to checkout main ", "err ", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "merge", "upstream/main")
	if err != nil {
		t.logger.Warn("⚠️ Warning: Failed to merge upstream/main ", "err ", err)
	}

	// STEP 3. Create and checkout new branch
	fileExists := utils.CheckFileExists(targetFile)
	t.logger.Info("📋 STEP 3: Checking for existing metadata and creating/updating branch...")
	if !fileExists {
		branchName = fmt.Sprintf("feat/add-rollup-%s", systemConfigAddress)
		t.logger.Info("Creating and checking out branch ", "branch ", branchName)
		newMetadataEntry = true
	} else {
		branchName = fmt.Sprintf("feat/update-rollup-%s", systemConfigAddress)
		t.logger.Info("✅ Metadata file already exists! ", "file ", targetFile)
		newMetadataEntry = false
	}
	// Checkout the branch
	err = t.handleBranchCheckout(ctx, branchName)
	if err != nil {
		return nil, err
	}

	t.logger.Info("📋 STEP 3.5: Checking for existing Pull Requests...")
	existingPR, err := utils.CheckExistingPRForBranch(creds.Username, creds.Token, MetadataRepoName, branchName)
	if err != nil {
		t.logger.Warn("⚠️ Warning: Could not check for existing PRs ", "err ", err)
	} else if existingPR != nil {
		t.logger.Info("✅ Found existing PR ", existingPR.Title)
		t.logger.Info("PR Status: ", existingPR.State)

		isOpenPR = utils.CheckPRStatus(existingPR)
	} else {
		t.logger.Info("✅ No existing PRs found for branch ", "branch ", branchName)
	}

	// STEP 4. Install dependencies
	t.logger.Info("📋 STEP 4: Installing dependencies...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && pnpm install", MetadataRepoName))
	if err != nil {
		t.logger.Error("Failed to install dependencies ", "err ", err)
		return nil, fmt.Errorf("failed to install dependencies: %w", err)
	}

	// STEP 5. Create metadata file
	t.logger.Info("📋 STEP 5: Creating metadata file...")
	if t.deployConfig.L1ChainID != constants.EthereumSepoliaChainID { // Sepolia chain ID
		t.logger.Error("Unsupported network ", "chainID ", t.deployConfig.L1ChainID)
		return nil, fmt.Errorf("unsupported network. Currently only Sepolia (chain ID: 11155111) is supported, got chain ID: %d", t.deployConfig.L1ChainID)
	}

	if newMetadataEntry {
		t.logger.Info("Creating new metadata file ", "file ", targetFile)
		err = utils.ExecuteCommandStream(ctx, t.logger, "cp", sourceFile, targetFile)
		if err != nil {
			t.logger.Error("Failed to copy example metadata file ", "err ", err)
			return nil, fmt.Errorf("failed to copy example metadata file: %w", err)
		}
	}

	t.logger.Info("Updating metadata file with deployment information...")

	metadataBytes, err := os.ReadFile(targetFile)
	if err != nil {
		t.logger.Error("Failed to read metadata file ", "err ", err)
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata types.RollupMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.logger.Error("Failed to unmarshal metadata ", "err ", err)
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	metadata.L1ChainId = t.deployConfig.L1ChainID
	metadata.L2ChainId = t.deployConfig.L2ChainID
	metadata.Name = t.deployConfig.ChainName
	metadata.Description = metadataInfo.Chain.Description
	metadata.Logo = metadataInfo.Chain.Logo
	metadata.Website = metadataInfo.Chain.Website

	info, ok := debug.ReadBuildInfo()
	if ok {
		version := info.Main.Version
		metadata.Stack.Version = version
	}

	timestamp := time.Now().UTC()
	if newMetadataEntry {
		metadata.CreatedAt = timestamp.Format(time.RFC3339)
	}
	metadata.LastUpdated = timestamp.Format(time.RFC3339)

	metadata.Status = "active"

	metadata.RpcUrl = t.deployConfig.L2RpcUrl
	if strings.HasPrefix(t.deployConfig.L2RpcUrl, "https://") {
		metadata.WsUrl = strings.Replace(t.deployConfig.L2RpcUrl, "https://", "wss://", 1)
	} else if strings.HasPrefix(t.deployConfig.L2RpcUrl, "http://") {
		metadata.WsUrl = strings.Replace(t.deployConfig.L2RpcUrl, "http://", "ws://", 1)
	}

	metadata.L1Contracts = types.L1Contracts{
		ProxyAdmin:                   contracts.ProxyAdmin,
		SystemConfig:                 contracts.SystemConfigProxy,
		AddressManager:               contracts.AddressManager,
		SuperchainConfig:             contracts.SuperchainConfigProxy,
		DisputeGameFactory:           contracts.DisputeGameFactoryProxy,
		L1CrossDomainMessenger:       contracts.L1CrossDomainMessengerProxy,
		L1ERC721Bridge:               contracts.L1ERC721BridgeProxy,
		L1StandardBridge:             contracts.L1StandardBridgeProxy,
		OptimismMintableERC20Factory: contracts.OptimismMintableERC20FactoryProxy,
		OptimismPortal:               contracts.OptimismPortalProxy,
		AnchorStateRegistry:          contracts.AnchorStateRegistryProxy,
		DelayedWETH:                  contracts.DelayedWETHProxy,
		L1UsdcBridge:                 contracts.L1UsdcBridgeProxy,
		L2OutputOracle:               contracts.L2OutputOracleProxy,
		Mips:                         contracts.Mips,
		PermissionedDelayedWETH:      contracts.PermissionedDelayedWETHProxy,
		PreimageOracle:               contracts.PreimageOracle,
		ProtocolVersions:             contracts.ProtocolVersionsProxy,
		SafeProxyFactory:             contracts.SafeProxyFactory,
		SafeSingleton:                contracts.SafeSingleton,
		SystemOwnerSafe:              contracts.SystemOwnerSafe,
	}

	metadata.L2Contracts = types.L2Contracts{
		NativeToken:                   "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		WETH:                          "0x4200000000000000000000000000000000000006",
		L2ToL1MessagePasser:           "0x4200000000000000000000000000000000000016",
		DeployerWhitelist:             "0x4200000000000000000000000000000000000002",
		L2CrossDomainMessenger:        "0x4200000000000000000000000000000000000007",
		GasPriceOracle:                "0x420000000000000000000000000000000000000F",
		L2StandardBridge:              "0x4200000000000000000000000000000000000010",
		SequencerFeeVault:             "0x4200000000000000000000000000000000000011",
		OptimismMintableERC20Factory:  "0x4200000000000000000000000000000000000012",
		L1BlockNumber:                 "0x4200000000000000000000000000000000000013",
		L1Block:                       "0x4200000000000000000000000000000000000015",
		GovernanceToken:               "0x4200000000000000000000000000000000000042",
		LegacyMessagePasser:           "0x4200000000000000000000000000000000000000",
		L2ERC721Bridge:                "0x4200000000000000000000000000000000000014",
		OptimismMintableERC721Factory: "0x4200000000000000000000000000000000000017",
		ProxyAdmin:                    "0x4200000000000000000000000000000000000018",
		BaseFeeVault:                  "0x4200000000000000000000000000000000000019",
		L1FeeVault:                    "0x420000000000000000000000000000000000001a",
		ETH:                           "0x4200000000000000000000000000000000000486",
	}

	if stackInfo.BridgeUrl != "" {
		metadata.Bridges = []types.Bridge{
			{
				Name:            metadataInfo.Bridge.Name,
				Type:            metadata.Bridges[0].Type,
				URL:             stackInfo.BridgeUrl,
				Status:          "active",
				SupportedTokens: metadata.Bridges[0].SupportedTokens,
			},
		}
	}

	if stackInfo.BlockExplorer != "" {
		metadata.Explorers = []types.Explorer{
			{
				Name:   metadataInfo.Explorer.Name,
				URL:    stackInfo.BlockExplorer,
				Type:   metadata.Explorers[0].Type,
				Status: "active",
			},
		}
	}

	if t.deployConfig.StakingInfo != nil {
		parsedTime, _ := time.Parse("2006-01-02 15:04:05 MST", t.deployConfig.StakingInfo.RegistrationTime)
		metadata.Staking = types.Staking{
			IsCandidate:           t.deployConfig.StakingInfo.IsCandidate,
			CandidateRegisteredAt: parsedTime.UTC().Format(time.RFC3339),
			CandidateStatus:       "active",
			RegistrationTxHash:    t.deployConfig.StakingInfo.RegistrationTxHash,
			CandidateAddress:      t.deployConfig.StakingInfo.CandidateAddress,
		}
	} else {
		metadata.Staking = types.Staking{
			IsCandidate: false,
		}
	}

	sequencerAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.SequencerPrivateKey)
	if err != nil {
		t.logger.Error("Failed to get sequencer address ", "err ", err)
		return nil, fmt.Errorf("failed to get sequencer address: %w", err)
	}

	batcherAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.BatcherPrivateKey)
	if err != nil {
		t.logger.Error("Failed to get batcher address ", "err ", err)
		return nil, fmt.Errorf("failed to get batcher address: %w", err)
	}

	proposerAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.ProposerPrivateKey)
	if err != nil {
		t.logger.Error("Failed to get proposer address ", "err ", err)
		return nil, fmt.Errorf("failed to get proposer address: %w", err)
	}

	metadata.Sequencer = types.SequencerInfo{
		Address:         sequencerAddress.String(),
		BatcherAddress:  batcherAddress.String(),
		ProposerAddress: proposerAddress.String(),
	}

	metadata.NetworkConfig = types.NetworkConfig{
		BlockTime:         int(t.deployConfig.ChainConfiguration.L2BlockTime),
		GasLimit:          "0x1c9c380",
		BaseFeePerGas:     "0x3b9aca00",
		PriorityFeePerGas: metadata.NetworkConfig.PriorityFeePerGas,
	}

	metadata.WithdrawalConfig = types.WithdrawalConfig{
		ChallengePeriod:         int(t.deployConfig.ChainConfiguration.ChallengePeriod),
		ExpectedWithdrawalDelay: max(int(t.deployConfig.ChainConfiguration.BatchSubmissionFrequency), int(t.deployConfig.ChainConfiguration.OutputRootFrequency)) + int(t.deployConfig.ChainConfiguration.ChallengePeriod),
		MonitoringInfo: types.MonitoringInfoMetadata{
			L2OutputOracleAddress:    contracts.L2OutputOracleProxy,
			OutputProposedEventTopic: metadata.WithdrawalConfig.MonitoringInfo.OutputProposedEventTopic,
		},
		BatchSubmissionFrequency: int(t.deployConfig.ChainConfiguration.BatchSubmissionFrequency),
		OutputRootFrequency:      int(t.deployConfig.ChainConfiguration.OutputRootFrequency),
	}

	metadata.SupportResources = types.SupportResources{
		StatusPageUrl:     metadataInfo.Support.StatusPageUrl,
		SupportContactUrl: metadataInfo.Support.SupportContactUrl,
		DocumentationUrl:  metadataInfo.Support.DocumentationUrl,
		CommunityUrl:      metadataInfo.Support.CommunityUrl,
		HelpCenterUrl:     metadataInfo.Support.HelpCenterUrl,
		AnnouncementUrl:   metadataInfo.Support.AnnouncementUrl,
	}

	var operation string
	if newMetadataEntry {
		operation = "register"
	} else {
		operation = "update"
	}

	message := fmt.Sprintf("Tokamak Rollup Registry\n"+
		"L1 Chain ID: %d\n"+
		"L2 Chain ID: %d\n"+
		"Operation: %s\n"+
		"SystemConfig: %s\n"+
		"Timestamp: %d",
		metadata.L1ChainId,
		metadata.L2ChainId,
		operation,
		strings.ToLower(metadata.L1Contracts.SystemConfig),
		timestamp.Unix())

	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixedMessage))

	privateKeyHex := strings.TrimPrefix(t.deployConfig.SequencerPrivateKey, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		t.logger.Error("Failed to parse sequencer private key ", "err ", err)
		return nil, fmt.Errorf("failed to parse sequencer private key: %w", err)
	}

	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		t.logger.Error("Failed to sign metadata ", "err ", err)
		return nil, fmt.Errorf("failed to sign metadata: %w", err)
	}

	if signature[64] < 27 {
		signature[64] += 27
	}

	signatureHex := "0x" + hex.EncodeToString(signature)

	metadata.Metadata = types.Metadata{
		Version:   "1.0.0",
		Signature: signatureHex,
		SignedBy:  sequencerAddress.String(),
	}

	updatedMetadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.logger.Error("Failed to marshal updated metadata ", "err ", err)
		return nil, fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	if err := os.WriteFile(targetFile, updatedMetadataBytes, 0644); err != nil {
		t.logger.Error("Failed to write updated metadata file ", "err ", err)
		return nil, fmt.Errorf("failed to write updated metadata file: %w", err)
	}
	t.logger.Info("✅ Metadata file updated successfully!")

	// STEP 6. Validate metadata
	var prTitle string
	if newMetadataEntry {
		prTitle = fmt.Sprintf("[Rollup] %s %s - %s", networkName, systemConfigAddress, t.deployConfig.ChainName)
	} else {
		prTitle = fmt.Sprintf("[Update] %s %s - %s", networkName, systemConfigAddress, t.deployConfig.ChainName)
	}

	validationPath := fmt.Sprintf("data/%s/%s.json", networkName, systemConfigAddress)
	validationCmd := fmt.Sprintf("cd %s && npm run validate -- --pr-title \"%s\" %s", MetadataRepoName, prTitle, validationPath)

	t.logger.Info("📋STEP 6: Running validation command: ", validationCmd)
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", validationCmd)
	if err != nil {
		return nil, fmt.Errorf("metadata validation failed: %w", err)
	}
	t.logger.Info("✅ Metadata file validated successfully!")

	// STEP 7. Setup git config and commit changes
	t.logger.Info("📋 STEP 7: Committing changes...")
	err = utils.SetupGitConfig(ctx, t.logger, MetadataRepoName, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to setup git config: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "add", ".")
	if err != nil {
		return nil, fmt.Errorf("failed to add changes: %w", err)
	}

	commitMessage := fmt.Sprintf("[Rollup] %s %s %s - %s", operation, networkName, systemConfigAddress, t.deployConfig.ChainName)
	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "commit", "-m", commitMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to commit changes: %w", err)
	}
	t.logger.Info("✅ Changes committed successfully!")

	// STEP 8. Push changes to user's fork
	t.logger.Info("📋 STEP 8: Pushing changes to your fork...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "git", "-C", MetadataRepoName, "push", "origin", branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to push changes to fork: %w", err)
	}
	t.logger.Info("✅ Changes pushed to fork successfully!")

	// STEP 9. Create Pull Request from user's fork to original repo
	if isOpenPR {
		t.logger.Info("📋 STEP 9: Existing open PR found ", t.deployConfig.MetadataPRLink)
		t.logger.Info("Skipping PR creation.")
		return &types.RegisterMetadataDaoResult{
			PRLink: t.deployConfig.MetadataPRLink,
		}, nil
	}
	t.logger.Info("📋 STEP 9: Creating Pull Request...")
	prLink, err := utils.CreateGitHubPRFromFork(prTitle, branchName, creds.Username, creds.Token, MetadataRepoName, newMetadataEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	t.deployConfig.MetadataPRLink = prLink
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Failed to write settings file ", "err ", err)
		return nil, err
	}

	t.logger.Info("🎉 Metadata registration process completed!")
	t.logger.Info("📋 Summary:")
	t.logger.Info("✅ Repository forked to ", creds.Username, " ", MetadataRepoName)
	t.logger.Info("✅ Metadata file created: ", targetFile)
	t.logger.Info("✅ Changes committed and pushed to fork")
	t.logger.Info("✅ Pull request created from fork to original repository")

	return &types.RegisterMetadataDaoResult{
		PRLink: prLink,
	}, nil
}
