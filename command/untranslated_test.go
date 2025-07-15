package command

import (
	"context"
	"flag"
	"path/filepath"
	"strings"
	"testing"

	"xckit/helper/test"
)

func TestUntranslatedCommand_Execute(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"translated_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Translated"}},
					"ja": {"stringUnit": {"state": "translated", "value": "翻訳済み"}}
				}
			},
			"untranslated_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Untranslated"}},
					"ja": {"stringUnit": {"state": "new", "value": "未翻訳"}}
				}
			},
			"error.network": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Network Error"}}
				}
			},
			"error.timeout": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Timeout Error"}},
					"ja": {"stringUnit": {"state": "new", "value": ""}}
				}
			}
		},
		"version": "1.0"
	}`

	tests := []struct {
		name           string
		args           []string
		expectedKeys   []string
		expectedStatus int
	}{
		{
			name:           "untranslated for specific language",
			args:           []string{"--lang", "ja"},
			expectedKeys:   []string{"untranslated_key", "error.network", "error.timeout"},
			expectedStatus: 0,
		},
		{
			name:           "untranslated for all languages",
			args:           []string{},
			expectedKeys:   []string{"untranslated_key", "error.network", "error.timeout"},
			expectedStatus: 0,
		},
		{
			name:           "untranslated with prefix",
			args:           []string{"--prefix", "error"},
			expectedKeys:   []string{"error.network", "error.timeout"},
			expectedStatus: 0,
		},
		{
			name:           "untranslated with prefix and language",
			args:           []string{"--lang", "ja", "--prefix", "error"},
			expectedKeys:   []string{"error.network", "error.timeout"},
			expectedStatus: 0,
		},
		{
			name:           "untranslated with non-matching prefix",
			args:           []string{"--prefix", "login"},
			expectedKeys:   []string{},
			expectedStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test.TempFile(t, "test.xcstrings", testContent)

			cmd := &UntranslatedCommand{}

			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)
			args := append([]string{"-f", filePath}, tt.args...)
			err := flagSet.Parse(args)
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), tt.expectedStatus)
			})

			if len(tt.expectedKeys) == 0 {
				if strings.Contains(output, "No untranslated keys found") || strings.Contains(output, "All keys are translated") {
					// Expected behavior for non-matching prefix or all translated
					return
				}
			}

			for _, expectedKey := range tt.expectedKeys {
				if !strings.Contains(output, expectedKey) {
					t.Errorf("output should contain %q, got: %q", expectedKey, output)
				}
			}
		})
	}
}

func TestUntranslatedCommand_Execute_AllTranslated(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"key1": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 1"}},
					"ja": {"stringUnit": {"state": "translated", "value": "キー1"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &UntranslatedCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "All keys are translated") {
		t.Errorf("output should indicate all keys are translated, got: %q", output)
	}
}

func TestUntranslatedCommand_Execute_AllTranslated_NoLang(t *testing.T) {
	// Test case with all languages translated, no --lang specified
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}},
					"es": {"stringUnit": {"state": "translated", "value": "Hola"}}
				}
			},
			"farewell": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Goodbye"}},
					"ja": {"stringUnit": {"state": "translated", "value": "さようなら"}},
					"es": {"stringUnit": {"state": "translated", "value": "Adiós"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &UntranslatedCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath}) // No --lang
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "All keys are fully translated in all languages") {
		t.Errorf("output should indicate all keys are fully translated, got: %q", output)
	}
}

func TestUntranslatedCommand_Execute_PartiallyTranslated_NoLang(t *testing.T) {
	// Test case with some untranslated, no --lang specified
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}},
					"es": {"stringUnit": {"state": "translated", "value": "Hola"}}
				}
			},
			"farewell": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Goodbye"}},
					"ja": {"stringUnit": {"state": "new", "value": ""}}
				}
			},
			"welcome": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Welcome"}},
					"es": {"stringUnit": {"state": "translated", "value": "Bienvenido"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &UntranslatedCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath}) // No --lang
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	// Should show untranslated keys
	if !strings.Contains(output, "farewell") {
		t.Errorf("output should contain 'farewell' key, got: %q", output)
	}
	if !strings.Contains(output, "welcome") {
		t.Errorf("output should contain 'welcome' key, got: %q", output)
	}
	// Should NOT show fully translated key
	if strings.Contains(output, "greeting:") {
		t.Errorf("output should not contain 'greeting:' key, got: %q", output)
	}
}

func TestUntranslatedCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &UntranslatedCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1) // ExitFailure
}

func TestUntranslatedCommand_Execute_WithFixtures(t *testing.T) {
	tests := []struct {
		name             string
		fixture          string
		args             []string
		expectedOutput   string
		shouldContain    []string
		shouldNotContain []string
	}{
		// 全言語が翻訳済みのケース
		{
			name:           "all translated - no language specified",
			fixture:        "all_translated.xcstrings",
			args:           []string{},
			expectedOutput: "All keys are fully translated in all languages",
		},
		{
			name:           "all translated - ja specified",
			fixture:        "all_translated.xcstrings",
			args:           []string{"--lang", "ja"},
			expectedOutput: "All keys are translated for language 'ja'",
		},
		// 日本語だけ未翻訳のケース
		{
			name:          "ja only untranslated - no language specified",
			fixture:       "ja_only_untranslated.xcstrings",
			args:          []string{},
			shouldContain: []string{"greeting", "farewell"},
		},
		{
			name:          "ja only untranslated - ja specified",
			fixture:       "ja_only_untranslated.xcstrings",
			args:          []string{"--lang", "ja"},
			shouldContain: []string{"greeting", "farewell"},
		},
		{
			name:           "ja only untranslated - es specified",
			fixture:        "ja_only_untranslated.xcstrings",
			args:           []string{"--lang", "es"},
			expectedOutput: "All keys are translated for language 'es'",
		},
		// スペイン語だけ未翻訳のケース
		{
			name:          "es only untranslated - no language specified",
			fixture:       "es_only_untranslated.xcstrings",
			args:          []string{},
			shouldContain: []string{"greeting", "farewell"},
		},
		{
			name:          "es only untranslated - es specified",
			fixture:       "es_only_untranslated.xcstrings",
			args:          []string{"--lang", "es"},
			shouldContain: []string{"greeting", "farewell"},
		},
		{
			name:           "es only untranslated - ja specified",
			fixture:        "es_only_untranslated.xcstrings",
			args:           []string{"--lang", "ja"},
			expectedOutput: "All keys are translated for language 'ja'",
		},
		// 全言語未翻訳のケース
		{
			name:          "all untranslated - no language specified",
			fixture:       "all_untranslated.xcstrings",
			args:          []string{},
			shouldContain: []string{"greeting", "farewell"},
		},
		{
			name:          "all untranslated - ja specified",
			fixture:       "all_untranslated.xcstrings",
			args:          []string{"--lang", "ja"},
			shouldContain: []string{"greeting", "farewell"},
		},
		// 部分的に翻訳済みのケース
		{
			name:             "partially translated - no language specified",
			fixture:          "partially_translated.xcstrings",
			args:             []string{},
			shouldContain:    []string{"farewell", "welcome"},
			shouldNotContain: []string{"greeting:"},
		},
		// 複雑な混在状態のケース
		{
			name:             "mixed states - no language specified",
			fixture:          "mixed_states.xcstrings",
			args:             []string{},
			shouldContain:    []string{"ja_missing", "es_untranslated", "only_en"},
			shouldNotContain: []string{"fully_translated:"},
		},
		{
			name:             "mixed states - ja specified",
			fixture:          "mixed_states.xcstrings",
			args:             []string{"--lang", "ja"},
			shouldContain:    []string{"ja_missing", "only_en"},
			shouldNotContain: []string{"fully_translated:", "es_untranslated:"},
		},
		{
			name:             "mixed states - es specified",
			fixture:          "mixed_states.xcstrings",
			args:             []string{"--lang", "es"},
			shouldContain:    []string{"es_untranslated", "only_en"},
			shouldNotContain: []string{"fully_translated:", "ja_missing:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixturePath := filepath.Join("../fixtures", tt.fixture)

			cmd := &UntranslatedCommand{}
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)

			args := append([]string{"-f", fixturePath}, tt.args...)
			err := flagSet.Parse(args)
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), 0)
			})

			if tt.expectedOutput != "" {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("output should contain %q, got: %q", tt.expectedOutput, output)
				}
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("output should contain %q, got: %q", expected, output)
				}
			}

			for _, notExpected := range tt.shouldNotContain {
				if strings.Contains(output, notExpected) {
					t.Errorf("output should not contain %q, got: %q", notExpected, output)
				}
			}
		})
	}
}
