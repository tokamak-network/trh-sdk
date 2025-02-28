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

	RollupConfigFlag = &cli.StringFlag{
		Name:     "rollup-config",
		Usage:    "Rollup config address",
		Required: true,
	}

	AmountFlag = &cli.FloatFlag{
		Name:     "amount",
		Usage:    "Amount of TON to stake (minimum 1000.1)",
		Required: true,
	}

	MemoFlag = &cli.StringFlag{
		Name:  "memo",
		Usage: "Memo for the registration",
		Value: "",
	}

	UseTonFlag = &cli.BoolFlag{
		Name:  "use-ton",
		Usage: "Use TON instead of WTON for staking",
		Value: false,
	}
)

var DeployContractsFlag = []cli.Flag{
	StackFlag,
	NetworkFlag,
	SaveConfigFlag,
	RollupConfigFlag,
	AmountFlag,
	MemoFlag,
	UseTonFlag,
}
