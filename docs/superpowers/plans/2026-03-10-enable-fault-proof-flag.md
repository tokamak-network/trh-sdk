# --enable-fault-proof Flag Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `deploy-contracts --enable-fault-proof` 플래그를 추가하여, 활성화 시 Fault Proof 컨트랙트가 배포되고 `deploy` 명령어에서도 op-challenger가 올바르게 설정된다.

**Architecture:** CLI 플래그 값이 `DeployContractsInput.EnableFaultProof`를 통해 `deploy_contracts.go`와 `initDeployConfigTemplate`으로 전달된다. 이미 절반 구현된 코드(주석 처리, 하드코딩 false)를 활성화하는 방식으로 구현한다. `deploy` 명령어는 `settings.json`에 저장된 `EnableFraudProof` 값을 읽으므로, `deploy-contracts`에서 올바르게 저장하면 자동으로 연동된다.

**Tech Stack:** Go 1.21+, `github.com/urfave/cli/v3`, 기존 `flags/`, `commands/`, `pkg/stacks/thanos/` 패키지

---

## 수정 파일 맵

| 파일 | 역할 | 변경 내용 |
|------|------|----------|
| `flags/flags.go` | CLI 플래그 정의 | `EnableFaultProofFlag` 추가, `DeployContractsFlag` 등록 |
| `commands/contracts.go` | CLI 액션 핸들러 | 플래그 값 읽어서 `InputDeployContracts`에 전달 |
| `pkg/stacks/thanos/input.go:35-41` | Input 구조체 | `DeployContractsInput`에 `EnableFaultProof bool` 필드 추가 |
| `pkg/stacks/thanos/input.go:253` | Input 수집 함수 | `InputDeployContracts` 시그니처 + 하드코딩 제거 |
| `pkg/stacks/thanos/input.go:405` | Return 구성 | `EnableFaultProof` 반환 값에 포함 |
| `pkg/stacks/thanos/input.go:1384` | Deploy config template | `enableFraudProof = deployConfigInputs.EnableFaultProof` |
| `pkg/stacks/thanos/deploy_contracts.go:234-245` | 계약 배포 | 하드코딩 false 제거, challenger key 주석 해제 |
| `pkg/stacks/thanos/deploy_chain.go:118-120` 근처 | Infra 배포 | challenger key 검증 추가 |

---

## Chunk 1: 플래그 정의 + 구조체 확장

### Task 1: `EnableFaultProofFlag` 추가 및 `DeployContractsInput` 확장

**Files:**
- Modify: `flags/flags.go`
- Modify: `pkg/stacks/thanos/input.go:35-41`

배경: `flags/flags.go`의 `NoCandidateFlag` 패턴을 그대로 따른다. `DeployContractsInput`은 `pkg/stacks/thanos/input.go:35`에 정의된 struct로, CLI에서 수집된 입력을 `DeployContracts()`로 전달하는 역할이다.

- [ ] **Step 1: `EnableFaultProofFlag` 추가**

`flags/flags.go`의 `ReuseDeploymentFlag` 블록 직후에 추가:
```go
EnableFaultProofFlag = &cli.BoolFlag{
    Name:    "enable-fault-proof",
    Usage:   "Enable the fault proof system (deploys DisputeGameFactory and FaultDisputeGame contracts)",
    Value:   false,
    Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "ENABLE_FAULT_PROOF")...),
}
```

그리고 `DeployContractsFlag` 슬라이스에 등록:
```go
var DeployContractsFlag = []cli.Flag{
    StackFlag,
    NetworkFlag,
    NoCandidateFlag,
    ReuseDeploymentFlag,
    EnableFaultProofFlag,  // 추가
}
```

- [ ] **Step 2: `DeployContractsInput`에 `EnableFaultProof` 필드 추가**

`pkg/stacks/thanos/input.go:35-41`:
```go
type DeployContractsInput struct {
    L1RPCurl           string
    ChainConfiguration *types.ChainConfiguration
    Operators          *types.Operators
    RegisterCandidate  *RegisterCandidateInput
    ReuseDeployment    bool
    EnableFaultProof   bool  // 추가
}
```

- [ ] **Step 3: 빌드 통과 확인**

```bash
go build ./...
```
예상: 에러 없음

- [ ] **Step 4: `--help`에 플래그 노출 확인**

```bash
go run . deploy-contracts --help 2>&1 | grep -i "fault"
```
예상:
```
--enable-fault-proof  Enable the fault proof system (deploys DisputeGameFactory and FaultDisputeGame contracts)
```

- [ ] **Step 5: 커밋**

```bash
git add flags/flags.go pkg/stacks/thanos/input.go
git commit -m "feat: add EnableFaultProofFlag and DeployContractsInput.EnableFaultProof field"
```

---

## Chunk 2: CLI → InputDeployContracts 연결

### Task 2: 플래그 값을 `InputDeployContracts`로 전달

**Files:**
- Modify: `commands/contracts.go:46`
- Modify: `pkg/stacks/thanos/input.go:253-415`

배경: `commands/contracts.go:46`에서 `thanos.InputDeployContracts(ctx)`를 호출한다. 여기서 플래그 값을 읽어 함수에 전달해야 한다. `InputDeployContracts` 내부에서는 `fraudProof := false` 하드코딩(line 272)을 제거하고 파라미터를 사용한다. 주석 처리된 interactive prompt (lines 273-278)는 삭제한다.

- [ ] **Step 1: `commands/contracts.go` 수정**

`commands/contracts.go:20` 근처에서 플래그 값 읽기 추가:
```go
enableRegisterCandidate := !cmd.Bool(flags.NoCandidateFlag.Name)
enableFaultProof := cmd.Bool(flags.EnableFaultProofFlag.Name)  // 추가
```

`commands/contracts.go:46` 수정:
```go
// Before:
deployContractsConfig, err := thanos.InputDeployContracts(ctx)

// After:
deployContractsConfig, err := thanos.InputDeployContracts(ctx, enableFaultProof)
```

- [ ] **Step 2: `InputDeployContracts` 시그니처 변경**

`pkg/stacks/thanos/input.go:253`:
```go
// Before:
func InputDeployContracts(ctx context.Context) (*DeployContractsInput, error) {

// After:
func InputDeployContracts(ctx context.Context, enableFaultProof bool) (*DeployContractsInput, error) {
```

- [ ] **Step 3: 함수 내부 하드코딩 제거**

`input.go:272-278` 현재:
```go
fraudProof := false
//fmt.Print("Would you like to enable the fault-proof system on your chain? [Y or N] (default: N): ")
//fraudProof, err = scanner.ScanBool()
//if err != nil {
//	fmt.Printf("Error while reading the fault-proof system setting: %s", err)
//	return nil, err
//}
```

수정 후 (주석 포함 블록 전체 교체):
```go
fraudProof := enableFaultProof
```

- [ ] **Step 4: 반환 값에 `EnableFaultProof` 포함**

`input.go:405-415`:
```go
return &DeployContractsInput{
    L1RPCurl: l1RPCUrl,
    ChainConfiguration: &types.ChainConfiguration{
        L2BlockTime:              l2BlockTime,
        L1BlockTime:              l1BlockTime,
        BatchSubmissionFrequency: batchSubmissionFrequency,
        ChallengePeriod:          challengePeriod,
        OutputRootFrequency:      outputFrequency,
    },
    Operators:        operators,
    EnableFaultProof: enableFaultProof,  // 추가
}, nil
```

- [ ] **Step 5: 빌드 통과 확인**

```bash
go build ./...
```
예상: 에러 없음

- [ ] **Step 6: 커밋**

```bash
git add commands/contracts.go pkg/stacks/thanos/input.go
git commit -m "feat: wire --enable-fault-proof flag through InputDeployContracts"
```

---

## Chunk 3: deploy_contracts.go — 하드코딩 제거 + challenger key 활성화

### Task 3: `EnableFraudProof` 연결 및 challenger key 주석 해제

**Files:**
- Modify: `pkg/stacks/thanos/deploy_contracts.go:234-245`

배경: `deploy_contracts.go:245`는 `t.deployConfig.EnableFraudProof = false`로 하드코딩되어 있다. `lines 234-239`는 challenger private key를 설정하는 코드가 주석 처리되어 있다. 두 곳을 수정한다.

- [ ] **Step 1: 현재 코드 확인**

```bash
sed -n '230,250p' pkg/stacks/thanos/deploy_contracts.go
```
예상:
```go
// if deployContractsConfig.FraudProof {
// 	if operators.ChallengerPrivateKey == "" {
// 		return fmt.Errorf("challenger operator is required for fault proof but was not found")
// 	}
// 	t.deployConfig.ChallengerPrivateKey = operators.ChallengerPrivateKey
// }
...
t.deployConfig.EnableFraudProof = false
```

- [ ] **Step 2: 주석 해제 + 조건 변수명 수정**

`deploy_contracts.go:234-239` 주석 해제 및 변수명 수정:
```go
if deployContractsConfig.EnableFaultProof {
    if operators.ChallengerPrivateKey == "" {
        return fmt.Errorf("challenger operator is required for fault proof but was not found")
    }
    t.deployConfig.ChallengerPrivateKey = operators.ChallengerPrivateKey
}
```

- [ ] **Step 3: 하드코딩 false 제거**

`deploy_contracts.go:245`:
```go
// Before:
t.deployConfig.EnableFraudProof = false

// After:
t.deployConfig.EnableFraudProof = deployContractsConfig.EnableFaultProof
```

- [ ] **Step 4: 빌드 통과 확인**

```bash
go build ./...
```
예상: 에러 없음

- [ ] **Step 5: 커밋**

```bash
git add pkg/stacks/thanos/deploy_contracts.go
git commit -m "feat: activate fault proof wiring in DeployContracts - remove hardcoded false and enable challenger key"
```

---

## Chunk 4: deploy config template 수정 + deploy_chain.go 검증

### Task 4: `initDeployConfigTemplate` 하드코딩 수정

**Files:**
- Modify: `pkg/stacks/thanos/input.go:1384`

배경: `initDeployConfigTemplate` 함수(input.go:1370)는 `DeployContractsInput`을 파라미터로 받는다. 내부에서 `enableFraudProof = false`로 하드코딩되어 있어 `deployConfigInputs.EnableFaultProof` 값을 무시한다.

- [ ] **Step 1: 현재 코드 확인**

```bash
sed -n '1378,1390p' pkg/stacks/thanos/input.go
```
예상:
```go
var (
    l2BlockTime                      = chainConfiguration.L2BlockTime
    l1ChainId                        = l1ChainID
    l2OutputOracleSubmissionInterval = chainConfiguration.GetL2OutputOracleSubmissionInterval()
    finalizationPeriods              = chainConfiguration.GetFinalizationPeriodSeconds()
    enableFraudProof                 = false
)
```

- [ ] **Step 2: 하드코딩 제거**

`input.go:1384`:
```go
// Before:
enableFraudProof = false

// After:
enableFraudProof = deployConfigInputs.EnableFaultProof
```

- [ ] **Step 3: 빌드 통과 확인**

```bash
go build ./...
```
예상: 에러 없음

---

### Task 5: `deployNetworkToAWS`에 challenger key 검증 추가

**Files:**
- Modify: `pkg/stacks/thanos/deploy_chain.go`
- Test: `pkg/stacks/thanos/deploy_chain_test.go`

배경: `deploy` 명령어는 `settings.json`에 저장된 `EnableFraudProof` 값을 읽는다. fault proof가 활성화됐는데 challenger key가 없으면 Terraform 단계까지 진행하다 실패하므로, 초기에 명확한 에러를 반환한다.

실제 `deployNetworkToAWS` 코드 구조 (`deploy_chain.go:93~125`):
```
line 93:  func (t *ThanosStack) deployNetworkToAWS(...) {
line 98:  STEP 1. 의존성 체크 (terraform, helm, aws, k8s) -- line 98~116
line 118: inputs == nil 체크
line 122: inputs.Validate(ctx)
```

검증 삽입 위치: `inputs.Validate(ctx)` 바로 다음 (line 122 이후). 이 시점은 모든 의존성/입력 검증이 완료된 후이므로 가장 자연스럽다.

- [ ] **Step 1: 검증 코드 추가**

`deploy_chain.go:122` (`inputs.Validate(ctx)` 에러 처리 블록) 직후에 추가:
```go
// Validate fault proof configuration: challenger key must be set when fault proof is enabled.
if t.deployConfig.EnableFraudProof && t.deployConfig.ChallengerPrivateKey == "" {
    return fmt.Errorf("fault proof is enabled but challenger private key is not set; " +
        "re-run 'deploy-contracts --enable-fault-proof' to configure the challenger account")
}
```

- [ ] **Step 2: 빌드 통과 확인**

```bash
go build ./...
```

- [ ] **Step 3: 단위 테스트 작성**

`types.Config`가 `EnableFraudProof`와 `ChallengerPrivateKey`를 가진 실제 타입이다 (`pkg/types/configuration.go:123`). `ThanosStack.deployConfig`는 `*types.Config` 타입이다 (`thanos_stack.go:15`).

`pkg/stacks/thanos/deploy_chain_test.go`에 추가:

```go
func TestDeployNetworkToAWSFaultProofValidation(t *testing.T) {
    t.Run("fault proof enabled with empty challenger key returns error", func(t *testing.T) {
        stack := &ThanosStack{
            deployConfig: &types.Config{
                EnableFraudProof:     true,
                ChallengerPrivateKey: "",
            },
        }
        err := stack.deployNetworkToAWS(context.Background(), &DeployInfraInput{ChainName: "test"})
        if err == nil {
            t.Fatal("expected error when fault proof enabled without challenger key")
        }
        if !strings.Contains(err.Error(), "challenger private key is not set") {
            t.Errorf("unexpected error message: %v", err)
        }
    })

    t.Run("fault proof disabled with empty challenger key does not error on that check", func(t *testing.T) {
        stack := &ThanosStack{
            deployConfig: &types.Config{
                EnableFraudProof:     false,
                ChallengerPrivateKey: "",
            },
        }
        // The function will fail on dependency checks (terraform not installed in test env),
        // but should NOT fail due to the challenger key check.
        err := stack.deployNetworkToAWS(context.Background(), &DeployInfraInput{ChainName: "test"})
        if err != nil && strings.Contains(err.Error(), "challenger private key is not set") {
            t.Errorf("should not fail on challenger key check when fault proof disabled: %v", err)
        }
    })
}
```

- [ ] **Step 4: 테스트 실행**

```bash
go test ./pkg/stacks/thanos/... -run TestDeployNetworkToAWSFaultProofValidation -v
```
예상:
```
--- PASS: TestDeployNetworkToAWSFaultProofValidation/fault_proof_enabled_with_empty_challenger_key_returns_error
--- PASS: TestDeployNetworkToAWSFaultProofValidation/fault_proof_disabled_with_empty_challenger_key_does_not_error_on_that_check
```

- [ ] **Step 5: 전체 테스트 통과 확인**

```bash
go test ./... 2>&1 | grep -E "^(FAIL|ok)"
```
예상: FAIL 없음

- [ ] **Step 6: 커밋**

```bash
git add pkg/stacks/thanos/input.go pkg/stacks/thanos/deploy_chain.go pkg/stacks/thanos/deploy_chain_test.go
git commit -m "feat: use EnableFaultProof in deploy config template and add challenger key validation"
```

---

## 최종 검증

모든 Task 완료 후:

```bash
# 전체 빌드
go build ./...

# 전체 테스트
go test ./...

# 플래그 노출 확인
go run . deploy-contracts --help | grep -i fault
```

예상 출력:
```
--enable-fault-proof   Enable the fault proof system (deploys DisputeGameFactory and FaultDisputeGame contracts) (default: false)
```

## 변경 요약

| Task | 파일 | 핵심 변경 |
|------|------|----------|
| 1 | `flags/flags.go`, `input.go:35` | 플래그 정의, 구조체 필드 |
| 2 | `commands/contracts.go`, `input.go:253,272,405` | 플래그 값 전달, 하드코딩 제거 |
| 3 | `deploy_contracts.go:234-245` | challenger key 주석 해제, false 제거 |
| 4 | `input.go:1384`, `deploy_chain.go` | template 수정, challenger 검증 |
