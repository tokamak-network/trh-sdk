package thanos

import (
	"context"
	"os"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

// --- Bridge local tests ---

func TestInstallBridge_LocalNilK8s(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.LocalTestnet,
		deployConfig: &types.Config{},
	}
	_, err := s.InstallBridge(context.Background())
	if err == nil {
		t.Fatal("expected error for nil K8s config")
	}
}

func TestUninstallBridge_LocalSkipsAWS(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.LocalTestnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil, // no AWS — should NOT error for local
		},
	}
	// Will fail on helm list (no cluster) but should NOT fail on AWS check
	err := s.UninstallBridge(context.Background())
	if err != nil && err.Error() == "AWS configuration is not set. Please run the deploy command first" {
		t.Fatal("local uninstall should not require AWS config")
	}
}

func TestUninstallBridge_CloudRequiresAWS(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.Testnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}
	err := s.UninstallBridge(context.Background())
	if err == nil {
		t.Fatal("cloud uninstall should require AWS config")
	}
}

// --- Block Explorer local tests ---

func TestUninstallBlockExplorer_LocalSkipsAWS(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.LocalTestnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}
	err := s.UninstallBlockExplorer(context.Background())
	if err != nil && err.Error() == "AWS configuration is not set" {
		t.Fatal("local uninstall should not require AWS config")
	}
}

func TestUninstallBlockExplorer_CloudRequiresAWS(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
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

func TestGetBlockExplorerURL_LocalReturnsLocalhost(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.LocalTestnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
		},
	}
	url, err := s.GetBlockExplorerURL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:4000" {
		t.Errorf("expected http://localhost:4000, got %s", url)
	}
}

// --- Monitoring local tests ---

func TestGetMonitoringConfig_LocalSkipsEFS(t *testing.T) {
	dir := t.TempDir()
	// Create chart dir so Stat passes
	chartDir := dir + "/tokamak-thanos-stack/charts/monitoring"
	if err := makeDir(chartDir); err != nil {
		t.Fatal(err)
	}

	s := &ThanosStack{
		logger:         zap.NewNop().Sugar(),
		network:        constants.LocalTestnet,
		deploymentPath: dir,
		deployConfig: &types.Config{
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
			ChainName: "test",
			L1RPCURL:  "https://sepolia.example.com",
			AWS:       nil,
		},
	}

	config, err := s.GetMonitoringConfig(context.Background(), "admin123", types.AlertManagerConfig{}, true)
	if err != nil {
		// getServiceNames needs kubectl — skip if cluster unavailable
		t.Skipf("skipping: requires kubectl cluster access: %v", err)
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

func TestGenerateValuesFile_LocalSkipsAWS(t *testing.T) {
	dir := t.TempDir()
	s := &ThanosStack{
		logger:         zap.NewNop().Sugar(),
		network:        constants.LocalTestnet,
		deploymentPath: dir,
		deployConfig: &types.Config{
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
			ChainName: "test",
			L1RPCURL:  "https://sepolia.example.com",
			AWS:       nil,
		},
	}

	config := &types.MonitoringConfig{
		Namespace:       "monitoring",
		HelmReleaseName: "monitoring-test",
		L1RpcUrl:        "https://sepolia.example.com",
		ChainName:       "test",
		ValuesFilePath:  dir + "/values.yaml",
	}

	err := s.generateValuesFile(config)
	if err != nil {
		t.Fatalf("local generateValuesFile should not require AWS, got: %v", err)
	}
}

func TestGenerateValuesFile_CloudRequiresAWS(t *testing.T) {
	s := &ThanosStack{
		logger:  zap.NewNop().Sugar(),
		network: constants.Testnet,
		deployConfig: &types.Config{
			K8s: &types.K8sConfig{Namespace: "test-ns"},
			AWS: nil,
		},
	}

	config := &types.MonitoringConfig{
		ValuesFilePath: "/tmp/values.yaml",
	}

	err := s.generateValuesFile(config)
	if err == nil {
		t.Fatal("cloud generateValuesFile should require AWS config")
	}
}

// helper
func makeDir(path string) error {
	return os.MkdirAll(path, 0755)
}
