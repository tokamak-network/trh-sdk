package thanos

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

const (
	MetadataRepoURL  = "https://github.com/tokamak-network/tokamak-rollup-metadata-repository.git"
	MetadataRepoName = "tokamak-rollup-metadata-repository"
)

// getGitHubCredentials prompts user for GitHub username and personal access token
func getGitHubCredentials() (*types.GitHubCredentials, error) {
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

// setupGitConfig ensures git config is set for the repository
func setupGitConfig(ctx context.Context, l *zap.SugaredLogger, repoPath string, creds *types.GitHubCredentials) error {
	// Check if git config is already set (either globally or locally)
	userName, err := utils.ExecuteCommand(ctx, "git", "-C", repoPath, "config", "user.name")
	if err == nil && strings.TrimSpace(userName) != "" {
		fmt.Printf("✅ Git user.name already configured: %s\n", strings.TrimSpace(userName))

		userEmail, err := utils.ExecuteCommand(ctx, "git", "-C", repoPath, "config", "user.email")
		if err == nil && strings.TrimSpace(userEmail) != "" {
			fmt.Printf("✅ Git user.email already configured: %s\n", strings.TrimSpace(userEmail))
			return nil
		}
	}

	fmt.Println("🔧 Git config not set. Please provide your git credentials for this repository:")

	// Set local git config for this repository
	fmt.Println("Setting git config for this repository...")

	err = utils.ExecuteCommandStream(ctx, l, "git", "-C", repoPath, "config", "user.name", creds.Username)
	if err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, l, "git", "-C", repoPath, "config", "user.email", creds.Email)
	if err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	fmt.Printf("✅ Git config set successfully!\n")
	fmt.Printf("   - user.name: %s\n", creds.Username)
	fmt.Printf("   - user.email: %s\n", creds.Email)

	return nil
}

func generateBasicPRDescription(newMetadataEntry bool) string {
	description := "## 🚀 Rollup Metadata Submission\n\n"

	if newMetadataEntry {
		description += "### 🏗️ Type of Submission\n"
		description += "- [x] **New rollup registration** (use `[Rollup]` in PR title)\n"
		description += "- [ ] **Update existing rollup metadata** (use `[Update]` in PR title)\n\n"
	} else {
		description += "### 🏗️ Type of Submission\n"
		description += "- [ ] **New rollup registration** (use `[Rollup]` in PR title)\n"
		description += "- [x] **Update existing rollup metadata** (use `[Update]` in PR title)\n\n"
	}

	description += "### ✅ Pre-submission Checklist\n"
	description += "- [x] My PR title follows the correct format\n"
	description += "- [x] I have used **lowercase** for the SystemConfig address\n"
	description += "- [x] My filename is `<systemConfig_address>.json` (lowercase)\n"
	description += "- [x] I have added only one rollup metadata file\n"
	description += "- [x] I have signed the metadata with the correct operation (`register` or `update`)\n"
	description += "- [x] I am the authorized sequencer of this L2 rollup\n\n"

	description += "---\n\n"
	description += "**All rollup details are in the metadata file. This submission confirms I have the authority to register/update this rollup.**"

	return description
}

// createGitHubPRFromFork creates a pull request from a forked repository using GitHub API
func createGitHubPRFromFork(prTitle, branchName, username, token string, newMetadataEntry bool) error {
	fmt.Println("\n🔗 Creating Pull Request from fork...")

	prDescription := generateBasicPRDescription(newMetadataEntry)

	prData := types.GitHubPR{
		Title: prTitle,
		Head:  fmt.Sprintf("%s:%s", username, branchName),
		Base:  "main",
		Body:  prDescription,
	}

	jsonData, err := json.Marshal(prData)
	if err != nil {
		return fmt.Errorf("failed to marshal PR data: %w", err)
	}

	// Create HTTP request
	url := "https://api.github.com/repos/tokamak-network/tokamak-rollup-metadata-repository/pulls"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 201 {
		// Successfully created
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if prURL, ok := result["html_url"].(string); ok {
				fmt.Printf("✅ Pull Request created successfully!\n")
				fmt.Printf("🔗 PR URL: %s\n", prURL)
				return nil
			}
		}
		fmt.Println("✅ Pull Request created successfully!")
		return nil
	} else if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - please check your GitHub token has the correct permissions")
	} else if resp.StatusCode == 422 {
		// Parse error details
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if errors, ok := errorResp["errors"].([]interface{}); ok && len(errors) > 0 {
				if firstError, ok := errors[0].(map[string]interface{}); ok {
					if message, ok := firstError["message"].(string); ok {
						return fmt.Errorf("PR creation failed: %s", message)
					}
				}
			}
		}
		return fmt.Errorf("PR creation failed - possibly branch already has a PR or validation error")
	} else {
		return fmt.Errorf("PR creation failed with status %d", resp.StatusCode)
	}
}

func (t *ThanosStack) RegisterMetadata(ctx context.Context) error {
	fmt.Println("🔄 Generating rollup metadata and submitting PR...")

	stackInfo, err := t.ShowInformation(ctx)
	if err != nil {
		return fmt.Errorf("failed to show stack information: %w", err)
	}

	// Get GitHub credentials first (we need them for forking)
	creds, err := getGitHubCredentials()
	if err != nil {
		return fmt.Errorf("failed to get GitHub credentials: %w", err)
	}

	var contracts *types.Contracts
	var metadataInfo *types.MetadataInfo
	var newMetadataEntry bool

	contracts, err = utils.ReadDeployementConfigFromJSONFile(t.deploymentPath, t.deployConfig.L1ChainID)
	if err != nil {
		return fmt.Errorf("failed to read deployment config: %w", err)
	}

	metadataInfo, err = utils.ReadMetadataInfoFromJSONFile(t.deploymentPath, t.deployConfig.L1ChainID)
	if err != nil {
		return fmt.Errorf("failed to read metadata info: %w", err)
	}

	systemConfigAddress := strings.ToLower(contracts.SystemConfigProxy)
	if systemConfigAddress == "" {
		return fmt.Errorf("SystemConfigProxy address not found in deployment contracts")
	}

	branchName := fmt.Sprintf("feat/add-rollup-%s", systemConfigAddress)
	// STEP 1. Fork the repository
	forkExists, err := t.checkIfForkExists(creds.Username, creds.Token, MetadataRepoName)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not check if fork exists: %v\n", err)
		forkExists = false
	}

	if forkExists {
		fmt.Printf("✅ Fork already exists at %s/%s\n", creds.Username, MetadataRepoName)
	} else {
		err = t.forkRepository(creds.Username, creds.Token, MetadataRepoName)
		if err != nil {
			return fmt.Errorf("failed to fork repository: %w", err)
		}
	}

	// STEP 2. Clone the user's forked repository locally
	forkURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", creds.Username, creds.Token, creds.Username, MetadataRepoName)
	err = t.cloneSourcecode(ctx, MetadataRepoName, forkURL)
	if err != nil {
		return fmt.Errorf("failed to clone fork: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "remote", "add", "upstream", MetadataRepoURL)
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to add upstream remote: %v\n", err)
	}

	// Fetch latest changes from upstream
	fmt.Println("📥 Fetching latest changes from upstream...")
	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "fetch", "upstream")
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to fetch upstream: %v\n", err)
	}

	fmt.Println("🔄 Updating main branch...")
	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "checkout", "main")
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to checkout main: %v\n", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "merge", "upstream/main")
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to merge upstream/main: %v\n", err)
	}

	// STEP 3. Create and checkout new branch
	fmt.Println("\n📋 STEP 3: Creating feature branch...")
	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "fetch", "origin", fmt.Sprintf("%s:%s", branchName, branchName))
	// Checking if the branch already exists
	if err != nil {
		fmt.Printf("Creating and checking out branch: %s\n", branchName)
		newMetadataEntry = true
		err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "checkout", "-b", branchName)
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch: %w", err)
		}
	} else {
		fmt.Println("✅ Branch already exists!")
		newMetadataEntry = false
		err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "checkout", branchName)
		if err != nil {
			return fmt.Errorf("failed to checkout branch from remote: %w", err)
		}
	}

	// STEP 4. Install dependencies
	fmt.Println("\n📋 STEP 4: Installing dependencies...")
	err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", fmt.Sprintf("cd %s && npm install", MetadataRepoName))
	if err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// STEP 5. Create metadata file
	fmt.Println("\n📋 STEP 5: Creating metadata file...")
	if t.deployConfig.L1ChainID != constants.EthereumSepoliaChainID { // Sepolia chain ID
		return fmt.Errorf("unsupported network. Currently only Sepolia (chain ID: 11155111) is supported, got chain ID: %d", t.deployConfig.L1ChainID)
	}
	networkDir := fmt.Sprintf("%s/data/sepolia", MetadataRepoName)

	metadataFileName := fmt.Sprintf("%s.json", systemConfigAddress)
	sourceFile := fmt.Sprintf("%s/schemas/example-rollup-metadata.json", MetadataRepoName)
	targetFile := fmt.Sprintf("%s/%s", networkDir, metadataFileName)

	if newMetadataEntry {
		fmt.Printf("Copying example metadata to %s...\n", targetFile)
		err = utils.ExecuteCommandStream(ctx, t.l, "cp", sourceFile, targetFile)
		if err != nil {
			return fmt.Errorf("failed to copy example metadata file: %w", err)
		}
	}

	fmt.Println("Updating metadata file with deployment information...")

	metadataBytes, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata types.RollupMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
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
		L1UsdcBridge:                 contracts.L1UsdcBridge,
		L2OutputOracle:               contracts.L2OutputOracleProxy,
		Mips:                         contracts.Mips,
		PermissionedDelayedWETH:      contracts.PermissionedDelayedWETHProxy,
		PreimageOracle:               contracts.PreimageOracle,
		ProtocolVersions:             contracts.ProtocolVersions,
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

	parsedTime, _ := time.Parse("2006-01-02 15:04:05 MST", t.deployConfig.StakingInfo.RegistrationTime)

	if t.deployConfig.StakingInfo != nil {
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

	metadata.Metadata = types.Metadata{
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

	// STEP 6. Validate metadata
	var prTitle string
	if newMetadataEntry {
		prTitle = fmt.Sprintf("[Rollup] sepolia %s - %s", systemConfigAddress, t.deployConfig.ChainName)
	} else {
		prTitle = fmt.Sprintf("[Update] sepolia %s - %s", systemConfigAddress, t.deployConfig.ChainName)
	}

	validationPath := fmt.Sprintf("data/sepolia/%s.json", systemConfigAddress)
	validationCmd := fmt.Sprintf("cd %s && npm run validate -- --pr-title \"%s\" %s", MetadataRepoName, prTitle, validationPath)

	fmt.Printf("Running validation command: %s\n", validationCmd)
	err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", validationCmd)
	if err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}
	fmt.Println("✅ Metadata file validated successfully!")

	// STEP 7. Setup git config and commit changes
	fmt.Println("\n📋 STEP 7: Committing changes...")
	err = setupGitConfig(ctx, t.l, MetadataRepoName, creds)
	if err != nil {
		return fmt.Errorf("failed to setup git config: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "add", ".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	commitMessage := fmt.Sprintf("[Rollup] %s sepolia %s - %s", operation, systemConfigAddress, t.deployConfig.ChainName)
	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "commit", "-m", commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	fmt.Println("✅ Changes committed successfully!")

	// STEP 8. Push changes to user's fork
	err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "push", "origin", branchName)
	if err != nil {
		return fmt.Errorf("failed to push changes to fork: %w", err)
	}
	fmt.Println("✅ Changes pushed to fork successfully!")

	// STEP 9. Create Pull Request from user's fork to original repo
	fmt.Println("\n📋 STEP 9: Creating Pull Request...")
	err = createGitHubPRFromFork(prTitle, branchName, creds.Username, creds.Token, newMetadataEntry)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	fmt.Println("\n🎉 Metadata registration process completed!")
	fmt.Printf("📋 Summary:\n")
	fmt.Printf("   ✅ Repository forked to %s/%s\n", creds.Username, MetadataRepoName)
	fmt.Printf("   ✅ Metadata file created: %s\n", targetFile)
	fmt.Printf("   ✅ Changes committed and pushed to fork\n")
	fmt.Printf("   ✅ Pull request created from fork to original repository\n")

	return nil
}
