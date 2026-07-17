package command

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"xckit/xcstrings"

	"github.com/google/subcommands"
)

type SetCommand struct {
	XCStringsCommand
	language         string
	plural           string
	device           string
	state            string
	force            bool
	allowNewLanguage bool
}

func (*SetCommand) Name() string {
	return "set"
}

func (*SetCommand) Synopsis() string {
	return "Set translation for a specific key and language"
}

func (*SetCommand) Usage() string {
	return "set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--state <state>] [--force] <key> <value>: Set translation, creating the key if it does not yet exist\n"
}

func (c *SetCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de)")
	f.StringVar(&c.plural, "plural", "", "Plural category (zero, one, two, few, many, other)")
	f.StringVar(&c.device, "device", "", "Device variation (iphone, ipad, mac, appletv, applewatch, applevision, other)")
	f.StringVar(&c.state, "state", "", "extractionState applied when the key is newly created (e.g. manual). Ignored when the key already exists.")
	f.BoolVar(&c.force, "force", false, "Suppress migration warning when converting plain stringUnit to variations")
	f.BoolVar(&c.allowNewLanguage, "allow-new-language", false, "Allow adding a language that is not yet present in the catalog")
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

	if err := c.validateLanguage(xcs); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitUsageError
	}

	created := false
	if c.plural != "" || c.device != "" {
		opts := xcstrings.VariationOptions{
			Plural: c.plural,
			Device: c.device,
		}
		migrated, wasCreated, err := xcs.SetVariationTranslation(key, c.language, value, opts, c.state)
		if err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		created = wasCreated
		if migrated && !c.force {
			fmt.Fprintf(os.Stderr, "Warning: existing plain stringUnit for key '%s' in language '%s' was migrated to variations\n", key, c.language)
		}
	} else {
		wasCreated, err := xcs.SetTranslation(key, c.language, value, c.state)
		if err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		created = wasCreated
	}

	filePath := c.filePath
	if filePath == "" {
		filePath = c.findXCStringsFile()
	}

	if err := xcs.SaveToFile(filePath); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error saving file: %v\n", err)
		return subcommands.ExitFailure
	}

	if created {
		fmt.Printf("Successfully created key '%s' with translation for language '%s'\n", key, c.language)
	} else {
		fmt.Printf("Successfully set translation for key '%s' in language '%s'\n", key, c.language)
	}
	return subcommands.ExitSuccess
}

// validateLanguage ensures c.language refers to a language already present in
// the catalog (or the catalog's source language), unless the catalog has no
// languages yet (nothing to compare against, so the first language addition
// is never blocked) or --allow-new-language was explicitly passed.
func (c *SetCommand) validateLanguage(xcs *xcstrings.XCStrings) error {
	existing := xcs.Languages()
	if len(existing) == 0 {
		// Nothing to validate against yet; do not block initial catalog setup.
		return nil
	}

	candidates := append(existing, xcs.SourceLanguage)
	if slices.Contains(candidates, c.language) {
		return nil
	}

	if c.allowNewLanguage {
		return nil
	}

	if suggestion := caseInsensitiveLanguageMatch(c.language, candidates); suggestion != "" {
		return fmt.Errorf("unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", c.language, suggestion)
	}

	if suggestion := nearestLanguageMatch(c.language, candidates); suggestion != "" {
		return fmt.Errorf("unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", c.language, suggestion)
	}

	return fmt.Errorf("unknown language '%s' is not present in the catalog. Use --allow-new-language to add a new language", c.language)
}

// caseInsensitiveLanguageMatch returns the candidate that matches input
// case-insensitively (e.g. "JA" matching existing "ja"), or "" if none.
func caseInsensitiveLanguageMatch(input string, candidates []string) string {
	for _, candidate := range candidates {
		if candidate != input && strings.EqualFold(candidate, input) {
			return candidate
		}
	}
	return ""
}

// nearestLanguageMatch returns the candidate closest to input by simple edit
// distance, useful for catching typos like "jp" -> "ja". Returns "" if no
// candidate is close enough to be a plausible suggestion.
func nearestLanguageMatch(input string, candidates []string) string {
	lowerInput := strings.ToLower(input)
	best := ""
	bestDistance := -1
	for _, candidate := range candidates {
		lowerCandidate := strings.ToLower(candidate)
		distance := levenshteinDistance(lowerInput, lowerCandidate)
		threshold := 1
		if len(lowerInput) > 3 {
			threshold = 2
		}
		if distance <= threshold && (bestDistance == -1 || distance < bestDistance) {
			bestDistance = distance
			best = candidate
		}
	}
	return best
}

// levenshteinDistance computes the classic edit distance between two strings.
func levenshteinDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[len(br)]
}
