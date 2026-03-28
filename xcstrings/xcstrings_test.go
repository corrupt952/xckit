package xcstrings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"xckit/helper/test"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantKeys  []string
		wantLangs []string
	}{
		{
			name: "valid xcstrings file",
			content: `{
				"sourceLanguage": "en",
				"strings": {
					"hello": {
						"localizations": {
							"en": {
								"stringUnit": {
									"state": "translated",
									"value": "Hello"
								}
							},
							"ja": {
								"stringUnit": {
									"state": "translated",
									"value": "こんにちは"
								}
							}
						}
					}
				},
				"version": "1.0"
			}`,
			wantErr:   false,
			wantKeys:  []string{"hello"},
			wantLangs: []string{"ja"}, // en is excluded as source language
		},
		{
			name:    "invalid json",
			content: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test.TempFile(t, "test.xcstrings", tt.content)

			xcstrings, err := Load(filePath)

			if tt.wantErr {
				test.AssertError(t, err)
				return
			}

			test.AssertNoError(t, err)

			if xcstrings == nil {
				t.Fatal("xcstrings should not be nil")
			}

			// Test keys
			keys := xcstrings.Keys()
			sort.Strings(keys)
			sort.Strings(tt.wantKeys)
			test.AssertSliceEqual(t, keys, tt.wantKeys)

			// Test languages
			langs := xcstrings.Languages()
			sort.Strings(langs)
			sort.Strings(tt.wantLangs)
			test.AssertSliceEqual(t, langs, tt.wantLangs)
		})
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	_, err := Load("nonexistent.xcstrings")
	test.AssertError(t, err)
}

func TestXCStrings_GetUntranslatedKeys(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"translated_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "こんにちは"}},
				},
			},
			"untranslated_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Untranslated"}},
					"ja": {StringUnit: &StringUnit{State: "new", Value: "未翻訳"}},
				},
			},
			"missing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Missing"}},
				},
			},
			"should_not_translate": {
				ShouldTranslate: func() *bool { b := false; return &b }(),
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Don't translate"}},
				},
			},
		},
	}

	tests := []struct {
		name     string
		language string
		want     []string
	}{
		{
			name:     "japanese untranslated",
			language: "ja",
			want:     []string{"untranslated_key", "missing_key"},
		},
		{
			name:     "english all translated",
			language: "en",
			want:     []string{},
		},
		{
			name:     "nonexistent language",
			language: "fr",
			want:     []string{"translated_key", "untranslated_key", "missing_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xcstrings.UntranslatedKeys(tt.language)
			sort.Strings(got)
			sort.Strings(tt.want)
			test.AssertSliceEqual(t, got, tt.want)
		})
	}
}

func TestXCStrings_GetKeysWithAnyUntranslated(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"all_translated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "こんにちは"}},
					"es": {StringUnit: &StringUnit{State: "translated", Value: "Hola"}},
				},
			},
			"ja_untranslated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: &StringUnit{State: "new", Value: ""}},
					"es": {StringUnit: &StringUnit{State: "translated", Value: "Español"}},
				},
			},
			"es_missing": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "English only"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "日本語"}},
					// es is missing - should be considered untranslated
				},
			},
			"only_en_translated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: &StringUnit{State: "new", Value: ""}},
					// es is missing
				},
			},
			"all_untranslated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "new", Value: ""}},
					"ja": {StringUnit: &StringUnit{State: "new", Value: ""}},
					"es": {StringUnit: &StringUnit{State: "new", Value: ""}},
				},
			},
			"should_not_translate": {
				ShouldTranslate: func() *bool { b := false; return &b }(),
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Don't translate"}},
					// ja and es are missing, but this should not appear in untranslated list
				},
			},
		},
	}

	got := xcstrings.KeysWithAnyUntranslated()
	// all_translated should NOT be in the list
	// should_not_translate should NOT be in the list because shouldTranslate=false
	want := []string{"ja_untranslated", "es_missing", "only_en_translated", "all_untranslated"}

	sort.Strings(got)
	sort.Strings(want)
	test.AssertSliceEqual(t, got, want)
}

func TestXCStrings_ShouldTranslateFlag(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"normal_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Normal"}},
				},
			},
			"should_translate_true": {
				ShouldTranslate: func() *bool { b := true; return &b }(),
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Translate me"}},
				},
			},
			"should_translate_false": {
				ShouldTranslate: func() *bool { b := false; return &b }(),
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Don't translate"}},
				},
			},
			"placeholder_key": {
				ShouldTranslate: func() *bool { b := false; return &b }(),
				Localizations:   map[string]Localization{},
			},
			"already_translated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Already translated"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "翻訳済み"}},
				},
			},
		},
	}

	t.Run("UntranslatedKeys should skip shouldTranslate=false", func(t *testing.T) {
		untranslated := xcstrings.UntranslatedKeys("ja")
		want := []string{"normal_key", "should_translate_true"}

		sort.Strings(untranslated)
		sort.Strings(want)
		test.AssertSliceEqual(t, untranslated, want)
	})

	t.Run("KeysWithAnyUntranslated should skip shouldTranslate=false", func(t *testing.T) {
		keysWithUntranslated := xcstrings.KeysWithAnyUntranslated()
		want := []string{"normal_key", "should_translate_true"}

		sort.Strings(keysWithUntranslated)
		sort.Strings(want)
		test.AssertSliceEqual(t, keysWithUntranslated, want)
	})
}

func TestXCStrings_SetTranslation(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"existing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Existing"}},
				},
			},
		},
	}

	tests := []struct {
		name     string
		key      string
		language string
		value    string
		wantErr  bool
	}{
		{
			name:     "set existing key",
			key:      "existing_key",
			language: "ja",
			value:    "既存",
			wantErr:  false,
		},
		{
			name:     "set nonexistent key",
			key:      "nonexistent_key",
			language: "ja",
			value:    "存在しない",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := xcstrings.SetTranslation(tt.key, tt.language, tt.value)

			if tt.wantErr {
				test.AssertError(t, err)
				return
			}

			test.AssertNoError(t, err)

			// Verify the translation was set
			localization, exists := xcstrings.Strings[tt.key].Localizations[tt.language]
			if !exists {
				t.Errorf("translation not set for key %s, language %s", tt.key, tt.language)
				return
			}

			test.AssertEqual(t, localization.StringUnit.State, "translated")
			test.AssertEqual(t, localization.StringUnit.Value, tt.value)
		})
	}
}

func TestXCStrings_SetTranslation_PreservesExistingLocalization(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"greeting": {
				Comment:         "A greeting message",
				ExtractionState: "manual",
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "こんにちは"}},
				},
			},
		},
	}

	// Update the Japanese translation
	err := xcstrings.SetTranslation("greeting", "ja", "やあ")
	test.AssertNoError(t, err)

	// Verify the translation was updated
	loc := xcstrings.Strings["greeting"].Localizations["ja"]
	test.AssertEqual(t, loc.StringUnit.State, "translated")
	test.AssertEqual(t, loc.StringUnit.Value, "やあ")

	// Verify other localizations are not affected
	enLoc := xcstrings.Strings["greeting"].Localizations["en"]
	test.AssertEqual(t, enLoc.StringUnit.Value, "Hello")

	// Verify StringDefinition-level fields are preserved
	def := xcstrings.Strings["greeting"]
	test.AssertEqual(t, def.Comment, "A greeting message")
	test.AssertEqual(t, def.ExtractionState, "manual")
}

func TestFilterKeysByPrefix(t *testing.T) {
	xcstrings := &XCStrings{}

	tests := []struct {
		name     string
		keys     []string
		prefix   string
		expected []string
	}{
		{
			name:     "empty prefix returns all keys",
			keys:     []string{"login.title", "login.button", "settings.title"},
			prefix:   "",
			expected: []string{"login.title", "login.button", "settings.title"},
		},
		{
			name:     "filter by login prefix",
			keys:     []string{"login.title", "login.button", "settings.title", "logout.button"},
			prefix:   "login",
			expected: []string{"login.title", "login.button"},
		},
		{
			name:     "no matching keys",
			keys:     []string{"login.title", "settings.title"},
			prefix:   "error",
			expected: []string{},
		},
		{
			name:     "exact match",
			keys:     []string{"login", "login.title", "settings"},
			prefix:   "login",
			expected: []string{"login", "login.title"},
		},
		{
			name:     "case sensitive",
			keys:     []string{"Login.title", "login.button"},
			prefix:   "login",
			expected: []string{"login.button"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xcstrings.FilterKeysByPrefix(tt.keys, tt.prefix)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(result))
				return
			}

			for i, key := range result {
				if key != tt.expected[i] {
					t.Errorf("expected key[%d] to be %s, got %s", i, tt.expected[i], key)
				}
			}
		})
	}
}

func TestXCStrings_SaveToFile(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"test_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Test"}},
				},
			},
		},
		Version: "1.0",
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.xcstrings")

	err := xcstrings.SaveToFile(filePath)
	test.AssertNoError(t, err)

	// Verify file was created and can be loaded
	_, err = os.Stat(filePath)
	test.AssertNoError(t, err)

	// Load and verify content
	loaded, err := Load(filePath)
	test.AssertNoError(t, err)

	test.AssertEqual(t, loaded.SourceLanguage, "en")
	test.AssertEqual(t, loaded.Version, "1.0")
	test.AssertEqual(t, len(loaded.Strings), 1)
}

func TestXCStrings_LoadEmptyLocalizationsInitialized(t *testing.T) {
	// Test that empty objects {} are loaded with initialized localizations map
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"%@": {},
			"test": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}}
				}
			}
		},
		"version": "1.0"
	}`

	tmpFile := test.TempFile(t, "empty_localizations.xcstrings", testContent)

	// Load the file
	xcstrings, err := Load(tmpFile)
	test.AssertNoError(t, err)

	// Verify that empty localizations are initialized
	formatKey, exists := xcstrings.Strings["%@"]
	if !exists {
		t.Fatal("Expected '%%@' key to exist")
	}

	if formatKey.Localizations == nil {
		t.Error("Expected localizations to be initialized, but it was nil")
	}

	// Modify another key
	err = xcstrings.SetTranslation("test", "ja", "テスト")
	test.AssertNoError(t, err)

	// Save the file
	tmpOutput := test.TempFile(t, "output.xcstrings", "")
	err = xcstrings.SaveToFile(tmpOutput)
	test.AssertNoError(t, err)

	// Reload and verify localizations are not null
	reloaded, err := Load(tmpOutput)
	test.AssertNoError(t, err)

	reloadedFormatKey, exists := reloaded.Strings["%@"]
	if !exists {
		t.Fatal("Expected '%%@' key to exist after reload")
	}

	if reloadedFormatKey.Localizations == nil {
		t.Error("Expected localizations to remain initialized after save/load, but it was nil")
	}

	// Verify it's an empty map, not nil
	test.AssertEqual(t, len(reloadedFormatKey.Localizations), 0)
}

func TestXCStrings_SaveToFile_AtomicWrite(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Value"}},
				},
			},
		},
		Version: "1.0",
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "atomic.xcstrings")

	err := xcstrings.SaveToFile(filePath)
	test.AssertNoError(t, err)

	// Verify no temp files remain in the directory
	entries, err := os.ReadDir(tmpDir)
	test.AssertNoError(t, err)

	if len(entries) != 1 {
		t.Errorf("expected exactly 1 file in dir, got %d", len(entries))
	}
	if entries[0].Name() != "atomic.xcstrings" {
		t.Errorf("expected file name atomic.xcstrings, got %s", entries[0].Name())
	}

	// Verify file content is correct after atomic write
	loaded, err := Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, loaded.SourceLanguage, "en")
	test.AssertEqual(t, loaded.Version, "1.0")
	test.AssertEqual(t, len(loaded.Strings), 1)
}

func TestXCStrings_SaveToFile_InvalidDirectory(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings:        map[string]StringDefinition{},
		Version:        "1.0",
	}

	err := xcstrings.SaveToFile("/nonexistent/directory/file.xcstrings")
	test.AssertError(t, err)
}

func TestXCStrings_SaveToFile_OverwriteExisting(t *testing.T) {
	original := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"old_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Old"}},
				},
			},
		},
		Version: "1.0",
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "overwrite.xcstrings")

	err := original.SaveToFile(filePath)
	test.AssertNoError(t, err)

	// Overwrite with new content
	updated := &XCStrings{
		SourceLanguage: "ja",
		Strings: map[string]StringDefinition{
			"new_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "New"}},
				},
			},
		},
		Version: "2.0",
	}

	err = updated.SaveToFile(filePath)
	test.AssertNoError(t, err)

	// Verify overwritten content
	loaded, err := Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, loaded.SourceLanguage, "ja")
	test.AssertEqual(t, loaded.Version, "2.0")
	test.AssertEqual(t, len(loaded.Strings), 1)

	// Verify no temp files remain
	entries, err := os.ReadDir(tmpDir)
	test.AssertNoError(t, err)
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 file after overwrite, got %d", len(entries))
	}
}

func TestXCStrings_GetTranslatedKeys(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"translated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "翻訳済み"}},
				},
			},
			"untranslated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "new", Value: "未翻訳"}},
				},
			},
			"missing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Missing"}},
				},
			},
		},
	}

	got := xcstrings.TranslatedKeys("ja")
	want := []string{"translated_key"}

	sort.Strings(got)
	sort.Strings(want)
	test.AssertSliceEqual(t, got, want)
}

// normalizeJSON re-marshals JSON to produce a canonical form for comparison.
func normalizeJSON(t *testing.T, data []byte) []byte {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("failed to unmarshal JSON for normalization: %v", err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON for normalization: %v", err)
	}
	return out
}

// assertRoundTrip loads a fixture, saves it, and verifies the output matches.
func assertRoundTrip(t *testing.T, fixtureName string) {
	t.Helper()

	fixturePath := test.FixturePath(fixtureName)

	// Read original fixture
	originalData, err := os.ReadFile(fixturePath)
	test.AssertNoError(t, err)

	// Load the fixture
	xcstringsData, err := Load(fixturePath)
	test.AssertNoError(t, err)

	// Save to a temp file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, fixtureName)
	err = xcstringsData.SaveToFile(outputPath)
	test.AssertNoError(t, err)

	// Read the saved output
	savedData, err := os.ReadFile(outputPath)
	test.AssertNoError(t, err)

	// Normalize both and compare
	normalizedOriginal := normalizeJSON(t, originalData)
	normalizedSaved := normalizeJSON(t, savedData)

	if string(normalizedOriginal) != string(normalizedSaved) {
		t.Errorf("round-trip mismatch for %s\noriginal:\n%s\nsaved:\n%s",
			fixtureName, string(normalizedOriginal), string(normalizedSaved))
	}

	// Verify the saved file can be loaded again
	reloaded, err := Load(outputPath)
	test.AssertNoError(t, err)
	if reloaded == nil {
		t.Fatal("reloaded xcstrings should not be nil")
	}
}

func TestRoundTrip_PluralVariations(t *testing.T) {
	assertRoundTrip(t, "plural_variations.xcstrings")

	// Also verify the structure was parsed correctly
	fixturePath := test.FixturePath("plural_variations.xcstrings")
	xcstringsData, err := Load(fixturePath)
	test.AssertNoError(t, err)

	def := xcstringsData.Strings["item_count"]
	enLoc := def.Localizations["en"]

	if enLoc.StringUnit != nil {
		t.Error("expected StringUnit to be nil for variation-based localization")
	}
	if enLoc.Variations == nil {
		t.Fatal("expected Variations to be non-nil")
	}
	if enLoc.Variations.Plural == nil {
		t.Fatal("expected Plural variations to be non-nil")
	}
	if len(enLoc.Variations.Plural) != 2 {
		t.Errorf("expected 2 plural categories, got %d", len(enLoc.Variations.Plural))
	}

	oneVariation := enLoc.Variations.Plural["one"]
	if oneVariation == nil || oneVariation.StringUnit == nil {
		t.Fatal("expected 'one' plural variation with stringUnit")
	}
	test.AssertEqual(t, oneVariation.StringUnit.Value, "%lld item")

	otherVariation := enLoc.Variations.Plural["other"]
	if otherVariation == nil || otherVariation.StringUnit == nil {
		t.Fatal("expected 'other' plural variation with stringUnit")
	}
	test.AssertEqual(t, otherVariation.StringUnit.Value, "%lld items")
}

func TestRoundTrip_DeviceVariations(t *testing.T) {
	assertRoundTrip(t, "device_variations.xcstrings")

	fixturePath := test.FixturePath("device_variations.xcstrings")
	xcstringsData, err := Load(fixturePath)
	test.AssertNoError(t, err)

	def := xcstringsData.Strings["welcome_message"]
	enLoc := def.Localizations["en"]

	if enLoc.Variations == nil {
		t.Fatal("expected Variations to be non-nil")
	}
	if enLoc.Variations.Device == nil {
		t.Fatal("expected Device variations to be non-nil")
	}
	if len(enLoc.Variations.Device) != 3 {
		t.Errorf("expected 3 device categories, got %d", len(enLoc.Variations.Device))
	}

	iphoneVariation := enLoc.Variations.Device["iphone"]
	if iphoneVariation == nil || iphoneVariation.StringUnit == nil {
		t.Fatal("expected 'iphone' device variation with stringUnit")
	}
	test.AssertEqual(t, iphoneVariation.StringUnit.Value, "Welcome to our iPhone app")
}

func TestRoundTrip_NestedVariations(t *testing.T) {
	assertRoundTrip(t, "nested_variations.xcstrings")

	fixturePath := test.FixturePath("nested_variations.xcstrings")
	xcstringsData, err := Load(fixturePath)
	test.AssertNoError(t, err)

	def := xcstringsData.Strings["photo_count"]
	enLoc := def.Localizations["en"]

	if enLoc.Variations == nil || enLoc.Variations.Device == nil {
		t.Fatal("expected device variations")
	}

	iphoneVariation := enLoc.Variations.Device["iphone"]
	if iphoneVariation == nil {
		t.Fatal("expected 'iphone' device variation")
	}
	if iphoneVariation.StringUnit != nil {
		t.Error("expected StringUnit to be nil for nested variation")
	}
	if iphoneVariation.Variations == nil || iphoneVariation.Variations.Plural == nil {
		t.Fatal("expected nested plural variations under iphone")
	}

	oneVariation := iphoneVariation.Variations.Plural["one"]
	if oneVariation == nil || oneVariation.StringUnit == nil {
		t.Fatal("expected 'one' plural variation under iphone")
	}
	test.AssertEqual(t, oneVariation.StringUnit.Value, "%lld photo on your iPhone")
}

func TestRoundTrip_Substitutions(t *testing.T) {
	assertRoundTrip(t, "substitutions.xcstrings")

	fixturePath := test.FixturePath("substitutions.xcstrings")
	xcstringsData, err := Load(fixturePath)
	test.AssertNoError(t, err)

	def := xcstringsData.Strings["upload_progress"]
	enLoc := def.Localizations["en"]

	if enLoc.StringUnit == nil {
		t.Fatal("expected StringUnit to be non-nil for substitution-based localization")
	}
	test.AssertEqual(t, enLoc.StringUnit.Value, "Uploading %#@file_count@ to %#@folder@")

	if enLoc.Substitutions == nil {
		t.Fatal("expected Substitutions to be non-nil")
	}
	if len(enLoc.Substitutions) != 2 {
		t.Errorf("expected 2 substitutions, got %d", len(enLoc.Substitutions))
	}

	fileCount := enLoc.Substitutions["file_count"]
	test.AssertEqual(t, fileCount.ArgNum, 1)
	test.AssertEqual(t, fileCount.FormatSpecifier, "lld")
	if fileCount.Variations.Plural == nil {
		t.Fatal("expected plural variations in file_count substitution")
	}

	oneVariation := fileCount.Variations.Plural["one"]
	if oneVariation == nil || oneVariation.StringUnit == nil {
		t.Fatal("expected 'one' variation in file_count substitution")
	}
	test.AssertEqual(t, oneVariation.StringUnit.Value, "%arg file")
}

func TestRoundTrip_SimpleStringUnit(t *testing.T) {
	assertRoundTrip(t, "simple.xcstrings")
}
