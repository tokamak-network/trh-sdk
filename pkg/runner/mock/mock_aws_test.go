package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

func TestAWSRunnerMock_GetCallerIdentityAccount(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnGetCallerIdentityAccount = func(_ context.Context) (string, error) {
		return "123456789012", nil
	}
	got, err := m.GetCallerIdentityAccount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "123456789012" {
		t.Fatalf("expected account 123456789012, got %q", got)
	}
	if m.CallCount("GetCallerIdentityAccount") != 1 {
		t.Fatalf("expected 1 call, got %d", m.CallCount("GetCallerIdentityAccount"))
	}
}

func TestAWSRunnerMock_IAMRoleExists_NotFound(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnIAMRoleExists = func(_ context.Context, roleName string) (bool, error) {
		if roleName != "my-role" {
			return false, errors.New("unexpected role name")
		}
		return false, nil
	}
	exists, err := m.IAMRoleExists(context.Background(), "my-role")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected role not to exist")
	}
}

func TestAWSRunnerMock_EFSDescribeFileSystems_ReturnsList(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEFSDescribeFileSystems = func(_ context.Context, _, _ string) ([]runner.EFSFileSystem, error) {
		return []runner.EFSFileSystem{
			{FileSystemID: "fs-123", Name: "my-fs", LifeCycleState: "available"},
		}, nil
	}
	fss, err := m.EFSDescribeFileSystems(context.Background(), "us-east-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fss) != 1 || fss[0].FileSystemID != "fs-123" {
		t.Fatalf("unexpected result: %v", fss)
	}
}

func TestAWSRunnerMock_NoHook_DefaultZeroValues(t *testing.T) {
	m := &mock.AWSRunner{}
	account, err := m.GetCallerIdentityAccount(context.Background())
	if err != nil || account != "" {
		t.Fatalf("expected empty account and nil error, got %q %v", account, err)
	}
	exists, err := m.EKSClusterExists(context.Background(), "us-east-1", "my-cluster")
	if err != nil || exists {
		t.Fatalf("expected false and nil error, got %v %v", exists, err)
	}
	buckets, err := m.S3ListBuckets(context.Background())
	if err != nil || buckets != nil {
		t.Fatalf("expected nil buckets, got %v %v", buckets, err)
	}
}

func TestAWSRunnerMock_BackupStartBackupJob_ReturnsJobID(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnBackupStartBackupJob = func(_ context.Context, _, _, _, _ string) (string, error) {
		return "job-abc123", nil
	}
	jobID, err := m.BackupStartBackupJob(context.Background(), "us-east-1", "my-vault", "arn:aws:efs:...", "arn:aws:iam:...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobID != "job-abc123" {
		t.Fatalf("expected job-abc123, got %q", jobID)
	}
}

func TestAWSRunnerMock_GetCalls_ThreadSafe(t *testing.T) {
	m := &mock.AWSRunner{}
	_, _ = m.GetCallerIdentityAccount(context.Background())
	_ = m.CheckVersion(context.Background())
	_, _ = m.S3ListBuckets(context.Background())

	calls := m.GetCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}
}
