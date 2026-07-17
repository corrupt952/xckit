package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"sort"

	"xckit/formatter"
	"xckit/xcstrings"

	"github.com/google/subcommands"
)

type ListCommand struct {
	XCStringsCommand
	prefix     string
	state      string
	jsonOutput bool
}

func (*ListCommand) Name() string {
	return "list"
}

func (*ListCommand) Synopsis() string {
	return "List all keys with translation status"
}

func (*ListCommand) Usage() string {
	return "list [-f file.xcstrings] [--prefix <prefix>] [--state <state>] [--json]: List all keys with translation status\n"
}

func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.prefix, "prefix", "", "Filter keys by prefix")
	f.StringVar(&c.state, "state", "", "Filter keys by extractionState (e.g. manual, stale, new)")
	f.BoolVar(&c.jsonOutput, "json", false, "Output a single JSON document to stdout instead of human-readable text")
}

func (c *ListCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	keysToShow := xcstrings.Keys()
	keysToShow = xcstrings.FilterKeysByPrefix(keysToShow, c.prefix)
	if c.state != "" {
		stateSet := make(map[string]bool, len(keysToShow))
		for _, k := range xcstrings.KeysByState(c.state) {
			stateSet[k] = true
		}
		filtered := keysToShow[:0]
		for _, k := range keysToShow {
			if stateSet[k] {
				filtered = append(filtered, k)
			}
		}
		keysToShow = filtered
	}
	sort.Strings(keysToShow)

	if c.jsonOutput {
		return c.printJSON(xcstrings, keysToShow)
	}

	if len(keysToShow) == 0 {
		switch {
		case c.prefix != "" && c.state != "":
			fmt.Printf("No keys found with prefix '%s' and state '%s'\n", c.prefix, c.state)
		case c.prefix != "":
			fmt.Printf("No keys found with prefix '%s'\n", c.prefix)
		case c.state != "":
			fmt.Printf("No keys found with state '%s'\n", c.state)
		default:
			fmt.Println("No keys found")
		}
		return subcommands.ExitSuccess
	}

	switch {
	case c.prefix != "" && c.state != "":
		fmt.Printf("Keys with prefix '%s' and state '%s':\n", c.prefix, c.state)
	case c.prefix != "":
		fmt.Printf("Keys with prefix '%s':\n", c.prefix)
	case c.state != "":
		fmt.Printf("Keys with state '%s':\n", c.state)
	default:
		fmt.Println("All keys with translation status:")
	}
	formatter.DisplayKeyDetails(xcstrings, keysToShow)
	return subcommands.ExitSuccess
}

// listJSONOutput is the top-level document printed by `list --json`.
type listJSONOutput struct {
	Keys []listJSONKeyEntry `json:"keys"`
}

// listJSONKeyEntry describes a single key and its per-language translation state.
type listJSONKeyEntry struct {
	Key             string                       `json:"key"`
	ExtractionState string                       `json:"extractionState,omitempty"`
	Languages       map[string]listJSONLangEntry `json:"languages"`
}

// listJSONLangEntry describes a key's translation state for one language.
type listJSONLangEntry struct {
	State string              `json:"state"`
	Value string              `json:"value,omitempty"`
	Units []listJSONUnitEntry `json:"units,omitempty"`
}

// listJSONUnitEntry describes a single leaf string unit within a variation/substitution tree.
type listJSONUnitEntry struct {
	Path  string `json:"path"`
	State string `json:"state"`
	Value string `json:"value"`
}

// printJSON marshals the given keys to a single JSON document and writes it to stdout.
func (c *ListCommand) printJSON(xcs *xcstrings.XCStrings, keys []string) subcommands.ExitStatus {
	languages := allDisplayLanguages(xcs)

	out := listJSONOutput{Keys: make([]listJSONKeyEntry, 0, len(keys))}
	for _, key := range keys {
		def := xcs.Strings[key]
		entry := listJSONKeyEntry{
			Key:             key,
			ExtractionState: def.ExtractionState,
			Languages:       make(map[string]listJSONLangEntry, len(languages)),
		}
		for _, lang := range languages {
			loc, exists := def.Localizations[lang]
			entry.Languages[lang] = buildLangEntry(loc, exists)
		}
		out.Keys = append(out.Keys, entry)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Println(string(data))
	return subcommands.ExitSuccess
}

// allDisplayLanguages returns the source language plus all catalog languages, sorted.
func allDisplayLanguages(xcs *xcstrings.XCStrings) []string {
	langSet := make(map[string]bool)
	if xcs.SourceLanguage != "" {
		langSet[xcs.SourceLanguage] = true
	}
	for _, l := range xcs.Languages() {
		langSet[l] = true
	}
	languages := make([]string, 0, len(langSet))
	for l := range langSet {
		languages = append(languages, l)
	}
	sort.Strings(languages)
	return languages
}

// buildLangEntry builds the JSON representation of a single localization entry.
func buildLangEntry(loc xcstrings.Localization, exists bool) listJSONLangEntry {
	if !exists {
		return listJSONLangEntry{State: "missing"}
	}

	entry := listJSONLangEntry{}
	if loc.StringUnit != nil {
		entry.Value = loc.StringUnit.Value
	}
	entry.Units = localizationUnits(loc)
	entry.State = aggregateUnitState(loc.AllStringUnits())
	return entry
}

// localizationUnits collects every leaf string unit within a localization's
// variations and substitutions, tagged with its variation path.
func localizationUnits(loc xcstrings.Localization) []listJSONUnitEntry {
	var units []listJSONUnitEntry
	units = append(units, variationUnits(loc.Variations, "")...)

	subNames := make([]string, 0, len(loc.Substitutions))
	for name := range loc.Substitutions {
		subNames = append(subNames, name)
	}
	sort.Strings(subNames)
	for _, name := range subNames {
		sub := loc.Substitutions[name]
		units = append(units, variationUnits(&sub.Variations, "substitutions."+name)...)
	}
	return units
}

// variationUnits recursively collects leaf string units from a Variations tree.
func variationUnits(v *xcstrings.Variations, prefix string) []listJSONUnitEntry {
	if v == nil {
		return nil
	}

	join := func(a, b string) string {
		if a == "" {
			return b
		}
		return a + "." + b
	}

	var units []listJSONUnitEntry

	categories := make([]string, 0, len(v.Plural))
	for cat := range v.Plural {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	for _, cat := range categories {
		vv := v.Plural[cat]
		if vv == nil {
			continue
		}
		path := join(prefix, "plural."+cat)
		if vv.StringUnit != nil {
			units = append(units, listJSONUnitEntry{Path: path, State: vv.StringUnit.State, Value: vv.StringUnit.Value})
		}
		units = append(units, variationUnits(vv.Variations, path)...)
	}

	devices := make([]string, 0, len(v.Device))
	for dev := range v.Device {
		devices = append(devices, dev)
	}
	sort.Strings(devices)
	for _, dev := range devices {
		vv := v.Device[dev]
		if vv == nil {
			continue
		}
		path := join(prefix, "device."+dev)
		if vv.StringUnit != nil {
			units = append(units, listJSONUnitEntry{Path: path, State: vv.StringUnit.State, Value: vv.StringUnit.Value})
		}
		units = append(units, variationUnits(vv.Variations, path)...)
	}

	return units
}

// aggregateUnitState summarizes a set of leaf string units into a single
// state: "missing" when there are none, "translated" when all leaves are
// translated, or the state of the first non-translated leaf otherwise.
func aggregateUnitState(units []*xcstrings.StringUnit) string {
	if len(units) == 0 {
		return "missing"
	}
	for _, u := range units {
		if u.State != "translated" {
			return u.State
		}
	}
	return "translated"
}
