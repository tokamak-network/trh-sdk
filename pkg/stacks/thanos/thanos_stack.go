package thanos

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

type ThanosStack struct {
	network      string
	stack        string
	deployConfig *types.Config
	s3Client     *s3.Client
}

func NewThanosStack(
	network string,
	stack string,
	config *types.Config,
) *ThanosStack {
	return &ThanosStack{
		network:      network,
		stack:        stack,
		deployConfig: config,
	}
}
