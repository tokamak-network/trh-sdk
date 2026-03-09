package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// EC2DescribeSubnetState returns the state of a subnet.
func (r *ShellOutAWSRunner) EC2DescribeSubnetState(ctx context.Context, region, subnetID string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-subnets",
		"--subnet-ids", subnetID,
		"--region", region,
		"--query", "Subnets[0].State",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("aws EC2DescribeSubnetState: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// EC2DescribeAddresses returns elastic IPs matching a namespace tag filter.
func (r *ShellOutAWSRunner) EC2DescribeAddresses(ctx context.Context, region, namespaceFilter string) ([]ElasticIPInfo, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-addresses",
		"--region", region,
		"--filters", fmt.Sprintf("Name=tag:Namespace,Values=%s", namespaceFilter),
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeAddresses: %w", err)
	}
	var result struct {
		Addresses []struct {
			AllocationID  string `json:"AllocationId"`
			AssociationID string `json:"AssociationId"`
		} `json:"Addresses"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws EC2DescribeAddresses: parse response: %w", err)
	}
	addrs := make([]ElasticIPInfo, 0, len(result.Addresses))
	for _, a := range result.Addresses {
		addrs = append(addrs, ElasticIPInfo{
			AllocationID:  a.AllocationID,
			AssociationID: a.AssociationID,
		})
	}
	return addrs, nil
}

// EC2DisassociateAddress disassociates an elastic IP.
func (r *ShellOutAWSRunner) EC2DisassociateAddress(ctx context.Context, region, associationID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "disassociate-address",
		"--association-id", associationID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DisassociateAddress: %w", err)
	}
	return nil
}

// EC2ReleaseAddress releases an elastic IP.
func (r *ShellOutAWSRunner) EC2ReleaseAddress(ctx context.Context, region, allocationID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "release-address",
		"--allocation-id", allocationID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2ReleaseAddress: %w", err)
	}
	return nil
}

// EC2DescribeNATGateways returns NAT gateway IDs matching a namespace tag filter.
func (r *ShellOutAWSRunner) EC2DescribeNATGateways(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-nat-gateways",
		"--region", region,
		"--filter", fmt.Sprintf("Name=tag:Namespace,Values=%s", namespaceFilter),
		"--query", "NatGateways[].NatGatewayId",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeNATGateways: %w", err)
	}
	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		return nil, fmt.Errorf("aws EC2DescribeNATGateways: parse response: %w", err)
	}
	return ids, nil
}

// EC2DeleteNATGateway deletes a NAT gateway.
func (r *ShellOutAWSRunner) EC2DeleteNATGateway(ctx context.Context, region, natGatewayID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-nat-gateway",
		"--nat-gateway-id", natGatewayID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteNATGateway: %w", err)
	}
	return nil
}

// EC2DescribeVPCs returns VPC IDs matching a namespace tag filter.
func (r *ShellOutAWSRunner) EC2DescribeVPCs(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-vpcs",
		"--region", region,
		"--filters", fmt.Sprintf("Name=tag:Namespace,Values=%s", namespaceFilter),
		"--query", "Vpcs[].VpcId",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeVPCs: %w", err)
	}
	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		return nil, fmt.Errorf("aws EC2DescribeVPCs: parse response: %w", err)
	}
	return ids, nil
}

// EC2DescribeVPCResources returns resource IDs for a VPC (subnets, route-tables, etc).
func (r *ShellOutAWSRunner) EC2DescribeVPCResources(ctx context.Context, region, vpcID, resourceType string) ([]string, error) {
	var queryField string
	switch resourceType {
	case "subnets":
		queryField = "Subnets[].SubnetId"
	case "internet-gateways":
		queryField = "InternetGateways[].InternetGatewayId"
	case "route-tables":
		queryField = "RouteTables[].RouteTableId"
	case "security-groups":
		queryField = "SecurityGroups[].GroupId"
	default:
		return nil, fmt.Errorf("aws EC2DescribeVPCResources: unsupported resource type: %s", resourceType)
	}
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", fmt.Sprintf("describe-%s", resourceType),
		"--region", region,
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcID),
		"--query", queryField,
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeVPCResources: %w", err)
	}
	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		return nil, fmt.Errorf("aws EC2DescribeVPCResources: parse response: %w", err)
	}
	return ids, nil
}

// EC2DetachInternetGateway detaches an internet gateway from a VPC.
func (r *ShellOutAWSRunner) EC2DetachInternetGateway(ctx context.Context, region, igwID, vpcID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "detach-internet-gateway",
		"--internet-gateway-id", igwID,
		"--vpc-id", vpcID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DetachInternetGateway: %w", err)
	}
	return nil
}

// EC2DeleteInternetGateway deletes an internet gateway.
func (r *ShellOutAWSRunner) EC2DeleteInternetGateway(ctx context.Context, region, igwID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-internet-gateway",
		"--internet-gateway-id", igwID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteInternetGateway: %w", err)
	}
	return nil
}

// EC2DeleteSubnet deletes a subnet.
func (r *ShellOutAWSRunner) EC2DeleteSubnet(ctx context.Context, region, subnetID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-subnet",
		"--subnet-id", subnetID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteSubnet: %w", err)
	}
	return nil
}

// EC2DeleteRouteTable deletes a route table.
func (r *ShellOutAWSRunner) EC2DeleteRouteTable(ctx context.Context, region, routeTableID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-route-table",
		"--route-table-id", routeTableID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteRouteTable: %w", err)
	}
	return nil
}

// EC2DeleteSecurityGroup deletes a security group.
func (r *ShellOutAWSRunner) EC2DeleteSecurityGroup(ctx context.Context, region, groupID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-security-group",
		"--group-id", groupID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteSecurityGroup: %w", err)
	}
	return nil
}

// EC2DeleteVPC deletes a VPC.
func (r *ShellOutAWSRunner) EC2DeleteVPC(ctx context.Context, region, vpcID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-vpc",
		"--vpc-id", vpcID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EC2DeleteVPC: %w", err)
	}
	return nil
}

// ELBDescribeLoadBalancers returns classic load balancer names matching a namespace filter.
func (r *ShellOutAWSRunner) ELBDescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "elb", "describe-load-balancers",
		"--region", region,
		"--query", "LoadBalancerDescriptions[].LoadBalancerName",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws ELBDescribeLoadBalancers: %w", err)
	}
	var allNames []string
	if err := json.Unmarshal([]byte(out), &allNames); err != nil {
		return nil, fmt.Errorf("aws ELBDescribeLoadBalancers: parse response: %w", err)
	}
	filtered := make([]string, 0)
	for _, name := range allNames {
		if strings.Contains(name, namespaceFilter) {
			filtered = append(filtered, name)
		}
	}
	return filtered, nil
}

// ELBDeleteLoadBalancer deletes a classic load balancer.
func (r *ShellOutAWSRunner) ELBDeleteLoadBalancer(ctx context.Context, region, name string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "elb", "delete-load-balancer",
		"--load-balancer-name", name,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws ELBDeleteLoadBalancer: %w", err)
	}
	return nil
}

// ELBv2DescribeLoadBalancers returns ALB/NLB ARNs matching a namespace filter.
func (r *ShellOutAWSRunner) ELBv2DescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "elbv2", "describe-load-balancers",
		"--region", region,
		"--query", "LoadBalancers[].[LoadBalancerArn,LoadBalancerName]",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws ELBv2DescribeLoadBalancers: %w", err)
	}
	var pairs [][]string
	if err := json.Unmarshal([]byte(out), &pairs); err != nil {
		return nil, fmt.Errorf("aws ELBv2DescribeLoadBalancers: parse response: %w", err)
	}
	filtered := make([]string, 0)
	for _, pair := range pairs {
		if len(pair) >= 2 && strings.Contains(pair[1], namespaceFilter) {
			filtered = append(filtered, pair[0])
		}
	}
	return filtered, nil
}

// ELBv2DeleteLoadBalancer deletes an ALB/NLB.
func (r *ShellOutAWSRunner) ELBv2DeleteLoadBalancer(ctx context.Context, region, arn string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "elbv2", "delete-load-balancer",
		"--load-balancer-arn", arn,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws ELBv2DeleteLoadBalancer: %w", err)
	}
	return nil
}

// EKSClusterExists checks if an EKS cluster exists.
func (r *ShellOutAWSRunner) EKSClusterExists(ctx context.Context, region, name string) (bool, error) {
	_, err := utils.ExecuteCommand(ctx, "aws", "eks", "describe-cluster",
		"--name", name,
		"--region", region)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("aws EKSClusterExists: %w", err)
	}
	return true, nil
}

// EKSListNodegroups returns node group names for an EKS cluster.
func (r *ShellOutAWSRunner) EKSListNodegroups(ctx context.Context, region, clusterName string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "eks", "list-nodegroups",
		"--cluster-name", clusterName,
		"--region", region,
		"--query", "nodegroups",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws EKSListNodegroups: %w", err)
	}
	var names []string
	if err := json.Unmarshal([]byte(out), &names); err != nil {
		return nil, fmt.Errorf("aws EKSListNodegroups: parse response: %w", err)
	}
	return names, nil
}

// EKSDeleteNodegroup deletes an EKS node group.
func (r *ShellOutAWSRunner) EKSDeleteNodegroup(ctx context.Context, region, clusterName, nodeGroupName string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "eks", "delete-nodegroup",
		"--cluster-name", clusterName,
		"--nodegroup-name", nodeGroupName,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EKSDeleteNodegroup: %w", err)
	}
	return nil
}

// EKSDeleteCluster deletes an EKS cluster.
func (r *ShellOutAWSRunner) EKSDeleteCluster(ctx context.Context, region, name string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "eks", "delete-cluster",
		"--name", name,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EKSDeleteCluster: %w", err)
	}
	return nil
}

// RDSInstanceExists checks if an RDS instance exists.
func (r *ShellOutAWSRunner) RDSInstanceExists(ctx context.Context, region, identifier string) (bool, error) {
	_, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--db-instance-identifier", identifier,
		"--region", region)
	if err != nil {
		if strings.Contains(err.Error(), "DBInstanceNotFound") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("aws RDSInstanceExists: %w", err)
	}
	return true, nil
}

// RDSDeleteInstance deletes an RDS instance (skip-final-snapshot, delete-automated-backups).
func (r *ShellOutAWSRunner) RDSDeleteInstance(ctx context.Context, region, identifier string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "rds", "delete-db-instance",
		"--db-instance-identifier", identifier,
		"--skip-final-snapshot",
		"--delete-automated-backups",
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws RDSDeleteInstance: %w", err)
	}
	return nil
}

// S3ListBuckets returns all S3 bucket names.
func (r *ShellOutAWSRunner) S3ListBuckets(ctx context.Context) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "s3api", "list-buckets",
		"--query", "Buckets[].Name",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws S3ListBuckets: %w", err)
	}
	var names []string
	if err := json.Unmarshal([]byte(out), &names); err != nil {
		return nil, fmt.Errorf("aws S3ListBuckets: parse response: %w", err)
	}
	return names, nil
}

// S3EmptyBucket removes all objects from an S3 bucket.
func (r *ShellOutAWSRunner) S3EmptyBucket(ctx context.Context, bucket string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "s3", "rm", "s3://"+bucket, "--recursive")
	if err != nil {
		return fmt.Errorf("aws S3EmptyBucket: %w", err)
	}
	return nil
}

// S3DeleteBucket deletes an S3 bucket (force).
func (r *ShellOutAWSRunner) S3DeleteBucket(ctx context.Context, bucket string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "s3", "rb", "s3://"+bucket, "--force")
	if err != nil {
		return fmt.Errorf("aws S3DeleteBucket: %w", err)
	}
	return nil
}

// LogsDescribeLogGroups checks if a log group exists.
func (r *ShellOutAWSRunner) LogsDescribeLogGroups(ctx context.Context, region, logGroupNamePrefix string) (bool, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "logs", "describe-log-groups",
		"--log-group-name-prefix", logGroupNamePrefix,
		"--region", region,
		"--query", "logGroups",
		"--output", "json")
	if err != nil {
		return false, fmt.Errorf("aws LogsDescribeLogGroups: %w", err)
	}
	var groups []json.RawMessage
	if err := json.Unmarshal([]byte(out), &groups); err != nil {
		return false, fmt.Errorf("aws LogsDescribeLogGroups: parse response: %w", err)
	}
	return len(groups) > 0, nil
}

// LogsPutRetentionPolicy sets retention policy on a log group.
func (r *ShellOutAWSRunner) LogsPutRetentionPolicy(ctx context.Context, region, logGroupName string, retentionDays int) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "logs", "put-retention-policy",
		"--log-group-name", logGroupName,
		"--retention-in-days", fmt.Sprintf("%d", retentionDays),
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws LogsPutRetentionPolicy: %w", err)
	}
	return nil
}

// CheckVersion verifies the aws CLI is available.
func (r *ShellOutAWSRunner) CheckVersion(ctx context.Context) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "--version")
	if err != nil {
		return fmt.Errorf("aws CheckVersion: %w", err)
	}
	return nil
}
