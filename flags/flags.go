package flags

import "github.com/urfave/cli/v3"

const envPrefix = "TRH_SDK"

func PrefixEnvVars(prefix, name string) []string {
	return []string{prefix + "_" + name}
}

var (
	StackFlag = &cli.StringFlag{
		Name:     "stack",
		Usage:    "Select stack(thanos)",
		Value:    "thanos",
		Sources:  cli.EnvVars(PrefixEnvVars(envPrefix, "STACK")...),
		Required: true,
	}

	NetworkFlag = &cli.StringFlag{
		Name:    "network",
		Usage:   "Select Network Environment [testnet, mainnet]",
		Value:   "testnet",
		Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "NETWORK")...),
	}

	NoCandidateFlag = &cli.BoolFlag{
		Name:    "no-candidate",
		Usage:   "Skip candidate registration after contract deployment",
		Value:   false,
		Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "NO_CANDIDATE")...),
	}
)

var DeployContractsFlag = []cli.Flag{
	StackFlag,
	NetworkFlag,
	NoCandidateFlag,
}
