package runner

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	fakek8s "k8s.io/client-go/kubernetes/fake"
)

// ─── fake-client tests ───────────────────────────────────────────────────────

func newTestRunner(objs ...runtime.Object) *NativeK8sRunner {
	client := fakek8s.NewSimpleClientset(objs...)
	// Register corev1 types so the fake dynamic client can find them.
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	dynClient := fake.NewSimpleDynamicClient(scheme, objs...)
	return &NativeK8sRunner{client: client, dynamic: dynClient, fieldManager: "trh-sdk"}
}

func TestNativeK8sRunner_Apply_CommentOnlyManifest(t *testing.T) {
	r := newTestRunner()
	// A manifest composed entirely of comments should be silently skipped;
	// no API calls are made and Apply must return nil.
	manifest := []byte("# this is a comment\n# another comment\n")
	if err := r.Apply(context.Background(), manifest); err != nil {
		t.Fatalf("unexpected error for comment-only manifest: %v", err)
	}
}

func TestNativeK8sRunner_NamespaceExists_True(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}}
	r := newTestRunner(ns)

	exists, err := r.NamespaceExists(context.Background(), "my-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestNativeK8sRunner_NamespaceExists_False(t *testing.T) {
	r := newTestRunner()

	exists, err := r.NamespaceExists(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false")
	}
}

func TestNativeK8sRunner_EnsureNamespace_Creates(t *testing.T) {
	r := newTestRunner()
	ctx := context.Background()

	if err := r.EnsureNamespace(ctx, "new-ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the namespace was created via the typed client.
	_, err := r.client.CoreV1().Namespaces().Get(ctx, "new-ns", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected namespace to exist after EnsureNamespace: %v", err)
	}
}

func TestNativeK8sRunner_EnsureNamespace_Idempotent(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "existing-ns"}}
	r := newTestRunner(ns)
	ctx := context.Background()

	// Should not return an error when the namespace already exists.
	if err := r.EnsureNamespace(ctx, "existing-ns"); err != nil {
		t.Fatalf("unexpected error on existing namespace: %v", err)
	}
}

// ─── input validation tests ──────────────────────────────────────────────────

func TestNativeK8sRunner_Delete_EmptyResource(t *testing.T) {
	r := newTestRunner()
	err := r.Delete(context.Background(), "", "my-obj", "default", false)
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestNativeK8sRunner_Delete_EmptyName(t *testing.T) {
	r := newTestRunner()
	err := r.Delete(context.Background(), "pods", "", "default", false)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNativeK8sRunner_Get_EmptyResource(t *testing.T) {
	r := newTestRunner()
	_, err := r.Get(context.Background(), "", "my-obj", "default")
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestNativeK8sRunner_Get_EmptyName(t *testing.T) {
	r := newTestRunner()
	_, err := r.Get(context.Background(), "pods", "", "default")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNativeK8sRunner_List_EmptyResource(t *testing.T) {
	r := newTestRunner()
	_, err := r.List(context.Background(), "", "default", "")
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestNativeK8sRunner_Patch_EmptyResource(t *testing.T) {
	r := newTestRunner()
	err := r.Patch(context.Background(), "", "my-obj", "default", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty resource")
	}
}

func TestNativeK8sRunner_Patch_EmptyName(t *testing.T) {
	r := newTestRunner()
	err := r.Patch(context.Background(), "pods", "", "default", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNativeK8sRunner_Patch_InvalidJSON(t *testing.T) {
	r := newTestRunner()
	err := r.Patch(context.Background(), "pods", "my-pod", "default", []byte(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON patch")
	}
}

func TestNativeK8sRunner_Logs_CancelledContext(t *testing.T) {
	r := newTestRunner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before call

	_, err := r.Logs(ctx, "my-pod", "default", "", false)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestNativeK8sRunner_Patch_NamespaceMutation(t *testing.T) {
	r := newTestRunner()
	patch := []byte(`{"metadata":{"namespace":"kube-system"}}`)
	err := r.Patch(context.Background(), "pods", "my-pod", "default", patch)
	if err == nil {
		t.Fatal("expected error when patch attempts to change namespace")
	}
}

// ─── pure-function: normaliseResourceName extras ─────────────────────────────

func TestNormaliseResourceName_NewAliases(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"endpoint", "endpoints"},
		{"rolebinding", "rolebindings"},
		{"networkpolicy", "networkpolicies"},
		{"crd", "customresourcedefinitions"},
		{"hpa", "horizontalpodautoscalers"},
		{"role", "roles"},
	}
	for _, tc := range cases {
		got := normaliseResourceName(tc.input)
		if got != tc.want {
			t.Errorf("normaliseResourceName(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestCheckCondition_ExactStatusMatch(t *testing.T) {
	// "true" (lowercase) must NOT match — Kubernetes spec requires exactly "True".
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "true", // lowercase — invalid K8s value
					},
				},
			},
		},
	}
	if checkCondition(obj, "Available") {
		t.Fatal("expected no match for status='true' (lowercase); only 'True' is valid")
	}
}

// ─── pure-function tests ─────────────────────────────────────────────────────

func TestNormaliseResourceName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"pod", "pods"},
		{"pods", "pods"},
		{"namespace", "namespaces"},
		{"namespaces", "namespaces"},
		{"pvc", "persistentvolumeclaims"},
		{"svc", "services"},
		{"service", "services"},
		{"deployment", "deployments"},
		{"configmap", "configmaps"},
		{"SECRET", "secrets"},       // case-insensitive lookup
		{"unknown-resource", "unknown-resource"}, // pass-through
	}
	for _, tc := range cases {
		got := normaliseResourceName(tc.input)
		if got != tc.want {
			t.Errorf("normaliseResourceName(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestSplitYAMLDocuments_Single(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: Pod")
	docs := splitYAMLDocuments(input)
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_Multi(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: Pod\n---\napiVersion: v1\nkind: Service")
	docs := splitYAMLDocuments(input)
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_SkipsEmpty(t *testing.T) {
	input := []byte("apiVersion: v1\nkind: Pod\n---\n\n---\napiVersion: v1\nkind: Service")
	docs := splitYAMLDocuments(input)
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs (empty skipped), got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_LeadingSeparator(t *testing.T) {
	// Documents that begin with "---" should be parsed correctly.
	input := []byte("---\napiVersion: v1\nkind: Pod\n---\napiVersion: v1\nkind: Service")
	docs := splitYAMLDocuments(input)
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs with leading ---, got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_ConsecutiveSeparators(t *testing.T) {
	// Consecutive --- separators (empty documents between them) should be skipped.
	input := []byte("---\n---\napiVersion: v1\nkind: Pod\nmetadata:\n  name: foo")
	docs := splitYAMLDocuments(input)
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc (empty docs skipped), got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_CRLF(t *testing.T) {
	// Windows-style CRLF line endings must be normalised before splitting.
	input := []byte("apiVersion: v1\r\nkind: Pod\r\n---\r\napiVersion: v1\r\nkind: Service")
	docs := splitYAMLDocuments(input)
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs with CRLF line endings, got %d", len(docs))
	}
}

func TestSplitYAMLDocuments_CommentOnly(t *testing.T) {
	// A document containing only a comment should be detected by isEmptyOrCommentOnlyYAML.
	commentOnly := []byte("# just a comment\n# another comment")
	if !isEmptyOrCommentOnlyYAML(commentOnly) {
		t.Fatal("expected comment-only document to be detected")
	}
	// Empty document is also considered "empty or comment-only".
	if !isEmptyOrCommentOnlyYAML([]byte("   \n   ")) {
		t.Fatal("expected whitespace-only document to be detected")
	}
	// A document with a comment followed by real YAML must NOT be treated as comment-only.
	mixed := []byte("# comment\napiVersion: v1")
	if isEmptyOrCommentOnlyYAML(mixed) {
		t.Fatal("expected mixed comment+YAML document to NOT be treated as comment-only")
	}
}

func TestCheckCondition_True(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "True",
					},
				},
			},
		},
	}
	if !checkCondition(obj, "Available") {
		t.Fatal("expected condition Available=True to match")
	}
}

func TestCheckCondition_CaseInsensitive(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		},
	}
	if !checkCondition(obj, "ready") {
		t.Fatal("expected case-insensitive match for condition 'ready'")
	}
}

func TestCheckCondition_False(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "False",
					},
				},
			},
		},
	}
	if checkCondition(obj, "Available") {
		t.Fatal("expected Available=False to not match")
	}
}

func TestCheckCondition_NoConditions(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if checkCondition(obj, "Available") {
		t.Fatal("expected false for object with no conditions")
	}
}
