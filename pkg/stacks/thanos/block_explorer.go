package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) InstallBlockExplorer(ctx context.Context, inputs *InstallBlockExplorerInput) (string, error) {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	if inputs == nil {
		t.logger.Error("inputs are not set. Please provide the inputs")
		return "", fmt.Errorf("inputs are not set. Please provide the inputs")
	}

	if err := inputs.Validate(ctx); err != nil {
		t.logger.Error("Error validating inputs", "err", err)
		return "", err
	}

	namespace := t.deployConfig.K8s.Namespace

	var vpcId string
	if !t.isLocal() {
		if t.deployConfig.AWS == nil {
			return "", fmt.Errorf("AWS configuration is not set")
		}
		vpcId = t.deployConfig.AWS.VpcID
	}

	blockExplorerPods, err := utils.GetPodsByName(ctx, namespace, "block-explorer")
	if err != nil {
		t.logger.Error("Error to get block explorer pods", "err", err)
		return "", err
	}
	if len(blockExplorerPods) > 0 {
		t.logger.Info("Block Explorer is running")
		if t.isLocal() {
			return "http://localhost:4000", nil
		}
		var blockExplorerURL string
		for {
			k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, "block-explorer")
			if err != nil {
				return "", err
			}
			if len(k8sIngresses) > 0 {
				blockExplorerURL = "http://" + k8sIngresses[0]
				break
			}
			time.Sleep(15 * time.Second)
		}
		return blockExplorerURL, nil
	}

	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		t.logger.Error("Error cloning repository", "err", err)
		return "", err
	}

	t.logger.Info("Installing a block explorer component...")

	var (
		databasePassword     = inputs.DatabasePassword
		databaseUserName     = inputs.DatabaseUsername
		coinmarketcapKey     = inputs.CoinmarketcapKey
		coinmarketcapTokenID = inputs.CoinmarketcapTokenID
		walletConnectID      = inputs.WalletConnectProjectID
	)

	var rdsConnectionUrl string

	if t.isLocal() {
		// Local: deploy a PostgreSQL pod instead of AWS RDS
		rdsConnectionUrl, err = t.deployLocalPostgres(ctx, namespace, databaseUserName, databasePassword)
		if err != nil {
			return "", fmt.Errorf("failed to deploy local PostgreSQL: %w", err)
		}
	} else {
		// Cloud: Terraform RDS provisioning
		err = makeBlockExplorerEnvs(
			fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath),
			".envrc",
			types.BlockExplorerEnvs{
				BlockExplorerDatabasePassword: databasePassword,
				BlockExplorerDatabaseUserName: databaseUserName,
				BlockExplorerDatabaseName:     "blockscout",
				VpcId:                         vpcId,
				AwsRegion:                     t.deployConfig.AWS.Region,
			},
		)
		if err != nil {
			return "", err
		}

		err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
			"-c",
			fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
			source .envrc && cd block-explorer &&
			terraform init && terraform plan && terraform apply -auto-approve
			`, t.deploymentPath),
		}...)
		if err != nil {
			return "", err
		}

		rdsOutput, err := utils.ExecuteCommand(ctx, "bash", []string{
			"-c",
			fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
			source .envrc && cd block-explorer &&
			terraform output -json rds_connection_url`, t.deploymentPath),
		}...)
		if err != nil {
			return "", fmt.Errorf("failed to get terraform output: %w", err)
		}
		rdsConnectionUrl = strings.Trim(rdsOutput, `"`)
	}

	chainReleaseName, err := utils.FilterHelmReleases(ctx, namespace, namespace)
	if err != nil {
		return "", err
	}
	if len(chainReleaseName) == 0 {
		return "", fmt.Errorf("no helm releases found")
	}
	releaseName := chainReleaseName[0]

	var opGethSVC string
	for {
		k8sSvc, err := utils.GetServiceNames(ctx, namespace, "op-geth")
		if err != nil {
			t.logger.Error("Error retrieving svc", "err", err, "details", k8sSvc)
			return "", err
		}

		if len(k8sSvc) > 0 {
			opGethSVC = k8sSvc[0]
			break
		}

		time.Sleep(15 * time.Second)
	}

	var opGethPublicUrl string
	if t.isLocal() {
		opGethPublicUrl = fmt.Sprintf("%s.%s.svc.cluster.local", opGethSVC, namespace)
	} else {
		for {
			k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, "op-geth")
			if err != nil {
				return "", err
			}
			if len(k8sIngresses) > 0 {
				opGethPublicUrl = k8sIngresses[0]
				break
			}
			time.Sleep(15 * time.Second)
		}
	}

	// generate the helm chart value file
	envValues := fmt.Sprintf(`
		export stack_deployments_path=%s
		export stack_l1_rpc_url=%s
		export stack_chain_id=%d
		export stack_coinmarketcap_api_key=%s
		export stack_coinmarketcap_coin_id=%s
		export stack_helm_release_name=%s
		export stack_network_name=%s
		export stack_wallet_connect_project_id=%s
		export rollup_path=%s
		export rds_connection_url=%s
		export l1_beacon_rpc_url=%s
		export op_geth_svc=%s
		export op_geth_public_url=%s
		export next_public_rollup_l1_base_url=%s
		export enable_fault_proof=%t
		export stack_nativetoken_name=%s
		export stack_nativetoken_symbol=%s
		`,
		t.deployConfig.DeploymentFilePath,
		t.deployConfig.L1RPCURL,
		t.deployConfig.L2ChainID,
		coinmarketcapKey,
		coinmarketcapTokenID,
		releaseName,
		t.deployConfig.ChainName,
		walletConnectID,
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", t.deploymentPath),
		rdsConnectionUrl,
		t.deployConfig.L1BeaconURL,
		opGethSVC,
		opGethPublicUrl,
		t.deployConfig.NextPublicRollupL1BaseUrl,
		t.deployConfig.EnableFraudProof,
		constants.GetFeeTokenConfig(t.deployConfig.FeeToken, t.deployConfig.L1ChainID).Name,
		constants.GetFeeTokenConfig(t.deployConfig.FeeToken, t.deployConfig.L1ChainID).Symbol,
	)
	_, err = utils.ExecuteCommand(ctx,
		"bash",
		"-c",
		fmt.Sprintf("cd %s/tokamak-thanos-stack/charts/blockscout-stack && echo '%s' > .env", t.deploymentPath, envValues),
	)
	if err != nil {
		t.logger.Error("❌ Make .env file failed!")
		return "", err
	}

	_, err = utils.ExecuteCommand(ctx,
		"bash",
		"-c",
		fmt.Sprintf("cd %s/tokamak-thanos-stack/charts/blockscout-stack && source .env && bash ./scripts/generate-blockscout.sh", t.deploymentPath),
	)
	if err != nil {
		t.logger.Error("❌ Make helm values failed!")
		return "", err
	}

	// Install backend first
	blockExplorerBackendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-be", time.Now().Unix())
	fileValue := fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", t.deploymentPath)
	_, err = t.helm(ctx, []string{
		"install",
		blockExplorerBackendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", t.deploymentPath),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		t.logger.Error("Error installing block explorer backend component", "err", err)
		return "", err
	}
	t.logger.Info("✅ Install block explorer backend component successfully")

	// Get backend URL for frontend configuration
	var blockExplorerUrl string
	if t.isLocal() {
		blockExplorerUrl = "localhost:4000"
		t.logger.Infof("Local block explorer. Access via: kubectl port-forward -n %s svc/%s 4000:4000", namespace, blockExplorerBackendReleaseName)
	} else {
		for {
			k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, blockExplorerBackendReleaseName)
			if err != nil {
				return "", err
			}
			if len(k8sIngresses) > 0 {
				blockExplorerUrl = k8sIngresses[0]
				break
			}
			time.Sleep(15 * time.Second)
		}
	}

	// update the values file
	err = utils.UpdateYAMLField(fileValue, "blockscout.enabled", false)
	if err != nil {
		t.logger.Error("Error updating blockscout.enabled field", "err", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.enabled", true)
	if err != nil {
		t.logger.Error("Error updating frontend.enabled field", "err", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_API_HOST", blockExplorerUrl)
	if err != nil {
		t.logger.Error("Error updating NEXT_PUBLIC_API_HOST field", "err", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_APP_HOST", blockExplorerUrl)
	if err != nil {
		t.logger.Error("Error updating NEXT_PUBLIC_APP_HOST field", "err", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.ingress.hostname", blockExplorerUrl)
	if err != nil {
		t.logger.Error("Error updating frontend.ingress.hostname field", "err", err)
		return "", err
	}

	if t.deployConfig.NextPublicRollupL1BaseUrl != "" {
		err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_ROLLUP_L1_BASE_URL", t.deployConfig.NextPublicRollupL1BaseUrl)
		if err != nil {
			t.logger.Error("Error updating NEXT_PUBLIC_ROLLUP_L1_BASE_URL field", "err", err)
			return "", err
		}
	}

	blockExplorerFrontendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-fe", time.Now().Unix())
	_, err = t.helm(ctx, []string{
		"install",
		blockExplorerFrontendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", t.deploymentPath),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		t.logger.Error("Error installing block explorer front-end component", "err", err)
		return "", err
	}

	t.logger.Infof("✅ Block Explorer frontend component installed successfully. Accessible at: %s", fmt.Sprintf("http://%s", blockExplorerUrl))

	return "http://" + blockExplorerUrl, nil
}

func (t *ThanosStack) UninstallBlockExplorer(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set")
	}
	if !t.isLocal() && t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set")
	}

	namespace := t.deployConfig.K8s.Namespace

	// 1. Uninstall helm charts
	releases, err := utils.FilterHelmReleases(ctx, namespace, "block-explorer")
	if err != nil {
		return err
	}
	for _, release := range releases {
		if _, err := t.helm(ctx, "uninstall", release, "--namespace", namespace); err != nil {
			t.logger.Error("❌ Error uninstalling block-explorer helm chart", "err", err)
			return err
		}
	}

	// 2. Clean up database
	if t.isLocal() {
		// Delete local PostgreSQL
		t.kubectl(ctx, "delete", "deployment", "blockscout-postgres", "-n", namespace, "--ignore-not-found=true")
		t.kubectl(ctx, "delete", "service", "blockscout-postgres", "-n", namespace, "--ignore-not-found=true")
	} else {
		if err := t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/block-explorer", t.deploymentPath)); err != nil {
			t.logger.Error("❌ Error running block-explorer terraform destroy", "err", err)
			return err
		}
	}

	t.logger.Info("✅ Uninstall block explorer components successfully")
	return nil
}

// deployLocalPostgres creates a PostgreSQL pod + service for local block explorer.
func (t *ThanosStack) deployLocalPostgres(ctx context.Context, namespace, username, password string) (string, error) {
	t.logger.Info("Deploying local PostgreSQL for block explorer...")

	manifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: blockscout-postgres
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: blockscout-postgres
  template:
    metadata:
      labels:
        app: blockscout-postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          value: "%s"
        - name: POSTGRES_PASSWORD
          value: "%s"
        - name: POSTGRES_DB
          value: "blockscout"
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: blockscout-postgres
  namespace: %s
spec:
  selector:
    app: blockscout-postgres
  ports:
  - port: 5432
    targetPort: 5432
`, namespace, username, password, namespace)

	if err := t.kubectlApplyManifest(ctx, manifest); err != nil {
		return "", fmt.Errorf("apply postgres manifest: %w", err)
	}

	// Wait for postgres pod
	t.logger.Info("Waiting for PostgreSQL to be ready...")
	for i := 0; i < 60; i++ {
		pods, err := utils.GetPodsByName(ctx, namespace, "blockscout-postgres")
		if err == nil && len(pods) > 0 {
			t.logger.Info("✅ Local PostgreSQL is ready")
			return fmt.Sprintf("postgresql://%s:%s@blockscout-postgres.%s.svc.cluster.local:5432/blockscout", username, password, namespace), nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return "", fmt.Errorf("PostgreSQL did not become ready")
}

func (t *ThanosStack) GetBlockExplorerURL(ctx context.Context) (string, error) {
	if t.deployConfig.K8s == nil {
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, "block-explorer")
	if err != nil {
		t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
		return "", err
	}

	if len(k8sIngresses) == 0 {
		t.logger.Error("block explorer ingress is not found")
		return "", fmt.Errorf("block explorer ingress is not found")
	}

	blockExplorerURL := "http://" + k8sIngresses[0]

	return blockExplorerURL, nil
}
