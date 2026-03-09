# 팀 작업 계획

- 팀명: kkirikkiri-development-phase2
- 목표: TRH SDK Phase 2 — helm 21회 + doctl 4회 shell-out 제거
- 생성 시각: 2026-03-09

## 팀 구성
| 이름 | 역할 | 모델 | 담당 업무 |
|------|------|------|----------|
| team-lead | 팀장 | Opus | 아키텍처 설계, 태스크 분배, 코드 리뷰, 통합 판단 |
| developer-1 | 개발자 1 | Opus | NativeHelmRunner 구현 (helm.sh/helm/v3) |
| developer-2 | 개발자 2 | Opus | NativeDORunner 구현 (github.com/digitalocean/godo) + call site 마이그레이션 |
| tester | 테스터 | Sonnet | 유닛 테스트 + mock 작성 |

## 컨텍스트: Phase 1 완료 상태
- pkg/runner/k8s.go — K8sRunner 인터페이스 (9개 메서드)
- pkg/runner/runner.go — ToolRunner 팩토리
- pkg/runner/native_k8s.go — NativeK8sRunner (client-go v0.29.3)
- pkg/runner/shellout.go — ShellOutK8sRunner (레거시)
- pkg/runner/mock/mock_k8s.go — K8sRunner mock
- go.mod에 k8s.io/client-go v0.29.3 이미 추가됨

## Phase 2 추가 대상
### helm (21개 call site)
- pkg/utils/helm.go (3) — ListReleases, Install, Uninstall 헬퍼
- pkg/stacks/thanos/deploy_chain.go (2) — repo add, search
- pkg/stacks/thanos/monitoring.go (3) — dependency update, install, uninstall
- pkg/stacks/thanos/bridge.go (2) — install, upgrade
- pkg/stacks/thanos/block_explorer.go (3) — install, upgrade, uninstall
- pkg/stacks/thanos/uptime_service.go (2) — install, upgrade
- pkg/stacks/thanos/update_network.go (1) — upgrade
- pkg/stacks/thanos/destroy_chain.go (2) — uninstall x2
- pkg/stacks/thanos/cross_trade.go (2) — install, upgrade
- pkg/dependencies/dependencies.go (1) — version check

### doctl (4개 call site)
- pkg/cloud-provider/digitalocean/digitalocean.go (3) — account get, region list, kubeconfig save (exec.CommandContext 직접 사용)
- pkg/dependencies/dependencies.go (1) — version check

## 신규 파일 목록
- pkg/runner/helm.go — HelmRunner 인터페이스
- pkg/runner/native_helm.go — NativeHelmRunner (helm.sh/helm/v3)
- pkg/runner/shellout_helm.go — ShellOutHelmRunner (폴백)
- pkg/runner/do.go — DORunner 인터페이스
- pkg/runner/native_do.go — NativeDORunner (godo)
- pkg/runner/shellout_do.go — ShellOutDORunner (폴백)
- pkg/runner/mock/mock_helm.go — HelmRunner mock
- pkg/runner/mock/mock_do.go — DORunner mock
- pkg/runner/native_helm_test.go — NativeHelmRunner 테스트
- pkg/runner/native_do_test.go — NativeDORunner 테스트

## 절대 규칙 (PRD/04_PROJECT_SPEC.md)
1. ExecuteCommand() 시그니처 변경 금지
2. ShellOutRunner는 항상 폴백으로 유지 (--legacy / TRHS_LEGACY=1)
3. 각 Runner는 독립적으로 테스트 가능해야 함
4. 바이너리 크기 100MB 초과 금지
5. context 전파 필수, 에러 항상 처리

## 태스크 목록
- [ ] T1: NativeHelmRunner 구현 → developer-1
- [ ] T2: NativeDORunner 구현 → developer-2
- [ ] T3: HelmRunner mock + 테스트 → tester
- [ ] T4: DORunner mock + 테스트 → tester
- [ ] T5: helm call site 마이그레이션 (pkg/utils/helm.go) → developer-1
- [ ] T6: helm call site 마이그레이션 (pkg/stacks/thanos/*) → developer-1
- [ ] T7: doctl call site 마이그레이션 → developer-2
- [ ] T8: runner.go에 Helm()/DO() 메서드 추가 → developer-1
- [ ] T9: 통합 빌드 검증 → team-lead

## 주요 결정사항
(팀장이 결정할 때마다 여기에 기록)
