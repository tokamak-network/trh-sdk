package thanos

import (
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

type ThanosStack struct {
	network           string
	stack             string
	deployConfig      *types.Config
	enableConfimation bool
	awsConfig         *types.AWSProfile
	l                 *zap.SugaredLogger
	deploymentPath    string
}

func NewThanosStack(
	l *zap.SugaredLogger,
	network string,
	stack string,
	config *types.Config,
	awsConfig *types.AWSProfile,
	enableConfirmation bool,
	deploymentPath string,
) *ThanosStack {
	return &ThanosStack{
		network:           network,
		stack:             stack,
		deployConfig:      config,
		enableConfimation: enableConfirmation,
		awsConfig:         awsConfig,
		l:                 l,
		deploymentPath:    deploymentPath,
	}
}
