package command

import (
	"context"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func TestStaleCommand_Execute_ListStale(t *testing.T) {
	filePath := test.FixturePath("stale.xcstrings")

	cmd := &StaleCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedContent := []string{
		"Stale keys (2):",
		"stale_key",
		"another_stale_key",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestStaleCommand_Execute_NoStale(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"key1": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 1"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &StaleCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "No stale keys found") {
		t.Errorf("output should contain 'No stale keys found', got: %q", output)
	}
}

func TestStaleCommand_Execute_DryRun(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"active": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Active"}}
				}
			},
			"old_key": {
				"extractionState": "stale",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Old"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &StaleCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--remove", "--dry-run"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Would remove 1 stale key(s)") {
		t.Errorf("output should contain dry-run message, got: %q", output)
	}
	if !strings.Contains(output, "old_key") {
		t.Errorf("output should list old_key, got: %q", output)
	}

	// Verify file was not modified
	loaded, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if len(loaded.Strings) != 2 {
		t.Errorf("expected 2 keys (file should not be modified), got %d", len(loaded.Strings))
	}
}

func TestStaleCommand_Execute_Remove(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"active": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Active"}}
				}
			},
			"old_key": {
				"extractionState": "stale",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Old"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &StaleCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--remove"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Removed 1 stale key(s)") {
		t.Errorf("output should contain removal message, got: %q", output)
	}

	// Verify file was modified
	loaded, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if len(loaded.Strings) != 1 {
		t.Errorf("expected 1 key after removal, got %d", len(loaded.Strings))
	}
	if _, exists := loaded.Strings["old_key"]; exists {
		t.Error("old_key should have been removed")
	}
}

func TestStaleCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &StaleCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}
