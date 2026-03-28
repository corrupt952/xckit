package command

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/subcommands"
	"xckit/helper/atomicwrite"
	"xckit/xcstrings"
)

type ImportCommand struct {
	XCStringsCommand
	format       string
	dryRun       bool
	backup       bool
	onMissingKey string
	clearEmpty   bool
}

func (*ImportCommand) Name() string {
	return "import"
}

func (*ImportCommand) Synopsis() string {
	return "Import translations from CSV"
}

func (*ImportCommand) Usage() string {
	return "import --format csv [-f file.xcstrings] [--dry-run] [--backup] [--on-missing-key skip|error] [--clear-empty] <csv-file>: Import translations from CSV\n"
}

func (c *ImportCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.format, "format", "", "Import format (csv)")
	f.BoolVar(&c.dryRun, "dry-run", false, "Show change summary without writing")
	f.BoolVar(&c.backup, "backup", false, "Copy original to .bak before writing")
	f.StringVar(&c.onMissingKey, "on-missing-key", "skip", "Action for missing keys: skip or error")
	f.BoolVar(&c.clearEmpty, "clear-empty", false, "Clear translations for empty CSV cells")
}

func (c *ImportCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.format != "csv" {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: --format csv is required\n")
		return subcommands.ExitFailure
	}

	if c.onMissingKey != "skip" && c.onMissingKey != "error" {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: --on-missing-key must be skip or error\n")
		return subcommands.ExitFailure
	}

	args := f.Args()
	if len(args) < 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: CSV file path is required\n")
		return subcommands.ExitFailure
	}
	csvPath := args[0]

	xc, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	csvFile, err := os.Open(csvPath)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	defer csvFile.Close()

	summary, err := importCSV(csvFile, xc, c.onMissingKey, c.clearEmpty)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.dryRun {
		fmt.Fprintf(os.Stdout, "Dry run: %d updated, %d skipped, %d cleared\n", summary.updated, summary.skipped, summary.cleared)
		return subcommands.ExitSuccess
	}

	xcPath := c.filePath
	if xcPath == "" {
		xcPath = c.findXCStringsFile()
	}

	if c.backup {
		if xcPath != "" {
			data, err := os.ReadFile(xcPath)
			if err != nil {
				fmt.Fprintf(flag.CommandLine.Output(), "Error creating backup: %v\n", err)
				return subcommands.ExitFailure
			}
			if err := atomicwrite.WriteFile(xcPath+".bak", data, 0644); err != nil {
				fmt.Fprintf(flag.CommandLine.Output(), "Error creating backup: %v\n", err)
				return subcommands.ExitFailure
			}
		}
	}
	if err := xc.SaveToFile(xcPath); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Fprintf(os.Stdout, "Imported: %d updated, %d skipped, %d cleared\n", summary.updated, summary.skipped, summary.cleared)
	return subcommands.ExitSuccess
}

type importSummary struct {
	updated int
	skipped int
	cleared int
}

// importCSV reads CSV data and applies translations to the xcstrings catalog.
func importCSV(r io.Reader, xc *xcstrings.XCStrings, onMissingKey string, clearEmpty bool) (*importSummary, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 1 {
		return &importSummary{}, nil
	}

	header := records[0]
	langColumns, err := parseHeader(header)
	if err != nil {
		return nil, err
	}

	summary := &importSummary{}

	for _, record := range records[1:] {
		if len(record) < 1 {
			continue
		}
		rawKey := record[0]
		// column 1 = comment (ignored)
		// column 2 = shouldTranslate (ignored)

		baseKey, variationPath := parseKeyBracket(rawKey)

		_, keyExists := xc.Strings[baseKey]
		if !keyExists {
			if onMissingKey == "error" {
				return nil, fmt.Errorf("key not found: %s", baseKey)
			}
			summary.skipped++
			continue
		}

		for _, lc := range langColumns {
			// Skip source language
			if lc.lang == xc.SourceLanguage {
				continue
			}

			if lc.valueIdx >= len(record) {
				continue
			}
			value := record[lc.valueIdx]

			if value == "" {
				if clearEmpty {
					if err := clearTranslation(xc, baseKey, lc.lang, variationPath); err == nil {
						summary.cleared++
					}
				}
				continue
			}

			if err := setTranslation(xc, baseKey, lc.lang, value, variationPath); err != nil {
				log.Printf("Warning: skipping key %q lang %q: %v", baseKey, lc.lang, err)
				summary.skipped++
				continue
			}
			summary.updated++
		}
	}

	return summary, nil
}

// langColumn represents a language column pair in the CSV header.
type langColumn struct {
	lang      string
	stateIdx  int // index of {lang}:state column
	valueIdx  int // index of {lang} value column
}

// parseHeader parses the CSV header to find language columns.
// Expected format: key, comment, shouldTranslate, {lang}:state, {lang}, ...
func parseHeader(header []string) ([]langColumn, error) {
	if len(header) < 3 {
		return nil, fmt.Errorf("CSV header must have at least 3 columns (key, comment, shouldTranslate)")
	}

	var cols []langColumn
	i := 3 // skip key, comment, shouldTranslate
	for i < len(header) {
		col := header[i]
		if strings.HasSuffix(col, ":state") {
			lang := strings.TrimSuffix(col, ":state")
			if i+1 < len(header) && header[i+1] == lang {
				cols = append(cols, langColumn{
					lang:     lang,
					stateIdx: i,
					valueIdx: i + 1,
				})
				i += 2
				continue
			}
		}
		i++
	}

	return cols, nil
}

// parseKeyBracket splits "key[suffix]" into (key, suffix).
// If no brackets, returns (key, "").
func parseKeyBracket(raw string) (string, string) {
	idx := strings.Index(raw, "[")
	if idx < 0 {
		return raw, ""
	}
	end := strings.LastIndex(raw, "]")
	if end <= idx {
		return raw, ""
	}
	return raw[:idx], raw[idx+1 : end]
}

// setTranslation sets a translation value, handling both simple and variation keys.
func setTranslation(xc *xcstrings.XCStrings, key, lang, value, variationPath string) error {
	if variationPath == "" {
		return xc.SetTranslation(key, lang, value)
	}

	parts := splitPath(variationPath)

	// Handle substitutions
	if len(parts) >= 2 && parts[0] == "substitutions" {
		return setSubstitutionTranslation(xc, key, lang, value, parts[1], parts[2:])
	}

	// Handle plural/device variations
	opts, err := parseVariationOpts(parts)
	if err != nil {
		return err
	}
	_, err = xc.SetVariationTranslation(key, lang, value, opts)
	return err
}

// clearTranslation clears a translation for a key/lang/variation path.
func clearTranslation(xc *xcstrings.XCStrings, key, lang, variationPath string) error {
	def, exists := xc.Strings[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	loc, exists := def.Localizations[lang]
	if !exists {
		return nil // nothing to clear
	}

	if variationPath == "" {
		if loc.StringUnit != nil {
			loc.StringUnit = nil
			def.Localizations[lang] = loc
			xc.Strings[key] = def
		}
		return nil
	}

	parts := splitPath(variationPath)

	if len(parts) >= 2 && parts[0] == "substitutions" {
		return clearSubstitutionVariation(&loc, &def, xc, key, lang, parts[1], parts[2:])
	}

	clearVariationUnit(loc.Variations, parts)
	def.Localizations[lang] = loc
	xc.Strings[key] = def
	return nil
}

// clearVariationUnit navigates and clears a variation leaf.
func clearVariationUnit(v *xcstrings.Variations, parts []string) {
	if v == nil || len(parts) < 2 {
		return
	}
	varType := parts[0]
	varKey := parts[1]
	remaining := parts[2:]

	switch varType {
	case "plural":
		if v.Plural == nil {
			return
		}
		vv := v.Plural[varKey]
		if vv == nil {
			return
		}
		if len(remaining) == 0 {
			vv.StringUnit = nil
		} else if vv.Variations != nil {
			clearVariationUnit(vv.Variations, remaining)
		}
	case "device":
		if v.Device == nil {
			return
		}
		vv := v.Device[varKey]
		if vv == nil {
			return
		}
		if len(remaining) == 0 {
			vv.StringUnit = nil
		} else if vv.Variations != nil {
			clearVariationUnit(vv.Variations, remaining)
		}
	}
}

// clearSubstitutionVariation clears a substitution variation leaf.
func clearSubstitutionVariation(loc *xcstrings.Localization, def *xcstrings.StringDefinition, xc *xcstrings.XCStrings, key, lang, subName string, parts []string) error {
	sub, ok := loc.Substitutions[subName]
	if !ok {
		return nil
	}
	clearVariationUnit(&sub.Variations, parts)
	loc.Substitutions[subName] = sub
	def.Localizations[lang] = *loc
	xc.Strings[key] = *def
	return nil
}

// setSubstitutionTranslation sets a translation within a substitution variation.
func setSubstitutionTranslation(xc *xcstrings.XCStrings, key, lang, value, subName string, parts []string) error {
	def, exists := xc.Strings[key]
	if !exists {
		return fmt.Errorf("key '%s' not found", key)
	}

	if def.Localizations == nil {
		def.Localizations = make(map[string]xcstrings.Localization)
	}

	loc := def.Localizations[lang]
	if loc.Substitutions == nil {
		loc.Substitutions = make(map[string]xcstrings.Substitution)
	}

	sub := loc.Substitutions[subName]

	unit := &xcstrings.StringUnit{
		State: "translated",
		Value: value,
	}

	setVariationUnit(&sub.Variations, parts, unit)

	loc.Substitutions[subName] = sub
	def.Localizations[lang] = loc
	xc.Strings[key] = def
	return nil
}

// setVariationUnit navigates and sets a variation leaf.
func setVariationUnit(v *xcstrings.Variations, parts []string, unit *xcstrings.StringUnit) {
	if len(parts) < 2 {
		return
	}
	varType := parts[0]
	varKey := parts[1]
	remaining := parts[2:]

	switch varType {
	case "plural":
		if v.Plural == nil {
			v.Plural = make(map[xcstrings.PluralCategory]*xcstrings.VariationValue)
		}
		if len(remaining) == 0 {
			v.Plural[varKey] = &xcstrings.VariationValue{StringUnit: unit}
		} else {
			vv := v.Plural[varKey]
			if vv == nil {
				vv = &xcstrings.VariationValue{}
			}
			if vv.Variations == nil {
				vv.Variations = &xcstrings.Variations{}
			}
			setVariationUnit(vv.Variations, remaining, unit)
			v.Plural[varKey] = vv
		}
	case "device":
		if v.Device == nil {
			v.Device = make(map[string]*xcstrings.VariationValue)
		}
		if len(remaining) == 0 {
			v.Device[varKey] = &xcstrings.VariationValue{StringUnit: unit}
		} else {
			vv := v.Device[varKey]
			if vv == nil {
				vv = &xcstrings.VariationValue{}
			}
			if vv.Variations == nil {
				vv.Variations = &xcstrings.Variations{}
			}
			setVariationUnit(vv.Variations, remaining, unit)
			v.Device[varKey] = vv
		}
	}
}

// parseVariationOpts converts path parts to VariationOptions.
// Supports: plural.X, device.X, device.X.plural.Y
// Returns an error if parts contain unrecognized segments or have an odd length.
func parseVariationOpts(parts []string) (xcstrings.VariationOptions, error) {
	opts := xcstrings.VariationOptions{}
	if len(parts)%2 != 0 {
		return opts, fmt.Errorf("invalid variation path: odd number of segments %v", parts)
	}
	for i := 0; i < len(parts)-1; i += 2 {
		switch parts[i] {
		case "plural":
			opts.Plural = parts[i+1]
		case "device":
			opts.Device = parts[i+1]
		default:
			return opts, fmt.Errorf("unrecognized variation segment %q in path %v", parts[i], parts)
		}
	}
	return opts, nil
}
