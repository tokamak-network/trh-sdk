# --enable-fault-proof Flag Design

## Goal

`deploy-contracts` 명령어에 `--enable-fault-proof` CLI 플래그를 추가하여, 활성화 시 Fault Proof 관련 컨트랙트(DisputeGameFactory, FaultDisputeGame 등)가 배포되고 이후 `deploy` 명령어에서도 op-challenger가 정상 작동하도록 한다.

## Background

코드베이스에 절반 구현된 infrastructure가 이미 존재한다:
- `types/configuration.go:123`: `EnableFraudProof bool` — persistent config 필드 존재
- `input.go:272-278`: interactive prompt가 주석 처리된 상태로 `fraudProof = false` 하드코딩
- `deploy_contracts.go:234-239`: challenger key 설정 코드가 주석 처리
- `deploy_contracts.go:245`: `EnableFraudProof = false` 하드코딩
- `SelectAccounts()`: `enableFraudProof bool` 파라미터로 challenger 계정 선택 이미 구현
- `deploy_chain.go:190`: `ChallengerKey` Terraform 전달 이미 연결

## Data Flow

```
CLI --enable-fault-proof
  → commands/contracts.go (flag read)
  → InputDeployContracts(ctx, enableFaultProof=true)
  → DeployContractsInput.EnableFaultProof = true
  → SelectAccounts(enableFaultProof=true) → challenger 계정 선택 프롬프트 표시
  → DeployContracts(deployContractsConfig)
  → t.deployConfig.EnableFraudProof = true
  → challenger private key 설정 (주석 해제)
  → initDeployConfigTemplate: enableFraudProof = true
  → deploy-config.json: "useFaultProofs": true
  → Deploy.s.sol: DisputeGameFactory 초기화
  → settings.json 저장: EnableFraudProof=true

trh-sdk deploy (별도 실행)
  → settings.json에서 EnableFraudProof=true 읽음
  → deployNetworkToAWS: challenger key 비어있으면 early error
  → TF_VAR_challenger_key 전달 (deploy_chain.go:190 기존 코드)
  → Terraform: op-challenger 배포
```

## Files to Modify

| 파일 | 변경 내용 |
|------|----------|
| `flags/flags.go` | `EnableFaultProofFlag` 추가, `DeployContractsFlag`에 등록 |
| `commands/contracts.go` | flag 읽어서 `InputDeployContracts`에 전달 |
| `pkg/stacks/thanos/input.go` | `DeployContractsInput`에 `EnableFaultProof bool` 추가, `InputDeployContracts` 시그니처 변경, 하드코딩 제거, `initDeployConfigTemplate` 수정 |
| `pkg/stacks/thanos/deploy_contracts.go` | 하드코딩 false 제거, challenger key 주석 해제 |
| `pkg/stacks/thanos/deploy_chain.go` | fault proof 활성화 시 challenger key 검증 추가 |

## Component Details

### 1. Flag 정의 (`flags/flags.go`)

```go
EnableFaultProofFlag = &cli.BoolFlag{
    Name:    "enable-fault-proof",
    Usage:   "Enable the fault proof system (deploys DisputeGameFactory and FaultDisputeGame contracts)",
    Value:   false,
    Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "ENABLE_FAULT_PROOF")...),
}
```

### 2. `DeployContractsInput` 구조체 확장

```go
type DeployContractsInput struct {
    L1RPCurl          string
    ChainConfiguration *types.ChainConfiguration
    Operators         *types.Operators
    RegisterCandidate *RegisterCandidateInput
    ReuseDeployment   bool
    EnableFaultProof  bool  // 추가
}
```

### 3. `InputDeployContracts` 시그니처 변경

```go
// Before
func InputDeployContracts(ctx context.Context) (*DeployContractsInput, error)

// After
func InputDeployContracts(ctx context.Context, enableFaultProof bool) (*DeployContractsInput, error)
```

내부에서 `fraudProof := enableFaultProof`로 사용 (주석 처리된 interactive prompt 제거).

### 4. `deploy_contracts.go` 수정

```go
// Before (line 245)
t.deployConfig.EnableFraudProof = false

// After
t.deployConfig.EnableFraudProof = deployContractsConfig.EnableFaultProof
```

주석 해제 (lines 234-239):
```go
if deployContractsConfig.EnableFaultProof {
    if operators.ChallengerPrivateKey == "" {
        return fmt.Errorf("challenger operator is required for fault proof but was not found")
    }
    t.deployConfig.ChallengerPrivateKey = operators.ChallengerPrivateKey
}
```

### 5. `initDeployConfigTemplate` 수정 (`input.go:1384`)

```go
// Before
enableFraudProof = false

// After
enableFraudProof = deployConfigInputs.EnableFaultProof
```

### 6. `deployNetworkToAWS` 검증 추가 (`deploy_chain.go`)

```go
if t.deployConfig.EnableFraudProof && t.deployConfig.ChallengerPrivateKey == "" {
    return fmt.Errorf("fault proof is enabled but challenger private key is not set; re-run deploy-contracts with --enable-fault-proof")
}
```

## Error Handling

- `--enable-fault-proof` 없이 `deploy-contracts` 실행 후 `deploy` 실행 시 challenger key가 없으므로 early error 반환
- fault proof 활성화 시 challenger 계정이 선택되지 않으면 `SelectAccounts`에서 오류

## Testing

- `--enable-fault-proof` 없을 때: `UseFaultProofs=false`, challenger 선택 프롬프트 없음
- `--enable-fault-proof` 있을 때: `UseFaultProofs=true`, challenger 선택 프롬프트 표시
- `deploy` 단독 실행 시 (fault proof 활성화 상태, challenger key 없음): 명확한 오류 메시지
