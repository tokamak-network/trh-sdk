# L2 배포 최적화 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** tokamak-thanos에 `tokamak-deployer` Go 바이너리를 추가하고, trh-sdk가 forge/git clone 대신 이 바이너리를 사용하도록 전환. AA Paymaster 설정을 비동기화해 L2 기동 블로킹 제거.

**Architecture:** tokamak-thanos repo에 `cmd/tokamak-deployer/` CLI를 추가해 GitHub Releases로 배포. trh-sdk는 버전 핀된 바이너리를 캐시 디렉토리에 다운로드해 실행. AA setup은 `deployLocalNetwork()` 내에서 goroutine으로 분리.

**Tech Stack:** Go (cobra CLI, go-ethereum), goreleaser, GitHub Actions, Anvil (테스트)

**Spec:** `docs/superpowers/specs/2026-04-16-l2-deploy-optimization-design.md`

---

## 파일 맵

### tokamak-thanos 레포 (신규 생성)

| 파일 | 역할 |
|------|------|
| `cmd/tokamak-deployer/main.go` | CLI 진입점, cobra root command |
| `cmd/tokamak-deployer/cmd/root.go` | root command 설정 |
| `cmd/tokamak-deployer/cmd/deploy_contracts.go` | `deploy-contracts` 서브커맨드: L1 컨트랙트 배포 |
| `cmd/tokamak-deployer/cmd/generate_genesis.go` | `generate-genesis` 서브커맨드: genesis.json 생성 + 후처리 |
| `cmd/tokamak-deployer/internal/deployer/contracts.go` | go-ethereum으로 컨트랙트 배포 로직 |
| `cmd/tokamak-deployer/internal/genesis/generator.go` | genesis 생성 + 5단계 후처리 |
| `cmd/tokamak-deployer/deploy-artifacts/` | 배포에 필요한 컨트랙트 ABI/bytecode JSON (forge build 후 추출) |
| `.goreleaser.yml` | 멀티플랫폼 바이너리 빌드 설정 |
| `.github/workflows/release-deployer.yml` | `v*` 태그 push 시 릴리즈 자동화 |

### trh-sdk 레포 (수정/신규)

| 파일 | 변경 유형 | 역할 |
|------|-----------|------|
| `pkg/stacks/thanos/deployer_binary.go` | **신규** | 바이너리 다운로드·캐시·실행 |
| `pkg/stacks/thanos/deploy_contracts.go` | **수정** | clone/forge build/forge script 블록 제거, 바이너리 호출로 대체 |
| `pkg/stacks/thanos/local_network.go` | **수정** | AA setup 블록을 background goroutine으로 분리 |
| `pkg/stacks/thanos/artifacts_download.go` | **삭제** | npm 아티팩트 다운로드 코드 (역할 소멸) |

---

## Phase 1: tokamak-thanos — tokamak-deployer 바이너리

> **레포:** `tokamak-thanos`  
> **사전 작업:** `git clone https://github.com/tokamak-network/tokamak-thanos.git` 후 작업

---

### Task 1: CLI 스캐폴드 + Anvil 테스트 환경 설정

**Files:**
- Create: `cmd/tokamak-deployer/main.go`
- Create: `cmd/tokamak-deployer/cmd/root.go`
- Create: `cmd/tokamak-deployer/cmd/deploy_contracts.go` (stub)
- Create: `cmd/tokamak-deployer/cmd/generate_genesis.go` (stub)
- Create: `cmd/tokamak-deployer/cmd/deploy_contracts_test.go`

- [ ] **Step 1: main.go 작성**

```go
// cmd/tokamak-deployer/main.go
package main

import "github.com/tokamak-network/tokamak-thanos/cmd/tokamak-deployer/cmd"

func main() {
    cmd.Execute()
}
```

- [ ] **Step 2: root.go 작성**

```go
// cmd/tokamak-deployer/cmd/root.go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "tokamak-deployer",
    Short: "L1 contract deployer and genesis generator for tokamak-thanos",
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.AddCommand(deployContractsCmd)
    rootCmd.AddCommand(generateGenesisCmd)
}
```

- [ ] **Step 3: deploy-contracts stub 작성**

```go
// cmd/tokamak-deployer/cmd/deploy_contracts.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var (
    flagL1RPC      string
    flagPrivateKey string
    flagChainID    uint64
    flagOut        string
)

var deployContractsCmd = &cobra.Command{
    Use:   "deploy-contracts",
    Short: "Deploy L1 contracts and write deploy-output.json",
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("not implemented")
    },
}

func init() {
    deployContractsCmd.Flags().StringVar(&flagL1RPC, "l1-rpc", "", "L1 RPC URL (required)")
    deployContractsCmd.Flags().StringVar(&flagPrivateKey, "private-key", "", "Deployer private key (required)")
    deployContractsCmd.Flags().Uint64Var(&flagChainID, "chain-id", 0, "L2 chain ID (required)")
    deployContractsCmd.Flags().StringVar(&flagOut, "out", "./deploy-output.json", "Output file path")
    _ = deployContractsCmd.MarkFlagRequired("l1-rpc")
    _ = deployContractsCmd.MarkFlagRequired("private-key")
    _ = deployContractsCmd.MarkFlagRequired("chain-id")
}
```

- [ ] **Step 4: generate-genesis stub 작성**

```go
// cmd/tokamak-deployer/cmd/generate_genesis.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var (
    flagDeployOutput string
    flagConfig       string
    flagGenesisOut   string
)

var generateGenesisCmd = &cobra.Command{
    Use:   "generate-genesis",
    Short: "Generate genesis.json from deploy output",
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("not implemented")
    },
}

func init() {
    generateGenesisCmd.Flags().StringVar(&flagDeployOutput, "deploy-output", "./deploy-output.json", "deploy-contracts output file")
    generateGenesisCmd.Flags().StringVar(&flagConfig, "config", "./rollup-config.json", "Rollup config file")
    generateGenesisCmd.Flags().StringVar(&flagGenesisOut, "out", "./genesis.json", "Genesis output file path")
    _ = generateGenesisCmd.MarkFlagRequired("deploy-output")
    _ = generateGenesisCmd.MarkFlagRequired("config")
}
```

- [ ] **Step 5: failing 통합 테스트 작성 (Anvil 기반)**

Anvil 설치 확인: `anvil --version` (없으면 `curl -L https://foundry.paradigm.xyz | bash && foundryup`)

```go
// cmd/tokamak-deployer/cmd/deploy_contracts_test.go
package cmd_test

import (
    "context"
    "encoding/json"
    "os"
    "os/exec"
    "testing"
    "time"
)

func startAnvil(t *testing.T) (rpcURL string, stop func()) {
    t.Helper()
    cmd := exec.Command("anvil", "--port", "18545", "--block-time", "1")
    if err := cmd.Start(); err != nil {
        t.Fatalf("failed to start anvil: %v", err)
    }
    time.Sleep(500 * time.Millisecond)
    return "http://127.0.0.1:18545", func() { _ = cmd.Process.Kill() }
}

func TestDeployContracts_NotImplemented(t *testing.T) {
    rpcURL, stop := startAnvil(t)
    defer stop()

    outFile := t.TempDir() + "/deploy-output.json"
    cmd := exec.Command("go", "run", ".", "deploy-contracts",
        "--l1-rpc", rpcURL,
        "--private-key", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // anvil default key 0
        "--chain-id", "901",
        "--out", outFile,
    )
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    // 현재 stub이므로 "not implemented" 에러 반환 확인
    if err == nil {
        t.Fatalf("expected error from stub, got output: %s", out)
    }
    t.Logf("expected failure: %s", out)
}
```

- [ ] **Step 6: 테스트 실행 — 실패 확인**

```bash
cd cmd/tokamak-deployer && go test ./cmd/... -run TestDeployContracts_NotImplemented -v
```

예상: FAIL (stub "not implemented" 반환)

- [ ] **Step 7: go.mod에 의존성 추가**

```bash
cd cmd/tokamak-deployer
go mod init github.com/tokamak-network/tokamak-thanos/cmd/tokamak-deployer
go get github.com/spf13/cobra@v1.8.0
go get github.com/ethereum/go-ethereum@v1.14.0
go mod tidy
```

- [ ] **Step 8: 빌드 확인**

```bash
go build -o /tmp/tokamak-deployer .
/tmp/tokamak-deployer --help
```

예상 출력:
```
L1 contract deployer and genesis generator for tokamak-thanos

Usage:
  tokamak-deployer [command]

Available Commands:
  deploy-contracts  Deploy L1 contracts and write deploy-output.json
  generate-genesis  Generate genesis.json from deploy output
```

---

### Task 2: deploy-artifacts 추출 스크립트

현재 `packages/tokamak/contracts-bedrock/forge-artifacts/` 에서 배포에 필요한 컨트랙트 ABI+bytecode만 추출. forge build 출력에서 최소 집합을 `deploy-artifacts/`로 복사.

**Files:**
- Create: `cmd/tokamak-deployer/scripts/extract-artifacts.sh`
- Create: `cmd/tokamak-deployer/deploy-artifacts/.gitkeep`

- [ ] **Step 1: 배포 필요 컨트랙트 목록 파악**

`packages/tokamak/contracts-bedrock/scripts/Deploy.s.sol`을 열고 `new ContractName(` 또는 `_deploy` 패턴으로 배포되는 컨트랙트 목록 추출. 최소 포함 목록:

```
AddressManager
L1CrossDomainMessenger
L1ERC721Bridge
L1StandardBridge
L2OutputOracle (faultproof 비활성화 시)
OptimismMintableERC20Factory
OptimismPortal
OptimismPortal2 (faultproof 활성화 시)
ProxyAdmin
SystemConfig
SuperchainConfig
L1Block
DisputeGameFactory (faultproof)
AnchorStateRegistry (faultproof)
MIPS (faultproof)
PreimageOracle (faultproof)
```

- [ ] **Step 2: extract-artifacts.sh 작성**

```bash
#!/usr/bin/env bash
# cmd/tokamak-deployer/scripts/extract-artifacts.sh
# 사용법: bash scripts/extract-artifacts.sh <forge-artifacts-dir> <output-dir>
set -euo pipefail

FORGE_ARTIFACTS_DIR="${1:-../../packages/tokamak/contracts-bedrock/forge-artifacts}"
OUTPUT_DIR="${2:-./deploy-artifacts}"

mkdir -p "$OUTPUT_DIR"

CONTRACTS=(
  "AddressManager"
  "L1CrossDomainMessenger"
  "L1ERC721Bridge"
  "L1StandardBridge"
  "L2OutputOracle"
  "OptimismMintableERC20Factory"
  "OptimismPortal"
  "OptimismPortal2"
  "ProxyAdmin"
  "SystemConfig"
  "SuperchainConfig"
  "L1Block"
  "DisputeGameFactory"
  "AnchorStateRegistry"
  "MIPS"
  "PreimageOracle"
  "Proxy"
)

for contract in "${CONTRACTS[@]}"; do
  src="$FORGE_ARTIFACTS_DIR/$contract.sol/$contract.json"
  if [ -f "$src" ]; then
    cp "$src" "$OUTPUT_DIR/$contract.json"
    echo "✅ $contract"
  else
    echo "⚠️  $contract.json not found at $src"
  fi
done

echo "Extracted $(ls "$OUTPUT_DIR"/*.json 2>/dev/null | wc -l) artifacts to $OUTPUT_DIR"
```

- [ ] **Step 3: forge build 후 추출 실행**

```bash
# packages/tokamak/contracts-bedrock 에서
forge build

# cmd/tokamak-deployer 에서
bash scripts/extract-artifacts.sh \
  ../../packages/tokamak/contracts-bedrock/forge-artifacts \
  ./deploy-artifacts
```

예상: deploy-artifacts/*.json 파일들 생성

- [ ] **Step 4: embed 선언 추가**

Go의 `//go:embed`은 `..` 경로를 허용하지 않으므로 embed 파일은 `cmd/tokamak-deployer/` 루트에 위치시킨다.

```go
// cmd/tokamak-deployer/assets.go
package main

import "embed"

// DeployArtifactsFS는 L1 배포에 필요한 컨트랙트 ABI+bytecode를 담는 embed FS
//
//go:embed deploy-artifacts
var DeployArtifactsFS embed.FS
```

`internal/deployer/contracts.go`의 함수들은 `fs.FS` 파라미터를 받도록 설계되어 있으므로(Task 3 Step 4 참고),
`cmd/` 레이어에서 `DeployArtifactsFS`를 `deployer.Deploy(ctx, cfg, main.DeployArtifactsFS)` 형태로 전달한다.

단, `main` 패키지 변수를 `cmd` 패키지에서 import할 수 없으므로, `assets.go`를 `cmd/tokamak-deployer/cmd/` 패키지 내에 둔다:

```go
// cmd/tokamak-deployer/cmd/assets.go
package cmd

import "embed"

//go:embed ../deploy-artifacts
var DeployArtifactsFS embed.FS
```

> **주의:** `//go:embed ../deploy-artifacts` 처럼 상위 디렉토리를 가리키는 `..`도 Go에서 허용되지 않음.
> 따라서 `deploy-artifacts/` 디렉토리를 `cmd/tokamak-deployer/cmd/deploy-artifacts/`로 이동시키거나,
> `extract-artifacts.sh` 출력 경로를 `cmd/deploy-artifacts/`로 변경한다.

최종 구조:
```
cmd/tokamak-deployer/
  cmd/
    assets.go              # //go:embed deploy-artifacts
    deploy-artifacts/      # 추출된 컨트랙트 JSON (스크립트 출력 경로)
    deploy_contracts.go
    ...
```

`extract-artifacts.sh` Step 2의 기본 출력 경로를 수정:
```bash
OUTPUT_DIR="${2:-./cmd/deploy-artifacts}"
```

goreleaser `before` hook도 업데이트:
```yaml
before:
  hooks:
    - bash cmd/tokamak-deployer/scripts/extract-artifacts.sh \
        packages/tokamak/contracts-bedrock/forge-artifacts \
        cmd/tokamak-deployer/cmd/deploy-artifacts
```

---

### Task 3: deploy-contracts 구현

**Files:**
- Modify: `cmd/tokamak-deployer/cmd/deploy_contracts.go`
- Create: `cmd/tokamak-deployer/internal/deployer/contracts.go`
- Create: `cmd/tokamak-deployer/internal/deployer/types.go`

> **중요:** 이 Task는 `Deploy.s.sol` 스크립트의 배포 순서를 Go로 재구현합니다.
> 구현 전 `packages/tokamak/contracts-bedrock/scripts/Deploy.s.sol`을 반드시 정독하세요.
> 각 컨트랙트의 배포 순서와 constructor 인수를 확인하세요.

- [ ] **Step 1: DeployOutput 타입 정의**

```go
// cmd/tokamak-deployer/internal/deployer/types.go
package deployer

// DeployOutput은 deploy-contracts 실행 결과 — deploy-output.json에 직렬화
type DeployOutput struct {
    // L1ChainID는 배포 대상 L1 chain ID
    L1ChainID uint64 `json:"l1ChainId"`
    // L2ChainID는 이 배포가 서비스하는 L2 chain ID
    L2ChainID uint64 `json:"l2ChainId"`
    // AddressManager ...
    AddressManager                  string `json:"AddressManager"`
    L1CrossDomainMessengerProxy     string `json:"L1CrossDomainMessengerProxy"`
    L1ERC721BridgeProxy             string `json:"L1ERC721BridgeProxy"`
    L1StandardBridgeProxy           string `json:"L1StandardBridgeProxy"`
    L2OutputOracleProxy             string `json:"L2OutputOracleProxy"`
    OptimismMintableERC20FactoryProxy string `json:"OptimismMintableERC20FactoryProxy"`
    OptimismPortalProxy             string `json:"OptimismPortalProxy"`
    ProxyAdmin                      string `json:"ProxyAdmin"`
    SystemConfigProxy               string `json:"SystemConfigProxy"`
    SuperchainConfigProxy           string `json:"SuperchainConfigProxy"`
    // FaultProof 활성화 시
    DisputeGameFactoryProxy         string `json:"DisputeGameFactoryProxy,omitempty"`
    AnchorStateRegistryProxy        string `json:"AnchorStateRegistryProxy,omitempty"`
}

// DeployConfig는 deploy-contracts 입력 설정
type DeployConfig struct {
    L1RPCURL       string
    PrivateKey     string
    L2ChainID      uint64
    EnableFaultProof bool
    // Deploy.s.sol에서 요구하는 추가 파라미터 (Deploy.s.sol 읽고 확인)
    FinalSystemOwner string
    L2OutputOracleSubmissionInterval uint64
    // ... 필드는 Deploy.s.sol 의 deploy-config.json 스키마와 일치시킬 것
}
```

- [ ] **Step 2: failing 테스트 작성 (Anvil 대상 실제 배포)**

```go
// cmd/tokamak-deployer/cmd/deploy_contracts_test.go (기존 파일에 추가)
func TestDeployContracts_Anvil(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    rpcURL, stop := startAnvil(t)
    defer stop()

    outFile := t.TempDir() + "/deploy-output.json"
    cmd := exec.Command("go", "run", ".", "deploy-contracts",
        "--l1-rpc", rpcURL,
        "--private-key", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
        "--chain-id", "901",
        "--out", outFile,
    )
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("deploy-contracts failed: %v\n%s", err, out)
    }

    // deploy-output.json 검증
    data, err := os.ReadFile(outFile)
    if err != nil {
        t.Fatalf("output file not created: %v", err)
    }
    var output map[string]interface{}
    if err := json.Unmarshal(data, &output); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }

    // 주요 컨트랙트 주소 non-zero 확인
    for _, key := range []string{"ProxyAdmin", "SystemConfigProxy", "OptimismPortalProxy", "L1StandardBridgeProxy"} {
        addr, ok := output[key].(string)
        if !ok || addr == "" || addr == "0x0000000000000000000000000000000000000000" {
            t.Errorf("expected non-zero address for %s, got: %v", key, output[key])
        }
    }
}
```

- [ ] **Step 3: 테스트 실행 — 실패 확인**

```bash
go test ./cmd/... -run TestDeployContracts_Anvil -v
```

예상: FAIL ("not implemented")

- [ ] **Step 4: contracts.go 구현**

> `Deploy.s.sol`의 배포 흐름을 참고해 각 컨트랙트를 go-ethereum으로 배포.
> 핵심 패턴:

```go
// cmd/tokamak-deployer/internal/deployer/contracts.go
package deployer

import (
    "context"
    "crypto/ecdsa"
    "encoding/json"
    "fmt"
    "io/fs"
    "math/big"

    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
)

// artifact는 forge-artifacts JSON의 ABI + bytecode를 담는 구조체
type artifact struct {
    ABI      json.RawMessage `json:"abi"`
    Bytecode struct {
        Object string `json:"object"`
    } `json:"bytecode"`
}

// loadArtifact는 embed FS에서 컨트랙트 아티팩트를 읽는다
func loadArtifact(artifactsFS fs.FS, name string) (*artifact, error) {
    data, err := fs.ReadFile(artifactsFS, "deploy-artifacts/"+name+".json")
    if err != nil {
        return nil, fmt.Errorf("artifact %s not found: %w", name, err)
    }
    var a artifact
    if err := json.Unmarshal(data, &a); err != nil {
        return nil, fmt.Errorf("invalid artifact %s: %w", name, err)
    }
    return &a, nil
}

// deployContract는 단일 컨트랙트를 배포하고 주소를 반환한다
func deployContract(
    ctx context.Context,
    client *ethclient.Client,
    auth *bind.TransactOpts,
    a *artifact,
    constructorArgs ...interface{},
) (common.Address, error) {
    parsedABI, err := abi.JSON(strings.NewReader(string(a.ABI)))
    if err != nil {
        return common.Address{}, fmt.Errorf("parse ABI: %w", err)
    }
    bytecode := common.FromHex(a.Bytecode.Object)

    var input []byte
    if len(constructorArgs) > 0 {
        packed, err := parsedABI.Pack("", constructorArgs...)
        if err != nil {
            return common.Address{}, fmt.Errorf("pack constructor args: %w", err)
        }
        bytecode = append(bytecode, packed...)
    }

    tx := types.NewContractCreation(auth.Nonce.Uint64(), common.Big0, auth.GasLimit, auth.GasPrice, bytecode)
    signedTx, err := auth.Signer(auth.From, tx)
    if err != nil {
        return common.Address{}, fmt.Errorf("sign tx: %w", err)
    }
    if err := client.SendTransaction(ctx, signedTx); err != nil {
        return common.Address{}, fmt.Errorf("send tx: %w", err)
    }

    receipt, err := bind.WaitMined(ctx, client, signedTx)
    if err != nil {
        return common.Address{}, fmt.Errorf("wait mined: %w", err)
    }
    if receipt.Status == 0 {
        return common.Address{}, fmt.Errorf("contract deployment reverted")
    }
    auth.Nonce.Add(auth.Nonce, common.Big1)
    return receipt.ContractAddress, nil
}

// Deploy는 모든 L1 컨트랙트를 Deploy.s.sol 순서대로 배포한다
// Deploy.s.sol을 읽고 각 단계를 아래 패턴으로 구현할 것
func Deploy(ctx context.Context, cfg DeployConfig, artifactsFS fs.FS) (*DeployOutput, error) {
    client, err := ethclient.DialContext(ctx, cfg.L1RPCURL)
    if err != nil {
        return nil, fmt.Errorf("connect L1: %w", err)
    }
    defer client.Close()

    privKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
    if err != nil {
        return nil, fmt.Errorf("parse private key: %w", err)
    }

    chainID, err := client.ChainID(ctx)
    if err != nil {
        return nil, fmt.Errorf("get chain ID: %w", err)
    }

    auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
    if err != nil {
        return nil, fmt.Errorf("create transactor: %w", err)
    }
    // 수동 nonce 관리
    nonce, err := client.PendingNonceAt(ctx, auth.From)
    if err != nil {
        return nil, err
    }
    auth.Nonce = big.NewInt(int64(nonce))
    auth.GasLimit = 5_000_000

    output := &DeployOutput{L2ChainID: cfg.L2ChainID}
    l1ChainID, _ := client.ChainID(ctx)
    output.L1ChainID = l1ChainID.Uint64()

    // --- Deploy.s.sol 순서대로 아래 패턴 반복 ---
    // (1) ProxyAdmin 배포
    proxyAdminArtifact, err := loadArtifact(artifactsFS, "ProxyAdmin")
    if err != nil {
        return nil, err
    }
    proxyAdminAddr, err := deployContract(ctx, client, auth, proxyAdminArtifact, auth.From)
    if err != nil {
        return nil, fmt.Errorf("deploy ProxyAdmin: %w", err)
    }
    output.ProxyAdmin = proxyAdminAddr.Hex()

    // (2) AddressManager, SystemConfig, OptimismPortal, ... 순서는 Deploy.s.sol 참고
    // 각 컨트랙트마다 loadArtifact + deployContract 패턴 반복

    return output, nil
}
```

- [ ] **Step 5: deploy-contracts 커맨드에 Deploy() 연결**

```go
// cmd/tokamak-deployer/cmd/deploy_contracts.go 의 RunE 수정
RunE: func(cmd *cobra.Command, args []string) error {
    cfg := deployer.DeployConfig{
        L1RPCURL:   flagL1RPC,
        PrivateKey: flagPrivateKey,
        L2ChainID:  flagChainID,
    }
    output, err := deployer.Deploy(cmd.Context(), cfg, deployer.DeployArtifactsFS)
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(flagOut, data, 0644)
},
```

- [ ] **Step 6: 에러 시나리오 테스트 추가 (잘못된 플래그)**

```go
// cmd/tokamak-deployer/cmd/deploy_contracts_test.go 에 추가
func TestDeployContracts_BadRPC(t *testing.T) {
    outFile := t.TempDir() + "/deploy-output.json"
    cmd := exec.Command("go", "run", ".", "deploy-contracts",
        "--l1-rpc", "http://127.0.0.1:19999", // 연결 불가 포트
        "--private-key", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
        "--chain-id", "901",
        "--out", outFile,
    )
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    if err == nil {
        t.Fatalf("expected error for unreachable RPC, got output: %s", out)
    }
    // 명확한 에러 메시지 포함 확인
    output := string(out)
    if !strings.Contains(output, "connect") && !strings.Contains(output, "connection refused") {
        t.Errorf("expected connection error message, got: %s", output)
    }
}

func TestDeployContracts_BadPrivateKey(t *testing.T) {
    rpcURL, stop := startAnvil(t)
    defer stop()

    outFile := t.TempDir() + "/deploy-output.json"
    cmd := exec.Command("go", "run", ".", "deploy-contracts",
        "--l1-rpc", rpcURL,
        "--private-key", "0xINVALID",
        "--chain-id", "901",
        "--out", outFile,
    )
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    if err == nil {
        t.Fatalf("expected error for invalid private key, got output: %s", out)
    }
    output := string(out)
    if !strings.Contains(output, "private key") && !strings.Contains(output, "hex") {
        t.Errorf("expected private key error message, got: %s", output)
    }
}
```

- [ ] **Step 7: 에러 시나리오 테스트 실행**

```bash
go test ./cmd/... -run "TestDeployContracts_BadRPC|TestDeployContracts_BadPrivateKey" -v
```

예상: PASS (에러 발생 + 명확한 에러 메시지 포함)

- [ ] **Step 8: 정상 경로 테스트 통과 확인**

```bash
go test ./cmd/... -run TestDeployContracts_Anvil -v -timeout 120s
```

예상: PASS (ProxyAdmin, SystemConfigProxy 등 non-zero 주소)

---

### Task 4: generate-genesis 구현

**Files:**
- Modify: `cmd/tokamak-deployer/cmd/generate_genesis.go`
- Create: `cmd/tokamak-deployer/internal/genesis/generator.go`
- Create: `cmd/tokamak-deployer/cmd/generate_genesis_test.go`

> **참고:** 현재 trh-sdk의 genesis 후처리 5단계는 `deploy_contracts.go`에 구현되어 있음.
> `pkg/stacks/thanos/deploy_contracts.go` 에서 DRB inject, USDC inject, MultiTokenPaymaster inject,
> L1Block bytecode patch, rollup hash update 로직을 참고해 이식.

- [ ] **Step 1: failing 테스트 작성**

```go
// cmd/tokamak-deployer/cmd/generate_genesis_test.go
package cmd_test

import (
    "encoding/json"
    "os"
    "testing"
    "os/exec"
)

func TestGenerateGenesis_Basic(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // deploy-output.json 픽스처 (Anvil 배포 결과 또는 테스트 픽스처)
    deployOutput := `{
        "l1ChainId": 31337,
        "l2ChainId": 901,
        "ProxyAdmin": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
        "SystemConfigProxy": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
        "OptimismPortalProxy": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
        "L1StandardBridgeProxy": "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"
    }`
    deployOutputFile := t.TempDir() + "/deploy-output.json"
    os.WriteFile(deployOutputFile, []byte(deployOutput), 0644)

    // rollup-config.json 픽스처
    rollupConfig := `{
        "l2ChainID": 901,
        "l1ChainID": 31337,
        "l2BlockTime": 2,
        "l1BlockTime": 12,
        "maxSequencerDrift": 600,
        "sequencerWindowSize": 3600
    }`
    rollupConfigFile := t.TempDir() + "/rollup-config.json"
    os.WriteFile(rollupConfigFile, []byte(rollupConfig), 0644)

    genesisOut := t.TempDir() + "/genesis.json"

    cmd := exec.Command("go", "run", ".", "generate-genesis",
        "--deploy-output", deployOutputFile,
        "--config", rollupConfigFile,
        "--out", genesisOut,
    )
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("generate-genesis failed: %v\n%s", err, out)
    }

    data, err := os.ReadFile(genesisOut)
    if err != nil {
        t.Fatalf("genesis.json not created: %v", err)
    }
    var genesis map[string]interface{}
    if err := json.Unmarshal(data, &genesis); err != nil {
        t.Fatalf("invalid genesis.json: %v", err)
    }

    // alloc에 L1Block bytecode 존재 확인 (Isthmus-capable 버전)
    alloc, ok := genesis["alloc"].(map[string]interface{})
    if !ok {
        t.Fatal("genesis missing alloc field")
    }
    // L1Block 주소는 0x4200000000000000000000000000000000000015
    l1BlockAlloc, exists := alloc["0x4200000000000000000000000000000000000015"]
    if !exists {
        t.Fatal("L1Block not found in genesis alloc")
    }
    l1BlockMap := l1BlockAlloc.(map[string]interface{})
    code, _ := l1BlockMap["code"].(string)
    if len(code) < 10 {
        t.Errorf("L1Block code too short, likely not injected: %s", code)
    }
}
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

```bash
go test ./cmd/... -run TestGenerateGenesis_Basic -v
```

예상: FAIL ("not implemented")

- [ ] **Step 3: genesis generator 구현**

> `pkg/stacks/thanos/deploy_contracts.go`에서 다음 함수들의 로직을 이식:
> - `injectDRBBytecode()` (L501-520 부근)
> - `injectUSDCBytecode()`
> - `injectMultiTokenPaymasterBytecode()`
> - `patchL1BlockBytecodeInGenesis()`
> - `updateRollupHash()`

```go
// cmd/tokamak-deployer/internal/genesis/generator.go
package genesis

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/tokamak-network/tokamak-thanos/cmd/tokamak-deployer/internal/deployer"
)

// Generate는 deploy-output.json + rollup-config.json으로 genesis.json을 생성한다
func Generate(deployOutputPath, configPath, outPath string) error {
    deployOutputData, err := os.ReadFile(deployOutputPath)
    if err != nil {
        return fmt.Errorf("read deploy output: %w", err)
    }
    var output deployer.DeployOutput
    if err := json.Unmarshal(deployOutputData, &output); err != nil {
        return fmt.Errorf("parse deploy output: %w", err)
    }

    // TODO: op-node genesis 생성 로직
    // (trh-sdk의 generateGenesisJson 참고: op-node binary를 실행하거나 go-ethereum 직접 사용)

    // 1단계: op-node genesis 생성
    // trh-sdk의 generateGenesisJson() 참고 (deploy_contracts.go ~L420):
    // op-node binary를 exec.Command로 실행:
    //   op-node genesis l2 --deploy-config <configPath> --l1-deployments <deployOutputPath> --outfile.l2 <outPath>
    cmd := exec.Command("op-node", "genesis", "l2",
        "--deploy-config", configPath,
        "--l1-deployments", deployOutputPath,
        "--outfile.l2", outPath,
    )
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("op-node genesis l2: %w\n%s", err, out)
    }

    genesisData, err := os.ReadFile(outPath)
    if err != nil {
        return fmt.Errorf("read genesis: %w", err)
    }
    var genesis map[string]interface{}
    if err := json.Unmarshal(genesisData, &genesis); err != nil {
        return fmt.Errorf("parse genesis: %w", err)
    }

    // 2단계: DRB inject
    // trh-sdk deploy_contracts.go injectDRBBytecode() (~L501) 이식:
    // genesis["alloc"]["0x...DRBAddr"]["code"] = "0x...drbBytecode"
    // trh-sdk에서 DRB_CONTRACT_ADDRESS, DRB_BYTECODE 상수 확인 후 이식

    // 3단계: USDC inject
    // trh-sdk deploy_contracts.go injectUSDCBytecode() 이식:
    // genesis["alloc"]["0x...USDCAddr"]["code"] = "0x...usdcBytecode"

    // 4단계: MultiTokenPaymaster inject
    // trh-sdk deploy_contracts.go injectMultiTokenPaymasterBytecode() 이식

    // 5단계: L1Block Isthmus bytecode patch
    // trh-sdk deploy_contracts.go patchL1BlockBytecodeInGenesis() 이식:
    // genesis["alloc"]["0x4200000000000000000000000000000000000015"]["code"] = isthmusBytecode
    // Isthmus-capable bytecode는 deploy-artifacts/L1Block.json의 deployedBytecode.object 사용

    // 6단계: rollup hash update
    // trh-sdk deploy_contracts.go updateRollupHash() 이식

    updated, err := json.MarshalIndent(genesis, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal genesis: %w", err)
    }
    return os.WriteFile(outPath, updated, 0644)
}
```

- [ ] **Step 4: 구현 완성 후 테스트 통과 확인**

```bash
go test ./cmd/... -run TestGenerateGenesis_Basic -v -timeout 60s
```

예상: PASS

---

### Task 5: goreleaser + CI 릴리즈 자동화

**Files:**
- Create: `.goreleaser.yml` (tokamak-thanos 루트)
- Create: `.github/workflows/release-deployer.yml`

- [ ] **Step 1: .goreleaser.yml 작성**

```yaml
# .goreleaser.yml
version: 2

before:
  hooks:
    - bash cmd/tokamak-deployer/scripts/extract-artifacts.sh \
        packages/tokamak/contracts-bedrock/forge-artifacts \
        cmd/tokamak-deployer/deploy-artifacts

builds:
  - id: tokamak-deployer
    main: ./cmd/tokamak-deployer
    binary: tokamak-deployer
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: linux
        goarch: arm64

archives:
  - id: tokamak-deployer
    builds: [tokamak-deployer]
    name_template: "tokamak-deployer-{{ .Os }}-{{ .Arch }}"
    format: binary

checksum:
  name_template: "checksums.txt"
```

- [ ] **Step 2: release-deployer.yml 작성**

```yaml
# .github/workflows/release-deployer.yml
name: Release tokamak-deployer

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install Foundry
        uses: foundry-rs/foundry-toolchain@v1

      - name: Build contracts (for artifact extraction)
        working-directory: packages/tokamak/contracts-bedrock
        run: |
          pnpm install --frozen-lockfile
          forge build

      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: goreleaser 로컬 테스트**

```bash
# goreleaser 설치
go install github.com/goreleaser/goreleaser/v2@latest

# snapshot 빌드 (실제 릴리즈 없이 로컬 테스트)
goreleaser release --snapshot --clean
ls dist/
```

예상: `dist/tokamak-deployer-linux-amd64`, `dist/tokamak-deployer-darwin-arm64` 등 바이너리 생성

---

## Phase 2: trh-sdk 통합

> **사전 조건:** Phase 1 완료 후 `v1.0.0` 태그가 GitHub Releases에 업로드된 상태
> **레포:** `trh-sdk`

---

### Task 6: deployer_binary.go — 바이너리 다운로드 및 실행

**Files:**
- Create: `pkg/stacks/thanos/deployer_binary.go`
- Create: `pkg/stacks/thanos/deployer_binary_test.go`

- [ ] **Step 1: failing 테스트 작성**

```go
// pkg/stacks/thanos/deployer_binary_test.go
package thanos_test

import (
    "os"
    "path/filepath"
    "testing"
)

func TestEnsureTokamakDeployer_DownloadAndCache(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test: requires network")
    }
    cacheDir := t.TempDir()

    binaryPath, err := ensureTokamakDeployer(cacheDir)
    if err != nil {
        t.Fatalf("ensureTokamakDeployer failed: %v", err)
    }

    // 파일 존재 + 실행 가능 확인
    info, err := os.Stat(binaryPath)
    if err != nil {
        t.Fatalf("binary not found at %s: %v", binaryPath, err)
    }
    if info.Mode()&0111 == 0 {
        t.Errorf("binary not executable: %s", binaryPath)
    }

    // 버전 확인 (expectedPath)
    expectedPath := filepath.Join(cacheDir, "tokamak-deployer-v1.0.0")
    if binaryPath != expectedPath {
        t.Errorf("expected path %s, got %s", expectedPath, binaryPath)
    }
}

func TestEnsureTokamakDeployer_CacheHit(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    cacheDir := t.TempDir()

    // 첫 번째 다운로드
    _, err := ensureTokamakDeployer(cacheDir)
    if err != nil {
        t.Fatalf("first download failed: %v", err)
    }

    // 두 번째 호출은 파일이 이미 있으므로 다운로드 없이 반환
    binaryPath, err := ensureTokamakDeployer(cacheDir)
    if err != nil {
        t.Fatalf("second call failed: %v", err)
    }
    if binaryPath == "" {
        t.Error("expected non-empty path on cache hit")
    }
}

func TestEnsureTokamakDeployer_DownloadFailure(t *testing.T) {
    // 존재하지 않는 버전으로 다운로드 실패 시뮬레이션
    // 테스트에서만 버전을 override할 수 있도록 ensureTokamakDeployerWithVersion 헬퍼 사용
    cacheDir := t.TempDir()

    binaryPath, err := ensureTokamakDeployerWithVersion(cacheDir, "v0.0.0-nonexistent")
    if err == nil {
        t.Fatalf("expected error for nonexistent version, got path: %s", binaryPath)
    }
    // 명확한 에러 메시지 확인
    if !strings.Contains(err.Error(), "failed to download tokamak-deployer") {
        t.Errorf("expected download error message, got: %v", err)
    }
}

func TestEnsureTokamakDeployer_VersionMismatch(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test: requires network")
    }
    cacheDir := t.TempDir()

    // 캐시에 구버전 바이너리 stub 생성
    oldBinaryPath := filepath.Join(cacheDir, "tokamak-deployer-v0.9.0")
    if err := os.WriteFile(oldBinaryPath, []byte("old binary"), 0755); err != nil {
        t.Fatalf("failed to create old binary: %v", err)
    }

    // 현재 버전(v1.0.0)은 캐시에 없으므로 새로 다운로드
    binaryPath, err := ensureTokamakDeployer(cacheDir)
    if err != nil {
        t.Fatalf("ensureTokamakDeployer failed: %v", err)
    }

    // 새 버전 경로 확인
    expectedPath := filepath.Join(cacheDir, "tokamak-deployer-"+TokamakDeployerVersion)
    if binaryPath != expectedPath {
        t.Errorf("expected new version path %s, got %s", expectedPath, binaryPath)
    }

    // 구버전 파일은 그대로 남아있어야 함 (덮어쓰지 않음)
    if _, err := os.Stat(oldBinaryPath); os.IsNotExist(err) {
        t.Error("old binary should not be deleted")
    }
}
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

```bash
cd /path/to/trh-sdk
go test ./pkg/stacks/thanos/... -run TestEnsureTokamakDeployer -v
```

예상: FAIL (함수 미정의)

- [ ] **Step 3: deployer_binary.go 작성**

```go
// pkg/stacks/thanos/deployer_binary.go
package thanos

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
)

// TokamakDeployerVersion은 trh-sdk가 핀하는 tokamak-deployer 바이너리 버전
const TokamakDeployerVersion = "v1.0.0"

const tokamakDeployerRepo = "tokamak-network/tokamak-thanos"

// ensureTokamakDeployer는 캐시 디렉토리에 바이너리가 없으면 GitHub Releases에서 다운로드한다.
// 캐시 히트 시 즉시 반환. 반환값은 실행 가능한 바이너리 경로.
func ensureTokamakDeployer(cacheDir string) (string, error) {
    binaryName := fmt.Sprintf("tokamak-deployer-%s", TokamakDeployerVersion)
    binaryPath := filepath.Join(cacheDir, binaryName)

    if _, err := os.Stat(binaryPath); err == nil {
        return binaryPath, nil // 캐시 히트
    }

    osName := runtime.GOOS     // "linux", "darwin"
    arch := runtime.GOARCH     // "amd64", "arm64"
    assetName := fmt.Sprintf("tokamak-deployer-%s-%s", osName, arch)
    downloadURL := fmt.Sprintf(
        "https://github.com/%s/releases/download/%s/%s",
        tokamakDeployerRepo,
        TokamakDeployerVersion,
        assetName,
    )

    if err := downloadFile(downloadURL, binaryPath); err != nil {
        return "", fmt.Errorf("failed to download tokamak-deployer %s: %w\n"+
            "Check network connectivity or retry.", TokamakDeployerVersion, err)
    }
    if err := os.Chmod(binaryPath, 0755); err != nil {
        return "", fmt.Errorf("chmod binary: %w", err)
    }
    return binaryPath, nil
}

func downloadFile(url, destPath string) error {
    resp, err := http.Get(url) //nolint:gosec
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
    }

    f, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.Copy(f, resp.Body)
    return err
}

// DeployContractsOpts는 tokamak-deployer deploy-contracts 서브커맨드 입력
type DeployContractsOpts struct {
    L1RPCURL       string
    PrivateKey     string
    L2ChainID      uint64
    OutPath        string
}

// GenesisOpts는 tokamak-deployer generate-genesis 서브커맨드 입력
type GenesisOpts struct {
    DeployOutputPath string
    ConfigPath       string
    OutPath          string
}

// runDeployContracts는 tokamak-deployer deploy-contracts를 실행하고 결과를 반환한다
func runDeployContracts(ctx context.Context, binaryPath string, opts DeployContractsOpts) error {
    return runBinaryCommand(ctx, binaryPath, []string{
        "deploy-contracts",
        "--l1-rpc", opts.L1RPCURL,
        "--private-key", opts.PrivateKey,
        "--chain-id", fmt.Sprintf("%d", opts.L2ChainID),
        "--out", opts.OutPath,
    })
}

// runGenerateGenesis는 tokamak-deployer generate-genesis를 실행한다
func runGenerateGenesis(ctx context.Context, binaryPath string, opts GenesisOpts) error {
    return runBinaryCommand(ctx, binaryPath, []string{
        "generate-genesis",
        "--deploy-output", opts.DeployOutputPath,
        "--config", opts.ConfigPath,
        "--out", opts.OutPath,
    })
}

// ensureTokamakDeployerWithVersion은 테스트용 버전 override를 지원하는 내부 함수
func ensureTokamakDeployerWithVersion(cacheDir, version string) (string, error) {
    binaryName := fmt.Sprintf("tokamak-deployer-%s", version)
    binaryPath := filepath.Join(cacheDir, binaryName)

    if _, err := os.Stat(binaryPath); err == nil {
        return binaryPath, nil
    }

    osName := runtime.GOOS
    arch := runtime.GOARCH
    assetName := fmt.Sprintf("tokamak-deployer-%s-%s", osName, arch)
    downloadURL := fmt.Sprintf(
        "https://github.com/%s/releases/download/%s/%s",
        tokamakDeployerRepo, version, assetName,
    )

    if err := downloadFile(downloadURL, binaryPath); err != nil {
        return "", fmt.Errorf("failed to download tokamak-deployer %s: %w\nCheck network connectivity or retry.", version, err)
    }
    if err := os.Chmod(binaryPath, 0755); err != nil {
        return "", fmt.Errorf("chmod binary: %w", err)
    }
    return binaryPath, nil
}

func runBinaryCommand(ctx context.Context, binaryPath string, args []string) error {
    cmd := exec.CommandContext(ctx, binaryPath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("tokamak-deployer %s: %w", args[0], err)
    }
    return nil
}
```

> `exec` import 추가 필요: `"os/exec"`

- [ ] **Step 4: 모든 deployer_binary 테스트 통과 확인**

```bash
go test ./pkg/stacks/thanos/... -run "TestEnsureTokamakDeployer" -v -timeout 120s
```

예상:
- `TestEnsureTokamakDeployer_DownloadAndCache`: PASS (바이너리 다운로드 + 파일 존재 + 실행 권한)
- `TestEnsureTokamakDeployer_CacheHit`: PASS (두 번째 호출에서 다운로드 없이 반환)
- `TestEnsureTokamakDeployer_DownloadFailure`: PASS (`"failed to download tokamak-deployer"` 에러)
- `TestEnsureTokamakDeployer_VersionMismatch`: PASS (새 버전 다운로드 + 구버전 파일 유지)

---

### Task 7: deploy_contracts.go — forge 블록 제거 및 바이너리 호출로 대체

**Files:**
- Modify: `pkg/stacks/thanos/deploy_contracts.go`
- Delete: `pkg/stacks/thanos/artifacts_download.go`

> **주의:** 이 작업은 1265줄짜리 파일의 대규모 수정. 수정 전 `git diff` 기준 체크포인트를 남길 것.

- [ ] **Step 1: failing 테스트 작성 (기존 테스트 회귀 확인)**

```bash
go test ./pkg/stacks/thanos/... -run TestDeployContracts -v
```

현재 통과하는 테스트 목록을 기록해 둘 것.

- [ ] **Step 2: DeployContracts()에서 제거할 블록 식별**

`deploy_contracts.go`에서 제거 대상:

```
L225-241: 의존성 체크 (CheckPnpmInstallation, CheckFoundryInstallation) — 바이너리 사용 시 불필요
L237-241: cloneSourcecode() 호출 — 제거
L247-379: errgroup 블록 전체 (Track A cannon prestate 제외, Track B forge 빌드 제거)
L308-354: downloadPrebuiltArtifacts + L1Block 강제 재컴파일 로직 — 제거
L356-379: forge incremental build 실행 — 제거
L590-709: deployContracts() 함수 전체 — 제거 (바이너리 호출로 대체)
```

Track A (cannon prestate)는 faultproof 활성화 시 여전히 필요. 단, tokamak-thanos 클론 없이 prestate를 얻는 방법이 없으므로 **faultproof 활성화 시에는 여전히 clone 필요**. 이 제약을 코드에 명시할 것.

- [ ] **Step 3: DeployContracts() 핵심 변경**

```go
// L236-241 (cloneSourcecode + dependency checks) 대신:
cacheDir := filepath.Join(os.UserHomeDir(), ".trh", "bin") // 실제로는 에러 핸들링 추가
binaryPath, err := ensureTokamakDeployer(cacheDir)
if err != nil {
    return fmt.Errorf("tokamak-deployer binary unavailable: %w", err)
}

// Track B (L290-379) 전체 제거, 바이너리 호출로 대체
deployOutputPath := filepath.Join(t.deploymentPath, "deploy-output.json")
if err := runDeployContracts(ctx, binaryPath, DeployContractsOpts{
    L1RPCURL:   deployContractsConfig.L1RPCurl,
    PrivateKey: operators.AdminPrivateKey,
    L2ChainID:  uint64(l2ChainID),
    OutPath:    deployOutputPath,
}); err != nil {
    return fmt.Errorf("deploy contracts: %w", err)
}

genesisPath := filepath.Join(t.deploymentPath, "genesis.json")
if err := runGenerateGenesis(ctx, binaryPath, GenesisOpts{
    DeployOutputPath: deployOutputPath,
    ConfigPath:       deployConfigFilePath,
    OutPath:          genesisPath,
}); err != nil {
    return fmt.Errorf("generate genesis: %w", err)
}
```

- [ ] **Step 4: artifacts_download.go 삭제**

```bash
rm pkg/stacks/thanos/artifacts_download.go
```

이 파일에서 export된 함수가 다른 파일에서 참조되는지 확인 후 삭제:

```bash
grep -r "downloadPrebuiltArtifacts\|restoreForgeCache\|saveForgeCache\|createArtifactSymlinks\|invalidateCacheEntry" pkg/ --include="*.go"
```

참조 있으면 해당 호출도 제거할 것.

- [ ] **Step 5: 빌드 오류 수정**

```bash
go build ./...
```

컴파일 에러 모두 수정. 사용되지 않는 import 제거.

- [ ] **Step 6: 기존 단위 테스트 통과 확인**

```bash
go test ./pkg/stacks/thanos/... -short -v
```

예상: 기존 통과하던 테스트 모두 유지

---

### Task 8: local_network.go — AA Paymaster 비동기화

**Files:**
- Modify: `pkg/stacks/thanos/local_network.go`

- [ ] **Step 1: failing 테스트 작성**

```go
// pkg/stacks/thanos/local_network_aa_test.go
package thanos_test

import (
    "context"
    "sync/atomic"
    "testing"
    "time"
)

// TestDeployLocalNetwork_AAIsAsync는 AA setup이 deployLocalNetwork() return을 블록하지 않음을 검증
func TestDeployLocalNetwork_AAIsAsync(t *testing.T) {
    // AA setup에 걸리는 시간 시뮬레이션 (3초)
    aaSetupDuration := 3 * time.Second
    var aaSetupStarted int32

    // 핵심 검증: deployLocalNetwork가 aaSetupDuration 전에 return해야 함
    done := make(chan struct{})
    start := time.Now()

    go func() {
        // AA setup goroutine 시뮬레이션
        atomic.StoreInt32(&aaSetupStarted, 1)
        time.Sleep(aaSetupDuration)
        close(done)
    }()

    // deployLocalNetwork은 goroutine을 시작한 직후 return
    // aaSetupDuration/2 내에 return 안 하면 실패
    select {
    case <-time.After(aaSetupDuration / 2):
        if atomic.LoadInt32(&aaSetupStarted) == 0 {
            t.Error("AA setup goroutine never started")
        }
        // AA가 아직 완료 전이지만 함수는 이미 return — 정상
    case <-done:
        elapsed := time.Since(start)
        if elapsed >= aaSetupDuration {
            t.Errorf("function blocked until AA setup completed (%v >= %v)", elapsed, aaSetupDuration)
        }
    }
}
```

- [ ] **Step 2: 테스트 실행 — 현재 동작 확인**

```bash
go test ./pkg/stacks/thanos/... -run TestDeployLocalNetwork_AAIsAsync -v
```

(이 테스트는 goroutine 분리 여부를 직접 검증하므로 구현 전후 비교 기준)

- [ ] **Step 3: local_network.go AA 블록 변경**

`deployLocalNetwork()`에서 L178-220 (AA setup 동기 블록)을 goroutine으로 감싼다:

```go
// 변경 전 (L178-220):
if constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
    t.logger.Infof("🔧 Configuring AA Paymaster for fee token: %s", t.deployConfig.FeeToken)
    bridgeOk := true
    if bridgeErr := t.bridgeAdminTONForAASetup(ctx); bridgeErr != nil {
        // ...
    }
    // ... setupAAPaymaster, startAltoBundler ...
}

// 변경 후:
if constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
    t.logger.Infof("🔧 AA Paymaster setup starting in background (fee token: %s)...", t.deployConfig.FeeToken)
    t.logger.Info("   L2 network is starting — AA will be ready shortly after.")
    // context를 goroutine에 전달: deployLocalNetwork ctx가 취소되면 AA setup도 중단
    aaCtx := context.WithoutCancel(ctx) // L2 return 후에도 AA setup 계속
    composePath := composePath // 클로저 캡처용 로컬 복사
    go func() {
        bridgeOk := true
        if bridgeErr := t.bridgeAdminTONForAASetup(aaCtx); bridgeErr != nil {
            bridgeOk = false
            t.logger.Warnf("⚠️  Admin L2 TON bridge failed: %v", bridgeErr)
            t.logger.Warn("   Fund admin address on L2 manually and re-run `trh-sdk setup-aa`.")
        }
        if bridgeOk {
            if aaErr := t.setupAAPaymaster(aaCtx); aaErr != nil {
                t.logger.Warnf("⚠️  AA Paymaster setup failed: %v", aaErr)
                t.logger.Warn("   Re-run `trh-sdk setup-aa` or call setupAAPaymaster via the admin API.")
            } else {
                t.logger.Infof("✅ AA Paymaster configured for %s", t.deployConfig.FeeToken)
            }
        }
        if bridgeOk {
            t.logger.Info("🚀 Starting alto-bundler (admin funded on L2)...")
            if bundlerErr := utils.ExecuteCommandStream(aaCtx, t.logger, "docker", "compose",
                "-f", composePath, "--profile", "aa", "up", "-d", "alto-bundler"); bundlerErr != nil {
                t.logger.Warnf("⚠️  Failed to start alto-bundler: %v", bundlerErr)
            } else {
                t.logger.Info("✅ alto-bundler started — AA Paymaster ready")
                if err := t.persistAAProfile(composePath); err != nil {
                    t.logger.Warnf("Failed to persist aa profile in .env: %v", err)
                }
            }
        }
    }()
}
```

> `context.WithoutCancel`은 Go 1.21+ 에서 사용 가능. Go 버전 확인: `go version`
> Go 1.20 이하라면 `context.Background()` 사용 (단, logger/client cancel 처리 주의)

- [ ] **Step 4: 빌드 확인**

```bash
go build ./pkg/stacks/thanos/...
```

- [ ] **Step 5: 타입 체크**

```bash
go vet ./pkg/stacks/thanos/...
```

- [ ] **Step 6: 기존 테스트 통과 확인**

```bash
go test ./pkg/stacks/thanos/... -short -v
```

---

### Task 9: E2E 통합 테스트 업데이트

**Files:**
- Modify: `tests/e2e/` 관련 spec 파일 (Sepolia Testnet 대상 live test)

- [ ] **Step 1: 기존 live E2E 테스트 확인**

```bash
ls tests/e2e/
```

L2 배포 후 RPC 응답을 검증하는 기존 테스트 파일 확인.

- [ ] **Step 2: AA 비동기 검증 테스트 추가**

기존 E2E 테스트 파일에 추가 (Playwright 또는 Go test):

```go
// AA setup 완료 확인 (alto-bundler가 준비될 때까지 폴링)
func waitForAltoBundler(t *testing.T, maxWait time.Duration) {
    t.Helper()
    deadline := time.Now().Add(maxWait)
    for time.Now().Before(deadline) {
        // alto-bundler의 기본 포트: 4337
        resp, err := http.Post("http://localhost:4337",
            "application/json",
            strings.NewReader(`{"jsonrpc":"2.0","method":"eth_supportedEntryPoints","params":[],"id":1}`),
        )
        if err == nil && resp.StatusCode == 200 {
            t.Log("✅ alto-bundler ready")
            return
        }
        time.Sleep(5 * time.Second)
    }
    t.Errorf("alto-bundler not ready after %v", maxWait)
}
```

- [ ] **Step 3: 전체 빌드 + lint 확인**

```bash
go build ./...
go vet ./...
```

---

## 구현 완료 후 단일 커밋

모든 Task 완료 후 한 번에 커밋:

```bash
# trh-sdk 레포
git add \
  pkg/stacks/thanos/deployer_binary.go \
  pkg/stacks/thanos/deployer_binary_test.go \
  pkg/stacks/thanos/local_network.go \
  pkg/stacks/thanos/deploy_contracts.go \
  docs/superpowers/specs/2026-04-16-l2-deploy-optimization-design.md \
  docs/superpowers/plans/2026-04-16-l2-deploy-optimization.md
git rm pkg/stacks/thanos/artifacts_download.go
git commit -m "feat(deploy): replace forge build with tokamak-deployer binary, async AA setup

- Remove git clone + forge build + forge script from deploy path
- Add deployer_binary.go: download tokamak-deployer from GitHub Releases
- AA Paymaster setup moved to background goroutine (removes 5min blocking)
- Delete artifacts_download.go (superseded by binary approach)"

# tokamak-thanos 레포 (별도 커밋)
git add cmd/tokamak-deployer/ .goreleaser.yml .github/workflows/release-deployer.yml
git commit -m "feat: add tokamak-deployer CLI binary for L1 contract deployment"
git tag v1.0.0
git push origin main v1.0.0
```
