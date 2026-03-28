// Package xcstrings provides functionality for working with Xcode String Catalogs (.xcstrings files).
package xcstrings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// XCStrings represents the structure of an Xcode String Catalog file.
type XCStrings struct {
	SourceLanguage string                      `json:"sourceLanguage"`
	Strings        map[string]StringDefinition `json:"strings"`
	Version        string                      `json:"version"`
}

// StringDefinition represents a string definition within an XCStrings file.
type StringDefinition struct {
	Comment         string                  `json:"comment,omitempty"`
	ExtractionState string                  `json:"extractionState,omitempty"`
	Localizations   map[string]Localization `json:"localizations"`
	ShouldTranslate *bool                   `json:"shouldTranslate,omitempty"`
}

// PluralCategory represents a CLDR plural category (zero, one, two, few, many, other).
type PluralCategory = string

// VariationValue represents a value within a variation that can itself contain
// a direct string unit or further nested variations.
type VariationValue struct {
	StringUnit *StringUnit `json:"stringUnit,omitempty"`
	Variations *Variations `json:"variations,omitempty"`
}

// Variations represents device and/or plural variations for a localization.
type Variations struct {
	Plural map[PluralCategory]*VariationValue `json:"plural,omitempty"`
	Device map[string]*VariationValue          `json:"device,omitempty"`
}

// Substitution represents a substitution within a localized string.
type Substitution struct {
	ArgNum          int        `json:"argNum"`
	FormatSpecifier string     `json:"formatSpecifier"`
	Variations      Variations `json:"variations"`
}

// Localization represents localization data for a specific language.
type Localization struct {
	StringUnit    *StringUnit              `json:"stringUnit,omitempty"`
	Variations    *Variations              `json:"variations,omitempty"`
	Substitutions map[string]Substitution  `json:"substitutions,omitempty"`
}

// StringUnit represents a string unit with translation state and value.
type StringUnit struct {
	State string `json:"state"`
	Value string `json:"value"`
}

// AllStringUnits recursively collects all leaf StringUnit pointers from a Localization.
// It traverses the top-level StringUnit, Variations (plural/device, possibly nested),
// and Substitutions to find every StringUnit in the tree.
func (l *Localization) AllStringUnits() []*StringUnit {
	var units []*StringUnit
	if l.StringUnit != nil {
		units = append(units, l.StringUnit)
	}
	if l.Variations != nil {
		units = append(units, l.Variations.allStringUnits()...)
	}
	for _, sub := range l.Substitutions {
		units = append(units, sub.Variations.allStringUnits()...)
	}
	return units
}

// allStringUnits recursively collects all leaf StringUnit pointers from Variations.
func (v *Variations) allStringUnits() []*StringUnit {
	var units []*StringUnit
	for _, vv := range v.Plural {
		if vv == nil {
			continue
		}
		if vv.StringUnit != nil {
			units = append(units, vv.StringUnit)
		}
		if vv.Variations != nil {
			units = append(units, vv.Variations.allStringUnits()...)
		}
	}
	for _, vv := range v.Device {
		if vv == nil {
			continue
		}
		if vv.StringUnit != nil {
			units = append(units, vv.StringUnit)
		}
		if vv.Variations != nil {
			units = append(units, vv.Variations.allStringUnits()...)
		}
	}
	return units
}

// Load reads and parses an XCStrings file from the given path.
func Load(path string) (*XCStrings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var xcstrings XCStrings
	if err := json.Unmarshal(data, &xcstrings); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Initialize nil localizations to empty maps to prevent null serialization
	for key, def := range xcstrings.Strings {
		if def.Localizations == nil {
			def.Localizations = make(map[string]Localization)
			xcstrings.Strings[key] = def
		}
	}

	return &xcstrings, nil
}

// SaveToFile writes the XCStrings data to a file at the given path using atomic writes.
// It writes to a temporary file in the same directory, syncs to disk, then renames
// to the target path to prevent data corruption from interrupted writes.
func (x *XCStrings) SaveToFile(path string) error {
	data, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".xcstrings-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Clean up temp file on error; after successful rename this is a no-op.
		os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Keys returns all string keys in the catalog.
func (x *XCStrings) Keys() []string {
	keys := make([]string, 0, len(x.Strings))
	for key := range x.Strings {
		keys = append(keys, key)
	}
	return keys
}

// StaleKeys returns keys that have extractionState "stale".
func (x *XCStrings) StaleKeys() []string {
	var keys []string
	for key, def := range x.Strings {
		if def.ExtractionState == "stale" {
			keys = append(keys, key)
		}
	}
	return keys
}

// ActiveKeys returns keys that are not stale (extractionState != "stale").
func (x *XCStrings) ActiveKeys() []string {
	var keys []string
	for key, def := range x.Strings {
		if def.ExtractionState != "stale" {
			keys = append(keys, key)
		}
	}
	return keys
}

// IsStale returns whether the given key has extractionState "stale".
func (x *XCStrings) IsStale(key string) bool {
	def, exists := x.Strings[key]
	if !exists {
		return false
	}
	return def.ExtractionState == "stale"
}

// RemoveStaleKeys removes all keys with extractionState "stale" from the catalog.
func (x *XCStrings) RemoveStaleKeys() int {
	count := 0
	for key, def := range x.Strings {
		if def.ExtractionState == "stale" {
			delete(x.Strings, key)
			count++
		}
	}
	return count
}

// UntranslatedKeys returns keys that are not translated for the given language.
// A key is untranslated if any leaf StringUnit (including within variations and
// substitutions) does not have the "translated" state. Stale keys are excluded.
func (x *XCStrings) UntranslatedKeys(language string) []string {
	var untranslated []string
	for key, definition := range x.Strings {
		if definition.ExtractionState == "stale" {
			continue
		}
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		localization, exists := definition.Localizations[language]
		if !exists {
			untranslated = append(untranslated, key)
			continue
		}
		units := localization.AllStringUnits()
		if len(units) == 0 {
			untranslated = append(untranslated, key)
			continue
		}
		for _, u := range units {
			if u.State != "translated" {
				untranslated = append(untranslated, key)
				break
			}
		}
	}
	return untranslated
}

// Languages returns all available languages in the catalog.
func (x *XCStrings) Languages() []string {
	languageSet := make(map[string]bool)
	for _, definition := range x.Strings {
		for lang := range definition.Localizations {
			// Exclude source language from the list
			if lang != x.SourceLanguage {
				languageSet[lang] = true
			}
		}
	}

	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}
	return languages
}

func (x *XCStrings) SetTranslation(key, language, value string) error {
	definition, exists := x.Strings[key]
	if !exists {
		return fmt.Errorf("key '%s' not found", key)
	}

	if definition.Localizations == nil {
		definition.Localizations = make(map[string]Localization)
	}

	loc := definition.Localizations[language]
	loc.StringUnit = &StringUnit{
		State: "translated",
		Value: value,
	}
	definition.Localizations[language] = loc

	x.Strings[key] = definition
	return nil
}

// KeysWithAnyUntranslated returns keys that have at least one untranslated language.
// A language is considered untranslated if any leaf StringUnit (including within
// variations and substitutions) does not have the "translated" state.
// Stale keys are excluded.
func (x *XCStrings) KeysWithAnyUntranslated() []string {
	var result []string
	languages := x.Languages()

	for key, definition := range x.Strings {
		if definition.ExtractionState == "stale" {
			continue
		}
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		hasUntranslated := false
		for _, lang := range languages {
			localization, exists := definition.Localizations[lang]
			if !exists {
				hasUntranslated = true
				break
			}
			units := localization.AllStringUnits()
			if len(units) == 0 {
				hasUntranslated = true
				break
			}
			for _, u := range units {
				if u.State != "translated" {
					hasUntranslated = true
					break
				}
			}
			if hasUntranslated {
				break
			}
		}
		if hasUntranslated {
			result = append(result, key)
		}
	}

	return result
}

// NeedsReviewKeys returns keys that have needs_review state for the given language.
func (x *XCStrings) NeedsReviewKeys(language string) []string {
	var keys []string
	for key, def := range x.Strings {
		if def.ShouldTranslate != nil && !*def.ShouldTranslate {
			continue
		}
		loc, exists := def.Localizations[language]
		if exists && loc.StringUnit != nil && loc.StringUnit.State == "needs_review" {
			keys = append(keys, key)
		}
	}
	return keys
}

// TranslatedKeys returns keys that are translated for the given language.
// A key is translated only if all leaf StringUnits (including within variations
// and substitutions) have the "translated" state.
func (x *XCStrings) TranslatedKeys(language string) []string {
	var translated []string
	for key, definition := range x.Strings {
		localization, exists := definition.Localizations[language]
		if !exists {
			continue
		}
		units := localization.AllStringUnits()
		if len(units) == 0 {
			continue
		}
		allTranslated := true
		for _, u := range units {
			if u.State != "translated" {
				allTranslated = false
				break
			}
		}
		if allTranslated {
			translated = append(translated, key)
		}
	}
	return translated
}

// UntranslatedDetail represents a single untranslated leaf with its path.
type UntranslatedDetail struct {
	Key      string
	Language string
	Path     string // e.g. "plural.other", "device.iphone.plural.one", "substitutions.files.plural.one"
}

// UntranslatedDetailsForLanguage returns detailed paths for all untranslated
// leaf string units for a given language across all keys.
func (x *XCStrings) UntranslatedDetailsForLanguage(language string) []UntranslatedDetail {
	var details []UntranslatedDetail
	for key, definition := range x.Strings {
		if definition.ExtractionState == "stale" {
			continue
		}
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		localization, exists := definition.Localizations[language]
		if !exists {
			details = append(details, UntranslatedDetail{Key: key, Language: language, Path: "missing"})
			continue
		}
		details = append(details, localization.untranslatedPaths(key, language)...)
	}
	return details
}

// UntranslatedDetailsForAllLanguages returns detailed paths for all untranslated
// leaf string units across all languages and keys.
func (x *XCStrings) UntranslatedDetailsForAllLanguages() []UntranslatedDetail {
	var details []UntranslatedDetail
	languages := x.Languages()
	for key, definition := range x.Strings {
		if definition.ExtractionState == "stale" {
			continue
		}
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		for _, lang := range languages {
			localization, exists := definition.Localizations[lang]
			if !exists {
				details = append(details, UntranslatedDetail{Key: key, Language: lang, Path: "missing"})
				continue
			}
			details = append(details, localization.untranslatedPaths(key, lang)...)
		}
	}
	return details
}

// untranslatedPaths collects paths of untranslated leaf string units.
func (l *Localization) untranslatedPaths(key, language string) []UntranslatedDetail {
	var details []UntranslatedDetail
	if l.StringUnit != nil {
		if l.StringUnit.State != "translated" {
			details = append(details, UntranslatedDetail{Key: key, Language: language, Path: l.StringUnit.State})
		}
	}
	if l.Variations != nil {
		details = append(details, l.Variations.untranslatedPaths(key, language, "")...)
	}
	for name, sub := range l.Substitutions {
		details = append(details, sub.Variations.untranslatedPaths(key, language, "substitutions."+name)...)
	}
	return details
}

// untranslatedPaths recursively collects paths of untranslated leaf string units.
func (v *Variations) untranslatedPaths(key, language, prefix string) []UntranslatedDetail {
	var details []UntranslatedDetail
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
		if vv.StringUnit != nil && vv.StringUnit.State != "translated" {
			details = append(details, UntranslatedDetail{Key: key, Language: language, Path: path})
		}
		if vv.Variations != nil {
			details = append(details, vv.Variations.untranslatedPaths(key, language, path)...)
		}
	}
	for dev, vv := range v.Device {
		if vv == nil {
			continue
		}
		path := join(prefix, "device."+dev)
		if vv.StringUnit != nil && vv.StringUnit.State != "translated" {
			details = append(details, UntranslatedDetail{Key: key, Language: language, Path: path})
		}
		if vv.Variations != nil {
			details = append(details, vv.Variations.untranslatedPaths(key, language, path)...)
		}
	}
	return details
}

// FilterKeysByPrefix returns keys that start with the given prefix.
func (x *XCStrings) FilterKeysByPrefix(keys []string, prefix string) []string {
	if prefix == "" {
		return keys
	}

	var filtered []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}
