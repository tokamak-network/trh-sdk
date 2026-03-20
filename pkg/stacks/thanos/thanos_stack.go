package thanos

import (
	"context"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
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
	kubeconfigPath    string // LocalTestnet only: path to kind kubeconfig
}

// isLocal returns true when the stack targets a local kind cluster.
func (t *ThanosStack) isLocal() bool {
	return t.network == constants.LocalTestnet
}

func NewThanosStack(
	ctx context.Context,
	l *zap.SugaredLogger,
	network string,
	usePromptInput bool,
	deploymentPath string,
	awsConfig *types.AWSConfig,
) (*ThanosStack, error) {
	l.Infof("Deployment Path: %s", deploymentPath)
	l.Infof("Network: %s", network)

	// get the config file
	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		l.Error("Error reading settings.json", "err", err)
		return nil, err
	}

	// Login AWS

	var awsProfile *types.AWSProfile

	if awsConfig != nil {
		if _, err := utils.SetAWSConfigFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS config file", "err", err)
			return nil, err
		}
		if _, err := utils.SetAWSCredentialsFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS credentials file", "err", err)
			return nil, err
		}
		if _, err := utils.SetKubeconfigFile(deploymentPath); err != nil {
			l.Error("Failed to set kubeconfig file", "err", err)
			return nil, err
		}

		awsProfile, err = aws.LoginAWS(ctx, awsConfig)
		if err != nil {
			l.Error("Failed to login aws", "err", err)
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

// NewLocalTestnetThanosStack creates a ThanosStack for LocalTestnet deployments.
// It skips all AWS/DO setup and uses the provided kubeconfig to access the kind cluster.
func NewLocalTestnetThanosStack(
	ctx context.Context,
	l *zap.SugaredLogger,
	deploymentPath string,
	kubeconfigPath string,
) (*ThanosStack, error) {
	l.Infof("Deployment Path: %s", deploymentPath)
	l.Infof("Network: %s", constants.LocalTestnet)

	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		l.Error("Error reading settings.json", "err", err)
		return nil, err
	}

	// Note: KUBECONFIG is NOT set via os.Setenv (process-global, race condition).
	// Instead, kubeconfigPath is stored and passed via --kubeconfig flag
	// to kubectl/helm commands by the caller.

	return &ThanosStack{
		network:        constants.LocalTestnet,
		usePromptInput: false,
		logger:         l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
		kubeconfigPath: kubeconfigPath,
	}, nil
}
