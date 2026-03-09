package thanos

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// minimalStack returns a ThanosStack with only the helmRunner set,
// sufficient for testing helper methods that do not access other fields.
func minimalStack(hr *mock.HelmRunner) *ThanosStack {
	return &ThanosStack{helmRunner: hr}
}

// ─── valueFiles guard tests ────────────────────────────────────────────────

func TestHelmInstallWithFiles_EmptyValueFiles(t *testing.T) {
	s := minimalStack(&mock.HelmRunner{})
	err := s.helmInstallWithFiles(context.Background(), "rel", "chart", "ns", nil)
	if err == nil {
		t.Fatal("expected error for empty valueFiles, got nil")
	}
}

func TestHelmUpgradeWithFiles_EmptyValueFiles(t *testing.T) {
	s := minimalStack(&mock.HelmRunner{})
	err := s.helmUpgradeWithFiles(context.Background(), "rel", "chart", "ns", nil)
	if err == nil {
		t.Fatal("expected error for empty valueFiles, got nil")
	}
}

func TestHelmUpgradeInstallWithFiles_EmptyValueFiles(t *testing.T) {
	s := minimalStack(&mock.HelmRunner{})
	err := s.helmUpgradeInstallWithFiles(context.Background(), "rel", "chart", "ns", nil)
	if err == nil {
		t.Fatal("expected error for empty valueFiles, got nil")
	}
}

// ─── helmRepoAdd ───────────────────────────────────────────────────────────

func TestHelmRepoAdd_UsesRunner(t *testing.T) {
	m := &mock.HelmRunner{}
	m.OnRepoAdd = func(_ context.Context, name, url string) error {
		if name != "my-repo" || url != "https://example.com" {
			return errors.New("unexpected args")
		}
		return nil
	}
	s := minimalStack(m)
	if err := s.helmRepoAdd(context.Background(), "my-repo", "https://example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("RepoAdd") != 1 {
		t.Fatalf("expected 1 RepoAdd call, got %d", m.CallCount("RepoAdd"))
	}
}

func TestHelmRepoAdd_RunnerError(t *testing.T) {
	m := &mock.HelmRunner{}
	m.OnRepoAdd = func(_ context.Context, _, _ string) error {
		return errors.New("repo add failed")
	}
	s := minimalStack(m)
	err := s.helmRepoAdd(context.Background(), "repo", "url")
	if err == nil {
		t.Fatal("expected error from runner, got nil")
	}
}

// ─── helmSearch ────────────────────────────────────────────────────────────

func TestHelmSearch_UsesRunner(t *testing.T) {
	m := &mock.HelmRunner{}
	m.OnSearch = func(_ context.Context, keyword string) (string, error) {
		if keyword != "thanos-stack" {
			return "", errors.New("unexpected keyword")
		}
		return "thanos-stack/thanos-stack\t0.1.0", nil
	}
	s := minimalStack(m)
	result, err := s.helmSearch(context.Background(), "thanos-stack")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty search result")
	}
	if m.CallCount("Search") != 1 {
		t.Fatalf("expected 1 Search call, got %d", m.CallCount("Search"))
	}
}

func TestHelmSearch_RunnerError(t *testing.T) {
	m := &mock.HelmRunner{}
	m.OnSearch = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("search failed")
	}
	s := minimalStack(m)
	_, err := s.helmSearch(context.Background(), "thanos-stack")
	if err == nil {
		t.Fatal("expected error from runner, got nil")
	}
}
