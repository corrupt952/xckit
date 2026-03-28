package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
	"xckit/formatter"
)

type StaleCommand struct {
	XCStringsCommand
	remove bool
	dryRun bool
}

func (*StaleCommand) Name() string {
	return "stale"
}

func (*StaleCommand) Synopsis() string {
	return "List or remove stale keys"
}

func (*StaleCommand) Usage() string {
	return "stale [-f file.xcstrings] [--remove] [--dry-run]: List or remove stale keys\n"
}

func (c *StaleCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.BoolVar(&c.remove, "remove", false, "Remove stale keys from the file")
	f.BoolVar(&c.dryRun, "dry-run", false, "Show what would be removed without modifying the file (use with --remove)")
}

func (c *StaleCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	staleKeys := xcstrings.StaleKeys()
	sort.Strings(staleKeys)

	if len(staleKeys) == 0 {
		fmt.Println("No stale keys found")
		return subcommands.ExitSuccess
	}

	if c.remove {
		if c.dryRun {
			fmt.Printf("Would remove %d stale key(s):\n", len(staleKeys))
			for _, key := range staleKeys {
				fmt.Printf("  %s\n", key)
			}
			return subcommands.ExitSuccess
		}

		count := xcstrings.RemoveStaleKeys()
		filePath := c.filePath
		if filePath == "" {
			filePath = c.findXCStringsFile()
		}
		if err := xcstrings.SaveToFile(filePath); err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		fmt.Printf("Removed %d stale key(s)\n", count)
		return subcommands.ExitSuccess
	}

	fmt.Printf("Stale keys (%d):\n", len(staleKeys))
	formatter.DisplayKeyDetails(xcstrings, staleKeys)
	return subcommands.ExitSuccess
}
