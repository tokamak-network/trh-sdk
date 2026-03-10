# 프로젝트 스펙 — TRH SDK Go 라이브러리 내재화

**버전**: 1.0
**작성일**: 2026-03-09

---

## 1. AI 행동 규칙

이 문서는 AI 코딩 에이전트가 이 프로젝트를 작업할 때 따라야 할 규칙이다.

### 절대 규칙

1. **`ExecuteCommand()` 시그니처 변경 금지**
   - 기존 호출 코드(`commands/`, `pkg/stacks/`)는 수정하지 않는다
   - 내부 구현만 교체한다

2. **ShellOutRunner는 항상 폴백으로 유지**
   - `--legacy` 플래그 또는 `TRHS_LEGACY=1` 환경변수로 활성화
   - NativeRunner 실패 시 자동 폴백하지 않는다 (명시적 폴백만 허용)

3. **각 Runner는 독립적으로 테스트 가능해야 한다**
   - mock 인터페이스를 반드시 제공한다
   - E2E 테스트는 실제 인프라 없이도 unit test 레벨에서 검증 가능해야 한다

4. **바이너리 크기 100MB 초과 빌드 금지**
   - 새 의존성 추가 전 `go build -v ./...` 후 크기 확인
   - 100MB 초과 시 Go build tags로 선택적 빌드 검토

### 코딩 컨벤션

```go
// ✅ 올바른 패턴 — 에러 항상 처리
func (r *NativeK8sRunner) Apply(ctx context.Context, manifest []byte) error {
    _, err := r.client.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
    if err != nil {
        return fmt.Errorf("k8s apply failed: %w", err)
    }
    return nil
}

// ❌ 금지 패턴 — 에러 무시
func (r *NativeK8sRunner) Apply(ctx context.Context, manifest []byte) error {
    r.client.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
    return nil
}
```

```go
// ✅ 올바른 패턴 — context 전파
func (r *NativeHelmRunner) Install(ctx context.Context, release, chart string, vals map[string]interface{}) error {
    install := action.NewInstall(r.cfg)
    install.ReleaseName = release
    // ctx를 통한 취소 지원
    return r.runWithContext(ctx, func() error {
        _, err := install.Run(chartLoaded, vals)
        return err
    })
}

// ❌ 금지 패턴 — context 무시
func (r *NativeHelmRunner) Install(release, chart string, vals map[string]interface{}) error {
    // context 없음 → 취소 불가
}
```

---

## 2. 디렉토리 구조

```
trh-sdk/
├── pkg/
│   ├── runner/             # 신규 (이번 PRD 범위)
│   │   ├── runner.go       # ToolRunner 인터페이스 + 팩토리
│   │   ├── k8s.go          # K8sRunner (Phase 1)
│   │   ├── helm.go         # HelmRunner (Phase 2)
│   │   ├── do.go           # DORunner (Phase 2)
│   │   ├── aws.go          # AWSRunner (Phase 3)
│   │   ├── tf.go           # TFRunner (Phase 3)
│   │   ├── shellout.go     # ShellOutRunner (폴백)
│   │   └── mock/           # 테스트용 mock
│   │       ├── mock_k8s.go
│   │       ├── mock_helm.go
│   │       └── ...
│   └── utils/
│       └── command.go      # ExecuteCommand() — 내부만 수정
├── commands/               # 변경 없음
├── pkg/stacks/             # 변경 없음
└── PRD/                    # 이 문서들
```

---

## 3. 테스트 요구사항

### Unit 테스트

각 Runner의 mock 구현:

```go
// pkg/runner/mock/mock_k8s.go
type MockK8sRunner struct {
    mock.Mock
}

func (m *MockK8sRunner) Apply(ctx context.Context, manifest []byte) error {
    args := m.Called(ctx, manifest)
    return args.Error(0)
}
```

커버리지 목표: **각 Runner 80% 이상**

### 통합 테스트

```bash
# 환경 변수로 실제/mock 전환
TRHS_USE_NATIVE=true go test ./pkg/runner/... -run TestK8sRunner
```

### E2E 테스트 (CI 환경)

```bash
# Phase 1 완료 검증
# kind 클러스터 + kubectl 없는 환경
kind create cluster
unalias kubectl 2>/dev/null; PATH_WITHOUT_KUBECTL=...
go test ./e2e/... -run TestDeployWithoutKubectl
```

---

## 4. 의존성 관리

### go.mod 추가 순서

Phase 1:
```
k8s.io/client-go v0.29.0
k8s.io/api v0.29.0
k8s.io/apimachinery v0.29.0
```

Phase 2:
```
helm.sh/helm/v3 v3.14.0
github.com/digitalocean/godo v1.109.0
```

Phase 3:
```
github.com/aws/aws-sdk-go-v2 v1.26.0
github.com/aws/aws-sdk-go-v2/service/eks
github.com/aws/aws-sdk-go-v2/service/cloudwatch
github.com/aws/aws-sdk-go-v2/service/efs
github.com/aws/aws-sdk-go-v2/service/sts
github.com/hashicorp/terraform-exec v0.21.0
github.com/hashicorp/hc-install v0.6.0
```

### 바이너리 크기 모니터링

```bash
# 빌드 후 크기 확인 스크립트
go build -o trh-sdk-binary ./main.go
ls -lh trh-sdk-binary
# 100MB 초과 시 CI 실패
```

---

## 5. 릴리스 기준

각 Phase 완료 후 다음을 검증한다:

| 항목 | 검증 방법 | 목표값 |
|------|---------|-------|
| 바이너리 크기 | `ls -lh` | Phase 1: ≤50MB, Phase 2: ≤70MB, Phase 3: ≤100MB |
| 빌드 시간 | CI 측정 | 현재 대비 2배 이내 |
| 단위 테스트 | `go test ./...` | 100% 통과 |
| E2E 테스트 | 도구 없는 환경 | 배포 성공 |
| 폴백 테스트 | `--legacy` 플래그 | 동일 동작 |

---

## 6. 금지 사항

- `ExecuteCommand()` 호출 사이트(`commands/`, `pkg/stacks/`) 직접 수정 — runner 내부에서 처리
- 외부 상태(전역 변수, 싱글턴) 무분별 사용 — `ToolRunner` 인스턴스에 상태 캡슐화
- `os.Exit()` 직접 호출 — 에러 반환으로 처리
- 자격증명 하드코딩 또는 로그 출력 — 환경변수/설정파일 경유
- terraform 바이너리 수동 설치 요구 — tfinstall로 자동화
