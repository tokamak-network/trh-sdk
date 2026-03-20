package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// ELBDescribeLoadBalancers returns classic load balancer names matching a namespace filter.
func (r *NativeAWSRunner) ELBDescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := r.elbClient(region).DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, fmt.Errorf("aws ELBDescribeLoadBalancers: %w", err)
	}
	filtered := make([]string, 0)
	for _, lb := range out.LoadBalancerDescriptions {
		name := aws.ToString(lb.LoadBalancerName)
		if strings.Contains(name, namespaceFilter) {
			filtered = append(filtered, name)
		}
	}
	return filtered, nil
}

// ELBDeleteLoadBalancer deletes a classic load balancer.
func (r *NativeAWSRunner) ELBDeleteLoadBalancer(ctx context.Context, region, name string) error {
	_, err := r.elbClient(region).DeleteLoadBalancer(ctx, &elasticloadbalancing.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("aws ELBDeleteLoadBalancer: %w", err)
	}
	return nil
}

// ELBv2DescribeLoadBalancers returns ALB/NLB ARNs matching a namespace filter.
func (r *NativeAWSRunner) ELBv2DescribeLoadBalancers(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := r.elbv2Client(region).DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, fmt.Errorf("aws ELBv2DescribeLoadBalancers: %w", err)
	}
	filtered := make([]string, 0)
	for _, lb := range out.LoadBalancers {
		name := aws.ToString(lb.LoadBalancerName)
		if strings.Contains(name, namespaceFilter) {
			filtered = append(filtered, aws.ToString(lb.LoadBalancerArn))
		}
	}
	return filtered, nil
}

// ELBv2DeleteLoadBalancer deletes an ALB/NLB.
func (r *NativeAWSRunner) ELBv2DeleteLoadBalancer(ctx context.Context, region, arn string) error {
	_, err := r.elbv2Client(region).DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(arn),
	})
	if err != nil {
		return fmt.Errorf("aws ELBv2DeleteLoadBalancer: %w", err)
	}
	return nil
}

// EKSClusterExists checks if an EKS cluster exists.
func (r *NativeAWSRunner) EKSClusterExists(ctx context.Context, region, name string) (bool, error) {
	_, err := r.eksClient(region).DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(name),
	})
	if err != nil {
		var rnfe *ekstypes.ResourceNotFoundException
		if errors.As(err, &rnfe) {
			return false, nil
		}
		return false, fmt.Errorf("aws EKSClusterExists: %w", err)
	}
	return true, nil
}

// EKSListNodegroups returns node group names for an EKS cluster.
func (r *NativeAWSRunner) EKSListNodegroups(ctx context.Context, region, clusterName string) ([]string, error) {
	out, err := r.eksClient(region).ListNodegroups(ctx, &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("aws EKSListNodegroups: %w", err)
	}
	return out.Nodegroups, nil
}

// EKSDeleteNodegroup deletes an EKS node group.
func (r *NativeAWSRunner) EKSDeleteNodegroup(ctx context.Context, region, clusterName, nodeGroupName string) error {
	_, err := r.eksClient(region).DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroupName),
	})
	if err != nil {
		return fmt.Errorf("aws EKSDeleteNodegroup: %w", err)
	}
	return nil
}

// EKSDeleteCluster deletes an EKS cluster.
func (r *NativeAWSRunner) EKSDeleteCluster(ctx context.Context, region, name string) error {
	_, err := r.eksClient(region).DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("aws EKSDeleteCluster: %w", err)
	}
	return nil
}

// RDSInstanceExists checks if an RDS instance exists.
func (r *NativeAWSRunner) RDSInstanceExists(ctx context.Context, region, identifier string) (bool, error) {
	_, err := r.rdsClient(region).DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(identifier),
	})
	if err != nil {
		if strings.Contains(err.Error(), "DBInstanceNotFound") {
			return false, nil
		}
		return false, fmt.Errorf("aws RDSInstanceExists: %w", err)
	}
	return true, nil
}

// RDSDeleteInstance deletes an RDS instance (skip-final-snapshot, delete-automated-backups).
func (r *NativeAWSRunner) RDSDeleteInstance(ctx context.Context, region, identifier string) error {
	_, err := r.rdsClient(region).DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   aws.String(identifier),
		SkipFinalSnapshot:      aws.Bool(true),
		DeleteAutomatedBackups: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("aws RDSDeleteInstance: %w", err)
	}
	return nil
}
