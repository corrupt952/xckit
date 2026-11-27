package command

import (
	"context"
	"flag"
	"fmt"
	"runtime/debug"

	"github.com/google/subcommands"
)

var (
	Version string
)

func version() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

type VersionCommand struct{}

func (*VersionCommand) Name() string {
	return "version"
}

func (*VersionCommand) Synopsis() string {
	return "Print xckit version"
}

func (*VersionCommand) Usage() string {
	return "version: xckit version\n"
}

func (*VersionCommand) SetFlags(f *flag.FlagSet) {
}

func (*VersionCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fmt.Println(version())
	return subcommands.ExitSuccess
}
