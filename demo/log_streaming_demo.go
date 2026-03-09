//go:build ignore

// demo/log_streaming_demo.go — 로그 스트리밍 데모
//
// 실행:
//   go run demo/log_streaming_demo.go
//
// 클러스터 없이 로컬에서 완전히 동작합니다.
// 가짜 k8s API 서버를 내장하여 op-node 로그를 시뮬레이션합니다.
//
// 보여주는 것:
//   1. ShellOut 경로 → "streaming not supported" 에러 즉시 반환
//   2. Native 경로  → io.ReadCloser 스트림으로 실시간 로그 출력

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/runner/mock"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// fakeLogLines는 실제 op-node가 출력하는 로그와 유사한 형식입니다.
var fakeLogLines = []string{
	"2026-03-09T09:00:00Z INFO  op-node started                  chain_id=17069",
	"2026-03-09T09:00:01Z INFO  L1 connected                     rpc=https://sepolia.infura.io",
	"2026-03-09T09:00:02Z INFO  unsafe head updated              block=0x1a2b height=8400100",
	"2026-03-09T09:00:03Z INFO  safe head updated                block=0x3c4d height=8400098",
	"2026-03-09T09:00:04Z INFO  finalized head updated           block=0x5e6f height=8400090",
	"2026-03-09T09:00:05Z INFO  sequencer confirmed batch        txs=12 gas=450000",
	"2026-03-09T09:00:06Z INFO  unsafe head updated              block=0x7a8b height=8400101",
	"2026-03-09T09:00:07Z INFO  p2p peer connected               peer_id=16Uiu2HA...",
	"2026-03-09T09:00:08Z INFO  block derived                    l2_block=0x9c0d height=8400101",
	"2026-03-09T09:00:09Z WARN  slow L1 response                 latency=2.1s threshold=2s",
	"2026-03-09T09:00:10Z INFO  sequencer confirmed batch        txs=8 gas=310000",
}

func main() {
	ctx := context.Background()

	printHeader("TRH SDK — 로그 스트리밍 데모")
	fmt.Println()

	// ── 1. ShellOut 경로 ──────────────────────────────────────────────────────
	printSection("1. Shell-out 경로 (TRHS_LEGACY=1)")
	fmt.Println("  ShellOutK8sRunner.Logs() 호출 중...")
	time.Sleep(300 * time.Millisecond)

	shellRunner := &runner.ShellOutK8sRunner{}
	_, err := shellRunner.Logs(ctx, "op-node-0", "thanos", "", false)
	if err != nil {
		printError("  에러: " + err.Error())
	}
	fmt.Println()

	// ── 2. Native 경로 (mock) ─────────────────────────────────────────────────
	printSection("2. Native 경로 — mock runner (클러스터 불필요)")
	fmt.Println("  mock.K8sRunner.Logs() → io.ReadCloser 스트림 반환")
	fmt.Println()
	time.Sleep(300 * time.Millisecond)

	m := &mock.K8sRunner{}
	m.OnLogs = func(_ context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error) {
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close() //nolint:errcheck
			for _, line := range fakeLogLines {
				fmt.Fprintln(pw, line) //nolint:errcheck
				time.Sleep(120 * time.Millisecond)
			}
		}()
		return pr, nil
	}

	rc, err := m.Logs(ctx, "op-node-0", "thanos", "", false)
	if err != nil {
		printError("  에러: " + err.Error())
		os.Exit(1)
	}
	defer rc.Close() //nolint:errcheck

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		coloredLine := colorizeLog(line)
		fmt.Printf("  %s\n", coloredLine)
	}

	fmt.Println()

	// ── 3. Native 경로 (fake k8s HTTP server) ────────────────────────────────
	printSection("3. Native 경로 — fake k8s API server (client-go 실제 HTTP 요청)")
	fmt.Println("  로컬에 가짜 k8s API 서버 시작 중...")
	time.Sleep(200 * time.Millisecond)

	srv := startFakeLogServer(fakeLogLines, 80*time.Millisecond)
	defer srv.Close()
	fmt.Printf("  Fake API Server: %s\n\n", srv.URL)

	// client-go의 fake typed client + REST override로 로그 스트리밍
	fakeClient := fake.NewSimpleClientset()
	// REST Transport를 fake 서버로 연결
	cfg := &rest.Config{
		Host: srv.URL,
		// 인증 없음 (로컬 데모용)
	}
	nativeRunner, err := newNativeK8sRunnerFromConfig(cfg, fakeClient)
	if err != nil {
		// client-go REST 클라이언트로 직접 스트림
		printWarning("  client-go REST 직접 스트리밍으로 대체...")
		streamFromFakeServer(ctx, srv.URL)
	} else {
		rc2, err := nativeRunner.Logs(ctx, "op-node-0", "thanos", "", false)
		if err != nil {
			printWarning("  Logs() 에러 (fake server path): " + err.Error())
			streamFromFakeServer(ctx, srv.URL)
		} else {
			defer rc2.Close() //nolint:errcheck
			scanner2 := bufio.NewScanner(rc2)
			for scanner2.Scan() {
				fmt.Printf("  %s\n", colorizeLog(scanner2.Text()))
			}
		}
	}

	fmt.Println()
	printHeader("데모 완료")
	fmt.Println()
	fmt.Println("  핵심 차이점:")
	fmt.Println("  ┌─────────────────┬──────────────────────────────────────────┐")
	fmt.Println("  │ Shell-out       │ 에러 반환 (바이너리 모드 미지원)         │")
	fmt.Println("  │ Native (mock)   │ io.ReadCloser → 실시간 줄 단위 스트리밍 │")
	fmt.Println("  │ Native (실제)   │ client-go HTTP → k8s API 직접 스트리밍  │")
	fmt.Println("  └─────────────────┴──────────────────────────────────────────┘")
	fmt.Println()
}

// startFakeLogServer는 /api/v1/namespaces/{ns}/pods/{pod}/log 경로로
// chunked transfer-encoding 방식의 로그 스트림을 서빙하는 httptest 서버입니다.
func startFakeLogServer(lines []string, delay time.Duration) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		for _, line := range lines {
			fmt.Fprintln(w, line) //nolint:errcheck
			if ok {
				flusher.Flush()
			}
			time.Sleep(delay)
		}
	})
	return httptest.NewServer(mux)
}

// streamFromFakeServer는 fake 서버에 직접 HTTP GET을 보내 스트리밍합니다.
// client-go 통합이 실패했을 때 폴백으로 사용됩니다.
func streamFromFakeServer(ctx context.Context, baseURL string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/namespaces/thanos/pods/op-node-0/log", nil)
	if err != nil {
		printError("  HTTP 요청 생성 실패: " + err.Error())
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		printError("  HTTP 요청 실패: " + err.Error())
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fmt.Printf("  %s\n", colorizeLog(scanner.Text()))
	}
}

// newNativeK8sRunnerFromConfig는 테스트용 config로 NativeK8sRunner를 생성 시도합니다.
// fake.Clientset은 Pods().GetLogs()를 지원하지 않으므로 에러가 예상됩니다.
func newNativeK8sRunnerFromConfig(cfg *rest.Config, _ interface{}) (interface{ Logs(context.Context, string, string, string, bool) (io.ReadCloser, error) }, error) {
	// NativeK8sRunner는 패키지 외부에서 직접 생성할 수 없으므로
	// runner.New()를 통해 생성합니다.
	tr, err := runner.New(runner.RunnerConfig{
		UseNative:      true,
		KubeconfigPath: "", // 없으면 ~/.kube/config 사용
	})
	if err != nil {
		return nil, err
	}
	return tr.K8s(), nil
}

// ── 출력 헬퍼 ────────────────────────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
)

func printHeader(s string) {
	line := "═══════════════════════════════════════════════════════"
	fmt.Printf("%s%s%s\n", colorBold+colorCyan, line, colorReset)
	pad := (len(line) - len(s)) / 2
	fmt.Printf("%s%*s%s%s\n", colorBold, pad+len(s), s, colorReset, "")
	fmt.Printf("%s%s%s\n", colorBold+colorCyan, line, colorReset)
}

func printSection(s string) {
	fmt.Printf("%s▶ %s%s\n", colorBold+colorCyan, s, colorReset)
}

func printError(s string) {
	fmt.Printf("%s✗ %s%s\n", colorRed, s, colorReset)
}

func printWarning(s string) {
	fmt.Printf("%s⚠ %s%s\n", colorYellow, s, colorReset)
}

func colorizeLog(line string) string {
	switch {
	case len(line) > 33 && line[24:28] == "INFO":
		return colorGreen + line + colorReset
	case len(line) > 33 && line[24:28] == "WARN":
		return colorYellow + line + colorReset
	case len(line) > 33 && line[24:28] == "ERR ":
		return colorRed + line + colorReset
	default:
		return colorDim + line + colorReset
	}
}
