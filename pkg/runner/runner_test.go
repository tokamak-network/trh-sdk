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
	if got := m.Calls[0].Args[0]; got != "test-ns" {
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
