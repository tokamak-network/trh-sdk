package runner

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
)

// EFSDescribeFileSystems returns file systems, optionally filtered by ID.
func (r *NativeAWSRunner) EFSDescribeFileSystems(ctx context.Context, region, fsID string) ([]EFSFileSystem, error) {
	input := &efs.DescribeFileSystemsInput{}
	if fsID != "" {
		input.FileSystemId = aws.String(fsID)
	}
	out, err := r.efsClient(region).DescribeFileSystems(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeFileSystems: %w", err)
	}
	result := make([]EFSFileSystem, 0, len(out.FileSystems))
	for _, fs := range out.FileSystems {
		name := ""
		for _, tag := range fs.Tags {
			if aws.ToString(tag.Key) == "Name" {
				name = aws.ToString(tag.Value)
			}
		}
		result = append(result, EFSFileSystem{
			FileSystemID:   aws.ToString(fs.FileSystemId),
			Name:           name,
			LifeCycleState: string(fs.LifeCycleState),
			CreationTime:   aws.ToTime(fs.CreationTime),
			ThroughputMode: string(fs.ThroughputMode),
		})
	}
	return result, nil
}

// EFSCreateMountTarget creates a mount target for the given file system.
func (r *NativeAWSRunner) EFSCreateMountTarget(ctx context.Context, region, fsID, subnetID string, securityGroups []string) error {
	input := &efs.CreateMountTargetInput{
		FileSystemId: aws.String(fsID),
		SubnetId:     aws.String(subnetID),
	}
	if len(securityGroups) > 0 {
		input.SecurityGroups = securityGroups
	}
	_, err := r.efsClient(region).CreateMountTarget(ctx, input)
	if err != nil {
		return fmt.Errorf("aws EFSCreateMountTarget: %w", err)
	}
	return nil
}

// EFSDescribeMountTargets returns mount targets for a file system.
func (r *NativeAWSRunner) EFSDescribeMountTargets(ctx context.Context, region, fsID string) ([]EFSMountTarget, error) {
	out, err := r.efsClient(region).DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(fsID),
	})
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargets: %w", err)
	}
	result := make([]EFSMountTarget, 0, len(out.MountTargets))
	for _, mt := range out.MountTargets {
		result = append(result, EFSMountTarget{
			MountTargetID:        aws.ToString(mt.MountTargetId),
			SubnetID:             aws.ToString(mt.SubnetId),
			AvailabilityZoneName: aws.ToString(mt.AvailabilityZoneName),
		})
	}
	return result, nil
}

// EFSDescribeMountTargetSecurityGroups returns security group IDs for a mount target.
func (r *NativeAWSRunner) EFSDescribeMountTargetSecurityGroups(ctx context.Context, region, mountTargetID string) ([]string, error) {
	out, err := r.efsClient(region).DescribeMountTargetSecurityGroups(ctx, &efs.DescribeMountTargetSecurityGroupsInput{
		MountTargetId: aws.String(mountTargetID),
	})
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargetSecurityGroups: %w", err)
	}
	return out.SecurityGroups, nil
}

// EFSDeleteMountTarget deletes a mount target.
func (r *NativeAWSRunner) EFSDeleteMountTarget(ctx context.Context, region, mountTargetID string) error {
	_, err := r.efsClient(region).DeleteMountTarget(ctx, &efs.DeleteMountTargetInput{
		MountTargetId: aws.String(mountTargetID),
	})
	if err != nil {
		return fmt.Errorf("aws EFSDeleteMountTarget: %w", err)
	}
	return nil
}

// EFSDeleteFileSystem deletes an EFS file system.
func (r *NativeAWSRunner) EFSDeleteFileSystem(ctx context.Context, region, fsID string) error {
	_, err := r.efsClient(region).DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(fsID),
	})
	if err != nil {
		return fmt.Errorf("aws EFSDeleteFileSystem: %w", err)
	}
	return nil
}

// EFSUpdateFileSystem updates an EFS file system (e.g. throughput mode).
func (r *NativeAWSRunner) EFSUpdateFileSystem(ctx context.Context, region, fsID, throughputMode string) error {
	_, err := r.efsClient(region).UpdateFileSystem(ctx, &efs.UpdateFileSystemInput{
		FileSystemId:   aws.String(fsID),
		ThroughputMode: efstypes.ThroughputMode(throughputMode),
	})
	if err != nil {
		return fmt.Errorf("aws EFSUpdateFileSystem: %w", err)
	}
	return nil
}

// EFSTagResource applies tags to an EFS resource.
func (r *NativeAWSRunner) EFSTagResource(ctx context.Context, region, resourceID string, tags map[string]string) error {
	efsTags := make([]efstypes.Tag, 0, len(tags))
	for k, v := range tags {
		efsTags = append(efsTags, efstypes.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	_, err := r.efsClient(region).TagResource(ctx, &efs.TagResourceInput{
		ResourceId: aws.String(resourceID),
		Tags:       efsTags,
	})
	if err != nil {
		return fmt.Errorf("aws EFSTagResource: %w", err)
	}
	return nil
}
