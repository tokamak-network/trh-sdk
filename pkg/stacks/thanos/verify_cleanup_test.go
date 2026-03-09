package thanos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// ─── deleteOrphanedLoadBalancers ───────────────────────────────────────────

// TestDeleteOrphanedLoadBalancers_ELBErrorDoesNotSkipALB verifies that a
// Classic ELB listing error does not prevent ALB/NLB processing.
func TestDeleteOrphanedLoadBalancers_ELBErrorDoesNotSkipALB(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnELBDescribeLoadBalancers = func(_ context.Context, _, _ string) ([]string, error) {
		return nil, errors.New("elb api unavailable")
	}
	albDeleted := false
	m.OnELBv2DescribeLoadBalancers = func(_ context.Context, _, _ string) ([]string, error) {
		return []string{"arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/test"}, nil
	}
	m.OnELBv2DeleteLoadBalancer = func(_ context.Context, _, _ string) error {
		albDeleted = true
		return nil
	}

	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	cleaned, failed := s.deleteOrphanedLoadBalancers(context.Background(), "us-east-1", "test-ns", 0, 0)

	if !albDeleted {
		t.Fatal("expected ALB deletion to proceed despite ELB listing error")
	}
	if cleaned != 1 {
		t.Fatalf("expected cleaned=1, got %d", cleaned)
	}
	if failed != 0 {
		t.Fatalf("expected failed=0, got %d", failed)
	}
}

// TestDeleteOrphanedLoadBalancers_BothEmpty returns zero counts when nothing is found.
func TestDeleteOrphanedLoadBalancers_BothEmpty(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnELBDescribeLoadBalancers = func(_ context.Context, _, _ string) ([]string, error) {
		return nil, nil
	}
	m.OnELBv2DescribeLoadBalancers = func(_ context.Context, _, _ string) ([]string, error) {
		return nil, nil
	}
	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	cleaned, failed := s.deleteOrphanedLoadBalancers(context.Background(), "us-east-1", "test-ns", 0, 0)
	if cleaned != 0 || failed != 0 {
		t.Fatalf("expected 0/0, got %d/%d", cleaned, failed)
	}
}

// ─── deleteOrphanedEKS ─────────────────────────────────────────────────────

// TestDeleteOrphanedEKS_NodeGroupDeletionFailureCounted verifies that
// failed node group deletions increment the failed counter.
func TestDeleteOrphanedEKS_NodeGroupDeletionFailureCounted(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEKSClusterExists = func(_ context.Context, _, _ string) (bool, error) {
		return true, nil
	}
	m.OnEKSListNodegroups = func(_ context.Context, _, _ string) ([]string, error) {
		return []string{"ng-1"}, nil
	}
	m.OnEKSDeleteNodegroup = func(_ context.Context, _, _, _ string) error {
		return errors.New("deletion failed")
	}
	m.OnEKSDeleteCluster = func(_ context.Context, _, _ string) error {
		return nil
	}

	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	// Pass a cancelled context so waitForNodeGroupsDeletion exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cleaned, failed := s.deleteOrphanedEKS(ctx, "us-east-1", "test-ns", 0, 0)

	if failed != 1 {
		t.Fatalf("expected failed=1 (node group), got %d", failed)
	}
	// Cluster deletion still succeeded despite node group failure.
	if cleaned != 1 {
		t.Fatalf("expected cleaned=1 (cluster), got %d", cleaned)
	}
}

// TestDeleteOrphanedEKS_ClusterNotFound returns unchanged counts when EKS cluster absent.
func TestDeleteOrphanedEKS_ClusterNotFound(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEKSClusterExists = func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	}
	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	cleaned, failed := s.deleteOrphanedEKS(context.Background(), "us-east-1", "test-ns", 5, 2)
	if cleaned != 5 || failed != 2 {
		t.Fatalf("expected 5/2 unchanged, got %d/%d", cleaned, failed)
	}
}

// TestDeleteOrphanedEKS_ClusterExistsAPIError verifies that an API error from
// EKSClusterExists returns unchanged counts and emits a Warn log containing
// "EKS cluster existence".
func TestDeleteOrphanedEKS_ClusterExistsAPIError(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEKSClusterExists = func(_ context.Context, _, _ string) (bool, error) {
		return false, errors.New("iam: access denied")
	}
	logger, logs := warnObserver()
	s := &ThanosStack{awsRunner: m, logger: logger}
	cleaned, failed := s.deleteOrphanedEKS(context.Background(), "us-east-1", "test-ns", 3, 1)
	if cleaned != 3 || failed != 1 {
		t.Fatalf("expected 3/1 unchanged on API error, got %d/%d", cleaned, failed)
	}
	assertWarnLogContains(t, logs, "EKS cluster existence")
}

// TestDeleteOrphanedRDS_InstanceExistsAPIError verifies that an API error from
// RDSInstanceExists returns unchanged counts and emits a Warn log containing
// "RDS instance existence".
func TestDeleteOrphanedRDS_InstanceExistsAPIError(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnRDSInstanceExists = func(_ context.Context, _, _ string) (bool, error) {
		return false, errors.New("throttling: rate exceeded")
	}
	logger, logs := warnObserver()
	s := &ThanosStack{awsRunner: m, logger: logger}
	cleaned, failed := s.deleteOrphanedRDS(context.Background(), "us-east-1", "test-ns", 2, 0)
	if cleaned != 2 || failed != 0 {
		t.Fatalf("expected 2/0 unchanged on API error, got %d/%d", cleaned, failed)
	}
	assertWarnLogContains(t, logs, "RDS instance existence")
}

// TestDeleteOrphanedVPC_DescribeAPIError verifies that an EC2DescribeVPCs API error
// returns unchanged counts and emits a Warn log containing "list VPCs".
func TestDeleteOrphanedVPC_DescribeAPIError(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEC2DescribeVPCs = func(_ context.Context, _, _ string) ([]string, error) {
		return nil, errors.New("ec2: request limit exceeded")
	}
	logger, logs := warnObserver()
	s := &ThanosStack{awsRunner: m, logger: logger}
	cleaned, failed := s.deleteOrphanedVPC(context.Background(), "us-east-1", "test-ns", 4, 0)
	if cleaned != 4 || failed != 0 {
		t.Fatalf("expected 4/0 unchanged on API error, got %d/%d", cleaned, failed)
	}
	assertWarnLogContains(t, logs, "list VPCs")
}

// ─── deleteOrphanedElasticIPs ──────────────────────────────────────────────

// TestDeleteOrphanedElasticIPs_ContextCancelledBeforeRelease verifies that
// a cancelled context stops processing before release without hanging.
func TestDeleteOrphanedElasticIPs_ContextCancelledBeforeRelease(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEC2DescribeAddresses = func(_ context.Context, _, _ string) ([]runner.ElasticIPInfo, error) {
		return []runner.ElasticIPInfo{{AllocationID: "eip-1", AssociationID: "assoc-1"}}, nil
	}
	m.OnEC2DisassociateAddress = func(_ context.Context, _, _ string) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled — select should choose ctx.Done() immediately

	releaseAttempted := false
	m.OnEC2ReleaseAddress = func(_ context.Context, _, _ string) error {
		releaseAttempted = true
		return nil
	}

	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	// Should return promptly without hanging on the sleep.
	s.deleteOrphanedElasticIPs(ctx, "us-east-1", "test-ns", 0, 0)

	if releaseAttempted {
		t.Fatal("expected release not to be attempted after context cancellation")
	}
}

// ─── deleteOrphanedNATGateways ─────────────────────────────────────────────

// TestDeleteOrphanedNATGateways_EmptyList returns immediately with unchanged counts.
func TestDeleteOrphanedNATGateways_EmptyList(t *testing.T) {
	m := &mock.AWSRunner{}
	// nil hook → (nil, nil)
	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	cleaned, failed := s.deleteOrphanedNATGateways(context.Background(), "us-east-1", "test-ns", 3, 1)
	if cleaned != 3 || failed != 1 {
		t.Fatalf("expected 3/1 unchanged, got %d/%d", cleaned, failed)
	}
}

// ─── waitForNodeGroupsDeletion ─────────────────────────────────────────────

// TestWaitForNodeGroupsDeletion_ExitsWhenContextCancelled verifies the wait
// loop exits promptly on context cancellation.
func TestWaitForNodeGroupsDeletion_ExitsWhenContextCancelled(t *testing.T) {
	m := &mock.AWSRunner{}
	callCount := 0
	m.OnEKSListNodegroups = func(_ context.Context, _, _ string) ([]string, error) {
		callCount++
		return []string{"ng-still-running"}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	s.waitForNodeGroupsDeletion(ctx, "us-east-1", "test-cluster")

	// The check-first pattern calls ListNodeGroups once before the ticker select.
	// Then ctx.Done() fires immediately, exiting without further polls.
	if callCount > 1 {
		t.Fatalf("expected at most 1 ListNodeGroups call, got %d", callCount)
	}
}

// TestWaitForNodeGroupsDeletion_ExitsWhenEmpty verifies the loop exits
// immediately when the node group list is already empty on first check.
// The check-first pattern means no ticker wait is needed.
func TestWaitForNodeGroupsDeletion_ExitsWhenEmpty(t *testing.T) {
	m := &mock.AWSRunner{}
	m.OnEKSListNodegroups = func(_ context.Context, _, _ string) ([]string, error) {
		return nil, nil // empty — deletion complete
	}

	s := &ThanosStack{awsRunner: m, logger: noopLogger()}
	// With the check-first pattern, empty list returns before the first ticker
	// tick — no 30-second wait occurs, so this test completes well within 1s.
	done := make(chan struct{})
	go func() {
		s.waitForNodeGroupsDeletion(context.Background(), "us-east-1", "test-cluster")
		close(done)
	}()

	select {
	case <-done:
		// passed — returned before ticker fired
	case <-time.After(5 * time.Second):
		t.Fatal("waitForNodeGroupsDeletion did not return promptly with empty node list")
	}
}
