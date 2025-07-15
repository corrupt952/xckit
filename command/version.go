package command

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

var (
	Version string
)

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
	if Version == "" {
		Version = "dev"
	}
	fmt.Println(Version)
	return subcommands.ExitSuccess
}
