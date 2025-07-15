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
					"en": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "こんにちは"}},
					"es": {StringUnit: xcstrings.StringUnit{State: "new", Value: "Hola"}},
				},
			},
			"goodbye": {
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "Goodbye"}},
					"ja": {StringUnit: xcstrings.StringUnit{State: "translated", Value: ""}}, // Empty value
				},
			},
			"missing_translations": {
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "Missing"}},
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
				"en: translated - Hello",
				"es: new - Hola",
				"ja: translated - こんにちは",
			},
		},
		{
			name: "key with empty value",
			keys: []string{"goodbye"},
			expectedOutput: []string{
				"goodbye:",
				"en: translated - Goodbye",
				"ja: translated - (empty)",
			},
		},
		{
			name: "key with missing translation",
			keys: []string{"missing_translations"},
			expectedOutput: []string{
				"missing_translations:",
				"en: translated - Missing",
				"ja: missing",
			},
		},
		{
			name: "multiple keys",
			keys: []string{"hello", "goodbye"},
			expectedOutput: []string{
				"hello:",
				"goodbye:",
				"en: translated - Hello",
				"en: translated - Goodbye",
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
					"en": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "Test"}},
					"ja": {StringUnit: xcstrings.StringUnit{State: "new", Value: "テスト"}},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"test_key"})
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Check basic structure
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines of output, got %d", len(lines))
	}

	// First line should be key name
	if !strings.Contains(lines[0], "test_key:") {
		t.Errorf("first line should contain key name, got: %q", lines[0])
	}

	// Subsequent lines should be indented with language info
	for i := 1; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "  ") {
			t.Errorf("line %d should be indented, got: %q", i+1, lines[i])
		}

		// Should contain language code and state
		if !strings.Contains(lines[i], ":") || !strings.Contains(lines[i], " - ") {
			t.Errorf("line %d should contain language info format, got: %q", i+1, lines[i])
		}
	}
}

func TestDisplayKeyDetails_LanguageSorting(t *testing.T) {
	xcstringsData := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"sort_test": {
				Localizations: map[string]xcstrings.Localization{
					"zh": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "中文"}},
					"en": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "English"}},
					"ja": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "日本語"}},
					"es": {StringUnit: xcstrings.StringUnit{State: "translated", Value: "Español"}},
				},
			},
		},
	}

	output := captureOutput(func() {
		DisplayKeyDetails(xcstringsData, []string{"sort_test"})
	})

	// Languages should appear in alphabetical order: en, es, ja, zh
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var languageOrder []string
	for _, line := range lines[1:] { // Skip the key name line
		if strings.HasPrefix(line, "  ") {
			parts := strings.Split(strings.TrimSpace(line), ":")
			if len(parts) > 0 {
				languageOrder = append(languageOrder, parts[0])
			}
		}
	}

	expectedOrder := []string{"en", "es", "ja", "zh"}
	test.AssertSliceEqual(t, languageOrder, expectedOrder)
}
