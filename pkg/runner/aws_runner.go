package runner

import (
	"context"
	"time"
)

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

	// BackupListBackupVaults lists backup vaults.
	BackupListBackupVaults(ctx context.Context, region string) ([]BackupVault, error)

	// BackupDeleteBackupVault deletes a backup vault.
	BackupDeleteBackupVault(ctx context.Context, region, vaultName string) error

	// BackupIsResourceProtected checks if a resource ARN is protected.
	BackupIsResourceProtected(ctx context.Context, region, resourceArn string) (bool, error)

	// BackupDescribeRecoveryPoint checks whether a recovery point exists in a vault.
	BackupDescribeRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error

	// BackupListBackupPlans lists backup plans.
	BackupListBackupPlans(ctx context.Context, region string) ([]BackupPlan, error)

	// BackupGetBackupPlan returns details for a specific backup plan.
	BackupGetBackupPlan(ctx context.Context, region, planID string) (*BackupPlanDetail, error)

	// --- EC2 ---

	// EC2DescribeSubnetState returns the state of a subnet.
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

	// EKSClusterExists checks if an EKS cluster exists.
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
	Status             string
	CreatedResourceArn string
	StatusMessage      string
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
