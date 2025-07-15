package command

import (
	"context"
	"flag"
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
			expectedKeys:   []string{"untranslated_key"},
			expectedStatus: 0,
		},
		{
			name:           "untranslated for all languages",
			args:           []string{},
			expectedKeys:   []string{"untranslated_key"},
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
