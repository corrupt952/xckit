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

func TestXCStrings_NeedsReviewKeys(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"translated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "翻訳済み"}},
				},
			},
			"needs_review_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "needs_review", Value: "レビュー必要"}},
				},
			},
			"untranslated_key": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "new", Value: ""}},
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
					"ja": {StringUnit: &StringUnit{State: "needs_review", Value: "翻訳不要"}},
				},
			},
		},
	}

	t.Run("returns only needs_review keys", func(t *testing.T) {
		got := xcstrings.NeedsReviewKeys("ja")
		want := []string{"needs_review_key"}

		sort.Strings(got)
		sort.Strings(want)
		test.AssertSliceEqual(t, got, want)
	})

	t.Run("needs_review keys are included in untranslated", func(t *testing.T) {
		untranslated := xcstrings.UntranslatedKeys("ja")
		sort.Strings(untranslated)

		// needs_review_key should be in untranslated since state != "translated"
		found := false
		for _, key := range untranslated {
			if key == "needs_review_key" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected needs_review_key to appear in UntranslatedKeys result")
		}
	})

	t.Run("no needs_review keys for language without any", func(t *testing.T) {
		got := xcstrings.NeedsReviewKeys("fr")
		if len(got) != 0 {
			t.Errorf("expected 0 needs_review keys for fr, got %d", len(got))
		}
	})

	t.Run("skips shouldTranslate=false", func(t *testing.T) {
		got := xcstrings.NeedsReviewKeys("ja")
		for _, key := range got {
			if key == "should_not_translate" {
				t.Error("should_not_translate key should be excluded from NeedsReviewKeys")
			}
		}
	})
}

func TestXCStrings_StaleKeys(t *testing.T) {
	xcstrings := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"active_key": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Active"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "アクティブ"}},
				},
			},
			"stale_key": {
				ExtractionState: "stale",
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Stale"}},
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "古い"}},
				},
			},
			"another_stale": {
				ExtractionState: "stale",
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Another"}},
				},
			},
			"manual_key": {
				ExtractionState: "manual",
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Manual"}},
				},
			},
		},
	}

	t.Run("StaleKeys returns only stale keys", func(t *testing.T) {
		got := xcstrings.StaleKeys()
		sort.Strings(got)
		want := []string{"another_stale", "stale_key"}
		test.AssertSliceEqual(t, got, want)
	})

	t.Run("ActiveKeys returns non-stale keys", func(t *testing.T) {
		got := xcstrings.ActiveKeys()
		sort.Strings(got)
		want := []string{"active_key", "manual_key"}
		test.AssertSliceEqual(t, got, want)
	})

	t.Run("IsStale returns true for stale keys", func(t *testing.T) {
		test.AssertEqual(t, xcstrings.IsStale("stale_key"), true)
		test.AssertEqual(t, xcstrings.IsStale("active_key"), false)
		test.AssertEqual(t, xcstrings.IsStale("nonexistent"), false)
	})

	t.Run("UntranslatedKeys excludes stale keys", func(t *testing.T) {
		// another_stale is missing ja but should not appear since it's stale
		got := xcstrings.UntranslatedKeys("ja")
		sort.Strings(got)
		want := []string{"manual_key"}
		test.AssertSliceEqual(t, got, want)
	})

	t.Run("KeysWithAnyUntranslated excludes stale keys", func(t *testing.T) {
		got := xcstrings.KeysWithAnyUntranslated()
		sort.Strings(got)
		// manual_key is missing ja, so it should appear
		// another_stale is missing ja but is stale, so it should NOT appear
		want := []string{"manual_key"}
		test.AssertSliceEqual(t, got, want)
	})

	t.Run("RemoveStaleKeys removes stale and returns count", func(t *testing.T) {
		// Create a copy to avoid mutating the shared test struct
		x := &XCStrings{
			SourceLanguage: "en",
			Strings: map[string]StringDefinition{
				"keep":   {Localizations: map[string]Localization{}},
				"remove": {ExtractionState: "stale", Localizations: map[string]Localization{}},
			},
		}
		count := x.RemoveStaleKeys()
		test.AssertEqual(t, count, 1)
		test.AssertEqual(t, len(x.Strings), 1)
		if _, exists := x.Strings["remove"]; exists {
			t.Error("stale key should have been removed")
		}
	})
}

func TestLocalization_AllStringUnits(t *testing.T) {
	t.Run("top-level StringUnit only", func(t *testing.T) {
		loc := Localization{
			StringUnit: &StringUnit{State: "translated", Value: "Hello"},
		}
		units := loc.AllStringUnits()
		if len(units) != 1 {
			t.Fatalf("expected 1 unit, got %d", len(units))
		}
		test.AssertEqual(t, units[0].Value, "Hello")
	})

	t.Run("plural variations", func(t *testing.T) {
		loc := Localization{
			Variations: &Variations{
				Plural: map[PluralCategory]*VariationValue{
					"one":   {StringUnit: &StringUnit{State: "translated", Value: "1 item"}},
					"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld items"}},
				},
			},
		}
		units := loc.AllStringUnits()
		if len(units) != 2 {
			t.Fatalf("expected 2 units, got %d", len(units))
		}
	})

	t.Run("substitutions", func(t *testing.T) {
		loc := Localization{
			StringUnit: &StringUnit{State: "translated", Value: "%#@files@"},
			Substitutions: map[string]Substitution{
				"files": {
					ArgNum:          1,
					FormatSpecifier: "lld",
					Variations: Variations{
						Plural: map[PluralCategory]*VariationValue{
							"one":   {StringUnit: &StringUnit{State: "translated", Value: "%arg file"}},
							"other": {StringUnit: &StringUnit{State: "translated", Value: "%arg files"}},
						},
					},
				},
			},
		}
		units := loc.AllStringUnits()
		// 1 top-level + 2 from substitution
		if len(units) != 3 {
			t.Fatalf("expected 3 units, got %d", len(units))
		}
	})

	t.Run("empty localization", func(t *testing.T) {
		loc := Localization{}
		units := loc.AllStringUnits()
		if len(units) != 0 {
			t.Fatalf("expected 0 units, got %d", len(units))
		}
	})
}

func TestUntranslatedKeys_WithVariations(t *testing.T) {
	xc := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"all_translated_plural": {
				Localizations: map[string]Localization{
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
			"partial_plural": {
				Localizations: map[string]Localization{
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"one":   {StringUnit: &StringUnit{State: "new", Value: ""}},
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
			"simple_translated": {
				Localizations: map[string]Localization{
					"ja": {StringUnit: &StringUnit{State: "translated", Value: "翻訳済み"}},
				},
			},
		},
	}

	got := xc.UntranslatedKeys("ja")
	sort.Strings(got)
	want := []string{"partial_plural"}
	test.AssertSliceEqual(t, got, want)
}

func TestTranslatedKeys_WithVariations(t *testing.T) {
	xc := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"all_translated_plural": {
				Localizations: map[string]Localization{
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
			"partial_plural": {
				Localizations: map[string]Localization{
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"one":   {StringUnit: &StringUnit{State: "new", Value: ""}},
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
		},
	}

	got := xc.TranslatedKeys("ja")
	sort.Strings(got)
	want := []string{"all_translated_plural"}
	test.AssertSliceEqual(t, got, want)
}

func TestKeysWithAnyUntranslated_WithVariations(t *testing.T) {
	xc := &XCStrings{
		SourceLanguage: "en",
		Strings: map[string]StringDefinition{
			"all_translated": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Hello"}},
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
			"partial_variation": {
				Localizations: map[string]Localization{
					"en": {StringUnit: &StringUnit{State: "translated", Value: "Hello"}},
					"ja": {
						Variations: &Variations{
							Plural: map[PluralCategory]*VariationValue{
								"one":   {StringUnit: &StringUnit{State: "new", Value: ""}},
								"other": {StringUnit: &StringUnit{State: "translated", Value: "%lld個"}},
							},
						},
					},
				},
			},
		},
	}

	got := xc.KeysWithAnyUntranslated()
	sort.Strings(got)
	want := []string{"partial_variation"}
	test.AssertSliceEqual(t, got, want)
}

func TestUntranslatedKeys_WithPluralVariationsFixture(t *testing.T) {
	xc, err := Load("../fixtures/plural_variations.xcstrings")
	test.AssertNoError(t, err)

	// All variations in the fixture are "translated", so no untranslated keys expected
	got := xc.UntranslatedKeys("ja")
	if len(got) != 0 {
		t.Errorf("expected 0 untranslated keys for ja, got %d: %v", len(got), got)
	}

	got = xc.UntranslatedKeys("en")
	if len(got) != 0 {
		t.Errorf("expected 0 untranslated keys for en, got %d: %v", len(got), got)
	}

	// TranslatedKeys should return the key for both languages
	translated := xc.TranslatedKeys("ja")
	sort.Strings(translated)
	want := []string{"%lld items"}
	test.AssertSliceEqual(t, translated, want)

	translated = xc.TranslatedKeys("en")
	sort.Strings(translated)
	test.AssertSliceEqual(t, translated, want)
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

func TestRoundTrip_Variations(t *testing.T) {
	fixtures := []struct {
		name string
		path string
	}{
		{"plural_variations", "../fixtures/plural_variations.xcstrings"},
		{"device_variations", "../fixtures/device_variations.xcstrings"},
		{"nested_variations", "../fixtures/nested_variations.xcstrings"},
		{"substitutions", "../fixtures/substitutions.xcstrings"},
		{"simple", "../fixtures/simple.xcstrings"},
	}

	for _, tt := range fixtures {
		t.Run(tt.name, func(t *testing.T) {
			// Load original
			original, err := Load(tt.path)
			test.AssertNoError(t, err)

			// Save to temp file
			tmpFile := filepath.Join(t.TempDir(), "roundtrip.xcstrings")
			err = original.SaveToFile(tmpFile)
			test.AssertNoError(t, err)

			// Load saved file
			reloaded, err := Load(tmpFile)
			test.AssertNoError(t, err)

			// Save again and compare bytes
			tmpFile2 := filepath.Join(t.TempDir(), "roundtrip2.xcstrings")
			err = reloaded.SaveToFile(tmpFile2)
			test.AssertNoError(t, err)

			data1, err := os.ReadFile(tmpFile)
			test.AssertNoError(t, err)
			data2, err := os.ReadFile(tmpFile2)
			test.AssertNoError(t, err)

			if string(data1) != string(data2) {
				t.Errorf("round-trip produced different output:\nfirst save:\n%s\nsecond save:\n%s", string(data1), string(data2))
			}
		})
	}
}

func TestLoad_PluralVariations(t *testing.T) {
	xc, err := Load("../fixtures/plural_variations.xcstrings")
	test.AssertNoError(t, err)

	def, exists := xc.Strings["%lld items"]
	if !exists {
		t.Fatal("expected key '%lld items' to exist")
	}

	loc := def.Localizations["en"]
	if loc.StringUnit != nil {
		t.Error("expected StringUnit to be nil for variation-only localization")
	}
	if loc.Variations == nil {
		t.Fatal("expected Variations to be non-nil")
	}
	if loc.Variations.Plural == nil {
		t.Fatal("expected Plural variations to be non-nil")
	}
	if one, ok := loc.Variations.Plural["one"]; !ok || one.StringUnit == nil {
		t.Error("expected plural 'one' variation with StringUnit")
	} else {
		test.AssertEqual(t, one.StringUnit.Value, "%lld item")
	}
}

func TestLoad_DeviceVariations(t *testing.T) {
	xc, err := Load("../fixtures/device_variations.xcstrings")
	test.AssertNoError(t, err)

	def := xc.Strings["welcome_message"]
	loc := def.Localizations["en"]
	if loc.Variations == nil || loc.Variations.Device == nil {
		t.Fatal("expected Device variations to be non-nil")
	}
	iphone := loc.Variations.Device["iphone"]
	if iphone == nil || iphone.StringUnit == nil {
		t.Fatal("expected iphone variation with StringUnit")
	}
	test.AssertEqual(t, iphone.StringUnit.Value, "Welcome to our iPhone app!")
}

func TestLoad_NestedVariations(t *testing.T) {
	xc, err := Load("../fixtures/nested_variations.xcstrings")
	test.AssertNoError(t, err)

	def := xc.Strings["%lld photos"]
	loc := def.Localizations["en"]
	if loc.Variations == nil || loc.Variations.Device == nil {
		t.Fatal("expected Device variations")
	}
	iphone := loc.Variations.Device["iphone"]
	if iphone == nil || iphone.Variations == nil || iphone.Variations.Plural == nil {
		t.Fatal("expected nested plural variations under iphone device")
	}
	one := iphone.Variations.Plural["one"]
	if one == nil || one.StringUnit == nil {
		t.Fatal("expected 'one' plural under iphone")
	}
	test.AssertEqual(t, one.StringUnit.Value, "%lld photo on iPhone")
}

func TestLoad_Substitutions(t *testing.T) {
	xc, err := Load("../fixtures/substitutions.xcstrings")
	test.AssertNoError(t, err)

	def := xc.Strings["%lld files in %lld folders"]
	loc := def.Localizations["en"]
	if loc.StringUnit == nil {
		t.Fatal("expected StringUnit for substitution-based localization")
	}
	test.AssertEqual(t, loc.StringUnit.Value, "%#@files@ in %#@folders@")

	if loc.Substitutions == nil {
		t.Fatal("expected Substitutions to be non-nil")
	}
	files, ok := loc.Substitutions["files"]
	if !ok {
		t.Fatal("expected 'files' substitution")
	}
	test.AssertEqual(t, files.ArgNum, 1)
	test.AssertEqual(t, files.FormatSpecifier, "lld")
	if files.Variations.Plural == nil {
		t.Fatal("expected plural variations in files substitution")
	}
	one := files.Variations.Plural["one"]
	if one == nil || one.StringUnit == nil {
		t.Fatal("expected 'one' plural in files substitution")
	}
	test.AssertEqual(t, one.StringUnit.Value, "%arg file")
}
