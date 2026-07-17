package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"sort"

	xcstringspkg "xckit/xcstrings"

	"github.com/google/subcommands"
)

type StatusCommand struct {
	XCStringsCommand
	jsonOutput bool
}

func (*StatusCommand) Name() string {
	return "status"
}

func (*StatusCommand) Synopsis() string {
	return "Show translation progress summary"
}

func (*StatusCommand) Usage() string {
	return "status [-f file.xcstrings] [--json]: Show translation progress summary\n"
}

func (c *StatusCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.BoolVar(&c.jsonOutput, "json", false, "Output a single JSON document to stdout instead of human-readable text")
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

	langStats := make([]statusLanguageStats, 0, len(languages))
	for _, lang := range languages {
		langStats = append(langStats, computeStatusLanguageStats(xcstrings, lang, activeKeys))
	}

	if c.jsonOutput {
		return c.printJSON(xcstrings, totalKeys, len(staleKeys), activeKeys, langStats)
	}

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

	for _, s := range langStats {
		fmt.Printf("%-6s: Keys %3d/%d (%.1f%%), Strings %3d/%d (%.1f%%), %d needs_review\n",
			s.Language, s.TranslatedKeys, s.TotalKeys, s.KeysPercentage,
			s.TranslatedUnits, s.TotalUnits, s.UnitsPercentage, s.NeedsReviewCount)
	}

	return subcommands.ExitSuccess
}

// statusLanguageStats holds the translation progress figures for a single language.
type statusLanguageStats struct {
	Language         string
	TranslatedKeys   int
	TotalKeys        int
	KeysPercentage   float64
	TranslatedUnits  int
	TotalUnits       int
	UnitsPercentage  float64
	NeedsReviewCount int
}

// computeStatusLanguageStats computes key-level and string-unit-level progress
// for a single language, matching the figures shown by the human-readable output.
func computeStatusLanguageStats(xcs *xcstringspkg.XCStrings, lang string, activeKeys int) statusLanguageStats {
	untranslated := xcs.UntranslatedKeys(lang)
	needsReview := xcs.NeedsReviewKeys(lang)
	translated := activeKeys - len(untranslated)
	percentage := float64(0)
	if activeKeys > 0 {
		percentage = float64(translated) / float64(activeKeys) * 100
	}

	totalUnits := 0
	translatedUnits := 0
	for _, key := range xcs.ActiveKeys() {
		def := xcs.Strings[key]
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

	return statusLanguageStats{
		Language:         lang,
		TranslatedKeys:   translated,
		TotalKeys:        activeKeys,
		KeysPercentage:   roundTo1Decimal(percentage),
		TranslatedUnits:  translatedUnits,
		TotalUnits:       totalUnits,
		UnitsPercentage:  roundTo1Decimal(unitsPercentage),
		NeedsReviewCount: len(needsReview),
	}
}

// roundTo1Decimal rounds a float64 to a single decimal place.
func roundTo1Decimal(v float64) float64 {
	return math.Round(v*10) / 10
}

// statusJSONOutput is the top-level document printed by `status --json`.
type statusJSONOutput struct {
	SourceLanguage string                `json:"sourceLanguage"`
	TotalKeys      int                   `json:"totalKeys"`
	StaleKeys      int                   `json:"staleKeys"`
	ActiveKeys     int                   `json:"activeKeys"`
	Languages      []statusJSONLangEntry `json:"languages"`
}

// statusJSONLangEntry is the per-language progress breakdown in `status --json`.
type statusJSONLangEntry struct {
	Language    string             `json:"language"`
	Keys        statusJSONProgress `json:"keys"`
	Strings     statusJSONProgress `json:"strings"`
	NeedsReview int                `json:"needsReview"`
}

// statusJSONProgress is a translated/total/percentage triple.
type statusJSONProgress struct {
	Translated int     `json:"translated"`
	Total      int     `json:"total"`
	Percentage float64 `json:"percentage"`
}

// printJSON marshals the status summary to a single JSON document and writes it to stdout.
func (c *StatusCommand) printJSON(xcs *xcstringspkg.XCStrings, totalKeys, staleKeys, activeKeys int, langStats []statusLanguageStats) subcommands.ExitStatus {
	out := statusJSONOutput{
		SourceLanguage: xcs.SourceLanguage,
		TotalKeys:      totalKeys,
		StaleKeys:      staleKeys,
		ActiveKeys:     activeKeys,
		Languages:      make([]statusJSONLangEntry, 0, len(langStats)),
	}
	for _, s := range langStats {
		out.Languages = append(out.Languages, statusJSONLangEntry{
			Language: s.Language,
			Keys: statusJSONProgress{
				Translated: s.TranslatedKeys,
				Total:      s.TotalKeys,
				Percentage: s.KeysPercentage,
			},
			Strings: statusJSONProgress{
				Translated: s.TranslatedUnits,
				Total:      s.TotalUnits,
				Percentage: s.UnitsPercentage,
			},
			NeedsReview: s.NeedsReviewCount,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Println(string(data))
	return subcommands.ExitSuccess
}
