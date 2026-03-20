# Shell-out vs Native Runner — 비교

## 한줄 요약

> shell-out은 호출마다 새 프로세스를 fork하므로 **1.4ms/call 고정 비용**이 발생한다.
> native는 Go 라이브러리를 직접 호출하므로 **125ns/call** — **약 11,000배 빠르다.**

---

## 벤치마크 결과 (Apple M4 Pro, 2026-03-09)

```
BenchmarkForkOverhead_True        1,371,128 ns/op   12 KB   80 allocs   # fork 1회
BenchmarkForkOverhead_Echo        1,418,117 ns/op   48 KB  110 allocs   # fork 1회 + 출력
BenchmarkNativeK8s_Get                125.3 ns/op  339  B    4 allocs   # native 1회
BenchmarkNativeHelm_List               72.6 ns/op  274  B    2 allocs   # native 1회
BenchmarkShellOut_Repeated10     17,899,878 ns/op  120 KB  800 allocs   # fork 10회
BenchmarkNative_Repeated10              918 ns/op    3 KB   50 allocs   # native 10회
```

| 항목 | Shell-out | Native | 배율 |
|------|----------:|-------:|-----:|
| 단일 호출 | ~1.4 ms | ~0.13 µs | **11,000×** |
| 10회 반복 | ~18 ms | ~0.92 µs | **19,500×** |
| 메모리/호출 | ~12 KB | ~340 B | **36×** |

---

## 코드 비교

### kubectl get pod

**Before (shell-out)**
```go
// 매 호출마다 kubectl 바이너리를 fork+exec
out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", podName,
    "-n", namespace, "-o", "json")
if err != nil {
    return err
}
// JSON 문자열을 직접 파싱해야 함
var pod corev1.Pod
json.Unmarshal([]byte(out), &pod)
```

**After (native — K8sRunner)**
```go
// client-go가 HTTP 요청을 직접 수행, 구조체로 바로 반환
data, err := t.k8sRunner.Get(ctx, "pods", podName, namespace)
if err != nil {
    return err
}
// data는 이미 JSON bytes — 또는 runner가 직접 typed struct 반환 가능
```

---

### helm install

**Before (shell-out)**
```go
args := []string{"install", release, chart,
    "--values", valueFile, "--namespace", namespace}
_, err := utils.ExecuteCommand(ctx, "helm", args...)
```

**After (native — HelmRunner)**
```go
err := t.helmInstallWithFiles(ctx, release, chart, namespace, valueFiles)
// 내부: t.helmRunner.UpgradeWithFiles() → helm.sh/helm/v3/pkg/action 직접 호출
```

---

## 실질적인 차이점

### 1. 테스트 가능성

```go
// Before: 실제 kubectl 바이너리 필요 → CI에서 통합 테스트만 가능
func TestDeploy(t *testing.T) {
    // kubectl, helm이 설치된 환경에서만 통과
}

// After: mock 주입 → 순수 단위 테스트 가능
func TestDeploy(t *testing.T) {
    stack := &ThanosStack{
        k8sRunner: &mock.K8sRunner{
            OnGet: func(...) ([]byte, error) { return testPayload, nil },
        },
    }
    // 바이너리 없이도 실행됨
}
```

### 2. 로그 스트리밍

```go
// Before: ShellOutK8sRunner.Logs → 에러 반환 (미지원)
// logs, err := t.Logs(ctx, pod, ns, "", false)
// → "streaming is not supported in legacy mode"

// After: NativeK8sRunner.Logs → io.ReadCloser 직접 반환
rc, err := t.k8sRunner.Logs(ctx, pod, namespace, container, false)
defer rc.Close()
io.Copy(os.Stdout, rc)  // 실시간 스트리밍
```

### 3. 에러 구조화

```go
// Before: kubectl stderr 텍스트를 파싱해야 함
if strings.Contains(err.Error(), "NotFound") { ... }
if strings.Contains(err.Error(), "AlreadyExists") { ... }

// After: k8s.io/apimachinery 에러 타입 직접 사용
import k8serrors "k8s.io/apimachinery/pkg/api/errors"
if k8serrors.IsNotFound(err) { ... }
if k8serrors.IsAlreadyExists(err) { ... }
```

### 4. Context 취소

```go
// Before: kubectl 프로세스가 이미 fork된 후에는 취소 불가능
// (os.Process.Kill()이 호출되지만 kubectl은 이미 API 요청 중)

// After: context 취소가 HTTP 요청 레벨에서 즉시 반영됨
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
// 30초 초과 시 k8s API 요청이 즉시 중단됨
```

---

## 폴백 동작

```
runner == nil  → shell-out (kubectl/helm/aws/terraform 바이너리 필요)
runner != nil  → native Go 라이브러리 (바이너리 불필요)
TRHS_LEGACY=1  → 강제로 shell-out (ShellOutRunner 주입)
```

---

## 벤치마크 재실행 방법

```bash
# 전체 벤치마크
GOMODCACHE=/tmp/gomodcache go test -c -o /tmp/runner.test ./pkg/runner/
/tmp/runner.test -test.bench=. -test.benchmem -test.benchtime=3s

# 특정 비교만
/tmp/runner.test -test.bench=BenchmarkShellOut_Repeated10 -test.benchmem
/tmp/runner.test -test.bench=BenchmarkNative_Repeated10 -test.benchmem
```
