package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

func TestTFRunnerMock_Init_CallsHook(t *testing.T) {
	m := &mock.TFRunner{}
	called := false
	m.OnInit = func(_ context.Context, workDir string, _ []string, _ []string) error {
		called = true
		if workDir != "/tmp/tf" {
			t.Errorf("unexpected workDir: %s", workDir)
		}
		return nil
	}
	if err := m.Init(context.Background(), "/tmp/tf", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("OnInit hook not called")
	}
	if m.CallCount("Init") != 1 {
		t.Fatalf("expected 1 Init call, got %d", m.CallCount("Init"))
	}
}

func TestTFRunnerMock_Apply_NoHook_ReturnsNil(t *testing.T) {
	m := &mock.TFRunner{}
	if err := m.Apply(context.Background(), "/tmp/tf", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTFRunnerMock_Destroy_HookError(t *testing.T) {
	m := &mock.TFRunner{}
	m.OnDestroy = func(_ context.Context, _ string, _ []string) error {
		return errors.New("destroy failed")
	}
	err := m.Destroy(context.Background(), "/tmp/tf", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTFRunnerMock_BackendConfigs(t *testing.T) {
	m := &mock.TFRunner{}
	var gotConfigs []string
	m.OnInit = func(_ context.Context, _ string, _ []string, configs []string) error {
		gotConfigs = configs
		return nil
	}
	_ = m.Init(context.Background(), ".", nil, []string{"bucket=my-bucket", "key=state.tfstate"})
	if len(gotConfigs) != 2 || gotConfigs[0] != "bucket=my-bucket" {
		t.Fatalf("unexpected backend configs: %v", gotConfigs)
	}
}

func TestTFRunnerMock_CallCount_MultipleOps(t *testing.T) {
	m := &mock.TFRunner{}
	_ = m.Init(context.Background(), ".", nil, nil)
	_ = m.Apply(context.Background(), ".", nil)
	_ = m.Apply(context.Background(), ".", nil)
	_ = m.Destroy(context.Background(), ".", nil)
	if m.CallCount("Init") != 1 {
		t.Fatalf("expected 1 Init, got %d", m.CallCount("Init"))
	}
	if m.CallCount("Apply") != 2 {
		t.Fatalf("expected 2 Apply, got %d", m.CallCount("Apply"))
	}
	if m.CallCount("Destroy") != 1 {
		t.Fatalf("expected 1 Destroy, got %d", m.CallCount("Destroy"))
	}
}

func TestTFRunnerMock_CheckVersion_NoHook(t *testing.T) {
	m := &mock.TFRunner{}
	if err := m.CheckVersion(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
