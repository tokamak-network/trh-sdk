package backup

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// ─── k8sListStatefulSetNames ────────────────────────────────────────────────

func TestK8sListStatefulSetNames_ReturnsList(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, namespace, _ string) ([]byte, error) {
		if resource != "statefulsets" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"items":[{"metadata":{"name":"sts-a"}},{"metadata":{"name":"sts-b"}}]}`), nil
	}
	c := NewBackupClient(m)
	names, err := c.k8sListStatefulSetNames(context.Background(), "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "sts-a" || names[1] != "sts-b" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestK8sListStatefulSetNames_RunnerError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("api error")
	}
	c := NewBackupClient(m)
	_, err := c.k8sListStatefulSetNames(context.Background(), "ns")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── k8sGetPVCExists ────────────────────────────────────────────────────────

func TestK8sGetPVCExists_Found(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, resource, name, namespace string) ([]byte, error) {
		if resource != "pvc" || name != "my-pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"metadata":{"name":"my-pvc"}}`), nil
	}
	c := NewBackupClient(m)
	exists, err := c.k8sGetPVCExists(context.Background(), "my-pvc", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true, got false")
	}
}

func TestK8sGetPVCExists_NotFound(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("not found")
	}
	c := NewBackupClient(m)
	exists, err := c.k8sGetPVCExists(context.Background(), "missing", "ns")
	if err != nil {
		t.Fatalf("expected nil error for not-found, got %v", err)
	}
	if exists {
		t.Fatal("expected exists=false for not-found PVC")
	}
}

func TestK8sGetPVCExists_NetworkError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("connection refused")
	}
	c := NewBackupClient(m)
	_, err := c.k8sGetPVCExists(context.Background(), "my-pvc", "ns")
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}

// ─── k8sListPodsUsingPVC ────────────────────────────────────────────────────

func TestK8sListPodsUsingPVC_MatchesClaim(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, _, _ string) ([]byte, error) {
		if resource != "pods" {
			return nil, errors.New("unexpected resource: " + resource)
		}
		return []byte(`{"items":[
			{"metadata":{"name":"pod-a"},"spec":{"volumes":[{"persistentVolumeClaim":{"claimName":"pvc-x"}}]}},
			{"metadata":{"name":"pod-b"},"spec":{"volumes":[{"persistentVolumeClaim":{"claimName":"pvc-y"}}]}}
		]}`), nil
	}
	c := NewBackupClient(m)
	names, err := c.k8sListPodsUsingPVC(context.Background(), "pvc-x", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "pod-a" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestK8sListPodsUsingPVC_NoClaim(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return []byte(`{"items":[{"metadata":{"name":"pod-a"},"spec":{"volumes":[]}}]}`), nil
	}
	c := NewBackupClient(m)
	names, err := c.k8sListPodsUsingPVC(context.Background(), "pvc-x", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected empty names, got %v", names)
	}
}

// ─── k8sGetPodLogs ──────────────────────────────────────────────────────────

func TestK8sGetPodLogs_ReturnsLogs(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnLogs = func(_ context.Context, name, namespace, _ string, _ bool) (io.ReadCloser, error) {
		if name != "pod-a" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return io.NopCloser(strings.NewReader("log line 1\nlog line 2")), nil
	}
	c := NewBackupClient(m)
	logs, err := c.k8sGetPodLogs(context.Background(), "pod-a", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(logs, "log line 1") {
		t.Fatalf("expected logs to contain 'log line 1', got %q", logs)
	}
}

func TestK8sGetPodLogs_RunnerError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnLogs = func(_ context.Context, _, _, _ string, _ bool) (io.ReadCloser, error) {
		return nil, errors.New("pod not found")
	}
	c := NewBackupClient(m)
	_, err := c.k8sGetPodLogs(context.Background(), "pod-a", "ns")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── k8sDeletePod ───────────────────────────────────────────────────────────

func TestK8sDeletePod_UsesRunner(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnDelete = func(_ context.Context, resource, name, namespace string, ignoreNotFound bool) error {
		if resource != "pod" || name != "pod-a" || namespace != "ns" || !ignoreNotFound {
			return errors.New("unexpected args")
		}
		return nil
	}
	c := NewBackupClient(m)
	if err := c.k8sDeletePod(context.Background(), "pod-a", "ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("Delete") != 1 {
		t.Fatalf("expected 1 Delete call, got %d", m.CallCount("Delete"))
	}
}

// ─── k8sListPodsUsingPVC error path ─────────────────────────────────────────

func TestK8sListPodsUsingPVC_RunnerError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("api error")
	}
	c := NewBackupClient(m)
	_, err := c.k8sListPodsUsingPVC(context.Background(), "pvc-x", "ns")
	if err == nil {
		t.Fatal("expected error from runner, got nil")
	}
}

// ─── k8sListPVCNames ────────────────────────────────────────────────────────

func TestK8sListPVCNames_ReturnsList(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, namespace, _ string) ([]byte, error) {
		if resource != "pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"items":[{"metadata":{"name":"pvc-a"}},{"metadata":{"name":"pvc-b"}}]}`), nil
	}
	c := NewBackupClient(m)
	names, err := c.k8sListPVCNames(context.Background(), "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "pvc-a" || names[1] != "pvc-b" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestK8sListPVCNames_RunnerError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("api error")
	}
	c := NewBackupClient(m)
	_, err := c.k8sListPVCNames(context.Background(), "ns")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── k8sGetPVCVolumeName ─────────────────────────────────────────────────────

func TestK8sGetPVCVolumeName_ReturnsName(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, resource, name, namespace string) ([]byte, error) {
		if resource != "pvc" || name != "my-pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"spec":{"volumeName":"pv-abc"}}`), nil
	}
	c := NewBackupClient(m)
	vol, err := c.k8sGetPVCVolumeName(context.Background(), "my-pvc", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vol != "pv-abc" {
		t.Fatalf("unexpected volumeName: %q", vol)
	}
}

// ─── k8sGetPVCPhase ──────────────────────────────────────────────────────────

func TestK8sGetPVCPhase_ReturnsBound(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, resource, name, namespace string) ([]byte, error) {
		if resource != "pvc" || name != "my-pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"status":{"phase":"Bound"}}`), nil
	}
	c := NewBackupClient(m)
	phase, err := c.k8sGetPVCPhase(context.Background(), "my-pvc", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phase != "Bound" {
		t.Fatalf("unexpected phase: %q", phase)
	}
}

func TestK8sGetPVCPhase_RunnerError(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("api error")
	}
	c := NewBackupClient(m)
	_, err := c.k8sGetPVCPhase(context.Background(), "my-pvc", "ns")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── k8sGetPodPhase ──────────────────────────────────────────────────────────

func TestK8sGetPodPhase_ReturnsRunning(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, resource, name, namespace string) ([]byte, error) {
		if resource != "pod" || name != "pod-a" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"status":{"phase":"Running"}}`), nil
	}
	c := NewBackupClient(m)
	phase, err := c.k8sGetPodPhase(context.Background(), "pod-a", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phase != "Running" {
		t.Fatalf("unexpected phase: %q", phase)
	}
}

// ─── k8sDeletePVCNoWait / k8sDeletePV ────────────────────────────────────────

func TestK8sDeletePVCNoWait_UsesRunner(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnDelete = func(_ context.Context, resource, name, namespace string, ignoreNotFound bool) error {
		if resource != "pvc" || name != "my-pvc" || namespace != "ns" || !ignoreNotFound {
			return errors.New("unexpected args")
		}
		return nil
	}
	c := NewBackupClient(m)
	if err := c.k8sDeletePVCNoWait(context.Background(), "my-pvc", "ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("Delete") != 1 {
		t.Fatalf("expected 1 Delete call, got %d", m.CallCount("Delete"))
	}
}

func TestK8sDeletePV_UsesRunner(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnDelete = func(_ context.Context, resource, name, namespace string, ignoreNotFound bool) error {
		if resource != "pv" || name != "pv-abc" || namespace != "" || !ignoreNotFound {
			return errors.New("unexpected args")
		}
		return nil
	}
	c := NewBackupClient(m)
	if err := c.k8sDeletePV(context.Background(), "pv-abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── SetDefaultK8sRunner ────────────────────────────────────────────────────

func TestSetDefaultK8sRunner_UpdatesDefaultClient(t *testing.T) {
	defaultMu.Lock()
	original := defaultClient
	defaultMu.Unlock()
	defer func() {
		defaultMu.Lock()
		defaultClient = original
		defaultMu.Unlock()
	}()

	m := &mock.K8sRunner{}
	SetDefaultK8sRunner(m)

	if getDefaultClient().k8sRunner == nil {
		t.Fatal("expected defaultClient.k8sRunner to be set after SetDefaultK8sRunner")
	}
}
