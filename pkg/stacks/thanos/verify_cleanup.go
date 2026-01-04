package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// VerifyAndCleanupResources  will scans for orphaned aws resources after destroy and removes them
//Resources are identified by namespace tag , name pattern following terraform conventions.
func (t *ThanosStack) VerifyAndCleanupResources(ctx context.Context, namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	region := t.awsProfile.AwsConfig.Region
	t.logger.Infof("Verifying resource cleanup for namespace: %s in region: %s", namespace, region)

	var cleaned, failed int

	// Delete in dependency order (reverse of creation)
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

	return nil
}

func (t *ThanosStack) deleteOrphanedLoadBalancers(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	// Classic ELB
	out, err := utils.ExecuteCommand(ctx, "aws", "elb", "describe-load-balancers",
		"--region", region,
		"--query", fmt.Sprintf("LoadBalancerDescriptions[?contains(LoadBalancerName, '%s')].LoadBalancerName", namespace),
		"--output", "json")
	if err == nil {
		var names []string
		if json.Unmarshal([]byte(out), &names) == nil {
			for _, name := range names {
				t.logger.Infof("Deleting ELB: %s", name)
				if _, err := utils.ExecuteCommand(ctx, "aws", "elb", "delete-load-balancer",
					"--region", region, "--load-balancer-name", name); err != nil {
					t.logger.Warnf("Failed to delete ELB %s: %v", name, err)
					failed++
				} else {
					cleaned++
				}
			}
		}
	}

	// ALB / NLB
	out, err = utils.ExecuteCommand(ctx, "aws", "elbv2", "describe-load-balancers",
		"--region", region,
		"--query", fmt.Sprintf("LoadBalancers[?contains(LoadBalancerName, '%s')].LoadBalancerArn", namespace),
		"--output", "json")
	if err == nil {
		var arns []string
		if json.Unmarshal([]byte(out), &arns) == nil {
			for _, arn := range arns {
				t.logger.Infof("Deleting ALB/NLB: %s", arn)
				if _, err := utils.ExecuteCommand(ctx, "aws", "elbv2", "delete-load-balancer",
					"--region", region, "--load-balancer-arn", arn); err != nil {
					t.logger.Warnf("Failed to delete ALB/NLB: %v", err)
					failed++
				} else {
					cleaned++
				}
			}
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedElasticIPs(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-addresses",
		"--region", region,
		"--query", fmt.Sprintf("Addresses[?Tags[?Key=='Name' && contains(Value, '%s')]].{ID:AllocationId,Assoc:AssociationId}", namespace),
		"--output", "json")
	if err != nil {
		return cleaned, failed
	}

	var eips []struct {
		ID    string `json:"ID"`
		Assoc string `json:"Assoc"`
	}
	if json.Unmarshal([]byte(out), &eips) != nil {
		return cleaned, failed
	}

	for _, eip := range eips {
		if eip.Assoc != "" {
			utils.ExecuteCommand(ctx, "aws", "ec2", "disassociate-address",
				"--region", region, "--association-id", eip.Assoc)
			time.Sleep(2 * time.Second)
		}

		t.logger.Infof("Releasing Elastic IP: %s", eip.ID)
		if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "release-address",
			"--region", region, "--allocation-id", eip.ID); err != nil {
			t.logger.Warnf("Failed to release Elastic IP %s: %v", eip.ID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedNATGateways(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-nat-gateways",
		"--region", region,
		"--filter", "Name=state,Values=available,pending",
		"--query", fmt.Sprintf("NatGateways[?Tags[?Key=='Name' && contains(Value, '%s')]].NatGatewayId", namespace),
		"--output", "json")
	if err != nil {
		return cleaned, failed
	}

	var ids []string
	if json.Unmarshal([]byte(out), &ids) != nil || len(ids) == 0 {
		return cleaned, failed
	}

	for _, id := range ids {
		t.logger.Infof("Deleting NAT Gateway: %s", id)
		if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-nat-gateway",
			"--region", region, "--nat-gateway-id", id); err != nil {
			t.logger.Warnf("Failed to delete NAT Gateway %s: %v", id, err)
			failed++
		} else {
			cleaned++
		}
	}

	if len(ids) > 0 {
		t.logger.Info("Waiting for NAT Gateway deletion...")
		time.Sleep(30 * time.Second)
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedEKS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	// this will check if cluster exists
	_, err := utils.ExecuteCommand(ctx, "aws", "eks", "describe-cluster",
		"--region", region, "--name", namespace)
	if err != nil {
		return cleaned, failed
	}

	// Deletes node groups first
	ngOut, _ := utils.ExecuteCommand(ctx, "aws", "eks", "list-nodegroups",
		"--region", region, "--cluster-name", namespace,
		"--query", "nodegroups", "--output", "json")

	var nodeGroups []string
	if json.Unmarshal([]byte(ngOut), &nodeGroups) == nil {
		for _, ng := range nodeGroups {
			t.logger.Infof("Deleting node group: %s", ng)
			utils.ExecuteCommand(ctx, "aws", "eks", "delete-nodegroup",
				"--region", region, "--cluster-name", namespace, "--nodegroup-name", ng)
		}

		if len(nodeGroups) > 0 {
			t.waitForNodeGroupsDeletion(ctx, region, namespace)
		}
	}

	t.logger.Infof("Deleting EKS cluster: %s", namespace)
	if _, err := utils.ExecuteCommand(ctx, "aws", "eks", "delete-cluster",
		"--region", region, "--name", namespace); err != nil {
		t.logger.Warnf("Failed to delete EKS cluster: %v", err)
		failed++
	} else {
		cleaned++
	}

	return cleaned, failed
}

func (t *ThanosStack) waitForNodeGroupsDeletion(ctx context.Context, region, cluster string) {
	t.logger.Info("Waiting for node groups deletion...")
	for i := 0; i < 20; i++ { // 10 minute timeout
		time.Sleep(30 * time.Second)

		out, err := utils.ExecuteCommand(ctx, "aws", "eks", "list-nodegroups",
			"--region", region, "--cluster-name", cluster,
			"--query", "nodegroups", "--output", "json")
		if err != nil {
			return
		}

		var ngs []string
		if json.Unmarshal([]byte(out), &ngs) != nil || len(ngs) == 0 {
			return
		}
	}
}

func (t *ThanosStack) deleteOrphanedEFS(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	out, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--query", "FileSystems[].{ID:FileSystemId,Name:Name}",
		"--output", "json")
	if err != nil {
		return cleaned, failed
	}

	var filesystems []struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	}
	if json.Unmarshal([]byte(out), &filesystems) != nil {
		return cleaned, failed
	}

	for _, fs := range filesystems {
		if !strings.HasPrefix(fs.Name, namespace) {
			continue
		}

		// Delete mount targets first
		mtOut, _ := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets",
			"--region", region, "--file-system-id", fs.ID,
			"--query", "MountTargets[].MountTargetId", "--output", "json")

		var mountTargets []string
		if json.Unmarshal([]byte(mtOut), &mountTargets) == nil {
			for _, mt := range mountTargets {
				utils.ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
					"--region", region, "--mount-target-id", mt)
			}
			if len(mountTargets) > 0 {
				time.Sleep(10 * time.Second)
			}
		}

		t.logger.Infof("Deleting EFS: %s (%s)", fs.Name, fs.ID)
		if _, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
			"--region", region, "--file-system-id", fs.ID); err != nil {
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

	_, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", region, "--db-instance-identifier", identifier)
	if err != nil {
		return cleaned, failed
	}

	t.logger.Infof("Deleting RDS: %s", identifier)
	if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "delete-db-instance",
		"--region", region, "--db-instance-identifier", identifier,
		"--skip-final-snapshot", "--delete-automated-backups"); err != nil {
		t.logger.Warnf("Failed to delete RDS %s: %v", identifier, err)
		failed++
	} else {
		cleaned++
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedS3(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	out, err := utils.ExecuteCommand(ctx, "aws", "s3api", "list-buckets",
		"--query", "Buckets[].Name", "--output", "json")
	if err != nil {
		return cleaned, failed
	}

	var buckets []string
	if json.Unmarshal([]byte(out), &buckets) != nil {
		return cleaned, failed
	}

	for _, bucket := range buckets {
		if !strings.Contains(bucket, namespace) {
			continue
		}

		t.logger.Infof("Deleting S3 bucket: %s", bucket)
		utils.ExecuteCommand(ctx, "aws", "s3", "rm", fmt.Sprintf("s3://%s", bucket), "--recursive")

		if _, err := utils.ExecuteCommand(ctx, "aws", "s3", "rb",
			fmt.Sprintf("s3://%s", bucket), "--force"); err != nil {
			t.logger.Warnf("Failed to delete S3 bucket %s: %v", bucket, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteOrphanedVPC(ctx context.Context, region, namespace string, cleaned, failed int) (int, int) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-vpcs",
		"--region", region,
		"--query", fmt.Sprintf("Vpcs[?Tags[?Key=='Name' && contains(Value, '%s')]].VpcId", namespace),
		"--output", "json")
	if err != nil {
		return cleaned, failed
	}

	var vpcIDs []string
	if json.Unmarshal([]byte(out), &vpcIDs) != nil || len(vpcIDs) == 0 {
		return cleaned, failed
	}

	for _, vpcID := range vpcIDs {
		t.logger.Infof("Cleaning up VPC: %s", vpcID)

		// Delete dependencies in order
		t.deleteVPCResources(ctx, region, vpcID, "internet-gateways",
			"InternetGateways[].InternetGatewayId",
			func(id string) {
				utils.ExecuteCommand(ctx, "aws", "ec2", "detach-internet-gateway",
					"--region", region, "--internet-gateway-id", id, "--vpc-id", vpcID)
				utils.ExecuteCommand(ctx, "aws", "ec2", "delete-internet-gateway",
					"--region", region, "--internet-gateway-id", id)
			})

		t.deleteVPCResources(ctx, region, vpcID, "subnets",
			"Subnets[].SubnetId",
			func(id string) {
				utils.ExecuteCommand(ctx, "aws", "ec2", "delete-subnet",
					"--region", region, "--subnet-id", id)
			})

		t.deleteVPCResources(ctx, region, vpcID, "route-tables",
			"RouteTables[?Associations[0].Main!=`true`].RouteTableId",
			func(id string) {
				utils.ExecuteCommand(ctx, "aws", "ec2", "delete-route-table",
					"--region", region, "--route-table-id", id)
			})

		t.deleteVPCResources(ctx, region, vpcID, "security-groups",
			"SecurityGroups[?GroupName!='default'].GroupId",
			func(id string) {
				utils.ExecuteCommand(ctx, "aws", "ec2", "delete-security-group",
					"--region", region, "--group-id", id)
			})

		time.Sleep(5 * time.Second)

		if _, err := utils.ExecuteCommand(ctx, "aws", "ec2", "delete-vpc",
			"--region", region, "--vpc-id", vpcID); err != nil {
			t.logger.Warnf("Failed to delete VPC %s: %v", vpcID, err)
			failed++
		} else {
			cleaned++
		}
	}

	return cleaned, failed
}

func (t *ThanosStack) deleteVPCResources(ctx context.Context, region, vpcID, resourceType, query string, deleteFn func(string)) {
	out, err := utils.ExecuteCommand(ctx, "aws", "ec2", fmt.Sprintf("describe-%s", resourceType),
		"--region", region,
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcID),
		"--query", query, "--output", "json")
	if err != nil {
		return
	}

	var ids []string
	if json.Unmarshal([]byte(out), &ids) != nil {
		return
	}

	for _, id := range ids {
		deleteFn(id)
	}
}
