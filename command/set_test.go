package command

import (
	"context"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func TestSetCommand_Execute(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "test_key", "テスト"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Successfully set translation") {
		t.Errorf("output should contain success message, got: %q", output)
	}

	// Verify the translation was actually set
	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	localization, exists := xcstringsData.Strings["test_key"].Localizations["ja"]
	if !exists {
		t.Error("Japanese translation should exist")
	} else {
		test.AssertEqual(t, localization.StringUnit.Value, "テスト")
		test.AssertEqual(t, localization.StringUnit.State, "translated")
	}
}

func TestSetCommand_Execute_MissingLanguage(t *testing.T) {
	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"test_key", "value"}) // Missing --lang
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 2) // ExitUsageError
}

func TestSetCommand_Execute_MissingArguments(t *testing.T) {
	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"--lang", "ja", "key"}) // Missing value
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 2) // ExitUsageError
}

func TestSetCommand_Execute_NonexistentKey(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "nonexistent_key", "value"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1) // ExitFailure
}

func TestSetCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings", "--lang", "ja", "key", "value"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1) // ExitFailure
}
