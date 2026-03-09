package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const gvrCacheTTL = 60 * time.Second

// gvrEntry is a single cached discovery result.
type gvrEntry struct {
	gvr        schema.GroupVersionResource
	namespaced bool
	cachedAt   time.Time
}

// NativeK8sRunner implements K8sRunner using k8s.io/client-go.
// No kubectl binary is required.
type NativeK8sRunner struct {
	client   kubernetes.Interface
	dynamic  dynamic.Interface
	gvrCache sync.Map // map[string]*gvrEntry, keyed by "gvk:<G>/<V>/<K>" or "res:<name>"
}

// newNativeK8sRunner creates a NativeK8sRunner from the given kubeconfig path.
// If kubeconfigPath is empty the runner uses in-cluster config or ~/.kube/config.
func newNativeK8sRunner(kubeconfigPath string) (*NativeK8sRunner, error) {
	cfg, err := loadKubeConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("native k8s runner: load kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("native k8s runner: build typed client: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("native k8s runner: build dynamic client: %w", err)
	}

	return &NativeK8sRunner{client: client, dynamic: dynClient}, nil
}

// loadKubeConfig returns a *rest.Config from the given path, in-cluster env,
// or the user's default kubeconfig (~/.kube/config).
func loadKubeConfig(path string) (*rest.Config, error) {
	if path != "" {
		return clientcmd.BuildConfigFromFlags("", path)
	}
	// Try in-cluster first (running inside a Pod).
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	// Fall back to ~/.kube/config.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home dir: %w", err)
	}
	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}

// Apply decodes a YAML/JSON manifest and applies it via server-side apply.
// Multi-document YAML (---) is supported.
func (r *NativeK8sRunner) Apply(ctx context.Context, manifest []byte) error {
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	docs := splitYAMLDocuments(manifest)

	for _, doc := range docs {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode(doc, nil, obj)
		if err != nil {
			return fmt.Errorf("native apply: decode manifest: %w", err)
		}

		// Validate identity before calling the API.
		if obj.GetName() == "" && obj.GetGenerateName() == "" {
			return fmt.Errorf("native apply: %s/%s manifest is missing metadata.name and metadata.generateName", gvk.Group, gvk.Kind)
		}

		entry, err := r.resolveGVR(ctx, gvk)
		if err != nil {
			return fmt.Errorf("native apply: resolve GVR for %s: %w", gvk.Kind, err)
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("native apply: marshal object: %w", err)
		}

		var dynRes dynamic.ResourceInterface
		if entry.namespaced {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = "default"
			}
			dynRes = r.dynamic.Resource(entry.gvr).Namespace(ns)
		} else {
			dynRes = r.dynamic.Resource(entry.gvr)
		}

		_, err = dynRes.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "trh-sdk",
		})
		if err != nil {
			return fmt.Errorf("native apply: server-side apply %s/%s: %w", gvk.Kind, obj.GetName(), err)
		}
	}
	return nil
}

// Delete removes the named resource from the cluster.
func (r *NativeK8sRunner) Delete(ctx context.Context, resource, name, namespace string, ignoreNotFound bool) error {
	if resource == "" {
		return fmt.Errorf("native delete: resource name cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("native delete: object name cannot be empty")
	}
	entry, err := r.resolveGVRByResource(ctx, resource)
	if err != nil {
		return fmt.Errorf("native delete: resolve resource %s: %w", resource, err)
	}

	var dynRes dynamic.ResourceInterface
	if namespace != "" {
		dynRes = r.dynamic.Resource(entry.gvr).Namespace(namespace)
	} else {
		dynRes = r.dynamic.Resource(entry.gvr)
	}

	err = dynRes.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if ignoreNotFound && k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("native delete %s/%s: %w", resource, name, err)
	}
	return nil
}

// Get fetches a resource and returns it as JSON bytes.
func (r *NativeK8sRunner) Get(ctx context.Context, resource, name, namespace string) ([]byte, error) {
	if resource == "" {
		return nil, fmt.Errorf("native get: resource name cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("native get: object name cannot be empty")
	}
	entry, err := r.resolveGVRByResource(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("native get: resolve resource %s: %w", resource, err)
	}

	var obj *unstructured.Unstructured
	if namespace != "" {
		obj, err = r.dynamic.Resource(entry.gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = r.dynamic.Resource(entry.gvr).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("native get %s/%s: %w", resource, name, err)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("native get: marshal: %w", err)
	}
	return data, nil
}

// List returns a JSON list of resources filtered by an optional label selector.
func (r *NativeK8sRunner) List(ctx context.Context, resource, namespace, labelSelector string) ([]byte, error) {
	if resource == "" {
		return nil, fmt.Errorf("native list: resource name cannot be empty")
	}
	entry, err := r.resolveGVRByResource(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("native list: resolve resource %s: %w", resource, err)
	}

	listOpts := metav1.ListOptions{}
	if labelSelector != "" {
		listOpts.LabelSelector = labelSelector
	}

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = r.dynamic.Resource(entry.gvr).Namespace(namespace).List(ctx, listOpts)
	} else {
		list, err = r.dynamic.Resource(entry.gvr).List(ctx, listOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("native list %s: %w", resource, err)
	}

	data, err := json.Marshal(list)
	if err != nil {
		return nil, fmt.Errorf("native list: marshal: %w", err)
	}
	return data, nil
}

// Patch applies a JSON merge-patch to an existing resource.
// Returns an error immediately if patch is not valid JSON.
func (r *NativeK8sRunner) Patch(ctx context.Context, resource, name, namespace string, patch []byte) error {
	if resource == "" {
		return fmt.Errorf("native patch: resource name cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("native patch: object name cannot be empty")
	}
	if !json.Valid(patch) {
		return fmt.Errorf("native patch: invalid JSON payload for %s/%s", resource, name)
	}

	entry, err := r.resolveGVRByResource(ctx, resource)
	if err != nil {
		return fmt.Errorf("native patch: resolve resource %s: %w", resource, err)
	}

	var dynRes dynamic.ResourceInterface
	if namespace != "" {
		dynRes = r.dynamic.Resource(entry.gvr).Namespace(namespace)
	} else {
		dynRes = r.dynamic.Resource(entry.gvr)
	}

	_, err = dynRes.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("native patch %s/%s: %w", resource, name, err)
	}
	return nil
}

// Wait polls until the named resource satisfies condition or the context is cancelled.
func (r *NativeK8sRunner) Wait(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error {
	entry, err := r.resolveGVRByResource(ctx, resource)
	if err != nil {
		return fmt.Errorf("native wait: resolve resource %s: %w", resource, err)
	}

	// PollUntilContextTimeout enforces the timeout internally; no child context needed.
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		// Use a closure-local error variable to avoid mutating the outer err.
		var obj *unstructured.Unstructured
		var pollErr error
		if namespace != "" {
			obj, pollErr = r.dynamic.Resource(entry.gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		} else {
			obj, pollErr = r.dynamic.Resource(entry.gvr).Get(ctx, name, metav1.GetOptions{})
		}
		if pollErr != nil {
			if k8serrors.IsNotFound(pollErr) {
				return false, nil // resource not yet created
			}
			return false, pollErr
		}
		return checkCondition(obj, condition), nil
	})
}

// EnsureNamespace creates the namespace if it does not already exist.
// It uses an optimistic create-then-check pattern to avoid TOCTOU races.
func (r *NativeK8sRunner) EnsureNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	_, err := r.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("native create namespace %s: %w", namespace, err)
	}
	return nil
}

// NamespaceExists reports whether the namespace exists in the cluster.
func (r *NativeK8sRunner) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := r.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("native namespace exists check: %w", err)
	}
	return true, nil
}

// Logs opens a streaming log connection to the named pod.
func (r *NativeK8sRunner) Logs(ctx context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("native logs: context already done: %w", err)
	}
	opts := &corev1.PodLogOptions{
		Container: container,
		Follow:    follow,
	}
	req := r.client.CoreV1().Pods(namespace).GetLogs(pod, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("native logs %s/%s: %w", namespace, pod, err)
	}
	return stream, nil
}

// ─── discovery helpers ───────────────────────────────────────────────────────

// resolveGVR maps a GVK to a GroupVersionResource via server-side discovery.
// Results are cached for gvrCacheTTL to avoid redundant API round-trips.
func (r *NativeK8sRunner) resolveGVR(ctx context.Context, gvk *schema.GroupVersionKind) (*gvrEntry, error) {
	key := fmt.Sprintf("gvk:%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	if cached := r.loadCachedGVR(key); cached != nil {
		return cached, nil
	}

	lists, err := r.client.Discovery().ServerPreferredResources()
	// ServerPreferredResources returns partial results even when some groups fail.
	// Only hard-fail when no lists were returned at all.
	if lists == nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if gv.Group != gvk.Group || gv.Version != gvk.Version {
			continue
		}
		for _, res := range list.APIResources {
			if res.Kind == gvk.Kind {
				entry := &gvrEntry{
					gvr: schema.GroupVersionResource{
						Group: gv.Group, Version: gv.Version, Resource: res.Name,
					},
					namespaced: res.Namespaced,
					cachedAt:   time.Now(),
				}
				r.gvrCache.Store(key, entry)
				return entry, nil
			}
		}
	}
	return nil, fmt.Errorf("resource not found for GVK %s", gvk)
}

// resolveGVRByResource maps a short resource name (e.g. "pods", "pvc") to a
// GroupVersionResource via server-side discovery. Results are cached for gvrCacheTTL.
func (r *NativeK8sRunner) resolveGVRByResource(ctx context.Context, resource string) (*gvrEntry, error) {
	target := normaliseResourceName(resource)
	key := "res:" + target
	if cached := r.loadCachedGVR(key); cached != nil {
		return cached, nil
	}

	lists, err := r.client.Discovery().ServerPreferredResources()
	if lists == nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for _, res := range list.APIResources {
			if strings.EqualFold(res.Name, target) || matchesShortName(res.ShortNames, target) {
				entry := &gvrEntry{
					gvr: schema.GroupVersionResource{
						Group: gv.Group, Version: gv.Version, Resource: res.Name,
					},
					namespaced: res.Namespaced,
					cachedAt:   time.Now(),
				}
				r.gvrCache.Store(key, entry)
				return entry, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown resource: %s", resource)
}

// loadCachedGVR returns a valid (non-expired) cached entry or nil.
func (r *NativeK8sRunner) loadCachedGVR(key string) *gvrEntry {
	v, ok := r.gvrCache.Load(key)
	if !ok {
		return nil
	}
	entry, ok := v.(*gvrEntry)
	if !ok || time.Since(entry.cachedAt) > gvrCacheTTL {
		r.gvrCache.Delete(key)
		return nil
	}
	return entry
}

// matchesShortName reports whether target equals any entry in shortNames (case-insensitive).
func matchesShortName(shortNames []string, target string) bool {
	for _, sn := range shortNames {
		if strings.EqualFold(sn, target) {
			return true
		}
	}
	return false
}

// ─── pure helpers (no IO) ────────────────────────────────────────────────────

// normaliseResourceName maps common short forms to their canonical plural names.
func normaliseResourceName(r string) string {
	aliases := map[string]string{
		"namespace":             "namespaces",
		"pod":                   "pods",
		"svc":                   "services",
		"service":               "services",
		"ingress":               "ingresses",
		"pvc":                   "persistentvolumeclaims",
		"persistentvolumeclaim": "persistentvolumeclaims",
		"pv":                    "persistentvolumes",
		"persistentvolume":      "persistentvolumes",
		"deployment":            "deployments",
		"configmap":             "configmaps",
		"clusterrolebinding":    "clusterrolebindings",
		"clusterrole":           "clusterroles",
		"serviceaccount":        "serviceaccounts",
		"secret":                "secrets",
		"storageclass":          "storageclasses",
		"statefulset":           "statefulsets",
		"daemonset":             "daemonsets",
		"job":                   "jobs",
		"cronjob":               "cronjobs",
	}
	if canonical, ok := aliases[strings.ToLower(r)]; ok {
		return canonical
	}
	return r
}

// checkCondition inspects an unstructured object's .status.conditions array.
func checkCondition(obj *unstructured.Unstructured, condition string) bool {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return false
	}
	target := strings.ToLower(condition)
	for _, c := range conditions {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := cMap["type"].(string)
		s, _ := cMap["status"].(string)
		// Kubernetes condition status values are exactly "True", "False", or "Unknown".
		if strings.ToLower(t) == target && s == "True" {
			return true
		}
	}
	return false
}

// splitYAMLDocuments splits a multi-document YAML byte slice on "---" separators.
// It correctly handles documents that start with "---" and skips empty documents.
func splitYAMLDocuments(data []byte) [][]byte {
	// Normalise: if the document starts with "---", drop that prefix so the
	// subsequent split on "\n---" handles it uniformly.
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("---")) {
		trimmed = trimmed[3:]
	}

	parts := bytes.Split(trimmed, []byte("\n---"))
	result := make([][]byte, 0, len(parts))
	for _, p := range parts {
		// Strip any leading "---" left after the split (edge case: "\n---\n---")
		p = bytes.TrimPrefix(bytes.TrimSpace(p), []byte("---"))
		if len(bytes.TrimSpace(p)) > 0 {
			result = append(result, p)
		}
	}
	return result
}
