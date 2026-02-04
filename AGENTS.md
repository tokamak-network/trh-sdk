# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Tokamak Rollup Hub SDK** is a Go CLI tool for deploying customized Layer 2 Rollups on Ethereum. It manages the complete deployment lifecycle: contract deployment on L1, infrastructure setup on AWS EKS, and L2 network configuration.

### Key Technologies
- **Language**: Go 1.24.11 (required - enforced in main.go)
- **CLI Framework**: urfave/cli/v3 (beta)
- **Blockchain**: Ethereum go-ethereum client
- **Cloud**: AWS SDK v2 (EKS, S3, KMS)
- **Container Orchestration**: Kubernetes via Terraform
- **Logging**: go.uber.org/zap

## Architecture & Key Concepts

### Command Structure (cli.go)
- `cli.go` (427 lines): Main CLI router registering all commands
- `commands/`: Individual command implementations (deploy, destroy, deploy-contracts, etc.)
- `flags/`: Centralized flag definitions used across commands

**Pattern**: Each command in `cli.go` maps to an action handler in `commands/` that parses flags and delegates to stack implementations.

### Stack System (pkg/stacks/)
The SDK supports multiple stack types (currently Thanos). Each stack is a complete deployment profile.

**Key Files**:
- `pkg/stacks/thanos/deploy_contracts.go`: L1 contract deployment with resumable state
  - Handles artifact reuse via `--reuse-deployment` flag (default: true)
  - Manages deployment config template with user's operators (admin, sequencer, batcher, proposer)
- `pkg/stacks/thanos/input.go`: User input collection
  - `InputDeployContracts()`: Collects L1 RPC, seed phrase, and operators
  - `initDeployConfigTemplate()`: Creates deployment config using user input
- `pkg/stacks/thanos/thanos.go`: Main stack orchestrator

**Important Detail**: The `DeployContractsInput` struct flows through:
1. CLI flag parsing (`--reuse-deployment`)
2. User input collection (L1 RPC, seed phrase, operators)
3. Validation (operators are always required)
4. Config template initialization
5. Deployment execution with state resumption

### Deployment State Management
- Deployment state is persisted in `settings.json` in the deployment directory
- The SDK can resume interrupted deployments via "Resume? (Y/n):" prompt
- State file includes deployment paths, RPC URLs, chain IDs, operator keys

### Configuration & Types (pkg/)
- `pkg/constants/`: Network-specific configs (testnet, mainnet) - L1 chain IDs, RPC defaults, native tokens
- `pkg/types/`: Core data structures (DeployContractsInput, DeployConfigTemplate, Operators, ChainConfiguration)
- `pkg/utils/`: Helper functions for RPC interaction, key derivation (BIP39 seed phrases), AWS operations
- `pkg/logging/`: Zap-based structured logging with file output

## Common Development Tasks

### Building & Running
```bash
# Build the CLI tool
go build -o trh-sdk

# Run with a specific command
./trh-sdk deploy-contracts --network testnet --stack thanos

# Build and test inline
go build ./... && go test ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/utils/

# Run a specific test (e.g., RDS validation)
go test ./pkg/utils -run TestRDS
```

Tests exist in:
- `pkg/utils/rds_test.go`: Database configuration validation
- `pkg/utils/tools_test.go`: Utility function tests

**Note**: tokamak-thanos submodule tests (in mainnet-* directories) are separate from SDK tests.

### Code Quality
```bash
# Check Go version (required by main.go)
go version  # Must be 1.24.11

# Run basic linting (if available)
golangci-lint run ./...

# Format code
go fmt ./...

# Check for type errors
go vet ./...
```

## Critical Implementation Details

### Flag Processing for Deploy Contracts
**Flags removed environment variable support** to ensure CLI flags are always authoritative:
- `--reuse-deployment` (default: true): Reuse existing artifacts, skip cloning/building/deploying
- `--no-candidate` (default: false): Skip candidate registration

The flag sources (env vars) were removed from `flags/flags.go` to prevent environment variable override of CLI flags. Always use explicit CLI flags for these options.

### ReuseDeployment Flag
When `--reuse-deployment=true` (default):
1. Skips repository cloning
2. Skips smart contract build
3. Skips contract deployment to L1
4. Reuses artifacts from previous deployment

This is reflected in `DeployConfigTemplate.ReuseDeployment` which must match user intent (not hardcoded).

### Sensitive Operations
- **Private key handling**: Seed phrase → BIP39 derivation → account selection → stored in settings.json
- **AWS credentials**: Required for EKS operations, validated in `commands/deploy.go`
- **L1 RPC URLs**: Validated for format and connectivity before use

## Key Files to Understand

### When Adding New Deployment Features
1. `cli.go`: Register new CLI command
2. `commands/`: Implement action handler
3. `flags/flags.go`: Define flags if needed
4. `pkg/stacks/thanos/`: Implement stack logic
5. `pkg/types/`: Add input/config types as needed

### When Modifying Contract Deployment
1. `pkg/stacks/thanos/deploy_contracts.go`: Core deployment logic
2. `pkg/stacks/thanos/input.go`: Input collection and validation
3. `flags/flags.go`: CLI flag definitions

### When Adding Validation
- `pkg/stacks/thanos/input.go`: `Validate()` method adds validation logic
- Already validates: L1 RPC connectivity, chain configuration, register candidate inputs

## User-Facing Documentation

- **README.md**: Covers setup, basic usage, all commands
- **docs/deploy-contracts-guide.md**: Comprehensive deployment guide with verification steps
- **docs/monitoring.md**: Monitoring plugin documentation
- **docs/uptime-service.md**: Uptime monitoring service details

When modifying deployment behavior, update both code AND docs/deploy-contracts-guide.md.
