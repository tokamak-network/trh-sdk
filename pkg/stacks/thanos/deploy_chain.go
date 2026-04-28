package thanos

import (
	"context"
	gocrypto "crypto/ecdsa"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

//go:embed artifacts/FaultDisputeGame.json
var faultDisputeGameArtifact []byte

//go:embed artifacts/OptimismPortal2.json
var optimismPortal2Artifact []byte

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
	case constants.Testnet:
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
		case constants.Local:
			t.deployConfig.ChainName = inputs.ChainName
			t.deployConfig.L1BeaconURL = inputs.L1BeaconURL
			t.deployConfig.Mnemonic = inputs.Mnemonic
			if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
				return fmt.Errorf("failed to write settings file: %w", err)
			}
			return t.deployLocalNetwork(ctx)
		default:
			t.logger.Error("infrastructure provider %s is not supported", infraOpt)
			return fmt.Errorf("infrastructure provider %s is not supported", infraOpt)
		}
	case constants.Mainnet:
		switch infraOpt {
		case constants.AWS:
			err := t.deployNetworkToAWS(ctx, inputs)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					t.logger.Warn("Deployment canceled")
					return err
				}
				t.logger.Error("Failed to deploy the mainnet chain", "err", err)

				if destroyErr := t.destroyInfraOnAWS(ctx); destroyErr != nil {
					t.logger.Error("Failed to destroy the mainnet chain after deploying the chain failed", "err", destroyErr)
				}

				return err
			}

			if inputs != nil && inputs.GithubCredentials != nil && inputs.MetadataInfo != nil {
				_, err = t.RegisterMetadata(ctx, inputs.GithubCredentials, inputs.MetadataInfo)
				if err != nil {
					t.logger.Error("Failed to register metadata", "err", err)
					return err
				}
			}
			return nil
		default:
			t.logger.Error("infrastructure provider %s is not supported for mainnet", infraOpt)
			return fmt.Errorf("infrastructure provider %s is not supported for mainnet; only aws is allowed", infraOpt)
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
	if inputs == nil {
		return fmt.Errorf("inputs is required")
	}

	if t.deployConfig.EnableFraudProof && t.deployConfig.ChallengerPrivateKey == "" {
		return fmt.Errorf("fault proof is enabled but challenger private key is not set; re-run deploy-contracts with --enable-fault-proof")
	}

	// Start parallel tool installation (non-blocking)
	arch, err := dependencies.GetArchitecture(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect architecture: %w", err)
	}
	toolReadiness := NewToolReadiness(t.logger, arch)
	toolReadiness.Start(ctx)

	if err := inputs.Validate(ctx); err != nil {
		t.logger.Error("Error validating inputs", "err", err)
		return err
	}

	// Check if the contracts deployed successfully
	if t.deployConfig.DeployContractState.Status != types.DeployContractStatusCompleted {
		return fmt.Errorf("contracts are not deployed successfully, please deploy the contracts first")
	}

	// STEP 1. Clone the charts repository
	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
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
	prestateHash := ""
	if t.deployConfig.EnableFraudProof {
		var readErr error
		prestateHash, readErr = readPrestateHash(prestateSrc)
		if readErr != nil {
			t.logger.Error("Failed to read cannon prestate hash",
				"err", readErr,
				"path", prestateSrc,
				"hint", "Ensure start-deploy.sh build completed successfully before deploying chain")
			return readErr
		}
		t.logger.Info("Cannon prestate hash loaded", "hash", prestateHash)
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	// L2 native token is always TON regardless of fee token.
	// Non-TON fee tokens are handled by the AA paymaster layer.
	tonConfig := constants.GetFeeTokenConfig(constants.FeeTokenTON, t.deployConfig.L1ChainID)
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
		NativeTokenName:     tonConfig.Name,
		NativeTokenSymbol:   tonConfig.Symbol,
		NativeTokenAddress:  tonConfig.L1Address,
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

	// Wait for terraform to be installed before infrastructure provisioning
	if err := toolReadiness.WaitFor(ctx, "terraform"); err != nil {
		return fmt.Errorf("tool installation failed before infrastructure provisioning: %w", err)
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

	// Wait for AWS CLI and kubectl before EKS configuration
	if err := toolReadiness.WaitFor(ctx, "aws-cli", "kubectl"); err != nil {
		return fmt.Errorf("tool installation failed before EKS configuration: %w", err)
	}

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
	// Wait for helm before chart deployment
	if err := toolReadiness.WaitFor(ctx, "helm"); err != nil {
		return fmt.Errorf("tool installation failed before Helm deployment: %w", err)
	}

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

	// Apply testnet resource optimizations to reduce Fargate costs.
	// Testnet runs on the same public infrastructure as mainnet, so it cannot
	// be torn down on-demand — explicit smaller requests are the correct lever.
	if t.network == constants.Testnet {
		testnetResources := map[string]string{
			"op_geth.resources.cpu":        "500m",
			"op_geth.resources.memory":     "1Gi",
			"op_node.resources.cpu":        "500m",
			"op_node.resources.memory":     "1Gi",
			"op_batcher.resources.cpu":     "250m",
			"op_batcher.resources.memory":  "512Mi",
			"op_proposer.resources.cpu":    "250m",
			"op_proposer.resources.memory": "512Mi",
			"redis.resources.cpu":          "250m",
			"redis.resources.memory":       "512Mi",
		}
		for field, value := range testnetResources {
			if err = utils.UpdateYAMLField(valueFile, field, value); err != nil {
				t.logger.Error("Error setting testnet resource", "field", field, "err", err)
				return err
			}
		}
		t.logger.Info("✅ Testnet resource optimizations applied")
	}

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

	// Step 8.2.5. Initialize AnchorStateRegistry genesis anchor state (fault proof chains only).
	// Without this, every FaultDisputeGame.initialize() reverts with AnchorRootNotFound because
	// the registry starts with bytes32(0) as the anchor root for every game type.
	if t.deployConfig.EnableFraudProof {
		deployedContracts, contractsErr := t.readDeploymentContracts()
		if contractsErr != nil {
			return fmt.Errorf("failed to read deployed contracts for anchor init: %w", contractsErr)
		}
		if deployedContracts.AnchorStateRegistryProxy == "" {
			return fmt.Errorf("AnchorStateRegistryProxy address not found in deployed contracts — cannot initialize genesis anchor state")
		}
		anchorErr := initGenesisAnchorState(
			ctx,
			t.logger,
			t.deployConfig.L1RPCURL,
			l2RPCUrl,
			t.deployConfig.AdminPrivateKey,
			deployedContracts.AnchorStateRegistryProxy,
			deployedContracts.ProxyAdmin,
			deployedContracts.AnchorStateRegistry, // impl addr for StorageSetter fallback restore
			t.deployConfig.L1ChainID,
			0, // gameType 0 = CANNON (default respected game type)
		)
		if anchorErr != nil {
			return fmt.Errorf("failed to initialize genesis anchor state (op-proposer will fail with AnchorRootNotFound): %w", anchorErr)
		}
		t.logger.Info("✅ Genesis anchor state initialized in AnchorStateRegistry")
	}

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

	// Auto-install preset modules
	if t.deployConfig.Preset != "" {
		if err := t.installPresetModules(ctx); err != nil {
			t.logger.Warnw("Some preset modules failed to install — deployment continues", "err", err)
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

// patchAnchorStateRegistry adds setInitialAnchorState to AnchorStateRegistry.sol after cloning.
// This function enables the guardian to bootstrap a fresh chain's anchor state without needing a
// resolved FaultDisputeGame (which is impossible on a brand-new chain).
// The patch is idempotent: calling it multiple times on the same file is safe.
func patchAnchorStateRegistry(tokamakThanosDir string) error {
	contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock", "src", "dispute")

	// Patch AnchorStateRegistry.sol
	solPath := filepath.Join(contractsDir, "AnchorStateRegistry.sol")
	solData, err := os.ReadFile(solPath)
	if err != nil {
		return fmt.Errorf("failed to read AnchorStateRegistry.sol: %w", err)
	}
	solContent := string(solData)
	const setInitialMarker = "function setInitialAnchorState("
	if !strings.Contains(solContent, setInitialMarker) {
		const insertBefore = "    /// @inheritdoc IAnchorStateRegistry\n    function setAnchorState(IFaultDisputeGame _game) external {"
		const newFn = `    /// @notice Sets the initial anchor state for a game type directly using an OutputRoot.
    ///         This is intended for bootstrapping new chains where no resolved game exists yet.
    ///         Only callable by the guardian. Does not require a resolved FaultDisputeGame.
    /// @param _gameType The game type to set the anchor state for.
    /// @param _outputRoot The initial anchor OutputRoot (l2BlockNumber + root hash).
    function setInitialAnchorState(GameType _gameType, OutputRoot calldata _outputRoot) external {
        if (msg.sender != superchainConfig.guardian()) revert Unauthorized();
        anchors[_gameType] = _outputRoot;
    }

    ` + insertBefore
		if !strings.Contains(solContent, insertBefore) {
			return fmt.Errorf("AnchorStateRegistry.sol: expected anchor text not found, cannot apply patch")
		}
		solContent = strings.Replace(solContent, insertBefore, newFn, 1)
		if err := os.WriteFile(solPath, []byte(solContent), 0644); err != nil {
			return fmt.Errorf("failed to write patched AnchorStateRegistry.sol: %w", err)
		}
	}

	// Patch IAnchorStateRegistry.sol
	ifacePath := filepath.Join(contractsDir, "interfaces", "IAnchorStateRegistry.sol")
	ifaceData, err := os.ReadFile(ifacePath)
	if err != nil {
		return fmt.Errorf("failed to read IAnchorStateRegistry.sol: %w", err)
	}
	ifaceContent := string(ifaceData)
	if !strings.Contains(ifaceContent, setInitialMarker) {
		const ifaceInsertBefore = "    function setAnchorState(IFaultDisputeGame _game) external;\n}"
		const ifaceNewFn = `    function setAnchorState(IFaultDisputeGame _game) external;

    /// @notice Sets the initial anchor state directly using an OutputRoot.
    ///         For bootstrapping new chains where no resolved game exists yet.
    ///         Only callable by the guardian.
    function setInitialAnchorState(GameType _gameType, OutputRoot calldata _outputRoot) external;
}`
		if !strings.Contains(ifaceContent, ifaceInsertBefore) {
			return fmt.Errorf("IAnchorStateRegistry.sol: expected anchor text not found, cannot apply patch")
		}
		ifaceContent = strings.Replace(ifaceContent, ifaceInsertBefore, ifaceNewFn, 1)
		if err := os.WriteFile(ifacePath, []byte(ifaceContent), 0644); err != nil {
			return fmt.Errorf("failed to write patched IAnchorStateRegistry.sol: %w", err)
		}
	}

	return nil
}

// initGenesisAnchorState bootstraps the AnchorStateRegistry for a brand-new chain.
//
// Problem: FaultDisputeGame.initialize() reverts with AnchorRootNotFound when
// anchors[gameType].root == bytes32(0). On a fresh chain there are no resolved games,
// so the only way to set the first anchor is via a guardian-privileged call. The patched
// AnchorStateRegistry.setInitialAnchorState satisfies this.
//
// The genesis output root is computed as:
//
//	keccak256(version(32) || stateRoot(32) || messagePasserStorageRoot(32) || blockHash(32))
//
// At genesis:
//   - version = bytes32(0)
//   - stateRoot = genesis block header Root field
//   - messagePasserStorageRoot = empty MPT root (L2ToL1MessagePasser has no storage at block 0)
//   - blockHash = genesis block hash
// storageSetterBytecode is a pre-compiled StorageSetter contract that exposes
// setBytes32(bytes32,bytes32). Deployed transiently during RC3 fallback to write
// the genesis anchor root directly into AnchorStateRegistry proxy storage.
// Compiled from Optimism's StorageSetter.sol at solc 0.8.15.
var storageSetterBytecode = common.FromHex("608060405234801561001057600080fd5b506103a9806100206000396000f3fe608060405234801561001057600080fd5b506004361061009e5760003560e01c8063a6ed563e11610066578063a6ed563e14610149578063abfdcced14610165578063bd02d0f514610149578063ca446dd914610173578063e2a4853a146100e857600080fd5b80630528afe2146100a357806321f8a721146100b85780634e91db08146100e857806354fd4d50146100fa5780637ae1cfca1461012b575b600080fd5b6100b66100b13660046101f4565b610181565b005b6100cb6100c6366004610269565b6101e4565b6040516001600160a01b0390911681526020015b60405180910390f35b6100b66100f6366004610282565b9055565b61011e604051806040016040528060058152602001640312e322e360dc1b81525081565b6040516100df91906102a4565b6101396100c6366004610269565b60405190151581526020016100df565b6101576100c6366004610269565b6040519081526020016100df565b6100b66100f63660046102f9565b6100b66100f636600461032e565b8060005b818110156101de576101cc8484838181106101a2576101a261035f565b905060400201600001358585848181106101be576101be61035f565b905060400201602001359055565b806101d681610375565b915050610185565b50505050565b60006101ee825490565b92915050565b6000806020838503121561020757600080fd5b823567ffffffffffffffff8082111561021f57600080fd5b818501915085601f83011261023357600080fd5b81358181111561024257600080fd5b8660208260061b850101111561025757600080fd5b60209290920196919550909350505050565b60006020828403121561027b57600080fd5b5035919050565b6000806040838503121561029557600080fd5b50508035926020909101359150565b600060208083528351808285015260005b818110156102d1578581018301518582016040015282016102b5565b818111156102e3576000604083870101525b50601f01601f1916929092016040019392505050565b6000806040838503121561030c57600080fd5b823591506020830135801515811461032357600080fd5b809150509250929050565b6000806040838503121561034157600080fd5b8235915060208301356001600160a01b038116811461032357600080fd5b634e487b7160e01b600052603260045260246000fd5b60006001820161039557634e487b7160e01b600052601160045260246000fd5b506001019056fea164736f6c634300080f000a")

// bootstrapAnchorStateViaStorageSetter writes outputRootHash to the AnchorStateRegistry proxy
// storage slot for anchors[gameType].root without requiring setInitialAnchorState on the impl.
// Used as an RC3 fallback when the deployed impl (embedded pre-compiled bytecode in
// tokamak-deployer) lacks the setInitialAnchorState function.
//
// Procedure mirrors scripts/fix-anchor-state-registry.mjs:
//  1. Verify ProxyAdmin owner is EOA (Gnosis Safe would require multisig approval)
//  2. Deploy StorageSetter transiently
//  3. ProxyAdmin.upgradeAndCall(proxy, storageSetter, setBytes32(slot, root))
//  4. Verify storage slot was written
//  5. ProxyAdmin.upgrade(proxy, originalImpl) — restore; loud-fail if this fails
//
// Storage slot: keccak256(abi.encode(uint256(gameType), uint256(1)))
// For gameType=0: 0xa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb49
// This assumes anchors mapping is at storage slot 1 in the current impl bytecode.
func bootstrapAnchorStateViaStorageSetter(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1Client *ethclient.Client,
	privKey *gocrypto.PrivateKey,
	adminAddr common.Address,
	chainID *big.Int,
	anchorProxy common.Address,
	proxyAdmin common.Address,
	originalImpl common.Address,
	outputRootHash common.Hash,
	gameType uint32,
) error {
	// 1. Verify ProxyAdmin owner is EOA.
	ownerSel := crypto.Keccak256([]byte("owner()"))[:4]
	ownerResult, err := l1Client.CallContract(ctx, ethereum.CallMsg{To: &proxyAdmin, Data: ownerSel}, nil)
	if err != nil || len(ownerResult) < 32 {
		return fmt.Errorf("failed to call ProxyAdmin.owner(): %w", err)
	}
	ownerAddr := common.BytesToAddress(ownerResult[12:32])
	ownerCode, err := l1Client.CodeAt(ctx, ownerAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to check ProxyAdmin owner code size: %w", err)
	}
	if len(ownerCode) > 0 {
		return fmt.Errorf("ProxyAdmin owner %s is a contract (Gnosis Safe?), not EOA — "+
			"StorageSetter fallback requires EOA ownership; run fix-anchor-state-registry.mjs manually", ownerAddr.Hex())
	}

	// If originalImpl was not provided (zero address), read it from the proxy's EIP-1967 impl slot.
	// The Go deployer's deploy-output.json may omit "AnchorStateRegistry" (impl), so we fall back here.
	if originalImpl == (common.Address{}) {
		eip1967ImplSlot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
		implSlotVal, slotErr := l1Client.StorageAt(ctx, anchorProxy, eip1967ImplSlot, nil)
		if slotErr != nil {
			return fmt.Errorf("originalImpl is zero and failed to read EIP-1967 impl slot from proxy: %w", slotErr)
		}
		originalImpl = common.BytesToAddress(implSlotVal[12:32])
		if originalImpl == (common.Address{}) {
			return fmt.Errorf("originalImpl is zero and EIP-1967 impl slot also returned zero — proxy may not be EIP-1967 compliant")
		}
		logger.Infof("originalImpl resolved from EIP-1967 slot: %s", originalImpl.Hex())
	}

	signer := ethtypes.NewEIP155Signer(chainID)

	sendAndWait := func(nonce uint64, to *common.Address, data []byte, gasLimit uint64, gasPrice *big.Int) (*ethtypes.Receipt, error) {
		var tx *ethtypes.Transaction
		if to == nil {
			tx = ethtypes.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, data)
		} else {
			tx = ethtypes.NewTransaction(nonce, *to, big.NewInt(0), gasLimit, gasPrice, data)
		}
		signed, signErr := ethtypes.SignTx(tx, signer, privKey)
		if signErr != nil {
			return nil, fmt.Errorf("sign tx: %w", signErr)
		}
		if sendErr := l1Client.SendTransaction(ctx, signed); sendErr != nil {
			return nil, fmt.Errorf("send tx: %w", sendErr)
		}
		receipt, waitErr := bind.WaitMined(ctx, l1Client, signed)
		if waitErr != nil {
			return nil, fmt.Errorf("wait mined: %w", waitErr)
		}
		if receipt.Status != ethtypes.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("tx reverted (hash: %s)", signed.Hash().Hex())
		}
		return receipt, nil
	}

	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	// 2. Deploy StorageSetter.
	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	deployReceipt, err := sendAndWait(nonce, nil, storageSetterBytecode, 500_000, gasPrice)
	if err != nil {
		return fmt.Errorf("StorageSetter deploy failed: %w", err)
	}
	storageSetterAddr := deployReceipt.ContractAddress
	logger.Infof("StorageSetter deployed at %s", storageSetterAddr.Hex())

	// 3. Build setBytes32(slot, root) calldata.
	// Storage slot for anchors[gameType].root (anchors mapping at storage slot 1):
	//   keccak256(abi.encode(uint256(gameType), uint256(1)))
	// For gameType=0 this is 0xa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb49.
	var slotPreimage [64]byte
	binary.BigEndian.PutUint32(slotPreimage[28:32], gameType) // uint256(gameType), right-aligned
	slotPreimage[63] = 1                                      // uint256(1) for slot index
	anchorsRootSlot := crypto.Keccak256Hash(slotPreimage[:])

	setBytes32Sel := crypto.Keccak256([]byte("setBytes32(bytes32,bytes32)"))[:4]
	setBytes32Data := make([]byte, 68)
	copy(setBytes32Data[0:4], setBytes32Sel)
	copy(setBytes32Data[4:36], anchorsRootSlot.Bytes())
	copy(setBytes32Data[36:68], outputRootHash.Bytes())

	// 4. Build upgradeAndCall(proxy, storageSetter, setBytes32Data) calldata.
	// ABI-encode (address, address, bytes): static head (3×32) + dynamic tail (length word + padded data).
	paddedLen := (len(setBytes32Data) + 31) / 32 * 32
	upgradeAndCallSel := crypto.Keccak256([]byte("upgradeAndCall(address,address,bytes)"))[:4]
	upgradeAndCallData := make([]byte, 4+3*32+32+paddedLen)
	copy(upgradeAndCallData[0:4], upgradeAndCallSel)
	copy(upgradeAndCallData[4+12:4+32], anchorProxy.Bytes())
	copy(upgradeAndCallData[36+12:68], storageSetterAddr.Bytes())
	upgradeAndCallData[4+64+31] = 96 // offset to bytes = 3×32 = 96
	binary.BigEndian.PutUint64(upgradeAndCallData[100+24:132], uint64(len(setBytes32Data)))
	copy(upgradeAndCallData[132:], setBytes32Data)

	nonce, err = l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for upgradeAndCall: %w", err)
	}
	if _, err = sendAndWait(nonce, &proxyAdmin, upgradeAndCallData, 300_000, gasPrice); err != nil {
		return fmt.Errorf("ProxyAdmin.upgradeAndCall (StorageSetter) failed: %w", err)
	}
	logger.Infof("upgradeAndCall(StorageSetter) confirmed — slot %s should be set", anchorsRootSlot.Hex())

	// 5. Verify storage slot was written before restoring impl.
	storedValue, err := l1Client.StorageAt(ctx, anchorProxy, anchorsRootSlot, nil)
	if err != nil {
		return fmt.Errorf("failed to verify storage slot after StorageSetter write: %w", err)
	}
	storedHash := common.BytesToHash(storedValue)
	if storedHash != outputRootHash {
		return fmt.Errorf("storage slot verification failed: expected %s, got %s", outputRootHash.Hex(), storedHash.Hex())
	}
	logger.Infof("✅ Storage slot verified: anchors[%d].root = %s", gameType, storedHash.Hex())

	// 6. Restore original impl via ProxyAdmin.upgrade(proxy, originalImpl).
	// LOUD-FAIL: if this fails the proxy is stuck pointing at StorageSetter — must not swallow.
	upgradeSel := crypto.Keccak256([]byte("upgrade(address,address)"))[:4]
	upgradeData := make([]byte, 4+32+32)
	copy(upgradeData[0:4], upgradeSel)
	copy(upgradeData[4+12:4+32], anchorProxy.Bytes())
	copy(upgradeData[36+12:68], originalImpl.Bytes())

	nonce, err = l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("CRITICAL: failed to get nonce for impl restore (proxy stuck at StorageSetter %s): %w", storageSetterAddr.Hex(), err)
	}
	if _, err = sendAndWait(nonce, &proxyAdmin, upgradeData, 200_000, gasPrice); err != nil {
		return fmt.Errorf("CRITICAL: ProxyAdmin.upgrade (restore impl) failed — proxy %s is stuck pointing at StorageSetter %s, manual intervention required: %w",
			anchorProxy.Hex(), storageSetterAddr.Hex(), err)
	}
	logger.Infof("✅ AnchorStateRegistry impl restored to %s", originalImpl.Hex())
	return nil
}

func initGenesisAnchorState(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	l2RPCURL string,
	adminPrivateKey string,
	anchorStateRegistryAddr string,
	proxyAdminAddr string,
	implAddr string,
	l1ChainID uint64,
	gameType uint32,
) error {
	// 1. Connect to L2 and wait for genesis block.
	l2Client, err := ethclient.DialContext(ctx, l2RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L2 RPC %s: %w", l2RPCURL, err)
	}
	defer l2Client.Close()

	var genesisBlock *ethtypes.Block
	for attempt := 1; attempt <= 20; attempt++ {
		genesisBlock, err = l2Client.BlockByNumber(ctx, big.NewInt(0))
		if err == nil {
			break
		}
		logger.Warnf("Waiting for L2 genesis block (attempt %d/20): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("L2 genesis block unavailable after retries: %w", err)
	}
	logger.Infof("Genesis block: hash=%s stateRoot=%s", genesisBlock.Hash().Hex(), genesisBlock.Root().Hex())

	// 2. Compute genesis output root.
	// At genesis the L2ToL1MessagePasser (0x4200...0016) has no storage, so its
	// storage root equals the canonical empty MPT root.
	emptyMPTRoot := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	var preimage [128]byte
	// [0:32]   = version = 0 (left as zero)
	copy(preimage[32:64], genesisBlock.Root().Bytes())  // stateRoot
	copy(preimage[64:96], emptyMPTRoot.Bytes())         // messagePasserStorageRoot
	copy(preimage[96:128], genesisBlock.Hash().Bytes()) // blockHash
	outputRootHash := crypto.Keccak256Hash(preimage[:])
	logger.Infof("Genesis output root: %s", outputRootHash.Hex())

	// 3. Build calldata for setInitialAnchorState(uint32,(bytes32,uint256)).
	// Static ABI layout (no dynamic types):
	//   [0:4]    selector
	//   [4:36]   gameType  (uint32, right-aligned in 32 bytes)
	//   [36:68]  root      (bytes32)
	//   [68:100] l2BlockNumber (uint256 = 0 for genesis)
	selector := crypto.Keccak256([]byte("setInitialAnchorState(uint32,(bytes32,uint256))"))[:4]
	calldata := make([]byte, 100)
	copy(calldata[0:4], selector)
	binary.BigEndian.PutUint32(calldata[32:36], gameType) // right-aligned uint32
	copy(calldata[36:68], outputRootHash.Bytes())
	// calldata[68:100] = 0 already (genesis l2BlockNumber = 0)

	// 4. Connect to L1 and send transaction from admin (= guardian).
	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1Client.Close()

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))
	anchorAddr := common.HexToAddress(anchorStateRegistryAddr)

	// Guard A: Idempotency — skip if anchor state already set.
	// Handles re-runs after a partial success or after fix-anchor-state-registry.mjs was applied.
	anchorsSelector := crypto.Keccak256([]byte("anchors(uint32)"))[:4]
	anchorsCalldata := make([]byte, 36)
	copy(anchorsCalldata[0:4], anchorsSelector)
	binary.BigEndian.PutUint32(anchorsCalldata[32:36], gameType)
	if existingResult, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &anchorAddr, Data: anchorsCalldata}, nil); callErr == nil && len(existingResult) >= 32 {
		existingRoot := common.BytesToHash(existingResult[:32])
		if existingRoot != (common.Hash{}) {
			logger.Infof("✅ Anchor state already set for gameType %d (root=%s), skipping", gameType, existingRoot.Hex())
			return nil
		}
	}

	// Guard B: Pre-flight eth_call simulation — detect missing setInitialAnchorState before wasting gas.
	// If the simulation reverts, the deployed impl (embedded pre-compiled bytecode in tokamak-deployer)
	// lacks setInitialAnchorState. Automatically fall back to the StorageSetter pattern.
	if _, simErr := l1Client.CallContract(ctx, ethereum.CallMsg{From: adminAddr, To: &anchorAddr, Data: calldata}, nil); simErr != nil {
		logger.Warnf("setInitialAnchorState simulation failed for %s — impl lacks function, applying StorageSetter fallback (RC3): %v", anchorStateRegistryAddr, simErr)
		return bootstrapAnchorStateViaStorageSetter(
			ctx, logger, l1Client, privKey, adminAddr, chainID,
			anchorAddr,
			common.HexToAddress(proxyAdminAddr),
			common.HexToAddress(implAddr),
			outputRootHash,
			gameType,
		)
	}

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}
	// Double gas price for reliable inclusion.
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := ethtypes.NewTransaction(nonce, anchorAddr, big.NewInt(0), 200_000, gasPrice, calldata)
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send setInitialAnchorState tx: %w", err)
	}
	logger.Infof("setInitialAnchorState tx sent: %s (waiting for receipt...)", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for setInitialAnchorState tx receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("setInitialAnchorState tx reverted (tx: %s, gas used: %d) — "+
			"the deployed AnchorStateRegistry may lack the setInitialAnchorState function; "+
			"check that forge recompiled the patched contract",
			signedTx.Hash().Hex(), receipt.GasUsed)
	}
	logger.Infof("✅ setInitialAnchorState confirmed in block %d (tx: %s)", receipt.BlockNumber.Uint64(), signedTx.Hash().Hex())
	return nil
}

// initL1CrossDomainMessenger calls initialize(SuperchainConfig, OptimismPortal, SystemConfig)
// on the L1CrossDomainMessengerProxy. tokamak-deployer only calls upgrade(proxy, impl), so the
// initialize() initializer is never invoked, leaving portal = 0x0 and CDM non-functional.
// Idempotency: skipped if portal() already returns a non-zero address.
func initL1CrossDomainMessenger(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	cdmProxyAddr string,
	superchainConfigAddr string,
	portalAddr string,
	systemConfigAddr string,
	l1ChainID uint64,
) error {
	for _, pair := range []struct{ name, val string }{
		{"cdmProxyAddr", cdmProxyAddr},
		{"superchainConfigAddr", superchainConfigAddr},
		{"portalAddr", portalAddr},
		{"systemConfigAddr", systemConfigAddr},
	} {
		if pair.val == "" {
			return fmt.Errorf("initL1CrossDomainMessenger: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for CDM init: %w", err)
	}
	defer l1Client.Close()

	cdmAddr := common.HexToAddress(cdmProxyAddr)

	// Idempotency guard: read portal() slot. If already non-zero, CDM is initialized.
	portalSelector := crypto.Keccak256([]byte("portal()"))[:4]
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &cdmAddr, Data: portalSelector}, nil); callErr == nil && len(result) >= 32 {
		existingPortal := common.BytesToAddress(result[12:32])
		if existingPortal != (common.Address{}) {
			logger.Infof("✅ L1CrossDomainMessenger already initialized (portal=%s), skipping", existingPortal.Hex())
			return nil
		}
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for CDM init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// Build initialize(address,address,address) calldata.
	// ABI encoding: 4-byte selector + 3 * 32-byte left-padded address slots.
	selector := crypto.Keccak256([]byte("initialize(address,address,address)"))[:4]
	calldata := make([]byte, 100)
	copy(calldata[0:4], selector)
	copy(calldata[16:36], common.HexToAddress(superchainConfigAddr).Bytes()) // slot 0
	copy(calldata[48:68], common.HexToAddress(portalAddr).Bytes())            // slot 1
	copy(calldata[80:100], common.HexToAddress(systemConfigAddr).Bytes())     // slot 2

	// Pre-flight: detect reverts before spending L1 gas (mirrors initGenesisAnchorState pattern)
	if _, simErr := l1Client.CallContract(ctx, ethereum.CallMsg{From: adminAddr, To: &cdmAddr, Data: calldata}, nil); simErr != nil {
		return fmt.Errorf("L1CrossDomainMessenger initialize pre-flight failed: %w", simErr)
	}

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for CDM init: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for CDM init: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := ethtypes.NewTransaction(nonce, cdmAddr, big.NewInt(0), 300_000, gasPrice, calldata)
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign CDM init tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send CDM init tx: %w", err)
	}
	logger.Infof("L1CrossDomainMessenger.initialize() tx sent: %s (waiting for receipt...)", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for CDM init tx receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("L1CrossDomainMessenger.initialize() tx reverted (tx: %s)", signedTx.Hash().Hex())
	}
	logger.Info("✅ L1CrossDomainMessenger initialized successfully")
	return nil
}

// initL2OutputOracle calls initialize(...) on the L2OutputOracleProxy.
// tokamak-deployer only calls upgrade(proxy, impl), so the initialize() initializer is
// never invoked, leaving proposer = address(0) and op-proposer unable to submit outputs.
// Idempotency: skipped if proposer() already returns a non-zero address.
func initL2OutputOracle(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	l2OOProxyAddr string,
	submissionInterval uint64,
	l2BlockTime uint64,
	startingBlockNumber uint64,
	startingTimestamp uint64,
	proposerAddr string,
	challengerAddr string,
	finalizationPeriodSeconds uint64,
	l1ChainID uint64,
) error {
	for _, pair := range []struct{ name, val string }{
		{"l2OOProxyAddr", l2OOProxyAddr},
		{"proposerAddr", proposerAddr},
		{"challengerAddr", challengerAddr},
	} {
		if pair.val == "" {
			return fmt.Errorf("initL2OutputOracle: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for L2OO init: %w", err)
	}
	defer l1Client.Close()

	l2OOAddr := common.HexToAddress(l2OOProxyAddr)

	// Idempotency guard: read proposer(). If already non-zero, L2OO is initialized.
	proposerSelector := crypto.Keccak256([]byte("proposer()"))[:4]
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &l2OOAddr, Data: proposerSelector}, nil); callErr == nil && len(result) >= 32 {
		existingProposer := common.BytesToAddress(result[12:32])
		if existingProposer != (common.Address{}) {
			logger.Infof("✅ L2OutputOracle already initialized (proposer=%s), skipping", existingProposer.Hex())
			return nil
		}
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for L2OO init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// Build initialize(uint256,uint256,uint256,uint256,address,address,uint256) calldata.
	// ABI: 4-byte selector + 7 × 32-byte slots (uint256 big-endian, address right-aligned).
	selector := crypto.Keccak256([]byte("initialize(uint256,uint256,uint256,uint256,address,address,uint256)"))[:4]
	calldata := make([]byte, 4+7*32)
	copy(calldata[0:4], selector)
	new(big.Int).SetUint64(submissionInterval).FillBytes(calldata[4+0*32 : 4+1*32])
	new(big.Int).SetUint64(l2BlockTime).FillBytes(calldata[4+1*32 : 4+2*32])
	new(big.Int).SetUint64(startingBlockNumber).FillBytes(calldata[4+2*32 : 4+3*32])
	new(big.Int).SetUint64(startingTimestamp).FillBytes(calldata[4+3*32 : 4+4*32])
	copy(calldata[4+4*32+12:4+5*32], common.HexToAddress(proposerAddr).Bytes())
	copy(calldata[4+5*32+12:4+6*32], common.HexToAddress(challengerAddr).Bytes())
	new(big.Int).SetUint64(finalizationPeriodSeconds).FillBytes(calldata[4+6*32 : 4+7*32])

	// Pre-flight simulation before spending L1 gas.
	if _, simErr := l1Client.CallContract(ctx, ethereum.CallMsg{From: adminAddr, To: &l2OOAddr, Data: calldata}, nil); simErr != nil {
		return fmt.Errorf("L2OutputOracle initialize pre-flight failed: %w", simErr)
	}

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for L2OO init: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for L2OO init: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := ethtypes.NewTransaction(nonce, l2OOAddr, big.NewInt(0), 300_000, gasPrice, calldata)
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign L2OO init tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send L2OO init tx: %w", err)
	}
	logger.Infof("L2OutputOracle.initialize() tx sent: %s (waiting for receipt...)", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for L2OO init tx receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("L2OutputOracle.initialize() tx reverted (tx: %s)", signedTx.Hash().Hex())
	}
	logger.Info("✅ L2OutputOracle initialized successfully")
	return nil
}

// initSystemConfig calls initialize(...) on the SystemConfigProxy.
// tokamak-deployer only calls upgrade(proxy, impl) and never invokes initialize(),
// leaving batcherHash, gasLimit, unsafeBlockSigner, and all L1 contract addresses at
// zero / empty. op-node reads batcherHash + gasLimit from SystemConfig for derivation,
// so the chain stalls without this call.
// Idempotency: skipped if gasLimit() already returns a non-zero value.
func initSystemConfig(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	systemConfigProxyAddr string,
	owner string,
	batcherAddr string,
	gasLimit uint64,
	unsafeBlockSigner string,
	batchInbox string,
	l1CDMProxy string,
	l1ERC721BridgeProxy string,
	l1StandardBridgeProxy string,
	disputeGameFactoryProxy string,
	optimismPortalProxy string,
	optimismMintableERC20FactoryProxy string,
	nativeTokenAddress string,
	l1ChainID uint64,
) error {
	for _, pair := range []struct{ name, val string }{
		{"systemConfigProxyAddr", systemConfigProxyAddr},
		{"owner", owner},
		{"batcherAddr", batcherAddr},
		{"unsafeBlockSigner", unsafeBlockSigner},
		{"batchInbox", batchInbox},
		{"l1CDMProxy", l1CDMProxy},
		{"l1StandardBridgeProxy", l1StandardBridgeProxy},
		{"optimismPortalProxy", optimismPortalProxy},
	} {
		if pair.val == "" {
			return fmt.Errorf("initSystemConfig: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for SystemConfig init: %w", err)
	}
	defer l1Client.Close()

	scAddr := common.HexToAddress(systemConfigProxyAddr)

	// Idempotency guard: read gasLimit(). If non-zero, SystemConfig is already initialized.
	gasLimitSelector := crypto.Keccak256([]byte("gasLimit()"))[:4]
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &scAddr, Data: gasLimitSelector}, nil); callErr == nil && len(result) >= 32 {
		existing := new(big.Int).SetBytes(result[24:32])
		if existing.Sign() > 0 {
			logger.Infof("✅ SystemConfig already initialized (gasLimit=%d), skipping", existing.Uint64())
			return nil
		}
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for SystemConfig init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// ABI selector for:
	// initialize(address,uint32,uint32,bytes32,uint64,address,
	//            (uint32,uint8,uint8,uint32,uint32,uint128),
	//            address,
	//            (address,address,address,address,address,address,address,address))
	selector := crypto.Keccak256([]byte(
		"initialize(address,uint32,uint32,bytes32,uint64,address,(uint32,uint8,uint8,uint32,uint32,uint128),address,(address,address,address,address,address,address,address,address))",
	))[:4]

	// Standard OP Stack EIP-4844 scalar defaults (used across Tokamak deployments).
	const (
		basefeeScalar     = uint32(1368)
		blobbasefeeScalar = uint32(810949)
	)

	// Standard ResourceConfig values matching the OP Stack reference deployment.
	const (
		rcMaxResourceLimit            = uint32(20_000_000)
		rcElasticityMultiplier        = uint8(10)
		rcBaseFeeMaxChangeDenominator = uint8(8)
		rcMinimumBaseFee              = uint32(1_000_000_000) // 1 gwei
		rcSystemTxMaxGas              = uint32(1_000_000)
	)
	// maximumBaseFee = type(uint128).max = 2^128 - 1
	maxUint128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

	// batcherHash is the batcher address zero-padded to bytes32 (left-aligned in uint256).
	var batcherHash [32]byte
	batcherAddrBytes := common.HexToAddress(batcherAddr).Bytes()
	copy(batcherHash[12:], batcherAddrBytes) // address occupies bytes 12-31

	// nativeTokenAddress defaults to address(0) if not set (ETH fee token).
	nativeTokenAddr := common.Address{}
	if nativeTokenAddress != "" && nativeTokenAddress != "0x0000000000000000000000000000000000000000" {
		nativeTokenAddr = common.HexToAddress(nativeTokenAddress)
	}

	// disputeGameFactory defaults to address(0) for L2OO mode.
	dgfAddr := common.Address{}
	if disputeGameFactoryProxy != "" {
		dgfAddr = common.HexToAddress(disputeGameFactoryProxy)
	}

	// Build calldata: 4-byte selector + 21 × 32-byte slots.
	calldata := make([]byte, 4+21*32)
	copy(calldata[0:4], selector)
	offset := 4

	// slot 0: _owner (address)
	copy(calldata[offset+12:offset+32], common.HexToAddress(owner).Bytes())
	offset += 32
	// slot 1: _basefeeScalar (uint32)
	new(big.Int).SetUint64(uint64(basefeeScalar)).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 2: _blobbasefeeScalar (uint32)
	new(big.Int).SetUint64(uint64(blobbasefeeScalar)).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 3: _batcherHash (bytes32)
	copy(calldata[offset:offset+32], batcherHash[:])
	offset += 32
	// slot 4: _gasLimit (uint64)
	new(big.Int).SetUint64(gasLimit).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 5: _unsafeBlockSigner (address)
	copy(calldata[offset+12:offset+32], common.HexToAddress(unsafeBlockSigner).Bytes())
	offset += 32
	// ResourceConfig tuple (6 slots):
	// slot 6: maxResourceLimit (uint32)
	new(big.Int).SetUint64(uint64(rcMaxResourceLimit)).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 7: elasticityMultiplier (uint8)
	calldata[offset+31] = byte(rcElasticityMultiplier)
	offset += 32
	// slot 8: baseFeeMaxChangeDenominator (uint8)
	calldata[offset+31] = byte(rcBaseFeeMaxChangeDenominator)
	offset += 32
	// slot 9: minimumBaseFee (uint32)
	new(big.Int).SetUint64(uint64(rcMinimumBaseFee)).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 10: systemTxMaxGas (uint32)
	new(big.Int).SetUint64(uint64(rcSystemTxMaxGas)).FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 11: maximumBaseFee (uint128)
	maxUint128.FillBytes(calldata[offset : offset+32])
	offset += 32
	// slot 12: _batchInbox (address)
	copy(calldata[offset+12:offset+32], common.HexToAddress(batchInbox).Bytes())
	offset += 32
	// Addresses tuple (8 slots):
	// slot 13: l1CrossDomainMessenger
	copy(calldata[offset+12:offset+32], common.HexToAddress(l1CDMProxy).Bytes())
	offset += 32
	// slot 14: l1ERC721Bridge
	l1ERC721 := common.Address{}
	if l1ERC721BridgeProxy != "" {
		l1ERC721 = common.HexToAddress(l1ERC721BridgeProxy)
	}
	copy(calldata[offset+12:offset+32], l1ERC721.Bytes())
	offset += 32
	// slot 15: l1StandardBridge
	copy(calldata[offset+12:offset+32], common.HexToAddress(l1StandardBridgeProxy).Bytes())
	offset += 32
	// slot 16: disputeGameFactory
	copy(calldata[offset+12:offset+32], dgfAddr.Bytes())
	offset += 32
	// slot 17: optimismPortal
	copy(calldata[offset+12:offset+32], common.HexToAddress(optimismPortalProxy).Bytes())
	offset += 32
	// slot 18: optimismMintableERC20Factory
	mintableFactory := common.Address{}
	if optimismMintableERC20FactoryProxy != "" {
		mintableFactory = common.HexToAddress(optimismMintableERC20FactoryProxy)
	}
	copy(calldata[offset+12:offset+32], mintableFactory.Bytes())
	offset += 32
	// slot 19: gasPayingToken (always address(0) — not used)
	offset += 32
	// slot 20: nativeTokenAddress
	copy(calldata[offset+12:offset+32], nativeTokenAddr.Bytes())

	// Pre-flight simulation before spending L1 gas.
	if _, simErr := l1Client.CallContract(ctx, ethereum.CallMsg{From: adminAddr, To: &scAddr, Data: calldata}, nil); simErr != nil {
		return fmt.Errorf("SystemConfig initialize pre-flight failed: %w", simErr)
	}

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for SystemConfig init: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for SystemConfig init: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := ethtypes.NewTransaction(nonce, scAddr, big.NewInt(0), 500_000, gasPrice, calldata)
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign SystemConfig init tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send SystemConfig init tx: %w", err)
	}
	logger.Infof("SystemConfig.initialize() tx sent: %s (waiting for receipt...)", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for SystemConfig init tx receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("SystemConfig.initialize() tx reverted (tx: %s)", signedTx.Hash().Hex())
	}
	logger.Info("✅ SystemConfig initialized successfully")
	return nil
}

// initOptimismPortal calls initialize(L2OutputOracle, SystemConfig, SuperchainConfig)
// on the OptimismPortalProxy. Used in L2OO mode (EnableFraudProof=false).
// tokamak-deployer only calls upgrade(proxy, impl) and never invokes initialize().
// Idempotency: skipped if l2Oracle() already returns a non-zero address.
func initOptimismPortal(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	portalProxyAddr string,
	l2OracleProxyAddr string,
	systemConfigProxyAddr string,
	superchainConfigProxyAddr string,
	l1ChainID uint64,
) error {
	for _, pair := range []struct{ name, val string }{
		{"portalProxyAddr", portalProxyAddr},
		{"l2OracleProxyAddr", l2OracleProxyAddr},
		{"systemConfigProxyAddr", systemConfigProxyAddr},
		{"superchainConfigProxyAddr", superchainConfigProxyAddr},
	} {
		if pair.val == "" {
			return fmt.Errorf("initOptimismPortal: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for OptimismPortal init: %w", err)
	}
	defer l1Client.Close()

	portalAddr := common.HexToAddress(portalProxyAddr)

	// Idempotency guard: read l2Oracle(). If already non-zero, portal is initialized.
	l2OracleSelector := crypto.Keccak256([]byte("l2Oracle()"))[:4]
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &portalAddr, Data: l2OracleSelector}, nil); callErr == nil && len(result) >= 32 {
		existing := common.BytesToAddress(result[12:32])
		if existing != (common.Address{}) {
			logger.Infof("✅ OptimismPortal already initialized (l2Oracle=%s), skipping", existing.Hex())
			return nil
		}
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for OptimismPortal init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// Build initialize(address,address,address) calldata.
	// Arg order: _l2Oracle, _systemConfig, _superchainConfig.
	selector := crypto.Keccak256([]byte("initialize(address,address,address)"))[:4]
	calldata := make([]byte, 100)
	copy(calldata[0:4], selector)
	copy(calldata[16:36], common.HexToAddress(l2OracleProxyAddr).Bytes())          // slot 0: _l2Oracle
	copy(calldata[48:68], common.HexToAddress(systemConfigProxyAddr).Bytes())       // slot 1: _systemConfig
	copy(calldata[80:100], common.HexToAddress(superchainConfigProxyAddr).Bytes())  // slot 2: _superchainConfig

	if _, simErr := l1Client.CallContract(ctx, ethereum.CallMsg{From: adminAddr, To: &portalAddr, Data: calldata}, nil); simErr != nil {
		return fmt.Errorf("OptimismPortal initialize pre-flight failed: %w", simErr)
	}

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for OptimismPortal init: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for OptimismPortal init: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := ethtypes.NewTransaction(nonce, portalAddr, big.NewInt(0), 300_000, gasPrice, calldata)
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign OptimismPortal init tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send OptimismPortal init tx: %w", err)
	}
	logger.Infof("OptimismPortal.initialize() tx sent: %s (waiting for receipt...)", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for OptimismPortal init tx receipt: %w", err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("OptimismPortal.initialize() tx reverted (tx: %s)", signedTx.Hash().Hex())
	}
	logger.Info("✅ OptimismPortal initialized successfully")
	return nil
}

// initDisputeGameFactory initializes the DisputeGameFactory proxy and registers
// the FaultDisputeGame implementation. tokamak-deployer only calls
// ProxyAdmin.upgrade(proxy, impl) and never calls initialize() or setImplementation().
// Steps:
//  1. DGF.initialize(adminAddr)        — idempotency: owner() non-zero → skip
//  2. Deploy FaultDisputeGame impl     — idempotency: gameImpls(gameType) non-zero → skip all
//  3. DGF.setImplementation(gameType, fdgAddr)
func initDisputeGameFactory(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	dgfProxyAddr string,
	mipsAddr string,
	delayedWETHProxyAddr string,
	anchorStateRegistryProxyAddr string,
	l1ChainID uint64,
	l2ChainID uint64,
	gameType uint32,
	absolutePrestate string,
	maxGameDepth uint64,
	splitDepth uint64,
	clockExtension uint64,
	maxClockDuration uint64,
) error {
	for _, pair := range []struct{ name, val string }{
		{"dgfProxyAddr", dgfProxyAddr},
		{"mipsAddr", mipsAddr},
		{"delayedWETHProxyAddr", delayedWETHProxyAddr},
		{"anchorStateRegistryProxyAddr", anchorStateRegistryProxyAddr},
	} {
		if pair.val == "" {
			return fmt.Errorf("initDisputeGameFactory: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for DGF init: %w", err)
	}
	defer l1Client.Close()

	dgfAddr := common.HexToAddress(dgfProxyAddr)

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for DGF init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// Idempotency: check gameImpls(gameType). If already set, skip all steps.
	gameImplsSelector := crypto.Keccak256([]byte("gameImpls(uint32)"))[:4]
	gameTypeSlot := make([]byte, 32)
	binary.BigEndian.PutUint32(gameTypeSlot[28:32], gameType)
	gameImplsCalldata := append(gameImplsSelector, gameTypeSlot...)
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &dgfAddr, Data: gameImplsCalldata}, nil); callErr == nil && len(result) >= 32 {
		existing := common.BytesToAddress(result[12:32])
		if existing != (common.Address{}) {
			logger.Infow("✅ DisputeGameFactory impl already registered, skipping", "gameType", gameType, "impl", existing.Hex())
			return nil
		}
	}

	// ---- Step 1: DGF.initialize(owner) ----
	ownerSelector := crypto.Keccak256([]byte("owner()"))[:4]
	ownerResult, _ := l1Client.CallContract(ctx, ethereum.CallMsg{To: &dgfAddr, Data: ownerSelector}, nil)
	if len(ownerResult) < 32 || common.BytesToAddress(ownerResult[12:32]) == (common.Address{}) {
		initSelector := crypto.Keccak256([]byte("initialize(address)"))[:4]
		initCalldata := make([]byte, 36)
		copy(initCalldata[0:4], initSelector)
		copy(initCalldata[16:36], adminAddr.Bytes())

		nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
		if err != nil {
			return fmt.Errorf("failed to get nonce for DGF initialize: %w", err)
		}
		gasPrice, err := l1Client.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("failed to get gas price for DGF initialize: %w", err)
		}
		gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

		tx := ethtypes.NewTransaction(nonce, dgfAddr, big.NewInt(0), 300_000, gasPrice, initCalldata)
		signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privKey)
		if err != nil {
			return fmt.Errorf("failed to sign DGF initialize tx: %w", err)
		}
		if err := l1Client.SendTransaction(ctx, signedTx); err != nil {
			return fmt.Errorf("failed to send DGF initialize tx: %w", err)
		}
		logger.Infof("DisputeGameFactory.initialize() tx sent: %s", signedTx.Hash().Hex())
		receipt, err := bind.WaitMined(ctx, l1Client, signedTx)
		if err != nil {
			return fmt.Errorf("failed to wait for DGF initialize tx: %w", err)
		}
		if receipt.Status != ethtypes.ReceiptStatusSuccessful {
			return fmt.Errorf("DisputeGameFactory.initialize() reverted (tx: %s)", signedTx.Hash().Hex())
		}
		logger.Info("✅ DisputeGameFactory initialized")
	} else {
		logger.Infow("✅ DisputeGameFactory already initialized, skipping", "owner", common.BytesToAddress(ownerResult[12:32]).Hex())
	}

	// ---- Step 2: Deploy FaultDisputeGame implementation ----
	var fdgArtifact struct {
		Bytecode string `json:"bytecode"`
	}
	if err := json.Unmarshal(faultDisputeGameArtifact, &fdgArtifact); err != nil {
		return fmt.Errorf("failed to parse FaultDisputeGame artifact: %w", err)
	}
	fdgBytecodeHex := strings.TrimPrefix(fdgArtifact.Bytecode, "0x")
	fdgBytecode, err := hex.DecodeString(fdgBytecodeHex)
	if err != nil {
		return fmt.Errorf("failed to decode FaultDisputeGame bytecode: %w", err)
	}

	// ABI-encode FDG constructor args (10 slots × 32 bytes each = 320 bytes):
	// uint32 _gameType, bytes32 _absolutePrestate, uint256 _maxGameDepth, uint256 _splitDepth,
	// uint64 _clockExtension, uint64 _maxClockDuration, address _vm, address _weth,
	// address _anchorStateRegistry, uint256 _l2ChainId
	constructorArgs := make([]byte, 320)
	binary.BigEndian.PutUint32(constructorArgs[28:32], gameType) // slot 0: uint32 right-aligned
	prestateHex := strings.TrimPrefix(absolutePrestate, "0x")
	prestateBytes, err := hex.DecodeString(prestateHex)
	if err != nil || len(prestateBytes) != 32 {
		return fmt.Errorf("invalid absolutePrestate: must be a 32-byte hex string (got %q)", absolutePrestate)
	}
	copy(constructorArgs[32:64], prestateBytes)                                          // slot 1: bytes32
	new(big.Int).SetUint64(maxGameDepth).FillBytes(constructorArgs[64:96])               // slot 2: uint256
	new(big.Int).SetUint64(splitDepth).FillBytes(constructorArgs[96:128])                // slot 3: uint256
	new(big.Int).SetUint64(clockExtension).FillBytes(constructorArgs[128:160])           // slot 4: uint256
	new(big.Int).SetUint64(maxClockDuration).FillBytes(constructorArgs[160:192])         // slot 5: uint256
	copy(constructorArgs[204:224], common.HexToAddress(mipsAddr).Bytes())                // slot 6: address (12 zero pad + 20 bytes)
	copy(constructorArgs[236:256], common.HexToAddress(delayedWETHProxyAddr).Bytes())    // slot 7: address
	copy(constructorArgs[268:288], common.HexToAddress(anchorStateRegistryProxyAddr).Bytes()) // slot 8: address
	new(big.Int).SetUint64(l2ChainID).FillBytes(constructorArgs[288:320])                // slot 9: uint256

	deployData := append(fdgBytecode, constructorArgs...)

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for FDG deploy: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for FDG deploy: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	deployTx := ethtypes.NewContractCreation(nonce, big.NewInt(0), 8_000_000, gasPrice, deployData)
	signedDeployTx, err := ethtypes.SignTx(deployTx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign FDG deploy tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedDeployTx); err != nil {
		return fmt.Errorf("failed to send FDG deploy tx: %w", err)
	}
	logger.Infof("FaultDisputeGame impl deploy tx sent: %s (waiting for receipt...)", signedDeployTx.Hash().Hex())

	deployReceipt, err := bind.WaitMined(ctx, l1Client, signedDeployTx)
	if err != nil {
		return fmt.Errorf("failed to wait for FDG deploy tx: %w", err)
	}
	if deployReceipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("FaultDisputeGame deploy reverted (tx: %s)", signedDeployTx.Hash().Hex())
	}
	fdgImplAddr := deployReceipt.ContractAddress
	logger.Infof("FaultDisputeGame impl deployed at %s", fdgImplAddr.Hex())

	// ---- Step 3: DGF.setImplementation(gameType, fdgImplAddr) ----
	setImplSelector := crypto.Keccak256([]byte("setImplementation(uint32,address)"))[:4]
	setImplCalldata := make([]byte, 68)
	copy(setImplCalldata[0:4], setImplSelector)
	binary.BigEndian.PutUint32(setImplCalldata[32:36], gameType) // slot 0: uint32 right-aligned (selector at [0:4], slot at [4:36])
	copy(setImplCalldata[48:68], fdgImplAddr.Bytes())             // slot 1: address

	nonce, err = l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for DGF setImplementation: %w", err)
	}
	gasPrice, err = l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for DGF setImplementation: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	setImplTx := ethtypes.NewTransaction(nonce, dgfAddr, big.NewInt(0), 300_000, gasPrice, setImplCalldata)
	signedSetImplTx, err := ethtypes.SignTx(setImplTx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign DGF setImplementation tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedSetImplTx); err != nil {
		return fmt.Errorf("failed to send DGF setImplementation tx: %w", err)
	}
	logger.Infof("DisputeGameFactory.setImplementation() tx sent: %s", signedSetImplTx.Hash().Hex())

	setImplReceipt, err := bind.WaitMined(ctx, l1Client, signedSetImplTx)
	if err != nil {
		return fmt.Errorf("failed to wait for DGF setImplementation tx: %w", err)
	}
	if setImplReceipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("DisputeGameFactory.setImplementation() reverted (tx: %s)", signedSetImplTx.Hash().Hex())
	}
	logger.Infow("✅ DisputeGameFactory FaultDisputeGame impl registered", "gameType", gameType, "impl", fdgImplAddr.Hex())
	return nil
}

// initOptimismPortal2 upgrades the OptimismPortal proxy to OptimismPortal2
// and initializes it for DGF mode. tokamak-deployer deploys OptimismPortal (L2OO
// version) for ALL presets — DGF mode requires a new Portal2 impl + proxy upgrade.
// Steps:
//  1. Deploy OptimismPortal2 impl (constructor: proofMaturityDelay, dgfFinalityDelay)
//  2. ProxyAdmin.upgrade(portalProxy, portal2Impl)
//  3. Portal2.initialize(dgf, systemConfig, superchainConfig, respectedGameType)
//
// Idempotency: skipped entirely if disputeGameFactory() already returns non-zero.
func initOptimismPortal2(
	ctx context.Context,
	logger *zap.SugaredLogger,
	l1RPCURL string,
	adminPrivateKey string,
	portalProxyAddr string,
	proxyAdminAddr string,
	dgfProxyAddr string,
	systemConfigProxyAddr string,
	superchainConfigProxyAddr string,
	l1ChainID uint64,
	proofMaturityDelaySeconds uint64,
	disputeGameFinalityDelaySeconds uint64,
	respectedGameType uint32,
) error {
	for _, pair := range []struct{ name, val string }{
		{"portalProxyAddr", portalProxyAddr},
		{"proxyAdminAddr", proxyAdminAddr},
		{"dgfProxyAddr", dgfProxyAddr},
		{"systemConfigProxyAddr", systemConfigProxyAddr},
		{"superchainConfigProxyAddr", superchainConfigProxyAddr},
	} {
		if pair.val == "" {
			return fmt.Errorf("initOptimismPortal2: %s is empty", pair.name)
		}
	}

	l1Client, err := ethclient.DialContext(ctx, l1RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC for Portal2 init: %w", err)
	}
	defer l1Client.Close()

	portalAddr := common.HexToAddress(portalProxyAddr)

	// Idempotency: if disputeGameFactory() returns non-zero, Portal2 is already initialized.
	dgfSelector := crypto.Keccak256([]byte("disputeGameFactory()"))[:4]
	if result, callErr := l1Client.CallContract(ctx, ethereum.CallMsg{To: &portalAddr, Data: dgfSelector}, nil); callErr == nil && len(result) >= 32 {
		existing := common.BytesToAddress(result[12:32])
		if existing != (common.Address{}) {
			logger.Infow("✅ OptimismPortal2 already initialized, skipping", "dgf", existing.Hex())
			return nil
		}
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(adminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key for Portal2 init: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(l1ChainID))

	// Verify ProxyAdmin owner is an EOA (not Gnosis Safe).
	proxyAdminAddress := common.HexToAddress(proxyAdminAddr)
	ownerSel := crypto.Keccak256([]byte("owner()"))[:4]
	ownerResult, err := l1Client.CallContract(ctx, ethereum.CallMsg{To: &proxyAdminAddress, Data: ownerSel}, nil)
	if err != nil || len(ownerResult) < 32 {
		return fmt.Errorf("initOptimismPortal2: failed to call ProxyAdmin.owner(): %w", err)
	}
	proxyAdminOwner := common.BytesToAddress(ownerResult[12:32])
	if proxyAdminOwner != adminAddr {
		return fmt.Errorf("initOptimismPortal2: ProxyAdmin.owner() is %s but admin key is %s — wrong key",
			proxyAdminOwner.Hex(), adminAddr.Hex())
	}
	ownerCode, err := l1Client.CodeAt(ctx, proxyAdminOwner, nil)
	if err != nil {
		return fmt.Errorf("initOptimismPortal2: failed to check ProxyAdmin owner code size: %w", err)
	}
	if len(ownerCode) > 0 {
		return fmt.Errorf("initOptimismPortal2: ProxyAdmin owner %s is a contract (Gnosis Safe?) — EOA required for direct upgrade call",
			proxyAdminOwner.Hex())
	}

	// ---- Step 1: Deploy OptimismPortal2 implementation ----
	var portal2Artifact struct {
		Bytecode string `json:"bytecode"`
	}
	if err := json.Unmarshal(optimismPortal2Artifact, &portal2Artifact); err != nil {
		return fmt.Errorf("failed to parse OptimismPortal2 artifact: %w", err)
	}
	portal2BytecodeHex := strings.TrimPrefix(portal2Artifact.Bytecode, "0x")
	portal2Bytecode, err := hex.DecodeString(portal2BytecodeHex)
	if err != nil {
		return fmt.Errorf("failed to decode OptimismPortal2 bytecode: %w", err)
	}

	// ABI-encode constructor args (2 slots × 32 bytes = 64 bytes):
	// uint256 _proofMaturityDelaySeconds, uint256 _disputeGameFinalityDelaySeconds
	portal2ConstructorArgs := make([]byte, 64)
	new(big.Int).SetUint64(proofMaturityDelaySeconds).FillBytes(portal2ConstructorArgs[0:32])
	new(big.Int).SetUint64(disputeGameFinalityDelaySeconds).FillBytes(portal2ConstructorArgs[32:64])

	portal2DeployData := append(portal2Bytecode, portal2ConstructorArgs...)

	nonce, err := l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for Portal2 deploy: %w", err)
	}
	gasPrice, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for Portal2 deploy: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	portal2DeployTx := ethtypes.NewContractCreation(nonce, big.NewInt(0), 8_000_000, gasPrice, portal2DeployData)
	signedPortal2DeployTx, err := ethtypes.SignTx(portal2DeployTx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign Portal2 deploy tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedPortal2DeployTx); err != nil {
		return fmt.Errorf("failed to send Portal2 deploy tx: %w", err)
	}
	logger.Infof("OptimismPortal2 impl deploy tx sent: %s (waiting for receipt...)", signedPortal2DeployTx.Hash().Hex())

	portal2DeployReceipt, err := bind.WaitMined(ctx, l1Client, signedPortal2DeployTx)
	if err != nil {
		return fmt.Errorf("failed to wait for Portal2 deploy tx: %w", err)
	}
	if portal2DeployReceipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("OptimismPortal2 deploy reverted (tx: %s)", signedPortal2DeployTx.Hash().Hex())
	}
	portal2ImplAddr := portal2DeployReceipt.ContractAddress
	logger.Infof("OptimismPortal2 impl deployed at %s", portal2ImplAddr.Hex())

	// ---- Step 2: ProxyAdmin.upgrade(portalProxy, portal2Impl) ----
	proxyAdminContractAddr := common.HexToAddress(proxyAdminAddr)
	upgradeSelector := crypto.Keccak256([]byte("upgrade(address,address)"))[:4]
	upgradeCalldata := make([]byte, 68)
	copy(upgradeCalldata[0:4], upgradeSelector)
	copy(upgradeCalldata[16:36], portalAddr.Bytes())          // slot 0: proxy
	copy(upgradeCalldata[48:68], portal2ImplAddr.Bytes())     // slot 1: impl

	nonce, err = l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for ProxyAdmin upgrade: %w", err)
	}
	gasPrice, err = l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for ProxyAdmin upgrade: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	upgradeTx := ethtypes.NewTransaction(nonce, proxyAdminContractAddr, big.NewInt(0), 300_000, gasPrice, upgradeCalldata)
	signedUpgradeTx, err := ethtypes.SignTx(upgradeTx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign ProxyAdmin upgrade tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedUpgradeTx); err != nil {
		return fmt.Errorf("failed to send ProxyAdmin upgrade tx: %w", err)
	}
	logger.Infof("ProxyAdmin.upgrade(portal → Portal2) tx sent: %s", signedUpgradeTx.Hash().Hex())

	upgradeReceipt, err := bind.WaitMined(ctx, l1Client, signedUpgradeTx)
	if err != nil {
		return fmt.Errorf("failed to wait for ProxyAdmin upgrade tx: %w", err)
	}
	if upgradeReceipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("ProxyAdmin.upgrade() reverted (tx: %s)", signedUpgradeTx.Hash().Hex())
	}
	logger.Info("✅ Portal proxy upgraded to OptimismPortal2")

	// ---- Step 3: Portal2.initialize(dgf, systemConfig, superchainConfig, respectedGameType) ----
	initSelector := crypto.Keccak256([]byte("initialize(address,address,address,uint32)"))[:4]
	initCalldata := make([]byte, 132)
	copy(initCalldata[0:4], initSelector)
	copy(initCalldata[16:36], common.HexToAddress(dgfProxyAddr).Bytes())            // slot 0: _disputeGameFactory
	copy(initCalldata[48:68], common.HexToAddress(systemConfigProxyAddr).Bytes())   // slot 1: _systemConfig
	copy(initCalldata[80:100], common.HexToAddress(superchainConfigProxyAddr).Bytes()) // slot 2: _superchainConfig
	binary.BigEndian.PutUint32(initCalldata[128:132], respectedGameType)            // slot 3: uint32 right-aligned

	nonce, err = l1Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce for Portal2 initialize: %w", err)
	}
	gasPrice, err = l1Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price for Portal2 initialize: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	initTx := ethtypes.NewTransaction(nonce, portalAddr, big.NewInt(0), 400_000, gasPrice, initCalldata)
	signedInitTx, err := ethtypes.SignTx(initTx, ethtypes.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return fmt.Errorf("failed to sign Portal2 initialize tx: %w", err)
	}
	if err := l1Client.SendTransaction(ctx, signedInitTx); err != nil {
		return fmt.Errorf("failed to send Portal2 initialize tx: %w", err)
	}
	logger.Infof("OptimismPortal2.initialize() tx sent: %s (waiting for receipt...)", signedInitTx.Hash().Hex())

	initReceipt, err := bind.WaitMined(ctx, l1Client, signedInitTx)
	if err != nil {
		return fmt.Errorf("failed to wait for Portal2 initialize tx: %w", err)
	}
	if initReceipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("OptimismPortal2.initialize() reverted (tx: %s)", signedInitTx.Hash().Hex())
	}
	logger.Info("✅ OptimismPortal2 initialized successfully")
	return nil
}

// installPresetModules installs all modules enabled for the configured preset.
// Modules that require user input (blockExplorer, crossTrade) are skipped with
// a guidance log; they must be installed manually via 'trh install <plugin>'.
// Failures are logged but do not abort the deployment — already-installed
// modules are not rolled back.
func (t *ThanosStack) installPresetModules(ctx context.Context) error {
	preset := t.deployConfig.Preset
	modules := constants.PresetModules[preset]
	t.logger.Infof("🔧 Installing preset modules for preset=%s", preset)

	var installErr error

	if modules["uptimeService"] {
		t.logger.Info("  ↳ uptime-service")
		cfg, err := t.GetUptimeServiceConfig(ctx)
		if err != nil {
			t.logger.Errorw("Failed to get uptime-service config", "err", err)
			installErr = err
		} else if _, err := t.InstallUptimeService(ctx, cfg); err != nil {
			t.logger.Errorw("Failed to install uptime-service", "err", err)
			installErr = err
		}
	}

	if modules["monitoring"] {
		t.logger.Info("  ↳ monitoring (auto-config: random admin password, alerts disabled)")
		monitoringInput, err := BuildDefaultMonitoringInput()
		if err != nil {
			t.logger.Errorw("Failed to build default monitoring config", "err", err)
			installErr = err
		} else {
			monitoringCfg, err := t.GetMonitoringConfig(ctx, monitoringInput.AdminPassword, monitoringInput.AlertManager, monitoringInput.LoggingEnabled)
			if err != nil {
				t.logger.Errorw("Failed to get monitoring config", "err", err)
				installErr = err
			} else if _, err := t.InstallMonitoring(ctx, monitoringCfg); err != nil {
				t.logger.Errorw("Failed to install monitoring", "err", err)
				installErr = err
			}
		}
	}

	if modules["drb"] {
		t.logger.Info("  ↳ drb-vrf")
		if err := t.InstallDRB(ctx); err != nil {
			t.logger.Errorw("Failed to install DRB VRF node", "err", err)
			installErr = err
		}
	}

	if modules["aaPaymaster"] {
		if t.deployConfig.FeeToken != constants.FeeTokenTON {
			t.logger.Info("  ↳ aa-paymaster")
			if err := t.setupAAPaymaster(ctx); err != nil {
				t.logger.Errorw("Failed to set up AA Paymaster", "err", err)
				installErr = err
			}
		} else {
			t.logger.Info("  ↳ aa-paymaster skipped: TON is the native fee token; AA paymaster is not required")
		}
	}

	// blockExplorer requires external API keys — cannot auto-install.
	if modules["blockExplorer"] {
		t.logger.Info("ℹ️  Block Explorer is included in your preset. Run 'trh install block-explorer' to configure and deploy it.")
	}

	// crossTrade: auto-install on AWS (K8s available); manual on local (separate Docker Compose).
	if modules["crossTrade"] {
		if t.deployConfig.K8s != nil {
			t.logger.Info("  ↳ cross-trade (AWS auto-install)")
			if err := t.autoInstallCrossTradeAWS(ctx); err != nil {
				t.logger.Errorw("Failed to auto-install CrossTrade", "err", err)
				installErr = err
			}
		} else {
			t.logger.Info("ℹ️  Cross-Chain Trade is included in your preset. Run 'trh install cross-trade' to configure and deploy it.")
		}
	}

	return installErr
}
