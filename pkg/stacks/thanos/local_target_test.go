package thanos

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// --------------- isLocal() ---------------

func TestIsLocal_LocalTestnet(t *testing.T) {
	s := &ThanosStack{network: constants.LocalTestnet}
	if !s.isLocal() {
		t.Fatal("expected isLocal()=true for LocalTestnet")
	}
}

func TestIsLocal_Testnet(t *testing.T) {
	s := &ThanosStack{network: constants.Testnet}
	if s.isLocal() {
		t.Fatal("expected isLocal()=false for Testnet")
	}
}

func TestIsLocal_Mainnet(t *testing.T) {
	s := &ThanosStack{network: constants.Mainnet}
	if s.isLocal() {
		t.Fatal("expected isLocal()=false for Mainnet")
	}
}

// --------------- InstallBridge local branch ---------------

func TestInstallBridge_NilK8sConfig(t *testing.T) {
	s := &ThanosStack{
		logger:       noopLogger(),
		network:      constants.LocalTestnet,
		deployConfig: &types.Config{},
	}
	_, err := s.InstallBridge(context.Background())
	if err == nil {
		t.Fatal("expected error for nil K8s config")
	}
}

func TestInstallBridge_LocalIngressDisabled(t *testing.T) {
	// Verify that for local deployment, the generated values file has ingress disabled.
	dir := t.TempDir()
	configFileDir := filepath.Join(dir, "tokamak-thanos-stack", "terraform", "thanos-stack")
	os.MkdirAll(configFileDir, 0755)

	// Create a minimal chart directory with Chart.yaml
	chartDir := filepath.Join(dir, "tokamak-thanos-stack", "charts", "op-bridge", "templates")
	os.MkdirAll(chartDir, 0755)
	os.WriteFile(filepath.Join(dir, "tokamak-thanos-stack", "charts", "op-bridge", "Chart.yaml"),
		[]byte("apiVersion: v2\nname: op-bridge\nversion: 0.1.0\n"), 0644)

	// Create a deployment config JSON for ReadDeployementConfigFromJSONFile
	deploymentsDir := filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments")
	os.MkdirAll(deploymentsDir, 0755)
	os.WriteFile(filepath.Join(deploymentsDir, "11155111-deploy.json"),
		[]byte(`{"L1StandardBridgeProxy":"0x1","AddressManager":"0x2","L1CrossDomainMessengerProxy":"0x3","OptimismPortalProxy":"0x4","L2OutputOracleProxy":"0x5","L1UsdcBridgeProxy":"0x6","DisputeGameFactoryProxy":"0x7"}`),
		0644)

	hr := &mock.HelmRunner{}
	hr.OnUpgradeWithFiles = func(_ context.Context, release, chart, namespace string, files []string) error {
		// Verify the values file contains ingress disabled
		if len(files) == 0 {
			t.Fatal("expected value files")
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("failed to read values file: %v", err)
		}
		content := string(data)
		if contains(content, "className: alb") {
			t.Error("local deployment should NOT have ALB ingress class")
		}
		if !contains(content, "enabled: false") {
			t.Error("local deployment should have ingress disabled")
		}
		return nil
	}

	s := &ThanosStack{
		logger:         noopLogger(),
		network:        constants.LocalTestnet,
		deploymentPath: dir,
		helmRunner:     hr,
		deployConfig: &types.Config{
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
			L1ChainID: constants.EthereumSepoliaChainID,
			L1RPCURL:  "https://sepolia.example.com",
			L2ChainID: 12345,
			ChainName: "test",
			ChainConfiguration: &types.ChainConfiguration{
				BatchSubmissionFrequency: 600,
				L1BlockTime:             12,
				L2BlockTime:             2,
				OutputRootFrequency:     600,
				ChallengePeriod:         86400,
			},
		},
	}

	// InstallBridge will try loadImageToKind (docker pull) which will fail in test env.
	// That's a non-fatal warning, so it continues to helm install.
	url, err := s.InstallBridge(context.Background())
	if err != nil {
		// Accept errors from docker/kind not being available in CI
		t.Skipf("skipping: bridge install requires docker/kind: %v", err)
	}
	if url != "http://localhost:3100" {
		t.Errorf("expected localhost:3100 URL, got %s", url)
	}
}

// --------------- UninstallBridge local branch ---------------

func TestUninstallBridge_LocalSkipsAWSCheck(t *testing.T) {
	hr := &mock.HelmRunner{}
	// Return empty list so no uninstall is attempted
	hr.OnList = func(_ context.Context, namespace string) ([]string, error) {
		return []string{}, nil
	}

	s := &ThanosStack{
		logger:     noopLogger(),
		network:    constants.LocalTestnet,
		helmRunner: hr,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil, // No AWS config — should NOT error for local
		},
	}

	err := s.UninstallBridge(context.Background())
	if err != nil {
		t.Fatalf("local uninstall should not require AWS config, got: %v", err)
	}
}

func TestUninstallBridge_CloudRequiresAWSConfig(t *testing.T) {
	s := &ThanosStack{
		logger:  noopLogger(),
		network: constants.Testnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil, // No AWS config — should error for cloud
		},
	}

	err := s.UninstallBridge(context.Background())
	if err == nil {
		t.Fatal("cloud uninstall should require AWS config")
	}
}

// --------------- UninstallBlockExplorer local branch ---------------

func TestUninstallBlockExplorer_LocalSkipsAWSCheck(t *testing.T) {
	hr := &mock.HelmRunner{}
	hr.OnList = func(_ context.Context, namespace string) ([]string, error) {
		return []string{}, nil
	}
	kr := &mock.K8sRunner{}

	s := &ThanosStack{
		logger:     noopLogger(),
		network:    constants.LocalTestnet,
		helmRunner: hr,
		k8sRunner:  kr,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}

	err := s.UninstallBlockExplorer(context.Background())
	if err != nil {
		t.Fatalf("local uninstall should not require AWS config, got: %v", err)
	}
}

func TestUninstallBlockExplorer_CloudRequiresAWSConfig(t *testing.T) {
	s := &ThanosStack{
		logger:  noopLogger(),
		network: constants.Testnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}

	err := s.UninstallBlockExplorer(context.Background())
	if err == nil {
		t.Fatal("cloud uninstall should require AWS config")
	}
}

// --------------- Monitoring local branch ---------------

func TestGetMonitoringConfig_LocalSkipsEFS(t *testing.T) {
	dir := t.TempDir()
	chartDir := filepath.Join(dir, "tokamak-thanos-stack", "charts", "monitoring")
	os.MkdirAll(chartDir, 0755)

	kr := &mock.K8sRunner{}
	// Return empty service list
	kr.OnList = func(_ context.Context, resource, namespace, labelSelector string) ([]byte, error) {
		return []byte(`{"items":[]}`), nil
	}

	s := &ThanosStack{
		logger:         noopLogger(),
		network:        constants.LocalTestnet,
		deploymentPath: dir,
		k8sRunner:      kr,
		deployConfig: &types.Config{
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
			ChainName: "test",
			L1RPCURL:  "https://sepolia.example.com",
			AWS:       nil, // No AWS — should not error for local
		},
	}

	config, err := s.GetMonitoringConfig(context.Background(), "admin123", types.AlertManagerConfig{}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.EnablePersistence {
		t.Error("local monitoring should have persistence disabled")
	}
	if config.EFSFileSystemId != "" {
		t.Error("local monitoring should have empty EFS filesystem ID")
	}
	if config.LoggingEnabled {
		t.Error("local monitoring should have logging disabled (no CloudWatch)")
	}
}

func TestGenerateValuesFile_LocalSkipsAWSCheck(t *testing.T) {
	dir := t.TempDir()
	s := &ThanosStack{
		logger:         noopLogger(),
		network:        constants.LocalTestnet,
		deploymentPath: dir,
		deployConfig: &types.Config{
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
			ChainName: "test",
			L1RPCURL:  "https://sepolia.example.com",
			AWS:       nil, // No AWS — should not error for local
		},
	}

	config := &types.MonitoringConfig{
		Namespace:       "monitoring",
		HelmReleaseName: "monitoring-test",
		L1RpcUrl:        "https://sepolia.example.com",
		ChainName:       "test",
		ValuesFilePath:  filepath.Join(dir, "values.yaml"),
	}

	err := s.generateValuesFile(config)
	if err != nil {
		t.Fatalf("local generateValuesFile should not require AWS, got: %v", err)
	}

	// Verify values file was written
	data, err := os.ReadFile(config.ValuesFilePath)
	if err != nil {
		t.Fatalf("values file not written: %v", err)
	}
	content := string(data)
	if contains(content, "efsFileSystemId") {
		t.Error("local values should not contain efsFileSystemId")
	}
	if contains(content, "awsRegion") {
		t.Error("local values should not contain awsRegion")
	}
}

func TestGenerateValuesFile_CloudRequiresAWSConfig(t *testing.T) {
	s := &ThanosStack{
		logger:  noopLogger(),
		network: constants.Testnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}

	config := &types.MonitoringConfig{
		ValuesFilePath: filepath.Join(t.TempDir(), "values.yaml"),
	}

	err := s.generateValuesFile(config)
	if err == nil {
		t.Fatal("cloud generateValuesFile should require AWS config")
	}
}

// --------------- BuildOnly ---------------

func TestDeployContractsInput_BuildOnlyField(t *testing.T) {
	// Verify BuildOnly field exists and can be set
	input := &DeployContractsInput{
		L1RPCurl:  "https://sepolia.example.com",
		BuildOnly: true,
	}
	if !input.BuildOnly {
		t.Fatal("BuildOnly field should be true")
	}

	input2 := &DeployContractsInput{
		L1RPCurl: "https://sepolia.example.com",
	}
	if input2.BuildOnly {
		t.Fatal("BuildOnly should default to false")
	}
}

// --------------- Helper ---------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
