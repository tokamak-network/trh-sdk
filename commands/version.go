package commands

import (
	"context"
	"runtime/debug"

	"github.com/urfave/cli/v3"
)

func ActionVersion() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		version := GetVersionInfo()
		if version == "" {
			return cli.Exit("Version information not available", 1)
		}
		println("Version:", version)
		return nil
	}
}

func GetVersionInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	version := info.Main.Version // usually "vX.Y.Z" or commit hash
	return version
}
