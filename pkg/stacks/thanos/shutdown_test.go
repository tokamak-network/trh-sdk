package thanos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

// TestReadBedrockDeployConfigTemplate_NewPath verifies that readBedrockDeployConfigTemplate
// loads deploy-config.json from the new tokamak-deployer location
// (<deploymentPath>/deploy-config.json) when the legacy Foundry path
// (<bedrockPath>/scripts/deploy-config.json) is absent.
// Regression test for Bug #7 (drb-local-compose-path-template-bugs).
func TestReadBedrockDeployConfigTemplate_NewPath(t *testing.T) {
	dir := t.TempDir()

	// Simulate the new tokamak-deployer layout: config at root of deploymentPath.
	// No legacy scripts/deploy-config.json exists. getBedrockPath will still
	// succeed because we create the contracts-bedrock directory (fault-proof
	// ON flow still clones tokamak-thanos).
	if err := os.MkdirAll(filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"), 0755); err != nil {
		t.Fatal(err)
	}

	tmpl := types.DeployConfigTemplate{L1ChainID: 11155111, L2ChainID: 424242}
	data, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deploy-config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	stack := &ThanosStack{
		deploymentPath: dir,
		logger:         zap.NewNop().Sugar(),
	}

	got, err := stack.readBedrockDeployConfigTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.L1ChainID != tmpl.L1ChainID || got.L2ChainID != tmpl.L2ChainID {
		t.Errorf("mismatch: got L1=%d L2=%d, want L1=%d L2=%d",
			got.L1ChainID, got.L2ChainID, tmpl.L1ChainID, tmpl.L2ChainID)
	}
}

// TestReadBedrockDeployConfigTemplate_LegacyFallback verifies backwards-compat
// with the legacy Foundry layout: <bedrockPath>/scripts/deploy-config.json.
func TestReadBedrockDeployConfigTemplate_LegacyFallback(t *testing.T) {
	dir := t.TempDir()

	bedrockPath := filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")
	if err := os.MkdirAll(filepath.Join(bedrockPath, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}

	tmpl := types.DeployConfigTemplate{L1ChainID: 1, L2ChainID: 2}
	data, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bedrockPath, "scripts", "deploy-config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	stack := &ThanosStack{
		deploymentPath: dir,
		logger:         zap.NewNop().Sugar(),
	}

	got, err := stack.readBedrockDeployConfigTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.L1ChainID != tmpl.L1ChainID || got.L2ChainID != tmpl.L2ChainID {
		t.Errorf("mismatch: got L1=%d L2=%d, want L1=%d L2=%d",
			got.L1ChainID, got.L2ChainID, tmpl.L1ChainID, tmpl.L2ChainID)
	}
}

// TestReadBedrockDeployConfigTemplate_NewPathPrecedence verifies that when
// both new and legacy paths exist, the new path wins (avoids silently using
// stale template data from a previous Foundry-era deployment).
func TestReadBedrockDeployConfigTemplate_NewPathPrecedence(t *testing.T) {
	dir := t.TempDir()

	bedrockPath := filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")
	if err := os.MkdirAll(filepath.Join(bedrockPath, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}

	// Legacy: one value.
	legacyTmpl := types.DeployConfigTemplate{L1ChainID: 111, L2ChainID: 222}
	legacyData, _ := json.Marshal(legacyTmpl)
	if err := os.WriteFile(filepath.Join(bedrockPath, "scripts", "deploy-config.json"), legacyData, 0644); err != nil {
		t.Fatal(err)
	}

	// New: different value. Expected to win.
	newTmpl := types.DeployConfigTemplate{L1ChainID: 333, L2ChainID: 444}
	newData, _ := json.Marshal(newTmpl)
	if err := os.WriteFile(filepath.Join(dir, "deploy-config.json"), newData, 0644); err != nil {
		t.Fatal(err)
	}

	stack := &ThanosStack{
		deploymentPath: dir,
		logger:         zap.NewNop().Sugar(),
	}

	got, err := stack.readBedrockDeployConfigTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.L1ChainID != newTmpl.L1ChainID {
		t.Errorf("expected new path to win, got L1=%d (legacy=%d, new=%d)",
			got.L1ChainID, legacyTmpl.L1ChainID, newTmpl.L1ChainID)
	}
}

// TestReadDeploymentContracts_FaultProofAddresses verifies that
// AnchorStateRegistryProxy and DisputeGameFactoryProxy written by
// tokamak-deployer v0.0.6 --fault-proof mode are surfaced by
// readDeploymentContracts. Regression test for the consumer half of Bug #8.
func TestReadDeploymentContracts_FaultProofAddresses(t *testing.T) {
	dir := t.TempDir()

	// Bedrock path so readBedrockDeployConfigTemplate has somewhere to look
	// even though we only need deploy-config.json at the new location.
	if err := os.MkdirAll(filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments"), 0755); err != nil {
		t.Fatal(err)
	}

	const l1ChainID = 11155111
	cfg := types.DeployConfigTemplate{L1ChainID: l1ChainID, L2ChainID: 424242}
	cfgData, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deploy-config.json"), cfgData, 0644); err != nil {
		t.Fatal(err)
	}

	// Contracts file with fault-proof addresses populated, mirroring what
	// tokamak-deployer v0.0.6 emits when --fault-proof is set.
	deployFile := filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments",
		"11155111-deploy.json")
	const wantAnchor = "0x1111111111111111111111111111111111111111"
	const wantDispute = "0x2222222222222222222222222222222222222222"
	contractsJSON := `{
		"ProxyAdmin": "0x3333333333333333333333333333333333333333",
		"SystemConfigProxy": "0x4444444444444444444444444444444444444444",
		"AnchorStateRegistryProxy": "` + wantAnchor + `",
		"DisputeGameFactoryProxy": "` + wantDispute + `"
	}`
	if err := os.WriteFile(deployFile, []byte(contractsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	stack := &ThanosStack{
		deploymentPath: dir,
		logger:         zap.NewNop().Sugar(),
	}

	got, err := stack.readDeploymentContracts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AnchorStateRegistryProxy != wantAnchor {
		t.Errorf("AnchorStateRegistryProxy mismatch: got %q, want %q",
			got.AnchorStateRegistryProxy, wantAnchor)
	}
	if got.DisputeGameFactoryProxy != wantDispute {
		t.Errorf("DisputeGameFactoryProxy mismatch: got %q, want %q",
			got.DisputeGameFactoryProxy, wantDispute)
	}
}

// TestReadBedrockDeployConfigTemplate_NoneFound verifies the error message
// when neither location has a deploy-config.json.
func TestReadBedrockDeployConfigTemplate_NoneFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"), 0755); err != nil {
		t.Fatal(err)
	}

	stack := &ThanosStack{
		deploymentPath: dir,
		logger:         zap.NewNop().Sugar(),
	}

	_, err := stack.readBedrockDeployConfigTemplate()
	if err == nil {
		t.Fatal("expected error when no deploy-config.json exists")
	}
	if !strings.Contains(err.Error(), "deploy config file not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}
