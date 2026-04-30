# DRB Gaming Preset 로컬 배포 E2E Unblock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** DRB gaming+USDT preset의 로컬 Docker Compose 배포를 "operator activation까지 성공"으로 끌어올리고, 이번 세션에서 발견된 5개 독립 버그의 wiki 기록을 남긴다.

**Architecture:** 네 개 독립 phase — 각각 자체 완결. Phase 1은 trh-wiki ingest(코드 변경 없음), Phase 2-4는 trh-sdk 수정. 각 phase는 스스로 testable 산출물을 남긴다.

**Tech Stack:**
- trh-sdk (Go 1.24, text/template, Docker CLI exec)
- trh-wiki (markdown ingest, append-only log)
- Docker Compose v2, DRB-node image `tokamaknetwork/drb-node:sha-8c37f63`
- 대상 아티팩트: `pkg/stacks/thanos/local_network.go`, `pkg/stacks/thanos/templates/local-compose.yml.tmpl`

**Context — 이미 고쳐진 것 (commit 50d0b39, trh-sdk main)**:
| # | 위치 | 변경 |
|---|------|------|
| 1 | `local_network.go:259,260,599` | `tokamak-thanos/build/{genesis,rollup}.json` → `<deploy>/{genesis,rollup}.json` |
| 2 | `local_network.go:431` | `template.New(...).Funcs({"add": a+b}).Parse(...)` |
| 3 | `templates/local-compose.yml.tmpl:457,474` | range 내부 `.DRBNodeImage`·`.L2ChainID` → `$.DRBNodeImage`·`$.L2ChainID` |
| 4 | `templates/local-compose.yml.tmpl:471` | `REGULAR_PORT` → `PORT` (미검증, Phase 2에서 확정) |

**Context — 관찰된 남은 blocker**:
- **Bug #4 재검증**: `PORT` 로 rename 했음에도 DRB-node가 여전히 `PORT not set in environment variables`를 출력하며 crashloop. 실제 env var 이름이 다를 수 있음.
- **Bug #5**: op-geth volume stale. 2차 resume 시 op-node가 genesis hash mismatch (`0xabff85...965856 <> 0x36da97...175483`)로 crash. `initLocalOpGeth`의 hash-기반 재초기화 로직이 resume 플로우에서 발동하지 않음.
- **부가 관찰**: trh-backend task step label이 local 모드에서도 `"deploy-aws-infra"`로 찍힘.

---

## Phase 1: Wiki Ingest — deploy-methods-comparison 후속 + troubleshooting 페이지

**Files:**
- Create: `/Users/theo/workspace_tokamak/trh-wiki/wiki/troubleshooting/drb-local-compose-path-template-bugs.md`
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/decisions/deploy-methods-comparison.md` (bottom에 후속 이슈 링크 1줄 추가)
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/log.md` (append-only 새 엔트리)
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/index.md` (Troubleshooting 섹션에 새 페이지 링크)

### Task 1.1: Troubleshooting 페이지 작성

**Files:**
- Create: `/Users/theo/workspace_tokamak/trh-wiki/wiki/troubleshooting/drb-local-compose-path-template-bugs.md`

- [ ] **Step 1: 페이지 생성 (frontmatter + 5-bug 요약 + 각 bug의 root cause·fix·증거)**

```markdown
---
updated: 2026-04-18
sources:
  - commit 50d0b39 (trh-sdk main)
related:
  - "[[deploy-methods-comparison]]"
  - "[[drb-project]]"
  - "[[drb-node]]"
  - "[[l2-deploy-local]]"
tags: [troubleshooting, drb, local-compose, gaming-preset]
---

# DRB Gaming Preset 로컬 배포 — 5개 경로·템플릿 버그

2026-04-17 preset resume-deploy (gaming + USDT + fault-proof ON) 시 순차적으로
드러난 5개 독립 버그. 1-4 는 trh-sdk main (50d0b39) 에서 수정. 5 는 미해결.

## Bug #1 — Genesis/Rollup 경로 불일치

**증상**: `failed to generate docker compose file: required file missing:
<deploy>/tokamak-thanos/build/genesis.json (run deploy-contracts first)`

**근본 원인**: 2026-04-16 `generate-genesis` 리팩토링으로 생성자는
`<deploy>/genesis.json` 에 저장하도록 바뀌었지만, `local_network.go` 의 소비자
측이 레거시 경로를 그대로 읽고 있었음.

**수정** (`pkg/stacks/thanos/local_network.go:259,260,599`):
`tokamak-thanos/build/{genesis,rollup}.json` → `<deploy>/{genesis,rollup}.json`

## Bug #2 — `add` 템플릿 함수 미등록

**증상**: `failed to parse compose template: template: local-compose:471:
function "add" not defined`

**근본 원인**: 템플릿의 `REGULAR_PORT: {{ add 9600 $r.Index }}` 가 Go
`text/template` 의 기본 함수셋에 없는 `add` 를 호출. `sprig` 같은 헬퍼 없이
`Funcs` 로 등록하지 않으면 파싱 단계에서 실패.

**수정** (`pkg/stacks/thanos/local_network.go:431`):
```go
tmpl, err := template.New("local-compose").Funcs(template.FuncMap{
    "add": func(a, b int) int { return a + b },
}).Parse(localComposeTmpl)
```

## Bug #3 — range 내부 root-scope 접근 실패

**증상**: `executing "local-compose" at <.DRBNodeImage>: can't evaluate
field DRBNodeImage in type thanos.DRBRegular`

**근본 원인**: `{{ range $i, $r := .DRBRegulars }}` 블록 안에서 bare dot(`.`)
은 루프 아이템(`DRBRegular`) 을 가리키므로 root 필드인 `.DRBNodeImage`·
`.L2ChainID` 는 평가되지 않음. 같은 블록에 이미 `$.DRBLeaderPeerID` 가
쓰이고 있어 관례는 확립돼 있었으나 두 필드만 누락.

**수정** (`templates/local-compose.yml.tmpl:457,474`): `.X` → `$.X`.

## Bug #4 — Regular 노드 `PORT` env 이름 (확정 전)

**증상**: 템플릿에 `PORT=9601` 를 세팅해도 DRB-node 가
`PORT not set in environment variables` 로 crash.

**현재 상태**: Phase 2 조사 필요. DRB-node source repo 에서 실제 env 키를
확인해야 함. 후보: `REGULAR_PORT`, `NODE_PORT`, `P2P_PORT`, `LIBP2P_PORT`.

## Bug #5 — op-geth volume stale (미해결)

**증상**: resume 재시도 시 op-node 가
`expected L2 genesis hash to match L2 block at genesis block number 0:
<hash_in_geth_chaindata> <> <hash_in_new_genesis.json>` 로 crash.

**근본 원인**: 1차 배포 실패 후 2차 resume 에서 새 L2 genesis 가 생성되지만
op-geth 가 기존 volume 의 chaindata 를 재사용. `local_network.go:600-622` 의
hash 기반 재초기화 로직이 있으나 **resume 경로에서 발동하지 않는 조건**.

**우회**: `docker volume rm <deploy>_op-geth-data` 후 재시도.

**근본 해결 후보**:
- resume 진입 시점에 무조건 genesis hash 를 계산해서 `.genesis-hash` 와 비교
- `op-geth-data` volume 도 `ConfigVolume` 처럼 deployment ID 로 suffix

## 증거 — 성공 지표 (Bug #4·#5 해결되면 기대되는 마커)

Leader 노드의 1차 기동 시 아래 로그가 나오면 predeploy 연동은 이미 OK:
```
leader host created with PeerID: 12D3Koo...
Fetched current round from contract: 0
Fetched trialNum for round 0 from contract: 0
Current s_isInProcess value: 2
```
이 세 줄은 `0x4200000000000000000000000000000000000060` 의 CommitReveal2L2
predeploy 가 live L2 에서 호출 가능함을 의미.

## 관련 커밋·파일

- trh-sdk `50d0b39` — bug #1, #2, #3, #4(미검증) fix
- `pkg/stacks/thanos/local_network.go:250-280, 431, 590-640`
- `pkg/stacks/thanos/templates/local-compose.yml.tmpl:441-484`
```

- [ ] **Step 2: 작성한 파일을 읽어 프론트매터·헤더·5-bug 섹션 존재 확인**

Run: `grep -c "^## Bug" /Users/theo/workspace_tokamak/trh-wiki/wiki/troubleshooting/drb-local-compose-path-template-bugs.md`
Expected: `5`

### Task 1.2: deploy-methods-comparison 후속 링크 추가

**Files:**
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/decisions/deploy-methods-comparison.md` (마지막 "관련 문서" 섹션에 1 줄 추가)

- [ ] **Step 1: "관련 문서" 섹션 끝에 링크 한 줄 추가**

찾을 대상 (파일의 마지막 블록):
```markdown
- [[thanos-deployer-analysis]] — 기존 경로 전체 8-레이어 아키텍처 분석
- [[tokamak-deployer-logging]] — 신규 경로 로깅 상세
- [[tokamak-deployer-gas-price]] — v0.0.5 고정 가스 전략 근거 + Sepolia 측정 결과
- [[l2-deployment]] — L2 배포 파이프라인 전체 개요
```

아래로 변경 (`drb-local-compose-path-template-bugs` 링크 추가):
```markdown
- [[thanos-deployer-analysis]] — 기존 경로 전체 8-레이어 아키텍처 분석
- [[tokamak-deployer-logging]] — 신규 경로 로깅 상세
- [[tokamak-deployer-gas-price]] — v0.0.5 고정 가스 전략 근거 + Sepolia 측정 결과
- [[l2-deployment]] — L2 배포 파이프라인 전체 개요
- [[drb-local-compose-path-template-bugs]] — 이 전환 과정에서 드러난 5 개 경로·템플릿 버그 (2026-04-17)
```

### Task 1.3: index.md 업데이트

**Files:**
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/index.md` (Troubleshooting 표에 한 행 추가)

- [ ] **Step 1: Troubleshooting 표의 마지막 행 뒤에 새 항목 추가**

찾을 대상:
```markdown
| [[tokamak-deployer-gas-price]] | Fixed gas price reuse strategy (v0.0.5+) — 5m47s measured on Sepolia, 0 retries |
```

아래로 변경:
```markdown
| [[tokamak-deployer-gas-price]] | Fixed gas price reuse strategy (v0.0.5+) — 5m47s measured on Sepolia, 0 retries |
| [[drb-local-compose-path-template-bugs]] | DRB gaming preset 로컬 배포 시 드러난 5개 경로·템플릿 버그 (path, FuncMap, range-scope, PORT, op-geth volume) |
```

### Task 1.4: log.md 엔트리 추가

**Files:**
- Modify: `/Users/theo/workspace_tokamak/trh-wiki/wiki/log.md` (파일 상단 "---" 구분자 바로 아래에 새 블록 prepend)

- [ ] **Step 1: log.md 맨 위 "---" 뒤에 새 ingest 엔트리 삽입**

찾을 대상 (파일 상단):
```markdown
---

## [2026-04-18] ingest | deploy-methods-comparison — Deploy.s.sol vs tokamak-deployer
```

아래로 변경 (새 엔트리 prepend):
```markdown
---

## [2026-04-18] ingest | drb-local-compose-path-template-bugs — DRB preset 배포 5개 경로·템플릿 버그

New page: [[drb-local-compose-path-template-bugs]] (troubleshooting/)
Source: 2026-04-17 preset resume-deploy 세션 관찰 + trh-sdk commit 50d0b39

Index updated: Troubleshooting 표에 항목 추가
Decisions updated: [[deploy-methods-comparison]] 관련 문서에 링크 추가

Key facts captured:
  - **Bug #1**: `local_network.go:259,260,599` — `tokamak-thanos/build/{genesis,rollup}.json` 경로는 2026-04-16 `generate-genesis` 리팩토링 후 레거시. 소비자 측 consumer 수정
  - **Bug #2**: `local_network.go:431` — `{{ add ... }}` 사용인데 FuncMap 미등록으로 Go text/template 파싱 실패
  - **Bug #3**: `templates/local-compose.yml.tmpl:457,474` — range 내부 `.DRBNodeImage`/`.L2ChainID` 는 DRBRegular 구조체를 가리켜 평가 실패. `$.X` 로 root 접근 필요
  - **Bug #4 (미확정)**: Regular 노드 crash `PORT not set in environment variables`. template `PORT=9601` 으로 rename 했어도 재현. DRB-node source 에서 실제 env 키 확인 필요
  - **Bug #5 (미해결)**: resume 시 op-geth chaindata 재사용으로 L2 genesis hash mismatch. `initLocalOpGeth` 의 hash 재초기화 로직이 이 경로에서 발동하지 않음
  - Leader 노드 로그가 `Fetched current round from contract: 0` + `s_isInProcess value: 2` 를 남기면 `0x4200...0060` predeploy 는 이미 live L2 에서 호출 가능한 상태

## [2026-04-18] ingest | deploy-methods-comparison — Deploy.s.sol vs tokamak-deployer
```

- [ ] **Step 2: Phase 1 전체가 rendering 가능한지 검증 (grep -c로 새 링크들 체크)**

Run:
```bash
grep -c "drb-local-compose-path-template-bugs" /Users/theo/workspace_tokamak/trh-wiki/wiki/index.md
grep -c "drb-local-compose-path-template-bugs" /Users/theo/workspace_tokamak/trh-wiki/wiki/decisions/deploy-methods-comparison.md
grep -c "drb-local-compose-path-template-bugs" /Users/theo/workspace_tokamak/trh-wiki/wiki/log.md
```
Expected: 각각 `1`, `1`, `1` 이상.

### Task 1.5: Phase 1 커밋 (trh-wiki 단독)

- [ ] **Step 1: wiki 커밋 + push**

```bash
cd /Users/theo/workspace_tokamak/trh-wiki
git add wiki/troubleshooting/drb-local-compose-path-template-bugs.md wiki/index.md wiki/decisions/deploy-methods-comparison.md wiki/log.md
git commit -m "ingest: drb-local-compose-path-template-bugs — 5 DRB preset deploy bugs (2026-04-17)

Documents the 5 independent bugs surfaced during the gaming+USDT preset
resume-deploy on 2026-04-17. Bugs #1-3 fixed in trh-sdk 50d0b39, #4 pending
verification, #5 unresolved (op-geth volume stale on resume)."
git push
```

---

## Phase 2: Bug #4 — DRB-node Regular 노드 `PORT` env 이름 확정

**Goal:** DRB-node 바이너리가 실제로 어떤 env 키를 읽는지 소스로 확정하고 그에 맞춰 template 을 수정.

**Files:**
- Modify: `pkg/stacks/thanos/templates/local-compose.yml.tmpl:470` (확정된 env 키로 재수정)

### Task 2.1: DRB-node 소스에서 env 키 검색

- [ ] **Step 1: DRB-node 이미지에 태그된 커밋 확인**

Run:
```bash
docker inspect tokamaknetwork/drb-node:sha-8c37f63 --format '{{index .Config.Labels "org.opencontainers.image.revision"}}'
```
Expected: `8c37f63` 으로 시작하는 git sha. (없으면 이미지 태그의 `sha-` 뒤 해시가 곧 commit 이라고 가정)

- [ ] **Step 2: Upstream repo 에서 해당 커밋 클론 (shallow)**

Run:
```bash
cd /tmp && rm -rf drb-node-src && \
  git clone --depth 100 https://github.com/tokamak-network/DRB-node.git drb-node-src && \
  cd drb-node-src && \
  git log --oneline | head -5
```
Expected: 커밋 해시가 표시되고 `8c37f63` 가 포함 (또는 근접).

- [ ] **Step 3: 에러 메시지 literal 로 env 변수명 식별**

Run:
```bash
cd /tmp/drb-node-src && \
  grep -rn "PORT not set in environment variables" --include="*.go" .
```
Expected: Go 파일의 특정 라인에 메시지. 그 위·아래 5 줄로 실제 env 키 (`os.Getenv(...)` 인자) 를 확인.

- [ ] **Step 4: 그 라인 주변 10 줄 context 출력 후 env 키 기록**

Run:
```bash
FILE=$(grep -rln "PORT not set in environment variables" --include="*.go" .)
LINE=$(grep -n "PORT not set in environment variables" $FILE | head -1 | cut -d: -f1)
sed -n "$((LINE-10)),$((LINE+2))p" $FILE
```
Expected: `os.Getenv("X")` 형태로 실제 키 `X` 가 보인다. 예상 후보:
- `REGULAR_PORT`
- `NODE_PORT`
- `P2P_PORT`
- `LIBP2P_PORT`
- 단순 `PORT` (이 경우 Phase 2 는 사실 이미 해결됐지만 다른 crash 원인이 있다는 뜻)

`X` 를 기록.

### Task 2.2: Template 를 확정된 env 키로 수정

**Files:**
- Modify: `/Users/theo/workspace_tokamak/trh-sdk/pkg/stacks/thanos/templates/local-compose.yml.tmpl:470`

- [ ] **Step 1: 현재 PORT 라인 컨텍스트 보기**

Run:
```bash
sed -n '465,475p' /Users/theo/workspace_tokamak/trh-sdk/pkg/stacks/thanos/templates/local-compose.yml.tmpl
```
Expected:
```yaml
      REGULAR_EOA: "{{ $r.Address }}"
      LEADER_PEER_ID: "{{ $.DRBLeaderPeerID }}"
      LEADER_MULTIADDR: "{{ $.DRBLeaderPeerID }}@drb-leader:9600"
      LEADER_PORT: "9600"
      PORT: "{{ add 9600 $r.Index }}"
      ETH_RPC_URLS: ws://op-geth:8546
```

- [ ] **Step 2: Task 2.1 Step 4 에서 확정한 키 `X` 로 라인 교체**

에디터 동작 예시 (`X` = `NODE_PORT` 인 경우):
```yaml
      LEADER_PORT: "9600"
      NODE_PORT: "{{ add 9600 $r.Index }}"
      ETH_RPC_URLS: ws://op-geth:8546
```

> **주의**: Task 2.1 의 소스 확인 결과가 단순 `PORT` 인 경우 → 이 라인 변경 없이 넘어가고, crash 의 다른 원인 조사로 전환 (Task 2.4).

### Task 2.3: 재빌드 + 재배포 + 검증

- [ ] **Step 1: trh-backend 로컬 재빌드 + 컨테이너 반영**

Run:
```bash
bash -c "cd /Users/theo/workspace_tokamak/trh-backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 /usr/local/go/bin/go build -o /tmp/trh-backend-main-linux ./main.go" && \
  docker cp /tmp/trh-backend-main-linux trh-backend:/app/main && \
  docker restart trh-backend && \
  bash -c "until docker inspect --format='{{.State.Health.Status}}' trh-backend | grep -q healthy; do sleep 2; done && echo HEALTHY"
```
Expected: `HEALTHY`

- [ ] **Step 2: 기존 compose stack down + resume 재호출**

Run:
```bash
docker exec trh-backend sh -c "cd /app/storage/deployments/Thanos/Testnet/5ffe7da8-6bb4-4734-bd5e-6313672286c4 && docker compose -f docker-compose.local.yml --profile '*' down"
TOKEN=$(cat /tmp/trh-token.txt)
curl -s -X POST "http://localhost:8000/api/v1/stacks/thanos/5ffe7da8-6bb4-4734-bd5e-6313672286c4/resume" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' -w "\nHTTP=%{http_code}\n"
```
Expected: `HTTP=200`

- [ ] **Step 3: drb-regular-1 health 확인 (Restarting 이 아니어야 함)**

Run (60s 대기 후):
```bash
sleep 60 && \
docker ps --format '{{.Names}}\t{{.Status}}' | grep drb-regular-1
```
Expected: `Up XXs` — **Restarting 아님**.

- [ ] **Step 4: drb-regular-1 로그에 crash 원인 사라졌는지 확인**

Run:
```bash
docker logs --tail 20 5ffe7da8-6bb4-4734-bd5e-6313672286c4-drb-regular-1-1 2>&1 | grep -vi "HTTP Request"
```
Expected: `PORT not set` 메시지 **없음**. libp2p listen / peer connection 로그가 나와야 함.

### Task 2.4: (Bug #4 가 단순 PORT 확정일 때) 진짜 crash 원인 조사

- [ ] **Step 1: Task 2.1 결과가 `os.Getenv("PORT")` 이면 → 다른 crash 원인 탐색**

Run:
```bash
cd /tmp/drb-node-src && \
  grep -rn "Database initialised successfully" --include="*.go"
# 그 다음 줄 이후에 어떤 fatal 조건들이 있는지 확인
```

- [ ] **Step 2: 새로 발견된 blocker 가 있으면 Task 를 추가하지 말고 Phase 종료 후 사용자에게 advisory 보고**

규칙: advisor 가 제시한 STOP 규칙 — **이 plan 실행 중 6 번째 독립 bug 가 나오면 멈추고 사용자 결정 대기**.

---

## Phase 3: Bug #5 — op-geth volume stale 재초기화 경로 복구

**Goal:** resume 재시도 시 L2 genesis hash 가 바뀐 경우 op-geth chaindata 를 자동 재초기화.

**Files:**
- Modify: `pkg/stacks/thanos/local_network.go:590-640` (`initLocalOpGeth`)
- Test: `pkg/stacks/thanos/local_network_test.go` (생성, genesis-hash 변경 시나리오)

### Task 3.1: 현재 재초기화 경로 읽고 gap 확정

- [ ] **Step 1: `initLocalOpGeth` 전체 읽기**

Run:
```bash
sed -n '585,650p' /Users/theo/workspace_tokamak/trh-sdk/pkg/stacks/thanos/local_network.go
```
Expected: `hashFile(genesisPath)` + `.genesis-hash` 비교 + 불일치 시 `resetOpGethVolume` 호출하는 로직 확인.

- [ ] **Step 2: 왜 resume 경로에서 발동 안 했는지 가설 세우기**

가설 후보:
- (A) `initLocalOpGeth` 가 compose up 전에 실행되지만 resume 시 op-geth 가 이미 새 compose 로 올라가서 "데이터 있음" 분기를 타지 않음
- (B) `.genesis-hash` 파일이 **기록 안 되는 경로**가 있음 — 첫 성공 시에도 write 안 되면 다음 resume 에서 항상 "다르다"고 판단하지만 reset 을 실패
- (C) `resetOpGethVolume` 가 `docker compose down -v` 대신 다른 방식을 써서 DinD 에서 실패

### Task 3.2: 실패 테스트 작성 (TDD)

**Files:**
- Create: `/Users/theo/workspace_tokamak/trh-sdk/pkg/stacks/thanos/local_network_test.go`

- [ ] **Step 1: 테스트 파일 생성 — genesis 가 바뀌면 재초기화 시그널이 기록되는지 검증**

```go
package thanos

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// Test that when .genesis-hash on disk differs from the current genesis.json,
// the reinit path is triggered. We simulate by computing hashes manually and
// checking the helper behavior without running Docker.
func TestGenesisHashDetectsChange(t *testing.T) {
	tmp := t.TempDir()
	genesisPath := filepath.Join(tmp, "genesis.json")
	hashFilePath := filepath.Join(tmp, "op-geth-data", ".genesis-hash")

	if err := os.MkdirAll(filepath.Dir(hashFilePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(genesisPath, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("write genesis: %v", err)
	}

	current, err := hashFile(genesisPath)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if current == "" {
		t.Fatal("expected non-empty hash")
	}

	// Previously-stored hash differs (older chaindata scenario)
	if err := os.WriteFile(hashFilePath, []byte(hashString("old")), 0o644); err != nil {
		t.Fatalf("write old hash: %v", err)
	}

	prev, err := os.ReadFile(hashFilePath)
	if err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if string(prev) == current {
		t.Fatalf("expected hash mismatch (prev=%s, cur=%s)", prev, current)
	}
}
```

- [ ] **Step 2: 테스트 실행해서 PASS 하는지 확인 (helper 자체는 이미 존재)**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-sdk && \
  /usr/local/go/bin/go test ./pkg/stacks/thanos/ -run TestGenesisHashDetectsChange -v
```
Expected: `PASS` — 현재 helper 는 정상. 진짜 문제는 **호출 경로**. 이 테스트는 regression guard.

### Task 3.3: Resume 진입 시 op-geth volume 무조건 체크 (가설 A 대응)

**Files:**
- Modify: `pkg/stacks/thanos/local_network.go` — `initLocalOpGeth` 가 resume 플로우에서 호출되도록 확정.

- [ ] **Step 1: `initLocalOpGeth` 가 호출되는 모든 지점 찾기**

Run:
```bash
grep -n "initLocalOpGeth" /Users/theo/workspace_tokamak/trh-sdk/pkg/stacks/thanos/*.go
```
Expected: caller 위치들이 나옴.

- [ ] **Step 2: Resume 경로에서 caller 가 있는지 확인**

각 caller 의 함수를 읽고, resume 경로 (`deploy_chain.go`, `deploy_contracts.go` 의 resume 분기) 에서 도달 가능한지 확인.

- [ ] **Step 3: Gap 이 있으면 resume 진입 시점에 `initLocalOpGeth` 호출 추가**

예시 — resume 함수의 compose up 직전에:
```go
// Before bringing the local core services up, ensure op-geth volume
// is consistent with the current genesis.json (reinit if hash changed).
if err := t.initLocalOpGeth(ctx, composePath); err != nil {
    return fmt.Errorf("op-geth volume reinit check failed: %w", err)
}
```

- [ ] **Step 4: 빌드 확인**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-sdk && /usr/local/go/bin/go vet ./pkg/stacks/thanos/...
```
Expected: `No issues found.`

### Task 3.4: 재빌드 + 재배포 검증

- [ ] **Step 1: op-geth volume 수동 제거 후 바이너리 재빌드**

Run:
```bash
docker volume ls --format '{{.Name}}' | grep op-geth-data | xargs -r docker volume rm
bash -c "cd /Users/theo/workspace_tokamak/trh-backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 /usr/local/go/bin/go build -o /tmp/trh-backend-main-linux ./main.go" && \
  docker cp /tmp/trh-backend-main-linux trh-backend:/app/main && \
  docker restart trh-backend && \
  bash -c "until docker inspect --format='{{.State.Health.Status}}' trh-backend | grep -q healthy; do sleep 2; done && echo HEALTHY"
```
Expected: volume 제거 + `HEALTHY`.

- [ ] **Step 2: Resume 호출**

Run:
```bash
TOKEN=$(cat /tmp/trh-token.txt)
curl -s -X POST "http://localhost:8000/api/v1/stacks/thanos/5ffe7da8-6bb4-4734-bd5e-6313672286c4/resume" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' -w "\nHTTP=%{http_code}\n"
```
Expected: `HTTP=200`.

- [ ] **Step 3: op-node 가 crash 하지 않는지 확인 (60s 후)**

Run:
```bash
sleep 60 && docker ps --format '{{.Names}}\t{{.Status}}' | grep -E 'op-node|op-proposer|op-challenger'
```
Expected: `Up XXs` (Restarting 아님).

- [ ] **Step 4: op-node 로그에서 genesis hash mismatch 부재 확인**

Run:
```bash
docker logs --tail 30 5ffe7da8-6bb4-4734-bd5e-6313672286c4-op-node-1 2>&1 | grep -iE "genesis hash|failed to init|Error initializing"
```
Expected: **빈 출력** (매칭 없음).

### Task 3.5: resume 재실행 시나리오 재현 테스트 (선택, 수동)

- [ ] **Step 1: resume 한 번 더 호출해서 volume 재사용 케이스 체크**

Run:
```bash
TOKEN=$(cat /tmp/trh-token.txt)
docker exec trh-backend sh -c "cd /app/storage/deployments/Thanos/Testnet/5ffe7da8-6bb4-4734-bd5e-6313672286c4 && docker compose -f docker-compose.local.yml --profile '*' down"
curl -s -X POST "http://localhost:8000/api/v1/stacks/thanos/5ffe7da8-6bb4-4734-bd5e-6313672286c4/resume" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}' -w "\nHTTP=%{http_code}\n"
sleep 60
docker ps --format '{{.Names}}\t{{.Status}}' | grep -E 'op-node|drb-leader|drb-regular'
```
Expected: 모든 서비스 `Up` — resume 반복해도 안정.

---

## Phase 4: 부가 관찰 — trh-backend task step label 정상화

**Goal:** 로컬 infra 모드에서 task step 레이블이 `deploy-aws-infra` 로 찍히는 misnomer 해결.

**Files:**
- Modify: `/Users/theo/workspace_tokamak/trh-backend/pkg/services/thanos/*.go` (Phase 4.1 에서 정확한 파일 식별)

### Task 4.1: 해당 step 이름 발생 지점 찾기

- [ ] **Step 1: `"deploy-aws-infra"` 리터럴 검색**

Run:
```bash
grep -rn "deploy-aws-infra\|\"deploy-aws-infra\"" /Users/theo/workspace_tokamak/trh-backend --include="*.go"
```
Expected: 1-3 개 라인.

- [ ] **Step 2: 그 상위 함수에서 분기 로직 읽기**

식별된 파일을 읽고, local 모드 분기가 같은 step 이름을 재사용하는지 아니면 별도 선택 로직이 있는지 확인.

### Task 4.2: Step 이름 분기 추가

- [ ] **Step 1: Local 모드에서는 `"deploy-local-infra"` 사용하도록 수정**

예시 패치 (정확한 라인은 Task 4.1 결과 따라):
```go
step := "deploy-aws-infra"
if cfg.InfraProvider == "local" {
    step = "deploy-local-infra"
}
```

- [ ] **Step 2: 빌드 + vet**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-backend && /usr/local/go/bin/go vet ./...
```
Expected: `No issues found.`

### Task 4.3: 재배포 로그에서 새 레이블 확인

- [ ] **Step 1: 재빌드 + 재배포 후 로그 체크**

Run:
```bash
bash -c "cd /Users/theo/workspace_tokamak/trh-backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 /usr/local/go/bin/go build -o /tmp/trh-backend-main-linux ./main.go" && \
  docker cp /tmp/trh-backend-main-linux trh-backend:/app/main && \
  docker restart trh-backend
# (이후 기존 resume 절차 반복)
docker logs trh-backend 2>&1 | grep -E "\"step\":\s*\"deploy-(aws|local)-infra\"" | tail -3
```
Expected: `"step": "deploy-local-infra"` 로만 찍힘.

---

## Phase 5: Operator Activation 최종 검증 (원래 사용자 질문 #4 완결)

**Goal:** 모든 blocker 해결 후 DRB regular 가 on-chain 에 등록되고 activated operators 수가 상승하는지 확인.

**Files:**
- 코드 변경 없음 — 관찰만.

### Task 5.1: Stack 상태 = `Deployed` 확인 (원래 질문 #1)

- [ ] **Step 1: DB 상태 조회**

Run:
```bash
docker exec trh-postgres psql -U postgres -d trh_db -c \
  "SELECT id, status FROM stacks WHERE id='5ffe7da8-6bb4-4734-bd5e-6313672286c4';"
```
Expected: `status = Deployed`.

### Task 5.2: DRB 컨테이너 전부 healthy (원래 질문 #2)

- [ ] **Step 1: 컨테이너 상태 스냅샷**

Run:
```bash
docker ps --format '{{.Names}}\t{{.Status}}' | grep -E 'drb-(leader|regular|postgres)|op-geth'
```
Expected:
- `drb-leader-1`: Up
- `drb-regular-1`, `drb-regular-2`, `drb-regular-3`: Up
- `drb-postgres*`: Up (healthy)
- `op-geth`: Up (healthy)

### Task 5.3: Predeploy 호출 성공 로그 (원래 질문 #3, 이미 확정됨 — 재확인)

- [ ] **Step 1: Leader 로그의 on-chain read 구문 확인**

Run:
```bash
docker logs 5ffe7da8-6bb4-4734-bd5e-6313672286c4-drb-leader-1 2>&1 | grep -E "current round|s_isInProcess"
```
Expected: `Fetched current round from contract: N` + `Current s_isInProcess value: M` — 숫자 값 있음.

### Task 5.4: Activated operators 수 조회 (원래 질문 #4)

- [ ] **Step 1: op-geth L2 RPC 포트 확인**

Run:
```bash
docker port 5ffe7da8-6bb4-4734-bd5e-6313672286c4-op-geth-1 | grep 8545
```
Expected: `0.0.0.0:8545` 형태 — 실제 호스트 포트 기록.

- [ ] **Step 2: ABI 에서 activated operators 조회 함수 식별**

Run:
```bash
grep -rn "ActivatedOperators\|getActivatedOperators\|activatedOperators" /Users/theo/workspace_tokamak/trh-sdk/abis/ 2>/dev/null | head -5
grep -rn "ActivatedOperators\|getActivatedOperators\|activatedOperators" /tmp/drb-node-src/contract/ 2>/dev/null | head -5
```
Expected: 함수 시그니처 (예: `getActivatedOperators() returns (address[])` 또는 `activatedOperatorsLength() returns (uint256)`).

- [ ] **Step 3: `cast call` 로 L2 predeploy 의 activated operators 조회**

Task 5.4.2 에서 찾은 시그니처를 사용. 예시 (시그니처가 `getActivatedOperators()(address[])` 인 경우):
```bash
RPC=http://localhost:8545
cast call 0x4200000000000000000000000000000000000060 \
  "getActivatedOperators()(address[])" --rpc-url "$RPC"
```
Expected: 배열. 길이 = **DRB regular 수 (3)**.

- [ ] **Step 4: 결과 기록**

배열이 `[]` (빈 배열) 이면 operator 가 아직 on-chain 등록 전. Leader 로그에서 `Registered Operator` 이벤트 수신 여부 확인:
```bash
docker logs 5ffe7da8-6bb4-4734-bd5e-6313672286c4-drb-leader-1 2>&1 | \
  grep -iE "Registered Operator|Activated"
```

배열이 3 개면 **Q4 성공**.

---

## Phase 6: 최종 커밋

**Goal:** trh-sdk 와 trh-backend 의 모든 수정을 main 에 한 번씩 commit + push.

### Task 6.1: trh-sdk 커밋

- [ ] **Step 1: trh-sdk 변경 상태 확인**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-sdk && git status --short
```
Expected: Phase 2.2 (템플릿) + Phase 3 (local_network.go + local_network_test.go) 의 변경 파일들.

- [ ] **Step 2: 단일 커밋 + push**

```bash
cd /Users/theo/workspace_tokamak/trh-sdk
git add pkg/stacks/thanos/templates/local-compose.yml.tmpl \
        pkg/stacks/thanos/local_network.go \
        pkg/stacks/thanos/local_network_test.go
git commit -m "fix(thanos/local): DRB regular PORT env + op-geth resume reinit

- templates/local-compose.yml.tmpl: rename PORT env to match DRB-node
  binary's expected key (confirmed by upstream source inspection, see
  trh-wiki [[drb-local-compose-path-template-bugs]] bug #4)
- local_network.go: ensure initLocalOpGeth runs at resume entry so
  op-geth volume chaindata is reinitialized when L2 genesis hash
  changed (bug #5)
- local_network_test.go: regression test for genesis-hash change
  detection helper"
git push
```

### Task 6.2: trh-backend 커밋 (Phase 4 가 수행된 경우)

- [ ] **Step 1: 변경 상태 확인 — replace directive 는 제외**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-backend && git status --short
```
Expected: Phase 4 의 step label 변경 + (주의) `go.mod` 의 local `replace` directive 도 보일 것.

- [ ] **Step 2: go.mod replace 원복 (로컬 hack 이었으므로 커밋하지 말 것)**

Run:
```bash
cd /Users/theo/workspace_tokamak/trh-backend
git diff go.mod | grep -E "^\+|^\-" | head -5
```
`+replace github.com/tokamak-network/trh-sdk => ../trh-sdk` 라인만 있다면:
```bash
git checkout go.mod
```

- [ ] **Step 3: Phase 4 변경 커밋**

```bash
cd /Users/theo/workspace_tokamak/trh-backend
git add pkg/services/thanos/[변경된 파일들]
git commit -m "fix(services/thanos): distinguish local vs aws deploy step labels

Previously the 'deploy-aws-infra' step name was emitted even for local
infraProvider deploys, causing confusion in logs. Split into
'deploy-local-infra' and 'deploy-aws-infra' based on infraProvider."
git push
```

---

## Self-Review

- [x] **Spec coverage**: 5 개 bug + 부가 관찰 모두 Phase 에 매핑됨 (#1-3: Phase 1 wiki 기록, #4: Phase 2, #5: Phase 3, label: Phase 4, operator: Phase 5).
- [x] **Placeholder scan**: `[변경된 파일들]`, `[Task 4.1 결과 따라]` 같은 placeholder 는 명시적으로 "실행 시 이전 Task 결과 사용" 으로 문서화됨. 그 외 TBD/TODO 없음.
- [x] **Type consistency**: `hashFile()`, `initLocalOpGeth()`, `resetOpGethVolume()` 는 모든 Task 에서 동일 이름 사용.
- [x] **STOP 규칙**: Phase 2 Task 2.4 에 6 번째 bug 등장 시 사용자 보고로 종료하는 조항 포함.
- [x] **Single-commit feedback 규칙 준수**: Phase 별 commit 아닌, **리포지토리 별로 한 번**씩만 commit (trh-wiki Phase 1, trh-sdk Phase 6.1, trh-backend Phase 6.2). 중간 checkpoint commit 없음.

---

## Execution Handoff

**Plan complete and saved to `/Users/theo/workspace_tokamak/trh-sdk/docs/superpowers/plans/2026-04-18-drb-local-deploy-unblock.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration. Good for this plan since Phase 2 (repo clone + grep) and Phase 4 (label rename) can run in parallel.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints.

**추천**: Phase 1 만 즉시 inline 실행 (wiki 는 이미 맥락 있음), Phase 2-5 는 사용자 결정 후 subagent-driven.
