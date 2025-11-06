# Uptime Kuma Installation and Refactor Storage Functions 

## Summary

This PR refactors redundant storage-related functions into generic, reusable utilities and improves the uptime-kuma installation logic to better handle existing installations and LoadBalancer services.

## Problem Statement

1. **Code Duplication**: Storage-related functions (`getEFSFileSystemId`, `generateStaticPVManifest`, `applyPVManifest`, etc.) were duplicated across `uptime_kuma.go` and `monitoring.go`, violating **DRY(Don't Repeat Yourself)** principles.

2. **LoadBalancer Service Support**: The code needed extra support for retrieving LoadBalancer service URLs for uptime-kuma access.

3. **Code Organization**: Storage utilities were scattered across stack-specific files instead of being in a shared location.

## Solution

### 1. Generic Storage Functions with Interface Pattern

Created a `StorageConfig` interface and moved common storage operations to `pkg/utils/storage.go` for reusability across different stacks.

### 2. LoadBalancer Service URL Retrieval

Added utility function to retrieve LoadBalancer service hostnames for uptime-kuma access.

## Changes Made

### New Files

- **`pkg/utils/storage.go`** - Generic storage utility functions
  - `GetEFSFileSystemId()` - Extracts EFS filesystem ID from existing PVs or AWS
  - `GetTimestampFromExistingPV()` - Extracts timestamp from existing PVs
  - `GenerateStaticPVManifest()` - Generates Kubernetes PV manifest
  - `ApplyPVManifest()` - Applies PV manifest to cluster
  - `GenerateStaticPVCManifest()` - Generates Kubernetes PVC manifest
  - `ApplyPVCManifest()` - Applies PVC manifest to cluster

### Modified Files

#### `pkg/types/uptime_kuma.go`
- Introduced `StorageConfig` interface with methods:
  - `GetNamespace() string`
  - `GetChainName() string`
  - `GetEFSFileSystemId() string`
  - `GetHelmReleaseName() string`
- Implemented `StorageConfig` interface for `UptimeKumaConfig`

#### `pkg/stacks/thanos/uptime_kuma.go`
- **Refactored to use generic storage functions** from `utils/storage.go`
- **Improved existing installation detection**:
  - Uses `FilterHelmReleases()` to find existing Helm release names
  - Prevents duplicate installations when pods are already running
  - Returns existing service URL instead of reinstalling
- **LoadBalancer support**:
  - Uses `GetAddressByService()` to retrieve LoadBalancer hostnames
  - Waits for LoadBalancer to be ready with proper retry logic
- **PVC binding wait**: Added `waitForPVCBound()` to ensure PVC is bound before Helm installation
- **Improved logging**: Added informative success messages with service URLs

#### `pkg/utils/kubectl.go`
- **Added `GetAddressByService()` function**:
  - Retrieves LoadBalancer service addresses by service name
  - Returns hostnames from LoadBalancer ingress
  - Supports filtering by service name pattern
- **Updated `SvcJSON` struct**: Added `LoadBalancer` status fields to support LoadBalancer service inspection

## Key Features

### 1. Generic Storage Functions

Storage operations are now reusable across different stacks through the `StorageConfig` interface:

```go
// Before: Duplicate code in multiple files
func (t *ThanosStack) getEFSFileSystemId(...) { ... }

// After: Generic reusable function
func GetEFSFileSystemId(ctx context.Context, chainName string, awsRegion string) (string, error) { ... }
```

### 2. Interface-Based Design

The `StorageConfig` interface allows any configuration type to use the generic storage functions:

```go
type StorageConfig interface {
    GetNamespace() string
    GetChainName() string
    GetEFSFileSystemId() string
    GetHelmReleaseName() string
}
```

### 3. Existing Installation Detection

When uptime-kuma is already running:
- Finds existing Helm release name using `FilterHelmReleases()`
- Retrieves LoadBalancer service URL from existing installation
- Returns immediately without reinstalling
- Provides clear success message with access URL

### 4. LoadBalancer Service Support

New utility function retrieves LoadBalancer service hostnames:

```go
addresses, err := GetAddressByService(ctx, namespace, serviceName)
```

## Testing

### Tested Scenarios

1. **Fresh Installation**
   - ✅ Creates new PVC and PV
   - ✅ Installs Helm chart with correct configuration
   - ✅ Waits for PVC to be bound before installation
   - ✅ Retrieves LoadBalancer URL after installation

2. **Existing Installation**
   - ✅ Detects existing pods
   - ✅ Finds existing Helm release name
   - ✅ Retrieves LoadBalancer URL from existing service
   - ✅ Returns without reinstalling


### Manual Testing

**Note**: Before testing, checkout the PR branch/commit to ensure you're testing the changes from this PR.

```bash
# Checkout the PR branch/commit
git checkout <pr-branch-name>
# or
git checkout <commit-hash>

# Test fresh installation
trh-sdk install uptime-kuma

# Test existing installation (run again)
trh install uptime-kuma

# Verify LoadBalancer service
kubectl get svc -n uptime-kuma
```



