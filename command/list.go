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
}

func (*ListCommand) Name() string {
	return "list"
}

func (*ListCommand) Synopsis() string {
	return "List all keys with translation status"
}

func (*ListCommand) Usage() string {
	return "list [-f file.xcstrings]: List all keys with translation status\n"
}

func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
}

func (c *ListCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	keysToShow := xcstrings.Keys()
	sort.Strings(keysToShow)

	if len(keysToShow) == 0 {
		fmt.Println("No keys found")
		return subcommands.ExitSuccess
	}

	fmt.Println("All keys with translation status:")
	formatter.DisplayKeyDetails(xcstrings, keysToShow)
	return subcommands.ExitSuccess
}
