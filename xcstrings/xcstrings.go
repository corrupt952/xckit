// Package xcstrings provides functionality for working with Xcode String Catalogs (.xcstrings files).
package xcstrings

import (
	"encoding/json"
	"fmt"
	"os"
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

// Localization represents localization data for a specific language.
type Localization struct {
	StringUnit StringUnit `json:"stringUnit"`
}

// StringUnit represents a string unit with translation state and value.
type StringUnit struct {
	State string `json:"state"`
	Value string `json:"value"`
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

// SaveToFile writes the XCStrings data to a file at the given path.
func (x *XCStrings) SaveToFile(path string) error {
	data, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
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

// UntranslatedKeys returns keys that are not translated for the given language.
func (x *XCStrings) UntranslatedKeys(language string) []string {
	var untranslated []string
	for key, definition := range x.Strings {
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		localization, exists := definition.Localizations[language]
		if !exists || localization.StringUnit.State != "translated" {
			untranslated = append(untranslated, key)
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

	definition.Localizations[language] = Localization{
		StringUnit: StringUnit{
			State: "translated",
			Value: value,
		},
	}

	x.Strings[key] = definition
	return nil
}

// KeysWithAnyUntranslated returns keys that have at least one untranslated language.
func (x *XCStrings) KeysWithAnyUntranslated() []string {
	var result []string
	languages := x.Languages()

	for key, definition := range x.Strings {
		if definition.ShouldTranslate != nil && *definition.ShouldTranslate == false {
			continue
		}
		hasUntranslated := false
		for _, lang := range languages {
			localization, exists := definition.Localizations[lang]
			if !exists || localization.StringUnit.State != "translated" {
				hasUntranslated = true
				break
			}
		}
		if hasUntranslated {
			result = append(result, key)
		}
	}

	return result
}

// TranslatedKeys returns keys that are translated for the given language.
func (x *XCStrings) TranslatedKeys(language string) []string {
	var translated []string
	for key, definition := range x.Strings {
		if localization, exists := definition.Localizations[language]; exists {
			if localization.StringUnit.State == "translated" {
				translated = append(translated, key)
			}
		}
	}
	return translated
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
