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
	language string
}

func (*ListCommand) Name() string {
	return "list"
}

func (*ListCommand) Synopsis() string {
	return "List all keys with translation status"
}

func (*ListCommand) Usage() string {
	return "list [-f file.xcstrings] [--lang <language>]: List all keys with translation status\n"
}

func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de) - optional")
}

func (c *ListCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	var keysToShow []string
	if c.language != "" {
		keysToShow = xcstrings.TranslatedKeys(c.language)
	} else {
		keysToShow = xcstrings.Keys()
	}

	sort.Strings(keysToShow)

	if len(keysToShow) == 0 {
		if c.language != "" {
			fmt.Printf("No keys are translated for language '%s'\n", c.language)
		} else {
			fmt.Println("No keys found")
		}
		return subcommands.ExitSuccess
	}

	if c.language != "" {
		fmt.Printf("Keys translated in language '%s':\n", c.language)
	} else {
		fmt.Println("All keys with translation status:")
	}

	formatter.DisplayKeyDetails(xcstrings, keysToShow)
	return subcommands.ExitSuccess
}
