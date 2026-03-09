package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
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
	names, err := utils.ListClassicLoadBalancers(ctx, t.awsRunner, region, namespace)
	if err == nil {
		for _, name := range names {
			t.logger.Infof("Deleting ELB: %s", name)
			if err := utils.DeleteClassicLoadBalancer(ctx, t.awsRunner, region, name); err != nil {
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
	arns, err := utils.ListALBLoadBalancers(ctx, t.awsRunner, region, namespace)
	if err != nil {
		t.logger.Warnf("Failed to list ALB/NLBs: %v", err)
		return cleaned, failed
	}
	for _, arn := range arns {
		t.logger.Infof("Deleting ALB/NLB: %s", arn)
		if err := utils.DeleteALBLoadBalancer(ctx, t.awsRunner, region, arn); err != nil {
			t.logger.Warnf("Failed to delete ALB/NLB: %v", err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedElasticIPs(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	eips, err := utils.ListElasticIPs(ctx, t.awsRunner, region, namespace)
	if err != nil {
		return cleaned, failed
	}

	for _, eip := range eips {
		if eip.Assoc != "" {
			if err := utils.DisassociateAddress(ctx, t.awsRunner, region, eip.Assoc); err != nil {
				t.logger.Warnf("Failed to disassociate address %s: %v", eip.Assoc, err)
			}
			select {
			case <-ctx.Done():
				return cleaned, failed
			case <-time.After(eipDisassociationGrace):
			}
		}

		t.logger.Infof("Releasing Elastic IP: %s", eip.ID)
		if err := utils.ReleaseAddress(ctx, t.awsRunner, region, eip.ID); err != nil {
			t.logger.Warnf("Failed to release Elastic IP %s: %v", eip.ID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedNATGateways(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	ids, err := utils.ListNATGateways(ctx, t.awsRunner, region, namespace)
	if err != nil || len(ids) == 0 {
		return cleaned, failed
	}

	for _, id := range ids {
		t.logger.Infof("Deleting NAT Gateway: %s", id)
		if err := utils.DeleteNATGateway(ctx, t.awsRunner, region, id); err != nil {
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
	if !utils.EKSClusterExists(ctx, t.awsRunner, region, namespace) {
		return cleaned, failed
	}

	// Delete node groups first.
	nodeGroups, err := utils.ListNodeGroups(ctx, t.awsRunner, region, namespace)
	if err == nil {
		for _, ng := range nodeGroups {
			t.logger.Infof("Deleting node group: %s", ng)
			if err := utils.DeleteNodeGroup(ctx, t.awsRunner, region, namespace, ng); err != nil {
				t.logger.Warnf("Failed to delete node group %s: %v", ng, err)
				failed++
			}
		}

		if len(nodeGroups) > 0 {
			t.waitForNodeGroupsDeletion(ctx, region, namespace)
		}
	}

	t.logger.Infof("Deleting EKS cluster: %s", namespace)
	if err := utils.DeleteEKSCluster(ctx, t.awsRunner, region, namespace); err != nil {
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
		ngs, err := utils.ListNodeGroups(ctx, t.awsRunner, region, cluster)
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
	filesystems, err := utils.ListEFSFileSystems(ctx, t.awsRunner, region)
	if err != nil {
		return cleaned, failed
	}

	for _, fs := range filesystems {
		if !strings.HasPrefix(fs.Name, namespace) {
			continue
		}

		// Delete mount targets first.
		mountTargets, err := utils.ListMountTargets(ctx, t.awsRunner, region, fs.ID)
		if err == nil {
			for _, mt := range mountTargets {
				if err := utils.DeleteMountTarget(ctx, t.awsRunner, region, mt); err != nil {
					t.logger.Warnf("Failed to delete mount target %s: %v", mt, err)
				}
			}
			if len(mountTargets) > 0 {
				select {
				case <-ctx.Done():
					return cleaned, failed
				case <-time.After(mountTargetDeletionGrace):
				}
			}
		}

		t.logger.Infof("Deleting EFS: %s (%s)", fs.Name, fs.ID)
		if err := utils.DeleteEFSFileSystem(ctx, t.awsRunner, region, fs.ID); err != nil {
			t.logger.Warnf("Failed to delete EFS %s: %v", fs.ID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedRDS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	identifier := fmt.Sprintf("%s-rds", namespace)

	if !utils.RDSInstanceExists(ctx, t.awsRunner, region, identifier) {
		return cleaned, failed
	}

	t.logger.Infof("Deleting RDS: %s", identifier)
	if err := utils.DeleteRDSInstance(ctx, t.awsRunner, region, identifier); err != nil {
		t.logger.Warnf("Failed to delete RDS %s: %v", identifier, err)
		failed++
	} else {
		cleaned++
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedS3(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	buckets, err := utils.ListS3Buckets(ctx, t.awsRunner)
	if err != nil {
		return cleaned, failed
	}

	for _, bucket := range buckets {
		if !strings.HasPrefix(bucket, namespace) {
			continue
		}

		t.logger.Infof("Deleting S3 bucket: %s", bucket)
		if err := utils.EmptyS3Bucket(ctx, t.awsRunner, bucket); err != nil {
			t.logger.Warnf("Failed to empty S3 bucket %s: %v", bucket, err)
		}

		if err := utils.DeleteS3Bucket(ctx, t.awsRunner, bucket); err != nil {
			t.logger.Warnf("Failed to delete S3 bucket %s: %v", bucket, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedVPC(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	vpcIDs, err := utils.ListVPCs(ctx, t.awsRunner, region, namespace)
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

		if err := utils.DeleteVPC(ctx, t.awsRunner, region, vpcID); err != nil {
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
	igws, err := utils.ListVPCResources(ctx, t.awsRunner, region, vpcID, "internet-gateways")
	if err != nil {
		t.logger.Warnf("Failed to list internet gateways for VPC %s: %v", vpcID, err)
	}
	for _, igw := range igws {
		if err := utils.DetachInternetGateway(ctx, t.awsRunner, region, igw, vpcID); err != nil {
			t.logger.Warnf("Failed to detach internet gateway %s: %v", igw, err)
		}
		if err := utils.DeleteInternetGateway(ctx, t.awsRunner, region, igw); err != nil {
			t.logger.Warnf("Failed to delete internet gateway %s: %v", igw, err)
		}
	}

	// subnets
	subnets, err := utils.ListVPCResources(ctx, t.awsRunner, region, vpcID, "subnets")
	if err != nil {
		t.logger.Warnf("Failed to list subnets for VPC %s: %v", vpcID, err)
	}
	for _, subnet := range subnets {
		if err := utils.DeleteSubnet(ctx, t.awsRunner, region, subnet); err != nil {
			t.logger.Warnf("Failed to delete subnet %s: %v", subnet, err)
		}
	}

	// route tables (not the main one)
	routeTables, err := utils.ListVPCResources(ctx, t.awsRunner, region, vpcID, "route-tables")
	if err != nil {
		t.logger.Warnf("Failed to list route tables for VPC %s: %v", vpcID, err)
	}
	for _, rt := range routeTables {
		if err := utils.DeleteRouteTable(ctx, t.awsRunner, region, rt); err != nil {
			t.logger.Warnf("Failed to delete route table %s: %v", rt, err)
		}
	}

	// security groups (not the default one)
	securityGroups, err := utils.ListVPCResources(ctx, t.awsRunner, region, vpcID, "security-groups")
	if err != nil {
		t.logger.Warnf("Failed to list security groups for VPC %s: %v", vpcID, err)
	}
	for _, sg := range securityGroups {
		if err := utils.DeleteSecurityGroup(ctx, t.awsRunner, region, sg); err != nil {
			t.logger.Warnf("Failed to delete security group %s: %v", sg, err)
		}
	}
}
