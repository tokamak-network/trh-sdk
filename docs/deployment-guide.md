# TRH-SDK L2 배포 가이드

## 목차

- [사전 준비](#사전-준비)
- [프리셋 개요](#프리셋-개요)
- [Step 1: 컨트랙트 배포 (deploy-contracts)](#step-1-컨트랙트-배포)
  - [General 프리셋](#general-프리셋-배포)
  - [DeFi 프리셋](#defi-프리셋-배포)
  - [Gaming 프리셋](#gaming-프리셋-배포)
  - [Full 프리셋](#full-프리셋-배포)
  - [Fault Proof 활성화](#fault-proof-활성화)
- [Step 2: 인프라 배포 (deploy)](#step-2-인프라-배포)
  - [Local 환경 배포](#local-환경-배포)
  - [AWS 환경 배포](#aws-환경-배포)
- [환경별 비교](#환경별-비교)
- [인프라 제거 (destroy)](#인프라-제거)
- [설정 참조](#설정-참조)

---

## 사전 준비

### 공통 의존성

| 도구 | 용도 | Local | AWS |
|------|------|:-----:|:---:|
| Go 1.21+ | trh-sdk 빌드 | O | O |
| pnpm | tokamak-thanos 패키지 관리 | O | O |
| Foundry (forge) | Solidity 컴파일 및 배포 | O | O |
| Docker / Docker Compose | 로컬 컨테이너 운영 | O | - |
| AWS CLI | AWS 리소스 관리 | - | O |
| Terraform | 인프라 프로비저닝 | - | O |
| Helm | Kubernetes 차트 배포 | - | O |
| kubectl | Kubernetes 클러스터 관리 | - | O |

### 운영자 지갑

| 운영자 | 역할 | 권장 잔액 |
|--------|------|----------|
| **Admin** | 컨트랙트 배포 및 거버넌스 | 1-2 ETH |
| **Sequencer** | L2 트랜잭션 시퀀싱 | 0.1+ ETH |
| **Batcher** | L1에 배치 제출 | 0.3+ ETH |
| **Proposer** | 상태 루트 제안 | 0.3+ ETH |
| **Challenger** | 분쟁 참여 (Fault Proof 시만) | 0.3+ ETH |

> 운영자 계정은 시드 문구(mnemonic)로부터 파생되거나, 개별 프라이빗 키로 제공할 수 있습니다.

### L1 네트워크 정보

| L1 체인 | Chain ID | Challenge Period | 비고 |
|---------|----------|-----------------|------|
| Ethereum Mainnet | 1 | 7일 (고정) | 프로덕션 |
| Ethereum Sepolia | 11155111 | 커스텀 가능 | 테스트넷 |
| Ethereum Holesky | 17000 | 커스텀 가능 | 테스트넷 |

### 필요한 엔드포인트

- **L1 RPC URL**: Infura, Alchemy 등 (예: `https://sepolia.infura.io/v3/YOUR_KEY`)
- **L1 Beacon URL**: Beacon API 엔드포인트 (예: `https://sepolia-beacon-api.staking.tokamak.network`)

---

## 프리셋 개요

| 프리셋 | 포함 구성요소 | 사용 사례 |
|--------|-------------|----------|
| **General** | Bridge, Block Explorer | 최소 기능, 커스텀 체인 |
| **DeFi** | General + Monitoring + Uniswap V3 + USDC | DeFi 프로토콜 운영 |
| **Gaming** | General + Monitoring + DRB (Commit-Reveal Random Beacon) + Account Abstraction (ERC-4337) | 게임/AA 기반 서비스 |
| **Full** | DeFi + Gaming 모두 포함 | 모든 기능 필요 시 |

### 수수료 토큰 옵션

| 토큰 | 설명 |
|------|------|
| **TON** | Tokamak Network 토큰 (기본값) |
| **ETH** | 네이티브 이더리움 |
| **USDT** | Tether USD |
| **USDC** | USD Coin |

---

## Step 1: 컨트랙트 배포

> 이 단계는 **Local/AWS 환경과 무관하게 동일**합니다. L1에 스마트 컨트랙트를 배포하고 L2 genesis 파일을 생성합니다.

### General 프리셋 배포

최소 predeploy만 포함하는 기본 L2 체인입니다. Bridge와 Block Explorer만 구성됩니다.

**언제 사용하나요?**

- 커스텀 DApp 전용 체인을 구축할 때
- DeFi/Gaming predeploy가 불필요할 때
- 가장 빠른 배포와 최소 가스비를 원할 때

```bash
# 대화형 모드
trh-sdk deploy-contracts --network testnet --stack thanos --preset general

# 비대화형 모드
trh-sdk deploy-contracts \
  --network testnet \
  --stack thanos \
  --preset general \
  --fee-token TON
```

### DeFi 프리셋 배포

General 프리셋에 DeFi 프로토콜 predeploy를 추가한 구성입니다.

**포함 구성요소**: Bridge, Block Explorer, Monitoring, **Uniswap V3**, **USDC 브릿지**

**언제 사용하나요?**

- L2에서 DEX 및 DeFi 서비스를 운영할 때
- USDC 네이티브 브릿지가 필요할 때
- 토큰 스왑 기능을 기본 제공하고 싶을 때

```bash
trh-sdk deploy-contracts \
  --network testnet \
  --stack thanos \
  --preset defi \
  --fee-token TON
```

> DeFi 프리셋은 USDC 브릿지 컨트랙트를 자동 배포하므로, 수수료 토큰과 관계없이 L1↔L2 USDC 전송이 가능합니다.

### Gaming 프리셋 배포

게임 및 Account Abstraction(AA) 중심의 구성입니다.

**포함 구성요소**: Bridge, Block Explorer, Monitoring, **DRB (Commit-Reveal Random Beacon)**, **Account Abstraction (ERC-4337)**, Uptime Service, Cross-Trade

**언제 사용하나요?**

- 온체인 게임을 구축할 때 (DRB로 검증 가능한 공정 난수 필요)
- 가스리스 트랜잭션이 필요할 때 (AA Paymaster)
- 사용자 경험을 위해 스마트 계정을 활용할 때

**추가 설정 항목:**

| 설정 | 설명 | 기본값 |
|------|------|-------|
| **DRB Admin Address** | DRB 컨트랙트 거버넌스 주소 | Admin 계정 주소 |
| **AA Paymaster Signer** | Paymaster 서명자 주소 | Admin 계정 주소 |

```bash
trh-sdk deploy-contracts \
  --network testnet \
  --stack thanos \
  --preset gaming \
  --fee-token ETH
```

### Full 프리셋 배포

DeFi와 Gaming의 모든 predeploy를 포함하는 완전한 구성입니다.

**포함 구성요소**: DeFi 전체 + Gaming 전체 (Uniswap V3, USDC, DRB, AA, Uptime Service, Cross-Trade)

**언제 사용하나요?**

- 범용 L2 체인을 구축할 때
- DeFi와 게이밍 모두 지원해야 할 때

```bash
trh-sdk deploy-contracts \
  --network testnet \
  --stack thanos \
  --preset full \
  --fee-token TON
```

> Full 프리셋은 Gaming과 동일한 추가 설정(VRF Admin, AA Paymaster Signer)을 요청합니다.

### Fault Proof 활성화

모든 프리셋에서 `--enable-fault-proof` 플래그를 추가하여 Fault Proof 시스템을 활성화할 수 있습니다.

Fault Proof는 L2 상태 전환의 정확성을 검증하는 분쟁 시스템입니다:

- **Challenger 운영자**가 추가로 필요합니다
- 잘못된 상태 루트에 대해 분쟁(challenge)을 제기할 수 있습니다
- 분쟁 게임(Dispute Game)을 통해 상태를 검증합니다

```bash
# 예: Full + Fault Proof
trh-sdk deploy-contracts \
  --network testnet \
  --stack thanos \
  --preset full \
  --fee-token ETH \
  --enable-fault-proof
```

**Fault Proof 배포 시 추가 프로세스:**

1. Challenger 운영자 계정 설정 (0.3+ ETH 필요)
2. AnchorStateRegistry 컨트랙트 패치 (자동)
3. Cannon prestate 빌드 (`op-program/bin/prestate.json`)
4. DisputeGameFactory 및 FaultDisputeGame 배포

### 배포 결과물

모든 프리셋의 컨트랙트 배포 완료 후 동일한 구조가 생성됩니다:

```
<workspace>/
├── settings.json                              # 전체 배포 설정
└── tokamak-thanos/
    ├── build/
    │   ├── genesis.json                       # L2 초기 상태
    │   └── rollup.json                        # 롤업 설정
    ├── op-program/bin/
    │   └── prestate.json                      # Cannon prestate (Fault Proof 시)
    └── packages/tokamak/contracts-bedrock/
        └── deployments/
            └── <l1_chain_id>-deploy.json      # L1 컨트랙트 주소
```

---

## Step 2: 인프라 배포

컨트랙트 배포 완료 후 L2 노드 인프라를 구성합니다. **Local**과 **AWS** 두 가지 환경을 지원합니다.

```bash
trh-sdk deploy
```

---

### Local 환경 배포

Docker Compose 기반으로 로컬 머신에서 L2 체인을 운영합니다. 개발 및 테스트 용도에 적합합니다.

#### 사전 준비 (Local)

| 항목 | 요구사항 |
|------|---------|
| Docker | Docker Engine + Docker Compose |
| 디스크 | 최소 50GB 여유 공간 |
| 메모리 | 최소 8GB RAM 권장 |
| 네트워크 | L1 RPC 접근 가능 |

#### 배포 방법

```bash
# deploy-contracts 완료 후
trh-sdk deploy
```

대화형 프롬프트에서:

| 항목 | 입력값 |
|------|-------|
| Network | `devnet` (Local Devnet) 또는 `testnet` (Local + Testnet) |
| Infrastructure Provider | `localhost` (자동 선택) |
| L1 Beacon URL | Beacon API 엔드포인트 |

#### Local Devnet vs Local Network

| 모드 | 설명 | L1 |
|------|------|-----|
| **Local Devnet** | 자체 L1 포함, 완전 독립 환경 | Mock L1 (Docker) |
| **Local Network** | 외부 L1 (Sepolia 등)에 연결 | 실제 L1 사용 |

**Local Devnet** (`--network devnet`):

```bash
# tokamak-thanos 저장소의 ops-bedrock을 사용하여 L1+L2 모두 로컬에서 구동
trh-sdk deploy-contracts --network devnet --stack thanos
trh-sdk deploy
```

**Local Network** (`--network testnet` + localhost):

```bash
# L1은 Sepolia, L2 노드만 로컬에서 구동
trh-sdk deploy-contracts --network testnet --stack thanos --preset defi --fee-token TON
trh-sdk deploy  # Infrastructure Provider: localhost 선택
```

#### 배포되는 서비스

Docker Compose로 다음 서비스들이 구동됩니다:

**핵심 서비스 (항상 배포):**

| 서비스 | 포트 | 설명 |
|--------|------|------|
| **op-geth** | 8545 (HTTP), 8546 (WS), 8551 (Auth) | L2 실행 클라이언트 |
| **op-node** | 9545 (RPC), 7300 (메트릭) | L2 합의 클라이언트 (Sequencer) |
| **op-batcher** | 8548 | L1 배치 제출 |
| **op-proposer** | 8560 | L2 상태 루트 제출 |

> Fault Proof 활성화 시 op-proposer 대신 **op-challenger**가 배포됩니다.

**선택적 서비스 (프리셋에 따라):**

| 서비스 | 포트 | 프로파일 | 포함 프리셋 |
|--------|------|---------|-----------|
| **Bridge UI** | 3001 | `bridge` | 모든 프리셋 |
| **Blockscout API** | 4000 | `blockExplorer` | 모든 프리셋 |
| **Blockscout Frontend** | 4001 | `blockExplorer` | 모든 프리셋 |
| **Blockscout DB** | (내부) | `blockExplorer` | PostgreSQL 15 |
| **Prometheus** | 9090 | `monitoring` | DeFi, Gaming, Full |
| **Grafana** | 3002 | `monitoring` | DeFi, Gaming, Full |
| **Uptime Kuma** | 3003 | `uptimeService` | DeFi, Gaming, Full |

#### Local 엔드포인트 요약

```
L2 RPC (HTTP):   http://localhost:8545
L2 RPC (WS):     ws://localhost:8546
Op-Node RPC:     http://localhost:9545
Bridge UI:       http://localhost:3001
Block Explorer:  http://localhost:4001
Grafana:         http://localhost:3002  (기본 계정: admin/admin)
Prometheus:      http://localhost:9090
Uptime Kuma:     http://localhost:3003
```

#### op-geth 설정

로컬 환경에서 op-geth는 다음과 같이 구성됩니다:

- **Archive 모드**: `--gcmode=archive` (전체 상태 보존)
- **P2P 비활성**: `--nodiscover`, `--maxpeers=0`
- **데이터 디렉토리**: `<workspace>/op-geth-data/chaindata`
- **JWT 인증**: 자동 생성된 `jwt.txt`로 op-node와 통신

#### Local 배포 예시 (프리셋별)

```bash
# General - 최소 로컬 테스트
trh-sdk deploy-contracts --network testnet --stack thanos --preset general --fee-token ETH
trh-sdk deploy  # → localhost 선택
# 결과: op-geth + op-node + op-batcher + op-proposer + Bridge

# DeFi - DEX 테스트 환경
trh-sdk deploy-contracts --network testnet --stack thanos --preset defi --fee-token TON
trh-sdk deploy  # → localhost 선택
# 결과: 위 + Blockscout + Prometheus + Grafana + Uptime Kuma

# Full + Fault Proof - 완전한 로컬 테스트
trh-sdk deploy-contracts --network testnet --stack thanos --preset full --fee-token ETH --enable-fault-proof
trh-sdk deploy  # → localhost 선택
# 결과: 모든 서비스 + op-challenger (op-proposer 대체)
```

---

### AWS 환경 배포

AWS EKS(Kubernetes) 기반으로 프로덕션급 L2 체인을 운영합니다. Testnet 및 Mainnet 배포에 사용됩니다.

#### 사전 준비 (AWS)

| 항목 | 요구사항 |
|------|---------|
| AWS 계정 | IAM 사용자 (EKS, VPC, EFS, RDS, S3, Secrets Manager 권한) |
| AWS CLI | 설치 및 인증 구성 |
| Terraform | v1.0+ |
| Helm | v3.0+ |
| kubectl | EKS 클러스터 버전과 호환 |

**필요한 AWS IAM 권한:**

- EKS 클러스터 관리 (생성, 삭제, Fargate 프로파일)
- VPC/서브넷/NAT 게이트웨이 관리
- EFS 파일시스템 관리
- RDS 인스턴스 관리
- S3 버킷 관리
- Secrets Manager 관리
- IAM 역할/정책 관리
- CloudWatch 로그 관리
- Elastic Load Balancer 관리

#### 배포 명령어

```bash
# deploy-contracts 완료 후
trh-sdk deploy
```

대화형 프롬프트에서:

| 항목 | 입력값 | 설명 |
|------|-------|------|
| Infrastructure Provider | `AWS` | 자동 선택 (testnet/mainnet) |
| AWS Access Key ID | IAM Access Key | AWS 인증 |
| AWS Secret Access Key | IAM Secret Key | AWS 인증 |
| AWS Region | 예: `us-east-1` | 인프라 배포 리전 |
| Chain Name | 14자 이하 | K8s 네임스페이스로 사용 |
| L1 Beacon URL | Beacon API 엔드포인트 | op-node 필수 |
| Backup 활성화 | Yes/No | testnet은 선택, mainnet은 기본 활성화 |

#### AWS 배포 프로세스 (9단계)

```
STEP 0: 의존성 검증
  └─ Terraform, Helm, AWS CLI, kubectl 설치 확인

STEP 1: 저장소 클론
  └─ tokamak-thanos-stack 저장소 클론

STEP 2: AWS 인증
  └─ Access Key, Secret Key, Region 저장

STEP 3: Terraform 환경 파일 생성
  └─ .envrc 생성 (변수 주입)
  └─ Fault Proof 시: Cannon prestate 빌드 및 해시 추출

STEP 4: 설정 파일 복사
  └─ genesis.json, rollup.json, prestate.json → Terraform config 디렉토리

STEP 5: Terraform 백엔드 초기화
  └─ S3 + DynamoDB 상태 백엔드 생성

STEP 6: 인프라 배포
  └─ VPC, EKS, EFS, RDS, Secrets Manager 등 생성
  └─ 소요 시간: ~15-20분

STEP 7: EKS 액세스 구성
  └─ kubeconfig 설정, 컨텍스트 전환, 클러스터 준비 확인

STEP 8: Helm 차트 배포
  └─ PVC 프로비저닝 → 서비스 배포 → 플러그인 설치
  └─ L2 RPC URL 추출 (Ingress 주소)

STEP 9: 플러그인 설치
  └─ Bridge, Block Explorer, Monitoring 등 (프리셋에 따라)
```

#### AWS 인프라 구성요소

**Terraform 모듈 구조:**

```
terraform/
├── backend/                      # S3 + DynamoDB 상태 관리
└── thanos-stack/
    └── modules/
        ├── vpc/                   # VPC, 서브넷, NAT 게이트웨이
        ├── eks/                   # EKS 클러스터, Fargate 프로파일
        ├── efs/                   # EFS 볼륨, AWS Backup
        ├── rds/                   # PostgreSQL 14 (Block Explorer용)
        ├── kubernetes/            # K8s 리소스, Load Balancer Controller
        ├── chain-config/          # S3에 genesis/rollup/prestate 저장
        └── secretsmanager/        # 운영자 키 안전 저장
```

**생성되는 AWS 리소스:**

| 리소스 | 구성 | 용도 |
|--------|------|------|
| **VPC** | CIDR 10.0.0.0/16, Public/Private 서브넷 | 네트워크 격리 |
| **EKS** | Fargate 프로파일, API 인증 모드 | Kubernetes 클러스터 |
| **EFS** | 암호화, Elastic 처리량 | 체인 데이터 영속 스토리지 |
| **RDS** | PostgreSQL 14, db.t3.medium | Block Explorer DB |
| **S3** | Genesis, Rollup, Prestate 저장 | 설정 파일 호스팅 |
| **Secrets Manager** | Sequencer/Batcher/Proposer/Challenger 키 | 운영자 키 관리 |
| **NAT Gateway** | 단일 NAT (single_nat_gateway) | Private 서브넷 외부 접근 |
| **ALB** | Application Load Balancer | L2 RPC Ingress |
| **AWS Backup** | EFS 일일 백업 (활성화 시) | 데이터 보호 |

#### Kubernetes 포드 구성

**핵심 포드 (항상 배포):**

| 포드 | 이미지 | 설명 |
|------|--------|------|
| **op-geth** | `tokamaknetwork/thanos-op-geth:latest` | L2 실행 클라이언트, EFS 마운트 |
| **op-node** | `tokamaknetwork/thanos-op-node:latest` | Sequencer 모드, L1 동기화 |
| **op-batcher** | `tokamaknetwork/thanos-op-batcher:latest` | L1 배치 제출 |
| **op-proposer** | `tokamaknetwork/thanos-op-proposer:latest` | L2 상태 제출 (기본) |
| **op-challenger** | `tokamaknetwork/thanos-op-challenger:latest` | 분쟁 참여 (Fault Proof 시) |

> 각 포드는 Secrets Manager에서 운영자 키를 주입받으며, Fargate에서 실행됩니다.

**선택적 포드 (프리셋에 따라):**

| 포드 | 포함 프리셋 | 의존성 |
|------|-----------|--------|
| Bridge | 모든 프리셋 | - |
| Blockscout | 모든 프리셋 | RDS (PostgreSQL) |
| Prometheus + Grafana | DeFi, Gaming, Full | EFS (선택적 영속) |
| Uptime Kuma | DeFi, Gaming, Full | - |

#### AWS 엔드포인트

```
L2 RPC:          https://<ingress-address>    (ALB를 통해 외부 노출)
Block Explorer:  https://<blockscout-ingress>
Grafana:         https://<grafana-ingress>     (인증 필요)
```

> 정확한 엔드포인트는 배포 완료 후 `settings.json`의 `l2_rpc_url`에 기록됩니다.

#### AWS 배포 예시 (프리셋별)

```bash
# General - 최소 프로덕션 배포
trh-sdk deploy-contracts --network testnet --stack thanos --preset general --fee-token TON
trh-sdk deploy
# → AWS, us-east-1, Chain Name: "my-l2-testnet"

# DeFi - DeFi 프로덕션 배포
trh-sdk deploy-contracts --network testnet --stack thanos --preset defi --fee-token USDC
trh-sdk deploy
# → AWS, ap-northeast-2, Chain Name: "defi-chain"

# Full + Fault Proof - 완전한 Mainnet 배포
trh-sdk deploy-contracts --network mainnet --stack thanos --preset full --fee-token TON --enable-fault-proof
trh-sdk deploy
# → AWS, us-east-1, Chain Name: "tokamak-mainnet"
# Backup: 자동 활성화 (mainnet)
```

#### AWS 비용 예상

| 리소스 | 월간 비용 (USD) | 비고 |
|--------|:---------------:|------|
| EKS 클러스터 | ~$73 | $0.10/시간 |
| Fargate 포드 | ~$100-200 | vCPU/메모리 기반 |
| EFS 스토리지 | ~$150 | 500GB 기준 |
| RDS (db.t3.medium) | ~$86 | Block Explorer용 |
| NAT Gateway | ~$33 | $0.045/시간 |
| ALB | ~$16 | 기본 요금 |
| AWS Backup | ~$25 | EFS 백업 |
| **합계** | **~$400-600** | 리전/트래픽에 따라 변동 |

---

## 환경별 비교

| 항목 | Local | AWS |
|------|-------|-----|
| **인프라** | Docker Compose | EKS (Kubernetes) + Terraform |
| **스토리지** | 로컬 Docker 볼륨 | EFS (영속, 암호화) |
| **데이터베이스** | PostgreSQL (Docker) | RDS (PostgreSQL 14, 관리형) |
| **키 관리** | 파일 기반 (settings.json) | AWS Secrets Manager |
| **백업** | 없음 | AWS Backup (EFS 일일 백업) |
| **네트워크** | 직접 포트 매핑 (localhost) | ALB + Ingress (도메인 연결 가능) |
| **모니터링** | Docker 프로파일 기반 | Helm 차트 (CloudWatch 연동 가능) |
| **배포 시간** | ~5-10분 | ~30-45분 |
| **월간 비용** | $0 | ~$400-600 |
| **확장성** | 단일 머신 제한 | Fargate 자동 스케일링 |
| **고가용성** | 없음 | Multi-AZ 지원 |
| **권장 용도** | 개발, 테스트, 데모 | Testnet, Mainnet 프로덕션 |

### 환경 선택 가이드

```
개발/테스트 목적?
  ├─ Yes → 빠른 반복이 필요한가?
  │         ├─ Yes → Local Devnet (자체 L1 포함)
  │         └─ No  → Local Network (실제 L1 연결)
  └─ No  → 프로덕션 배포
            ├─ Testnet → AWS (testnet)
            └─ Mainnet → AWS (mainnet, 백업 자동 활성화)
```

---

## 인프라 제거

### Local 환경 제거

```bash
trh-sdk destroy
```

내부적으로 실행되는 명령:

```bash
docker compose -f docker-compose.local.yml down -v --remove-orphans
```

- 모든 컨테이너 중지 및 제거
- Docker 볼륨 삭제 (체인 데이터 포함)
- 네트워크 제거

### AWS 환경 제거

```bash
trh-sdk destroy
```

**제거 프로세스:**

```
1. AWS Backup 리소스 정리
2. Helm 릴리스 제거 (모든 포드 삭제)
3. 모니터링/Uptime 서비스 제거
4. K8s 네임스페이스 삭제
5. EFS 마운트 타겟 삭제 (30초 대기)
6. Terraform destroy (thanos-stack)
   └─ EKS, VPC, EFS, RDS, S3, Secrets Manager 삭제
7. Terraform destroy (backend)
   └─ S3 상태 버킷, DynamoDB 잠금 테이블 삭제
8. 리소스 정리 검증
```

> **주의**: AWS 인프라 제거는 되돌릴 수 없습니다. 체인 데이터(EFS)와 데이터베이스(RDS)가 모두 삭제됩니다.

---

## 설정 참조

### CLI 플래그 전체 목록

| 플래그 | 환경변수 | 기본값 | 설명 |
|--------|---------|-------|------|
| `--network` | `TRH_SDK_NETWORK` | `testnet` | 네트워크 (`devnet`, `testnet`, `mainnet`) |
| `--stack` | `TRH_SDK_STACK` | `thanos` | 스택 (`thanos`) |
| `--preset` | `TRH_SDK_PRESET` | (대화형) | 프리셋 |
| `--fee-token` | `TRH_SDK_FEE_TOKEN` | (대화형) | 수수료 토큰 |
| `--no-candidate` | `TRH_SDK_NO_CANDIDATE` | `false` | 후보자 등록 스킵 |
| `--enable-fault-proof` | `TRH_SDK_ENABLE_FAULT_PROOF` | `false` | Fault Proof 활성화 |
| `--reuse-deployment` | - | `true` | 기존 배포 재사용 |

### settings.json 구조

```jsonc
{
  // 운영자 키
  "admin_private_key": "0x...",
  "sequencer_private_key": "0x...",
  "batcher_private_key": "0x...",
  "proposer_private_key": "0x...",
  "challenger_private_key": "0x...",       // Fault Proof 시

  // L1 연결
  "l1_rpc_url": "https://sepolia.infura.io/v3/...",
  "l1_beacon_url": "https://...",
  "l1_rpc_provider": "infura",
  "l1_chain_id": 11155111,
  "l2_chain_id": 12345,

  // 배포 설정
  "stack": "thanos",
  "network": "testnet",
  "preset": "defi",
  "fee_token": "TON",
  "enable_fraud_proof": false,

  // 체인 설정
  "chain_configuration": {
    "batch_submission_frequency": 1440,
    "challenge_period": 12,
    "output_root_frequency": 240,
    "l2_block_time": 2,
    "l1_block_time": 12
  },

  // 배포 상태
  "deploy_contract_state": {
    "status": 2                             // 1=InProgress, 2=Completed
  },
  "deployment_file_path": "path/to/deploy.json",

  // AWS 설정 (AWS 환경만)
  "aws": {
    "region": "us-east-1",
    "access_key_id": "...",
    "secret_access_key": "..."
  },

  // K8s 설정 (AWS 환경만)
  "k8s": {
    "namespace": "my-chain"
  },

  // 체인 정보
  "chain_name": "my-chain",
  "backup_config": {
    "enabled": true
  }
}
```

### 수수료 토큰 L1 주소

**Sepolia:**

| 토큰 | L1 주소 |
|------|---------|
| TON | `0xa30fe40285B8f5c0457DbC3B7C8A280373c40044` |
| ETH | `0x0000000000000000000000000000000000000000` |
| USDT | `0xaa8e23fb1079ea71e0a56f48a2aa51851d8433d0` |
| USDC | `0x1c7d4b196cb0c7b01d743fbc6116a902379c7238` |

### 체인 설정 기본값

| 설정 | Sepolia | Mainnet |
|------|---------|---------|
| L2 Block Time | 2초 | 2초 |
| L1 Block Time | 12초 | 12초 |
| Finalization Period | 12초 | 604800초 (7일) |
| L2 Output Oracle Submission Interval | 120 블록 | 10800 블록 |
| Max Channel Duration | 120 블록 | 1500 블록 |

### 빠른 참조: 전체 배포 명령어

```bash
# ============================================================
# Local 환경
# ============================================================

# Local Devnet (자체 L1 포함, 완전 독립)
trh-sdk deploy-contracts --network devnet --stack thanos --preset general --fee-token ETH
trh-sdk deploy

# Local + Sepolia (실제 L1 연결, DeFi)
trh-sdk deploy-contracts --network testnet --stack thanos --preset defi --fee-token TON
trh-sdk deploy  # → localhost 선택

# Local + Fault Proof (Full 프리셋)
trh-sdk deploy-contracts --network testnet --stack thanos --preset full --fee-token ETH --enable-fault-proof
trh-sdk deploy  # → localhost 선택

# ============================================================
# AWS 환경
# ============================================================

# AWS Testnet (General, 최소 비용)
trh-sdk deploy-contracts --network testnet --stack thanos --preset general --fee-token TON
trh-sdk deploy  # → AWS, region, chain name 입력

# AWS Testnet (DeFi + USDC)
trh-sdk deploy-contracts --network testnet --stack thanos --preset defi --fee-token USDC
trh-sdk deploy  # → AWS 선택

# AWS Testnet (Full + Fault Proof)
trh-sdk deploy-contracts --network testnet --stack thanos --preset full --fee-token ETH --enable-fault-proof
trh-sdk deploy  # → AWS 선택

# AWS Mainnet (프로덕션, 백업 자동 활성화)
trh-sdk deploy-contracts --network mainnet --stack thanos --preset full --fee-token TON --enable-fault-proof
trh-sdk deploy  # → AWS 선택, 백업 자동 활성화

# ============================================================
# 인프라 제거
# ============================================================
trh-sdk destroy
```
