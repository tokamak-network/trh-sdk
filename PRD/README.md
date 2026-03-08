# TRH SDK — Go 라이브러리 내재화 PRD

**프로젝트**: tokamak-network/trh-sdk
**작성일**: 2026-03-09
**상태**: 기획 완료, Phase 1 착수 예정

---

## 한 줄 요약

`kubectl`, `helm`, `doctl`, `aws`, `terraform` 5개 외부 바이너리 의존성을 Go 라이브러리로 내재화하여, 설치 필요 도구를 11개 → 1개(trh-sdk 바이너리)로 줄인다.

---

## 문서 목록

| 문서 | 내용 |
|------|------|
| [01_PRD.md](01_PRD.md) | 배경, 목표, 핵심 설계, 사용자 스토리, 성공 지표 |
| [02_DATA_MODEL.md](02_DATA_MODEL.md) | ToolRunner 인터페이스, 서브 Runner 상세, 라이브러리 매핑 |
| [03_PHASES.md](03_PHASES.md) | Phase 1~3 주차별 계획, 마일스톤, 리스크 |
| [04_PROJECT_SPEC.md](04_PROJECT_SPEC.md) | AI 행동 규칙, 코딩 컨벤션, 테스트 요구사항 |

---

## 빠른 참조

### 교체 라이브러리

| 도구 | 호출 횟수 | 교체 라이브러리 | Phase |
|------|----------|----------------|-------|
| kubectl | 114회 | `k8s.io/client-go` | 1 |
| helm | 21회 | `helm.sh/helm/v3/pkg/action` | 2 |
| doctl | 4회 | `github.com/digitalocean/godo` | 2 |
| aws CLI | 78회 | `github.com/aws/aws-sdk-go-v2` | 3 |
| terraform | 6회 | `tfinstall` 자동 설치 | 3 |

### 핵심 설계 원칙

- `ExecuteCommand()` 인터페이스 유지 — 기존 호출 코드 변경 없음
- `ShellOutRunner` ↔ `NativeRunner` — `--legacy` 플래그로 전환
- context 전파 필수, 에러 래핑 필수

### 성공 지표

| 지표 | 현재 | 목표 |
|------|------|------|
| 설치 필요 도구 | 11개 | 1개 |
| 설치 시간 | 10~30분 | 30초 |
| 바이너리 크기 | ~10MB (+ 외부 도구 수GB) | ≤100MB |

---

## 개발 시작 방법

```bash
# Phase 1 브랜치 생성
git checkout -b feat/runner-k8s-native

# 신규 패키지 초기화
mkdir -p pkg/runner/mock
touch pkg/runner/runner.go pkg/runner/k8s.go pkg/runner/shellout.go

# 의존성 추가
go get k8s.io/client-go@v0.29.0
go get k8s.io/api@v0.29.0
go get k8s.io/apimachinery@v0.29.0
```
