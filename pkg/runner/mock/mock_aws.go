package mock

import (
	"context"
	"sync"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
)

// AWSRunner is a thread-safe, hand-written mock for runner.AWSRunner.
// Configure expected results via the On* fields before calling methods.
// Recorded calls are available via Calls / CallCount / GetCalls after the test.
type AWSRunner struct {
	mu    sync.Mutex
	Calls []Call

	// STS
	OnGetCallerIdentityAccount func(ctx context.Context) (string, error)

	// IAM
	OnIAMRoleExists func(ctx context.Context, roleName string) (bool, error)

	// EFS
	OnEFSDescribeFileSystems               func(ctx context.Context, region, fsID string) ([]runner.EFSFileSystem, error)
	OnEFSCreateMountTarget                 func(ctx context.Context, region, fsID, subnetID string, securityGroups []string) error
	OnEFSDescribeMountTargets              func(ctx context.Context, region, fsID string) ([]runner.EFSMountTarget, error)
	OnEFSDescribeMountTargetSecurityGroups func(ctx context.Context, region, mountTargetID string) ([]string, error)
	OnEFSDeleteMountTarget                 func(ctx context.Context, region, mountTargetID string) error
	OnEFSDeleteFileSystem                  func(ctx context.Context, region, fsID string) error
	OnEFSUpdateFileSystem                  func(ctx context.Context, region, fsID string, throughputMode string) error
	OnEFSTagResource                       func(ctx context.Context, region, resourceID string, tags map[string]string) error

	// Backup
	OnBackupStartBackupJob               func(ctx context.Context, region, vaultName, resourceArn, iamRoleArn string) (string, error)
	OnBackupDescribeBackupJob            func(ctx context.Context, region, jobID string) (string, error)
	OnBackupStartRestoreJob              func(ctx context.Context, region, recoveryPointArn, iamRoleArn string, metadata map[string]string) (string, error)
	OnBackupDescribeRestoreJob           func(ctx context.Context, region, jobID string) (runner.BackupRestoreJobStatus, error)
	OnBackupListRecoveryPointsByResource func(ctx context.Context, region, resourceArn string) ([]runner.BackupRecoveryPoint, error)
	OnBackupListRecoveryPointsByVault    func(ctx context.Context, region, vaultName string) ([]string, error)
	OnBackupDeleteRecoveryPoint          func(ctx context.Context, region, vaultName, recoveryPointArn string) error
	OnBackupListBackupVaults             func(ctx context.Context, region string) ([]runner.BackupVault, error)
	OnBackupDeleteBackupVault            func(ctx context.Context, region, vaultName string) error
	OnBackupIsResourceProtected          func(ctx context.Context, region, resourceArn string) (bool, error)
	OnBackupDescribeRecoveryPoint        func(ctx context.Context, region, vaultName, recoveryPointArn string) error
	OnBackupListBackupPlans              func(ctx context.Context, region string) ([]runner.BackupPlan, error)
	OnBackupGetBackupPlan                func(ctx context.Context, region, planID string) (*runner.BackupPlanDetail, error)

	// EC2
	OnEC2DescribeSubnetState   func(ctx context.Context, region, subnetID string) (string, error)
	OnEC2DescribeAddresses     func(ctx context.Context, region, namespaceFilter string) ([]runner.ElasticIPInfo, error)
	OnEC2DisassociateAddress   func(ctx context.Context, region, associationID string) error
	OnEC2ReleaseAddress        func(ctx context.Context, region, allocationID string) error
	OnEC2DescribeNATGateways   func(ctx context.Context, region, namespaceFilter string) ([]string, error)
	OnEC2DeleteNATGateway      func(ctx context.Context, region, natGatewayID string) error
	OnEC2DescribeVPCs          func(ctx context.Context, region, namespaceFilter string) ([]string, error)
	OnEC2DescribeVPCResources  func(ctx context.Context, region, vpcID, resourceType string) ([]string, error)
	OnEC2DetachInternetGateway func(ctx context.Context, region, igwID, vpcID string) error
	OnEC2DeleteInternetGateway func(ctx context.Context, region, igwID string) error
	OnEC2DeleteSubnet          func(ctx context.Context, region, subnetID string) error
	OnEC2DeleteRouteTable      func(ctx context.Context, region, routeTableID string) error
	OnEC2DeleteSecurityGroup   func(ctx context.Context, region, groupID string) error
	OnEC2DeleteVPC             func(ctx context.Context, region, vpcID string) error

	// ELB
	OnELBDescribeLoadBalancers   func(ctx context.Context, region, namespaceFilter string) ([]string, error)
	OnELBDeleteLoadBalancer      func(ctx context.Context, region, name string) error
	OnELBv2DescribeLoadBalancers func(ctx context.Context, region, namespaceFilter string) ([]string, error)
	OnELBv2DeleteLoadBalancer    func(ctx context.Context, region, arn string) error

	// EKS
	OnEKSClusterExists   func(ctx context.Context, region, name string) (bool, error)
	OnEKSListNodegroups  func(ctx context.Context, region, clusterName string) ([]string, error)
	OnEKSDeleteNodegroup func(ctx context.Context, region, clusterName, nodeGroupName string) error
	OnEKSDeleteCluster   func(ctx context.Context, region, name string) error

	// RDS
	OnRDSInstanceExists func(ctx context.Context, region, identifier string) (bool, error)
	OnRDSDeleteInstance  func(ctx context.Context, region, identifier string) error

	// S3
	OnS3ListBuckets  func(ctx context.Context) ([]string, error)
	OnS3EmptyBucket  func(ctx context.Context, bucket string) error
	OnS3DeleteBucket func(ctx context.Context, bucket string) error

	// CloudWatch Logs
	OnLogsDescribeLogGroups  func(ctx context.Context, region, logGroupNamePrefix string) (bool, error)
	OnLogsPutRetentionPolicy func(ctx context.Context, region, logGroupName string, retentionDays int) error

	// Version
	OnCheckVersion func(ctx context.Context) error
}

func (m *AWSRunner) record(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Method: method, Args: args})
}

// CallCount returns how many times method was called.
func (m *AWSRunner) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, c := range m.Calls {
		if c.Method == method {
			count++
		}
	}
	return count
}

// GetCalls returns a snapshot of all recorded calls, safe for concurrent reads.
func (m *AWSRunner) GetCalls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make([]Call, len(m.Calls))
	copy(snapshot, m.Calls)
	return snapshot
}

// --- STS ---

func (m *AWSRunner) GetCallerIdentityAccount(ctx context.Context) (string, error) {
	m.record("GetCallerIdentityAccount")
	if m.OnGetCallerIdentityAccount != nil {
		return m.OnGetCallerIdentityAccount(ctx)
	}
	return "", nil
}

// --- IAM ---

func (m *AWSRunner) IAMRoleExists(ctx context.Context, roleName string) (bool, error) {
	m.record("IAMRoleExists", roleName)
	if m.OnIAMRoleExists != nil {
		return m.OnIAMRoleExists(ctx, roleName)
	}
	return false, nil
}

// --- EFS ---

func (m *AWSRunner) EFSDescribeFileSystems(ctx context.Context, region, fsID string) ([]runner.EFSFileSystem, error) {
	m.record("EFSDescribeFileSystems", region, fsID)
	if m.OnEFSDescribeFileSystems != nil {
		return m.OnEFSDescribeFileSystems(ctx, region, fsID)
	}
	return nil, nil
}

func (m *AWSRunner) EFSCreateMountTarget(ctx context.Context, region, fsID, subnetID string, securityGroups []string) error {
	m.record("EFSCreateMountTarget", region, fsID, subnetID, securityGroups)
	if m.OnEFSCreateMountTarget != nil {
		return m.OnEFSCreateMountTarget(ctx, region, fsID, subnetID, securityGroups)
	}
	return nil
}

func (m *AWSRunner) EFSDescribeMountTargets(ctx context.Context, region, fsID string) ([]runner.EFSMountTarget, error) {
	m.record("EFSDescribeMountTargets", region, fsID)
	if m.OnEFSDescribeMountTargets != nil {
		return m.OnEFSDescribeMountTargets(ctx, region, fsID)
	}
	return nil, nil
}

func (m *AWSRunner) EFSDescribeMountTargetSecurityGroups(ctx context.Context, region, mountTargetID string) ([]string, error) {
	m.record("EFSDescribeMountTargetSecurityGroups", region, mountTargetID)
	if m.OnEFSDescribeMountTargetSecurityGroups != nil {
		return m.OnEFSDescribeMountTargetSecurityGroups(ctx, region, mountTargetID)
	}
	return nil, nil
}

func (m *AWSRunner) EFSDeleteMountTarget(ctx context.Context, region, mountTargetID string) error {
	m.record("EFSDeleteMountTarget", region, mountTargetID)
	if m.OnEFSDeleteMountTarget != nil {
		return m.OnEFSDeleteMountTarget(ctx, region, mountTargetID)
	}
	return nil
}

func (m *AWSRunner) EFSDeleteFileSystem(ctx context.Context, region, fsID string) error {
	m.record("EFSDeleteFileSystem", region, fsID)
	if m.OnEFSDeleteFileSystem != nil {
		return m.OnEFSDeleteFileSystem(ctx, region, fsID)
	}
	return nil
}

func (m *AWSRunner) EFSUpdateFileSystem(ctx context.Context, region, fsID string, throughputMode string) error {
	m.record("EFSUpdateFileSystem", region, fsID, throughputMode)
	if m.OnEFSUpdateFileSystem != nil {
		return m.OnEFSUpdateFileSystem(ctx, region, fsID, throughputMode)
	}
	return nil
}

func (m *AWSRunner) EFSTagResource(ctx context.Context, region, resourceID string, tags map[string]string) error {
	m.record("EFSTagResource", region, resourceID, tags)
	if m.OnEFSTagResource != nil {
		return m.OnEFSTagResource(ctx, region, resourceID, tags)
	}
	return nil
}

// --- Backup ---

func (m *AWSRunner) BackupStartBackupJob(ctx context.Context, region, vaultName, resourceArn, iamRoleArn string) (string, error) {
	m.record("BackupStartBackupJob", region, vaultName, resourceArn, iamRoleArn)
	if m.OnBackupStartBackupJob != nil {
		return m.OnBackupStartBackupJob(ctx, region, vaultName, resourceArn, iamRoleArn)
	}
	return "", nil
}

func (m *AWSRunner) BackupDescribeBackupJob(ctx context.Context, region, jobID string) (string, error) {
	m.record("BackupDescribeBackupJob", region, jobID)
	if m.OnBackupDescribeBackupJob != nil {
		return m.OnBackupDescribeBackupJob(ctx, region, jobID)
	}
	return "", nil
}

func (m *AWSRunner) BackupStartRestoreJob(ctx context.Context, region, recoveryPointArn, iamRoleArn string, metadata map[string]string) (string, error) {
	m.record("BackupStartRestoreJob", region, recoveryPointArn, iamRoleArn, metadata)
	if m.OnBackupStartRestoreJob != nil {
		return m.OnBackupStartRestoreJob(ctx, region, recoveryPointArn, iamRoleArn, metadata)
	}
	return "", nil
}

func (m *AWSRunner) BackupDescribeRestoreJob(ctx context.Context, region, jobID string) (runner.BackupRestoreJobStatus, error) {
	m.record("BackupDescribeRestoreJob", region, jobID)
	if m.OnBackupDescribeRestoreJob != nil {
		return m.OnBackupDescribeRestoreJob(ctx, region, jobID)
	}
	return runner.BackupRestoreJobStatus{}, nil
}

func (m *AWSRunner) BackupListRecoveryPointsByResource(ctx context.Context, region, resourceArn string) ([]runner.BackupRecoveryPoint, error) {
	m.record("BackupListRecoveryPointsByResource", region, resourceArn)
	if m.OnBackupListRecoveryPointsByResource != nil {
		return m.OnBackupListRecoveryPointsByResource(ctx, region, resourceArn)
	}
	return nil, nil
}

func (m *AWSRunner) BackupListRecoveryPointsByVault(ctx context.Context, region, vaultName string) ([]string, error) {
	m.record("BackupListRecoveryPointsByVault", region, vaultName)
	if m.OnBackupListRecoveryPointsByVault != nil {
		return m.OnBackupListRecoveryPointsByVault(ctx, region, vaultName)
	}
	return nil, nil
}

func (m *AWSRunner) BackupDeleteRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	m.record("BackupDeleteRecoveryPoint", region, vaultName, recoveryPointArn)
	if m.OnBackupDeleteRecoveryPoint != nil {
		return m.OnBackupDeleteRecoveryPoint(ctx, region, vaultName, recoveryPointArn)
	}
	return nil
}

func (m *AWSRunner) BackupListBackupVaults(ctx context.Context, region string) ([]runner.BackupVault, error) {
	m.record("BackupListBackupVaults", region)
	if m.OnBackupListBackupVaults != nil {
		return m.OnBackupListBackupVaults(ctx, region)
	}
	return nil, nil
}

func (m *AWSRunner) BackupDeleteBackupVault(ctx context.Context, region, vaultName string) error {
	m.record("BackupDeleteBackupVault", region, vaultName)
	if m.OnBackupDeleteBackupVault != nil {
		return m.OnBackupDeleteBackupVault(ctx, region, vaultName)
	}
	return nil
}

func (m *AWSRunner) BackupIsResourceProtected(ctx context.Context, region, resourceArn string) (bool, error) {
	m.record("BackupIsResourceProtected", region, resourceArn)
	if m.OnBackupIsResourceProtected != nil {
		return m.OnBackupIsResourceProtected(ctx, region, resourceArn)
	}
	return false, nil
}

func (m *AWSRunner) BackupDescribeRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	m.record("BackupDescribeRecoveryPoint", region, vaultName, recoveryPointArn)
	if m.OnBackupDescribeRecoveryPoint != nil {
		return m.OnBackupDescribeRecoveryPoint(ctx, region, vaultName, recoveryPointArn)
	}
	return nil
}

func (m *AWSRunner) BackupListBackupPlans(ctx context.Context, region string) ([]runner.BackupPlan, error) {
	m.record("BackupListBackupPlans", region)
	if m.OnBackupListBackupPlans != nil {
		return m.OnBackupListBackupPlans(ctx, region)
	}
	return nil, nil
}

func (m *AWSRunner) BackupGetBackupPlan(ctx context.Context, region, planID string) (*runner.BackupPlanDetail, error) {
	m.record("BackupGetBackupPlan", region, planID)
	if m.OnBackupGetBackupPlan != nil {
		return m.OnBackupGetBackupPlan(ctx, region, planID)
	}
	return nil, nil
}

// --- EC2 ---

func (m *AWSRunner) EC2DescribeSubnetState(ctx context.Context, region, subnetID string) (string, error) {
	m.record("EC2DescribeSubnetState", region, subnetID)
	if m.OnEC2DescribeSubnetState != nil {
		return m.OnEC2DescribeSubnetState(ctx, region, subnetID)
	}
	return "", nil
}

func (m *AWSRunner) EC2DescribeAddresses(ctx context.Context, region, namespaceFilter string) ([]runner.ElasticIPInfo, error) {
	m.record("EC2DescribeAddresses", region, namespaceFilter)
	if m.OnEC2DescribeAddresses != nil {
		return m.OnEC2DescribeAddresses(ctx, region, namespaceFilter)
	}
	return nil, nil
}

func (m *AWSRunner) EC2DisassociateAddress(ctx context.Context, region, associationID string) error {
	m.record("EC2DisassociateAddress", region, associationID)
	if m.OnEC2DisassociateAddress != nil {
		return m.OnEC2DisassociateAddress(ctx, region, associationID)
	}
	return nil
}

func (m *AWSRunner) EC2ReleaseAddress(ctx context.Context, region, allocationID string) error {
	m.record("EC2ReleaseAddress", region, allocationID)
	if m.OnEC2ReleaseAddress != nil {
		return m.OnEC2ReleaseAddress(ctx, region, allocationID)
	}
	return nil
}

func (m *AWSRunner) EC2DescribeNATGateways(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	m.record("EC2DescribeNATGateways", region, namespaceFilter)
	if m.OnEC2DescribeNATGateways != nil {
		return m.OnEC2DescribeNATGateways(ctx, region, namespaceFilter)
	}
	return nil, nil
}

func (m *AWSRunner) EC2DeleteNATGateway(ctx context.Context, region, natGatewayID string) error {
	m.record("EC2DeleteNATGateway", region, natGatewayID)
	if m.OnEC2DeleteNATGateway != nil {
		return m.OnEC2DeleteNATGateway(ctx, region, natGatewayID)
	}
	return nil
}

func (m *AWSRunner) EC2DescribeVPCs(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	m.record("EC2DescribeVPCs", region, namespaceFilter)
	if m.OnEC2DescribeVPCs != nil {
		return m.OnEC2DescribeVPCs(ctx, region, namespaceFilter)
	}
	return nil, nil
}

func (m *AWSRunner) EC2DescribeVPCResources(ctx context.Context, region, vpcID, resourceType string) ([]string, error) {
	m.record("EC2DescribeVPCResources", region, vpcID, resourceType)
	if m.OnEC2DescribeVPCResources != nil {
		return m.OnEC2DescribeVPCResources(ctx, region, vpcID, resourceType)
	}
	return nil, nil
}

func (m *AWSRunner) EC2DetachInternetGateway(ctx context.Context, region, igwID, vpcID string) error {
	m.record("EC2DetachInternetGateway", region, igwID, vpcID)
	if m.OnEC2DetachInternetGateway != nil {
		return m.OnEC2DetachInternetGateway(ctx, region, igwID, vpcID)
	}
	return nil
}

func (m *AWSRunner) EC2DeleteInternetGateway(ctx context.Context, region, igwID string) error {
	m.record("EC2DeleteInternetGateway", region, igwID)
	if m.OnEC2DeleteInternetGateway != nil {
		return m.OnEC2DeleteInternetGateway(ctx, region, igwID)
	}
	return nil
}

func (m *AWSRunner) EC2DeleteSubnet(ctx context.Context, region, subnetID string) error {
	m.record("EC2DeleteSubnet", region, subnetID)
	if m.OnEC2DeleteSubnet != nil {
		return m.OnEC2DeleteSubnet(ctx, region, subnetID)
	}
	return nil
}

func (m *AWSRunner) EC2DeleteRouteTable(ctx context.Context, region, routeTableID string) error {
	m.record("EC2DeleteRouteTable", region, routeTableID)
	if m.OnEC2DeleteRouteTable != nil {
		return m.OnEC2DeleteRouteTable(ctx, region, routeTableID)
	}
	return nil
}

func (m *AWSRunner) EC2DeleteSecurityGroup(ctx context.Context, region, groupID string) error {
	m.record("EC2DeleteSecurityGroup", region, groupID)
	if m.OnEC2DeleteSecurityGroup != nil {
		return m.OnEC2DeleteSecurityGroup(ctx, region, groupID)
	}
	return nil
}

func (m *AWSRunner) EC2DeleteVPC(ctx context.Context, region, vpcID string) error {
	m.record("EC2DeleteVPC", region, vpcID)
	if m.OnEC2DeleteVPC != nil {
		return m.OnEC2DeleteVPC(ctx, region, vpcID)
	}
	return nil
}

// --- ELB ---

func (m *AWSRunner) ELBDescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	m.record("ELBDescribeLoadBalancers", region, namespaceFilter)
	if m.OnELBDescribeLoadBalancers != nil {
		return m.OnELBDescribeLoadBalancers(ctx, region, namespaceFilter)
	}
	return nil, nil
}

func (m *AWSRunner) ELBDeleteLoadBalancer(ctx context.Context, region, name string) error {
	m.record("ELBDeleteLoadBalancer", region, name)
	if m.OnELBDeleteLoadBalancer != nil {
		return m.OnELBDeleteLoadBalancer(ctx, region, name)
	}
	return nil
}

func (m *AWSRunner) ELBv2DescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	m.record("ELBv2DescribeLoadBalancers", region, namespaceFilter)
	if m.OnELBv2DescribeLoadBalancers != nil {
		return m.OnELBv2DescribeLoadBalancers(ctx, region, namespaceFilter)
	}
	return nil, nil
}

func (m *AWSRunner) ELBv2DeleteLoadBalancer(ctx context.Context, region, arn string) error {
	m.record("ELBv2DeleteLoadBalancer", region, arn)
	if m.OnELBv2DeleteLoadBalancer != nil {
		return m.OnELBv2DeleteLoadBalancer(ctx, region, arn)
	}
	return nil
}

// --- EKS ---

func (m *AWSRunner) EKSClusterExists(ctx context.Context, region, name string) (bool, error) {
	m.record("EKSClusterExists", region, name)
	if m.OnEKSClusterExists != nil {
		return m.OnEKSClusterExists(ctx, region, name)
	}
	return false, nil
}

func (m *AWSRunner) EKSListNodegroups(ctx context.Context, region, clusterName string) ([]string, error) {
	m.record("EKSListNodegroups", region, clusterName)
	if m.OnEKSListNodegroups != nil {
		return m.OnEKSListNodegroups(ctx, region, clusterName)
	}
	return nil, nil
}

func (m *AWSRunner) EKSDeleteNodegroup(ctx context.Context, region, clusterName, nodeGroupName string) error {
	m.record("EKSDeleteNodegroup", region, clusterName, nodeGroupName)
	if m.OnEKSDeleteNodegroup != nil {
		return m.OnEKSDeleteNodegroup(ctx, region, clusterName, nodeGroupName)
	}
	return nil
}

func (m *AWSRunner) EKSDeleteCluster(ctx context.Context, region, name string) error {
	m.record("EKSDeleteCluster", region, name)
	if m.OnEKSDeleteCluster != nil {
		return m.OnEKSDeleteCluster(ctx, region, name)
	}
	return nil
}

// --- RDS ---

func (m *AWSRunner) RDSInstanceExists(ctx context.Context, region, identifier string) (bool, error) {
	m.record("RDSInstanceExists", region, identifier)
	if m.OnRDSInstanceExists != nil {
		return m.OnRDSInstanceExists(ctx, region, identifier)
	}
	return false, nil
}

func (m *AWSRunner) RDSDeleteInstance(ctx context.Context, region, identifier string) error {
	m.record("RDSDeleteInstance", region, identifier)
	if m.OnRDSDeleteInstance != nil {
		return m.OnRDSDeleteInstance(ctx, region, identifier)
	}
	return nil
}

// --- S3 ---

func (m *AWSRunner) S3ListBuckets(ctx context.Context) ([]string, error) {
	m.record("S3ListBuckets")
	if m.OnS3ListBuckets != nil {
		return m.OnS3ListBuckets(ctx)
	}
	return nil, nil
}

func (m *AWSRunner) S3EmptyBucket(ctx context.Context, bucket string) error {
	m.record("S3EmptyBucket", bucket)
	if m.OnS3EmptyBucket != nil {
		return m.OnS3EmptyBucket(ctx, bucket)
	}
	return nil
}

func (m *AWSRunner) S3DeleteBucket(ctx context.Context, bucket string) error {
	m.record("S3DeleteBucket", bucket)
	if m.OnS3DeleteBucket != nil {
		return m.OnS3DeleteBucket(ctx, bucket)
	}
	return nil
}

// --- CloudWatch Logs ---

func (m *AWSRunner) LogsDescribeLogGroups(ctx context.Context, region, logGroupNamePrefix string) (bool, error) {
	m.record("LogsDescribeLogGroups", region, logGroupNamePrefix)
	if m.OnLogsDescribeLogGroups != nil {
		return m.OnLogsDescribeLogGroups(ctx, region, logGroupNamePrefix)
	}
	return false, nil
}

func (m *AWSRunner) LogsPutRetentionPolicy(ctx context.Context, region, logGroupName string, retentionDays int) error {
	m.record("LogsPutRetentionPolicy", region, logGroupName, retentionDays)
	if m.OnLogsPutRetentionPolicy != nil {
		return m.OnLogsPutRetentionPolicy(ctx, region, logGroupName, retentionDays)
	}
	return nil
}

// --- Version ---

func (m *AWSRunner) CheckVersion(ctx context.Context) error {
	m.record("CheckVersion")
	if m.OnCheckVersion != nil {
		return m.OnCheckVersion(ctx)
	}
	return nil
}
