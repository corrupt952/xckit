package command

import (
	"context"
	"flag"
	"path/filepath"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func TestStaleCommand_Execute(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name:           "list stale keys",
			args:           []string{},
			expectedStatus: 0,
			shouldContain:  []string{"stale_key", "another_stale_key", "Stale keys (2)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixturePath := filepath.Join("../fixtures", "stale.xcstrings")

			cmd := &StaleCommand{}
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)

			args := append([]string{"-f", fixturePath}, tt.args...)
			err := flagSet.Parse(args)
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), tt.expectedStatus)
			})

			for _, expected := range tt.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("output should contain %q, got: %q", expected, output)
				}
			}
		})
	}
}

func TestStaleCommand_Execute_NoStaleKeys(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"key1": {
				"extractionState": "manual",
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
	fixturePath := filepath.Join("../fixtures", "stale.xcstrings")

	cmd := &StaleCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", fixturePath, "--remove", "--dry-run"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Would remove 2 stale key(s)") {
		t.Errorf("output should contain dry-run message, got: %q", output)
	}
	if !strings.Contains(output, "stale_key") {
		t.Errorf("output should contain 'stale_key', got: %q", output)
	}
	if !strings.Contains(output, "another_stale_key") {
		t.Errorf("output should contain 'another_stale_key', got: %q", output)
	}

	// Verify the fixture was not modified by loading it again
	loaded, err := xcstrings.Load(fixturePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, len(loaded.StaleKeys()), 2)
}

func TestStaleCommand_Execute_Remove(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"active_key": {
				"extractionState": "manual",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Active"}}
				}
			},
			"stale_key": {
				"extractionState": "stale",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Stale"}}
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

	// Verify the file was modified
	loaded, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, len(loaded.Strings), 1)
	test.AssertEqual(t, len(loaded.StaleKeys()), 0)
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

func TestStaleCommand_Metadata(t *testing.T) {
	cmd := &StaleCommand{}

	test.AssertEqual(t, cmd.Name(), "stale")
	test.AssertEqual(t, cmd.Synopsis(), "List or remove stale keys")

	usage := cmd.Usage()
	if !strings.Contains(usage, "stale") {
		t.Errorf("usage should contain 'stale', got: %q", usage)
	}
}
