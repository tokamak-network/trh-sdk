# PRD — TRH SDK Go 라이브러리 내재화

**버전**: 1.0
**작성일**: 2026-03-09
**프로젝트**: tokamak-network/trh-sdk

---

## 1. 배경 및 문제

TRH SDK는 L2 체인 배포를 자동화하는 Go CLI 도구다. 현재 `pkg/utils/command.go`의 `ExecuteCommand()`가 모든 외부 도구를 subprocess로 실행한다.

### 현재 의존성

| 도구 | 호출 횟수 | 역할 |
|------|----------|------|
| kubectl | **114회** | K8s 리소스 조회/생성/삭제 |
| aws CLI | **78회** | EKS, CloudWatch, EFS, IAM |
| helm | **21회** | 차트 설치/업그레이드/삭제 |
| terraform | **6회** | 인프라 프로비저닝 |
| doctl | **4회** | DO 토큰 검증, kubeconfig |
| **합계** | **223회** | 로컬 바이너리 5개 필요 |

추가로 setup.sh는 설치 전 11개 도구를 설치한다:
Homebrew, Git, Xcode Tools, Terraform, AWS CLI, Helm, kubectl, Node.js, pnpm, Foundry, Docker

### 사용자가 겪는 문제

```
"L2 배포해보자" 마음먹은 시점 → 첫 trh deploy 실행 시점
= 최소 10~30분 + 성공 불확실성
```

community 버전(self-hosted) 구조상 서버사이드 실행은 불가능하므로, 로컬 의존성을 제거하는 유일한 방법은 Go 라이브러리 내재화다.

---

## 2. 목표

### 사용자 목표
- `curl -sSL https://get.trh-sdk.io | bash` 한 줄로 설치 완료
- `trh deploy` 실행 전 추가 도구 설치 불필요 (terraform 제외, 자동 설치)

### 기술 목표
- `ExecuteCommand()` 인터페이스를 유지하면서 내부 구현만 교체 (호출 코드 변경 최소화)
- `ToolRunner` 인터페이스 도입으로 ShellOut ↔ NativeImpl 교체 가능하게

### 비목표
- Docker, Foundry, Node.js/pnpm: devnet 및 contract 배포에 필요 — 이번 범위 외
- Terraform 완전 제거: tfinstall 자동 설치로 처리, 제거 대상 아님

---

## 3. 핵심 설계

### ToolRunner 인터페이스

```go
// pkg/runner/runner.go
type ToolRunner interface {
    K8s() K8sRunner
    Helm() HelmRunner
    AWS() AWSRunner
    DO() DORunner
    TF() TFRunner
}
```

`ShellOutRunner` (현재 동작 유지 / 폴백)와 `NativeRunner` (라이브러리 직접 호출) 두 구현체를 두고, 빌드 플래그 또는 런타임 감지로 전환한다.

### 교체 라이브러리 매핑

| 모듈 | 현재 | 교체 라이브러리 |
|------|------|----------------|
| K8sRunner | `kubectl` | `k8s.io/client-go` |
| HelmRunner | `helm` | `helm.sh/helm/v3/pkg/action` |
| DORunner | `doctl` | `github.com/digitalocean/godo` |
| AWSRunner | `aws` CLI | `github.com/aws/aws-sdk-go-v2` |
| TFRunner | `terraform` | `github.com/hashicorp/terraform-exec` + tfinstall |

---

## 4. 사용자 스토리

| ID | 스토리 | 수용 기준 |
|----|--------|----------|
| US-01 | 오퍼레이터가 kubectl 없이 K8s 리소스를 관리할 수 있다 | `which kubectl` 실패해도 `trh deploy` 성공 |
| US-02 | 오퍼레이터가 helm 없이 차트를 설치할 수 있다 | `which helm` 실패해도 Helm 차트 설치 성공 |
| US-03 | DO 사용자가 doctl 없이 인증/kubeconfig를 구성할 수 있다 | `which doctl` 실패해도 DOKS 배포 성공 |
| US-04 | AWS 사용자가 aws CLI 없이 EKS/CloudWatch를 사용할 수 있다 | `which aws` 실패해도 EKS 배포 성공 |
| US-05 | terraform이 없어도 첫 실행 시 자동 설치된다 | `which terraform` 실패 → 자동 다운로드 후 실행 |

---

## 5. 비기능 요구사항

- **바이너리 크기**: 내재화 후 최대 100MB (현재 ~10MB + 외부 도구 합산 수GB)
- **빌드 시간**: 현재 대비 2배 이내
- **하위 호환**: 기존 ShellOut 동작은 `--legacy` 플래그로 폴백 가능
- **테스트**: 각 Runner 교체 전/후 E2E 테스트 필수

---

## 6. 성공 지표

| 지표 | 현재 | 목표 (Phase 3 완료 후) |
|------|------|----------------------|
| 설치 필요 도구 수 | 11개 | 1개 (trh-sdk 바이너리) |
| 설치 시간 | 10~30분 | 30초 이내 |
| 설치 실패 가능 포인트 | 11개 | 1개 |
| 로컬 의존성 디스크 사용량 | 수GB | ~100MB |
