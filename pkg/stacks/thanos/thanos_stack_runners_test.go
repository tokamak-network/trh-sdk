package thanos

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// ─── PodLogs ──────────────────────────────────────────────────────────────────

// TestPodLogs_UsesK8sRunnerWhenSinceIsZero verifies that PodLogs delegates to
// K8sRunner.Logs when the runner is set and since is zero.
func TestPodLogs_UsesK8sRunnerWhenSinceIsZero(t *testing.T) {
	m := &mock.K8sRunner{}
	callCount := 0
	m.OnLogs = func(_ context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error) {
		callCount++
		if pod != "my-pod" || namespace != "my-ns" {
			t.Errorf("unexpected args: pod=%q ns=%q", pod, namespace)
		}
		return io.NopCloser(strings.NewReader("log-line-1\n")), nil
	}

	s := &ThanosStack{k8sRunner: m, logger: noopLogger()}
	out, err := s.PodLogs(context.Background(), "my-pod", "my-ns", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 K8sRunner.Logs call, got %d", callCount)
	}
	if !strings.Contains(string(out), "log-line-1") {
		t.Fatalf("expected log content in output, got %q", string(out))
	}
}

// TestPodLogs_FallsBackWhenSinceIsNonZero verifies that PodLogs bypasses the
// K8sRunner and uses shell-out when since > 0, preserving time-window correctness.
// The shell-out call will fail (kubectl not present in CI), but we verify that
// K8sRunner.Logs was NOT called — the fallback path was taken.
func TestPodLogs_FallsBackWhenSinceIsNonZero(t *testing.T) {
	m := &mock.K8sRunner{}
	runnerCalled := false
	m.OnLogs = func(_ context.Context, _, _, _ string, _ bool) (io.ReadCloser, error) {
		runnerCalled = true
		return io.NopCloser(strings.NewReader("")), nil
	}

	s := &ThanosStack{k8sRunner: m, logger: noopLogger()}
	// since > 0 → must NOT use runner (runner lacks time-window support)
	s.PodLogs(context.Background(), "pod", "ns", "", 1*time.Hour) //nolint:errcheck
	if runnerCalled {
		t.Fatal("expected K8sRunner to be bypassed when since > 0, but it was called")
	}
}

// TestPodLogs_FallsBackWhenRunnerIsNil verifies the shell-out path is taken
// when k8sRunner is nil. kubectl is absent in the test environment so the
// call returns an error; we assert: no panic, no nil-deref, and the error is
// non-nil (proving exec was attempted, not the runner).
func TestPodLogs_FallsBackWhenRunnerIsNil(t *testing.T) {
	s := &ThanosStack{k8sRunner: nil, logger: noopLogger()}
	_, err := s.PodLogs(context.Background(), "pod", "ns", "", 0)
	// kubectl is absent → shell-out must fail. A nil error here would mean
	// the native runner path was taken (impossible since k8sRunner == nil).
	if err == nil {
		t.Fatal("expected an error from kubectl shell-out when runner is nil, got nil")
	}
}

// TestPodLogs_LimitReader verifies that PodLogs caps memory usage:
// a stream larger than maxLogBytes must not cause a panic or error — the read
// is simply capped at maxLogBytes.
func TestPodLogs_LimitReader(t *testing.T) {
	m := &mock.K8sRunner{}
	// Produce a reader that reports 200 MiB (larger than maxLogBytes = 100 MiB)
	// using a LimitedReader so the test is fast.
	const size = 200 << 20
	m.OnLogs = func(_ context.Context, _, _, _ string, _ bool) (io.ReadCloser, error) {
		return io.NopCloser(io.LimitReader(zeroReader{}, size)), nil
	}

	s := &ThanosStack{k8sRunner: m, logger: noopLogger()}
	out, err := s.PodLogs(context.Background(), "pod", "ns", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(len(out)) > maxLogBytes {
		t.Fatalf("expected at most %d bytes, got %d", maxLogBytes, len(out))
	}
}

// zeroReader is an infinite source of zero bytes for TestPodLogs_LimitReader.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// ─── injectRunners ────────────────────────────────────────────────────────────

// TestInjectRunners_EitherInjectsOrWarns verifies that injectRunners either
// populates all four runner fields (when native init succeeds) or emits a Warn
// log and leaves them nil (when native init fails due to missing credentials).
// Both outcomes are valid; the invariant is: no panic, no silent partial state.
func TestInjectRunners_EitherInjectsOrWarns(t *testing.T) {
	t.Setenv("TRHS_LEGACY", "")

	logger, logs := warnObserver()
	stack := &ThanosStack{logger: logger}
	injectRunners(stack, logger, "")

	allNil := stack.helmRunner == nil && stack.k8sRunner == nil &&
		stack.tfRunner == nil && stack.awsRunner == nil
	allSet := stack.helmRunner != nil && stack.k8sRunner != nil &&
		stack.tfRunner != nil && stack.awsRunner != nil

	if !allNil && !allSet {
		t.Fatalf("injectRunners must either wire all runners or none, got partial state: "+
			"helm=%v k8s=%v tf=%v aws=%v",
			stack.helmRunner != nil, stack.k8sRunner != nil,
			stack.tfRunner != nil, stack.awsRunner != nil)
	}
	if allNil {
		// Failure path — a Warn must have been emitted.
		assertWarnLogContains(t, logs, "Native runner init failed")
	}
}

// TestInjectRunners_ShellOutAlwaysSucceeds verifies that TRHS_LEGACY=1 causes
// injectRunners to inject ShellOutRunner variants (non-nil) unconditionally,
// regardless of whether native runner initialisation would succeed or fail.
func TestInjectRunners_ShellOutAlwaysSucceeds(t *testing.T) {
	t.Setenv("TRHS_LEGACY", "1")

	stack := &ThanosStack{logger: noopLogger()}
	injectRunners(stack, noopLogger(), "")

	// ShellOutRunner always succeeds — all four fields must be non-nil.
	if stack.helmRunner == nil || stack.k8sRunner == nil ||
		stack.tfRunner == nil || stack.awsRunner == nil {
		t.Errorf("expected all runner fields non-nil with TRHS_LEGACY=1: "+
			"helm=%v k8s=%v tf=%v aws=%v",
			stack.helmRunner != nil, stack.k8sRunner != nil,
			stack.tfRunner != nil, stack.awsRunner != nil)
	}
}
