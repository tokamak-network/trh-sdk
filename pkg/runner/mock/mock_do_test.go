package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// TestMockDORunner_ValidateToken verifies default behaviour (no error, call recorded).
func TestMockDORunner_ValidateToken(t *testing.T) {
	m := &mock.DORunner{}
	ctx := context.Background()

	err := m.ValidateToken(ctx, "dop_v1_test_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("ValidateToken"); count != 1 {
		t.Fatalf("expected 1 ValidateToken call, got %d", count)
	}
	calls := m.GetCalls()
	if got := calls[0].Args[0]; got != "dop_v1_test_token" {
		t.Fatalf("expected token=dop_v1_test_token, got %v", got)
	}
}

// TestMockDORunner_ValidateToken_Error verifies error propagation.
func TestMockDORunner_ValidateToken_Error(t *testing.T) {
	m := &mock.DORunner{}
	ctx := context.Background()
	want := errors.New("invalid token")
	m.OnValidateToken = func(_ context.Context, _ string) error { return want }

	err := m.ValidateToken(ctx, "bad-token")
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// TestMockDORunner_ListRegions verifies fixture return from ListRegions.
func TestMockDORunner_ListRegions(t *testing.T) {
	m := &mock.DORunner{}
	ctx := context.Background()

	fixture := []string{"nyc1", "sfo3", "ams3"}
	m.OnListRegions = func(_ context.Context, _ string) ([]string, error) {
		return fixture, nil
	}

	regions, err := m.ListRegions(ctx, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 3 {
		t.Fatalf("expected 3 regions, got %d", len(regions))
	}
	if regions[0] != "nyc1" {
		t.Fatalf("expected first region=nyc1, got %v", regions[0])
	}
}

// TestMockDORunner_GetKubeconfig verifies call recording for GetKubeconfig.
func TestMockDORunner_GetKubeconfig(t *testing.T) {
	m := &mock.DORunner{}
	ctx := context.Background()

	err := m.GetKubeconfig(ctx, "my-cluster", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("GetKubeconfig"); count != 1 {
		t.Fatalf("expected 1 GetKubeconfig call, got %d", count)
	}
	calls := m.GetCalls()
	if got := calls[0].Args[0]; got != "my-cluster" {
		t.Fatalf("expected clusterName=my-cluster, got %v", got)
	}
}

// TestMockDORunner_CheckVersion verifies default no-op behaviour.
func TestMockDORunner_CheckVersion(t *testing.T) {
	m := &mock.DORunner{}
	ctx := context.Background()

	err := m.CheckVersion(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := m.CallCount("CheckVersion"); count != 1 {
		t.Fatalf("expected 1 CheckVersion call, got %d", count)
	}
}
