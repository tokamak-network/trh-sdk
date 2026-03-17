package thanos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, infraOpt string, inputs *DeployInfraInput) error {
	switch t.network {
	case constants.LocalDevnet:
		err := t.deployLocalDevnet(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			t.logger.Error("Failed to deploy the devnet", "err", err)

			if destroyErr := t.destroyDevnet(ctx); destroyErr != nil {
				t.logger.Error("Failed to destroy the devnet chain after deploying the chain failed", "err", destroyErr)
			}
			return err
		}
		return nil
	case constants.Testnet, constants.Mainnet:
		switch infraOpt {
		case constants.AWS:
			err := t.deployNetworkToAWS(ctx, inputs)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					t.logger.Warn("Deployment canceled")
					return err
				}
				t.logger.Error("Failed to deploy the testnet chain", "err", err)

				if destroyErr := t.destroyInfraOnAWS(ctx); destroyErr != nil {
					t.logger.Error("Failed to destroy the testnet chain after deploying the chain failed", "err", destroyErr)
				}

				return err
			}

			if inputs.GithubCredentials != nil && inputs.MetadataInfo != nil {
				_, err = t.RegisterMetadata(ctx, inputs.GithubCredentials, inputs.MetadataInfo)
				if err != nil {
					t.logger.Error("Failed to register metadata", "err", err)
					return err
				}
			}
			return nil
		default:
			t.logger.Error("infrastructure provider %s is not supported", infraOpt)
			return fmt.Errorf("infrastructure provider %s is not supported", infraOpt)
		}
	default:
		t.logger.Error("network %s is not supported", t.network)
		return fmt.Errorf("network %s is not supported", t.network)
	}

}

func (t *ThanosStack) deployLocalDevnet(ctx context.Context) error {
	err := t.cloneSourcecode(ctx, "tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	// Start the devnet
	t.logger.Info("Starting the devnet...")

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos && export DEVNET_L2OO=true && make devnet-up", t.deploymentPath))
	if err != nil {
		t.logger.Error("❌ Failed to start devnet!")
		return err
	}

	t.logger.Info("✅ Devnet started successfully!")

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(ctx context.Context, inputs *DeployInfraInput) error {
	shellConfigFile := utils.GetShellConfigDefault()

	if inputs == nil {
		return fmt.Errorf("inputs is required")
	}

	if t.deployConfig.EnableFraudProof && t.deployConfig.ChallengerPrivateKey == "" {
		return fmt.Errorf("fault proof is enabled but challenger private key is not set; re-run deploy-contracts with --enable-fault-proof")
	}

	// Check dependencies
	// STEP 1. Verify required dependencies
	if !dependencies.CheckTerraformInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil
	}

	if !dependencies.CheckHelmInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil
	}

	if !dependencies.CheckAwsCLIInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil
	}

	if !dependencies.CheckK8sInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil
	}

	if err := inputs.Validate(ctx); err != nil {
		t.logger.Error("Error validating inputs", "err", err)
		return err
	}

	// Check if the contracts deployed successfully
	if t.deployConfig.DeployContractState.Status != types.DeployContractStatusCompleted {
		return fmt.Errorf("contracts are not deployed successfully, please deploy the contracts first")
	}

	// STEP 1. Clone the charts repository
	err := t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		t.logger.Error("Error cloning repository", "err", err)
		return err
	}

	// STEP 2. AWS Authentication
	if t.awsProfile == nil {
		return fmt.Errorf("AWS configuration is not set")
	}
	awsAccountProfile := t.awsProfile.AccountProfile
	awsLoginInputs := t.awsProfile.AwsConfig

	t.deployConfig.AWS = awsLoginInputs
	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		t.logger.Error("failed to write settings file", "err", err)
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	t.logger.Info("⚡️Removing the previous deployment state...")
	err = t.clearTerraformState(ctx)
	if err != nil {
		t.logger.Error("Failed to clear the existing terraform state", "err", err)
		return err
	}

	t.logger.Info("✅ Removed the previous deployment state...")

	var (
		chainConfiguration = t.deployConfig.ChainConfiguration
	)

	if chainConfiguration == nil {
		t.logger.Error("chain configuration is not set")
		return fmt.Errorf("chain configuration is not set")
	}

	// STEP 3. Create .envrc file
	// Read prestate hash from the cannon prestate file built by start-deploy.sh build.
	// When EnableFraudProof is set and the file is missing (e.g., re-deploy after env reset),
	// automatically build it by running 'make cannon-prestate' in tokamak-thanos.
	prestateSrc := fmt.Sprintf("%s/tokamak-thanos/op-program/bin/prestate.json", t.deploymentPath)
	if t.deployConfig.EnableFraudProof {
		if _, statErr := os.Stat(prestateSrc); os.IsNotExist(statErr) {
			t.logger.Info("Cannon prestate not found, building...", "path", prestateSrc)
			if cloneErr := t.cloneSourcecode(ctx, "tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git"); cloneErr != nil {
				t.logger.Error("Failed to clone tokamak-thanos for cannon build", "err", cloneErr)
				return fmt.Errorf("failed to clone tokamak-thanos for cannon build: %w", cloneErr)
			}
			tokamakThanosDir := fmt.Sprintf("%s/tokamak-thanos", t.deploymentPath)
			if buildErr := buildCannonPrestate(ctx, t.logger, tokamakThanosDir); buildErr != nil {
				t.logger.Error("Failed to build cannon prestate", "err", buildErr,
					"hint", "Ensure Go toolchain and build tools are installed")
				return fmt.Errorf("failed to build cannon prestate: %w", buildErr)
			}
			t.logger.Info("✅ Cannon prestate built successfully")
		}
	}
	prestateHash, err := readPrestateHash(prestateSrc)
	if err != nil {
		t.logger.Error("Failed to read cannon prestate hash",
			"err", err,
			"path", prestateSrc,
			"hint", "Ensure start-deploy.sh build completed successfully before deploying chain")
		return err
	}
	t.logger.Info("Cannon prestate hash loaded", "hash", prestateHash)

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	feeTokenConfig := constants.GetFeeTokenConfig(t.deployConfig.FeeToken, t.deployConfig.L1ChainID)
	err = makeTerraformEnvFile(fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath), types.TerraformEnvConfig{
		Namespace:           namespace,
		AwsRegion:           awsLoginInputs.Region,
		SequencerKey:        t.deployConfig.SequencerPrivateKey,
		BatcherKey:          t.deployConfig.BatcherPrivateKey,
		ProposerKey:         t.deployConfig.ProposerPrivateKey,
		ChallengerKey:       t.deployConfig.ChallengerPrivateKey,
		EksClusterAdmins:    awsAccountProfile.Arn,
		DeploymentFilePath:  t.deployConfig.DeploymentFilePath,
		L1BeaconUrl:         inputs.L1BeaconURL,
		L1RpcUrl:            t.deployConfig.L1RPCURL,
		L1RpcProvider:       t.deployConfig.L1RPCProvider,
		Azs:                 awsAccountProfile.AvailabilityZones,
		ThanosStackImageTag: constants.DockerImageTag[t.deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[t.deployConfig.Network].OpGethImageTag,
		MaxChannelDuration:  chainConfiguration.GetMaxChannelDuration(),
		TxmgrCellProofTime:  t.deployConfig.TxmgrCellProofTime,
		PrestateHash:        prestateHash,
		EnableFaultProof:    t.deployConfig.EnableFraudProof,
		Preset:              t.deployConfig.Preset,
		NativeTokenName:     feeTokenConfig.Name,
		NativeTokenSymbol:   feeTokenConfig.Symbol,
		NativeTokenAddress:  feeTokenConfig.L1Address,
	})
	if err != nil {
		t.logger.Error("Error generating Terraform environment configuration", "err", err)
		return err
	}

	// STEP 4. Copy configuration files
	err = utils.CopyFile(
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", t.deploymentPath),
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files/rollup.json", t.deploymentPath),
	)
	if err != nil {
		t.logger.Error("Error copying rollup configuration", "err", err)
		return err
	}

	err = utils.CopyFile(
		fmt.Sprintf("%s/tokamak-thanos/build/genesis.json", t.deploymentPath),
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files/genesis.json", t.deploymentPath),
	)
	if err != nil {
		t.logger.Error("Error copying genesis configuration", "err", err)
		return err
	}

	// Copy cannon prestate to Terraform config-files directory.
	// The prestate is built by start-deploy.sh build (make cannon-prestate target)
	// and placed at op-program/bin/prestate.json.
	prestateDstPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files/prestate.json", t.deploymentPath)
	err = utils.CopyFile(prestateSrc, prestateDstPath)
	if err != nil {
		t.logger.Error("Error copying cannon prestate file",
			"err", err,
			"src", prestateSrc,
			"hint", "Run 'make cannon-prestate' in tokamak-thanos to build the prestate")
		return err
	}

	// STEP 5. Initialize Terraform backend
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd backend &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`, t.deploymentPath),
	}...)
	if err != nil {
		t.logger.Error("Error initializing Terraform backend", "err", err)
		return err
	}

	t.logger.Info("Deploying Thanos stack infrastructure")
	// STEP 6. Deploy Thanos stack infrastructure
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve`, t.deploymentPath),
	}...)
	if err != nil {
		t.logger.Error("Error deploying Thanos stack infrastructure", "err", err)
		return err
	}

	// Get VPC ID
	vpcIdOutput, err := utils.ExecuteCommand(ctx, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform output -json vpc_id`, t.deploymentPath),
	}...)
	if err != nil {
		return fmt.Errorf("failed to get terraform output for %s: %w", "vpc_id", err)
	}

	t.deployConfig.AWS.VpcID = strings.Trim(vpcIdOutput, `"`)
	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	thanosStackValueFileExist := utils.CheckFileExists(fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", t.deploymentPath))
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	t.deployConfig.ChainName = inputs.ChainName
	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Sleep for 30 seconds to allow the infrastructure to be fully deployed
	time.Sleep(30 * time.Second)

	// Step 7. Configure EKS access
	if _, err := utils.SetAWSConfigFile(t.deploymentPath); err != nil {
		t.logger.Error("Error setting AWS config file", "err", err)
		return err
	}
	if _, err := utils.SetAWSCredentialsFile(t.deploymentPath); err != nil {
		t.logger.Error("Error setting AWS credentials file", "err", err)
		return err
	}
	if _, err := utils.SetKubeconfigFile(t.deploymentPath); err != nil {
		t.logger.Error("Error setting kubeconfig file", "err", err)
		return err
	}
	err = utils.SwitchKubernetesContext(ctx, namespace, awsLoginInputs.Region)
	if err != nil {
		t.logger.Error("Error switching Kubernetes context", "err", err)
		return err
	}

	// Step 7.1. Check if K8s cluster is ready
	t.logger.Info("Checking if K8s cluster is ready...")
	k8sReady, err := utils.CheckK8sReady(ctx, namespace)
	if err != nil {
		t.logger.Error("❌ Error checking K8s cluster readiness", "err", err)
		return err
	}
	t.logger.Infof("✅ K8s cluster is ready: %t", k8sReady)

	// ---------------------------------------- Deploy chain --------------------------//
	// Step 8. Add Helm repository
	helmAddOuput, err := utils.ExecuteCommand(ctx, "helm", []string{
		"repo",
		"add",
		"thanos-stack",
		"https://tokamak-network.github.io/tokamak-thanos-stack",
	}...)
	if err != nil {
		t.logger.Error("Error adding Helm repository", "err", err, "details", helmAddOuput)
		return err
	}

	// Step 8.1 Search available Helm charts
	helmSearchOutput, err := utils.ExecuteCommand(ctx, "helm", []string{
		"search",
		"repo",
		"thanos-stack",
	}...)
	if err != nil {
		t.logger.Error("Error searching Helm charts", "err", err, "details", helmSearchOutput)
		return err
	}
	t.logger.Info("Helm repository added successfully: \n", helmSearchOutput)

	// Step 8.2. Install Helm charts
	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())
	chartFile := fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", t.deploymentPath)
	valueFile := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", t.deploymentPath)

	// Install the PVC first
	err = utils.UpdateYAMLField(valueFile, "enable_vpc", true)
	if err != nil {
		t.logger.Error("Error updating `enable_vpc` configuration", "err", err)
		return err
	}
	err = utils.InstallHelmRelease(ctx, helmReleaseName, chartFile, valueFile, namespace)
	if err != nil {
		t.logger.Error("Error installing Helm charts", "err", err)
		return err
	}

	t.logger.Info("Wait for the VPCs to be created...")
	err = utils.WaitPVCReady(ctx, namespace)
	if err != nil {
		t.logger.Error("Error waiting for PVC to be ready", "err", err)
		return err
	}

	// Install the rest of the charts
	err = utils.UpdateYAMLField(valueFile, "enable_deployment", true)
	if err != nil {
		t.logger.Error("Error updating `enable_deployment` configuration", "err", err)
	}

	err = utils.InstallHelmRelease(ctx, helmReleaseName, chartFile, valueFile, namespace)
	if err != nil {
		t.logger.Error("Error installing Helm charts", "err", err)
		return err
	}

	t.logger.Info("✅ Helm charts installed successfully")

	var l2RPCUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, helmReleaseName)
		if err != nil {
			t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
			return err
		}

		if len(k8sIngresses) > 0 {
			l2RPCUrl = "http://" + k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	t.logger.Info("✅ Network deployment completed successfully!")
	t.logger.Infof("🌐 RPC endpoint: %s", l2RPCUrl)

	t.deployConfig.K8s = &types.K8sConfig{
		Namespace: namespace,
	}
	t.deployConfig.L2RpcUrl = l2RPCUrl
	t.deployConfig.L1BeaconURL = inputs.L1BeaconURL

	backupEnabled := false
	if t.network == constants.Mainnet {
		// Mainnet always has backup enabled
		backupEnabled = true
	} else if inputs.BackupConfig != nil {
		backupEnabled = inputs.BackupConfig.Enabled
	}

	t.deployConfig.BackupConfig = &types.BackupConfiguration{
		Enabled: backupEnabled,
	}

	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Error saving configuration file", "err", err)
		return err
	}
	t.logger.Infof("Configuration saved successfully to: %s/settings.json", t.deploymentPath)

	// Step 8.3. Initialize backup system (conditional - only if BackupConfig.Enabled is true)
	if backupEnabled {
		fmt.Println("Initializing backup system...")
		err = t.initializeBackupSystem(ctx, inputs.ChainName)
		if err != nil {
			t.logger.Warnf("Warning: Failed to initialize backup system: %v\n", err)
			// Continue deployment even if backup initialization fails
		} else {
			t.logger.Info("✅ Backup system initialized successfully")
		}
	} else {
		t.logger.Info("⏭️ Backup system disabled, skipping initialization")
	}

	// After installing the infra successfully, we install the bridge
	if !inputs.IgnoreInstallBridge {
		_, err = t.InstallBridge(ctx)
		if err != nil {
			t.logger.Error("Error installing bridge", "err", err)
		}
	}

	t.logger.Info("🎉 Thanos Stack installation completed successfully!")
	t.logger.Info("🚀 Your network is now up and running.")
	t.logger.Info("🔧 You can start interacting with your deployed infrastructure.")

	return nil
}

// buildCannonPrestate builds the cannon prestate artifacts in the given tokamak-thanos
// directory and writes prestate.json containing the "pre" hash for FaultDisputeGame.
//
// The root Makefile's cannon-prestate target references op-program-client.elf (without
// the 64 suffix) but the build actually produces op-program-client64.elf. We therefore
// drive the steps manually:
//  1. make op-program  — builds op-program-client64.elf (MIPS64) and cannon binaries
//  2. make cannon      — builds cannon/bin/cannon64-impl
//  3. cannon64-impl load-elf  → prestate.bin.gz + meta.json
//  4. cannon run at step 0    → 0.json (proof), which contains the "pre" hash
//  5. copy 0.json → prestate.json  (readPrestateHash reads "pre" from this file)
func buildCannonPrestate(ctx context.Context, logger *zap.SugaredLogger, tokamakThanosDir string) error {
	steps := []struct {
		desc string
		args []string
	}{
		{"build op-program (MIPS64)", []string{"make", "op-program"}},
		{"build cannon binary", []string{"make", "cannon"}},
	}
	for _, s := range steps {
		logger.Info("Cannon prestate: "+s.desc, "dir", tokamakThanosDir)
		if err := utils.ExecuteCommandStreamInDir(ctx, logger, tokamakThanosDir, s.args[0], s.args[1:]...); err != nil {
			return fmt.Errorf("cannon prestate build step %q failed: %w", s.desc, err)
		}
	}

	elfPath := filepath.Join(tokamakThanosDir, "op-program", "bin", "op-program-client64.elf")
	prestateBin := filepath.Join(tokamakThanosDir, "op-program", "bin", "prestate.bin.gz")
	metaPath := filepath.Join(tokamakThanosDir, "op-program", "bin", "meta.json")
	proofFmt := filepath.Join(tokamakThanosDir, "op-program", "bin", "%d.json")
	proofStep0 := filepath.Join(tokamakThanosDir, "op-program", "bin", "0.json")
	prestateJSON := filepath.Join(tokamakThanosDir, "op-program", "bin", "prestate.json")
	cannonImpl := filepath.Join(tokamakThanosDir, "cannon", "bin", "cannon64-impl")

	logger.Info("Cannon prestate: load-elf → prestate.bin.gz")
	if err := utils.ExecuteCommandStreamInDir(ctx, logger, tokamakThanosDir,
		cannonImpl, "load-elf",
		"--type", "multithreaded64-5",
		"--path", elfPath,
		"--out", prestateBin,
		"--meta", metaPath,
	); err != nil {
		return fmt.Errorf("cannon load-elf failed: %w", err)
	}

	logger.Info("Cannon prestate: run step 0 → 0.json")
	if err := utils.ExecuteCommandStreamInDir(ctx, logger, tokamakThanosDir,
		cannonImpl, "run",
		"--proof-at", "=0",
		"--stop-at", "=1",
		"--input", prestateBin,
		"--meta", metaPath,
		"--proof-fmt", proofFmt,
		"--output", "",
	); err != nil {
		return fmt.Errorf("cannon run step 0 failed: %w", err)
	}

	data, err := os.ReadFile(proofStep0)
	if err != nil {
		return fmt.Errorf("cannon run did not produce 0.json at %s: %w", proofStep0, err)
	}
	if err := os.WriteFile(prestateJSON, data, 0o644); err != nil {
		return fmt.Errorf("failed to write prestate.json: %w", err)
	}
	return nil
}

// readPrestateHash reads the absolute prestate hash from the cannon prestate JSON file.
// The prestate JSON is generated by buildCannonPrestate and contains the "pre" field
// with the hash used to initialize the FaultDisputeGame contract.
func readPrestateHash(prestatePath string) (string, error) {
	data, err := os.ReadFile(prestatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read prestate file %s: %w", prestatePath, err)
	}
	var prestate struct {
		Pre string `json:"pre"`
	}
	if err := json.Unmarshal(data, &prestate); err != nil {
		return "", fmt.Errorf("failed to parse prestate JSON: %w", err)
	}
	if prestate.Pre == "" {
		return "", fmt.Errorf("prestate file %s has empty 'pre' field", prestatePath)
	}
	return prestate.Pre, nil
}
