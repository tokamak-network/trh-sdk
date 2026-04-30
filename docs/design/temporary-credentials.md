# ADR ②: AWS Temporary Credential (Session Token) 공식 지원

```
Status: Draft
Date: 2026-04-14
Owner: trh-platform team
Relates-to: trh-backend/docs/design/credential-storage.md (ADR ③)
Tracked-by: trh-platform/docs/design/preset-aws-rollout.md
```

---

## Context

Electron 앱은 AWS IAM Identity Center(SSO)를 통해 자격증명을 획득한다. SSO `GetRoleCredentials`가 반환하는 creds는 `{AccessKeyId, SecretAccessKey, SessionToken, Expiration}` 4-튜플이다.

**현재 trh-sdk 문제**: `pkg/cloud-provider/aws/aws.go:103`

```go
// 현재 코드
cred := credentials.NewStaticCredentialsProvider(input.AccessKey, input.SecretKey, "")
//                                                                               ^^^
//                                                       SessionToken 자리에 빈 문자열 하드코딩
```

`NewStaticCredentialsProvider`의 세 번째 인자가 SessionToken인데 빈 문자열로 고정되어 있다. SSO temp creds를 사용하면 실제 `SessionToken`이 있는데 이를 버리는 셈이다.

**현재 동작 방식**: Electron이 `sessionToken`을 포함한 temp creds를 전달해도, trh-sdk가 `sessionToken` 없이 AWS SDK 클라이언트를 초기화하기 때문에 일부 IAM action에서 `InvalidClientTokenId` 또는 `AccessDenied` 오류가 발생할 수 있다. 현재 `DescribeRegions`/`GetCallerIdentity` 수준에서는 동작하지만, EKS 클러스터 생성 등 IAM role chaining이 필요한 작업에서는 session token 부재가 실패로 이어질 수 있다.

**trh-sdk 독립 실행(CLI 모드) 시**: `trh deploy`는 `~/.aws/credentials`를 읽는 표준 SDK 경로를 사용하므로 SSO 흐름이 이미 있다. 이 ADR은 **trh-backend가 trh-sdk를 Go module로 직접 임포트할 때** 전달되는 `AWSConfig` 구조체 경로를 수정하는 것이다.

---

## Decision

**`AWSConfig` 구조체에 `SessionToken` 필드를 추가하고, `NewStaticCredentialsProvider` 호출에 세 번째 인자로 전달한다.**

### 변경 대상: `pkg/cloud-provider/aws/aws.go`

```go
// 현재 AWSConfig (추정)
type AWSConfig struct {
    AccessKey string
    SecretKey string
    Region    string
}

// 변경 후
type AWSConfig struct {
    AccessKey    string
    SecretKey    string
    SessionToken string // 추가 — SSO/AssumeRole temp creds용. 비어있으면 기존 static 경로.
    Region       string
    Expiration   *time.Time // 추가 (optional) — refresh 훅을 위한 참고값
}
```

```go
// 변경 후 credentials 생성
cred := credentials.NewStaticCredentialsProvider(
    input.AccessKey,
    input.SecretKey,
    input.SessionToken, // "" 이면 IAM long-lived creds 경로, 값 있으면 temp creds
)
```

### trh-backend 인터페이스 변경

`pkg/stacks/thanos/thanos_stack.go`의 `NewThanosSDKClient`:

```go
// 현재
thanosTypes.AWSConfig{
    AccessKey: cfg.AwsAccessKey,
    SecretKey: cfg.AwsSecretAccessKey,
    Region:    cfg.AwsRegion,
}

// 변경 후
thanosTypes.AWSConfig{
    AccessKey:    cfg.AwsAccessKey,
    SecretKey:    cfg.AwsSecretAccessKey,
    SessionToken: cfg.AwsSessionToken, // ADR ③에서 backend DTO에 추가
    Region:       cfg.AwsRegion,
    Expiration:   cfg.AwsCredExpiration,
}
```

### 하위 호환성

- `SessionToken`이 `""` (빈 문자열)이면 기존 동작과 동일.
- 기존 `trh deploy` CLI 경로는 `AWSConfig`를 직접 생성하지 않고 SDK 표준 credential chain을 사용 → 영향 없음.

---

## Consequences

- **Good**: Electron SSO temp creds의 session token이 실제로 전달되어 IAM 작업 실패 위험 제거.
- **Good**: 명시적인 session token 지원으로 `AssumeRole` 체이닝 시나리오도 수용 가능.
- **Good**: 코드 변경이 2줄 수준으로 최소화.
- **Trade-off**: `Expiration` 필드는 SDK 자체에서 refresh를 하지 않음. 만료 전 재전달은 trh-backend 또는 Electron이 책임(ADR ③ 참조). trh-sdk는 expiration 값을 로그에만 남기고 별도 액션 없음.

---

## Alternatives considered

- **AWS SDK ChainCredentials + OIDC provider**: 완전한 SSO 흐름을 SDK에 구현. 범위가 넓어 이 ADR 목적(현재 배포 경로 수정)을 초과 → 후속 작업으로 분리.
- **환경 변수 주입 (`AWS_SESSION_TOKEN`)**: trh-sdk `aws.go:83-86`이 이미 `os.Setenv`로 subprocess에 주입하는 경로가 있음. 그러나 SDK 클라이언트 자체가 session token 없이 초기화되면 환경변수도 덮어써짐 → 코드 수정이 더 명확.

---

## Implementation checklist

- [ ] `pkg/cloud-provider/aws/aws.go` 의 `AWSConfig`에 `SessionToken`, `Expiration` 필드 추가
- [ ] `credentials.NewStaticCredentialsProvider(key, secret, token)` 세 번째 인자 수정
- [ ] `os.Setenv("AWS_SESSION_TOKEN", input.SessionToken)` 추가 (기존 subprocess 경로도 커버)
- [ ] trh-backend `thanos_stack.go`의 `NewThanosSDKClient` 호출에 `SessionToken` 전달 (ADR ③ 완료 후)
- [ ] 단위 테스트: mock STS `GetCallerIdentity`에 session token 포함 케이스
- [ ] 통합 테스트: LocalStack 또는 AWS sandbox에서 session token으로 `DescribeRegions` + EKS API 호출 검증
- [ ] `Status: Accepted` 로 업데이트
- [ ] 구현 PR merge 후 `Status: Shipped` + trh-wiki gap #4 제거
