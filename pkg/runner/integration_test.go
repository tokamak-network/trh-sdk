//go:build integration

package runner

// integration_test.go — NativeK8sRunner 통합 테스트
//
// 실행:
//   KUBECONFIG=/tmp/trh-test.kubeconfig \
//   GOMODCACHE=/tmp/gomodcache \
//   go test -v -tags=integration -timeout=120s ./pkg/runner/
//
// 전제 조건:
//   kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig
//
// 각 테스트는 독립적으로 실행 가능하며, 사용한 리소스는 테스트 종료 시 정리됩니다.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ─── 테스트 헬퍼 ──────────────────────────────────────────────────────────────

func integrationRunner(t *testing.T) *NativeK8sRunner {
	t.Helper()
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// 기본 경로 시도
		kubeconfig = "/tmp/trh-test.kubeconfig"
	}
	if _, err := os.Stat(kubeconfig); err != nil {
		t.Skipf("kubeconfig 없음 (%s): kind 클러스터를 먼저 생성하세요", kubeconfig)
	}
	r, err := newNativeK8sRunner(kubeconfig)
	if err != nil {
		t.Fatalf("NativeK8sRunner 생성 실패: %v", err)
	}
	return r
}

// testNamespace는 테스트용 네임스페이스를 생성하고 t.Cleanup으로 삭제를 등록합니다.
func testNamespace(t *testing.T, r *NativeK8sRunner, name string) string {
	t.Helper()
	ctx := context.Background()
	if err := r.EnsureNamespace(ctx, name); err != nil {
		t.Fatalf("EnsureNamespace(%q) 실패: %v", name, err)
	}
	t.Cleanup(func() {
		// 네임스페이스 삭제 (내부 리소스 포함)
		_ = r.client.CoreV1().Namespaces().Delete(
			context.Background(), name, metav1.DeleteOptions{},
		)
	})
	return name
}

// ─── 1. Namespace 테스트 ──────────────────────────────────────────────────────

func TestIntegration_EnsureNamespace(t *testing.T) {
	r := integrationRunner(t)
	ctx := context.Background()
	ns := "trh-test-ns-" + randomSuffix()

	// 생성
	if err := r.EnsureNamespace(ctx, ns); err != nil {
		t.Fatalf("EnsureNamespace 실패: %v", err)
	}
	t.Cleanup(func() {
		_ = r.client.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	})

	// 존재 확인
	exists, err := r.NamespaceExists(ctx, ns)
	if err != nil {
		t.Fatalf("NamespaceExists 실패: %v", err)
	}
	if !exists {
		t.Fatal("EnsureNamespace 후 NamespaceExists가 false 반환")
	}

	// 멱등성: 두 번 호출해도 에러 없음
	if err := r.EnsureNamespace(ctx, ns); err != nil {
		t.Fatalf("두 번째 EnsureNamespace 실패 (멱등성 위반): %v", err)
	}
	t.Logf("✅ 네임스페이스 생성 및 멱등성 확인: %s", ns)
}

// ─── 2. Apply + Get + List 테스트 ────────────────────────────────────────────

const configMapManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: trh-test-cm
  namespace: %s
data:
  key: "runner-integration-test"
  timestamp: "%s"
`

func TestIntegration_Apply_Get_List(t *testing.T) {
	r := integrationRunner(t)
	ctx := context.Background()
	ns := testNamespace(t, r, "trh-integ-"+randomSuffix())

	manifest := []byte(sprintf(configMapManifest, ns, time.Now().Format(time.RFC3339)))

	// Apply
	if err := r.Apply(ctx, manifest); err != nil {
		t.Fatalf("Apply 실패: %v", err)
	}
	t.Logf("✅ Apply 성공")

	// Get
	data, err := r.Get(ctx, "configmaps", "trh-test-cm", ns)
	if err != nil {
		t.Fatalf("Get 실패: %v", err)
	}
	var cm map[string]interface{}
	if err := json.Unmarshal(data, &cm); err != nil {
		t.Fatalf("Get 결과 JSON 파싱 실패: %v", err)
	}
	cmData := cm["data"].(map[string]interface{})
	if cmData["key"] != "runner-integration-test" {
		t.Fatalf("ConfigMap data.key 불일치: %v", cmData["key"])
	}
	t.Logf("✅ Get 성공: data.key=%q", cmData["key"])

	// List
	listData, err := r.List(ctx, "configmaps", ns, "")
	if err != nil {
		t.Fatalf("List 실패: %v", err)
	}
	var list map[string]interface{}
	if err := json.Unmarshal(listData, &list); err != nil {
		t.Fatalf("List 결과 JSON 파싱 실패: %v", err)
	}
	items := list["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("List 결과가 비어있음")
	}
	t.Logf("✅ List 성공: %d개 configmap", len(items))

	// Apply 멱등성 (서버사이드 apply → 재적용해도 에러 없음)
	if err := r.Apply(ctx, manifest); err != nil {
		t.Fatalf("두 번째 Apply 실패 (멱등성 위반): %v", err)
	}
	t.Logf("✅ Apply 멱등성 확인")
}

// ─── 3. Pod Apply + Wait + Logs 테스트 (핵심) ────────────────────────────────

const logPodManifest = `
apiVersion: v1
kind: Pod
metadata:
  name: trh-log-test
  namespace: %s
spec:
  restartPolicy: Never
  containers:
  - name: logger
    image: busybox:1.36
    command: ["sh", "-c", "for i in 1 2 3 4 5; do echo \"line-$i op-node block=$i\"; sleep 0.2; done"]
`

func TestIntegration_PodLogs_Streaming(t *testing.T) {
	r := integrationRunner(t)
	ctx := context.Background()
	ns := testNamespace(t, r, "trh-logs-"+randomSuffix())

	// Pod 배포
	manifest := []byte(sprintf(logPodManifest, ns))
	if err := r.Apply(ctx, manifest); err != nil {
		t.Fatalf("Pod Apply 실패: %v", err)
	}
	t.Logf("✅ Pod 배포됨, 완료 대기 중...")

	// Pod가 Succeeded 상태가 될 때까지 대기 (최대 60초)
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := waitForPodSucceeded(t, r, waitCtx, "trh-log-test", ns); err != nil {
		t.Fatalf("Pod 완료 대기 실패: %v", err)
	}
	t.Logf("✅ Pod 완료")

	// 로그 스트리밍 — 핵심 테스트
	logCtx, logCancel := context.WithTimeout(ctx, 10*time.Second)
	defer logCancel()

	rc, err := r.Logs(logCtx, "trh-log-test", ns, "logger", false)
	if err != nil {
		t.Fatalf("Logs() 실패: %v", err)
	}
	defer rc.Close() //nolint:errcheck

	var lines []string
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		t.Logf("  📋 %s", line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("로그 스캔 중 에러: %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("로그 라인이 하나도 없음")
	}
	if !strings.Contains(lines[0], "line-1") {
		t.Fatalf("첫 번째 로그 라인이 예상과 다름: %q", lines[0])
	}
	t.Logf("✅ 로그 스트리밍 성공: %d줄 수신", len(lines))
}

// ─── 4. Delete 테스트 ─────────────────────────────────────────────────────────

func TestIntegration_Delete(t *testing.T) {
	r := integrationRunner(t)
	ctx := context.Background()
	ns := testNamespace(t, r, "trh-del-"+randomSuffix())

	// ConfigMap 생성
	manifest := []byte(sprintf(configMapManifest, ns, "delete-test"))
	if err := r.Apply(ctx, manifest); err != nil {
		t.Fatalf("Apply 실패: %v", err)
	}

	// Delete
	if err := r.Delete(ctx, "configmaps", "trh-test-cm", ns, false); err != nil {
		t.Fatalf("Delete 실패: %v", err)
	}
	t.Logf("✅ Delete 성공")

	// 삭제 후 Get 하면 에러여야 함
	_, err := r.Get(ctx, "configmaps", "trh-test-cm", ns)
	if err == nil {
		t.Fatal("Delete 후 Get이 성공함 — 삭제 안 된 것으로 보임")
	}
	t.Logf("✅ 삭제 확인: Get → 에러 반환")

	// ignoreNotFound=true 로 중복 삭제해도 에러 없음
	if err := r.Delete(ctx, "configmaps", "trh-test-cm", ns, true); err != nil {
		t.Fatalf("ignoreNotFound=true 중복 삭제 실패: %v", err)
	}
	t.Logf("✅ ignoreNotFound 멱등성 확인")
}

// ─── 5. Patch 테스트 ──────────────────────────────────────────────────────────

func TestIntegration_Patch(t *testing.T) {
	r := integrationRunner(t)
	ctx := context.Background()
	ns := testNamespace(t, r, "trh-patch-"+randomSuffix())

	// ConfigMap 생성
	manifest := []byte(sprintf(configMapManifest, ns, "before-patch"))
	if err := r.Apply(ctx, manifest); err != nil {
		t.Fatalf("Apply 실패: %v", err)
	}

	// Patch
	patch := []byte(`{"data":{"key":"patched-value"}}`)
	if err := r.Patch(ctx, "configmaps", "trh-test-cm", ns, patch); err != nil {
		t.Fatalf("Patch 실패: %v", err)
	}

	// 패치 결과 확인
	data, err := r.Get(ctx, "configmaps", "trh-test-cm", ns)
	if err != nil {
		t.Fatalf("Patch 후 Get 실패: %v", err)
	}
	var cm map[string]interface{}
	json.Unmarshal(data, &cm) //nolint:errcheck
	cmData := cm["data"].(map[string]interface{})
	if cmData["key"] != "patched-value" {
		t.Fatalf("Patch 반영 안 됨: data.key=%q", cmData["key"])
	}
	t.Logf("✅ Patch 성공: data.key=%q", cmData["key"])
}

// ─── 헬퍼 ────────────────────────────────────────────────────────────────────

// waitForPodSucceeded는 Pod가 Succeeded 상태가 될 때까지 폴링합니다.
func waitForPodSucceeded(t *testing.T, r *NativeK8sRunner, ctx context.Context, podName, namespace string) error {
	t.Helper()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		pod, err := r.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return nil
		case corev1.PodFailed:
			return nil // 실패해도 로그는 남아있음
		}
		t.Logf("  Pod 상태: %s", pod.Status.Phase)
	}
}

func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func randomSuffix() string {
	return time.Now().Format("150405") // HHMMSS
}
