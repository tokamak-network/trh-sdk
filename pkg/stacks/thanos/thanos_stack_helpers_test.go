package thanos

import (
	"context"
	"errors"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// noopLogger returns a no-op SugaredLogger suitable for tests that do not
// assert on log output.
func noopLogger() *zap.SugaredLogger { return zap.NewNop().Sugar() }

// observedLogger returns a SugaredLogger whose output is captured in the returned
// *observer.ObservedLogs, allowing tests to assert on log calls.
func observedLogger(lvl zapcore.Level) (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, logs := observer.New(lvl)
	return zap.New(core).Sugar(), logs
}

// minimalStack returns a ThanosStack with only the helmRunner set,
// sufficient for testing helper methods that do not access other fields.
func minimalStack(hr *mock.HelmRunner) *ThanosStack {
	return &ThanosStack{helmRunner: hr}
}

// minimalK8sStack returns a ThanosStack with only the k8sRunner set.
func minimalK8sStack(kr *mock.K8sRunner) *ThanosStack {
	return &ThanosStack{k8sRunner: kr}
}

// ─── valueFiles guard tests ────────────────────────────────────────────────

// TestHelmUpgradeInstallWithFiles_ExtraArgsRejectedByRunner verifies that
// passing extraArgs while helmRunner is set returns an error (args would be silently dropped).
func TestHelmUpgradeInstallWithFiles_ExtraArgsRejectedByRunner(t *testing.T) {
	m := &mock.HelmRunner{}
	s := minimalStack(m)
	err := s.helmUpgradeInstallWithFiles(context.Background(), "rel", "chart", "ns",
		[]string{"values.yaml"}, "--atomic", "--timeout=5m")
	if err == nil {
		t.Fatal("expected error when extraArgs passed with helmRunner, got nil")
	}
	if m.CallCount("UpgradeWithFiles") != 0 {
		t.Fatal("expected UpgradeWithFiles not to be called when extraArgs are rejected")
	}
}

// TestHelmUpgradeInstallWithFiles_NoExtraArgsUsesRunner verifies the happy path.
func TestHelmUpgradeInstallWithFiles_NoExtraArgsUsesRunner(t *testing.T) {
	m := &mock.HelmRunner{}
	s := minimalStack(m)
	err := s.helmUpgradeInstallWithFiles(context.Background(), "rel", "chart", "ns",
		[]string{"values.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("UpgradeWithFiles") != 1 {
		t.Fatalf("expected 1 UpgradeWithFiles call, got %d", m.CallCount("UpgradeWithFiles"))
	}
}

// TestHelmInstallWithFiles_CallsUpgradeWithFilesOnRunner verifies that
// helmInstallWithFiles delegates to UpgradeWithFiles when helmRunner is set.
func TestHelmInstallWithFiles_CallsUpgradeWithFilesOnRunner(t *testing.T) {
	m := &mock.HelmRunner{}
	var gotRelease, gotChart, gotNamespace string
	var gotFiles []string
	m.OnUpgradeWithFiles = func(_ context.Context, release, chart, namespace string, files []string) error {
		gotRelease, gotChart, gotNamespace, gotFiles = release, chart, namespace, files
		return nil
	}
	s := minimalStack(m)
	err := s.helmInstallWithFiles(context.Background(), "my-rel", "my-chart", "my-ns",
		[]string{"v.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRelease != "my-rel" || gotChart != "my-chart" || gotNamespace != "my-ns" {
		t.Fatalf("unexpected args: release=%q chart=%q ns=%q", gotRelease, gotChart, gotNamespace)
	}
	if len(gotFiles) != 1 || gotFiles[0] != "v.yaml" {
		t.Fatalf("unexpected files: %v", gotFiles)
	}
}

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

// ─── k8sPVCPhase ───────────────────────────────────────────────────────────

func TestK8sPVCPhase_Bound(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, resource, name, namespace string) ([]byte, error) {
		if resource != "pvc" || name != "my-pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"status":{"phase":"Bound"}}`), nil
	}
	s := minimalK8sStack(m)
	phase, err := s.k8sPVCPhase(context.Background(), "my-pvc", "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phase != "Bound" {
		t.Fatalf("expected Bound, got %q", phase)
	}
	if m.CallCount("Get") != 1 {
		t.Fatalf("expected 1 Get call, got %d", m.CallCount("Get"))
	}
}

func TestK8sPVCPhase_NotFound(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return nil, errors.New("not found")
	}
	s := minimalK8sStack(m)
	phase, err := s.k8sPVCPhase(context.Background(), "missing", "ns")
	if err != nil {
		t.Fatalf("expected nil error for not-found, got %v", err)
	}
	if phase != "" {
		t.Fatalf("expected empty phase for not-found, got %q", phase)
	}
}

// ─── k8sPVCNames ───────────────────────────────────────────────────────────

func TestK8sPVCNames_ReturnsList(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, namespace, _ string) ([]byte, error) {
		if resource != "pvc" || namespace != "ns" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"items":[{"metadata":{"name":"pvc-a"}},{"metadata":{"name":"pvc-b"}}]}`), nil
	}
	s := minimalK8sStack(m)
	names, err := s.k8sPVCNames(context.Background(), "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "pvc-a" || names[1] != "pvc-b" {
		t.Fatalf("unexpected names: %v", names)
	}
}

// ─── k8sPodPVCClaims ───────────────────────────────────────────────────────

func TestK8sPodPVCClaims_ExtractsClaims(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, _, _ string) ([]byte, error) {
		if resource != "pods" {
			return nil, errors.New("unexpected resource: " + resource)
		}
		return []byte(`{"items":[{"spec":{"volumes":[{"persistentVolumeClaim":{"claimName":"pvc-x"}},{"name":"emptydir"}]}}]}`), nil
	}
	s := minimalK8sStack(m)
	claims, err := s.k8sPodPVCClaims(context.Background(), "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) != 1 || claims[0] != "pvc-x" {
		t.Fatalf("unexpected claims: %v", claims)
	}
}

// ─── k8sPVList ─────────────────────────────────────────────────────────────

func TestK8sPVList_ReturnsEntries(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnList = func(_ context.Context, resource, namespace, _ string) ([]byte, error) {
		if resource != "pv" || namespace != "" {
			return nil, errors.New("unexpected args")
		}
		return []byte(`{"items":[{"metadata":{"name":"pv-1"},"status":{"phase":"Released"}},{"metadata":{"name":"pv-2"},"status":{"phase":"Bound"}}]}`), nil
	}
	s := minimalK8sStack(m)
	entries, err := s.k8sPVList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "pv-1" || entries[0].Phase != "Released" {
		t.Fatalf("unexpected entry[0]: %+v", entries[0])
	}
}

// ─── k8sDeletePVC ──────────────────────────────────────────────────────────

func TestK8sDeletePVC_UsesRunner(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnDelete = func(_ context.Context, resource, name, namespace string, ignoreNotFound bool) error {
		if resource != "pvc" || name != "pvc-a" || namespace != "ns" || !ignoreNotFound {
			return errors.New("unexpected args")
		}
		return nil
	}
	s := minimalK8sStack(m)
	if err := s.k8sDeletePVC(context.Background(), "pvc-a", "ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("Delete") != 1 {
		t.Fatalf("expected 1 Delete call, got %d", m.CallCount("Delete"))
	}
}

// ─── k8sPatchPV ────────────────────────────────────────────────────────────

func TestK8sPatchPV_UsesRunnerWithEmptyNamespace(t *testing.T) {
	m := &mock.K8sRunner{}
	m.OnPatch = func(_ context.Context, resource, name, namespace string, patch []byte) error {
		if resource != "pv" || name != "pv-1" || namespace != "" {
			return errors.New("unexpected args")
		}
		return nil
	}
	s := minimalK8sStack(m)
	patch := []byte(`{"spec":{"claimRef":null}}`)
	if err := s.k8sPatchPV(context.Background(), "pv-1", patch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CallCount("Patch") != 1 {
		t.Fatalf("expected 1 Patch call, got %d", m.CallCount("Patch"))
	}
}
