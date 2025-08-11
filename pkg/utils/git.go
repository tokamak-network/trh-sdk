package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

const gitHubAPIBaseURL string = "https://api.github.com/repos"

func CloneRepo(ctx context.Context, l *zap.SugaredLogger, deploymentPath string, url string, folderName string) error {
	var clonePath string
	if filepath.IsAbs(deploymentPath) {
		clonePath = filepath.Join(deploymentPath, folderName)
	} else {
		clonePath = filepath.Join(".", deploymentPath, folderName)
	}

	// Check if the target directory already exists
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists", clonePath)
	}

	return ExecuteCommandStream(ctx, l, "git", "clone", url, clonePath)
}

func PullLatestCode(ctx context.Context, l *zap.SugaredLogger, deploymentPath string, folderName string) error {
	var clonePath string
	if filepath.IsAbs(deploymentPath) {
		clonePath = filepath.Join(deploymentPath, folderName)
	} else {
		clonePath = filepath.Join(".", deploymentPath, folderName)
	}

	// Check if the target directory exists
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		return fmt.Errorf("path '%s' does not exist", clonePath)
	}

	// Save the current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to the target directory
	if err := os.Chdir(clonePath); err != nil {
		return fmt.Errorf("failed to change directory to '%s': %v", clonePath, err)
	}

	// Execute the git pull command
	return ExecuteCommandStream(ctx, l, "git", "pull")
}

func CheckIfForkExists(username, token, repoName string) (bool, error) {
	url := fmt.Sprintf("%s/%s/%s", gitHubAPIBaseURL, username, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

func ForkRepository(username, token, repoName string) error {
	fmt.Printf("🍴 Creating fork of %s in %s's account...\n", repoName, username)

	// GitHub API endpoint to create a fork
	url := fmt.Sprintf("%s/tokamak-network/%s/forks", gitHubAPIBaseURL, repoName)

	// Create empty body for fork request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create fork request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create fork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 202 {
		fmt.Println("✅ Fork created successfully!")
		// Wait a moment for the fork to be ready
		time.Sleep(3 * time.Second)
		return nil
	} else if resp.StatusCode == 200 {
		fmt.Println("✅ Fork already exists!")
		return nil
	} else {
		return fmt.Errorf("failed to create fork, status code: %d", resp.StatusCode)
	}
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
func CreateGitHubPRFromFork(prTitle, branchName, username, token, repoName string, newMetadataEntry bool) error {
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
	url := fmt.Sprintf("%s/tokamak-network/%s/pulls", gitHubAPIBaseURL, repoName)
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

// setupGitConfig ensures git config is set for the repository
func SetupGitConfig(ctx context.Context, l *zap.SugaredLogger, repoPath string, creds *types.GitHubCredentials) error {
	// Check if git config is already set (either globally or locally)
	userName, err := ExecuteCommand(ctx, "git", "-C", repoPath, "config", "user.name")
	if err == nil && strings.TrimSpace(userName) != "" {
		fmt.Printf("✅ Git user.name already configured: %s\n", strings.TrimSpace(userName))

		userEmail, err := ExecuteCommand(ctx, "git", "-C", repoPath, "config", "user.email")
		if err == nil && strings.TrimSpace(userEmail) != "" {
			fmt.Printf("✅ Git user.email already configured: %s\n", strings.TrimSpace(userEmail))
			return nil
		}
	}

	fmt.Println("🔧 Git config not set. Please provide your git credentials for this repository:")

	// Set local git config for this repository
	fmt.Println("Setting git config for this repository...")

	err = ExecuteCommandStream(ctx, l, "git", "-C", repoPath, "config", "user.name", creds.Username)
	if err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	err = ExecuteCommandStream(ctx, l, "git", "-C", repoPath, "config", "user.email", creds.Email)
	if err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	fmt.Printf("✅ Git config set successfully!\n")
	fmt.Printf("   - user.name: %s\n", creds.Username)
	fmt.Printf("   - user.email: %s\n", creds.Email)

	return nil
}
