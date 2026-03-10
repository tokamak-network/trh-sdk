# 발견 사항 & 공유 자료

## 핵심 아키텍처 (이번 팀 모두 숙지 필수)

### ThanosStack runner 필드 (thanos_stack.go)
```go
type ThanosStack struct {
    // ...
    helmRunner runner.HelmRunner  // nil이면 shellout 폴백
    k8sRunner  runner.K8sRunner   // nil이면 shellout 폴백
    tfRunner   runner.TFRunner    // nil이면 shellout 폴백
    awsRunner  runner.AWSRunner   // nil이면 shellout 폴백
}
```

### 이미 존재하는 헬퍼 메서드 (thanos_stack.go에서 재사용)
- `t.helmList(ctx, ns)` — helm list
- `t.helmUninstall(ctx, release, ns)` — helm uninstall
- `t.helmInstallWithFiles(ctx, release, chart, ns, files)` — helm install
- `t.helmUpgradeWithFiles(ctx, release, chart, ns, files)` — helm upgrade
- `t.helmUpgradeInstallWithFiles(ctx, release, chart, ns, files, extraArgs...)` — helm upgrade --install
- `t.helmDependencyUpdate(ctx, chartPath)` — helm dependency update
- `t.helmRepoAdd(ctx, name, url)` — helm repo add
- `t.helmSearch(ctx, keyword)` — helm search repo
- `t.helmFilterReleases(ctx, ns, name)` — helm list + filter
- `t.tfInit(ctx, dir, env, backends)` — tf init
- `t.tfApply(ctx, dir, env)` — tf apply
- `t.tfDestroy(ctx, dir, env)` — tf destroy
- `t.PodLogs(ctx, pod, ns, container, since)` — kubectl logs

### runner 인터페이스 참조
- `pkg/runner/aws.go` — AWSRunner 인터페이스 (메서드 목록 확인)
- `pkg/runner/k8s.go` — K8sRunner 인터페이스
- `pkg/runner/helm.go` — HelmRunner 인터페이스
- `pkg/runner/mock/mock_aws.go` — AWSRunner mock (테스트용)
- `pkg/runner/mock/mock_k8s.go` — K8sRunner mock (테스트용)

### 이중 경로 패턴 (dual-path pattern)
```go
// 올바른 패턴
func (t *ThanosStack) someMethod(ctx context.Context) error {
    if t.awsRunner != nil {
        return t.awsRunner.SomeMethod(ctx, args...)
    }
    // shellout fallback
    _, err := utils.ExecuteCommand(ctx, "aws", "subcommand", args...)
    return err
}
```

### 빌드 명령
```bash
GOMODCACHE=/tmp/gomodcache go build ./...
GOMODCACHE=/tmp/gomodcache go test -race ./...
```

## 남은 마이그레이션 대상

### developer-1 대상 (thanos helm 7개)
- pkg/stacks/thanos/block_explorer.go: install, upgrade, uninstall
- pkg/stacks/thanos/uptime_service.go: install/upgrade, uninstall
- pkg/stacks/thanos/deploy_chain.go: repo add, repo search

### developer-2 대상 (backup/*.go)
파일 목록:
- pkg/stacks/thanos/backup/initialize.go
- pkg/stacks/thanos/backup/configure.go
- pkg/stacks/thanos/backup/list.go
- pkg/stacks/thanos/backup/cleanup.go
- pkg/stacks/thanos/backup/attach.go
- pkg/stacks/thanos/backup/snapshot.go
- pkg/stacks/thanos/backup/status.go
- pkg/stacks/thanos/backup/restore.go
- pkg/stacks/thanos/backup/k8s_helpers.go

주의: backup 패키지가 ThanosStack을 직접 받는지 또는 별도 struct를 갖는지 확인 필요.
만약 별도 struct라면 awsRunner/k8sRunner 필드를 그 struct에 추가.

## backup 패키지 구조 (T4 완료 후 발견 사항)

### BackupClient struct (client.go)
```go
type BackupClient struct {
    k8sRunner runner.K8sRunner // optional; nil → shellout
}
// SetDefaultK8sRunner(kr runner.K8sRunner) 로 패키지 레벨 클라이언트 초기화
```

### AWS 처리 방식 - 파라미터 전달 방식
```go
// ThanosStack 필드 방식이 아닌, 함수 파라미터 방식을 사용
func SomeBackupFunc(ctx context.Context, ar runner.AWSRunner, ...) error {
    if ar != nil {
        return ar.SomeMethod(ctx, ...)
    }
    // shellout fallback
    _, err := utils.ExecuteCommand(ctx, "aws", ...)
    return err
}
```

### kubectl 처리 방식 - BackupClient 메서드 방식
```go
// k8s_helpers.go의 모든 메서드가 dual-path 구현됨
func (b *BackupClient) k8sDeletePod(ctx context.Context, name, namespace string) error {
    if b.k8sRunner != nil {
        return b.k8sRunner.Delete(ctx, "pod", name, namespace, true)
    }
    _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", ...)
    return err
}
```

### 의도적 shellout (runner 동등 없음)
- `kubectl rollout restart` — K8sRunner에 없음
- `kubectl rollout status` — K8sRunner에 없음
- `kubectl version --client`, `kubectl cluster-info` — ValidateAttachPrerequisites에서만 사용

### 특이 사항
- `MonitorEFSRestoreJob`은 AWS SDK (aws-sdk-go-v2)를 직접 사용 (runner 경유 아님)
- `configure.go`의 terraform/direnv 호출은 마이그레이션 대상 아님 (helm/aws/kubectl 아님)

## DEAD_ENDS (시도했으나 실패한 접근)
(초기 상태 — 없음)
