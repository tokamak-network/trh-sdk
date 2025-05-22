package thanos

import (
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

type ThanosStack struct {
	network           string
	stack             string
	deployConfig      *types.Config
	enableConfimation bool
	awsConfig         *types.AWSProfile
}

func NewThanosStack(
	network string,
	stack string,
	config *types.Config,
	awsConfig *types.AWSProfile,
	enableConfirmation bool,
) *ThanosStack {
	return &ThanosStack{
		network:           network,
		stack:             stack,
		deployConfig:      config,
		enableConfimation: enableConfirmation,
		awsConfig:         awsConfig,
	}
}
