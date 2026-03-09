package runner

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic/fake"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/apimachinery/pkg/runtime"
)

// ─── fake-client tests ───────────────────────────────────────────────────────

func newTestRunner(objs ...runtime.Object) *NativeK8sRunner {
	client := fakek8s.NewSimpleClientset(objs...)
	dynClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
	return &NativeK8sRunner{client: client, dynamic: dynClient}
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

	// Verify the namespace was created.
	exists, err := r.NamespaceExists(ctx, "new-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected namespace to exist after EnsureNamespace")
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
