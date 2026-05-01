# L2 배포 최적화 설계

**날짜:** 2026-04-16  
**범위:** trh-sdk + tokamak-thanos  
**목표:** 기능에 영향 없는 한도 내에서 L2 배포 총 시간 최소화

---

## 배경

TRH 로컬 L2 배포는 현재 end-to-end 약 10-15분 소요. Optimism의 `op-deployer` 방식과 비교해 식별한 주요 병목:

1. `git clone tokamak-thanos` monorepo (~30-60s)
2. `forge incremental build` — npm 패키지에 Isthmus-capable L1Block.sol 아티팩트 미포함으로 항상 강제 재컴파일 (~27-51s)
3. AA Paymaster 설정이 L2 기동을 블로킹 — `bridgeAdminTONForAASetup` + `setupAAPaymaster`가 L1→L2 브리지 폴링을 동기 대기 (최대 5분)
4. L1 tx 확정 대기 외에 `forge script` 실행 오버헤드 추가

## 선택한 접근법: tokamak-deployer 바이너리 (Approach B)

`tokamak-thanos`에서 빌드하는 Go 바이너리로, pre-compiled forge-artifacts를 내장. trh-sdk는 git clone + forge build + forge script 대신 바이너리 다운로드 + 실행으로 대체.

방안 2 (AA Paymaster 비동기화)는 trh-sdk에서 독립적으로 적용.

**방안 1, 3은 방안 6에 흡수** — 바이너리가 모든 아티팩트를 내장하므로 별도 npm 패키지나 tarball 불필요.

---

## 아키텍처

### 현재 (Before)

```
trh-sdk
  └─ git clone tokamak-thanos (~30-60s)
  └─ npm download forge-artifacts
  └─ forge incremental build (~27-51s, L1Block 때문에 항상 실행)
  └─ forge script deploy-contracts (~3-8분, L1 tx 대기)
  └─ [블로킹] AA setup: bridgeAdminTONForAASetup → L2 브리지 폴링 (최대 5분)
  └─ setupAAPaymaster
  └─ alto-bundler start
  └─ L2 core services start
```

### 목표 (After)

```
trh-sdk
  └─ download tokamak-deployer binary (~3-5s 캐시 미스, ~0s 캐시 히트)
  └─ tokamak-deployer deploy-contracts (~3-8분, L1 tx 대기 — 동일)
  └─ tokamak-deployer generate-genesis
  └─ L2 core services start (즉시)
  └─ [백그라운드 goroutine] AA setup: bridgeAdminTONForAASetup → setupAAPaymaster → alto-bundler
```

### 시간 비교

| 단계 | Before | After |
|------|--------|-------|
| tokamak-thanos clone | ~30-60s | **0s** (제거) |
| forge build | ~27-51s | **0s** (제거) |
| 바이너리 다운로드 (캐시 미스) | — | ~3-5s |
| 바이너리 다운로드 (캐시 히트) | — | ~0s |
| L1 deploy tx 확정 대기 | ~3-8분 | ~3-8분 (동일) |
| AA Paymaster 설정 | ~5분 (블로킹) | **0s** (비동기) |
| **체감 총 시간** | **~10-15분** | **~4-9분** |

---

## 컴포넌트 1: tokamak-deployer 바이너리

### 레포: tokamak-thanos

새 디렉토리: `cmd/tokamak-deployer/`

```
cmd/tokamak-deployer/
  main.go
  deploy_contracts.go    # go-ethereum으로 L1 컨트랙트 배포 (forge 불필요)
  generate_genesis.go    # genesis.json 생성 + 후처리
```

### 서브커맨드

```bash
tokamak-deployer deploy-contracts \
  --l1-rpc $L1_RPC \
  --private-key $DEPLOYER_KEY \
  --chain-id $L2_CHAIN_ID \
  --out ./deploy-output.json

tokamak-deployer generate-genesis \
  --deploy-output ./deploy-output.json \
  --config ./rollup-config.json \
  --out ./genesis.json
```

`generate-genesis`는 현재의 5단계 후처리를 모두 포함 (DRB inject, USDC inject, MultiTokenPaymaster inject, L1Block bytecode patch, rollup hash update).

### 아티팩트 번들 방법

- `tokamak-thanos` CI가 릴리즈 빌드 과정에서 `forge build` 실행
- 전체 `forge-artifacts/`가 아닌 **L1 배포에 필요한 컨트랙트만** 선택적으로 embed (전체 디렉토리는 100MB 초과 가능)
- `deploy-artifacts/` 디렉토리에 최소 집합만 추출 (예: `OptimismPortal`, `L1CrossDomainMessenger`, `L1StandardBridge`, `L1Block`, `SystemConfig` 등)
- `//go:embed deploy-artifacts/*.json` 으로 바이너리에 포함
- Isthmus-capable `L1Block.sol` 아티팩트 포함 (방안 1 흡수)
- 배포 실행은 go-ethereum `bind.DeployContract()` 또는 raw tx — 런타임에 forge 불필요

### 바이너리 다운로드 실패 처리

forge build fallback 없음 — 다운로드 실패 시 명확한 에러 반환:
```
"failed to download tokamak-deployer v1.0.0: <reason>. Check network or retry."
```
trh-sdk에서 forge build 경로는 완전히 제거.

### 빌드 및 릴리즈

```
.goreleaser.yml
.github/workflows/
  release-deployer.yml   # v* 태그 push 시 트리거
    Steps:
      1. forge build (아티팩트 생성)
      2. deploy-artifacts/ 추출
      3. goreleaser build (아티팩트 embed, 바이너리 생성)
      4. GitHub Releases 업로드:
           tokamak-deployer-linux-amd64
           tokamak-deployer-darwin-arm64
           tokamak-deployer-darwin-amd64
```

### 버전 관리 및 업데이트 절차

- trh-sdk가 버전 상수 하나로 핀: `const TokamakDeployerVersion = "v1.0.0"` (`deployer_binary.go`)
- 바이너리 캐시 경로: `~/.trh/bin/tokamak-deployer-{version}`
- 버전 불일치 시 재다운로드

#### tokamak-thanos 코드 업데이트 시 절차

tokamak-thanos에 컨트랙트 변경 또는 배포 로직 변경이 생기면 두 단계를 거친다:

```
1. tokamak-thanos에서:
   - 변경 사항 merge → 새 태그 push (예: v1.1.0)
   - GitHub Actions가 새 바이너리 자동 빌드 + GitHub Releases 업로드

2. trh-sdk에서:
   - deployer_binary.go의 TokamakDeployerVersion = "v1.1.0" 으로 bump
   - trh-sdk 릴리즈
```

**버전 호환성 정책:**
- `tokamak-deployer`의 CLI 인터페이스(서브커맨드, 플래그)는 semver minor 변경까지 하위 호환 유지
- 컨트랙트 변경이 배포 결과물(genesis 포맷, 컨트랙트 주소 체계)에 영향을 주면 major 버전 bump
- trh-sdk는 major 버전이 바뀔 경우 반드시 통합 테스트 후 업데이트

**오래된 바이너리 캐시 처리:**
- `~/.trh/bin/` 에 이전 버전 바이너리가 남아있어도 덮어쓰지 않음 (버전별 파일명)
- 사용자가 수동으로 삭제하지 않아도 무방 (디스크 용량이 문제가 될 경우 `trh cache clean` 명령 추가 — scope 외)

---

## 컴포넌트 2: AA Paymaster 비동기화 (trh-sdk)

### 파일: `pkg/stacks/thanos/local_network.go`

**현재** (`deployLocalNetwork()`, ~L181-220):

```go
// 전체 배포를 최대 5분 블로킹
if err := bridgeAdminTONForAASetup(...); err != nil {
    return err
}
if err := setupAAPaymaster(...); err != nil {
    return err
}
startAltoBundler(...)
```

**변경 후:**

```go
// L2 core services 즉시 기동
if err := startLocalCoreServices(...); err != nil {
    return err
}

// AA setup은 백그라운드 goroutine — deployLocalNetwork() return을 블로킹하지 않음
if config.AAEnabled {
    go func() {
        if err := bridgeAdminTONForAASetup(...); err != nil {
            log.Error("AA bridge setup failed: %v", err)
            return
        }
        if err := setupAAPaymaster(...); err != nil {
            log.Error("AA paymaster setup failed: %v", err)
            return
        }
        if err := startAltoBundler(...); err != nil {
            log.Error("alto-bundler start failed: %v", err)
            return
        }
        log.Info("AA Paymaster ready")
    }()
}
```

### 동작 명세

| 시나리오 | 동작 |
|---------|------|
| AA 준비 전 사용자가 bundler 호출 | alto-bundler 미기동 → connection refused (AA 비활성화 시와 동일) |
| AA setup 실패 | 에러 로그 출력, 재시도 없음 (수동 재실행 필요 — scope 외) |
| AA setup 완료 알림 | 기존 log streaming으로 "AA Paymaster ready" 전달 |
| `deployLocalNetwork()` return 시점 | L2 core services 기동 완료 후 즉시 return — AA 대기 없음 |

---

## 컴포넌트 3: trh-sdk 통합 변경

### 새 파일: `pkg/stacks/thanos/deployer_binary.go`

```go
const TokamakDeployerVersion = "v1.0.0"

// 캐시 확인 후 GitHub Releases에서 없으면 다운로드
func ensureTokamakDeployer(cacheDir string) (binaryPath string, err error)

// tokamak-deployer deploy-contracts 서브커맨드 실행
func runDeployContracts(binaryPath string, opts DeployOpts) (*DeployOutput, error)

// tokamak-deployer generate-genesis 서브커맨드 실행
func runGenerateGenesis(binaryPath string, opts GenesisOpts) error
```

### `deploy_contracts.go` 변경

**제거:**
- `CloneRepo(tokamakThanosURL, ...)` 호출
- `downloadPrebuiltArtifacts()` 호출
- `L1BlockArtifactDir` 삭제 + cache invalidation 로직
- forge incremental build 블록 전체
- 직접 `forge script` 호출

**대체:**
```go
// Track B (기존: forge build ~27-51s → 변경: 바이너리 확보 ~0-5s)
binaryPath, err := ensureTokamakDeployer(cacheDir)

// 컨트랙트 배포
output, err := runDeployContracts(binaryPath, deployOpts)

// genesis 생성 (5단계 후처리 포함)
err = runGenerateGenesis(binaryPath, genesisOpts)
```

Track A (`cannon prestate build`, ~34.5s)는 변경 없음 — errgroup 병렬 실행 유지.

### 디렉토리 구조 변화

```
pkg/stacks/thanos/
  deploy_contracts.go      # 대폭 단순화: clone/build/forge 블록 제거
  deployer_binary.go       # 신규: 바이너리 다운로드 + 실행
  aa_bridge.go             # 변경 없음
  local_network.go         # AA setup goroutine 분리
```

---

## 테스트 전략

### tokamak-deployer 단위 테스트 (tokamak-thanos 레포, Anvil 사용)

| 시나리오 | 검증 항목 |
|---------|---------|
| `deploy-contracts` 정상 실행 | deploy-output.json 생성, 주요 컨트랙트 주소 non-zero, L1 Anvil에서 코드 존재 확인 |
| `generate-genesis` 정상 실행 | genesis.json 생성, `alloc` 에 L1Block bytecode 포함 (Isthmus-capable 버전), chainId 일치 |
| DRB / USDC / MultiTokenPaymaster inject | genesis alloc에 각 컨트랙트 bytecode + storage 포함 확인 |
| 잘못된 `--l1-rpc` 플래그 | 명확한 에러 메시지 + non-zero exit code |
| 잘못된 private key | 명확한 에러 메시지 + non-zero exit code |

### trh-sdk 통합 테스트

| 시나리오 | 검증 항목 |
|---------|---------|
| 바이너리 캐시 미스 (최초 실행) | `~/.trh/bin/tokamak-deployer-{version}` 파일 생성, 배포 정상 완료 |
| 바이너리 캐시 히트 (재실행) | 다운로드 없이 기존 바이너리 사용, 배포 정상 완료 |
| 바이너리 다운로드 실패 (네트워크 오류 시뮬레이션) | `"failed to download tokamak-deployer..."` 에러 반환, 배포 중단 |
| 버전 불일치 (캐시에 구버전 존재) | 새 버전 다운로드 후 배포 정상 완료 |
| L2 전체 배포 플로우 (AA 비활성화) | L2 core services 기동 후 `eth_blockNumber` RPC 응답 확인 |
| L2 전체 배포 플로우 (AA 활성화) | L2 core services 기동 즉시 확인, AA setup은 백그라운드 완료 후 alto-bundler 응답 확인 |

### AA Paymaster 비동기 시나리오

| 시나리오 | 검증 항목 |
|---------|---------|
| AA 활성화 배포 후 즉시 L2 RPC 호출 | L2 기동 완료 확인 (AA 대기 없이 return) |
| AA setup 진행 중 bundler 호출 | connection refused (에러지만 L2는 정상) |
| AA setup 완료 후 bundler 호출 | `eth_supportedEntryPoints` 응답 정상 |
| AA setup 실패 (브리지 타임아웃 시뮬레이션) | 에러 로그 출력, L2는 계속 정상 운영 |

---

## Scope 외

- AA setup 실패 시 재시도 / 자동 복구
- `reuseDeployment` / CREATE2 최적화 (L1 tx 수 감소 — 별도 최적화 방안)
- Testnet-AWS / Mainnet-AWS 배포 경로 (로컬 Docker만 해당)
- L1 컨트랙트 병렬 배포

---

## 구현 순서

1. **tokamak-thanos**: `cmd/tokamak-deployer/` 추가, goreleaser 설정, CI workflow 추가 → `v1.0.0` 태그
2. **trh-sdk**: `deployer_binary.go` 추가, `deploy_contracts.go` 단순화, `local_network.go` AA goroutine 분리
3. **검증**: 새 경로로 E2E live test — 로컬 L2 배포 전체 플로우 확인
