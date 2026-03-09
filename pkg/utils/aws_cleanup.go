package utils

import (
	"context"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
)

// returns classic ELB names matching the namespace
func ListClassicLoadBalancers(ctx context.Context, ar runner.AWSRunner, region, namespace string) ([]string, error) {
	return ar.ELBDescribeLoadBalancers(ctx, region, namespace)
}

// deletes a classic ELB
func DeleteClassicLoadBalancer(ctx context.Context, ar runner.AWSRunner, region, name string) error {
	return ar.ELBDeleteLoadBalancer(ctx, region, name)
}

// returns ALB/NLB ARNs matching the namespace
func ListALBLoadBalancers(ctx context.Context, ar runner.AWSRunner, region, namespace string) ([]string, error) {
	return ar.ELBv2DescribeLoadBalancers(ctx, region, namespace)
}

// deletes an ALB/NLB
func DeleteALBLoadBalancer(ctx context.Context, ar runner.AWSRunner, region, arn string) error {
	return ar.ELBv2DeleteLoadBalancer(ctx, region, arn)
}

// ElasticIPInfo holds elastic IP details
type ElasticIPInfo struct {
	ID    string
	Assoc string
}

// returns elastic IPs matching the namespace
func ListElasticIPs(ctx context.Context, ar runner.AWSRunner, region, namespace string) ([]ElasticIPInfo, error) {
	infos, err := ar.EC2DescribeAddresses(ctx, region, namespace)
	if err != nil {
		return nil, err
	}
	eips := make([]ElasticIPInfo, 0, len(infos))
	for _, e := range infos {
		eips = append(eips, ElasticIPInfo{
			ID:    e.AllocationID,
			Assoc: e.AssociationID,
		})
	}
	return eips, nil
}

// disassociates an elastic ip
func DisassociateAddress(ctx context.Context, ar runner.AWSRunner, region, associationID string) error {
	return ar.EC2DisassociateAddress(ctx, region, associationID)
}

// releases an elastic ip
func ReleaseAddress(ctx context.Context, ar runner.AWSRunner, region, allocationID string) error {
	return ar.EC2ReleaseAddress(ctx, region, allocationID)
}

// returns nat gateway ids matching the namespace
func ListNATGateways(ctx context.Context, ar runner.AWSRunner, region, namespace string) ([]string, error) {
	return ar.EC2DescribeNATGateways(ctx, region, namespace)
}

// deletes a nat gateway
func DeleteNATGateway(ctx context.Context, ar runner.AWSRunner, region, id string) error {
	return ar.EC2DeleteNATGateway(ctx, region, id)
}

// checks if an eks cluster exists
func EKSClusterExists(ctx context.Context, ar runner.AWSRunner, region, name string) bool {
	exists, err := ar.EKSClusterExists(ctx, region, name)
	if err != nil {
		return false
	}
	return exists
}

// returns node group names for a cluster
func ListNodeGroups(ctx context.Context, ar runner.AWSRunner, region, clusterName string) ([]string, error) {
	return ar.EKSListNodegroups(ctx, region, clusterName)
}

// deletes an eks node group
func DeleteNodeGroup(ctx context.Context, ar runner.AWSRunner, region, clusterName, nodeGroupName string) error {
	return ar.EKSDeleteNodegroup(ctx, region, clusterName, nodeGroupName)
}

// deletes an eks cluster
func DeleteEKSCluster(ctx context.Context, ar runner.AWSRunner, region, name string) error {
	return ar.EKSDeleteCluster(ctx, region, name)
}

// EFSInfo holds efs file system details
type EFSInfo struct {
	ID   string
	Name string
}

// returns all efs file systems
func ListEFSFileSystems(ctx context.Context, ar runner.AWSRunner, region string) ([]EFSInfo, error) {
	filesystems, err := ar.EFSDescribeFileSystems(ctx, region, "")
	if err != nil {
		return nil, err
	}
	result := make([]EFSInfo, 0, len(filesystems))
	for _, fs := range filesystems {
		result = append(result, EFSInfo{
			ID:   fs.FileSystemID,
			Name: fs.Name,
		})
	}
	return result, nil
}

// returns mount target ids for an EFS
func ListMountTargets(ctx context.Context, ar runner.AWSRunner, region, fileSystemID string) ([]string, error) {
	targets, err := ar.EFSDescribeMountTargets(ctx, region, fileSystemID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(targets))
	for _, mt := range targets {
		ids = append(ids, mt.MountTargetID)
	}
	return ids, nil
}

// deletes an efs mount target
func DeleteMountTarget(ctx context.Context, ar runner.AWSRunner, region, mountTargetID string) error {
	return ar.EFSDeleteMountTarget(ctx, region, mountTargetID)
}

// deletes an efs file system
func DeleteEFSFileSystem(ctx context.Context, ar runner.AWSRunner, region, fileSystemID string) error {
	return ar.EFSDeleteFileSystem(ctx, region, fileSystemID)
}

// checks if an RDS instance exists
func RDSInstanceExists(ctx context.Context, ar runner.AWSRunner, region, identifier string) bool {
	exists, err := ar.RDSInstanceExists(ctx, region, identifier)
	if err != nil {
		return false
	}
	return exists
}

// deletes an RDS instance
func DeleteRDSInstance(ctx context.Context, ar runner.AWSRunner, region, identifier string) error {
	return ar.RDSDeleteInstance(ctx, region, identifier)
}

// returns all S3 bucket names
func ListS3Buckets(ctx context.Context, ar runner.AWSRunner) ([]string, error) {
	return ar.S3ListBuckets(ctx)
}

// removes all objects from an S3 bucket
func EmptyS3Bucket(ctx context.Context, ar runner.AWSRunner, bucket string) error {
	return ar.S3EmptyBucket(ctx, bucket)
}

// deletes an S3 bucket
func DeleteS3Bucket(ctx context.Context, ar runner.AWSRunner, bucket string) error {
	return ar.S3DeleteBucket(ctx, bucket)
}

// returns vpc ids matching the namespace
func ListVPCs(ctx context.Context, ar runner.AWSRunner, region, namespace string) ([]string, error) {
	return ar.EC2DescribeVPCs(ctx, region, namespace)
}

// returns resource IDs for a vpc
func ListVPCResources(ctx context.Context, ar runner.AWSRunner, region, vpcID, resourceType string) ([]string, error) {
	return ar.EC2DescribeVPCResources(ctx, region, vpcID, resourceType)
}

// detaches an internet gateway from a vpc
func DetachInternetGateway(ctx context.Context, ar runner.AWSRunner, region, igwID, vpcID string) error {
	return ar.EC2DetachInternetGateway(ctx, region, igwID, vpcID)
}

// deletes an internet gateway
func DeleteInternetGateway(ctx context.Context, ar runner.AWSRunner, region, igwID string) error {
	return ar.EC2DeleteInternetGateway(ctx, region, igwID)
}

// deletes a subnet
func DeleteSubnet(ctx context.Context, ar runner.AWSRunner, region, subnetID string) error {
	return ar.EC2DeleteSubnet(ctx, region, subnetID)
}

// deletes a route table
func DeleteRouteTable(ctx context.Context, ar runner.AWSRunner, region, routeTableID string) error {
	return ar.EC2DeleteRouteTable(ctx, region, routeTableID)
}

// deletes a security group
func DeleteSecurityGroup(ctx context.Context, ar runner.AWSRunner, region, groupID string) error {
	return ar.EC2DeleteSecurityGroup(ctx, region, groupID)
}

// deletes a VPC
func DeleteVPC(ctx context.Context, ar runner.AWSRunner, region, vpcID string) error {
	return ar.EC2DeleteVPC(ctx, region, vpcID)
}
