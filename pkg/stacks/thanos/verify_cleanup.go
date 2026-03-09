package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Timing constants for AWS resource deletion propagation delays.
const (
	eipDisassociationGrace   = 2 * time.Second
	natGwDeletionPropagation = 30 * time.Second
	mountTargetDeletionGrace = 10 * time.Second
	vpcDependencyGrace       = 5 * time.Second
	nodeGroupPollInterval    = 30 * time.Second
	nodeGroupPollMaxAttempts = 20 // ~10-minute timeout
)

// VerifyAndCleanupResources scans for orphaned AWS resources after destroy and removes them.
// Resources are identified by namespace tag or name pattern following Terraform conventions.
func (t *ThanosStack) VerifyAndCleanupResources(ctx context.Context, namespace string) error {
	region := t.awsProfile.AwsConfig.Region
	t.logger.Infof("Verifying resource cleanup for namespace: %s in region: %s", namespace, region)

	var cleaned, failed int

	// Delete in dependency order (reverse of creation).
	cleaned, failed = t.deleteOrphanedLoadBalancers(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedElasticIPs(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedNATGateways(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedEKS(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedEFS(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedRDS(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedS3(ctx, region, namespace, cleaned, failed)
	cleaned, failed = t.deleteOrphanedVPC(ctx, region, namespace, cleaned, failed)

	if cleaned == 0 && failed == 0 {
		t.logger.Info("No orphaned resources found")
	} else {
		t.logger.Infof("Cleanup complete: %d deleted, %d failed", cleaned, failed)
	}

	if failed > 0 {
		return fmt.Errorf("cleanup failed for %d resource(s)", failed)
	}
	return nil
}

func (t *ThanosStack) deleteOrphanedLoadBalancers(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	// Classic ELB — independent of ALB/NLB; a listing error here does not skip other categories.
	names, err := t.awsRunner.ELBDescribeLoadBalancers(ctx, region, namespace)
	if err == nil {
		for _, name := range names {
			t.logger.Infof("Deleting ELB: %s", name)
			if err := t.awsRunner.ELBDeleteLoadBalancer(ctx, region, name); err != nil {
				t.logger.Warnf("Failed to delete ELB %s: %v", name, err)
				failed++
			} else {
				cleaned++
			}
		}
	} else {
		t.logger.Warnf("Failed to list Classic ELBs: %v", err)
	}

	// ALB / NLB
	arns, err := t.awsRunner.ELBv2DescribeLoadBalancers(ctx, region, namespace)
	if err != nil {
		t.logger.Warnf("Failed to list ALB/NLBs: %v", err)
		return cleaned, failed
	}
	for _, arn := range arns {
		t.logger.Infof("Deleting ALB/NLB: %s", arn)
		if err := t.awsRunner.ELBv2DeleteLoadBalancer(ctx, region, arn); err != nil {
			t.logger.Warnf("Failed to delete ALB/NLB: %v", err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedElasticIPs(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	eips, err := t.awsRunner.EC2DescribeAddresses(ctx, region, namespace)
	if err != nil {
		return cleaned, failed
	}

	for _, eip := range eips {
		if eip.AssociationID != "" {
			if err := t.awsRunner.EC2DisassociateAddress(ctx, region, eip.AssociationID); err != nil {
				t.logger.Warnf("Failed to disassociate address %s: %v", eip.AssociationID, err)
			}
			select {
			case <-ctx.Done():
				return cleaned, failed
			case <-time.After(eipDisassociationGrace):
			}
		}

		t.logger.Infof("Releasing Elastic IP: %s", eip.AllocationID)
		if err := t.awsRunner.EC2ReleaseAddress(ctx, region, eip.AllocationID); err != nil {
			t.logger.Warnf("Failed to release Elastic IP %s: %v", eip.AllocationID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedNATGateways(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	ids, err := t.awsRunner.EC2DescribeNATGateways(ctx, region, namespace)
	if err != nil {
		t.logger.Warnf("Failed to list NAT Gateways: %v", err)
		return cleaned, failed
	}
	if len(ids) == 0 {
		return cleaned, failed
	}

	for _, id := range ids {
		t.logger.Infof("Deleting NAT Gateway: %s", id)
		if err := t.awsRunner.EC2DeleteNATGateway(ctx, region, id); err != nil {
			t.logger.Warnf("Failed to delete NAT Gateway %s: %v", id, err)
			failed++
		} else {
			cleaned++
		}
	}

	t.logger.Info("Waiting for NAT Gateway deletion...")
	select {
	case <-ctx.Done():
		return cleaned, failed
	case <-time.After(natGwDeletionPropagation):
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedEKS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	// EKS cluster name matches namespace by TRH convention.
	exists, err := t.awsRunner.EKSClusterExists(ctx, region, namespace)
	if err != nil || !exists {
		return cleaned, failed
	}

	// Delete node groups first.
	nodeGroups, err := t.awsRunner.EKSListNodegroups(ctx, region, namespace)
	if err == nil {
		for _, ng := range nodeGroups {
			t.logger.Infof("Deleting node group: %s", ng)
			if err := t.awsRunner.EKSDeleteNodegroup(ctx, region, namespace, ng); err != nil {
				t.logger.Warnf("Failed to delete node group %s: %v", ng, err)
				failed++
			}
		}

		if len(nodeGroups) > 0 {
			t.waitForNodeGroupsDeletion(ctx, region, namespace)
		}
	}

	t.logger.Infof("Deleting EKS cluster: %s", namespace)
	if err := t.awsRunner.EKSDeleteCluster(ctx, region, namespace); err != nil {
		t.logger.Warnf("Failed to delete EKS cluster: %v", err)
		failed++
	} else {
		cleaned++
	}

	return cleaned, failed
}

func (t *ThanosStack) waitForNodeGroupsDeletion(ctx context.Context, region, cluster string) {
	t.logger.Info("Waiting for node groups deletion...")
	ticker := time.NewTicker(nodeGroupPollInterval)
	defer ticker.Stop()

	for i := 0; i < nodeGroupPollMaxAttempts; i++ {
		// Check first, then wait — avoids an unnecessary 30-second delay on
		// the first iteration when node groups may already be gone.
		ngs, err := t.awsRunner.EKSListNodegroups(ctx, region, cluster)
		if err != nil {
			// Transient error — keep polling rather than treating as done.
			t.logger.Warnf("Failed to list node groups (attempt %d/%d): %v", i+1, nodeGroupPollMaxAttempts, err)
		} else if len(ngs) == 0 {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
	t.logger.Warnf("Node groups may not have fully deleted after %d attempts", nodeGroupPollMaxAttempts)
}

func (t *ThanosStack) deleteOrphanedEFS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	filesystems, err := t.awsRunner.EFSDescribeFileSystems(ctx, region, "")
	if err != nil {
		return cleaned, failed
	}

	for _, fs := range filesystems {
		if !strings.HasPrefix(fs.Name, namespace+"-") && fs.Name != namespace {
			continue
		}

		// Delete mount targets first.
		mountTargets, err := t.awsRunner.EFSDescribeMountTargets(ctx, region, fs.FileSystemID)
		if err == nil {
			for _, mt := range mountTargets {
				if err := t.awsRunner.EFSDeleteMountTarget(ctx, region, mt.MountTargetID); err != nil {
					t.logger.Warnf("Failed to delete mount target %s: %v", mt.MountTargetID, err)
				}
			}
			if len(mountTargets) > 0 {
				select {
				case <-ctx.Done():
					return cleaned, failed
				case <-time.After(mountTargetDeletionGrace):
				}
			}
		} else {
			t.logger.Warnf("Failed to list mount targets for EFS %s: %v", fs.FileSystemID, err)
		}

		t.logger.Infof("Deleting EFS: %s (%s)", fs.Name, fs.FileSystemID)
		if err := t.awsRunner.EFSDeleteFileSystem(ctx, region, fs.FileSystemID); err != nil {
			t.logger.Warnf("Failed to delete EFS %s: %v", fs.FileSystemID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedRDS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	identifier := fmt.Sprintf("%s-rds", namespace)

	exists, err := t.awsRunner.RDSInstanceExists(ctx, region, identifier)
	if err != nil || !exists {
		return cleaned, failed
	}

	t.logger.Infof("Deleting RDS: %s", identifier)
	if err := t.awsRunner.RDSDeleteInstance(ctx, region, identifier); err != nil {
		t.logger.Warnf("Failed to delete RDS %s: %v", identifier, err)
		failed++
	} else {
		cleaned++
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedS3(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	buckets, err := t.awsRunner.S3ListBuckets(ctx)
	if err != nil {
		return cleaned, failed
	}

	for _, bucket := range buckets {
		if !strings.HasPrefix(bucket, namespace+"-") && bucket != namespace {
			continue
		}

		t.logger.Infof("Deleting S3 bucket: %s", bucket)
		if err := t.awsRunner.S3EmptyBucket(ctx, bucket); err != nil {
			t.logger.Warnf("Failed to empty S3 bucket %s: %v", bucket, err)
		}

		if err := t.awsRunner.S3DeleteBucket(ctx, bucket); err != nil {
			t.logger.Warnf("Failed to delete S3 bucket %s: %v", bucket, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedVPC(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	vpcIDs, err := t.awsRunner.EC2DescribeVPCs(ctx, region, namespace)
	if err != nil || len(vpcIDs) == 0 {
		return cleaned, failed
	}

	for _, vpcID := range vpcIDs {
		t.logger.Infof("Cleaning up VPC: %s", vpcID)

		// Delete dependencies in order.
		t.cleanupVPCDependencies(ctx, region, vpcID)

		select {
		case <-ctx.Done():
			return cleaned, failed
		case <-time.After(vpcDependencyGrace):
		}

		if err := t.awsRunner.EC2DeleteVPC(ctx, region, vpcID); err != nil {
			t.logger.Warnf("Failed to delete VPC %s: %v", vpcID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) cleanupVPCDependencies(ctx context.Context, region, vpcID string) {
	// internet gateways
	igws, err := t.awsRunner.EC2DescribeVPCResources(ctx, region, vpcID, "internet-gateways")
	if err != nil {
		t.logger.Warnf("Failed to list internet gateways for VPC %s: %v", vpcID, err)
	}
	for _, igw := range igws {
		if err := t.awsRunner.EC2DetachInternetGateway(ctx, region, igw, vpcID); err != nil {
			t.logger.Warnf("Failed to detach internet gateway %s: %v", igw, err)
		}
		if err := t.awsRunner.EC2DeleteInternetGateway(ctx, region, igw); err != nil {
			t.logger.Warnf("Failed to delete internet gateway %s: %v", igw, err)
		}
	}

	// subnets
	subnets, err := t.awsRunner.EC2DescribeVPCResources(ctx, region, vpcID, "subnets")
	if err != nil {
		t.logger.Warnf("Failed to list subnets for VPC %s: %v", vpcID, err)
	}
	for _, subnet := range subnets {
		if err := t.awsRunner.EC2DeleteSubnet(ctx, region, subnet); err != nil {
			t.logger.Warnf("Failed to delete subnet %s: %v", subnet, err)
		}
	}

	// route tables (not the main one)
	routeTables, err := t.awsRunner.EC2DescribeVPCResources(ctx, region, vpcID, "route-tables")
	if err != nil {
		t.logger.Warnf("Failed to list route tables for VPC %s: %v", vpcID, err)
	}
	for _, rt := range routeTables {
		if err := t.awsRunner.EC2DeleteRouteTable(ctx, region, rt); err != nil {
			t.logger.Warnf("Failed to delete route table %s: %v", rt, err)
		}
	}

	// security groups (not the default one)
	securityGroups, err := t.awsRunner.EC2DescribeVPCResources(ctx, region, vpcID, "security-groups")
	if err != nil {
		t.logger.Warnf("Failed to list security groups for VPC %s: %v", vpcID, err)
	}
	for _, sg := range securityGroups {
		if err := t.awsRunner.EC2DeleteSecurityGroup(ctx, region, sg); err != nil {
			t.logger.Warnf("Failed to delete security group %s: %v", sg, err)
		}
	}
}
