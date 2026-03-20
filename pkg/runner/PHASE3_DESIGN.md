# Phase 3 Design: AWSRunner + TFRunner

## AWSRunner Interface

```go
package runner

import "context"

// AWSRunner defines AWS operations used across TRH SDK.
// It replaces ~73 aws CLI subprocess calls spanning EFS, Backup, EC2, ELB,
// EKS, RDS, S3, IAM, STS, CloudWatch Logs, and VPC management.
//
// NativeAWSRunner uses aws-sdk-go-v2 directly;
// ShellOutAWSRunner shells out to the aws binary as a fallback.
type AWSRunner interface {
    // --- STS ---

    // GetCallerIdentityAccount returns the AWS account ID of the current caller.
    GetCallerIdentityAccount(ctx context.Context) (string, error)

    // --- IAM ---

    // IAMRoleExists checks whether the named IAM role exists.
    IAMRoleExists(ctx context.Context, roleName string) (bool, error)

    // --- EFS ---

    // EFSDescribeFileSystems returns file systems, optionally filtered by ID.
    // If fsID is empty, all file systems are returned.
    EFSDescribeFileSystems(ctx context.Context, region string, fsID string) ([]EFSFileSystem, error)

    // EFSCreateMountTarget creates a mount target for the given file system.
    EFSCreateMountTarget(ctx context.Context, region, fsID, subnetID string, securityGroups []string) error

    // EFSDescribeMountTargets returns mount targets for a file system.
    EFSDescribeMountTargets(ctx context.Context, region, fsID string) ([]EFSMountTarget, error)

    // EFSDescribeMountTargetSecurityGroups returns security group IDs for a mount target.
    EFSDescribeMountTargetSecurityGroups(ctx context.Context, region, mountTargetID string) ([]string, error)

    // EFSDeleteMountTarget deletes a mount target.
    EFSDeleteMountTarget(ctx context.Context, region, mountTargetID string) error

    // EFSDeleteFileSystem deletes an EFS file system.
    EFSDeleteFileSystem(ctx context.Context, region, fsID string) error

    // EFSUpdateFileSystem updates an EFS file system (e.g. throughput mode).
    EFSUpdateFileSystem(ctx context.Context, region, fsID string, throughputMode string) error

    // EFSTagResource applies tags to an EFS resource.
    EFSTagResource(ctx context.Context, region, resourceID string, tags map[string]string) error

    // --- AWS Backup ---

    // BackupStartBackupJob starts an on-demand backup job and returns the job ID.
    BackupStartBackupJob(ctx context.Context, region, vaultName, resourceArn, iamRoleArn string) (string, error)

    // BackupDescribeBackupJob returns the status of a backup job.
    BackupDescribeBackupJob(ctx context.Context, region, jobID string) (string, error)

    // BackupStartRestoreJob starts a restore job and returns the job ID.
    BackupStartRestoreJob(ctx context.Context, region, recoveryPointArn, iamRoleArn string, metadata map[string]string) (string, error)

    // BackupDescribeRestoreJob returns the status and created resource ARN of a restore job.
    BackupDescribeRestoreJob(ctx context.Context, region, jobID string) (BackupRestoreJobStatus, error)

    // BackupListRecoveryPointsByResource lists recovery points for a resource ARN.
    BackupListRecoveryPointsByResource(ctx context.Context, region, resourceArn string) ([]BackupRecoveryPoint, error)

    // BackupListRecoveryPointsByVault lists recovery points in a backup vault.
    BackupListRecoveryPointsByVault(ctx context.Context, region, vaultName string) ([]string, error)

    // BackupDeleteRecoveryPoint deletes a recovery point from a vault.
    BackupDeleteRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error

    // BackupListBackupVaults lists backup vaults, optionally filtered by name prefix.
    BackupListBackupVaults(ctx context.Context, region string) ([]BackupVault, error)

    // BackupDeleteBackupVault deletes a backup vault.
    BackupDeleteBackupVault(ctx context.Context, region, vaultName string) error

    // BackupListProtectedResources checks if a resource ARN is protected.
    BackupIsResourceProtected(ctx context.Context, region, resourceArn string) (bool, error)

    // BackupDescribeRecoveryPoint checks whether a recovery point exists in a vault.
    BackupDescribeRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error

    // BackupListBackupPlans lists backup plans, optionally filtered by name.
    BackupListBackupPlans(ctx context.Context, region string) ([]BackupPlan, error)

    // BackupGetBackupPlan returns details for a specific backup plan.
    BackupGetBackupPlan(ctx context.Context, region, planID string) (*BackupPlanDetail, error)

    // --- EC2 ---

    // EC2DescribeSubnet returns the state of a subnet.
    EC2DescribeSubnetState(ctx context.Context, region, subnetID string) (string, error)

    // EC2DescribeAddresses returns elastic IPs matching a namespace tag filter.
    EC2DescribeAddresses(ctx context.Context, region, namespaceFilter string) ([]ElasticIPInfo, error)

    // EC2DisassociateAddress disassociates an elastic IP.
    EC2DisassociateAddress(ctx context.Context, region, associationID string) error

    // EC2ReleaseAddress releases an elastic IP.
    EC2ReleaseAddress(ctx context.Context, region, allocationID string) error

    // EC2DescribeNATGateways returns NAT gateway IDs matching a namespace tag filter.
    EC2DescribeNATGateways(ctx context.Context, region, namespaceFilter string) ([]string, error)

    // EC2DeleteNATGateway deletes a NAT gateway.
    EC2DeleteNATGateway(ctx context.Context, region, natGatewayID string) error

    // EC2DescribeVPCs returns VPC IDs matching a namespace tag filter.
    EC2DescribeVPCs(ctx context.Context, region, namespaceFilter string) ([]string, error)

    // EC2DescribeVPCResources returns resource IDs for a VPC (subnets, route-tables, etc).
    EC2DescribeVPCResources(ctx context.Context, region, vpcID, resourceType string) ([]string, error)

    // EC2DetachInternetGateway detaches an internet gateway from a VPC.
    EC2DetachInternetGateway(ctx context.Context, region, igwID, vpcID string) error

    // EC2DeleteInternetGateway deletes an internet gateway.
    EC2DeleteInternetGateway(ctx context.Context, region, igwID string) error

    // EC2DeleteSubnet deletes a subnet.
    EC2DeleteSubnet(ctx context.Context, region, subnetID string) error

    // EC2DeleteRouteTable deletes a route table.
    EC2DeleteRouteTable(ctx context.Context, region, routeTableID string) error

    // EC2DeleteSecurityGroup deletes a security group.
    EC2DeleteSecurityGroup(ctx context.Context, region, groupID string) error

    // EC2DeleteVPC deletes a VPC.
    EC2DeleteVPC(ctx context.Context, region, vpcID string) error

    // --- ELB (Classic) ---

    // ELBDescribeLoadBalancers returns classic load balancer names matching a namespace filter.
    ELBDescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error)

    // ELBDeleteLoadBalancer deletes a classic load balancer.
    ELBDeleteLoadBalancer(ctx context.Context, region, name string) error

    // --- ELBv2 (ALB/NLB) ---

    // ELBv2DescribeLoadBalancers returns ALB/NLB ARNs matching a namespace filter.
    ELBv2DescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error)

    // ELBv2DeleteLoadBalancer deletes an ALB/NLB.
    ELBv2DeleteLoadBalancer(ctx context.Context, region, arn string) error

    // --- EKS ---

    // EKSDescribeCluster checks if an EKS cluster exists. Returns true if it does.
    EKSClusterExists(ctx context.Context, region, name string) (bool, error)

    // EKSListNodegroups returns node group names for an EKS cluster.
    EKSListNodegroups(ctx context.Context, region, clusterName string) ([]string, error)

    // EKSDeleteNodegroup deletes an EKS node group.
    EKSDeleteNodegroup(ctx context.Context, region, clusterName, nodeGroupName string) error

    // EKSDeleteCluster deletes an EKS cluster.
    EKSDeleteCluster(ctx context.Context, region, name string) error

    // --- RDS ---

    // RDSInstanceExists checks if an RDS instance exists.
    RDSInstanceExists(ctx context.Context, region, identifier string) (bool, error)

    // RDSDeleteInstance deletes an RDS instance (skip-final-snapshot, delete-automated-backups).
    RDSDeleteInstance(ctx context.Context, region, identifier string) error

    // --- S3 ---

    // S3ListBuckets returns all S3 bucket names.
    S3ListBuckets(ctx context.Context) ([]string, error)

    // S3EmptyBucket removes all objects from an S3 bucket.
    S3EmptyBucket(ctx context.Context, bucket string) error

    // S3DeleteBucket deletes an S3 bucket (force).
    S3DeleteBucket(ctx context.Context, bucket string) error

    // --- CloudWatch Logs ---

    // LogsDescribeLogGroups checks if a log group exists.
    LogsDescribeLogGroups(ctx context.Context, region, logGroupNamePrefix string) (bool, error)

    // LogsPutRetentionPolicy sets retention policy on a log group.
    LogsPutRetentionPolicy(ctx context.Context, region, logGroupName string, retentionDays int) error

    // --- Version Check ---

    // CheckVersion verifies the aws CLI is available (for legacy/shellout mode).
    CheckVersion(ctx context.Context) error
}
```

## AWS Data Types

```go
package runner

import "time"

// EFSFileSystem represents an EFS file system.
type EFSFileSystem struct {
    FileSystemID   string
    Name           string
    LifeCycleState string
    CreationTime   time.Time
    ThroughputMode string
}

// EFSMountTarget represents an EFS mount target.
type EFSMountTarget struct {
    MountTargetID        string
    SubnetID             string
    AvailabilityZoneName string
}

// ElasticIPInfo holds elastic IP details.
type ElasticIPInfo struct {
    AllocationID  string
    AssociationID string
}

// BackupRecoveryPoint represents a recovery point.
type BackupRecoveryPoint struct {
    RecoveryPointArn string
    BackupVaultName  string
    CreationDate     time.Time
    ExpiryDate       *time.Time
    Status           string
}

// BackupRestoreJobStatus holds restore job status.
type BackupRestoreJobStatus struct {
    Status            string
    CreatedResourceArn string
    StatusMessage     string
}

// BackupVault represents a backup vault.
type BackupVault struct {
    BackupVaultName string
}

// BackupPlan represents a backup plan summary.
type BackupPlan struct {
    BackupPlanID   string
    BackupPlanName string
}

// BackupPlanDetail contains backup plan rules.
type BackupPlanDetail struct {
    Rules []BackupPlanRule
}

// BackupPlanRule represents a single backup plan rule.
type BackupPlanRule struct {
    ScheduleExpression string
    DeleteAfterDays    int // 0 means unlimited retention
}
```

## TFRunner Interface

```go
package runner

import (
    "context"
    "io"
)

// TFRunner defines Terraform operations used across TRH SDK.
// It replaces 5 terraform subprocess calls (init, apply, destroy).
//
// NativeTFRunner uses github.com/hashicorp/terraform-exec/tfexec;
// ShellOutTFRunner shells out to the terraform binary as a fallback.
type TFRunner interface {
    // Init runs terraform init in the given working directory.
    // backendConfigs are key=value pairs passed via -backend-config.
    Init(ctx context.Context, workDir string, env []string, backendConfigs []string) error

    // Apply runs terraform apply -auto-approve in the given working directory.
    Apply(ctx context.Context, workDir string, env []string) error

    // Destroy runs terraform destroy -auto-approve in the given working directory.
    Destroy(ctx context.Context, workDir string, env []string) error

    // SetStdout sets the writer for terraform stdout streaming.
    // This mirrors ExecuteCommandStreamWithEnvInDir behavior.
    SetStdout(w io.Writer)

    // CheckVersion verifies terraform is available (for legacy/shellout mode).
    CheckVersion(ctx context.Context) error
}
```

## ToolRunner Extension

```go
// ToolRunner is updated to include AWS() and TF() accessors:
type ToolRunner interface {
    K8s() K8sRunner
    Helm() HelmRunner
    DO() DORunner
    AWS() AWSRunner
    TF() TFRunner
}
```

## File Structure

### New Files

- `pkg/runner/aws_runner.go` — AWSRunner interface + data types
- `pkg/runner/native_aws.go` — NativeAWSRunner (aws-sdk-go-v2)
- `pkg/runner/shellout_aws.go` — ShellOutAWSRunner (shells out to `aws` CLI)
- `pkg/runner/tf_runner.go` — TFRunner interface
- `pkg/runner/native_tf.go` — NativeTFRunner (terraform-exec)
- `pkg/runner/shellout_tf.go` — ShellOutTFRunner (shells out to `terraform`)
- `pkg/runner/mock/mock_aws.go` — MockAWSRunner
- `pkg/runner/mock/mock_tf.go` — MockTFRunner

### Modified Files

- `pkg/runner/runner.go` — Add `AWS() AWSRunner` and `TF() TFRunner` to `ToolRunner` interface; update `NativeRunner`, `ShellOutRunner`, `RunnerConfig`, and `New()`.

### Stack Migration Files (later phase)

- `pkg/stacks/thanos/thanos_stack.go` — Add `awsRunner` and `tfRunner` fields
- `pkg/utils/aws_cleanup.go` — Migrate each function to accept `AWSRunner` or add helper methods
- `pkg/stacks/thanos/backup/*.go` — Migrate all backup functions
- `pkg/stacks/thanos/monitoring.go` — Migrate CloudWatch calls
- `pkg/stacks/thanos/deploy_chain.go` — Migrate terraform calls
- `pkg/stacks/thanos/destroy_chain.go` — Migrate terraform calls

## AWS Call Site Mapping

### STS

| Method | Callers | Notes |
|--------|---------|-------|
| `GetCallerIdentityAccount` | `utils/backup.go:40`, `backup/cleanup.go:28`, `backup/snapshot.go:18`, `backup/status.go:19`, `backup/restore.go:29,236` | Returns account ID string |

### IAM

| Method | Callers | Notes |
|--------|---------|-------|
| `IAMRoleExists` | `backup/restore.go:574` | Check if IAM role exists for restore |

### EFS

| Method | Callers | Notes |
|--------|---------|-------|
| `EFSDescribeFileSystems` | `monitoring.go:1061`, `backup/cleanup.go:136`, `backup/restore.go:589`, `utils/aws_cleanup.go:164` | Various query patterns; native returns full struct, callers filter |
| `EFSCreateMountTarget` | `backup/attach.go:285` | Create mount target with security groups |
| `EFSDescribeMountTargets` | `backup/attach.go:250,299`, `backup/cleanup.go:226`, `utils/aws_cleanup.go:181` | List mount targets for a filesystem |
| `EFSDescribeMountTargetSecurityGroups` | `backup/attach.go:273,331` | Get SGs for a mount target |
| `EFSDeleteMountTarget` | `backup/cleanup.go:245`, `utils/aws_cleanup.go:197` | Delete by mount target ID |
| `EFSDeleteFileSystem` | `backup/cleanup.go:204`, `utils/aws_cleanup.go:204` | Delete by filesystem ID |
| `EFSUpdateFileSystem` | `backup/restore.go:617` | Set throughput mode to elastic |
| `EFSTagResource` | `backup/restore.go:640` | Apply Name tag |

### AWS Backup

| Method | Callers | Notes |
|--------|---------|-------|
| `BackupStartBackupJob` | `backup/initialize.go:59`, `backup/snapshot.go:37` | Returns job ID |
| `BackupDescribeBackupJob` | `backup/initialize.go:79`, `backup/snapshot.go:75` | Returns status string |
| `BackupStartRestoreJob` | `backup/restore.go:427-435` | With metadata map |
| `BackupDescribeRestoreJob` | `backup/restore.go:462,533` | Native SDK already used in `MonitorEFSRestoreJob`; shellout in `HandleEFSRestoreCompletion` |
| `BackupListRecoveryPointsByResource` | `backup/cleanup.go:71`, `backup/list.go:22`, `backup/status.go:113,125`, `backup/restore.go:247` | Various JMESPath queries; native returns full list, caller filters |
| `BackupListRecoveryPointsByVault` | `backup/cleanup.go:330` | Returns ARN list |
| `BackupDeleteRecoveryPoint` | `backup/cleanup.go:100,355` | By vault name + ARN |
| `BackupListBackupVaults` | `backup/cleanup.go:267`, `backup/restore.go:385` | List all vaults |
| `BackupDeleteBackupVault` | `backup/cleanup.go:306` | Delete by name |
| `BackupIsResourceProtected` | `backup/status.go:102` | Check protection status |
| `BackupDescribeRecoveryPoint` | `backup/restore.go:395,406` | Validate recovery point exists |
| `BackupListBackupPlans` | `backup/status.go:159` | For schedule info |
| `BackupGetBackupPlan` | `backup/status.go:189` | For detailed rules |

### EC2

| Method | Callers | Notes |
|--------|---------|-------|
| `EC2DescribeSubnetState` | `backup/attach.go:323` | Check subnet availability |
| `EC2DescribeAddresses` | `utils/aws_cleanup.go:65` | Filter by namespace tag |
| `EC2DisassociateAddress` | `utils/aws_cleanup.go:82` | By association ID |
| `EC2ReleaseAddress` | `utils/aws_cleanup.go:89` | By allocation ID |
| `EC2DescribeNATGateways` | `utils/aws_cleanup.go:96` | Filter by namespace tag |
| `EC2DeleteNATGateway` | `utils/aws_cleanup.go:114` | By NAT gateway ID |
| `EC2DescribeVPCs` | `utils/aws_cleanup.go:253` | Filter by namespace tag |
| `EC2DescribeVPCResources` | `utils/aws_cleanup.go:270` | Generic VPC sub-resource query |
| `EC2DetachInternetGateway` | `utils/aws_cleanup.go:287` | Detach from VPC |
| `EC2DeleteInternetGateway` | `utils/aws_cleanup.go:294` | Delete by ID |
| `EC2DeleteSubnet` | `utils/aws_cleanup.go:301` | Delete by ID |
| `EC2DeleteRouteTable` | `utils/aws_cleanup.go:308` | Delete by ID |
| `EC2DeleteSecurityGroup` | `utils/aws_cleanup.go:315` | Delete by group ID |
| `EC2DeleteVPC` | `utils/aws_cleanup.go:322` | Delete by VPC ID |

### ELB / ELBv2

| Method | Callers | Notes |
|--------|---------|-------|
| `ELBDescribeLoadBalancers` | `utils/aws_cleanup.go:11` | Classic LBs by namespace filter |
| `ELBDeleteLoadBalancer` | `utils/aws_cleanup.go:28` | Classic LB by name |
| `ELBv2DescribeLoadBalancers` | `utils/aws_cleanup.go:35` | ALB/NLB by namespace filter |
| `ELBv2DeleteLoadBalancer` | `utils/aws_cleanup.go:52` | ALB/NLB by ARN |

### EKS

| Method | Callers | Notes |
|--------|---------|-------|
| `EKSClusterExists` | `utils/aws_cleanup.go:121` | Check cluster existence |
| `EKSListNodegroups` | `utils/aws_cleanup.go:128` | List node groups for cluster |
| `EKSDeleteNodegroup` | `utils/aws_cleanup.go:144` | Delete node group |
| `EKSDeleteCluster` | `utils/aws_cleanup.go:151` | Delete cluster |

### RDS

| Method | Callers | Notes |
|--------|---------|-------|
| `RDSInstanceExists` | `utils/aws_cleanup.go:211` | Check instance existence |
| `RDSDeleteInstance` | `utils/aws_cleanup.go:218` | Skip final snapshot + delete backups |

### S3

| Method | Callers | Notes |
|--------|---------|-------|
| `S3ListBuckets` | `utils/aws_cleanup.go:226` | All bucket names |
| `S3EmptyBucket` | `utils/aws_cleanup.go:241` | Recursive delete all objects |
| `S3DeleteBucket` | `utils/aws_cleanup.go:247` | Force delete bucket |

### CloudWatch Logs

| Method | Callers | Notes |
|--------|---------|-------|
| `LogsDescribeLogGroups` | `monitoring.go:1796,1874` | Check log group existence / retention |
| `LogsPutRetentionPolicy` | `monitoring.go:1816` | Set retention days |

### Version Check

| Method | Callers | Notes |
|--------|---------|-------|
| `CheckVersion` | `backup/attach.go:38`, `dependencies/dependencies.go:91` | Verify aws CLI installed |

## TF Call Site Mapping

| Method | Callers | Notes |
|--------|---------|-------|
| `Init` | `deploy_chain.go:529` (backend, no backend-config), `deploy_chain.go:559` (thanos-stack, with backend-config) | Working dir + env vars + optional backend configs |
| `Apply` | `deploy_chain.go:532` (backend), `deploy_chain.go:572` (thanos-stack) | Working dir + env vars |
| `Destroy` | `destroy_chain.go:195` | Working dir + env vars |
| `CheckVersion` | `dependencies/dependencies.go:80` | Verify terraform installed |

## Migration Strategy

### Phase 3a: AWSRunner (AWS Dev Agent)

**Step 1: Create interface and types** (`aws_runner.go`)
- Define `AWSRunner` interface with all methods listed above
- Define data types: `EFSFileSystem`, `EFSMountTarget`, `ElasticIPInfo`, `BackupRecoveryPoint`, `BackupRestoreJobStatus`, `BackupVault`, `BackupPlan`, `BackupPlanDetail`, `BackupPlanRule`

**Step 2: Create ShellOutAWSRunner** (`shellout_aws.go`)
- Implement each method by shelling out to `aws` CLI, mirroring the existing `utils.ExecuteCommand` calls exactly
- This is a 1:1 translation of the existing code into the interface pattern
- Error wrapping: `fmt.Errorf("aws <methodName>: %w", err)`

**Step 3: Create NativeAWSRunner** (`native_aws.go`)
- Use `aws-sdk-go-v2` clients: `sts`, `iam`, `efs`, `backup`, `ec2`, `elasticloadbalancing`, `elasticloadbalancingv2`, `eks`, `rds`, `s3`, `cloudwatchlogs`
- Load config once with `config.LoadDefaultConfig(ctx, config.WithRegion(region))` in constructor
- Note: Region is per-call (passed to each method), not per-runner. Create clients lazily or accept region per-call.
- Pattern: Use goroutine+select for SDK calls where the SDK doesn't accept context natively (rare in aws-sdk-go-v2 since most calls accept ctx)
- Thread safety: Use `sync.Mutex` where needed for shared state

**Step 4: Create MockAWSRunner** (`mock/mock_aws.go`)
- Follow `mock/mock_helm.go` pattern exactly
- `On<MethodName>` hooks for every method
- `record()` + `CallCount()` + `GetCalls()` with `sync.Mutex`

**Step 5: Wire into runner.go**
- Add `AWS() AWSRunner` to `ToolRunner` interface
- Add `aws AWSRunner` field to `NativeRunner`
- Add `ShellOutAWSRunner` to `ShellOutRunner`
- Update `New()` and `newNativeRunner()` to construct AWSRunner

**Step 6: Migrate call sites** (gradual, dual-path)
- Add `awsRunner runner.AWSRunner` field to `ThanosStack`
- Add `SetAWSRunner(ar runner.AWSRunner)` method
- Create dual-path helpers in `thanos_stack.go` (e.g., `t.awsGetCallerIdentity(ctx)`)
- Migrate `utils/aws_cleanup.go` — functions should accept `AWSRunner` as parameter
- Migrate `backup/*.go` — functions should accept `AWSRunner` as parameter
- Migrate `monitoring.go` CloudWatch calls

### Phase 3b: TFRunner (TF Dev Agent)

**Step 1: Create interface** (`tf_runner.go`)
- Define `TFRunner` interface

**Step 2: Create ShellOutTFRunner** (`shellout_tf.go`)
- Shell out to `terraform` binary using `utils.ExecuteCommandStreamWithEnvInDir`
- Supports streaming stdout to a logger/writer

**Step 3: Create NativeTFRunner** (`native_tf.go`)
- Use `github.com/hashicorp/terraform-exec/tfexec`
- Use `github.com/hashicorp/hc-install` for auto-installing terraform if needed
- `tfexec.NewTerraform(workingDir, execPath)` for each call
- Map env vars via `tf.SetEnv()`
- `Init()` maps backend-configs to `tfexec.BackendConfig()` options
- `Apply()` maps to `tf.Apply(ctx, tfexec.Lock(true))` (streams stdout)
- `Destroy()` maps to `tf.Destroy(ctx, tfexec.Lock(true))`

**Step 4: Create MockTFRunner** (`mock/mock_tf.go`)
- Same pattern as mock_helm.go

**Step 5: Wire into runner.go**
- Add `TF() TFRunner` to `ToolRunner` interface
- Add `tf TFRunner` field to `NativeRunner`
- Add `ShellOutTFRunner` to `ShellOutRunner`

**Step 6: Migrate call sites** (gradual, dual-path)
- Add `tfRunner runner.TFRunner` field to `ThanosStack`
- Add `SetTFRunner(tr runner.TFRunner)` method
- Replace `utils.ExecuteCommandStreamWithEnvInDir(ctx, t.logger, dir, env, "terraform", ...)` with runner calls
- In `deploy_chain.go`: `t.tfRunner.Init(ctx, backendDir, backendEnv, nil)` then `t.tfRunner.Apply(ctx, backendDir, backendEnv)`
- In `destroy_chain.go`: `t.tfRunner.Destroy(ctx, thanosStackDir, destroyEnv)`

## NativeAWSRunner: Region Handling

The AWS CLI calls in this codebase pass `--region` per-command. The native runner should match this:

```go
type NativeAWSRunner struct {
    // cfg is the base AWS config (without region).
    // Each method call creates a region-specific client.
    cfg aws.Config

    // clientCache caches service clients per region to avoid re-creation.
    mu          sync.RWMutex
    efsClients  map[string]*efs.Client
    // ... other client caches
}

func (r *NativeAWSRunner) efsClient(region string) *efs.Client {
    r.mu.RLock()
    if c, ok := r.efsClients[region]; ok {
        r.mu.RUnlock()
        return c
    }
    r.mu.RUnlock()
    r.mu.Lock()
    defer r.mu.Unlock()
    c := efs.NewFromConfig(r.cfg, func(o *efs.Options) { o.Region = region })
    r.efsClients[region] = c
    return c
}
```

## NativeTFRunner: Exec Path

```go
type NativeTFRunner struct {
    execPath string    // path to terraform binary
    stdout   io.Writer // where to stream output
}

func newNativeTFRunner() (*NativeTFRunner, error) {
    // Use hc-install to find or install terraform
    installer := &releases.LatestVersion{
        Product: product.Terraform,
    }
    execPath, err := installer.Install(context.Background())
    if err != nil {
        // Fallback: try to find terraform in PATH
        execPath, err = exec.LookPath("terraform")
        if err != nil {
            return nil, fmt.Errorf("terraform not found: %w", err)
        }
    }
    return &NativeTFRunner{execPath: execPath, stdout: os.Stdout}, nil
}
```

## Dependencies to Add (go.mod)

### AWSRunner (aws-sdk-go-v2)

```
github.com/aws/aws-sdk-go-v2
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/sts
github.com/aws/aws-sdk-go-v2/service/iam
github.com/aws/aws-sdk-go-v2/service/efs
github.com/aws/aws-sdk-go-v2/service/backup
github.com/aws/aws-sdk-go-v2/service/ec2
github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing
github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2
github.com/aws/aws-sdk-go-v2/service/eks
github.com/aws/aws-sdk-go-v2/service/rds
github.com/aws/aws-sdk-go-v2/service/s3
github.com/aws/aws-sdk-go-v2/service/s3/s3manager
github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs
```

Note: Some of these are already imported. Check `go.mod` before adding.
The project already imports `github.com/aws/aws-sdk-go-v2/service/backup` and `github.com/aws/aws-sdk-go-v2/service/efs` (used in `backup/restore.go:12-15`).

### TFRunner (terraform-exec)

```
github.com/hashicorp/terraform-exec/tfexec
github.com/hashicorp/hc-install
github.com/hashicorp/hc-install/product
github.com/hashicorp/hc-install/releases
```

## Key Design Decisions

1. **Region per-call, not per-runner**: Unlike DO where the token identifies the environment, AWS region varies per operation within the same session. Each AWS method accepts `region` explicitly.

2. **Rich return types, not raw JSON**: The interface returns Go structs, not raw strings. Callers no longer parse JSON themselves. This is cleaner than the shellout pattern and matches how the native SDK returns typed responses.

3. **S3EmptyBucket**: The aws CLI `s3 rm --recursive` has no direct single-API-call equivalent. The native implementation should use `s3.ListObjectsV2` + `s3.DeleteObjects` in a paginated loop.

4. **TFRunner uses streaming**: Terraform operations can take minutes. Both `ShellOutTFRunner` and `NativeTFRunner` must stream stdout to the provided writer (typically a logger) in real-time, matching the existing `ExecuteCommandStreamWithEnvInDir` behavior.

5. **TFRunner env handling**: Terraform env vars (TF_VAR_*, AWS_*, etc.) are passed explicitly per-call, not set globally. `tfexec.SetEnv()` maps this cleanly.

6. **EC2DescribeVPCResources is generic**: The existing code uses a dynamic `describe-<resourceType>` pattern with a `vpc-id` filter. The native implementation should have specific methods for each resource type (subnets, internet-gateways, route-tables, security-groups) but the interface exposes a single `EC2DescribeVPCResources` that dispatches internally based on `resourceType`.

7. **Existing SDK usage preserved**: `backup/restore.go` already uses `aws-sdk-go-v2` for `MonitorEFSRestoreJob`. The AWSRunner consolidates this pattern — the existing direct SDK usage in `restore.go` should be migrated to use AWSRunner methods.
