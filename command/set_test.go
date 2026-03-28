package command

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

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

func TestSetCommand_Execute_PluralVariation(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"item_count": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "%lld items"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--plural", "other", "item_count", "%lldつのアイテム"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Successfully set translation") {
		t.Errorf("output should contain success message, got: %q", output)
	}

	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	loc := xcstringsData.Strings["item_count"].Localizations["ja"]
	if loc.Variations == nil {
		t.Fatal("variations should exist")
	}
	if loc.Variations.Plural == nil {
		t.Fatal("plural variations should exist")
	}
	pluralOther := loc.Variations.Plural["other"]
	if pluralOther == nil || pluralOther.StringUnit == nil {
		t.Fatal("plural other variation should exist")
	}
	test.AssertEqual(t, pluralOther.StringUnit.Value, "%lldつのアイテム")
	test.AssertEqual(t, pluralOther.StringUnit.State, "translated")
}

func TestSetCommand_Execute_DeviceVariation(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"tap_message": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Tap here"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--device", "ipad", "tap_message", "ここをタップ"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Successfully set translation") {
		t.Errorf("output should contain success message, got: %q", output)
	}

	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	loc := xcstringsData.Strings["tap_message"].Localizations["ja"]
	if loc.Variations == nil {
		t.Fatal("variations should exist")
	}
	if loc.Variations.Device == nil {
		t.Fatal("device variations should exist")
	}
	deviceIPad := loc.Variations.Device["ipad"]
	if deviceIPad == nil || deviceIPad.StringUnit == nil {
		t.Fatal("device ipad variation should exist")
	}
	test.AssertEqual(t, deviceIPad.StringUnit.Value, "ここをタップ")
	test.AssertEqual(t, deviceIPad.StringUnit.State, "translated")
}

func TestSetCommand_Execute_PluralAndDeviceVariation(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"item_count": {
				"localizations": {}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--plural", "one", "--device", "iphone", "item_count", "1つのアイテム"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Successfully set translation") {
		t.Errorf("output should contain success message, got: %q", output)
	}

	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	loc := xcstringsData.Strings["item_count"].Localizations["ja"]
	if loc.Variations == nil {
		t.Fatal("variations should exist")
	}
	if loc.Variations.Device == nil {
		t.Fatal("device variations should exist")
	}
	deviceIPhone := loc.Variations.Device["iphone"]
	if deviceIPhone == nil || deviceIPhone.Variations == nil {
		t.Fatal("device iphone variation with nested variations should exist")
	}
	if deviceIPhone.Variations.Plural == nil {
		t.Fatal("nested plural variations should exist")
	}
	pluralOne := deviceIPhone.Variations.Plural["one"]
	if pluralOne == nil || pluralOne.StringUnit == nil {
		t.Fatal("nested plural one variation should exist")
	}
	test.AssertEqual(t, pluralOne.StringUnit.Value, "1つのアイテム")
	test.AssertEqual(t, pluralOne.StringUnit.State, "translated")
}

func TestSetCommand_Execute_MigrationWarning(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"ja": {"stringUnit": {"state": "translated", "value": "テスト"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--plural", "other", "test_key", "テスト複数"})
	test.AssertNoError(t, err)

	var stderrOutput string
	captureOutput(func() {
		stderrOutput = captureStderr(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	if !strings.Contains(stderrOutput, "Warning: existing plain stringUnit") {
		t.Errorf("stderr should contain migration warning, got: %q", stderrOutput)
	}

	// Verify the plain stringUnit was cleared
	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	loc := xcstringsData.Strings["test_key"].Localizations["ja"]
	if loc.StringUnit != nil {
		t.Error("plain stringUnit should have been cleared after migration")
	}
	if loc.Variations == nil || loc.Variations.Plural == nil {
		t.Fatal("plural variations should exist after migration")
	}
}

func TestSetCommand_Execute_ForceFlag(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"ja": {"stringUnit": {"state": "translated", "value": "テスト"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--plural", "other", "--force", "test_key", "テスト複数"})
	test.AssertNoError(t, err)

	var stderrOutput string
	captureOutput(func() {
		stderrOutput = captureStderr(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	if strings.Contains(stderrOutput, "Warning") {
		t.Errorf("stderr should NOT contain migration warning with --force, got: %q", stderrOutput)
	}
}

func TestSetCommand_Execute_InvalidPluralCategory(t *testing.T) {
	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"--lang", "ja", "--plural", "invalid", "key", "value"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 2) // ExitUsageError
}

func TestSetCommand_Execute_InvalidDeviceCategory(t *testing.T) {
	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"--lang", "ja", "--device", "invalid", "key", "value"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 2) // ExitUsageError
}

func TestSetCommand_Execute_NoMigrationWarningForNewLocalization(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--plural", "other", "test_key", "テスト"})
	test.AssertNoError(t, err)

	var stderrOutput string
	captureOutput(func() {
		stderrOutput = captureStderr(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	if strings.Contains(stderrOutput, "Warning") {
		t.Errorf("stderr should NOT contain migration warning for new localization, got: %q", stderrOutput)
	}
}

func TestSetCommand_Execute_MultipleDeviceVariations(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"tap_message": {
				"localizations": {}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	// Set first device variation
	cmd1 := &SetCommand{}
	flagSet1 := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd1.SetFlags(flagSet1)
	err := flagSet1.Parse([]string{"-f", filePath, "--lang", "ja", "--device", "iphone", "tap_message", "タップ(iPhone)"})
	test.AssertNoError(t, err)

	captureOutput(func() {
		status := cmd1.Execute(context.Background(), flagSet1)
		test.AssertEqual(t, int(status), 0)
	})

	// Set second device variation
	cmd2 := &SetCommand{}
	flagSet2 := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd2.SetFlags(flagSet2)
	err = flagSet2.Parse([]string{"-f", filePath, "--lang", "ja", "--device", "mac", "tap_message", "クリック(Mac)"})
	test.AssertNoError(t, err)

	captureOutput(func() {
		status := cmd2.Execute(context.Background(), flagSet2)
		test.AssertEqual(t, int(status), 0)
	})

	xcstringsData, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	loc := xcstringsData.Strings["tap_message"].Localizations["ja"]
	if loc.Variations == nil || loc.Variations.Device == nil {
		t.Fatal("device variations should exist")
	}

	iphone := loc.Variations.Device["iphone"]
	if iphone == nil || iphone.StringUnit == nil {
		t.Fatal("iphone variation should exist")
	}
	test.AssertEqual(t, iphone.StringUnit.Value, "タップ(iPhone)")

	mac := loc.Variations.Device["mac"]
	if mac == nil || mac.StringUnit == nil {
		t.Fatal("mac variation should exist")
	}
	test.AssertEqual(t, mac.StringUnit.Value, "クリック(Mac)")
}
