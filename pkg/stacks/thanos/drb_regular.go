package thanos

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

var drbRegularRequiredEnvKeys = []string{
	"LEADER_IP",
	"LEADER_PORT",
	"LEADER_PEER_ID",
	"LEADER_EOA",
	"PORT",
	"EOA_PRIVATE_KEY",
	"POSTGRES_PORT",
	"DRB_NODE_IMAGE",
	"CHAIN_ID",
	"ETH_RPC_URLS",
	"CONTRACT_ADDRESS",
}

func (t *ThanosStack) GetDRBRegularNodeInput(_ context.Context) (*types.DRBRegularNodeInput, error) {
	fmt.Println("\n--------------------------------")
	fmt.Println("DRB Regular Node Setup (EC2)")
	fmt.Println("--------------------------------")

	fmt.Print("Path to .env file: ")
	envPath, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file path: %w", err)
	}
	envPath = strings.TrimSpace(envPath)
	if envPath == "" {
		return nil, fmt.Errorf(".env file path cannot be empty")
	}
	if _, err := os.Stat(envPath); err != nil {
		return nil, fmt.Errorf("failed to access .env file: %w", err)
	}
	content, err := os.ReadFile(envPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}
	env, err := loadEnv(content, drbRegularRequiredEnvKeys)
	if err != nil {
		return nil, err
	}

	input := &types.DRBRegularNodeInput{
		NodeType:       "regular",
		EnvFilePath:    envPath,
		EnvFileContent: string(content),
	}
	leaderPort, err := envInt(env, "LEADER_PORT")
	if err != nil {
		return nil, err
	}
	nodePort, err := envInt(env, "PORT")
	if err != nil {
		return nil, err
	}
	postgresPort, err := envInt(env, "POSTGRES_PORT")
	if err != nil {
		return nil, err
	}
	input.LeaderIP = env["LEADER_IP"]
	input.LeaderPort = leaderPort
	input.LeaderPeerID = env["LEADER_PEER_ID"]
	input.LeaderEOA = strings.TrimSpace(env["LEADER_EOA"])
	input.NodePort = nodePort
	input.EOAPrivateKey = env["EOA_PRIVATE_KEY"]
	input.DrbNodeImage = env["DRB_NODE_IMAGE"]
	input.ChainID = strings.TrimSpace(env["CHAIN_ID"])
	input.EthRpcUrls = strings.TrimSpace(env["ETH_RPC_URLS"]) // comma-separated URLs
	input.ContractAddress = strings.TrimSpace(env["CONTRACT_ADDRESS"])

	fmt.Println("\n--------------------------------")
	fmt.Println("Database Selection for DRB Regular Node")
	fmt.Println("--------------------------------")
	fmt.Println("[1] AWS RDS PostgreSQL")
	fmt.Println("[2] Local PostgreSQL (Docker Compose on this EC2)")
	fmt.Print("Please select database type (1-2): ")
	dbOption, err := scanner.ScanInt()
	if err != nil {
		return nil, fmt.Errorf("failed to scan database option: %w", err)
	}

	dbConfig, err := buildDRBRegularDatabaseConfig(dbOption, postgresPort)
	if err != nil {
		return nil, err
	}
	input.DatabaseConfig = dbConfig

	fmt.Println("\n--------------------------------")
	fmt.Println("EC2 Key Pair")
	fmt.Println("--------------------------------")
	fmt.Print("EC2 key pair name (required for SSH): ")
	keyPairName, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read key pair name: %w", err)
	}
	keyPairName = strings.TrimSpace(keyPairName)
	if keyPairName == "" {
		return nil, fmt.Errorf("EC2 key pair name is required for SSH access")
	}
	input.KeyPairName = keyPairName

	if t.awsProfile == nil || t.awsProfile.AwsConfig == nil {
		return nil, fmt.Errorf("AWS credentials and region are required; run install from the plugin command (e.g. trh-sdk install drb regular-node)")
	}
	input.Region = t.awsProfile.AwsConfig.Region
	input.InstanceType = "t3.small"
	input.InstanceName = fmt.Sprintf("drb-regular-node-%d", time.Now().Unix())
	return input, nil
}

// buildDRBRegularDatabaseConfig prompts for password and returns DRBDatabaseConfig for the chosen option (1=RDS, 2=local).
func buildDRBRegularDatabaseConfig(dbOption int, postgresPort int) (*types.DRBDatabaseConfig, error) {
	var label string
	switch dbOption {
	case 1:
		label = "Database Password (for RDS)"
	case 2:
		label = "Database Password (for local Postgres container)"
	default:
		return nil, fmt.Errorf("invalid database option: %d. Please select 1 or 2", dbOption)
	}
	fmt.Println("\n--------------------------------")
	fmt.Println(label)
	fmt.Println("--------------------------------")
	dbPassword, err := scanner.ScanPasswordWithConfirmation()
	if err != nil {
		return nil, fmt.Errorf("failed to scan database password: %w", err)
	}
	if dbPassword == "" {
		return nil, fmt.Errorf("database password cannot be empty")
	}
	if dbOption == 1 {
		if !utils.IsValidRDSPassword(dbPassword) {
			return nil, fmt.Errorf("database password is invalid. RDS password must be 8-128 characters and cannot contain /, ', \", @, or spaces")
		}
		return &types.DRBDatabaseConfig{
			Type:         "rds",
			Username:     "postgres",
			Password:     dbPassword,
			DatabaseName: "drb",
		}, nil
	}
	return &types.DRBDatabaseConfig{
		Type:          "local",
		Username:      "postgres",
		DatabaseName:  "drb",
		Password:      dbPassword,
		ConnectionURL: fmt.Sprintf("postgres://postgres@regular-postgres:%d/drb", postgresPort),
	}, nil
}

func (t *ThanosStack) InstallDRBRegularNode(ctx context.Context, input *types.DRBRegularNodeInput) error {
	if input == nil {
		return fmt.Errorf("regular node input is required")
	}
	if t.awsProfile == nil || t.awsProfile.AwsConfig == nil {
		return fmt.Errorf("AWS profile is not initialized")
	}

	if err := utils.SwitchAWSRegion(ctx, input.Region); err != nil {
		return err
	}

	outputDir := filepath.Join(t.deploymentPath, "drb-regular")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git"); err != nil {
		return fmt.Errorf("failed to clone tokamak-thanos-stack: %w", err)
	}
	if err := utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos-stack && git fetch origin && git checkout feat/add-drb-node && git pull origin feat/add-drb-node", t.deploymentPath)); err != nil {
		return fmt.Errorf("failed to checkout feat/add-drb-node: %w", err)
	}

	vpcID := input.VpcID
	if vpcID == "" {
		var err error
		vpcID, err = getDefaultVPCID(ctx, input.Region)
		if err != nil {
			return err
		}
	}

	// If Type is "rds", deploy RDS on AWS
	if input.DatabaseConfig != nil && input.DatabaseConfig.Type == "rds" && input.DatabaseConfig.Password != "" {
		if t.deployConfig == nil {
			t.deployConfig = &types.Config{}
		}
		if t.deployConfig.AWS == nil {
			t.deployConfig.AWS = &types.AWSConfig{}
		}
		t.deployConfig.AWS.VpcID = vpcID
		t.deployConfig.AWS.Region = input.Region
		rdsURL, err := t.deployDRBDatabaseRDS(ctx, input.DatabaseConfig)
		if err != nil {
			return fmt.Errorf("failed to deploy RDS for regular node: %w", err)
		}
		input.DatabaseConfig.ConnectionURL = strings.Trim(rdsURL, `"`)
		host, port := hostPortFromPostgresURL(input.DatabaseConfig.ConnectionURL)
		t.logger.Infof("RDS endpoint set for regular node: %s:%d", host, port)
	}

	envContent, err := buildRegularNodeEnvContent(input)
	if err != nil {
		return err
	}
	envOutputPath := filepath.Join(outputDir, ".env")
	if err := os.WriteFile(envOutputPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	// Use DRBDatabaseConfig.Type: "local" => docker-compose.yaml (includes postgres), "rds" => docker-compose.standalone.yaml (app only).
	chartDir := filepath.Join(t.deploymentPath, "tokamak-thanos-stack", "charts", "drb-node")
	composeFileName := "docker-compose.standalone.yaml"
	if input.DatabaseConfig != nil && input.DatabaseConfig.Type == "local" {
		composeFileName = "docker-compose.yaml"
	}
	composePath := filepath.Join(chartDir, composeFileName)
	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to read compose file %s: %w", composePath, err)
	}
	composeDest := filepath.Join(outputDir, "docker-compose.yaml")
	if err := os.WriteFile(composeDest, composeContent, 0644); err != nil {
		return fmt.Errorf("failed to copy compose file to output: %w", err)
	}

	subnetID := input.SubnetID
	if subnetID == "" {
		subnetID, err = getDefaultSubnetID(ctx, input.Region, vpcID)
		if err != nil {
			return err
		}
	}

	amiID := input.AmiID
	if amiID == "" {
		amiID, err = getLatestUbuntuAmi(ctx, input.Region)
		if err != nil {
			return err
		}
	}

	if input.KeyPairName == "" {
		return fmt.Errorf("EC2 key pair name is required; provide it when running the install")
	}

	sgName := fmt.Sprintf("%s-sg", input.InstanceName)
	sgID, err := createRegularNodeSecurityGroup(ctx, input.Region, vpcID, sgName)
	if err != nil {
		return err
	}

	userDataPath := filepath.Join(outputDir, "drb-regular-user-data.sh")
	installDir := "/home/ubuntu/" + input.InstanceName
	userDataContent := buildRegularNodeUserData(installDir, envContent, string(composeContent))
	if err := os.WriteFile(userDataPath, []byte(userDataContent), 0644); err != nil {
		return fmt.Errorf("failed to write user-data script: %w", err)
	}

	userDataAbs, err := filepath.Abs(userDataPath)
	if err != nil {
		return fmt.Errorf("failed to resolve user-data path: %w", err)
	}

	instanceID, err := runRegularNodeInstance(ctx, input.Region, amiID, input.InstanceType, subnetID, sgID, input.KeyPairName, userDataAbs, input.InstanceName)
	if err != nil {
		return err
	}

	if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "wait", "instance-running", "--region", input.Region, "--instance-ids", instanceID); err != nil {
		return fmt.Errorf("failed to wait for instance to run: %w", err)
	}

	publicIP, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-instances",
		"--region", input.Region,
		"--instance-ids", instanceID,
		"--query", "Reservations[0].Instances[0].PublicIpAddress",
		"--output", "text",
	)
	if err != nil {
		return fmt.Errorf("failed to get instance public IP: %w", err)
	}

	t.logger.Infof("✅ Regular node instance created: %s", instanceID)
	t.logger.Infof("Public IP: %s", strings.TrimSpace(publicIP))
	t.logger.Infof("Local artifacts saved to: %s", outputDir)
	t.logger.Info("EC2 bootstrap will start Docker Compose automatically.")

	usedRDS := input.DatabaseConfig != nil && input.DatabaseConfig.Type == "rds"
	state := drbRegularNodeState{
		InstanceID:      instanceID,
		SecurityGroupID: sgID,
		Region:          input.Region,
		UsedRDS:         usedRDS,
	}
	statePath := filepath.Join(outputDir, "regular-node-state.json")
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal regular node state: %w", err)
	}
	if err := os.WriteFile(statePath, stateBytes, 0600); err != nil {
		return fmt.Errorf("failed to write regular node state: %w", err)
	}
	return nil
}

// drbRegularNodeState is persisted under drb-regular/regular-node-state.json for uninstall (internal use only)
type drbRegularNodeState struct {
	InstanceID      string `json:"instance_id"`
	SecurityGroupID string `json:"security_group_id"`
	Region          string `json:"region"`
	UsedRDS         bool   `json:"used_rds"` // true if user selected AWS RDS PostgreSQL
}

func (t *ThanosStack) UninstallDRBRegularNode(ctx context.Context) error {
	outputDir := filepath.Join(t.deploymentPath, "drb-regular")
	statePath := filepath.Join(outputDir, "regular-node-state.json")
	stateBytes, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			t.logger.Info("No regular node state found (nothing was installed from this deployment path).")
			return nil
		}
		return fmt.Errorf("failed to read regular node state: %w", err)
	}
	var state drbRegularNodeState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return fmt.Errorf("failed to parse regular node state: %w", err)
	}
	if state.InstanceID == "" || state.Region == "" {
		t.logger.Warn("Regular node state is incomplete; skipping uninstall.")
		return nil
	}

	t.logger.Info("Starting DRB regular node uninstall...")
	if err := utils.SwitchAWSRegion(ctx, state.Region); err != nil {
		return err
	}

	// Terminate the EC2 instance
	t.logger.Infof("Terminating instance %s...", state.InstanceID)
	if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "terminate-instances",
		"--region", state.Region,
		"--instance-ids", state.InstanceID,
	); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}
	if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "wait", "instance-terminated",
		"--region", state.Region,
		"--instance-ids", state.InstanceID,
	); err != nil {
		return fmt.Errorf("failed to wait for instance termination: %w", err)
	}
	t.logger.Infof("Instance %s terminated.", state.InstanceID)

	// Delete the security group (after instance is terminated)
	if state.SecurityGroupID != "" {
		t.logger.Infof("Deleting security group %s...", state.SecurityGroupID)
		if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-security-group",
			"--region", state.Region,
			"--group-id", state.SecurityGroupID,
		); err != nil {
			t.logger.Warnf("Failed to delete security group %s: %v (you may need to delete it manually)", state.SecurityGroupID, err)
		} else {
			t.logger.Infof("Security group %s deleted.", state.SecurityGroupID)
		}
	}

	// Destroy RDS (if user had selected AWS RDS PostgreSQL)
	if state.UsedRDS {
		drbTerraformPath := filepath.Join(t.deploymentPath, "tokamak-thanos-stack", "terraform", "drb")
		t.logger.Info("Destroying RDS (DRB terraform)...")
		if err := t.destroyTerraform(ctx, drbTerraformPath); err != nil {
			t.logger.Warnf("Failed to destroy DRB RDS terraform resources: %v (you may need to delete RDS manually)", err)
		} else {
			t.logger.Info("RDS (DRB terraform) destroyed.")
		}
	}

	// Remove state file so a future uninstall does not try to delete already-removed resources
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		t.logger.Warnf("Failed to remove state file: %v", err)
	}
	t.logger.Info("✅ DRB regular node uninstalled successfully.")
	t.logger.Infof("Local artifacts remain in %s (remove manually if desired).", outputDir)
	return nil
}

func buildRegularNodeEnvContent(input *types.DRBRegularNodeInput) (string, error) {
	if input.DrbNodeImage == "" {
		return "", fmt.Errorf("DRB node image is required")
	}
	if input.DatabaseConfig == nil || input.DatabaseConfig.ConnectionURL == "" {
		return "", fmt.Errorf("database config with ConnectionURL is required")
	}

	host, port := hostPortFromPostgresURL(input.DatabaseConfig.ConnectionURL)
	lines := []string{
		fmt.Sprintf("LEADER_IP=%s", input.LeaderIP),
		fmt.Sprintf("LEADER_PORT=%d", input.LeaderPort),
		fmt.Sprintf("LEADER_PEER_ID=%s", input.LeaderPeerID),
		fmt.Sprintf("LEADER_EOA=%s", input.LeaderEOA),
		fmt.Sprintf("PORT=%d", input.NodePort),
		fmt.Sprintf("EOA_PRIVATE_KEY=%s", input.EOAPrivateKey),
		"NODE_TYPE=regular",
		fmt.Sprintf("POSTGRES_HOST=%s", host),
		fmt.Sprintf("POSTGRES_PORT=%d", port),
		fmt.Sprintf("DRB_NODE_IMAGE=%s", input.DrbNodeImage),
		fmt.Sprintf("CHAIN_ID=%s", input.ChainID),
		fmt.Sprintf("ETH_RPC_URLS=%s", input.EthRpcUrls), // comma-separated RPC URLs
		fmt.Sprintf("CONTRACT_ADDRESS=%s", input.ContractAddress),
	}
	if input.DatabaseConfig.Password != "" {
		lines = append(lines, fmt.Sprintf("POSTGRES_PASSWORD=%s", input.DatabaseConfig.Password))
	}

	lines = append(lines,
		"OFF_CHAIN_SUBMISSION_PERIOD=40",
		"OFF_CHAIN_SUBMISSION_PERIOD_PER_OPERATOR=30",
		"ON_CHAIN_SUBMISSION_PERIOD=60",
		"ON_CHAIN_SUBMISSION_PERIOD_PER_OPERATOR=20",
		"REQUEST_OR_SUBMIT_OR_FAIL_DECISION_PERIOD=30",
	)
	return strings.Join(lines, "\n") + "\n", nil
}

func buildRegularNodeUserData(installDir, envContent, composeContent string) string {
	if installDir == "" {
		installDir = "/home/ubuntu/drb-regular"
	}
	logPath := "/var/log/drb-regular-bootstrap.log"
	var builder strings.Builder
	builder.WriteString("#!/bin/bash\n")
	builder.WriteString("set -euo pipefail\n")
	builder.WriteString(fmt.Sprintf("exec > >(tee -a %s) 2>&1\n", logPath))
	builder.WriteString("export DEBIAN_FRONTEND=noninteractive\n")
	builder.WriteString("apt-get update -y\n")
	builder.WriteString("apt-get install -y ca-certificates curl\n")
	builder.WriteString("install -m 0755 -d /etc/apt/keyrings\n")
	builder.WriteString("curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc\n")
	builder.WriteString("chmod a+r /etc/apt/keyrings/docker.asc\n")
	builder.WriteString("echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable\" | tee /etc/apt/sources.list.d/docker.list > /dev/null\n")
	builder.WriteString("apt-get update -y\n")
	builder.WriteString("apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin\n")
	builder.WriteString("systemctl enable --now docker\n")
	builder.WriteString("usermod -aG docker ubuntu\n")
	builder.WriteString(fmt.Sprintf("mkdir -p %s\n", installDir))
	builder.WriteString(fmt.Sprintf("cat <<'EOF_ENV' > %s/.env\n", installDir))
	builder.WriteString(envContent)
	builder.WriteString("EOF_ENV\n")
	builder.WriteString(fmt.Sprintf("cat <<'EOF_COMPOSE' > %s/docker-compose.yaml\n", installDir))
	builder.WriteString(composeContent)
	builder.WriteString("EOF_COMPOSE\n")
	builder.WriteString(fmt.Sprintf("chown -R ubuntu:ubuntu %s\n", installDir))
	builder.WriteString("echo 'Creating systemd unit to start DRB containers after Docker is ready...'\n")
	builder.WriteString("cat > /etc/systemd/system/drb-regular-start.service << EOF_UNIT\n")
	builder.WriteString("[Unit]\n")
	builder.WriteString("Description=Start DRB regular node containers\n")
	builder.WriteString("After=docker.service network-online.target\n")
	builder.WriteString("Requires=docker.service\n")
	builder.WriteString("\n")
	builder.WriteString("[Service]\n")
	builder.WriteString("Type=oneshot\n")
	builder.WriteString(fmt.Sprintf("ExecStart=/usr/bin/docker compose -f %s/docker-compose.yaml --env-file %s/.env up -d\n", installDir, installDir))
	builder.WriteString("RemainAfterExit=yes\n")
	builder.WriteString("\n")
	builder.WriteString("[Install]\n")
	builder.WriteString("WantedBy=multi-user.target\n")
	builder.WriteString("EOF_UNIT\n")
	builder.WriteString("systemctl daemon-reload\n")
	builder.WriteString("systemctl enable drb-regular-start.service\n")
	builder.WriteString("systemctl start drb-regular-start.service\n")
	return builder.String()
}

// loadEnv parses KEY=value lines from content into a map.
func loadEnv(content []byte, required []string) (map[string]string, error) {
	env := make(map[string]string)
	s := bufio.NewScanner(strings.NewReader(string(content)))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		env[k] = strings.Trim(strings.Trim(v, `"`), "'")
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse .env: %w", err)
	}
	for _, k := range required {
		if strings.TrimSpace(env[k]) == "" {
			return nil, fmt.Errorf(".env missing or empty: %s", k)
		}
	}
	return env, nil
}

func envInt(env map[string]string, key string) (int, error) {
	v := strings.TrimSpace(env[key])
	if v == "" {
		return 0, fmt.Errorf(".env missing or invalid %s", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf(".env invalid %s: %w", key, err)
	}
	return n, nil
}

func hostPortFromPostgresURL(dbURL string) (host string, port int) {
	dbURL = strings.Trim(dbURL, `"`)
	port = 5432
	if strings.Contains(dbURL, "@") {
		parts := strings.Split(dbURL, "@")
		if len(parts) == 2 {
			hostPort := strings.Split(parts[1], "/")[0]
			hp := strings.Split(hostPort, ":")
			host = hp[0]
			if len(hp) >= 2 && hp[1] != "" {
				if p, err := strconv.Atoi(hp[1]); err == nil && p > 0 {
					port = p
				}
			}
			return host, port
		}
	}
	return dbURL, port
}

func getDefaultVPCID(ctx context.Context, region string) (string, error) {
	vpcID, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-vpcs",
		"--region", region,
		"--filters", "Name=isDefault,Values=true",
		"--query", "Vpcs[0].VpcId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to get default VPC ID: %w", err)
	}
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" || vpcID == "None" {
		return "", fmt.Errorf("default VPC not found. Please provide a VPC ID.")
	}
	return vpcID, nil
}

func getDefaultSubnetID(ctx context.Context, region, vpcID string) (string, error) {
	subnetID, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-subnets",
		"--region", region,
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcID), "Name=default-for-az,Values=true",
		"--query", "Subnets[0].SubnetId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to get default subnet ID: %w", err)
	}
	subnetID = strings.TrimSpace(subnetID)
	if subnetID == "" || subnetID == "None" {
		subnetID, err = utils.ExecuteCommand(ctx, "aws", "ec2", "describe-subnets",
			"--region", region,
			"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcID),
			"--query", "Subnets[0].SubnetId",
			"--output", "text",
		)
		if err != nil {
			return "", fmt.Errorf("failed to get subnet ID: %w", err)
		}
		subnetID = strings.TrimSpace(subnetID)
	}
	if subnetID == "" || subnetID == "None" {
		return "", fmt.Errorf("no subnet found for VPC %s. Please provide a subnet ID.", vpcID)
	}
	return subnetID, nil
}

func getLatestUbuntuAmi(ctx context.Context, region string) (string, error) {
	amiID, err := utils.ExecuteCommand(ctx, "aws", "ssm", "get-parameter",
		"--region", region,
		"--name", "/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err == nil {
		amiID = strings.TrimSpace(amiID)
		if amiID != "" && amiID != "None" {
			return amiID, nil
		}
	}
	// Fallback: SSM parameter may be unavailable in some regions (e.g. ap-south-1); use describe-images.
	amiID, err = utils.ExecuteCommand(ctx, "aws", "ec2", "describe-images",
		"--region", region,
		"--owners", "099720109477",
		"--filters", "Name=name,Values=ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*", "Name=state,Values=available",
		"--query", "sort_by(Images,&CreationDate)[-1].ImageId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest Ubuntu 22.04 AMI (tried SSM and describe-images): %w", err)
	}
	amiID = strings.TrimSpace(amiID)
	if amiID == "" || amiID == "None" {
		return "", fmt.Errorf("latest Ubuntu 22.04 AMI not found in region %s", region)
	}
	return amiID, nil
}

func createRegularNodeSecurityGroup(ctx context.Context, region, vpcID, name string) (string, error) {
	sgID, err := utils.ExecuteCommand(ctx, "aws", "ec2", "create-security-group",
		"--region", region,
		"--group-name", name,
		"--description", "DRB regular node security group",
		"--vpc-id", vpcID,
		"--tag-specifications", fmt.Sprintf("ResourceType=security-group,Tags=[{Key=Name,Value=%s}]", name),
		"--query", "GroupId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %w", err)
	}
	sgID = strings.TrimSpace(sgID)
	if sgID == "" || sgID == "None" {
		return "", fmt.Errorf("security group ID is empty")
	}
	_, err = utils.ExecuteCommand(ctx, "aws", "ec2", "authorize-security-group-ingress",
		"--region", region,
		"--group-id", sgID,
		"--protocol", "tcp",
		"--port", "22",
		"--cidr", "0.0.0.0/0",
	)
	if err != nil {
		return "", fmt.Errorf("failed to add SSH inbound rule to security group: %w", err)
	}
	return sgID, nil
}

func runRegularNodeInstance(ctx context.Context, region, amiID, instanceType, subnetID, sgID, keyPairName, userDataPath, instanceName string) (string, error) {
	networkInterface := fmt.Sprintf("DeviceIndex=0,SubnetId=%s,Groups=%s,AssociatePublicIpAddress=true", subnetID, sgID)
	instanceID, err := utils.ExecuteCommand(ctx, "aws", "ec2", "run-instances",
		"--region", region,
		"--image-id", amiID,
		"--instance-type", instanceType,
		"--key-name", keyPairName,
		"--network-interfaces", networkInterface,
		"--user-data", fmt.Sprintf("file://%s", userDataPath),
		"--tag-specifications", fmt.Sprintf("ResourceType=instance,Tags=[{Key=Name,Value=%s}]", instanceName),
		"--query", "Instances[0].InstanceId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to run EC2 instance: %w", err)
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || instanceID == "None" {
		return "", fmt.Errorf("instance ID is empty")
	}
	return instanceID, nil
}
