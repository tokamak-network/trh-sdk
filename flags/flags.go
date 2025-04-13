package flags

import "github.com/urfave/cli/v3"

const envPrefix = "TRH_SDK"

func PrefixEnvVars(prefix, name string) []string {
	return []string{prefix + "_" + name}
}

var (
	StackFlag = &cli.StringFlag{
		Name:     "stack",
		Usage:    "Select stack",
		Value:    "thanos",
		Sources:  cli.EnvVars(PrefixEnvVars(envPrefix, "STACK")...),
		Required: true,
	}

	SaveConfigFlag = &cli.BoolFlag{
		Name:    "saveconfig",
		Usage:   "Save the config file",
		Value:   false,
		Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "SAVE_CONFIG")...),
	}

	NetworkFlag = &cli.StringFlag{
		Name:    "network",
		Usage:   "Select Network Environment [localhost, testnet, mainnet]",
		Value:   "localhost",
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
	SaveConfigFlag,
	NoCandidateFlag,
}

var VerifyRegisterCandidateFlag = []cli.Flag{
	StackFlag,
	NetworkFlag,
}
