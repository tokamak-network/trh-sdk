# TRH-SDK Documentation

## Table of Contents

- [DRB (Distributed Random Beacon)](#drb-distributed-random-beacon)
- [Recent Changes](#recent-changes)

---

## DRB (Distributed Random Beacon)

DRB is a plugin for deploying and managing Distributed Random Beacon nodes. It uses the Commit-Reveal2 protocol (CommitReveal2L2 contract) and can run independently without an existing L2 chain deployment.

### Architecture

| Node Type | Infrastructure | Role |
|-----------|----------------|------|
| **Leader** | EKS (Kubernetes) | Deploys contracts, coordinates regular nodes, owns the CommitReveal2L2 contract |
| **Regular** | EC2 (single instance) | Connects to leader, participates in the DRB network |

### Leader Node

The leader node:

- Deploys **Commit-Reveal2** contracts (CommitReveal2L2 or CommitReveal2 on Sepolia/mainnet) to the target chain
- Deploys **ConsumerExampleV2** (optional consumer contract)
- Provisions a dedicated EKS cluster with a separate VPC
- Runs the DRB leader application via Helm on Kubernetes
- Stores connection info in `drb-leader-info.json` for regular nodes to use

**Database options:**

- **AWS RDS PostgreSQL** – Managed database in the same VPC
- **Local PostgreSQL** – Helm chart (in-cluster)

**Prerequisites:**

- AWS credentials
- Leader private key with balance on the target chain (for contract deployment)
- RPC URL(s) for the target network (e.g. L2, Sepolia, mainnet)

### Regular Node

The regular node:

- Connects to the leader via IP, port, and peer ID
- Runs on a single EC2 instance (t3.small by default)
- Uses Docker Compose to run the DRB application and optionally PostgreSQL
- Requires a `.env` file with leader connection details (from `drb-leader-info.json`)

**Database options:**

- **AWS RDS PostgreSQL** – Managed database
- **Local PostgreSQL** – Docker Compose on the same EC2 instance

**Prerequisites:**

- Leader node deployed first (`trh-sdk install drb --type leader`)
- `.env` file with: `LEADER_IP`, `LEADER_PORT`, `LEADER_PEER_ID`, `LEADER_EOA`, `PORT`, `EOA_PRIVATE_KEY`, `POSTGRES_PORT`, `DRB_NODE_IMAGE`, `CHAIN_ID`, `ETH_RPC_URLS`, `CONTRACT_ADDRESS`
- AWS credentials and EC2 key pair for SSH

### Installation Order

1. Install the leader node: `trh-sdk install drb --type leader`
2. Run `trh-sdk drb leader-info` to get leader connection details
3. Create a `.env` file with the required keys (see leader info output)
4. Install the regular node: `trh-sdk install drb --type regular`

### Leader Info File

After leader deployment, `drb-leader-info.json` is written in the current directory. It includes:

- Leader URL, IP, port, peer ID, EOA
- CommitReveal2L2 and ConsumerExampleV2 contract addresses
- Chain ID, RPC URL
- Cluster name and namespace

View it anytime with:

```bash
trh-sdk drb leader-info
```

---

## Recent Changes

### DRB Plugin CLI Refactor

The DRB plugin now uses a unified `--type` flag instead of separate plugin names.

#### New CLI Commands

| Action | Old Command | New Command |
|--------|-------------|-------------|
| Install leader node | `trh-sdk install drb` | `trh-sdk install drb --type leader` |
| Install regular node | `trh-sdk install drb regular-node` | `trh-sdk install drb --type regular` |
| Uninstall leader node | `trh-sdk uninstall drb` | `trh-sdk uninstall drb --type leader` |
| Uninstall regular node | `trh-sdk uninstall drb regular-node` | `trh-sdk uninstall drb --type regular` |

**Backwards compatibility:** Omitting `--type` defaults to `leader` for both install and uninstall.

#### Examples

```bash
# Install DRB leader node (EKS-based)
trh-sdk install drb --type leader

# Install DRB regular node (EC2-based)
trh-sdk install drb --type regular

# Uninstall DRB leader node
trh-sdk uninstall drb --type leader

# Uninstall DRB regular node
trh-sdk uninstall drb --type regular

# View leader connection info
trh-sdk drb leader-info
```

### Plugins That Work Without a Chain

DRB (leader and regular) can be installed without an existing chain deployment. The logic is extensible: additional plugins can be added to `PluginsThatWorkWithoutChain` in `pkg/constants/plugins.go` when they support standalone deployment.


