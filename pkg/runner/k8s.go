package runner

import (
	"context"
	"io"
	"time"
)

// K8sRunner defines the Kubernetes operations used across TRH SDK.
// It replaces 114 kubectl subprocess calls.
type K8sRunner interface {
	// Apply applies a YAML/JSON manifest (equivalent to kubectl apply -f).
	Apply(ctx context.Context, manifest []byte) error

	// Delete removes a named resource (equivalent to kubectl delete <resource> <name>).
	// ignoreNotFound suppresses errors when the resource does not exist.
	Delete(ctx context.Context, resource, name, namespace string, ignoreNotFound bool) error

	// Get fetches a single resource and returns its JSON representation.
	Get(ctx context.Context, resource, name, namespace string) ([]byte, error)

	// List returns a JSON list of resources matching an optional label selector.
	List(ctx context.Context, resource, namespace, labelSelector string) ([]byte, error)

	// Patch applies a JSON merge-patch to a resource.
	Patch(ctx context.Context, resource, name, namespace string, patch []byte) error

	// Wait blocks until the named resource satisfies the given condition or the
	// context is cancelled (equivalent to kubectl wait --for=condition=<cond>).
	Wait(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error

	// EnsureNamespace creates the namespace if it does not already exist.
	EnsureNamespace(ctx context.Context, namespace string) error

	// NamespaceExists reports whether the namespace currently exists in the cluster.
	NamespaceExists(ctx context.Context, namespace string) (bool, error)

	// Logs streams logs from the named pod/container.
	Logs(ctx context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error)
}
