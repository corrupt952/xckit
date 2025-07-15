package xcstrings

import (
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
					"en": {StringUnit: StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: StringUnit{State: "translated", Value: "こんにちは"}},
				},
			},
			"untranslated_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "Untranslated"}},
					"ja": {StringUnit: StringUnit{State: "new", Value: "未翻訳"}},
				},
			},
			"missing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "Missing"}},
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
					"en": {StringUnit: StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: StringUnit{State: "translated", Value: "こんにちは"}},
					"es": {StringUnit: StringUnit{State: "translated", Value: "Hola"}},
				},
			},
			"ja_untranslated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: StringUnit{State: "new", Value: ""}},
					"es": {StringUnit: StringUnit{State: "translated", Value: "Español"}},
				},
			},
			"es_missing": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "English only"}},
					"ja": {StringUnit: StringUnit{State: "translated", Value: "日本語"}},
					// es is missing - should be considered untranslated
				},
			},
			"only_en_translated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: StringUnit{State: "new", Value: ""}},
					// es is missing
				},
			},
			"all_untranslated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "new", Value: ""}},
					"ja": {StringUnit: StringUnit{State: "new", Value: ""}},
					"es": {StringUnit: StringUnit{State: "new", Value: ""}},
				},
			},
		},
	}

	got := xcstrings.KeysWithAnyUntranslated()
	// all_translated should NOT be in the list
	want := []string{"ja_untranslated", "es_missing", "only_en_translated", "all_untranslated"}

	sort.Strings(got)
	sort.Strings(want)
	test.AssertSliceEqual(t, got, want)
}

func TestXCStrings_SetTranslation(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"existing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "Existing"}},
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
					"en": {StringUnit: StringUnit{State: "translated", Value: "Test"}},
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

func TestXCStrings_GetTranslatedKeys(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"translated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: StringUnit{State: "translated", Value: "翻訳済み"}},
				},
			},
			"untranslated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: StringUnit{State: "new", Value: "未翻訳"}},
				},
			},
			"missing_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: StringUnit{State: "translated", Value: "Missing"}},
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
