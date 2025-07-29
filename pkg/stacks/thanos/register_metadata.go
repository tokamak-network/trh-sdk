package thanos

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
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
	Logo        string `json:"logo,omitempty"`
	Website     string `json:"website,omitempty"`

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
	Name          string `json:"name"`
	Version       string `json:"version"`
	Commit        string `json:"commit,omitempty"`        // Optional
	Documentation string `json:"documentation,omitempty"` // Optional
}

type NativeToken struct {
	Type        string `json:"type"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    int    `json:"decimals"`
	L1Address   string `json:"l1Address"`
	LogoUrl     string `json:"logoUrl,omitempty"`     // Optional
	CoingeckoId string `json:"coingeckoId,omitempty"` // Optional
}

// Updated L1Contracts to match the raw data structure
type L1Contracts struct {
	ProxyAdmin                   string `json:"ProxyAdmin"`
	SystemConfig                 string `json:"SystemConfig"`
	AddressManager               string `json:"AddressManager"`
	SuperchainConfig             string `json:"SuperchainConfig"`
	DisputeGameFactory           string `json:"DisputeGameFactory"`
	L1CrossDomainMessenger       string `json:"L1CrossDomainMessenger"`
	L1ERC721Bridge               string `json:"L1ERC721Bridge"`
	L1StandardBridge             string `json:"L1StandardBridge"`
	OptimismMintableERC20Factory string `json:"OptimismMintableERC20Factory"`
	OptimismPortal               string `json:"OptimismPortal"`
	AnchorStateRegistry          string `json:"AnchorStateRegistry"`
	DelayedWETH                  string `json:"DelayedWETH"`
	L1UsdcBridge                 string `json:"L1UsdcBridge,omitempty"`
	L2OutputOracle               string `json:"L2OutputOracle"`
	Mips                         string `json:"Mips"`
	PermissionedDelayedWETH      string `json:"PermissionedDelayedWETH"`
	PreimageOracle               string `json:"PreimageOracle"`
	ProtocolVersions             string `json:"ProtocolVersions"`
	SafeProxyFactory             string `json:"SafeProxyFactory"`
	SafeSingleton                string `json:"SafeSingleton"`
	SystemOwnerSafe              string `json:"SystemOwnerSafe"`
}

type L2Contracts struct {
	NativeToken                   string `json:"NativeToken"`
	WETH                          string `json:"WETH"`
	L2ToL1MessagePasser           string `json:"L2ToL1MessagePasser"`
	DeployerWhitelist             string `json:"DeployerWhitelist"`
	L2CrossDomainMessenger        string `json:"L2CrossDomainMessenger"`
	GasPriceOracle                string `json:"GasPriceOracle"`
	L2StandardBridge              string `json:"L2StandardBridge"`
	SequencerFeeVault             string `json:"SequencerFeeVault"`
	OptimismMintableERC20Factory  string `json:"OptimismMintableERC20Factory"`
	L1BlockNumber                 string `json:"L1BlockNumber"`
	L1Block                       string `json:"L1Block"`
	GovernanceToken               string `json:"GovernanceToken"`
	LegacyMessagePasser           string `json:"LegacyMessagePasser"`
	L2ERC721Bridge                string `json:"L2ERC721Bridge"`
	OptimismMintableERC721Factory string `json:"OptimismMintableERC721Factory"`
	ProxyAdmin                    string `json:"ProxyAdmin"`
	BaseFeeVault                  string `json:"BaseFeeVault"`
	L1FeeVault                    string `json:"L1FeeVault"`
	ETH                           string `json:"ETH"`
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
	ApiUrl string `json:"apiUrl,omitempty"` // Optional
}

type Sequencer struct {
	Address           string `json:"address"`
	BatcherAddress    string `json:"batcherAddress"`
	ProposerAddress   string `json:"proposerAddress"`
	AggregatorAddress string `json:"aggregatorAddress,omitempty"` // Optional
	TrustedSequencer  bool   `json:"trustedSequencer,omitempty"`  // Optional
}

type Staking struct {
	IsCandidate           bool   `json:"isCandidate"`
	CandidateRegisteredAt string `json:"candidateRegisteredAt,omitempty"` // Optional
	CandidateStatus       string `json:"candidateStatus,omitempty"`       // Optional
	RegistrationTxHash    string `json:"registrationTxHash,omitempty"`    // Optional
	CandidateAddress      string `json:"candidateAddress,omitempty"`      // Optional
	RollupConfigAddress   string `json:"rollupConfigAddress,omitempty"`   // Optional
	StakingServiceName    string `json:"stakingServiceName,omitempty"`    // Optional
}

type NetworkConfig struct {
	BlockTime         int    `json:"blockTime"`
	GasLimit          string `json:"gasLimit"`
	BaseFeePerGas     string `json:"baseFeePerGas"`
	PriorityFeePerGas string `json:"priorityFeePerGas"`
}

// Updated WithdrawalConfig to match the raw data structure
type WithdrawalConfig struct {
	ChallengePeriod          int            `json:"challengePeriod"`
	ExpectedWithdrawalDelay  int            `json:"expectedWithdrawalDelay"`
	MonitoringInfo           MonitoringInfo `json:"monitoringInfo"`
	BatchSubmissionFrequency int            `json:"batchSubmissionFrequency"`
	OutputRootFrequency      int            `json:"outputRootFrequency"`
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

// GitHubPR represents the structure for creating a GitHub PR
type GitHubPR struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// GitHubCredentials holds GitHub authentication details
type GitHubCredentials struct {
	Username string
	Token    string
	Email    string
}

// executeGitPushWithCredentials handles git push with authentication
func executeGitPushWithCredentials(workingDir, branchName string, creds *GitHubCredentials) error {
	if creds == nil {
		// Use regular git push for SSH
		fmt.Printf("Executing: git push origin %s\n", branchName)
		cmd := exec.Command("git", "push", "origin", branchName)
		cmd.Dir = workingDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()

		return cmd.Run()
	}

	// Use authenticated URL for HTTPS
	authenticatedURL := fmt.Sprintf("https://%s:%s@github.com/tokamak-network/tokamak-rollup-metadata-repository.git",
		creds.Username, creds.Token)

	fmt.Printf("Executing: git push %s %s\n", "[authenticated-url]", branchName)
	fmt.Println("🔐 Using provided GitHub credentials for authentication")

	cmd := exec.Command("git", "push", authenticatedURL, branchName)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}

// getGitHubCredentials prompts user for GitHub username and personal access token
func getGitHubCredentials() (*GitHubCredentials, error) {
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

	return &GitHubCredentials{
		Username: strings.TrimSpace(username),
		Token:    strings.TrimSpace(token),
		Email:    strings.TrimSpace(email),
	}, nil
}

// pushChangesInteractive attempts to push changes with proper user interaction
func pushChangesInteractive(repoPath, branchName string) error {
	fmt.Println("\n🚀 Preparing to push changes...")

	// Check git remote and auth setup, get credentials if needed
	creds, err := getGitHubCredentials()
	if err != nil {
		return err
	}

	// Ask user to confirm push
	fmt.Print("\n📤 Ready to push changes. Continue? (Y/n): ")
	confirmation, err := scanner.ScanBool(true)
	if err != nil {
		return fmt.Errorf("error reading confirmation: %w", err)
	}

	if !confirmation {
		return fmt.Errorf("push cancelled by user")
	}

	// Perform the push with interactive capability
	fmt.Println("\n🔄 Pushing changes...")
	if creds == nil {
		fmt.Println("(Using SSH authentication)")
	} else {
		fmt.Printf("(Using HTTPS authentication for user: %s)\n", creds.Username)
	}

	// Try to push, with option to retry credentials if authentication fails
	var pushAttempts int
	maxAttempts := 2

	for pushAttempts < maxAttempts {
		err = executeGitPushWithCredentials(repoPath, branchName, creds)

		if err == nil {
			break // Success!
		}

		// Check if it's an authentication error and we have credentials to retry
		errStr := err.Error()
		isAuthError := strings.Contains(errStr, "403") || strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "invalid username or password")

		if isAuthError && creds != nil && pushAttempts < maxAttempts-1 {
			fmt.Println("\n🔑 Authentication failed. Let's try entering credentials again...")

			fmt.Print("Would you like to re-enter your GitHub credentials? (Y/n): ")
			retry, retryErr := scanner.ScanBool(true)
			if retryErr != nil || !retry {
				break
			}

			// Get new credentials
			newCreds, credErr := getGitHubCredentials()
			if credErr != nil {
				fmt.Printf("Error getting credentials: %v\n", credErr)
				break
			}

			creds = newCreds
			fmt.Printf("🔄 Retrying push with new credentials for user: %s\n", creds.Username)
		} else {
			break // Not an auth error or no more attempts
		}

		pushAttempts++
	}
	if err != nil {
		fmt.Println("\n❌ Push failed!")
		return fmt.Errorf("failed to push changes: %w", err)
	}

	fmt.Println("✅ Changes pushed successfully!")
	return nil
}

// setupGitConfig ensures git config is set for the repository
func setupGitConfig(ctx context.Context, l *zap.SugaredLogger, repoPath string, creds *GitHubCredentials) error {
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

// createGitHubPR creates a pull request using GitHub API
func createGitHubPR(systemConfigAddress, chainName, branchName string) error {
	fmt.Println("\n🔗 Creating Pull Request...")

	// Ask user for GitHub token
	var githubToken string
	fmt.Println("\n📝 To create a PR automatically, we need a GitHub Personal Access Token.")
	fmt.Println("   Create one at: https://github.com/settings/tokens/new")
	fmt.Println("   Required scopes: repo (or public_repo for public repos)")
	fmt.Print("\nEnter your GitHub token (or press Enter to skip PR creation): ")

	token, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("error reading GitHub token: %w", err)
	}

	githubToken = strings.TrimSpace(token)
	if githubToken == "" {
		fmt.Println("\n⏭️  Skipping automatic PR creation.")
		fmt.Printf("\n📋 To create the PR manually:\n")
		fmt.Printf("   1. Go to: https://github.com/tokamak-network/tokamak-rollup-metadata-repository\n")
		fmt.Printf("   2. Click 'Compare & pull request' for branch: %s\n", branchName)
		fmt.Printf("   3. Use this title: [Rollup] sepolia %s - %s\n", systemConfigAddress, chainName)
		fmt.Printf("   4. Add description and create the PR\n\n")
		return nil
	}

	// Create PR using GitHub API
	prTitle := fmt.Sprintf("[Rollup] sepolia %s - %s", systemConfigAddress, chainName)

	prData := GitHubPR{
		Title: prTitle,
		Head:  branchName,
		Base:  "main",
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

	req.Header.Set("Authorization", "token "+githubToken)
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

// createGitHubPRFromFork creates a pull request from a forked repository using GitHub API
func createGitHubPRFromFork(systemConfigAddress, chainName, branchName, username, token string) error {
	fmt.Println("\n🔗 Creating Pull Request from fork...")

	// Create PR using GitHub API
	prTitle := fmt.Sprintf("[Rollup] sepolia %s - %s", systemConfigAddress, chainName)

	prData := GitHubPR{
		Title: prTitle,
		Head:  fmt.Sprintf("%s:%s", username, branchName),
		Base:  "tokamak-network:main",
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

	cwd, err := os.Getwd()
	fmt.Println("Current working directory:", cwd)
	if err != nil {
		fmt.Println("Error determining current directory:", err)
		return err
	}

	file, err := os.Open(fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", cwd, fmt.Sprintf("%d-deploy.json", t.deployConfig.L1ChainID)))
	if err != nil {
		fmt.Println("Error opening deployment file:", err)
		return err
	}
	defer file.Close()

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
	// STEP 1. Fork the repository
	err = t.forkRepository(creds.Username, creds.Token, MetadataRepoName)
	if err != nil {
		return fmt.Errorf("failed to fork repository: %w", err)
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
	currentBranch, err := utils.ExecuteCommand(ctx, "git", "-C", MetadataRepoName, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	fmt.Printf("Current branch: %s\n", currentBranch)

	if branchName == strings.TrimSpace(currentBranch) {
		fmt.Println("✅ Branch already exists!")
	} else {
		fmt.Printf("Creating and checking out branch: %s\n", branchName)
		err = utils.ExecuteCommandStream(ctx, t.l, "git", "-C", MetadataRepoName, "checkout", "-b", branchName)
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch: %w", err)
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
	if t.deployConfig.L1ChainID != 11155111 { // Sepolia chain ID
		return fmt.Errorf("unsupported network. Currently only Sepolia (chain ID: 11155111) is supported, got chain ID: %d", t.deployConfig.L1ChainID)
	}
	networkDir := fmt.Sprintf("%s/data/sepolia", MetadataRepoName)

	metadataFileName := fmt.Sprintf("%s.json", systemConfigAddress)
	sourceFile := fmt.Sprintf("%s/schemas/example-rollup-metadata.json", MetadataRepoName)
	targetFile := fmt.Sprintf("%s/%s", networkDir, metadataFileName)

	fmt.Printf("Copying example metadata to %s...\n", targetFile)
	err = utils.ExecuteCommandStream(ctx, t.l, "cp", sourceFile, targetFile)
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

	metadata.L1ChainId = t.deployConfig.L1ChainID
	metadata.L2ChainId = t.deployConfig.L2ChainID
	metadata.Name = t.deployConfig.ChainName

	info, ok := debug.ReadBuildInfo()
	if ok {
		version := info.Main.Version
		metadata.Stack.Version = version
	}

	timestamp := time.Now().UTC()
	metadata.CreatedAt = timestamp.Format(time.RFC3339)
	metadata.LastUpdated = timestamp.Format(time.RFC3339)

	metadata.Status = "active"

	metadata.RpcUrl = t.deployConfig.L2RpcUrl
	if strings.HasPrefix(t.deployConfig.L2RpcUrl, "https://") {
		metadata.WsUrl = strings.Replace(t.deployConfig.L2RpcUrl, "https://", "wss://", 1)
	} else if strings.HasPrefix(t.deployConfig.L2RpcUrl, "http://") {
		metadata.WsUrl = strings.Replace(t.deployConfig.L2RpcUrl, "http://", "ws://", 1)
	}

	metadata.L1Contracts = L1Contracts{
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

	metadata.L2Contracts = L2Contracts{
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
		metadata.Bridges = []Bridge{
			{
				Name:            metadata.Bridges[0].Name,
				Type:            metadata.Bridges[0].Type,
				URL:             stackInfo.BridgeUrl,
				Status:          "active",
				SupportedTokens: metadata.Bridges[0].SupportedTokens,
			},
		}
	} else {
		metadata.Bridges = []Bridge{
			{
				Name:            metadata.Bridges[0].Name,
				Type:            metadata.Bridges[0].Type,
				URL:             metadata.Bridges[0].URL,
				Status:          metadata.Bridges[0].Status,
				SupportedTokens: metadata.Bridges[0].SupportedTokens,
			},
		}
	}

	if stackInfo.BlockExplorer != "" {
		metadata.Explorers = []Explorer{
			{
				Name:   metadata.Explorers[0].Name,
				URL:    stackInfo.BlockExplorer,
				Type:   metadata.Explorers[0].Type,
				Status: "active",
			},
		}
	} else {
		metadata.Explorers = []Explorer{
			{
				Name:   metadata.Explorers[0].Name,
				URL:    metadata.Explorers[0].URL,
				Type:   metadata.Explorers[0].Type,
				Status: metadata.Explorers[0].Status,
			},
		}
	}

	parsedTime, _ := time.Parse("2006-01-02 15:04:05 MST", t.deployConfig.StakingInfo.RegistrationTime)

	if t.deployConfig.StakingInfo != nil {
		metadata.Staking = Staking{
			IsCandidate:           t.deployConfig.StakingInfo.IsCandidate,
			CandidateRegisteredAt: parsedTime.UTC().Format(time.RFC3339),
			CandidateStatus:       "active",
			RegistrationTxHash:    t.deployConfig.StakingInfo.RegistrationTxHash,
			CandidateAddress:      t.deployConfig.StakingInfo.CandidateAddress,
		}
	} else {
		metadata.Staking = Staking{
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

	metadata.Sequencer = Sequencer{
		Address:         sequencerAddress.String(),
		BatcherAddress:  batcherAddress.String(),
		ProposerAddress: proposerAddress.String(),
	}

	metadata.NetworkConfig = NetworkConfig{
		BlockTime:         int(t.deployConfig.ChainConfiguration.L2BlockTime),
		GasLimit:          "0x1c9c380",
		BaseFeePerGas:     "0x3b9aca00",
		PriorityFeePerGas: metadata.NetworkConfig.PriorityFeePerGas,
	}

	metadata.WithdrawalConfig = WithdrawalConfig{
		ChallengePeriod:         int(t.deployConfig.ChainConfiguration.ChallengePeriod),
		ExpectedWithdrawalDelay: max(int(t.deployConfig.ChainConfiguration.BatchSubmissionFrequency), int(t.deployConfig.ChainConfiguration.OutputRootFrequency)) + int(t.deployConfig.ChainConfiguration.ChallengePeriod),
		MonitoringInfo: MonitoringInfo{
			L2OutputOracleAddress:    contracts.L2OutputOracleProxy,
			OutputProposedEventTopic: metadata.WithdrawalConfig.MonitoringInfo.OutputProposedEventTopic,
		},
		BatchSubmissionFrequency: int(t.deployConfig.ChainConfiguration.BatchSubmissionFrequency),
		OutputRootFrequency:      int(t.deployConfig.ChainConfiguration.OutputRootFrequency),
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

	// STEP 6. Validate metadata
	validationPath := fmt.Sprintf("data/sepolia/%s.json", systemConfigAddress)
	validationCmd := fmt.Sprintf("cd %s && npm run validate %s", MetadataRepoName, validationPath)

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

	commitMessage := fmt.Sprintf("[Rollup] sepolia %s - %s", systemConfigAddress, t.deployConfig.ChainName)
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
	err = createGitHubPRFromFork(systemConfigAddress, t.deployConfig.ChainName, branchName, creds.Username, creds.Token)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to create PR automatically: %v\n", err)
		fmt.Printf("\n📋 You can create the PR manually:\n")
		fmt.Printf("   1. Go to: https://github.com/%s/tokamak-rollup-metadata-repository\n", creds.Username)
		fmt.Printf("   2. Click 'Compare & pull request' for branch: %s\n", branchName)
		fmt.Printf("   3. Use this title: [Rollup] sepolia %s - %s\n", systemConfigAddress, t.deployConfig.ChainName)
		fmt.Printf("   4. Set base to: tokamak-network:main\n")
		fmt.Printf("   5. Add description and create the PR\n")
	}

	fmt.Println("\n🎉 Metadata registration process completed!")
	fmt.Printf("📋 Summary:\n")
	fmt.Printf("   ✅ Repository forked to %s/%s\n", creds.Username, MetadataRepoName)
	fmt.Printf("   ✅ Metadata file created: %s\n", targetFile)
	fmt.Printf("   ✅ Changes committed and pushed to fork\n")
	fmt.Printf("   ✅ Pull request created from fork to original repository\n")

	return nil
}
