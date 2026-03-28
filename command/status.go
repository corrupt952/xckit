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

		fmt.Printf("%-6s: %3d/%d translated, %d needs_review (%.1f%%)\n", lang, translated, activeKeys, len(needsReview), percentage)
	}

	return subcommands.ExitSuccess
}
