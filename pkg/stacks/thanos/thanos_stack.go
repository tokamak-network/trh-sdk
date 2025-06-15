package thanos

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

type ThanosStack struct {
	network        string
	deployConfig   *types.Config
	usePromptInput bool
	awsProfile     *types.AWSProfile
	l              *zap.SugaredLogger
	deploymentPath string
}

func NewThanosStack(
	ctx context.Context,
	l *zap.SugaredLogger,
	network string,
	usePromptInput bool,
	deploymentPath string,
	awsConfig *types.AWSConfig,
) (*ThanosStack, error) {
	fmt.Println("Deployment Path:", deploymentPath)
	fmt.Println("Network:", network)

	// get the config file
	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		fmt.Println("Error reading settings.json")
		return nil, err
	}

	// Login AWS

	var awsProfile *types.AWSProfile

	if awsConfig != nil {
		awsProfile, err = aws.LoginAWS(ctx, awsConfig)
		if err != nil {
			fmt.Println("Failed to login aws", "err", err)
			return nil, err
		}

		fmt.Println("AWS Profile:", awsConfig.SwitchAWSContext)

		if awsConfig.SwitchAWSContext {
			if config == nil {
				return nil, fmt.Errorf("config is nil")
			}

			if config.K8s == nil {
				return nil, fmt.Errorf("k8s is nil")
			}

			// Switch to this context
			err = utils.SwitchKubernetesContext(ctx, config.K8s.Namespace, awsConfig.Region)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ThanosStack{
		network:        network,
		usePromptInput: usePromptInput,
		awsProfile:     awsProfile,
		l:              l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
	}, nil
}
