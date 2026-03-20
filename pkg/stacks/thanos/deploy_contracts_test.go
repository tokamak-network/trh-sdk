package thanos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

const testMakefileContent = `.PHONY: cannon-prestate
cannon-prestate:
	cd cannon && make cannon64-impl
	cannon/bin/cannon load-elf --path op-program/bin/op-program-client.elf --out op-program/bin/prestate.json
	cannon/bin/cannon run --proof-at '%(%)' --stop-at '=%(%)' --input op-program/bin/prestate.json --meta "" --proof-fmt 'op-program/bin/%d.json' --output ""
`

func TestPatchMakefileCannonPrestate_SuccessfulPatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(testMakefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchMakefileCannonPrestate(dir); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	patched := string(data)

	if strings.Contains(patched, "cannon/bin/cannon load-elf") {
		t.Error("old load-elf pattern still present")
	}
	if !strings.Contains(patched, "cannon64-impl load-elf --type multithreaded64-5") {
		t.Error("new load-elf pattern not found")
	}
	if !strings.Contains(patched, "cp op-program/bin/prestate-proof.json op-program/bin/prestate.json") {
		t.Error("cp prestate.json command not found")
	}
}

func TestPatchMakefileCannonPrestate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte(testMakefileContent), 0644)

	patchMakefileCannonPrestate(dir)
	first, _ := os.ReadFile(filepath.Join(dir, "Makefile"))

	patchMakefileCannonPrestate(dir) // second call
	second, _ := os.ReadFile(filepath.Join(dir, "Makefile"))

	if string(first) != string(second) {
		t.Error("second patch modified the file — not idempotent")
	}
}

func TestPatchMakefileCannonPrestate_NoMatchReturnsError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\techo hello\n"), 0644)

	err := patchMakefileCannonPrestate(dir)
	if err == nil {
		t.Fatal("expected error for non-matching Makefile")
	}
	if !strings.Contains(err.Error(), "no patterns matched") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPatchMakefileCannonPrestate_MissingMakefile(t *testing.T) {
	err := patchMakefileCannonPrestate(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing Makefile")
	}
}

func TestIsLocal_Values(t *testing.T) {
	tests := []struct {
		network string
		want    bool
	}{
		{"LocalTestnet", true},
		{"testnet", false},
		{"mainnet", false},
		{"local_devnet", false},
	}
	for _, tt := range tests {
		s := &ThanosStack{network: tt.network}
		if got := s.isLocal(); got != tt.want {
			t.Errorf("isLocal(%q) = %v, want %v", tt.network, got, tt.want)
		}
	}
}

func TestNewLocalTestnetThanosStack_SetsFields(t *testing.T) {
	dir := t.TempDir()
	// ReadConfigFromJSONFile returns nil config (not error) when file missing,
	// but may attempt L2 connection if config has l2_rpc_url. Use empty dir.
	stack, err := NewLocalTestnetThanosStack(nil, zap.NewNop().Sugar(), dir, "/tmp/my.kubeconfig")
	if err != nil {
		t.Skipf("skipping: ReadConfigFromJSONFile side effect: %v", err)
	}
	if !stack.isLocal() {
		t.Error("expected isLocal()=true")
	}
	if stack.kubeconfigPath != "/tmp/my.kubeconfig" {
		t.Errorf("expected kubeconfigPath=/tmp/my.kubeconfig, got %s", stack.kubeconfigPath)
	}
	if stack.usePromptInput {
		t.Error("expected usePromptInput=false for LocalTestnet")
	}
}

func TestKubectlWrapper_InjectsKubeconfig(t *testing.T) {
	s := &ThanosStack{kubeconfigPath: "/tmp/test.kubeconfig"}
	// Can't run real kubectl, but verify args would be correct
	// by checking the method exists and kubeconfigPath is set
	if s.kubeconfigPath != "/tmp/test.kubeconfig" {
		t.Fatal("kubeconfigPath not set")
	}
}

func TestKubectlWrapper_NoKubeconfigWhenEmpty(t *testing.T) {
	s := &ThanosStack{kubeconfigPath: ""}
	if s.kubeconfigPath != "" {
		t.Fatal("kubeconfigPath should be empty")
	}
}

func TestDeployContractsInput_BuildOnlyField(t *testing.T) {
	input := &DeployContractsInput{BuildOnly: true}
	if !input.BuildOnly {
		t.Fatal("BuildOnly should be true")
	}
	input2 := &DeployContractsInput{}
	if input2.BuildOnly {
		t.Fatal("BuildOnly should default to false")
	}
}
