package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// GetCallerIdentityAccount returns the AWS account ID of the current caller.
func (r *NativeAWSRunner) GetCallerIdentityAccount(ctx context.Context) (string, error) {
	out, err := r.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("aws GetCallerIdentityAccount: %w", err)
	}
	return aws.ToString(out.Account), nil
}

// IAMRoleExists checks whether the named IAM role exists.
func (r *NativeAWSRunner) IAMRoleExists(ctx context.Context, roleName string) (bool, error) {
	_, err := r.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		var nse *iamtypes.NoSuchEntityException
		if errors.As(err, &nse) {
			return false, nil
		}
		return false, fmt.Errorf("aws IAMRoleExists: %w", err)
	}
	return true, nil
}
