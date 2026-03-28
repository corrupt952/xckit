package formatter

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestDisplayKeyDetails(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"hello": {
				Comment:         "A greeting",
				ExtractionState: "manual",
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "こんにちは"}},
					"es": {StringUnit: &xcstrings.StringUnit{State: "new", Value: "Hola"}},
				},
			},
			"goodbye": {
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Goodbye"}},
					"ja": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: ""}}, // Empty value
				},
			},
			"missing_translations": {
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Missing"}},
					// ja is missing
				},
			},
		},
	}

	tests := []struct {
		name           string
		keys           []string
		expectedOutput []string // Patterns to check in output
	}{
		{
			name: "single key with all translations",
			keys: []string{"hello"},
			expectedOutput: []string{
				"hello:",
				"new - Hola",
				"translated - こんにちは",
			},
		},
		{
			name: "key with empty value",
			keys: []string{"goodbye"},
			expectedOutput: []string{
				"goodbye:",
				"translated - (empty)",
			},
		},
		{
			name: "key with missing translation",
			keys: []string{"missing_translations"},
			expectedOutput: []string{
				"missing_translations:",
				"ja: missing",
			},
		},
		{
			name: "multiple keys",
			keys: []string{"hello", "goodbye"},
			expectedOutput: []string{
				"hello:",
				"goodbye:",
				"new - Hola",
				"translated - こんにちは",
				"translated - (empty)",
			},
		},
		{
			name:           "empty keys",
			keys:           []string{},
			expectedOutput: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				DisplayKeyDetails(xcstringsData, tt.keys)
			})

			// Check that all expected patterns are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}

			// For empty keys, output should be empty
			if len(tt.keys) == 0 && strings.TrimSpace(output) != "" {
				t.Errorf("expected empty output for empty keys, got: %q", output)
			}
		})
	}
}

func TestDisplayKeyDetails_OutputFormat(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"test_key": {
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Test"}},
					"ja": {StringUnit: &xcstrings.StringUnit{State: "new", Value: "テスト"}},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"test_key"})
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Check basic structure (1 key line + at least 1 language line + at least 1 value line)
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines of output, got %d", len(lines))
	}

	// First line should be key name
	if !strings.Contains(lines[0], "test_key:") {
		t.Errorf("first line should contain key name, got: %q", lines[0])
	}

	// Check that language lines and value lines are present and indented
	hasLangLine := false
	hasValueLine := false
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line should be indented, got: %q", line)
		}
		if strings.HasPrefix(line, "  ") && strings.Contains(line, ":") && !strings.Contains(line, " - ") {
			hasLangLine = true
		}
		if strings.Contains(line, " - ") {
			hasValueLine = true
		}
	}
	if !hasLangLine {
		t.Error("expected at least one language line")
	}
	if !hasValueLine {
		t.Error("expected at least one value line with state and value")
	}
}

func TestDisplayKeyDetails_LanguageSorting(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"sort_test": {
				Localizations: map[string]xcstrings.Localization{
					"zh": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "中文"}},
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "日本語"}},
					"es": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Español"}},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"sort_test"})
	})

	// Languages should appear in alphabetical order: es, ja, zh (en excluded as source language)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var languageOrder []string
	for _, line := range lines[1:] { // Skip the key name line
		// Language lines are indented with 2 spaces and end with ":"
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			lang := strings.TrimSuffix(trimmed, ":")
			languageOrder = append(languageOrder, lang)
		}
	}

	expectedOrder := []string{"es", "ja", "zh"} // en is excluded as source language
	if len(languageOrder) != len(expectedOrder) {
		t.Errorf("expected %d languages, got %d", len(expectedOrder), len(languageOrder))
		return
	}
	test.AssertSliceEqual(t, languageOrder, expectedOrder)
}

func TestDisplayKeyDetails_Substitutions(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"%lld files in %lld folders": {
				Localizations: map[string]xcstrings.Localization{
					"en": {
						StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%#@files@ in %#@folders@"},
					},
					"ja": {
						StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%#@files@が%#@folders@にあります"},
						Substitutions: map[string]xcstrings.Substitution{
							"files": {
								ArgNum:          1,
								FormatSpecifier: "lld",
								Variations: xcstrings.Variations{
									Plural: map[string]*xcstrings.VariationValue{
										"other": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%argファイル"}},
									},
								},
							},
							"folders": {
								ArgNum:          2,
								FormatSpecifier: "lld",
								Variations: xcstrings.Variations{
									Plural: map[string]*xcstrings.VariationValue{
										"other": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%argフォルダ"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"%lld files in %lld folders"})
	})

	expectedPatterns := []string{
		"%lld files in %lld folders:",
		"ja:",
		"translated - %#@files@が%#@folders@にあります",
		"substitutions.files:",
		"plural.other: translated - %argファイル",
		"substitutions.folders:",
		"plural.other: translated - %argフォルダ",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got:\n%s", pattern, output)
		}
	}

	// Verify substitution names are sorted (files before folders)
	filesIdx := strings.Index(output, "substitutions.files:")
	foldersIdx := strings.Index(output, "substitutions.folders:")
	if filesIdx >= foldersIdx {
		t.Errorf("expected substitutions.files before substitutions.folders in output")
	}
}

func TestDisplayKeyDetails_Variations(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"%lld items": {
				Localizations: map[string]xcstrings.Localization{
					"ja": {
						Variations: &xcstrings.Variations{
							Plural: map[string]*xcstrings.VariationValue{
								"one":   {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%lldアイテム"}},
								"other": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "%lldアイテム"}},
							},
						},
					},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"%lld items"})
	})

	expectedPatterns := []string{
		"%lld items:",
		"ja:",
		"plural.one: translated - %lldアイテム",
		"plural.other: translated - %lldアイテム",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got:\n%s", pattern, output)
		}
	}
}

func TestDisplayKeyDetails_SubstitutionsFromFixture(t *testing.T) {
	xc, err := xcstrings.Load("../fixtures/substitutions.xcstrings")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xc, []string{"%lld files in %lld folders"})
	})

	// The fixture only has "en" which is the source language, so Languages() returns empty.
	// This tests that the function handles it without panicking.
	if strings.Contains(output, "substitutions.files:") {
		// If en is rendered (shouldn't be since it's source language),
		// verify substitution content is correct.
		expectedPatterns := []string{
			"substitutions.files:",
			"plural.one: translated - %arg file",
			"plural.other: translated - %arg files",
			"substitutions.folders:",
			"plural.one: translated - %arg folder",
			"plural.other: translated - %arg folders",
		}
		for _, pattern := range expectedPatterns {
			if !strings.Contains(output, pattern) {
				t.Errorf("expected output to contain %q, got:\n%s", pattern, output)
			}
		}
	}
}
