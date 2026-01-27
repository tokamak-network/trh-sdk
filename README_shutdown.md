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

```bash
./trh-sdk shutdown status
```
-   Displays a summary of currently configured RPC info, execution history, existence of asset files, etc.

---

## 🔒 Implementation Features & Safeguards

1.  **Persistence**: Automatically records the success results of each step in `settings.json` to ensure process continuity.
2.  **Simulation Mode (Dry-Run)**: Supports predicting results and generating Safe transaction hashes via Forge simulation before actual execution.
3.  **Monorepo Integration**: Dynamically detects and executes Forge and Python environments based on `thanos_root`.
