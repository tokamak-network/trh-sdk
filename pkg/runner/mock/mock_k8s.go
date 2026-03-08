// Package mock provides test doubles for the runner interfaces.
// Use these in unit tests to avoid real cluster dependencies.
package mock

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

// Call records a single invocation of a mock method.
type Call struct {
	Method string
	Args   []interface{}
}

// K8sRunner is a minimal mock for runner.K8sRunner.
// Configure expected results via the On* fields before calling methods.
// Recorded calls are available via Calls after the test.
//
// Example:
//
//	m := &mock.K8sRunner{}
//	m.OnEnsureNamespace = func(ctx context.Context, ns string) error { return nil }
//	err := m.EnsureNamespace(ctx, "my-ns")
type K8sRunner struct {
	mu    sync.Mutex
	Calls []Call

	// Configurable handlers — set these before calling the method.
	// If a handler is nil the method returns a zero value and nil error.
	OnApply            func(ctx context.Context, manifest []byte) error
	OnDelete           func(ctx context.Context, resource, name, namespace string, ignoreNotFound bool) error
	OnGet              func(ctx context.Context, resource, name, namespace string) ([]byte, error)
	OnList             func(ctx context.Context, resource, namespace, labelSelector string) ([]byte, error)
	OnPatch            func(ctx context.Context, resource, name, namespace string, patch []byte) error
	OnWait             func(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error
	OnEnsureNamespace  func(ctx context.Context, namespace string) error
	OnNamespaceExists  func(ctx context.Context, namespace string) (bool, error)
	OnLogs             func(ctx context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error)
}

func (m *K8sRunner) record(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

// CallCount returns how many times method was called.
func (m *K8sRunner) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, c := range m.Calls {
		if c.Method == method {
			count++
		}
	}
	return count
}

func (m *K8sRunner) Apply(ctx context.Context, manifest []byte) error {
	m.record("Apply", manifest)
	if m.OnApply != nil {
		return m.OnApply(ctx, manifest)
	}
	return nil
}

func (m *K8sRunner) Delete(ctx context.Context, resource, name, namespace string, ignoreNotFound bool) error {
	m.record("Delete", resource, name, namespace, ignoreNotFound)
	if m.OnDelete != nil {
		return m.OnDelete(ctx, resource, name, namespace, ignoreNotFound)
	}
	return nil
}

func (m *K8sRunner) Get(ctx context.Context, resource, name, namespace string) ([]byte, error) {
	m.record("Get", resource, name, namespace)
	if m.OnGet != nil {
		return m.OnGet(ctx, resource, name, namespace)
	}
	return nil, nil
}

func (m *K8sRunner) List(ctx context.Context, resource, namespace, labelSelector string) ([]byte, error) {
	m.record("List", resource, namespace, labelSelector)
	if m.OnList != nil {
		return m.OnList(ctx, resource, namespace, labelSelector)
	}
	return []byte(`{"apiVersion":"v1","kind":"List","items":[]}`), nil
}

func (m *K8sRunner) Patch(ctx context.Context, resource, name, namespace string, patch []byte) error {
	m.record("Patch", resource, name, namespace, patch)
	if m.OnPatch != nil {
		return m.OnPatch(ctx, resource, name, namespace, patch)
	}
	return nil
}

func (m *K8sRunner) Wait(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error {
	m.record("Wait", resource, name, namespace, condition, timeout)
	if m.OnWait != nil {
		return m.OnWait(ctx, resource, name, namespace, condition, timeout)
	}
	return nil
}

func (m *K8sRunner) EnsureNamespace(ctx context.Context, namespace string) error {
	m.record("EnsureNamespace", namespace)
	if m.OnEnsureNamespace != nil {
		return m.OnEnsureNamespace(ctx, namespace)
	}
	return nil
}

func (m *K8sRunner) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	m.record("NamespaceExists", namespace)
	if m.OnNamespaceExists != nil {
		return m.OnNamespaceExists(ctx, namespace)
	}
	return false, nil
}

func (m *K8sRunner) Logs(ctx context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error) {
	m.record("Logs", pod, namespace, container, follow)
	if m.OnLogs != nil {
		return m.OnLogs(ctx, pod, namespace, container, follow)
	}
	return nil, errors.New("mock: Logs not configured")
}
