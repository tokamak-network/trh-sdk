# 데이터 모델 — TRH SDK Go 라이브러리 내재화

**버전**: 1.0
**작성일**: 2026-03-09

---

## 1. 핵심 인터페이스

### ToolRunner (최상위 인터페이스)

```
ToolRunner
├── K8sRunner    → kubectl 114회 호출 대체
├── HelmRunner   → helm 21회 호출 대체
├── DORunner     → doctl 4회 호출 대체
├── AWSRunner    → aws CLI 78회 호출 대체
└── TFRunner     → terraform 6회 호출 대체
```

### 인터페이스 정의

```go
// pkg/runner/runner.go

type ToolRunner interface {
    K8s() K8sRunner
    Helm() HelmRunner
    AWS() AWSRunner
    DO() DORunner
    TF() TFRunner
}

// ShellOutRunner — 현재 동작 유지 (폴백)
type ShellOutRunner struct{}

// NativeRunner — Go 라이브러리 직접 호출 (내재화 후)
type NativeRunner struct {
    kubeConfig *rest.Config
    helmEnv    *cli.EnvSettings
    awsCfg     aws.Config
    doClient   *godo.Client
}
```

---

## 2. 서브 인터페이스 상세

### K8sRunner

```go
// pkg/runner/k8s.go
type K8sRunner interface {
    Apply(ctx context.Context, manifest []byte) error
    Delete(ctx context.Context, resource, name, namespace string) error
    Get(ctx context.Context, resource, name, namespace string) ([]byte, error)
    List(ctx context.Context, resource, namespace string) ([]byte, error)
    Wait(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error
    Exec(ctx context.Context, pod, namespace string, cmd []string) (string, error)
    Logs(ctx context.Context, pod, namespace string) (io.ReadCloser, error)
}

// NativeK8sRunner 구현
type NativeK8sRunner struct {
    client    kubernetes.Interface
    dynamic   dynamic.Interface
    mapper    meta.RESTMapper
}
```

**의존 라이브러리**: `k8s.io/client-go v0.29+`

### HelmRunner

```go
// pkg/runner/helm.go
type HelmRunner interface {
    Install(ctx context.Context, release, chart string, vals map[string]interface{}) error
    Upgrade(ctx context.Context, release, chart string, vals map[string]interface{}) error
    Uninstall(ctx context.Context, release, namespace string) error
    Status(ctx context.Context, release, namespace string) (*release.Release, error)
    List(ctx context.Context, namespace string) ([]*release.Release, error)
}

type NativeHelmRunner struct {
    settings *cli.EnvSettings
    cfg      *action.Configuration
}
```

**의존 라이브러리**: `helm.sh/helm/v3 v3.14+`

### DORunner

```go
// pkg/runner/do.go
type DORunner interface {
    ValidateToken(ctx context.Context, token string) error
    GetKubeconfig(ctx context.Context, clusterID string) ([]byte, error)
    ListClusters(ctx context.Context) ([]godo.KubernetesCluster, error)
}

type NativeDORunner struct {
    client *godo.Client
}
```

**의존 라이브러리**: `github.com/digitalocean/godo v1.109+`

### AWSRunner

```go
// pkg/runner/aws.go
type AWSRunner interface {
    // EKS
    GetEKSKubeconfig(ctx context.Context, clusterName, region string) ([]byte, error)
    ListEKSClusters(ctx context.Context) ([]string, error)
    // CloudWatch
    PutMetricData(ctx context.Context, namespace string, data []types.MetricDatum) error
    // EFS
    CreateEFS(ctx context.Context, name, region string) (string, error)
    DescribeEFS(ctx context.Context, fileSystemID string) (*efs.FileSystem, error)
    // IAM
    GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error)
}

type NativeAWSRunner struct {
    cfg        aws.Config
    eksClient  *eks.Client
    cwClient   *cloudwatch.Client
    efsClient  *efs.Client
    stsClient  *sts.Client
}
```

**의존 라이브러리**: `github.com/aws/aws-sdk-go-v2 v1.26+`

### TFRunner

```go
// pkg/runner/tf.go
type TFRunner interface {
    Init(ctx context.Context, workdir string) error
    Apply(ctx context.Context, workdir string, vars map[string]string) error
    Destroy(ctx context.Context, workdir string, vars map[string]string) error
    Output(ctx context.Context, workdir string) (map[string]tfexec.OutputMeta, error)
}

type NativeTFRunner struct {
    tf         *tfexec.Terraform
    execPath   string  // tfinstall로 자동 설치된 경로
}
```

**의존 라이브러리**: `github.com/hashicorp/terraform-exec v0.21+` + `tfinstall`

---

## 3. 팩토리 패턴

```go
// pkg/runner/factory.go

type RunnerConfig struct {
    UseNative bool   // false → ShellOutRunner (폴백)
    KubeconfigPath string
    AWSRegion string
    DOToken string
}

func NewToolRunner(cfg RunnerConfig) (ToolRunner, error) {
    if !cfg.UseNative {
        return &ShellOutRunner{}, nil
    }
    return newNativeRunner(cfg)
}
```

---

## 4. 마이그레이션 전략

### ExecuteCommand() 호환 레이어

기존 호출 코드 변경 없이 내부만 교체:

```go
// 현재 (변경 없음)
ExecuteCommand("kubectl", "apply", "-f", manifestPath)

// 내부 라우팅 (신규)
func ExecuteCommand(name string, args ...string) error {
    switch name {
    case "kubectl":
        return globalRunner.K8s().dispatchShellArgs(args...)
    case "helm":
        return globalRunner.Helm().dispatchShellArgs(args...)
    // ...
    default:
        return shellOut(name, args...)
    }
}
```

### 전환 플래그

```bash
# 레거시 모드 (Shell-out 유지)
trh-sdk deploy --legacy

# 기본 모드 (Native, Phase 1 완료 후 기본값)
trh-sdk deploy
```

---

## 5. 라이브러리 버전 매트릭스

| Runner | 라이브러리 | 최소 버전 | go.mod 모듈 경로 |
|--------|-----------|----------|-----------------|
| K8sRunner | client-go | v0.29.0 | `k8s.io/client-go` |
| HelmRunner | helm/v3 | v3.14.0 | `helm.sh/helm/v3` |
| DORunner | godo | v1.109.0 | `github.com/digitalocean/godo` |
| AWSRunner | aws-sdk-go-v2 | v1.26.0 | `github.com/aws/aws-sdk-go-v2` |
| TFRunner | terraform-exec | v0.21.0 | `github.com/hashicorp/terraform-exec` |
| TFRunner | tfinstall | v0.21.0 | `github.com/hashicorp/hc-install` |
