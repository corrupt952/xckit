package command

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/google/subcommands"
	"xckit/xcstrings"
)

type ExportCommand struct {
	XCStringsCommand
	format string
	output string
}

func (*ExportCommand) Name() string {
	return "export"
}

func (*ExportCommand) Synopsis() string {
	return "Export strings to CSV"
}

func (*ExportCommand) Usage() string {
	return "export --format csv [-f file.xcstrings] [-o output.csv]: Export strings to CSV\n"
}

func (c *ExportCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.format, "format", "", "Export format (csv)")
	f.StringVar(&c.output, "o", "", "Output file path (default: stdout)")
}

func (c *ExportCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.format != "csv" {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: --format csv is required\n")
		return subcommands.ExitFailure
	}

	xc, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	var w io.Writer
	if c.output != "" {
		file, err := os.Create(c.output)
		if err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		defer file.Close()
		w = file
	} else {
		w = os.Stdout
	}

	if err := writeCSV(w, xc); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// csvRow represents a single row in the CSV output.
type csvRow struct {
	key             string
	comment         string
	shouldTranslate string
	// langData maps language code to [state, value].
	langData map[string][2]string
}

// writeCSV writes the xcstrings data as CSV to the given writer.
func writeCSV(w io.Writer, xc *xcstrings.XCStrings) error {
	langs := buildLanguageOrder(xc)
	rows := buildRows(xc, langs)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"key", "comment", "shouldTranslate"}
	for _, lang := range langs {
		header = append(header, lang+":state", lang)
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, row := range rows {
		record := []string{row.key, row.comment, row.shouldTranslate}
		for _, lang := range langs {
			if data, ok := row.langData[lang]; ok {
				record = append(record, data[0], data[1])
			} else {
				record = append(record, "", "")
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// buildLanguageOrder returns languages with source language first, then others alphabetically.
func buildLanguageOrder(xc *xcstrings.XCStrings) []string {
	langSet := make(map[string]bool)
	for _, def := range xc.Strings {
		for lang := range def.Localizations {
			langSet[lang] = true
		}
	}

	var others []string
	for lang := range langSet {
		if lang != xc.SourceLanguage {
			others = append(others, lang)
		}
	}
	sort.Strings(others)

	langs := []string{xc.SourceLanguage}
	langs = append(langs, others...)
	return langs
}

// buildRows flattens xcstrings into CSV rows, expanding variations.
func buildRows(xc *xcstrings.XCStrings, langs []string) []csvRow {
	keys := make([]string, 0, len(xc.Strings))
	for k := range xc.Strings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var rows []csvRow
	for _, key := range keys {
		def := xc.Strings[key]
		rows = append(rows, flattenKey(key, def, langs)...)
	}
	return rows
}

// flattenKey produces one or more csvRows for a key, expanding variations.
func flattenKey(key string, def xcstrings.StringDefinition, langs []string) []csvRow {
	comment := def.Comment
	shouldTranslate := ""
	if def.ShouldTranslate != nil && !*def.ShouldTranslate {
		shouldTranslate = "false"
	}

	// Determine the variation structure from the first language that has localizations.
	// We need to collect all variation suffixes across all languages.
	suffixes := collectVariationSuffixes(def, langs)

	if len(suffixes) == 0 {
		// Simple string, no variations
		row := csvRow{
			key:             key,
			comment:         comment,
			shouldTranslate: shouldTranslate,
			langData:        make(map[string][2]string),
		}
		for _, lang := range langs {
			loc, ok := def.Localizations[lang]
			if !ok {
				continue
			}
			if loc.StringUnit != nil {
				row.langData[lang] = [2]string{loc.StringUnit.State, loc.StringUnit.Value}
			}
		}
		return []csvRow{row}
	}

	// For keys with variations, produce one row per suffix.
	// Sort suffixes for deterministic output.
	sort.Strings(suffixes)

	var rows []csvRow
	for _, suffix := range suffixes {
		row := csvRow{
			key:             key + "[" + suffix + "]",
			comment:         comment,
			shouldTranslate: shouldTranslate,
			langData:        make(map[string][2]string),
		}
		// Only put comment on the first row
		if len(rows) > 0 {
			row.comment = ""
		}
		for _, lang := range langs {
			loc, ok := def.Localizations[lang]
			if !ok {
				continue
			}
			unit := resolveVariationUnit(loc, suffix)
			if unit != nil {
				row.langData[lang] = [2]string{unit.State, unit.Value}
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// collectVariationSuffixes gathers all unique variation path suffixes across all languages.
func collectVariationSuffixes(def xcstrings.StringDefinition, langs []string) []string {
	suffixSet := make(map[string]bool)
	for _, lang := range langs {
		loc, ok := def.Localizations[lang]
		if !ok {
			continue
		}
		if loc.Variations != nil {
			collectVariationPaths(loc.Variations, "", suffixSet)
		}
		for subName, sub := range loc.Substitutions {
			collectVariationPaths(&sub.Variations, "substitutions."+subName, suffixSet)
		}
	}

	suffixes := make([]string, 0, len(suffixSet))
	for s := range suffixSet {
		suffixes = append(suffixes, s)
	}
	return suffixes
}

// collectVariationPaths recursively collects leaf paths from Variations.
func collectVariationPaths(v *xcstrings.Variations, prefix string, out map[string]bool) {
	join := func(a, b string) string {
		if a == "" {
			return b
		}
		return a + "." + b
	}
	for cat, vv := range v.Plural {
		if vv == nil {
			continue
		}
		path := join(prefix, "plural."+cat)
		if vv.StringUnit != nil {
			out[path] = true
		}
		if vv.Variations != nil {
			collectVariationPaths(vv.Variations, path, out)
		}
	}
	for dev, vv := range v.Device {
		if vv == nil {
			continue
		}
		path := join(prefix, "device."+dev)
		if vv.StringUnit != nil {
			out[path] = true
		}
		if vv.Variations != nil {
			collectVariationPaths(vv.Variations, path, out)
		}
	}
}

// resolveVariationUnit finds the StringUnit for a given variation suffix path
// within a localization.
func resolveVariationUnit(loc xcstrings.Localization, suffix string) *xcstrings.StringUnit {
	// Parse the suffix path and navigate through the localization structure.
	// Examples: "plural.one", "device.iphone", "device.iphone.plural.one",
	// "substitutions.files.plural.one"
	parts := splitPath(suffix)
	if len(parts) == 0 {
		return nil
	}

	// Handle substitutions prefix
	if parts[0] == "substitutions" && len(parts) >= 2 {
		subName := parts[1]
		sub, ok := loc.Substitutions[subName]
		if !ok {
			return nil
		}
		return navigateVariations(&sub.Variations, parts[2:])
	}

	// Handle top-level variations
	if loc.Variations == nil {
		return nil
	}
	return navigateVariations(loc.Variations, parts)
}

// navigateVariations walks a Variations tree following the given path segments.
func navigateVariations(v *xcstrings.Variations, parts []string) *xcstrings.StringUnit {
	if len(parts) < 2 {
		return nil
	}

	varType := parts[0] // "plural" or "device"
	varKey := parts[1]
	remaining := parts[2:]

	var vv *xcstrings.VariationValue
	switch varType {
	case "plural":
		if v.Plural != nil {
			vv = v.Plural[varKey]
		}
	case "device":
		if v.Device != nil {
			vv = v.Device[varKey]
		}
	default:
		return nil
	}

	if vv == nil {
		return nil
	}

	if len(remaining) == 0 {
		return vv.StringUnit
	}

	if vv.Variations != nil {
		return navigateVariations(vv.Variations, remaining)
	}
	return nil
}

// splitPath splits a dot-separated path into segments.
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	parts = append(parts, path[start:])
	return parts
}
