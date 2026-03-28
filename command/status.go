package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
)

type StatusCommand struct {
	XCStringsCommand
}

func (*StatusCommand) Name() string {
	return "status"
}

func (*StatusCommand) Synopsis() string {
	return "Show translation progress summary"
}

func (*StatusCommand) Usage() string {
	return "status [-f file.xcstrings]: Show translation progress summary\n"
}

func (c *StatusCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
}

func (c *StatusCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	totalKeys := len(xcstrings.Strings)
	staleKeys := xcstrings.StaleKeys()
	activeKeys := totalKeys - len(staleKeys)
	languages := xcstrings.Languages()
	sort.Strings(languages)

	fmt.Printf("Translation Status\n")
	fmt.Printf("==================\n")
	fmt.Printf("Source Language: %s\n", xcstrings.SourceLanguage)
	fmt.Printf("Total Keys: %d\n", totalKeys)
	if len(staleKeys) > 0 {
		fmt.Printf("Stale Keys: %d\n", len(staleKeys))
		fmt.Printf("Active Keys: %d\n", activeKeys)
	}
	fmt.Printf("Languages: %s\n\n", languages)

	fmt.Printf("Progress by Language:\n")
	fmt.Printf("--------------------\n")

	for _, lang := range languages {
		untranslated := xcstrings.UntranslatedKeys(lang)
		needsReview := xcstrings.NeedsReviewKeys(lang)
		translated := activeKeys - len(untranslated)
		percentage := float64(0)
		if activeKeys > 0 {
			percentage = float64(translated) / float64(activeKeys) * 100
		}

		totalUnits := 0
		translatedUnits := 0
		for _, key := range xcstrings.ActiveKeys() {
			def := xcstrings.Strings[key]
			if def.ShouldTranslate != nil && *def.ShouldTranslate == false {
				continue
			}
			loc, exists := def.Localizations[lang]
			if !exists {
				totalUnits++
				continue
			}
			units := loc.AllStringUnits()
			if len(units) == 0 {
				totalUnits++
				continue
			}
			totalUnits += len(units)
			for _, u := range units {
				if u.State == "translated" {
					translatedUnits++
				}
			}
		}
		unitsPercentage := float64(0)
		if totalUnits > 0 {
			unitsPercentage = float64(translatedUnits) / float64(totalUnits) * 100
		}

		fmt.Printf("%-6s: Keys %3d/%d (%.1f%%), Strings %3d/%d (%.1f%%), %d needs_review\n",
			lang, translated, activeKeys, percentage,
			translatedUnits, totalUnits, unitsPercentage, len(needsReview))
	}

	return subcommands.ExitSuccess
}
