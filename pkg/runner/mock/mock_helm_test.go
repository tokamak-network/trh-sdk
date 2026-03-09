package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// TestMockHelmRunner_Install verifies call recording for Install.
func TestMockHelmRunner_Install(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	err := m.Install(ctx, "my-release", "my-chart", "default", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("Install"); count != 1 {
		t.Fatalf("expected 1 Install call, got %d", count)
	}
	calls := m.GetCalls()
	if got := calls[0].Args[0]; got != "my-release" {
		t.Fatalf("expected release=my-release, got %v", got)
	}
	if got := calls[0].Args[1]; got != "my-chart" {
		t.Fatalf("expected chart=my-chart, got %v", got)
	}
	if got := calls[0].Args[2]; got != "default" {
		t.Fatalf("expected namespace=default, got %v", got)
	}
}

// TestMockHelmRunner_Install_Error verifies error propagation for Install.
func TestMockHelmRunner_Install_Error(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()
	want := errors.New("chart not found")
	m.OnInstall = func(_ context.Context, _, _, _ string, _ map[string]interface{}) error { return want }

	err := m.Install(ctx, "my-release", "my-chart", "default", nil)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// TestMockHelmRunner_Uninstall_Error verifies error propagation for Uninstall.
func TestMockHelmRunner_Uninstall_Error(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()
	want := errors.New("release not found")
	m.OnUninstall = func(_ context.Context, _, _ string) error { return want }

	err := m.Uninstall(ctx, "my-release", "default")
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// TestMockHelmRunner_List verifies fixture return from List.
func TestMockHelmRunner_List(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	fixture := []string{"release-a", "release-b"}
	m.OnList = func(_ context.Context, _ string) ([]string, error) {
		return fixture, nil
	}

	releases, err := m.List(ctx, "thanos")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
	if releases[0] != "release-a" || releases[1] != "release-b" {
		t.Fatalf("unexpected releases: %v", releases)
	}
	if count := m.CallCount("List"); count != 1 {
		t.Fatalf("expected 1 List call, got %d", count)
	}
}

// TestMockHelmRunner_List_Default verifies nil return when OnList is not set.
func TestMockHelmRunner_List_Default(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	releases, err := m.List(ctx, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if releases != nil {
		t.Fatalf("expected nil releases, got %v", releases)
	}
}

// TestMockHelmRunner_Upgrade verifies call recording for Upgrade.
func TestMockHelmRunner_Upgrade(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	err := m.Upgrade(ctx, "my-release", "my-chart", "default", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("Upgrade"); count != 1 {
		t.Fatalf("expected 1 Upgrade call, got %d", count)
	}
}

// TestMockHelmRunner_RepoAdd verifies call recording for RepoAdd.
func TestMockHelmRunner_RepoAdd(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	err := m.RepoAdd(ctx, "stable", "https://charts.helm.sh/stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("RepoAdd"); count != 1 {
		t.Fatalf("expected 1 RepoAdd call, got %d", count)
	}
	calls := m.GetCalls()
	if got := calls[0].Args[0]; got != "stable" {
		t.Fatalf("expected name=stable, got %v", got)
	}
}

// TestMockHelmRunner_Status verifies Status returns configured value.
func TestMockHelmRunner_Status(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	m.OnStatus = func(_ context.Context, _, _ string) (string, error) {
		return "deployed", nil
	}

	status, err := m.Status(ctx, "my-release", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "deployed" {
		t.Fatalf("expected deployed, got %s", status)
	}
}

// TestMockHelmRunner_UpgradeWithFiles verifies call recording for UpgradeWithFiles.
func TestMockHelmRunner_UpgradeWithFiles(t *testing.T) {
	m := &mock.HelmRunner{}
	ctx := context.Background()

	err := m.UpgradeWithFiles(ctx, "my-release", "my-chart", "default", []string{"values.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("UpgradeWithFiles"); count != 1 {
		t.Fatalf("expected 1 UpgradeWithFiles call, got %d", count)
	}
}
