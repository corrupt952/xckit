package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"xckit/formatter"
	"xckit/xcstrings"

	"github.com/google/subcommands"
)

type UntranslatedCommand struct {
	XCStringsCommand
	language   string
	prefix     string
	detail     bool
	jsonOutput bool
	failIfAny  bool
}

func (*UntranslatedCommand) Name() string {
	return "untranslated"
}

func (*UntranslatedCommand) Synopsis() string {
	return "List untranslated keys for a specific language"
}

func (*UntranslatedCommand) Usage() string {
	return "untranslated [-f file.xcstrings] [--lang <language>] [--prefix <prefix>] [--detail] [--json] [--fail-if-any]: List untranslated keys with translation status\n"
}

func (c *UntranslatedCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de) - optional")
	f.StringVar(&c.prefix, "prefix", "", "Filter keys by prefix")
	f.BoolVar(&c.detail, "detail", false, "Show per-variation-path untranslated details")
	f.BoolVar(&c.jsonOutput, "json", false, "Output a single JSON document to stdout instead of human-readable text")
	f.BoolVar(&c.failIfAny, "fail-if-any", false, "Exit with status 1 if any untranslated string is found")
}

func (c *UntranslatedCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcs, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.jsonOutput {
		return c.executeJSON(xcs)
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
	return c.exitStatus(len(untranslatedKeys) > 0)
}

// exitStatus returns ExitFailure when hasUntranslated is true and --fail-if-any
// was requested, otherwise ExitSuccess.
func (c *UntranslatedCommand) exitStatus(hasUntranslated bool) subcommands.ExitStatus {
	if c.failIfAny && hasUntranslated {
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// collectFilteredDetails returns per-language, per-variation-path untranslated
// details, filtered by --prefix and sorted by key, then language, then path.
func (c *UntranslatedCommand) collectFilteredDetails(xcs *xcstrings.XCStrings) []xcstrings.UntranslatedDetail {
	var details []xcstrings.UntranslatedDetail
	if c.language != "" {
		details = xcs.UntranslatedDetailsForLanguage(c.language)
	} else {
		details = xcs.UntranslatedDetailsForAllLanguages()
	}

	if c.prefix != "" {
		var filtered []xcstrings.UntranslatedDetail
		for _, d := range details {
			if strings.HasPrefix(d.Key, c.prefix) {
				filtered = append(filtered, d)
			}
		}
		details = filtered
	}

	sort.Slice(details, func(i, j int) bool {
		if details[i].Key != details[j].Key {
			return details[i].Key < details[j].Key
		}
		if details[i].Language != details[j].Language {
			return details[i].Language < details[j].Language
		}
		return details[i].Path < details[j].Path
	})

	return details
}

// untranslatedJSONOutput is the top-level document printed by `untranslated --json`.
type untranslatedJSONOutput struct {
	Untranslated []untranslatedJSONItem `json:"untranslated"`
}

// untranslatedJSONItem is a single untranslated leaf, at --detail granularity.
type untranslatedJSONItem struct {
	Key      string `json:"key"`
	Language string `json:"language"`
	Path     string `json:"path"`
}

// executeJSON prints a single JSON document describing every untranslated
// leaf string unit (always at --detail granularity, regardless of --detail)
// and applies --fail-if-any to the exit status.
func (c *UntranslatedCommand) executeJSON(xcs *xcstrings.XCStrings) subcommands.ExitStatus {
	details := c.collectFilteredDetails(xcs)

	items := make([]untranslatedJSONItem, 0, len(details))
	for _, d := range details {
		items = append(items, untranslatedJSONItem{Key: d.Key, Language: d.Language, Path: d.Path})
	}

	data, err := json.MarshalIndent(untranslatedJSONOutput{Untranslated: items}, "", "  ")
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Println(string(data))
	return c.exitStatus(len(items) > 0)
}

func (c *UntranslatedCommand) executeDetail(xcs *xcstrings.XCStrings) subcommands.ExitStatus {
	details := c.collectFilteredDetails(xcs)

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

	for _, d := range details {
		fmt.Printf("%s > %s > %s\n", d.Key, d.Language, d.Path)
	}
	return c.exitStatus(len(details) > 0)
}
