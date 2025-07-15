package command

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type SetCommand struct {
	XCStringsCommand
	language string
}

func (*SetCommand) Name() string {
	return "set"
}

func (*SetCommand) Synopsis() string {
	return "Set translation for a specific key and language"
}

func (*SetCommand) Usage() string {
	return "set [-f file.xcstrings] --lang <language> <key> <value>: Set translation for a specific key and language\n"
}

func (c *SetCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de)")
}

func (c *SetCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.language == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: --lang flag is required\n")
		fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	if f.NArg() < 2 {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: key and value arguments are required\n")
		fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	key := f.Arg(0)
	value := f.Arg(1)

	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := xcstrings.SetTranslation(key, c.language, value); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	filePath := c.filePath
	if filePath == "" {
		filePath = c.findXCStringsFile()
	}

	if err := xcstrings.SaveToFile(filePath); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error saving file: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully set translation for key '%s' in language '%s'\n", key, c.language)
	return subcommands.ExitSuccess
}
