package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
	"xckit/formatter"
)

type UntranslatedCommand struct {
	XCStringsCommand
	language string
}

func (*UntranslatedCommand) Name() string {
	return "untranslated"
}

func (*UntranslatedCommand) Synopsis() string {
	return "List untranslated keys for a specific language"
}

func (*UntranslatedCommand) Usage() string {
	return "untranslated [-f file.xcstrings] [--lang <language>]: List untranslated keys with translation status\n"
}

func (c *UntranslatedCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de) - optional")
}

func (c *UntranslatedCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	var untranslatedKeys []string
	if c.language != "" {
		untranslatedKeys = xcstrings.UntranslatedKeys(c.language)
	} else {
		untranslatedKeys = xcstrings.KeysWithAnyUntranslated()
	}

	sort.Strings(untranslatedKeys)

	if len(untranslatedKeys) == 0 {
		if c.language != "" {
			fmt.Printf("All keys are translated for language '%s'\n", c.language)
		} else {
			fmt.Println("All keys are fully translated in all languages")
		}
		return subcommands.ExitSuccess
	}

	if c.language != "" {
		fmt.Printf("Untranslated keys for language '%s':\n", c.language)
	} else {
		fmt.Println("Keys with untranslated content:")
	}

	formatter.DisplayKeyDetails(xcstrings, untranslatedKeys)
	return subcommands.ExitSuccess
}
