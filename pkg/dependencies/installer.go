package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const stateDir = "/app/storage/.tool-install-state"

type InstallState struct {
	Version   string `json:"version"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func readState(tool string) *InstallState {
	data, err := os.ReadFile(filepath.Join(stateDir, tool+".json"))
	if err != nil {
		return nil
	}
	var state InstallState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

func writeState(tool string, state *InstallState) error {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state dir: %w", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(stateDir, tool+".json.tmp")
	finalFile := filepath.Join(stateDir, tool+".json")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, finalFile)
}

func checkBinaryVersion(ctx context.Context, binary, expectedVersion string, versionArgs ...string) bool {
	cmd := exec.CommandContext(ctx, binary, versionArgs...)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), expectedVersion)
}

func CheckDiskSpace(path string, minMB int) error {
	cmd := exec.Command("df", "-m", path)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("unexpected df output")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return fmt.Errorf("unexpected df output format")
	}
	available, err := strconv.Atoi(fields[3])
	if err != nil {
		return fmt.Errorf("failed to parse available space: %w", err)
	}
	if available < minMB {
		return fmt.Errorf("insufficient disk space: %dMB available, %dMB required", available, minMB)
	}
	return nil
}

func runShell(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func isToolReady(ctx context.Context, tool, binary, expectedVersion string, versionArgs ...string) bool {
	state := readState(tool)
	if state == nil || state.Status != "installed" || state.Version != expectedVersion {
		return false
	}
	return checkBinaryVersion(ctx, binary, expectedVersion, versionArgs...)
}

func InstallTerraform(ctx context.Context, logger *zap.SugaredLogger, arch string) error {
	tool := "terraform"
	if isToolReady(ctx, tool, "terraform", TerraformVersion, "--version") {
		logger.Infof("[tool-install] %s@%s: already installed, skipping", tool, TerraformVersion)
		return nil
	}

	logger.Infof("[tool-install] %s@%s: downloading...", tool, TerraformVersion)
	start := time.Now()

	url := TerraformDownloadURL(arch)
	script := fmt.Sprintf(`
		set -e
		cd /tmp
		curl -fsSL -o terraform.zip "%s"
		unzip -o terraform.zip -d /tmp/terraform-extract
		install -o root -g root -m 0755 /tmp/terraform-extract/terraform /usr/local/bin/terraform
		rm -rf terraform.zip /tmp/terraform-extract
	`, url)

	if _, err := runShell(ctx, script); err != nil {
		writeState(tool, &InstallState{Version: TerraformVersion, Status: "failed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return fmt.Errorf("terraform installation failed: %w", err)
	}

	if !checkBinaryVersion(ctx, "terraform", TerraformVersion, "--version") {
		return fmt.Errorf("terraform installed but version verification failed")
	}

	writeState(tool, &InstallState{Version: TerraformVersion, Status: "installed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	logger.Infof("[tool-install] %s@%s: installed successfully (%ds)", tool, TerraformVersion, int(time.Since(start).Seconds()))
	return nil
}

func InstallAwsCLI(ctx context.Context, logger *zap.SugaredLogger, arch string) error {
	tool := "aws-cli"
	if isToolReady(ctx, tool, "aws", AwsCLIVersion, "--version") {
		logger.Infof("[tool-install] %s@%s: already installed, skipping", tool, AwsCLIVersion)
		return nil
	}

	logger.Infof("[tool-install] %s@%s: downloading...", tool, AwsCLIVersion)
	start := time.Now()

	url := AwsCLIDownloadURL(arch)
	script := fmt.Sprintf(`
		set -e
		cd /tmp
		curl -fsSL -o awscliv2.zip "%s"
		unzip -o awscliv2.zip
		./aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli --update
		rm -rf aws awscliv2.zip
	`, url)

	if _, err := runShell(ctx, script); err != nil {
		writeState(tool, &InstallState{Version: AwsCLIVersion, Status: "failed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return fmt.Errorf("aws-cli installation failed: %w", err)
	}

	if !checkBinaryVersion(ctx, "aws", "aws-cli/"+AwsCLIVersion, "--version") {
		return fmt.Errorf("aws-cli installed but version verification failed")
	}

	writeState(tool, &InstallState{Version: AwsCLIVersion, Status: "installed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	logger.Infof("[tool-install] %s@%s: installed successfully (%ds)", tool, AwsCLIVersion, int(time.Since(start).Seconds()))
	return nil
}

func InstallKubectl(ctx context.Context, logger *zap.SugaredLogger, arch string) error {
	tool := "kubectl"
	if isToolReady(ctx, tool, "kubectl", KubectlVersion, "version", "--client") {
		logger.Infof("[tool-install] %s@%s: already installed, skipping", tool, KubectlVersion)
		return nil
	}

	logger.Infof("[tool-install] %s@%s: downloading...", tool, KubectlVersion)
	start := time.Now()

	binURL := KubectlDownloadURL(arch)
	shaURL := KubectlSha256URL(arch)
	script := fmt.Sprintf(`
		set -e
		cd /tmp
		curl -fsSL -o kubectl "%s"
		curl -fsSL -o kubectl.sha256 "%s"
		echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check
		install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
		rm -f kubectl kubectl.sha256
	`, binURL, shaURL)

	if _, err := runShell(ctx, script); err != nil {
		writeState(tool, &InstallState{Version: KubectlVersion, Status: "failed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return fmt.Errorf("kubectl installation failed: %w", err)
	}

	if !checkBinaryVersion(ctx, "kubectl", KubectlVersion, "version", "--client") {
		return fmt.Errorf("kubectl installed but version verification failed")
	}

	writeState(tool, &InstallState{Version: KubectlVersion, Status: "installed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	logger.Infof("[tool-install] %s@%s: installed successfully (%ds)", tool, KubectlVersion, int(time.Since(start).Seconds()))
	return nil
}

func InstallHelm(ctx context.Context, logger *zap.SugaredLogger, arch string) error {
	tool := "helm"
	if isToolReady(ctx, tool, "helm", HelmVersion, "version") {
		logger.Infof("[tool-install] %s@%s: already installed, skipping", tool, HelmVersion)
		return nil
	}

	logger.Infof("[tool-install] %s@%s: downloading...", tool, HelmVersion)
	start := time.Now()

	url := HelmDownloadURL(arch)
	script := fmt.Sprintf(`
		set -e
		cd /tmp
		curl -fsSL -o helm.tar.gz "%s"
		tar -xzf helm.tar.gz
		install -o root -g root -m 0755 linux-%s/helm /usr/local/bin/helm
		rm -rf helm.tar.gz linux-%s
	`, url, arch, arch)

	if _, err := runShell(ctx, script); err != nil {
		writeState(tool, &InstallState{Version: HelmVersion, Status: "failed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
		return fmt.Errorf("helm installation failed: %w", err)
	}

	if !checkBinaryVersion(ctx, "helm", HelmVersion, "version") {
		return fmt.Errorf("helm installed but version verification failed")
	}

	writeState(tool, &InstallState{Version: HelmVersion, Status: "installed", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	logger.Infof("[tool-install] %s@%s: installed successfully (%ds)", tool, HelmVersion, int(time.Since(start).Seconds()))
	return nil
}
