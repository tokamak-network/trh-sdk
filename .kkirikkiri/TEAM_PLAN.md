# 팀 작업 계획

- 팀명: kkirikkiri-dev-migration-final
- 목표: TRH SDK 남은 shell-out 완전 제거 (backup/*.go 58개 + thanos 미완 helm 7개)
- 생성 시각: 2026-03-09

## 팀 구성
| 이름 | 역할 | 모델 | 담당 업무 |
|------|------|------|----------|
| team-lead | 팀장 | Opus | 아키텍처 감독, 코드 리뷰, 통합 빌드 검증 |
| developer-1 | 개발자 1 | Opus | 남은 thanos helm call site 마이그레이션 (block_explorer.go, uptime_service.go, deploy_chain.go) |
| developer-2 | Opus | Opus | backup/*.go 전체 마이그레이션 (aws + kubectl 58개) |
| tester | 테스터 | Sonnet | backup/*.go 마이그레이션 함수 단위 테스트 |

## 이전 팀(kkirikkiri-development-phase2) 완료 항목
- ✅ pkg/runner/helm.go — HelmRunner 인터페이스
- ✅ pkg/runner/native_helm.go — NativeHelmRunner
- ✅ pkg/runner/shellout_helm.go — ShellOutHelmRunner
- ✅ pkg/runner/do.go + native_do.go + shellout_do.go — DORunner
- ✅ pkg/runner/mock/mock_helm.go, mock_do.go — 모든 mock
- ✅ pkg/runner/runner.go — Helm(), DO(), AWS(), TF(), K8s() 전부
- ✅ pkg/stacks/thanos/thanos_stack.go — helmRunner, k8sRunner, tfRunner, awsRunner 필드 + 헬퍼 전부
- ✅ destroy_chain.go, monitoring.go, bridge.go, cross_trade.go, update_network.go — helm 마이그레이션 완료

## 남은 작업 (이번 팀 목표)

### developer-1: 남은 thanos helm 호출 (7개)
- pkg/stacks/thanos/block_explorer.go — install, upgrade, uninstall (3개)
- pkg/stacks/thanos/uptime_service.go — install/upgrade, uninstall (2개)
- pkg/stacks/thanos/deploy_chain.go — repo add, repo search (2개)
- 패턴: thanos_stack.go의 helmInstallWithFiles, helmUpgradeWithFiles, helmUninstall, helmRepoAdd, helmSearch 헬퍼 사용

### developer-2: backup/*.go 마이그레이션 (58개)
- pkg/stacks/thanos/backup/*.go 전체
- aws 호출: awsRunner.S3* / awsRunner.EC2* / awsRunner.EKS* 등 (awsRunner 필드 사용)
- kubectl 호출: k8sRunner 메서드 사용
- 패턴: `if t.awsRunner != nil { awsRunner.Method() } else { utils.ExecuteCommand(ctx, "aws", ...) }`
- 주의: ThanosStack에 awsRunner, k8sRunner 필드 이미 존재 (thanos_stack.go 확인)

### tester: 단위 테스트
- backup/ 마이그레이션 함수에 대한 mock 기반 테스트
- mock/mock_aws.go, mock/mock_k8s.go 사용
- 패턴: thanos_stack_runners_test.go 참조

## 절대 규칙
1. ExecuteCommand() 시그니처 변경 금지
2. ShellOutRunner는 항상 폴백으로 유지 (runner nil → shellout)
3. context 전파 필수, 에러 항상 처리
4. 바이너리 크기 150MB 초과 금지 (CI 한도)
5. go build ./... 항상 성공해야 함

## 빌드 명령
```bash
GOMODCACHE=/tmp/gomodcache go build ./...
GOMODCACHE=/tmp/gomodcache go test -race ./...
```
(go module cache root 권한 문제 우회)

## 태스크 목록
- [ ] T1: block_explorer.go helm 마이그레이션 → developer-1
- [ ] T2: uptime_service.go helm 마이그레이션 → developer-1
- [ ] T3: deploy_chain.go helm 마이그레이션 → developer-1
- [ ] T4: backup/*.go aws/kubectl 마이그레이션 → developer-2
- [ ] T5: backup/ 단위 테스트 → tester
- [ ] T6: 통합 빌드 검증 → team-lead
