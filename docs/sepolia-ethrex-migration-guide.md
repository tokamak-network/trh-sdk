# Sepolia Execution Client Migration: op-geth -> py-ethclient(ethrex)

## 1. Overview

This document describes the complete plan for replacing the default execution client (`op-geth`) with `py-ethclient` (a Python port of ethrex) when deploying a Sepolia-based L2 using `trh-sdk`.

- Repositories involved
  - `trh-sdk`: https://github.com/tokamak-network/trh-sdk
  - `py-ethclient`: https://github.com/tokamak-network/py-ethclient
  - `tokamak-thanos-stack`: https://github.com/tokamak-network/tokamak-thanos-stack
  - `tokamak-thanos-geth`: https://github.com/tokamak-network/tokamak-thanos-geth (reference)
- Target environment
  - L1: Ethereum Sepolia
  - L2 Stack: Thanos Stack (`trh-sdk deploy-contracts`, `trh-sdk deploy`)
- Strategy: **Strategy A** — Keep Kubernetes resource names as `op-geth`, only swap Docker image and entrypoint

---

## 2. Current Architecture — Image Pipeline

Understanding the full end-to-end image pipeline is essential before making changes.

```
trh-sdk                                       tokamak-thanos-stack
============                                  ====================

docker_images.go                              var_global.tf
  OpGethImageTag: "f8c04dcb"                    variable "stack_op_geth_image_tag"
       |                                              |
  deploy_chain.go / update_network.go           k8s.tf (terraform_data.thanos_stack_values)
       |                                              |
  input.go                                    generate-thanos-stack-values.sh
    .envrc: TF_VAR_stack_op_geth_image_tag          |
       |                                        Constructs full image string:
       +------> Terraform apply ------->        "tokamaknetwork/thanos-op-geth:nightly-{tag}"
                                                      |
                                                thanos-stack-values.yaml
                                                  op_geth.image: "tokamaknetwork/thanos-op-geth:nightly-f8c04dcb"
                                                      |
                                                Helm install/upgrade
                                                      |
                                                StatefulSet
                                                  image: {{ .Values.op_geth.image }}
                                                  command: ["/bin/sh", "/op-geth-scripts/entrypoint-op-geth.sh"]
```

**Key insight**: The image repository (`tokamaknetwork/thanos-op-geth`) and tag prefix (`nightly-`) are hardcoded in `generate-thanos-stack-values.sh`, not in `trh-sdk`. Swapping to py-ethclient requires changes in **both** repositories.

---

## 3. Current op-geth Entrypoint Analysis

The execution client is started via a ConfigMap-mounted shell script, not the Docker image's own entrypoint.

**File**: `tokamak-thanos-stack/charts/thanos-stack/files/op-geth/entrypoint-op-geth.sh`

```bash
#!/bin/sh
set -exu

# 1. Download genesis file
wget "${GENESIS_FILE_URL}" -q -O genesis.json

# 2. Initialize chaindata if not present
if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
    geth --verbosity="$VERBOSITY" init --datadir="$GETH_DATA_DIR" "$GENESIS_FILE_PATH"
fi

# 3. Start geth with L2-specific flags
exec geth \
    --datadir="$GETH_DATA_DIR" \
    --http --http.addr=0.0.0.0 --http.port="$RPC_PORT" \
    --http.api=web3,debug,eth,txpool,net,engine \
    --ws --ws.addr=0.0.0.0 --ws.port="$WS_PORT" \
    --ws.api=debug,eth,txpool,net,engine \
    --syncmode=full --nodiscover --maxpeers=0 \
    --networkid=$CHAIN_ID \
    --authrpc.addr="0.0.0.0" --authrpc.port="8551" \
    --authrpc.vhosts="*" --authrpc.jwtsecret=/op-geth-auth/jwt.txt \
    --gcmode=archive \
    --metrics --metrics.addr=0.0.0.0 --metrics.port=6060 \
    --rollup.disabletxpoolgossip=true \
    --rpc.batch-request-limit=1000000 \
    --rpc.batch-response-max-size=25000000000 \
    "$@"
```

**Critical interfaces exposed by op-geth**:

| Port | Protocol | Purpose | Used By |
|------|----------|---------|---------|
| 8545 | HTTP | JSON-RPC (eth, net, web3, engine, debug, txpool) | Block Explorer, Bridge, External users |
| 8546 | WebSocket | JSON-RPC (eth, net, engine, debug, txpool) | Real-time subscriptions |
| 8551 | HTTP+JWT | Engine API (engine_*) | **op-node** (consensus layer) |
| 30303 | TCP/UDP | P2P | Peer discovery (disabled: `--nodiscover`) |
| 6060 | HTTP | Prometheus metrics | Monitoring stack |

---

## 4. py-ethclient Gap Analysis — Prerequisites

Before any infrastructure changes, `py-ethclient` must implement the following interfaces to be usable as an L2 execution client.

### 4.1 Critical: Engine API (Port 8551)

op-node communicates with the execution client exclusively through the Engine API. Without it, **L2 block production is impossible**.

**Required Engine API methods** (from `tokamak-thanos-geth/eth/catalyst/api.go`):

| Method | Priority | Description |
|--------|----------|-------------|
| `engine_newPayloadV2` | **Must** | Apply new L2 block to engine state |
| `engine_newPayloadV3` | **Must** | Same, with blob support |
| `engine_forkchoiceUpdatedV2` | **Must** | Update canonical chain head, trigger block building |
| `engine_forkchoiceUpdatedV3` | **Must** | Same, with extended attributes |
| `engine_getPayloadV2` | **Must** | Retrieve prepared execution payload |
| `engine_getPayloadV3` | **Must** | Same, with blob data |
| `engine_exchangeTransitionConfigurationV1` | Should | Merge transition config exchange |
| `engine_getPayloadBodiesByHashV1` | Should | Batch payload body retrieval |
| `engine_getPayloadBodiesByRangeV1` | Should | Range-based payload body retrieval |
| `engine_getClientVersionV1` | Nice-to-have | Client identification |

**Current py-ethclient status**: Engine API is **not implemented**. The Dockerfile only exposes ports 30303 and 8545.

### 4.2 Critical: JWT Authentication

Engine API (port 8551) requires JWT authentication. op-node presents a JWT token derived from a shared 32-byte hex secret file (`jwt.txt`).

**Requirements**:
- Read JWT secret from a file path (CLI flag: `--authrpc-jwtsecret`)
- Validate JWT `iat` (issued-at) claim within +/- 60 seconds
- Reject unauthenticated Engine API requests

**Current py-ethclient status**: JWT authentication is **not implemented**.

### 4.3 Required: JSON-RPC API Completeness

op-geth exposes these API namespaces via `--http.api=web3,debug,eth,txpool,net,engine`:

| Namespace | Key Methods | py-ethclient Status |
|-----------|-------------|---------------------|
| `eth_*` | blockNumber, getBalance, getBlockByNumber, call, estimateGas, sendRawTransaction, getLogs, getTransactionReceipt | Partially implemented |
| `net_*` | version, listening, peerCount | Needs verification |
| `web3_*` | clientVersion, sha3 | Needs verification |
| `debug_*` | traceTransaction, traceBlockByNumber | Not implemented (needed for block explorer) |
| `txpool_*` | status, content, inspect | Not implemented |
| `engine_*` | (See 4.1 above) | **Not implemented** |

### 4.4 Required: Genesis Initialization

op-geth uses a two-step process:
1. `geth init --datadir=/db genesis.json` (creates chaindata directory structure)
2. `geth --datadir=/db ...` (starts the node)

py-ethclient uses:
- `python3 -m ethclient.main --genesis genesis.json --data-dir /db`

The initialization must be compatible with the genesis.json format produced by `trh-sdk deploy-contracts`.

### 4.5 Recommended: Metrics Endpoint (Port 6060)

Prometheus scrapes metrics from port 6060. Without it:
- `OpGethDown` alert will fire (Prometheus job `op-geth` reports no target)
- CloudWatch log sidecar may still work (collects stdout/stderr)
- Grafana dashboards will show no execution client metrics

**Mitigation if not implemented**: Modify Prometheus alert rules to suppress execution client metric-based alerts temporarily.

### 4.6 Recommended: Archive Mode

op-geth runs with `--gcmode=archive` to retain all historical state. If py-ethclient does not support archive mode:
- Historical `eth_getBalance`, `eth_call` at past blocks will fail
- Block explorer historical data will be incomplete

### 4.7 Implementation Priority Order

```
[Must - Blocks deployment]
  1. Engine API (engine_newPayloadV3, engine_forkchoiceUpdatedV3, engine_getPayloadV3)
  2. JWT authentication on authrpc port 8551
  3. Genesis initialization compatibility

[Should - Blocks full functionality]
  4. Complete eth_* namespace (getLogs, getTransactionReceipt, etc.)
  5. debug_* namespace (block explorer needs traceTransaction)
  6. Archive mode support

[Nice-to-have - Blocks monitoring]
  7. Prometheus metrics endpoint (port 6060)
  8. txpool_* namespace
  9. WebSocket support (port 8546)
```

---

## 5. Changes Required — Repository by Repository

### 5.1 trh-sdk (5 files)

Changes are minimal — add image repository support alongside existing tag support.

#### 5.1.1 `pkg/constants/docker_images.go`
Add `OpGethImageRepo` field to the struct. Set py-ethclient repo for Testnet, empty for Mainnet (empty = use default op-geth).

```go
var DockerImageTag = map[string]struct {
    OpGethImageRepo     string // NEW: execution client image repository
    OpGethImageTag      string
    ThanosStackImageTag string
}{
    Testnet: {
        OpGethImageRepo:     "py-ethclient",  // ECR URI, determined after image build
        OpGethImageTag:      "latest",         // py-ethclient image tag
        ThanosStackImageTag: "80a6da51",
    },
    Mainnet: {
        OpGethImageRepo:     "",  // empty = default tokamaknetwork/thanos-op-geth
        OpGethImageTag:      "a7c74c7e",
        ThanosStackImageTag: "49e37d47",
    },
}
```

#### 5.1.2 `pkg/types/terraform.go`
Add `OpGethImageRepo string` to both `TerraformEnvConfig` and `UpdateTerraformEnvConfig`.

#### 5.1.3 `pkg/stacks/thanos/input.go` (2 locations)
- `makeTerraformEnvFile` (near line 1506): Conditionally write `TF_VAR_stack_op_geth_image_repo` when not empty
- `updateTerraformEnvFile` (near line 1573): Conditionally add to `newValues` map

#### 5.1.4 `pkg/stacks/thanos/deploy_chain.go` (near line 184)
Pass `OpGethImageRepo` from constants.

#### 5.1.5 `pkg/stacks/thanos/update_network.go` (near line 63)
Pass `OpGethImageRepo` from constants.

#### Files NOT changed (Strategy A — resource names preserved)
- `monitoring.go` — CoreComponents, CloudWatch, pod patterns
- `backup/restore.go`, `backup/attach.go` — PVC/STS defaults
- `block_explorer.go` — service/ingress discovery
- `show_information.go`, `uptime_service.go` — component lists
- `alert_customization.go`, `constants/alerts.go` — alert rules
- `utils/storage.go` — EFS detection patterns

---

### 5.2 tokamak-thanos-stack (6 files)

#### 5.2.1 `terraform/variables/var_global.tf`
Add new variable:
```hcl
variable "stack_op_geth_image_repo" {
  description = "Execution client image repository override (empty = default tokamaknetwork/thanos-op-geth)"
  type        = string
  default     = ""
}
```

#### 5.2.2 `terraform/thanos-stack/modules/kubernetes/variables.tf`
Add same variable at module level.

#### 5.2.3 `terraform/thanos-stack/modules/kubernetes/k8s.tf` (line ~183-198)
Add to `terraform_data.thanos_stack_values` environment block:
```hcl
stack_op_geth_image_repo = var.stack_op_geth_image_repo
```

#### 5.2.4 `terraform/thanos-stack/scripts/generate-thanos-stack-values.sh`
Modify image construction logic:

```bash
# Near line 87
op_geth_image_tag="$stack_op_geth_image_tag"
op_geth_image_repo="${stack_op_geth_image_repo:-}"

# Determine client type based on repo override
if [ -n "$op_geth_image_repo" ]; then
  op_geth_image="${op_geth_image_repo}:${op_geth_image_tag}"
  op_geth_client_type="py-ethclient"
else
  op_geth_image="tokamaknetwork/thanos-op-geth:nightly-${op_geth_image_tag}"
  op_geth_client_type="geth"
fi
```

In YAML generation:
```yaml
op_geth:
  client_type: "$op_geth_client_type"
  image: "$op_geth_image"
```

#### 5.2.5 `charts/thanos-stack/files/op-geth/entrypoint-py-ethclient.sh` (NEW)
New entrypoint script for py-ethclient:

```bash
#!/bin/sh
set -exu

# Download genesis file (same as op-geth entrypoint)
wget "${GENESIS_FILE_URL}" -q -O genesis.json

# Read settings from environment variables (shared with op-geth ConfigMap)
DATA_DIR="${GETH_DATA_DIR:-/db}"
RPC_PORT="${RPC_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"

exec python3 -m ethclient.main \
    --genesis genesis.json \
    --data-dir "$DATA_DIR" \
    --rpc-port "$RPC_PORT" \
    --port 30303 \
    --authrpc-port 8551 \
    --authrpc-jwtsecret /op-geth-auth/jwt.txt \
    --log-level "$LOG_LEVEL" \
    --sync-mode full \
    "$@"
```

> **Note**: CLI flags (`--authrpc-port`, `--authrpc-jwtsecret`) must match py-ethclient's actual implementation. Finalize after Engine API is implemented.

#### 5.2.6 Helm Template Changes

**`charts/thanos-stack/values.yaml`** — Add `client_type`:
```yaml
op_geth:
  client_type: "geth"  # "geth" or "py-ethclient"
  image: ""
```

**`templates/op-geth-cm.yaml`** — Conditional script loading:
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "thanos-stack.fullname" . }}-op-geth-scripts
data:
{{- if eq .Values.op_geth.client_type "py-ethclient" }}
{{- (.Files.Glob "files/op-geth/entrypoint-py-ethclient.sh").AsConfig | nindent 2}}
{{- else }}
{{- (.Files.Glob "files/op-geth/entrypoint-op-geth.sh").AsConfig | nindent 2}}
{{- end }}
```

**`templates/op-geth-statefulset.yaml`** — Conditional command:
```yaml
{{- if eq .Values.op_geth.client_type "py-ethclient" }}
          command: ["/bin/sh", "/op-geth-scripts/entrypoint-py-ethclient.sh"]
{{- else }}
          command: ["/bin/sh", "/op-geth-scripts/entrypoint-op-geth.sh"]
{{- end }}
```

---

### 5.3 py-ethclient (Prerequisites — separate development)

See Section 4 for the full gap analysis. The minimum viable changes:

1. Implement Engine API methods (engine_newPayloadV3, engine_forkchoiceUpdatedV3, engine_getPayloadV3)
2. Add JWT authentication on a separate authrpc port (8551)
3. Add CLI flags: `--authrpc-port`, `--authrpc-jwtsecret`
4. Update Dockerfile to expose ports: `EXPOSE 30303/tcp 30303/udp 8545/tcp 8551/tcp`
5. Verify genesis.json initialization compatibility with trh-sdk output format

---

## 6. Entrypoint Script Comparison

| Aspect | op-geth (entrypoint-op-geth.sh) | py-ethclient (entrypoint-py-ethclient.sh) |
|--------|----------------------------------|-------------------------------------------|
| Binary | `geth` (Go, single static binary) | `python3 -m ethclient.main` (Python, interpreted) |
| Genesis init | `geth init --datadir=/db genesis.json` | `--genesis genesis.json` (flag-based) |
| Chaindata check | `if [ ! -d "$GETH_CHAINDATA_DIR" ]` | Handled internally by py-ethclient |
| HTTP RPC | `--http --http.addr=0.0.0.0 --http.port=8545` | `--rpc-port 8545` |
| Engine API | `--authrpc.addr=0.0.0.0 --authrpc.port=8551 --authrpc.jwtsecret=...` | `--authrpc-port 8551 --authrpc-jwtsecret ...` (TBD) |
| P2P | `--nodiscover --maxpeers=0` | `--port 30303` (P2P config TBD) |
| Sync mode | `--syncmode=full` | `--sync-mode full` |
| Archive mode | `--gcmode=archive` | Not yet supported |
| Metrics | `--metrics --metrics.addr=0.0.0.0 --metrics.port=6060` | Not yet supported |
| OP Stack flags | `--rollup.disabletxpoolgossip=true` | N/A (py-ethclient has no txpool gossip) |
| Batch RPC limits | `--rpc.batch-request-limit=1000000` | Not yet supported |
| Environment vars | `GETH_VERBOSITY`, `GETH_DATA_DIR`, `CHAIN_ID`, `RPC_PORT`, `WS_PORT`, `GENESIS_FILE_URL` | Reuses same env vars via entrypoint script |

---

## 7. Risk Analysis and Mitigation

### Risk 1 (Critical): Engine API Not Implemented
- **Impact**: op-node cannot communicate with execution client. L2 block production fails completely.
- **Probability**: Certain until implemented.
- **Mitigation**:
  - Implement Engine API in py-ethclient **before** any deployment attempt.
  - Test with a standalone op-node in devnet mode first.
  - Verify the full forkchoiceUpdated → getPayload → newPayload cycle.

### Risk 2 (Critical): JWT Authentication Not Implemented
- **Impact**: op-node's Engine API requests are rejected (401 Unauthorized).
- **Probability**: Certain until implemented.
- **Mitigation**:
  - Implement JWT validation in py-ethclient's authrpc handler.
  - Test with the same `jwt.txt` file used in the Helm chart (hardcoded 32-byte hex).

### Risk 3 (High): Data Directory Incompatibility
- **Impact**: op-geth stores data in `/db/geth/chaindata/`. py-ethclient may use a different structure.
- **Probability**: High — different implementations use different storage layouts.
- **Mitigation**:
  - For new deployments: no issue (fresh PVC).
  - For existing deployments: do NOT reuse op-geth PVC data. Create a new PVC or wipe the existing subpath.
  - Document that migration from op-geth to py-ethclient requires chain resync.

### Risk 4 (Medium): Incomplete RPC API Coverage
- **Impact**: Block explorer, bridge, and external tools fail on unsupported RPC methods.
- **Probability**: Medium — py-ethclient implements core eth_* methods but may miss edge cases.
- **Mitigation**:
  - Run RPC regression tests comparing op-geth and py-ethclient responses.
  - Test with Blockscout (block explorer) specifically — it uses `debug_traceTransaction`.
  - Maintain a list of failing methods and implement progressively.

### Risk 5 (Medium): No Metrics Endpoint
- **Impact**: Prometheus `OpGethDown` alert fires. Grafana dashboards show no data.
- **Probability**: Certain until metrics endpoint is implemented.
- **Mitigation**:
  - Temporarily disable the `OpGethDown` Prometheus alert rule.
  - CloudWatch log collection still works (collects container stdout/stderr).
  - Implement a basic `/metrics` endpoint in py-ethclient as a follow-up.

### Risk 6 (Low): Mainnet Regression
- **Impact**: Mainnet deployment accidentally uses py-ethclient image.
- **Probability**: Very low — design uses conditional logic (empty repo = default op-geth).
- **Mitigation**:
  - Mainnet `OpGethImageRepo` is empty string `""` → `TF_VAR_stack_op_geth_image_repo` is not set → `generate-thanos-stack-values.sh` defaults to `tokamaknetwork/thanos-op-geth`.
  - Verify with automated test that Mainnet .envrc does not contain `TF_VAR_stack_op_geth_image_repo`.

---

## 8. Implementation Phases

### Phase 0: Documentation (this document)
- Documented all changes across 3 repositories.
- Identified py-ethclient prerequisites.
- Created risk mitigation strategies.

### Phase 1: py-ethclient Development (separate repo, separate team)
1. Implement Engine API (engine_newPayloadV3, engine_forkchoiceUpdatedV3, engine_getPayloadV3)
2. Implement JWT authentication on authrpc port 8551
3. Add CLI flags: `--authrpc-port`, `--authrpc-jwtsecret`, `--data-dir`, `--genesis`
4. Update Dockerfile to expose port 8551
5. Build Docker image and push to ECR
6. Test Engine API cycle with standalone op-node

### Phase 2: tokamak-thanos-stack Changes
1. Add `stack_op_geth_image_repo` Terraform variable (var_global.tf, variables.tf)
2. Pass variable through k8s.tf to generate script
3. Modify `generate-thanos-stack-values.sh` to support image repo override + client_type
4. Create `entrypoint-py-ethclient.sh` in Helm chart files
5. Add `client_type` to values.yaml
6. Add conditional branching in Helm templates (ConfigMap, StatefulSet)

### Phase 3: trh-sdk Changes
1. `docker_images.go` — Add `OpGethImageRepo` field
2. `terraform.go` — Add field to both config structs
3. `input.go` — Conditionally output `TF_VAR_stack_op_geth_image_repo`
4. `deploy_chain.go` — Pass `OpGethImageRepo`
5. `update_network.go` — Pass `OpGethImageRepo`

### Phase 4: Integration Testing
1. Build py-ethclient Docker image → push to ECR
2. Deploy testnet with `trh-sdk deploy` (Sepolia)
3. Verify L2 block production
4. Run RPC regression tests
5. Test monitoring, backup, block explorer integration

---

## 9. Verification Checklist

### Build Verification
```bash
# trh-sdk
cd trh-sdk && go build ./... && go test ./...

# tokamak-thanos-stack
cd tokamak-thanos-stack && helm lint charts/thanos-stack/
```

### Backward Compatibility Verification (Mainnet)
- [ ] Mainnet .envrc does NOT contain `TF_VAR_stack_op_geth_image_repo`
- [ ] `generate-thanos-stack-values.sh` with empty repo produces `tokamaknetwork/thanos-op-geth:nightly-*` image
- [ ] `client_type` defaults to `"geth"`, existing entrypoint used
- [ ] All existing `trh-sdk` commands work without regression

### Functional Verification (Sepolia)
- [ ] py-ethclient Docker image builds successfully
- [ ] `trh-sdk deploy-contracts --network testnet --stack thanos` succeeds
- [ ] `trh-sdk deploy` succeeds
- [ ] `trh-sdk info` shows healthy status
- [ ] L2 blocks are being produced (Engine API working)
- [ ] JSON-RPC endpoint responds correctly (eth_blockNumber, eth_getBalance)
- [ ] op-node logs show successful Engine API communication
- [ ] op-batcher submits batches successfully
- [ ] op-proposer submits output roots successfully

### Operational Verification
- [ ] CloudWatch log collection works for execution client pod
- [ ] `trh-sdk install monitoring` completes successfully
- [ ] `backup-manager --attach` paths resolve correctly
- [ ] Block explorer connects and indexes blocks
- [ ] Deposit (L1→L2) transaction works
- [ ] Withdrawal (L2→L1) transaction works

### Rollback Procedure
If issues are found:
1. In `trh-sdk/pkg/constants/docker_images.go`: revert `OpGethImageRepo` to `""` and `OpGethImageTag` to original value
2. In `tokamak-thanos-stack/generate-thanos-stack-values.sh`: empty repo triggers default op-geth image
3. Run `terraform apply` + `helm upgrade` to redeploy with op-geth
4. No data migration needed — op-geth resumes from PVC data

---

## 10. File Reference

### trh-sdk
| File | Change |
|------|--------|
| `pkg/constants/docker_images.go` | Add `OpGethImageRepo` field |
| `pkg/types/terraform.go` | Add `OpGethImageRepo` to structs |
| `pkg/stacks/thanos/input.go` | Conditionally output TF_VAR_stack_op_geth_image_repo |
| `pkg/stacks/thanos/deploy_chain.go` | Pass `OpGethImageRepo` |
| `pkg/stacks/thanos/update_network.go` | Pass `OpGethImageRepo` |

### tokamak-thanos-stack
| File | Change |
|------|--------|
| `terraform/variables/var_global.tf` | Add `stack_op_geth_image_repo` variable |
| `terraform/thanos-stack/modules/kubernetes/variables.tf` | Add variable (module level) |
| `terraform/thanos-stack/modules/kubernetes/k8s.tf` | Pass to generate script |
| `terraform/thanos-stack/scripts/generate-thanos-stack-values.sh` | Image repo override + client_type |
| `charts/thanos-stack/files/op-geth/entrypoint-py-ethclient.sh` | NEW: py-ethclient entrypoint |
| `charts/thanos-stack/values.yaml` | Add `client_type` |
| `charts/thanos-stack/templates/op-geth-cm.yaml` | Conditional script loading |
| `charts/thanos-stack/templates/op-geth-statefulset.yaml` | Conditional command |

### py-ethclient (Prerequisites)
| Feature | Status |
|---------|--------|
| Engine API (engine_*) | Not implemented — **MUST** before deployment |
| JWT Authentication (8551) | Not implemented — **MUST** before deployment |
| Dockerfile port 8551 | Not exposed — **MUST** update |
| debug_* RPC namespace | Not implemented — needed for block explorer |
| Prometheus metrics (6060) | Not implemented — needed for monitoring |
| Archive mode | Unknown — needed for historical queries |
