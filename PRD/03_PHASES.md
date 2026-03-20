# Phase 분리 계획 — TRH SDK Go 라이브러리 내재화

**버전**: 1.0
**작성일**: 2026-03-09

---

## 전체 로드맵

```
Phase 1 (3주)     Phase 2 (2주)      Phase 3 (3주)
kubectl           helm + doctl       aws + terraform
114 call sites    25 call sites      84 call sites
────────────      ────────────────   ───────────────────
k8s.io/client-go  helm.sh/helm/v3    aws-sdk-go-v2
                  godo               tfinstall
```

**총 기간**: 8주 (2개월)
**총 제거 대상**: 223회 shell-out 호출

---

## Phase 1 — kubectl 내재화 (3주)

**목표**: `which kubectl` 실패해도 `trh deploy` 성공

### 범위

- 제거 대상: kubectl subprocess 호출 **114회**
- 도입 라이브러리: `k8s.io/client-go v0.29+`
- 기반 인프라 구축 (다른 Phase가 재사용)

### 주차별 계획

#### Week 1 — 인프라 구축
- [ ] `pkg/runner/` 패키지 생성
- [ ] `ToolRunner` 인터페이스 정의 (`runner.go`)
- [ ] `ShellOutRunner` 구현 (기존 동작 래핑)
- [ ] `NativeK8sRunner` 골격 생성 (`k8s.go`)
- [ ] `RunnerFactory` + `--legacy` 플래그 연결
- [ ] kubeconfig 로딩 유틸리티 구현
- [ ] Unit test 골격 (mock K8sRunner)

#### Week 2 — 핵심 CRUD 구현
- [ ] `Apply()` — `kubectl apply -f` 대체
- [ ] `Delete()` — `kubectl delete` 대체
- [ ] `Get()` / `List()` — `kubectl get` 대체
- [ ] `Wait()` — `kubectl wait --for=condition` 대체
- [ ] 114개 호출 사이트 매핑 문서 작성
- [ ] 상위 50개 호출 사이트 마이그레이션

#### Week 3 — 완성 및 검증
- [ ] 나머지 64개 호출 사이트 마이그레이션
- [ ] `Exec()` / `Logs()` 구현
- [ ] E2E 테스트: kubectl 없는 환경에서 배포 성공 확인
- [ ] `--legacy` 폴백 검증
- [ ] 바이너리 크기 측정 (목표: 50MB 이하)

### 완료 기준 (US-01)

```bash
# kubectl 제거
sudo rm $(which kubectl)

# 배포 성공 확인
trh-sdk deploy --network testnet --stack thanos
# → 성공
```

---

## Phase 2 — helm + doctl 내재화 (2주)

**목표**: helm/doctl 없이 차트 설치 및 DO 인증 가능

### 범위

- helm: **21회** shell-out → `helm.sh/helm/v3/pkg/action`
- doctl: **4회** shell-out → `github.com/digitalocean/godo`
- Phase 1 인프라(ToolRunner, RunnerFactory) 재사용

### 주차별 계획

#### Week 4 — helm 내재화
- [ ] `NativeHelmRunner` 구현 (`helm.go`)
- [ ] `Install()` / `Upgrade()` / `Uninstall()` 구현
- [ ] `Status()` / `List()` 구현
- [ ] 21개 호출 사이트 마이그레이션
- [ ] Helm 저장소 관리 (`repo add`, `repo update`) 구현
- [ ] Unit test + E2E: helm 없는 환경에서 차트 설치 성공

#### Week 5 — doctl 내재화
- [ ] `NativeDORunner` 구현 (`do.go`)
- [ ] `ValidateToken()` 구현 — doctl auth validate 대체
- [ ] `GetKubeconfig()` 구현 — doctl k8s cluster kubeconfig save 대체
- [ ] `ListClusters()` 구현
- [ ] 4개 호출 사이트 마이그레이션
- [ ] E2E: doctl 없는 환경에서 DOKS 배포 성공 (US-03)
- [ ] 바이너리 크기 측정 (목표: 70MB 이하)

### 완료 기준 (US-02, US-03)

```bash
# helm + doctl 제거
sudo rm $(which helm) $(which doctl)

# AWS 배포 (helm 없이)
trh-sdk deploy --network testnet --stack thanos
# → 성공

# DO 배포 (doctl 없이)
trh-sdk deploy --network testnet --stack thanos --provider digitalocean
# → 성공
```

---

## Phase 3 — aws CLI + terraform 내재화 (3주)

**목표**: aws CLI 없이 EKS/CloudWatch 사용, terraform 자동 설치

### 범위

- aws CLI: **78회** shell-out → `github.com/aws/aws-sdk-go-v2`
- terraform: **6회** shell-out → `tfinstall` 자동 다운로드 + `terraform-exec`
- 최종 의존성: trh-sdk 바이너리 1개 (+ terraform 자동 설치)

### 주차별 계획

#### Week 6 — AWS SDK 기반 구축
- [ ] `NativeAWSRunner` 구현 (`aws.go`)
- [ ] AWS 자격증명 설정 (`aws.Config` 로드)
- [ ] EKS: `GetEKSKubeconfig()` / `ListEKSClusters()` 구현
- [ ] STS: `GetCallerIdentity()` 구현 (aws sts get-caller-identity 대체)
- [ ] 상위 40개 aws CLI 호출 사이트 마이그레이션

#### Week 7 — AWS 완성 + terraform 자동화
- [ ] CloudWatch: `PutMetricData()` 구현
- [ ] EFS: `CreateEFS()` / `DescribeEFS()` 구현
- [ ] IAM 관련 호출 마이그레이션
- [ ] 나머지 38개 aws CLI 호출 사이트 마이그레이션
- [ ] `NativeTFRunner` 구현 (`tf.go`)
- [ ] `tfinstall` 통합 — 첫 실행 시 terraform 바이너리 자동 다운로드
- [ ] 6개 terraform 호출 사이트 마이그레이션

#### Week 8 — 통합 검증 및 릴리스
- [ ] 전체 E2E 테스트: 5개 도구 모두 없는 클린 환경에서 배포 성공
- [ ] `--legacy` 폴백 모드 최종 검증
- [ ] 바이너리 크기 측정 (목표: 100MB 이하)
- [ ] 설치 시간 측정 (목표: 30초 이내)
- [ ] setup.sh 업데이트 (불필요한 도구 설치 제거)
- [ ] 릴리스 노트 작성

### 완료 기준 (US-04, US-05)

```bash
# 클린 환경 (도구 없음)
which kubectl  # 실패
which helm     # 실패
which doctl    # 실패
which aws      # 실패
which terraform # 실패

# 단일 바이너리 설치
curl -sSL https://get.trh-sdk.io | bash

# 배포 실행 (30초 이내 완료)
trh-sdk deploy --network testnet --stack thanos
# → terraform 자동 다운로드 후 배포 성공
```

---

## 마일스톤 요약

| 마일스톤 | 완료 시점 | 제거된 도구 | 남은 의존성 |
|---------|----------|------------|-----------|
| Phase 1 완료 | 3주차 | kubectl | helm, doctl, aws, terraform |
| Phase 2 완료 | 5주차 | kubectl, helm, doctl | aws, terraform |
| Phase 3 완료 | 8주차 | kubectl, helm, doctl, aws | terraform (자동 설치) |
| 최종 목표 | 8주차 | 11개 도구 | trh-sdk 바이너리 1개 |

---

## 리스크 및 대응

| 리스크 | 가능성 | 대응 |
|--------|-------|------|
| client-go API 복잡성 | 높음 | kubectl 소스코드 참조, 2주차에 PoC 우선 |
| 바이너리 100MB 초과 | 중간 | Go build tags로 미사용 모듈 제외, UPX 압축 검토 |
| AWS SDK 엣지 케이스 | 중간 | 78개 호출 사이트 우선순위화, 핵심 20% 먼저 |
| terraform 자동 설치 실패 | 낮음 | tfinstall 실패 시 명확한 에러 메시지 + 수동 설치 가이드 |
| 기존 호출 코드 회귀 | 낮음 | --legacy 폴백으로 즉시 롤백 가능 |
