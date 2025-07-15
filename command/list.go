package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
	"xckit/formatter"
)

type ListCommand struct {
	XCStringsCommand
	prefix string
}

func (*ListCommand) Name() string {
	return "list"
}

func (*ListCommand) Synopsis() string {
	return "List all keys with translation status"
}

func (*ListCommand) Usage() string {
	return "list [-f file.xcstrings] [--prefix <prefix>]: List all keys with translation status\n"
}

func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.prefix, "prefix", "", "Filter keys by prefix")
}

func (c *ListCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	keysToShow := xcstrings.Keys()
	keysToShow = xcstrings.FilterKeysByPrefix(keysToShow, c.prefix)
	sort.Strings(keysToShow)

	if len(keysToShow) == 0 {
		if c.prefix != "" {
			fmt.Printf("No keys found with prefix '%s'\n", c.prefix)
		} else {
			fmt.Println("No keys found")
		}
		return subcommands.ExitSuccess
	}

	if c.prefix != "" {
		fmt.Printf("Keys with prefix '%s':\n", c.prefix)
	} else {
		fmt.Println("All keys with translation status:")
	}
	formatter.DisplayKeyDetails(xcstrings, keysToShow)
	return subcommands.ExitSuccess
}
