package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"

	"xckit/command"
)

func main() {
	subcommands.Register(&command.UntranslatedCommand{}, "")
	subcommands.Register(&command.ListCommand{}, "")
	subcommands.Register(&command.SetCommand{}, "")
	subcommands.Register(&command.StatusCommand{}, "")
	subcommands.Register(&command.VersionCommand{}, "")
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
