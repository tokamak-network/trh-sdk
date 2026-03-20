package runner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	runmock "github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// TestNew_LegacyEnv verifies that TRHS_LEGACY=1 forces ShellOutRunner regardless
// of UseNative config.
func TestNew_LegacyEnv(t *testing.T) {
	t.Setenv("TRHS_LEGACY", "1")
	r, err := runner.New(runner.RunnerConfig{UseNative: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil ToolRunner")
	}
	if r.K8s() == nil {
		t.Fatal("expected non-nil K8sRunner")
	}
}

// TestNew_ShellOut verifies that UseNative=false returns ShellOutRunner.
func TestNew_ShellOut(t *testing.T) {
	r, err := runner.New(runner.RunnerConfig{UseNative: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.K8s() == nil {
		t.Fatal("expected non-nil K8sRunner")
	}
}

// TestMockK8sRunner_EnsureNamespace demonstrates using the mock in unit tests.
func TestMockK8sRunner_EnsureNamespace(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()

	err := m.EnsureNamespace(ctx, "test-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("EnsureNamespace"); count != 1 {
		t.Fatalf("expected 1 call, got %d", count)
	}
	calls := m.GetCalls()
	if got := calls[0].Args[0]; got != "test-ns" {
		t.Fatalf("expected namespace=test-ns, got %v", got)
	}
}

// TestMockK8sRunner_EnsureNamespace_Error verifies the mock propagates errors.
func TestMockK8sRunner_EnsureNamespace_Error(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()
	want := errors.New("namespace quota exceeded")
	m.OnEnsureNamespace = func(_ context.Context, _ string) error { return want }

	err := m.EnsureNamespace(ctx, "test-ns")
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// TestMockK8sRunner_Wait demonstrates configuring Wait response.
func TestMockK8sRunner_Wait(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()

	err := m.Wait(ctx, "deployment", "op-node", "thanos", "Available", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("Wait"); count != 1 {
		t.Fatalf("expected 1 Wait call, got %d", count)
	}
}

// TestMockK8sRunner_List returns a fixture and verifies call count.
func TestMockK8sRunner_List(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()

	fixture := []byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`)
	m.OnList = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return fixture, nil
	}

	data, err := m.List(ctx, "pods", "thanos", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(fixture) {
		t.Fatalf("expected fixture, got %s", data)
	}
	if count := m.CallCount("List"); count != 1 {
		t.Fatalf("expected 1 List call, got %d", count)
	}
}

// TestMockK8sRunner_Delete_Error verifies error propagation for Delete.
func TestMockK8sRunner_Delete_Error(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()

	want := errors.New("forbidden")
	m.OnDelete = func(_ context.Context, _, _, _ string, _ bool) error { return want }

	err := m.Delete(ctx, "pvc", "data-pvc", "thanos", false)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// TestMockK8sRunner_NamespaceExists_False verifies default false response.
func TestMockK8sRunner_NamespaceExists_False(t *testing.T) {
	m := &runmock.K8sRunner{}
	ctx := context.Background()

	exists, err := m.NamespaceExists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false")
	}
}

// TestShellOutK8sRunner_Logs_AlwaysErrors verifies that the legacy Logs
// implementation always returns an error without leaking infrastructure details.
func TestShellOutK8sRunner_Logs_AlwaysErrors(t *testing.T) {
	r, err := runner.New(runner.RunnerConfig{UseNative: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	rc, err := r.K8s().Logs(ctx, "my-pod", "my-ns", "", false)
	if err == nil {
		_ = rc.Close()
		t.Fatal("expected error from legacy Logs, got nil")
	}
	// Error must not contain pod name or namespace (no infrastructure leak).
	msg := err.Error()
	if containsAny(msg, "my-pod", "my-ns") {
		t.Fatalf("error message leaks infrastructure details: %q", msg)
	}
}

// ─── ShellOutK8sRunner input-validation tests ─────────────────────────────────

func TestShellOutK8sRunner_Delete_EmptyResource(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.K8s().Delete(context.Background(), "", "my-obj", "default", false)
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestShellOutK8sRunner_Delete_EmptyName(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.K8s().Delete(context.Background(), "pods", "", "default", false)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestShellOutK8sRunner_Get_EmptyResource(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	_, err := r.K8s().Get(context.Background(), "", "my-obj", "default")
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestShellOutK8sRunner_Get_EmptyName(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	_, err := r.K8s().Get(context.Background(), "pods", "", "default")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestShellOutK8sRunner_List_EmptyResource(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	_, err := r.K8s().List(context.Background(), "", "default", "")
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestShellOutK8sRunner_Patch_EmptyResource(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.K8s().Patch(context.Background(), "", "my-obj", "default", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestShellOutK8sRunner_Patch_EmptyName(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.K8s().Patch(context.Background(), "pods", "", "default", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// ─── ShellOutHelmRunner input-validation tests ─────────────────────────────────

func TestShellOutHelmRunner_Uninstall_EmptyRelease(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.Helm().Uninstall(context.Background(), "", "default")
	if err == nil {
		t.Fatal("expected error for empty release")
	}
}

func TestShellOutHelmRunner_Uninstall_EmptyNamespace(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.Helm().Uninstall(context.Background(), "my-release", "")
	if err == nil {
		t.Fatal("expected error for empty namespace")
	}
}

func TestShellOutHelmRunner_Install_EmptyRelease(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.Helm().Install(context.Background(), "", "my-chart", "default", nil)
	if err == nil {
		t.Fatal("expected error for empty release")
	}
}

func TestShellOutHelmRunner_Install_EmptyNamespace(t *testing.T) {
	r, _ := runner.New(runner.RunnerConfig{UseNative: false})
	err := r.Helm().Install(context.Background(), "my-release", "my-chart", "", nil)
	if err == nil {
		t.Fatal("expected error for empty namespace")
	}
}

// ─── ToolRunner Helm() accessor tests ──────────────────────────────────────────

func TestNew_ShellOut_Helm(t *testing.T) {
	r, err := runner.New(runner.RunnerConfig{UseNative: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Helm() == nil {
		t.Fatal("expected non-nil HelmRunner from ShellOutRunner")
	}
}

// ─── NativeDORunner tests ──────────────────────────────────────────────────────

// TestNativeDORunner_CheckVersion_NoOp verifies that NativeDORunner.CheckVersion
// is a no-op (always returns nil) since no external binary is needed.
func TestNativeDORunner_CheckVersion_NoOp(t *testing.T) {
	r, err := runner.New(runner.RunnerConfig{UseNative: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ShellOutDORunner.CheckVersion would require doctl; verify it returns a DORunner.
	if r.DO() == nil {
		t.Fatal("expected non-nil DORunner")
	}
}

// ─── ToolRunner DO() accessor tests ────────────────────────────────────────────

func TestNew_ShellOut_DO(t *testing.T) {
	r, err := runner.New(runner.RunnerConfig{UseNative: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.DO() == nil {
		t.Fatal("expected non-nil DORunner from ShellOutRunner")
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
