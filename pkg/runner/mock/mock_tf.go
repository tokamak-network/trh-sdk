package mock

import (
	"context"
	"io"
	"sync"
)

// TFRunner is a thread-safe mock for runner.TFRunner.
// Configure expected results via the On* fields before calling methods.
// Recorded calls are available via Calls / CallCount / GetCalls after the test.
type TFRunner struct {
	mu    sync.Mutex
	Calls []Call

	OnInit         func(ctx context.Context, workDir string, env []string, backendConfigs []string) error
	OnApply        func(ctx context.Context, workDir string, env []string) error
	OnDestroy      func(ctx context.Context, workDir string, env []string) error
	OnSetStdout    func(w io.Writer)
	OnCheckVersion func(ctx context.Context) error
}

func (m *TFRunner) record(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

// CallCount returns how many times method was called.
func (m *TFRunner) CallCount(method string) int {
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
func (m *TFRunner) GetCalls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make([]Call, len(m.Calls))
	copy(snapshot, m.Calls)
	return snapshot
}

func (m *TFRunner) Init(ctx context.Context, workDir string, env []string, backendConfigs []string) error {
	m.record("Init", workDir, env, backendConfigs)
	if m.OnInit != nil {
		return m.OnInit(ctx, workDir, env, backendConfigs)
	}
	return nil
}

func (m *TFRunner) Apply(ctx context.Context, workDir string, env []string) error {
	m.record("Apply", workDir, env)
	if m.OnApply != nil {
		return m.OnApply(ctx, workDir, env)
	}
	return nil
}

func (m *TFRunner) Destroy(ctx context.Context, workDir string, env []string) error {
	m.record("Destroy", workDir, env)
	if m.OnDestroy != nil {
		return m.OnDestroy(ctx, workDir, env)
	}
	return nil
}

func (m *TFRunner) SetStdout(w io.Writer) {
	m.record("SetStdout")
	if m.OnSetStdout != nil {
		m.OnSetStdout(w)
	}
}

func (m *TFRunner) CheckVersion(ctx context.Context) error {
	m.record("CheckVersion")
	if m.OnCheckVersion != nil {
		return m.OnCheckVersion(ctx)
	}
	return nil
}
