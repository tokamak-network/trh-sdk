package runner

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3ListBuckets returns all S3 bucket names.
func (r *NativeAWSRunner) S3ListBuckets(ctx context.Context) ([]string, error) {
	out, err := r.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("aws S3ListBuckets: %w", err)
	}
	names := make([]string, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		names = append(names, aws.ToString(b.Name))
	}
	return names, nil
}

// S3EmptyBucket removes all objects from an S3 bucket.
func (r *NativeAWSRunner) S3EmptyBucket(ctx context.Context, bucket string) error {
	paginator := s3.NewListObjectsV2Paginator(r.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("aws S3EmptyBucket list: %w", err)
		}
		if len(page.Contents) == 0 {
			continue
		}
		objs := make([]s3types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objs = append(objs, s3types.ObjectIdentifier{Key: obj.Key})
		}
		_, err = r.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3types.Delete{Objects: objs},
		})
		if err != nil {
			return fmt.Errorf("aws S3EmptyBucket delete batch: %w", err)
		}
	}
	return nil
}

// S3DeleteBucket deletes an S3 bucket.
func (r *NativeAWSRunner) S3DeleteBucket(ctx context.Context, bucket string) error {
	_, err := r.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("aws S3DeleteBucket: %w", err)
	}
	return nil
}

// LogsDescribeLogGroups checks if a log group exists.
func (r *NativeAWSRunner) LogsDescribeLogGroups(ctx context.Context, region, logGroupNamePrefix string) (bool, error) {
	out, err := r.logsClient(region).DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(logGroupNamePrefix),
	})
	if err != nil {
		return false, fmt.Errorf("aws LogsDescribeLogGroups: %w", err)
	}
	return len(out.LogGroups) > 0, nil
}

// LogsPutRetentionPolicy sets retention policy on a log group.
func (r *NativeAWSRunner) LogsPutRetentionPolicy(ctx context.Context, region, logGroupName string, retentionDays int) error {
	_, err := r.logsClient(region).PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(logGroupName),
		RetentionInDays: aws.Int32(int32(retentionDays)),
	})
	if err != nil {
		return fmt.Errorf("aws LogsPutRetentionPolicy: %w", err)
	}
	return nil
}
