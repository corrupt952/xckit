package command

import (
	"context"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

const removeFixture = `{
	"sourceLanguage": "en",
	"strings": {
		"active.key": {
			"localizations": {
				"en": {"stringUnit": {"state": "translated", "value": "Active"}}
			}
		},
		"manual.key": {
			"extractionState": "manual",
			"localizations": {
				"en": {"stringUnit": {"state": "translated", "value": "Manual"}}
			}
		},
		"stale.one": {
			"extractionState": "stale",
			"localizations": {
				"en": {"stringUnit": {"state": "translated", "value": "Stale 1"}}
			}
		},
		"stale.two": {
			"extractionState": "stale",
			"localizations": {
				"en": {"stringUnit": {"state": "translated", "value": "Stale 2"}}
			}
		}
	},
	"version": "1.0"
}`

func runRemove(t *testing.T, filePath string, args ...string) (int, string) {
	t.Helper()
	cmd := &RemoveCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	parseArgs := append([]string{"-f", filePath}, args...)
	if err := flagSet.Parse(parseArgs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	var status int
	output := captureOutput(func() {
		status = int(cmd.Execute(context.Background(), flagSet))
	})
	return status, output
}

func TestRemoveCommand_Execute_SingleKey(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, output := runRemove(t, filePath, "manual.key")
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "Removed 1 key(s)") {
		t.Errorf("expected success message, got %q", output)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["manual.key"]; exists {
		t.Error("manual.key should have been removed")
	}
}

func TestRemoveCommand_Execute_ByStateStale(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, _ := runRemove(t, filePath, "--state", "stale")
	test.AssertEqual(t, status, 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["stale.one"]; exists {
		t.Error("stale.one should have been removed")
	}
	if _, exists := xc.Strings["stale.two"]; exists {
		t.Error("stale.two should have been removed")
	}
	if _, exists := xc.Strings["active.key"]; !exists {
		t.Error("active.key should still exist")
	}
}

func TestRemoveCommand_Execute_ByStateManual(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, _ := runRemove(t, filePath, "--state", "manual")
	test.AssertEqual(t, status, 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["manual.key"]; exists {
		t.Error("manual.key should have been removed")
	}
}

func TestRemoveCommand_Execute_KeyStateMismatchRefused(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, _ := runRemove(t, filePath, "--state", "stale", "manual.key")
	test.AssertEqual(t, status, 1)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["manual.key"]; !exists {
		t.Error("manual.key should still exist after state mismatch refusal")
	}
}

func TestRemoveCommand_Execute_SingleKeyDryRun(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, output := runRemove(t, filePath, "--dry-run", "manual.key")
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "Would remove 1 key(s)") {
		t.Errorf("expected dry-run summary, got %q", output)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["manual.key"]; !exists {
		t.Error("dry-run must not modify the file")
	}
}

func TestRemoveCommand_Execute_DryRun(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)

	status, output := runRemove(t, filePath, "--state", "stale", "--dry-run")
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "Would remove 2 key(s)") {
		t.Errorf("expected dry-run summary, got %q", output)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["stale.one"]; !exists {
		t.Error("dry-run must not modify the file")
	}
}

func TestRemoveCommand_Execute_MissingKey(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)
	status, _ := runRemove(t, filePath, "nonexistent")
	test.AssertEqual(t, status, 1)
}

func TestRemoveCommand_Execute_NoArgs(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)
	status, _ := runRemove(t, filePath)
	test.AssertEqual(t, status, 2) // ExitUsageError
}

func TestRemoveCommand_Execute_StateWithNoMatches(t *testing.T) {
	filePath := test.TempFile(t, "test.xcstrings", removeFixture)
	status, output := runRemove(t, filePath, "--state", "new")
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "No keys found with state 'new'") {
		t.Errorf("expected no-keys message, got %q", output)
	}
}
