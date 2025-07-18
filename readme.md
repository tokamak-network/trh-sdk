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
- L1 PRC URL (You can can get it from [Alchemy](https://www.alchemy.com/), [Infura](https://infura.io/), [QuickNode](https://www.quicknode.com/), etc.)
- Beacon Chain RPC URL (You can can get it from [QuickNode](https://www.quicknode.com/))
- Prepare AWS credentials & configuration to access AWS EKS.
  - [What is IAM?](https://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html) (\*_Note: This IAM user has to have the following policies_)
    - `arn:aws:iam::aws:policy/aws-service-role/AmazonEKSServiceRolePolicy`
  - [How to create aws access key and secret key for a IAM user](https://repost.aws/knowledge-center/create-access-key).
- Prepare seed phrase for the L1 account.

### Deploy L1 contracts

> To deploy the testnet and mainnet network, we must deploy the L1 contracts first

The first step is to deploy the L1 contracts to the L1 network. The output of this step is we generate the rollup, genesis file, and deployment file.


```bash
trh-sdk deploy-contracts --network [] --stack []
```

Example:

```bash
trh-sdk deploy-contracts --network testnet --stack thanos
```

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

Same as the deploy infra command, this command looks the config files located at the current directory to choose the network and stack

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
trh-sdk install bridge
# Uninstall block explorer
trh-sdk install block-explorer
# Uninstall monitoring plugin
trh-sdk install monitoring
```

### Get the chain information
After deploying the chain successfully, we can get the chain information by:
```bash
trh-sdk info
```

### Alert Configuration

The SDK provides comprehensive alert configuration capabilities for monitoring your Thanos Stack deployment.

#### Quick Start
```bash
# Check current alert status
trh-sdk alert-config --status

# Configure email alerts
trh-sdk alert-config --channel email --configure

# Configure telegram alerts  
trh-sdk alert-config --channel telegram --configure

# Configure alert rules interactively
trh-sdk alert-config --rule set

# Reset all rules to default values
trh-sdk alert-config --rule reset
```

#### Features
- **Email & Telegram Notifications**: Set up multiple notification channels
- **Configurable Alert Rules**: Adjust thresholds for balance, CPU, memory, and more
- **Interactive Configuration**: User-friendly command-line interface
- **Status Monitoring**: Real-time alert status and configuration details