# Tokamak Rollup Hub SDK

## Introduction

The tokamak rollup hub SDK allows anyone to quickly deploy customized and autonomous Layer 2 Rollups on the Ethereum network.

## Set up the SDK

1. Download the setup.sh file

   ```bash
   wget https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/setup.sh
   ```

2. Run the setup.sh file

   ```bash
   chmod +x setup.sh
   ./setup.sh
   ```

3. Source the shell config
    
    First, get your current shell by:
    ```bash
    echo $SHELL
    ```

   - if the output is `/bin/zsh`

   ```bash
   source ~/.zshrc
   ```

   - if the output is `/bin/bash`

   ```bash
   source ~/.bashrc
   ```

4. Verify the installation

   ```bash
   trh-sdk version
   ```

## Local-Devnet deployment
- Deploy

  ```bash
  trh-sdk deploy
  ```

  If you successfully deploy the local-devnet, you will get the following output:

  ```bash
    ...
    Container ops-bedrock-l1-1  Running
    Container ops-bedrock-l2-1  Running
    Container ops-bedrock-op-node-1  Running
    Container ops-bedrock-op-challenger-1  Recreate
    Container ops-bedrock-op-challenger-1  Recreated
    Container ops-bedrock-op-challenger-1  Starting
    Container ops-bedrock-op-challenger-1  Started
    ✅ Devnet up!
  ```

- Destroy
  ```bash
  trh-sdk destroy
  ```
  If you successfully destroy the local-devnet, you will get the following output:
  ```bash
    Destroying the devnet network...
    ✅ Destroyed the devnet network successfully!
  ```

## Testnet/Mainnet deployment

### Prerequisites

#### AWS (default)
- L1 PRC URL (You can can get it from [Alchemy](https://www.alchemy.com/), [Infura](https://infura.io/), [QuickNode](https://www.quicknode.com/), etc.)
- Beacon Chain RPC URL (You can can get it from [QuickNode](https://www.quicknode.com/))
- Prepare AWS credentials & configuration to access AWS EKS.
  - [What is IAM?](https://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html) (\*_Note: This IAM user has to have the following policies_)
    - `arn:aws:iam::aws:policy/aws-service-role/AmazonEKSServiceRolePolicy`
  - [How to create aws access key and secret key for a IAM user](https://repost.aws/knowledge-center/create-access-key).
- Prepare seed phrase for the L1 account.

#### DigitalOcean
- L1 RPC URL and Beacon Chain RPC URL (same as above)
- **DigitalOcean API Token** — for DOKS cluster and resource provisioning.
  Get it from: [DigitalOcean API settings](https://cloud.digitalocean.com/account/api/tokens)
- **Spaces HMAC credentials** — for Terraform state storage (S3-compatible).
  These are **separate** from the API token. Get them from: DigitalOcean → API → Spaces Keys.
  - `Spaces Access Key` (looks like an AWS access key)
  - `Spaces Secret Key` (entered masked at deploy time; **never saved to disk**)
- [`doctl`](https://docs.digitalocean.com/reference/doctl/how-to/install/) CLI installed and authenticated.
- Prepare seed phrase for the L1 account.

### Deploy L1 contracts

> To deploy the testnet and mainnet network, we must deploy the L1 contracts first

The first step is to deploy the L1 contracts to the L1 network. The output of this step is we generate the rollup, genesis file, and deployment file.

#### Basic Command

```bash
trh-sdk deploy-contracts --network [testnet|mainnet] --stack [thanos] [options]
```

#### Examples

**Basic Deployment (with Candidate Registration)**
```bash
trh-sdk deploy-contracts --network testnet --stack thanos
```

**Deployment without Candidate Registration**
```bash
trh-sdk deploy-contracts --network testnet --stack thanos --no-candidate
```

#### Advanced Options

**`--reuse-deployment`**: Reuse existing deployment artifacts (default: true)
- Skips repository cloning, smart contract building, and contract deployment steps.
- Enabled by default. To perform a full deployment, disable it with `--reuse-deployment=false`.
```bash
# Default behavior (Reuse artifacts)
trh-sdk deploy-contracts --network testnet --stack thanos

# Full deployment (Regenerate artifacts)
trh-sdk deploy-contracts --network testnet --stack thanos --reuse-deployment=false
```

For detailed usage and verification methods, refer to the [Deploy Contracts Guide](docs/deploy-contracts-guide.md).

### Deploy stack
To deploy the testnet/mainnet network, we must deploy the L1 contracts successfully first.
```bash
trh-sdk deploy
```


The deployment config file is located at the deployment folder. `settings.json` file looks like:

```json
{
  "admin_private_key": "your admin private key",
  "sequencer_private_key": "your sequencer private key",
  "batcher_private_key": "your batcher private key",
  "proposer_private_key": "your proposer private key",
  "deployment_path": "./tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/11155111-deploy.json",
  "l1_rpc_url": "your_l1_rpc",
  "l1_beacon_url": "your_l1_beacon_rpc",
  "l1_rpc_provider": "debug_geth",
  "l1_chain_id": 11155111,
  "l2_chain_id": <your_l2_chain_id>,
  "stack": "thanos",
  "network": "testnet",
  "enable_fraud_proof": false,
  "l2_rpc_url": "your_l2_rpc",
  "aws": {
    "secret_key": "your_secret_key",
    "access_key": "your_access_key",
    "region": "your aws region",
    "default_format": "json"
  },
  "k8s": {
    "namespace": "your namespace"
  },
  "chain_name": "your chain name"
}
```


### Destroy the stack

To terminate the network, we can run the command looks like:

```bash
trh-sdk destroy
```

Same as the deploy infra command, this command looks the config files located at the current directory to choose the network and stack.

> **DigitalOcean note**: The Spaces Secret Key is never saved to disk. You will be prompted to re-enter it when running `trh-sdk destroy`.

### Install the plugin
```bash
# Install bridge (Installed by default when deploying L2)
trh-sdk install bridge
# Install block explorer
trh-sdk install block-explorer
# Install monitoring plugin
trh-sdk install monitoring
```

### Uninstall the plugin
```bash
# Uninstall bridge
trh-sdk uninstall bridge
# Uninstall block explorer
trh-sdk uninstall block-explorer
# Uninstall monitoring plugin
trh-sdk uninstall monitoring
```

### Get the chain information
After deploying the chain successfully, we can get the chain information by:
```bash
trh-sdk info
```

## Native Runner Architecture

TRH SDK uses native Go libraries instead of shelling out to external CLI tools. This means **you do not need to pre-install kubectl, helm, aws CLI, or doctl** to run `trh-sdk deploy`.

### How it works

| Tool | Replaced by | Call sites eliminated |
|------|------------|----------------------|
| `kubectl` | `k8s.io/client-go` | 114 |
| `helm` | `helm.sh/helm/v3` | 21 |
| `aws` CLI | `aws-sdk-go-v2` | 78 |
| `terraform` | `hashicorp/terraform-exec` | 6 |
| `doctl` | `digitalocean/godo` | 4 |

### Performance

Native library calls are **~11,000× faster** than fork+exec shell-outs:

| Method | Latency / call | Memory / call |
|--------|---------------|---------------|
| Shell-out | ~1.4 ms | ~12 KB |
| Native | ~0.13 µs | ~340 B |

### Fallback to legacy mode

If you need the old shell-out behaviour for debugging:

```bash
TRHS_LEGACY=1 trh-sdk deploy
```

For detailed analysis and before/after code comparisons, see [docs/runner-comparison.md](docs/runner-comparison.md).

---

## Testing

### Unit tests

```bash
go test ./...
```

No external tools (kubectl, helm, etc.) required — all runners use mock interfaces in tests.

### Integration tests (requires kind)

Integration tests run against a real local Kubernetes cluster via [kind](https://kind.sigs.k8s.io/).

```bash
# 1. Install kind (macOS)
brew install kind

# 2. Create a test cluster
kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig

# 3. Run integration tests
KUBECONFIG=/tmp/trh-test.kubeconfig \
go test -v -tags=integration -timeout=120s ./pkg/runner/
```

### Benchmarks

```bash
# Compile test binary first (avoids output filtering by shell hooks)
go test -c -o /tmp/runner.test ./pkg/runner/

# Run benchmarks with memory stats
/tmp/runner.test -test.bench=. -test.benchmem
```

### Log streaming demo

Demonstrates native vs shell-out log streaming without a real cluster:

```bash
bash demo/run-log-streaming.sh
```

---

## Known Limitations

See [TODO.md](TODO.md) for the full list. Key items:

| # | Issue | Impact |
|---|-------|--------|
| TODO-1 | `terraform` auto-downloaded at runtime if not in PATH | Air-gap deployments may fail |
| TODO-2 | Helm `context.Cancel()` stops Go goroutine but not internal Helm action | Partial deploy state possible on timeout |
| TODO-3 | `extraArgs` not forwarded to `HelmRunner` | Native mode silently ignores extra Helm flags |
| TODO-5 | `kubectl rollout` not in `K8sRunner` interface | Some backup paths still shell out |

---

## Monitoring Plugin

The Monitoring plugin provides comprehensive monitoring, alerting and log collection capabilities for the Thanos Stack. For detailed documentation on monitoring features, including alert customization and log collection management, see the [Monitoring Plugin Documentation](docs/monitoring.md).

### Quick Start
```bash
# Install monitoring plugin
trh-sdk install monitoring

# Configure alerts
trh-sdk alert-config --${flag}

# Manage log collection
trh-sdk log-collection --${flag}

# Uninstall monitoring plugin
trh-sdk uninstall monitoring
```