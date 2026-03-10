package mock

import (
	"context"
	"sync"
)

// DORunner is a thread-safe, hand-written mock for runner.DORunner.
// Configure expected results via the On* fields before calling methods.
// Recorded calls are available via Calls / CallCount / GetCalls after the test.
//
// Example:
//
//	m := &mock.DORunner{}
//	m.OnValidateToken = func(ctx context.Context, token string) error { return nil }
//	err := m.ValidateToken(ctx, "my-token")
type DORunner struct {
	mu    sync.Mutex
	Calls []Call

	OnValidateToken func(ctx context.Context, token string) error
	OnListRegions   func(ctx context.Context, token string) ([]string, error)
	OnGetKubeconfig func(ctx context.Context, clusterName, token string) error
	OnCheckVersion  func(ctx context.Context) error
}

func (m *DORunner) record(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

// CallCount returns how many times method was called.
func (m *DORunner) CallCount(method string) int {
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

// GetCalls returns a snapshot of all recorded calls, safe for concurrent reads.
func (m *DORunner) GetCalls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make([]Call, len(m.Calls))
	copy(snapshot, m.Calls)
	return snapshot
}

func (m *DORunner) ValidateToken(ctx context.Context, token string) error {
	m.record("ValidateToken", token)
	if m.OnValidateToken != nil {
		return m.OnValidateToken(ctx, token)
	}
	return nil
}

func (m *DORunner) ListRegions(ctx context.Context, token string) ([]string, error) {
	m.record("ListRegions", token)
	if m.OnListRegions != nil {
		return m.OnListRegions(ctx, token)
	}
	return nil, nil
}

func (m *DORunner) GetKubeconfig(ctx context.Context, clusterName, token string) error {
	m.record("GetKubeconfig", clusterName, token)
	if m.OnGetKubeconfig != nil {
		return m.OnGetKubeconfig(ctx, clusterName, token)
	}
	return nil
}

func (m *DORunner) CheckVersion(ctx context.Context) error {
	m.record("CheckVersion")
	if m.OnCheckVersion != nil {
		return m.OnCheckVersion(ctx)
	}
	return nil
}
