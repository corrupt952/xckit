package command

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"

	"xckit/xcstrings"

	"github.com/google/subcommands"
)

type SetCommand struct {
	XCStringsCommand
	language string
	plural   string
	device   string
	force    bool
}

func (*SetCommand) Name() string {
	return "set"
}

func (*SetCommand) Synopsis() string {
	return "Set translation for a specific key and language"
}

func (*SetCommand) Usage() string {
	return "set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--force] <key> <value>: Set translation for a specific key and language\n"
}

func (c *SetCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de)")
	f.StringVar(&c.plural, "plural", "", "Plural category (zero, one, two, few, many, other)")
	f.StringVar(&c.device, "device", "", "Device variation (iphone, ipad, mac, appletv, applewatch, applevision, other)")
	f.BoolVar(&c.force, "force", false, "Suppress migration warning when converting plain stringUnit to variations")
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

	if c.plural != "" && !slices.Contains(xcstrings.ValidPluralCategories, c.plural) {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: invalid plural category '%s' (valid: zero, one, two, few, many, other)\n", c.plural)
		return subcommands.ExitUsageError
	}

	if c.device != "" && !slices.Contains(xcstrings.ValidDeviceCategories, c.device) {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: invalid device '%s' (valid: iphone, ipad, mac, appletv, applewatch, applevision, other)\n", c.device)
		return subcommands.ExitUsageError
	}

	key := f.Arg(0)
	value := f.Arg(1)

	xcs, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.plural != "" || c.device != "" {
		opts := xcstrings.VariationOptions{
			Plural: c.plural,
			Device: c.device,
		}
		migrated, err := xcs.SetVariationTranslation(key, c.language, value, opts)
		if err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		if migrated && !c.force {
			fmt.Fprintf(os.Stderr, "Warning: existing plain stringUnit for key '%s' in language '%s' was migrated to variations\n", key, c.language)
		}
	} else {
		if err := xcs.SetTranslation(key, c.language, value); err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
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

	fmt.Printf("Successfully set translation for key '%s' in language '%s'\n", key, c.language)
	return subcommands.ExitSuccess
}
