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

> We only need to run this script on the testnet and mainnet network

The first step is to deploy the L1 contracts to the L1 network. The output of this step is we generate the rollup, genesis file, and deployment file.

→ We will create the `settings.json` file after deploying successfully to reuse in the deploy infra step.

`settings.json` file looks like:

```json
{
  "admin_private_key": "012347185fc76118346627e44f8a7e9318dad70544711001...",
  "sequencer_private_key": "eb845bb0aba96394a99bd44163471ea0305cd4880280e0f...",
  "batcher_private_key": "12399af5811bf105f8731d848874d5b7c4be3f69...",
  "proposer_private_key": "9461b46f763b2e68077ea87e76c537bc7488432...",
  "deployment_path": "/home/user/tokamak/trh-sdk/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/11155111-deploy.json",
  "l1_rpc_url": "https://sepolia.rpc.tokamak.network",
  "l1_rpc_provider": "",
  "stack": "thanos",
  "network": "testnet",
  "enable_fraud_proof": false,
  "k8s_namespace": "",
  "helm_release_name": "",
  "l2_rpc_url": ""
}
```

```bash
./trh-sdk deploy-contracts --network [] --stack []
```

Example:

```bash
./trh-sdk deploy-contracts --network testnet --stack thanos
```

### Deploy stack

```bash
./trh-sdk deploy
```

→ If the `settings.json file exists, we will deploy the stack by the network and stack written on the config file. And if the settings.json file doesn't exist. We will deploy the local-devnet network and the stack is Thanos by default.

### Destroy the stack

To terminate the network, we can run the command looks like:

```bash
./trh-sdk destroy
```

Same as the deploy infra command, this command looks the config files located at the current directory to choose the network and stack

### Install the plugin

```bash
./trh-sdk install bridge
```
