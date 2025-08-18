package thanos

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"go.uber.org/zap"
)

type ThanosStack struct {
	network           string
	deployConfig      *types.Config
	usePromptInput    bool
	awsProfile        *types.AWSProfile
	logger            *zap.SugaredLogger
	deploymentPath    string
	registerCandidate bool
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

		// Switch to this context
		if config != nil && config.K8s != nil {
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
		logger:         l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
	}, nil
}
