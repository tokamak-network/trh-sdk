# TODO — 알려진 약점 및 개선 항목

## 중간 우선순위

### [TODO-1] hc-install 런타임 다운로드 제거
- **파일**: `pkg/runner/native_tf.go`
- **문제**: PATH에 terraform 1.9.8이 없으면 HashiCorp CDN에서 런타임 다운로드.
  에어갭(air-gap) 환경 배포 실패, 네트워크 의존성 발생.
- **해결 방향**:
  1. 배포 전 terraform pre-install을 요구하도록 강제 (`hc-install` 폴백 제거)
  2. 또는 에어갭용 `TERRAFORM_BIN_PATH` 환경변수 지원 추가
- **영향 범위**: `native_tf.go`, CI/CD 배포 가이드

### [TODO-2] Helm context 취소 불완전
- **파일**: `pkg/runner/native_helm.go`
- **문제**: goroutine+select 패턴으로 context 취소 시 Go 함수는 반환되지만
  내부 Helm action은 계속 실행됨. 타임아웃 발생 시 클러스터에
  부분 배포(partial deploy) 상태가 남을 수 있음.
- **해결 방향**:
  - `HelmRunner` 인터페이스에 `Stop(ctx context.Context) error` 메서드 추가
  - 또는 helm action 실행 전 `action.Configuration`에 ctx 취소 훅 주입
- **영향 범위**: `helm.go` (인터페이스), `native_helm.go`, `mock_helm.go`, 관련 테스트

### [TODO-3] extraArgs 미지원 (HelmRunner)
- **파일**: `pkg/stacks/thanos/thanos_stack.go` — `helmUpgradeInstallWithFiles()`
- **문제**: `helmRunner != nil`일 때 extraArgs를 전달하면 런타임 에러 반환.
  향후 호출자가 extraArgs를 추가하면 네이티브 모드에서만 조용히 실패할 위험.
- **해결 방향**:
  - `HelmRunner.UpgradeWithFiles()` 시그니처에 `extraArgs ...string` 추가
  - 또는 `HelmUpgradeOptions` struct로 옵션 패턴 전환
- **영향 범위**: `helm.go`, `native_helm.go`, `shellout_helm.go`, `mock_helm.go`

## 낮은 우선순위

### [TODO-4] 의존성 CVE 주기적 스캔
- **문제**: helm/v3, aws-sdk-go-v2, terraform-exec, godo, client-go 포함으로
  의존성 CVE 표면이 증가함. 현재 자동화 스캔 없음.
- **해결 방향**:
  ```bash
  # 로컬 검사
  govulncheck ./...
  ```
  CI에 `govulncheck` 단계 추가 (`.github/workflows/ci.yaml`의 lint job 옆)
- **영향 범위**: `.github/workflows/ci.yaml`

### [TODO-5] kubectl rollout 명령어 K8sRunner 미지원
- **파일**: `pkg/stacks/thanos/backup/` 내 일부 호출
- **문제**: `kubectl rollout restart`, `kubectl rollout status`가
  K8sRunner 인터페이스에 없어 의도적 shell-out으로 남아있음.
- **해결 방향**:
  - `K8sRunner`에 `Rollout(ctx, action, resource, namespace string) error` 추가
- **영향 범위**: `k8s.go`, `native_k8s.go`, `shellout.go`, `mock_k8s.go`
