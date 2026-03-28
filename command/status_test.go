package command

import (
	"context"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
)

func TestStatusCommand_Execute(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"key1": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 1"}},
					"ja": {"stringUnit": {"state": "translated", "value": "キー1"}}
				}
			},
			"key2": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 2"}},
					"ja": {"stringUnit": {"state": "new", "value": "キー2"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &StatusCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedContent := []string{
		"Translation Status",
		"Source Language: en",
		"Total Keys: 2",
		"Stale Keys: 0",
		"Active Keys: 2",
		"Languages:",
		"en",
		"ja",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestStatusCommand_Execute_WithStaleKeys(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"active_key": {
				"extractionState": "manual",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Active"}},
					"ja": {"stringUnit": {"state": "translated", "value": "アクティブ"}}
				}
			},
			"stale_key": {
				"extractionState": "stale",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Stale"}},
					"ja": {"stringUnit": {"state": "translated", "value": "古い"}}
				}
			},
			"untranslated_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Untranslated"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &StatusCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedContent := []string{
		"Total Keys: 3",
		"Stale Keys: 1",
		"Active Keys: 2",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}

	// Progress should be based on active keys (2), not total keys (3)
	// ja: 1 translated out of 2 active = 50.0%
	if !strings.Contains(output, "1/2 translated (50.0%)") {
		t.Errorf("output should show progress based on active keys, got: %q", output)
	}
}

func TestStatusCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &StatusCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1) // ExitFailure
}
