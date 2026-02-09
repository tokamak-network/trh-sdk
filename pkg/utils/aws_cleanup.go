package utils

import (
	"context"
	"encoding/json"
	"fmt"
)

//returns classic ELB names matching the namespace
func ListClassicLoadBalancers(ctx context.Context, region, namespace string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "elb", "describe-load-balancers",
		"--region", region,
		"--query", fmt.Sprintf("LoadBalancerDescriptions[?contains(LoadBalancerName, '%s')].LoadBalancerName", namespace),
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var names []string
	if err := json.Unmarshal([]byte(out), &names); err != nil {
		return nil, err
	}
	return names, nil
}

//this deletes a classic ELB
func DeleteClassicLoadBalancer(ctx context.Context, region, name string) error {
	_, err := ExecuteCommand(ctx, "aws", "elb", "delete-load-balancer",
		"--region", region, "--load-balancer-name", name)
	return err
}

//returns ALB/NLB ARNs matching the namespace
func ListALBLoadBalancers(ctx context.Context, region, namespace string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "elbv2", "describe-load-balancers",
		"--region", region,
		"--query", fmt.Sprintf("LoadBalancers[?contains(LoadBalancerName, '%s')].LoadBalancerArn", namespace),
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var arns []string
	if err := json.Unmarshal([]byte(out), &arns); err != nil {
		return nil, err
	}
	return arns, nil
}

//this deletes an ALB/NLB
func DeleteALBLoadBalancer(ctx context.Context, region, arn string) error {
	_, err := ExecuteCommand(ctx, "aws", "elbv2", "delete-load-balancer",
		"--region", region, "--load-balancer-arn", arn)
	return err
}

//this will holds elastic IP details
type ElasticIPInfo struct {
	ID    string `json:"ID"`
	Assoc string `json:"Assoc"`
}

//returns elastic IPs matching the namespace
func ListElasticIPs(ctx context.Context, region, namespace string) ([]ElasticIPInfo, error) {
	out, err := ExecuteCommand(ctx, "aws", "ec2", "describe-addresses",
		"--region", region,
		"--query", fmt.Sprintf("Addresses[?Tags[?Key=='Name' && contains(Value, '%s')]].{ID:AllocationId,Assoc:AssociationId}", namespace),
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var eips []ElasticIPInfo
	if err := json.Unmarshal([]byte(out), &eips); err != nil {
		return nil, err
	}
	return eips, nil
}

//disassociates an elastic ip
func DisassociateAddress(ctx context.Context, region, associationID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "disassociate-address",
		"--region", region, "--association-id", associationID)
	return err
}

//releases an elastic ip
func ReleaseAddress(ctx context.Context, region, allocationID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "release-address",
		"--region", region, "--allocation-id", allocationID)
	return err
}

//returns nat gateway ids matching the namespace
func ListNATGateways(ctx context.Context, region, namespace string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "ec2", "describe-nat-gateways",
		"--region", region,
		"--filter", "Name=state,Values=available,pending",
		"--query", fmt.Sprintf("NatGateways[?Tags[?Key=='Name' && contains(Value, '%s')]].NatGatewayId", namespace),
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

//this deletes a nat gateway
func DeleteNATGateway(ctx context.Context, region, id string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-nat-gateway",
		"--region", region, "--nat-gateway-id", id)
	return err
}

//this checks if an eks cluster exists
func EKSClusterExists(ctx context.Context, region, name string) bool {
	_, err := ExecuteCommand(ctx, "aws", "eks", "describe-cluster",
		"--region", region, "--name", name)
	return err == nil
}

//returns node group names for a cluster
func ListNodeGroups(ctx context.Context, region, clusterName string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "eks", "list-nodegroups",
		"--region", region, "--cluster-name", clusterName,
		"--query", "nodegroups", "--output", "json")
	if err != nil {
		return nil, err
	}

	var nodeGroups []string
	if err := json.Unmarshal([]byte(out), &nodeGroups); err != nil {
		return nil, err
	}
	return nodeGroups, nil
}

//deletes an eks node group
func DeleteNodeGroup(ctx context.Context, region, clusterName, nodeGroupName string) error {
	_, err := ExecuteCommand(ctx, "aws", "eks", "delete-nodegroup",
		"--region", region, "--cluster-name", clusterName, "--nodegroup-name", nodeGroupName)
	return err
}

//deletes an eks cluster
func DeleteEKSCluster(ctx context.Context, region, name string) error {
	_, err := ExecuteCommand(ctx, "aws", "eks", "delete-cluster",
		"--region", region, "--name", name)
	return err
}

//holds efs file system details
type EFSInfo struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

//returns all efs file systems
func ListEFSFileSystems(ctx context.Context, region string) ([]EFSInfo, error) {
	out, err := ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--query", "FileSystems[].{ID:FileSystemId,Name:Name}",
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var filesystems []EFSInfo
	if err := json.Unmarshal([]byte(out), &filesystems); err != nil {
		return nil, err
	}
	return filesystems, nil
}

//returns mount target ids for an EFS
func ListMountTargets(ctx context.Context, region, fileSystemID string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets",
		"--region", region, "--file-system-id", fileSystemID,
		"--query", "MountTargets[].MountTargetId", "--output", "json")
	if err != nil {
		return nil, err
	}

	var mountTargets []string
	if err := json.Unmarshal([]byte(out), &mountTargets); err != nil {
		return nil, err
	}
	return mountTargets, nil
}

//deletes an efs mount target
func DeleteMountTarget(ctx context.Context, region, mountTargetID string) error {
	_, err := ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
		"--region", region, "--mount-target-id", mountTargetID)
	return err
}

//deletes an efs file system
func DeleteEFSFileSystem(ctx context.Context, region, fileSystemID string) error {
	_, err := ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
		"--region", region, "--file-system-id", fileSystemID)
	return err
}

//checks if an RDS instance exists
func RDSInstanceExists(ctx context.Context, region, identifier string) bool {
	_, err := ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", region, "--db-instance-identifier", identifier)
	return err == nil
}

//deletes an RDS instance
func DeleteRDSInstance(ctx context.Context, region, identifier string) error {
	_, err := ExecuteCommand(ctx, "aws", "rds", "delete-db-instance",
		"--region", region, "--db-instance-identifier", identifier,
		"--skip-final-snapshot", "--delete-automated-backups")
	return err
}

//returns all S3 bucket names
func ListS3Buckets(ctx context.Context) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "s3api", "list-buckets",
		"--query", "Buckets[].Name", "--output", "json")
	if err != nil {
		return nil, err
	}

	var buckets []string
	if err := json.Unmarshal([]byte(out), &buckets); err != nil {
		return nil, err
	}
	return buckets, nil
}

//removes all objects from an S3 bucket
func EmptyS3Bucket(ctx context.Context, bucket string) error {
	_, err := ExecuteCommand(ctx, "aws", "s3", "rm", fmt.Sprintf("s3://%s", bucket), "--recursive")
	return err
}

//deletes an S3 bucket
func DeleteS3Bucket(ctx context.Context, bucket string) error {
	_, err := ExecuteCommand(ctx, "aws", "s3", "rb", fmt.Sprintf("s3://%s", bucket), "--force")
	return err
}

//returns vpc ids matching the namespace
func ListVPCs(ctx context.Context, region, namespace string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "ec2", "describe-vpcs",
		"--region", region,
		"--query", fmt.Sprintf("Vpcs[?Tags[?Key=='Name' && contains(Value, '%s')]].VpcId", namespace),
		"--output", "json")
	if err != nil {
		return nil, err
	}

	var vpcIDs []string
	if err := json.Unmarshal([]byte(out), &vpcIDs); err != nil {
		return nil, err
	}
	return vpcIDs, nil
}

//returns resource IDs for a vpc
func ListVPCResources(ctx context.Context, region, vpcID, resourceType, query string) ([]string, error) {
	out, err := ExecuteCommand(ctx, "aws", "ec2", fmt.Sprintf("describe-%s", resourceType),
		"--region", region,
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcID),
		"--query", query, "--output", "json")
	if err != nil {
		return nil, err
	}

	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

//detaches an internet gateway from a vpc
func DetachInternetGateway(ctx context.Context, region, igwID, vpcID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "detach-internet-gateway",
		"--region", region, "--internet-gateway-id", igwID, "--vpc-id", vpcID)
	return err
}

//deletes an internet gateway
func DeleteInternetGateway(ctx context.Context, region, igwID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-internet-gateway",
		"--region", region, "--internet-gateway-id", igwID)
	return err
}

//deletes a subnet
func DeleteSubnet(ctx context.Context, region, subnetID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-subnet",
		"--region", region, "--subnet-id", subnetID)
	return err
}

//delete a route table
func DeleteRouteTable(ctx context.Context, region, routeTableID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-route-table",
		"--region", region, "--route-table-id", routeTableID)
	return err
}

//deletes a security group
func DeleteSecurityGroup(ctx context.Context, region, groupID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-security-group",
		"--region", region, "--group-id", groupID)
	return err
}

//deletes a VPC
func DeleteVPC(ctx context.Context, region, vpcID string) error {
	_, err := ExecuteCommand(ctx, "aws", "ec2", "delete-vpc",
		"--region", region, "--vpc-id", vpcID)
	return err
}
