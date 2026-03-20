package runner_test

// bench_test.go — Shell-out vs Native runner overhead comparison.
//
// 실행 방법:
//   go test -bench=. -benchmem ./pkg/runner/
//   go test -bench=BenchmarkForkOverhead -benchtime=5s ./pkg/runner/
//
// 이 벤치마크는 실제 클러스터 없이 실행됩니다.
// "Shell-out" 경로는 fork+exec 비용을 직접 측정하고,
// "Native" 경로는 mock runner(함수 호출 + 채널)를 통해 라이브러리 호출 오버헤드를 측정합니다.

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
)

// ─── 1. fork+exec 오버헤드 ─────────────────────────────────────────────────
// 모든 shell-out 호출은 이 비용을 기본으로 지불합니다.
// kubectl, helm, aws, terraform 모두 동일하게 적용됩니다.

// BenchmarkForkOverhead_True — 실제 fork+exec 비용 (최소 명령어 'true')
func BenchmarkForkOverhead_True(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := exec.CommandContext(ctx, "true").Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkForkOverhead_Echo — 가벼운 출력 포함 fork+exec
func BenchmarkForkOverhead_Echo(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out, err := exec.CommandContext(ctx, "echo", "benchmark").Output()
		if err != nil {
			b.Fatal(err)
		}
		_ = out
	}
}

// ─── 2. Native (mock) 오버헤드 ────────────────────────────────────────────
// 실제 클러스터 없이 native 경로의 호출 오버헤드를 측정합니다.
// mock runner = 함수 호출 + record() + 핸들러 실행 — 네트워크 없음.

// BenchmarkNativeK8s_Get — K8sRunner.Get 모의 호출 오버헤드
func BenchmarkNativeK8s_Get(b *testing.B) {
	ctx := context.Background()
	m := &mock.K8sRunner{}
	payload := []byte(`{"kind":"Pod","metadata":{"name":"test"}}`)
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return payload, nil
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := m.Get(ctx, "pods", "test-pod", "default")
		if err != nil {
			b.Fatal(err)
		}
		_ = out
	}
}

// BenchmarkNativeHelm_List — HelmRunner.List 모의 호출 오버헤드
func BenchmarkNativeHelm_List(b *testing.B) {
	ctx := context.Background()
	m := &mock.HelmRunner{}
	releases := []string{"release-a", "release-b", "release-c"}
	m.OnList = func(_ context.Context, _ string) ([]string, error) {
		return releases, nil
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := m.List(ctx, "default")
		if err != nil {
			b.Fatal(err)
		}
		_ = out
	}
}

// ─── 3. 반복 호출 비교: 10회 shell-out vs 10회 native ─────────────────────
// 실제 배포에서는 단일 호출이 아닌 여러 번 연속 호출합니다.
// kubectl get pod을 10번 연속으로 호출하는 비용 차이를 보여줍니다.

// BenchmarkShellOut_Repeated10 — echo 10회 연속 shell-out
func BenchmarkShellOut_Repeated10(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			if err := exec.CommandContext(ctx, "true").Run(); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkNative_Repeated10 — mock K8sRunner.Get 10회 연속 호출
func BenchmarkNative_Repeated10(b *testing.B) {
	ctx := context.Background()
	m := &mock.K8sRunner{}
	m.OnGet = func(_ context.Context, _, _, _ string) ([]byte, error) {
		return []byte(`{}`), nil
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_, err := m.Get(ctx, "pods", "pod", "ns")
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// ─── 4. 로그 스트리밍 비교 ────────────────────────────────────────────────
// shell-out은 로그 스트리밍을 지원하지 않습니다 (ShellOutK8sRunner.Logs → 에러).
// native는 io.Reader를 직접 반환합니다.

// BenchmarkNativeLogs_ReadAll — K8sRunner.Logs mock + io.ReadAll 오버헤드
func BenchmarkNativeLogs_ReadAll(b *testing.B) {
	ctx := context.Background()
	const logData = "2026-03-09T00:00:00Z INFO  block produced height=100\n"
	m := &mock.K8sRunner{}
	m.OnLogs = func(_ context.Context, _, _, _ string, _ bool) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(logData)), nil
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc, err := m.Logs(ctx, "op-node-0", "default", "", false)
		if err != nil {
			b.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		if err != nil {
			b.Fatal(err)
		}
		rc.Close() //nolint:errcheck
		_ = data
	}
}
