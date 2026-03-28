package command

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/google/subcommands"
	"xckit/formatter"
	"xckit/xcstrings"
)

type UntranslatedCommand struct {
	XCStringsCommand
	language string
	prefix   string
	detail   bool
}

func (*UntranslatedCommand) Name() string {
	return "untranslated"
}

func (*UntranslatedCommand) Synopsis() string {
	return "List untranslated keys for a specific language"
}

func (*UntranslatedCommand) Usage() string {
	return "untranslated [-f file.xcstrings] [--lang <language>] [--prefix <prefix>]: List untranslated keys with translation status\n"
}

func (c *UntranslatedCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de) - optional")
	f.StringVar(&c.prefix, "prefix", "", "Filter keys by prefix")
	f.BoolVar(&c.detail, "detail", false, "Show per-variation-path untranslated details")
}

func (c *UntranslatedCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcs, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.detail {
		return c.executeDetail(xcs)
	}

	var untranslatedKeys []string
	if c.language != "" {
		untranslatedKeys = xcs.UntranslatedKeys(c.language)
	} else {
		untranslatedKeys = xcs.KeysWithAnyUntranslated()
	}

	untranslatedKeys = xcs.FilterKeysByPrefix(untranslatedKeys, c.prefix)
	sort.Strings(untranslatedKeys)

	if len(untranslatedKeys) == 0 {
		if c.prefix != "" && c.language != "" {
			fmt.Printf("No untranslated keys found with prefix '%s' for language '%s'\n", c.prefix, c.language)
		} else if c.prefix != "" {
			fmt.Printf("No untranslated keys found with prefix '%s'\n", c.prefix)
		} else if c.language != "" {
			fmt.Printf("All keys are translated for language '%s'\n", c.language)
		} else {
			fmt.Println("All keys are fully translated in all languages")
		}
		return subcommands.ExitSuccess
	}

	if c.prefix != "" && c.language != "" {
		fmt.Printf("Untranslated keys with prefix '%s' for language '%s':\n", c.prefix, c.language)
	} else if c.prefix != "" {
		fmt.Printf("Untranslated keys with prefix '%s':\n", c.prefix)
	} else if c.language != "" {
		fmt.Printf("Untranslated keys for language '%s':\n", c.language)
	} else {
		fmt.Println("Keys with untranslated content:")
	}

	formatter.DisplayKeyDetails(xcs, untranslatedKeys)
	return subcommands.ExitSuccess
}

func (c *UntranslatedCommand) executeDetail(xcs *xcstrings.XCStrings) subcommands.ExitStatus {
	var details []xcstrings.UntranslatedDetail
	if c.language != "" {
		details = xcs.UntranslatedDetailsForLanguage(c.language)
	} else {
		details = xcs.UntranslatedDetailsForAllLanguages()
	}

	// Filter by prefix
	if c.prefix != "" {
		var filtered []xcstrings.UntranslatedDetail
		for _, d := range details {
			if strings.HasPrefix(d.Key, c.prefix) {
				filtered = append(filtered, d)
			}
		}
		details = filtered
	}

	if len(details) == 0 {
		if c.prefix != "" && c.language != "" {
			fmt.Printf("No untranslated keys found with prefix '%s' for language '%s'\n", c.prefix, c.language)
		} else if c.prefix != "" {
			fmt.Printf("No untranslated keys found with prefix '%s'\n", c.prefix)
		} else if c.language != "" {
			fmt.Printf("All keys are translated for language '%s'\n", c.language)
		} else {
			fmt.Println("All keys are fully translated in all languages")
		}
		return subcommands.ExitSuccess
	}

	// Sort by key, then language, then path
	sort.Slice(details, func(i, j int) bool {
		if details[i].Key != details[j].Key {
			return details[i].Key < details[j].Key
		}
		if details[i].Language != details[j].Language {
			return details[i].Language < details[j].Language
		}
		return details[i].Path < details[j].Path
	})

	for _, d := range details {
		fmt.Printf("%s > %s > %s\n", d.Key, d.Language, d.Path)
	}
	return subcommands.ExitSuccess
}
