package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
)

type RemoveCommand struct {
	XCStringsCommand
	state  string
	dryRun bool
}

func (*RemoveCommand) Name() string {
	return "remove"
}

func (*RemoveCommand) Synopsis() string {
	return "Remove a key, or all keys matching an extractionState"
}

func (*RemoveCommand) Usage() string {
	return "remove [-f file.xcstrings] [--state <state>] [--dry-run] [<key>]: Remove a single key by name, or all keys matching --state. --dry-run prints what would be removed without modifying the file.\n"
}

func (c *RemoveCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.state, "state", "", "Remove every key whose extractionState matches this value (e.g. stale, manual)")
	f.BoolVar(&c.dryRun, "dry-run", false, "Print what would be removed without modifying the file")
}

func (c *RemoveCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	hasKey := f.NArg() >= 1
	if !hasKey && c.state == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: either <key> or --state is required\n")
		fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	xcs, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	var targets []string
	if hasKey {
		key := f.Arg(0)
		if _, exists := xcs.Strings[key]; !exists {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: key '%s' not found\n", key)
			return subcommands.ExitFailure
		}
		if c.state != "" && xcs.ExtractionStateOf(key) != c.state {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: key '%s' has extractionState '%s', not '%s'\n", key, xcs.ExtractionStateOf(key), c.state)
			return subcommands.ExitFailure
		}
		targets = []string{key}
	} else {
		targets = xcs.KeysByState(c.state)
		sort.Strings(targets)
	}

	if len(targets) == 0 {
		fmt.Printf("No keys found with state '%s'\n", c.state)
		return subcommands.ExitSuccess
	}

	if c.dryRun {
		fmt.Printf("Would remove %d key(s):\n", len(targets))
		for _, key := range targets {
			fmt.Printf("  %s\n", key)
		}
		return subcommands.ExitSuccess
	}

	removed := 0
	for _, key := range targets {
		if xcs.RemoveKey(key) {
			removed++
		}
	}

	filePath := c.filePath
	if filePath == "" {
		filePath = c.findXCStringsFile()
	}
	if err := xcs.SaveToFile(filePath); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error saving file: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Removed %d key(s)\n", removed)
	return subcommands.ExitSuccess
}
