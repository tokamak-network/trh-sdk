package runner

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2DescribeSubnetState returns the state of a subnet.
func (r *NativeAWSRunner) EC2DescribeSubnetState(ctx context.Context, region, subnetID string) (string, error) {
	out, err := r.ec2Client(region).DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		return "", fmt.Errorf("aws EC2DescribeSubnetState: %w", err)
	}
	if len(out.Subnets) == 0 {
		return "", fmt.Errorf("aws EC2DescribeSubnetState: subnet %s not found", subnetID)
	}
	return string(out.Subnets[0].State), nil
}

// EC2DescribeAddresses returns elastic IPs matching a namespace tag filter.
func (r *NativeAWSRunner) EC2DescribeAddresses(ctx context.Context, region, namespaceFilter string) ([]ElasticIPInfo, error) {
	out, err := r.ec2Client(region).DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Namespace"),
				Values: []string{namespaceFilter},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeAddresses: %w", err)
	}
	addrs := make([]ElasticIPInfo, 0, len(out.Addresses))
	for _, a := range out.Addresses {
		addrs = append(addrs, ElasticIPInfo{
			AllocationID:  aws.ToString(a.AllocationId),
			AssociationID: aws.ToString(a.AssociationId),
		})
	}
	return addrs, nil
}

// EC2DisassociateAddress disassociates an elastic IP.
func (r *NativeAWSRunner) EC2DisassociateAddress(ctx context.Context, region, associationID string) error {
	_, err := r.ec2Client(region).DisassociateAddress(ctx, &ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DisassociateAddress: %w", err)
	}
	return nil
}

// EC2ReleaseAddress releases an elastic IP.
func (r *NativeAWSRunner) EC2ReleaseAddress(ctx context.Context, region, allocationID string) error {
	_, err := r.ec2Client(region).ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: aws.String(allocationID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2ReleaseAddress: %w", err)
	}
	return nil
}

// EC2DescribeNATGateways returns NAT gateway IDs matching a namespace tag filter.
func (r *NativeAWSRunner) EC2DescribeNATGateways(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := r.ec2Client(region).DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []ec2types.Filter{
			{
				Name:   aws.String("tag:Namespace"),
				Values: []string{namespaceFilter},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeNATGateways: %w", err)
	}
	ids := make([]string, 0, len(out.NatGateways))
	for _, ng := range out.NatGateways {
		ids = append(ids, aws.ToString(ng.NatGatewayId))
	}
	return ids, nil
}

// EC2DeleteNATGateway deletes a NAT gateway.
func (r *NativeAWSRunner) EC2DeleteNATGateway(ctx context.Context, region, natGatewayID string) error {
	_, err := r.ec2Client(region).DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(natGatewayID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteNATGateway: %w", err)
	}
	return nil
}

// EC2DescribeVPCs returns VPC IDs matching a namespace tag filter.
func (r *NativeAWSRunner) EC2DescribeVPCs(ctx context.Context, region, namespaceFilter string) ([]string, error) {
	out, err := r.ec2Client(region).DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Namespace"),
				Values: []string{namespaceFilter},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("aws EC2DescribeVPCs: %w", err)
	}
	ids := make([]string, 0, len(out.Vpcs))
	for _, vpc := range out.Vpcs {
		ids = append(ids, aws.ToString(vpc.VpcId))
	}
	return ids, nil
}

// EC2DescribeVPCResources returns resource IDs for a VPC (subnets, route-tables, etc).
func (r *NativeAWSRunner) EC2DescribeVPCResources(ctx context.Context, region, vpcID, resourceType string) ([]string, error) {
	vpcFilter := ec2types.Filter{
		Name:   aws.String("vpc-id"),
		Values: []string{vpcID},
	}
	switch resourceType {
	case "subnets":
		out, err := r.ec2Client(region).DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			Filters: []ec2types.Filter{vpcFilter},
		})
		if err != nil {
			return nil, fmt.Errorf("aws EC2DescribeVPCResources: %w", err)
		}
		ids := make([]string, 0, len(out.Subnets))
		for _, s := range out.Subnets {
			ids = append(ids, aws.ToString(s.SubnetId))
		}
		return ids, nil
	case "internet-gateways":
		out, err := r.ec2Client(region).DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("attachment.vpc-id"),
					Values: []string{vpcID},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("aws EC2DescribeVPCResources: %w", err)
		}
		ids := make([]string, 0, len(out.InternetGateways))
		for _, igw := range out.InternetGateways {
			ids = append(ids, aws.ToString(igw.InternetGatewayId))
		}
		return ids, nil
	case "route-tables":
		out, err := r.ec2Client(region).DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []ec2types.Filter{vpcFilter},
		})
		if err != nil {
			return nil, fmt.Errorf("aws EC2DescribeVPCResources: %w", err)
		}
		ids := make([]string, 0, len(out.RouteTables))
		for _, rt := range out.RouteTables {
			ids = append(ids, aws.ToString(rt.RouteTableId))
		}
		return ids, nil
	case "security-groups":
		out, err := r.ec2Client(region).DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []ec2types.Filter{vpcFilter},
		})
		if err != nil {
			return nil, fmt.Errorf("aws EC2DescribeVPCResources: %w", err)
		}
		ids := make([]string, 0, len(out.SecurityGroups))
		for _, sg := range out.SecurityGroups {
			ids = append(ids, aws.ToString(sg.GroupId))
		}
		return ids, nil
	default:
		return nil, fmt.Errorf("aws EC2DescribeVPCResources: unsupported resource type: %s", resourceType)
	}
}

// EC2DetachInternetGateway detaches an internet gateway from a VPC.
func (r *NativeAWSRunner) EC2DetachInternetGateway(ctx context.Context, region, igwID, vpcID string) error {
	_, err := r.ec2Client(region).DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DetachInternetGateway: %w", err)
	}
	return nil
}

// EC2DeleteInternetGateway deletes an internet gateway.
func (r *NativeAWSRunner) EC2DeleteInternetGateway(ctx context.Context, region, igwID string) error {
	_, err := r.ec2Client(region).DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteInternetGateway: %w", err)
	}
	return nil
}

// EC2DeleteSubnet deletes a subnet.
func (r *NativeAWSRunner) EC2DeleteSubnet(ctx context.Context, region, subnetID string) error {
	_, err := r.ec2Client(region).DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: aws.String(subnetID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteSubnet: %w", err)
	}
	return nil
}

// EC2DeleteRouteTable deletes a route table.
func (r *NativeAWSRunner) EC2DeleteRouteTable(ctx context.Context, region, routeTableID string) error {
	_, err := r.ec2Client(region).DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: aws.String(routeTableID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteRouteTable: %w", err)
	}
	return nil
}

// EC2DeleteSecurityGroup deletes a security group.
func (r *NativeAWSRunner) EC2DeleteSecurityGroup(ctx context.Context, region, groupID string) error {
	_, err := r.ec2Client(region).DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(groupID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteSecurityGroup: %w", err)
	}
	return nil
}

// EC2DeleteVPC deletes a VPC.
func (r *NativeAWSRunner) EC2DeleteVPC(ctx context.Context, region, vpcID string) error {
	_, err := r.ec2Client(region).DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("aws EC2DeleteVPC: %w", err)
	}
	return nil
}
