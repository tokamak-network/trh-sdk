# Thanos L2 Shutdown Helper Guide

`trh-sdk shutdown` is an orchestration tool designed to automate the secure shutdown of Thanos L2 chains, facilitate liquidity retrieval, and enable force withdrawals of user assets on L1.

## Prerequisites

The following elements must be prepared for all shutdown phases:

1.  **`settings.json`**: The core configuration file for the SDK.
    *   Must contain accurate `l1_rpc_url`, `l2_rpc_url`, `l1_chain_id`, and `l2_chain_id`.
2.  **`tokamak-thanos` Monorepo**: The source for actual smart contracts and execution scripts.
    *   Must include deployment address metadata (`*.json`) for the target network within the `packages/tokamak/contracts-bedrock/deployments/` directory.

---

## 🛠️ CLI Command Details

Operators execute the following commands sequentially from the project root.

### Integrated Run (Execute All Steps)
Automatically executes all shutdown phases from Step 1 to Step 5 in sequence.
```bash
./trh-sdk shutdown run [--dry-run] [--skip-fetch]
```
-   **Sequential Execution**: Performs `Block` → `Fetch` → `Gen` → `Activate` → `Withdraw` without interruption.
-   **`--dry-run` (Recommended)**: Uses Forge's simulation engine to verify the success of the entire scenario without spending actual gas or changing live state.
-   **`--skip-fetch`**: Skips the time-consuming data collection (Step 2) phase if valid asset data files already exist in the `data/` folder.

### Step 1: Block (Pause Deposits/Withdrawals)
Halts L1 bridge functions and records block information at the time of shutdown.
```bash
./trh-sdk shutdown block [--dry-run]
```
-   **Actions**: Upgrades `OptimismPortal` (to Closing mode), pauses `SuperchainConfig`.
-   **Outcome**: Automatically records `l2_start_block` in `settings.json`.

### Step 2: Fetch (Collect Asset Data)
Queries Explorer APIs and node data to collect the state of all assets within L2.
```bash
./trh-sdk shutdown fetch
```
-   **Actions**: Executes Python scripts to collect data on holders, contracts, tokens, and unclaimed withdrawal lists. (Requires Python environment setup)
-   **Outputs**: `l2-holders`, `l2-contracts`, `l2-tokens`, `unclaimed-withdrawals`, `l2-burns` (in JSON format).

### Step 3: Gen (Generate Asset Snapshot)
Validates collected data and generates the final snapshot JSON for registration on L1 contracts.
```bash
./trh-sdk shutdown gen [--l2-start-block <block_number>] [--dry-run]
```
-   **Prerequisites**: Data files generated in Step 2 must exist in the `data/` folder.
-   **Actions**: Verifies on-chain data integrity and generates `generate-assets-*.json` (Final Snapshot).

### Step 4: Activate (Upgrade & Activate Bridge)
Upgrades the L1 bridge to support force withdrawal mode and registers the snapshot to activate it.
```bash
./trh-sdk shutdown activate [--dry-run]
```
-   **Prerequisites**: The result of Step 3 (`generate-assets-*.json`) must be ready.
-   **Outcome**: Bridge state changes to `ACTIVE`.

### Step 5: Withdraw (Settlement & Claims)
Retrieves system liquidity and processes unclaimed withdrawals either in bulk or individually.
```bash
./trh-sdk shutdown withdraw [--storage-address <address>] [--dry-run]
```
-   **Prerequisites**: The result of Step 3 (`generate-assets-*.json`) must be ready.
-   **Actions**: Liquidity sweeping (Sweep), asset transfer based on unclaimed data.

---

## ⚙️ Configuration Details (settings.json)

| Key | Description |
| :--- | :--- |
| `thanos_root` | Absolute path to the `tokamak-thanos` monorepo |
| `deployment_path` | Path where `trh-sdk` or project metadata is stored |
| `network` | Target network name (e.g., `testnet`, `mainnet`) |

---

## 📊 Status Check (Dashboard)

The `status` command provides a comprehensive dashboard of the current shutdown process, including configuration validity, execution history, and resource availability.

```bash
./trh-sdk shutdown status
```

### 1. Output Example

```text
📊 Shutdown Status
==================
Reading config from: /path/to/settings.json

🔧 Current Configuration:
   Deployment Path: /Users/theo/workspace_tokamak/trh-sdk
   SDK Path: /Users/theo/workspace_tokamak/tokamak-thanos/packages/tokamak/sdk
   L1 Chain ID: 11155111
   L2 Chain ID: 111551119090
   Network: testnet
   Thanos Root: /Users/theo/workspace_tokamak/tokamak-thanos/packages
   Deployments Path: 11155111-deploy.json

📜 Execution History:
   Last Command: gen
   Last Gen: 2026-02-02T18:30:00Z
   Last Snapshot: data/generate-assets-111551119090.json
   Last Dry-Run: 2026-02-02T18:35:00Z
   Last Send: (never)

📁 Deployment Contracts:
   ✅ Loaded: 20 contracts available
      Bridge Proxy: 0x1234...abcd

📁 Assets File:
   ✅ Found: data/generate-assets-111551119090.json
   Last modified: 2026-02-02 18:30:00
```

### 2. Execution History & State File

The SDK maintains an execution history to ensure **process continuity** and **prevent redundant operations**. This state is stored locally in `~/.trh/thanos_shutdown_state.json`.

**Why it is stored:**
- **Resume Capability:** Allows operators to pick up where they left off (e.g., after `fetch` or `gen`).
- **Audit:** Records when critical actions (Dry-Run, Send) were last performed.
- **Path Caching:** Remembers the location of the generated asset snapshot for subsequent steps.

**State File Example (`~/.trh/thanos_shutdown_state.json`):**
```json
{
  "chainId": 11155111,
  "l2ChainId": 111551119090,
  "thanosRoot": "/Users/theo/workspace_tokamak/tokamak-thanos/packages",
  "deploymentsPath": "11155111-deploy.json",
  "dataDir": "",
  "lastGenAt": "2026-02-02T18:30:00Z",
  "lastDryRunAt": "2026-02-02T18:35:00Z",
  "lastSnapshotPath": "data/generate-assets-111551119090.json",
  "lastCommand": "dry-run"
}
```

---

## 🔒 Implementation Features & Safeguards

1.  **Persistence**: Automatically records the success results of each step in `settings.json` to ensure process continuity.
2.  **Simulation Mode (Dry-Run)**: Supports predicting results and generating Safe transaction hashes via Forge simulation before actual execution.
3.  **Monorepo Integration**: Dynamically detects and executes Forge and Python environments based on `thanos_root`.
