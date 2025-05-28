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
	network           string
	deployConfig      *types.Config
	ignorePromptInput bool
	awsProfile        *types.AWSProfile
	l                 *zap.SugaredLogger
	deploymentPath    string
}

func NewThanosStack(
	l *zap.SugaredLogger,
	network string,
	ignorePromptInput bool,
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
		awsProfile, err = aws.LoginAWS(context.Background(), awsConfig)
		if err != nil {
			fmt.Println("Failed to login aws", "err", err)
			return nil, err
		}
	}

	if awsConfig != nil && config.K8s != nil && config.K8s.Namespace != "" {
		// Switch to this context
		err = utils.SwitchKubernetesContext(context.Background(), config.K8s.Namespace, awsConfig.Region)
		if err != nil {
			fmt.Println("Failed to switch k8s context", "err", err)
			return nil, err
		}
	}

	return &ThanosStack{
		network:           network,
		ignorePromptInput: ignorePromptInput,
		awsProfile:        awsProfile,
		l:                 l,
		deploymentPath:    deploymentPath,
		deployConfig:      config,
	}, nil
}
