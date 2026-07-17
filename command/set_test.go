package command

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"

	"github.com/google/subcommands"
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

// withStdin temporarily replaces os.Stdin with a pipe fed by content, runs
// fn, and restores the original os.Stdin afterward.
func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()

	old := os.Stdin
	r, w, err := os.Pipe()
	test.AssertNoError(t, err)

	go func() {
		w.WriteString(content)
		w.Close()
	}()

	os.Stdin = r
	defer func() { os.Stdin = old }()

	fn()
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

func TestSetCommand_Execute_CreatesNewKey(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "new_key", "値"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	def, exists := xc.Strings["new_key"]
	if !exists {
		t.Fatal("new_key should have been created")
	}
	test.AssertEqual(t, def.Localizations["ja"].StringUnit.Value, "値")
	test.AssertEqual(t, def.ExtractionState, "")
}

func TestSetCommand_Execute_StateIgnoredOnExistingKey(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"existing": {
				"extractionState": "manual",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Old"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "en", "--state", "stale", "existing", "New"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["existing"].ExtractionState, "manual")
	test.AssertEqual(t, xc.Strings["existing"].Localizations["en"].StringUnit.Value, "New")
}

func TestSetCommand_Execute_CreatesNewKeyWithState(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "en", "--state", "manual", "manual_key", "Hello"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["manual_key"].ExtractionState, "manual")
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

func TestSetCommand_Execute_UnknownLanguageRejected(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}},
					"ja": {"stringUnit": {"state": "translated", "value": "テスト"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "jp", "test_key", "value"})
	test.AssertNoError(t, err)

	var errOutput string
	captureOutput(func() {
		errOutput = captureStderr(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 2) // ExitUsageError
		})
	})

	if !strings.Contains(errOutput, `did you mean "ja"?`) {
		t.Errorf("error should suggest 'ja' for typo 'jp', got: %q", errOutput)
	}

	// Verify the catalog was not modified
	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	if _, exists := xc.Strings["test_key"].Localizations["jp"]; exists {
		t.Error("unknown language 'jp' should not have been added to the catalog")
	}
}

func TestSetCommand_Execute_CaseMismatchLanguageSuggestsCorrectCode(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}},
					"ja": {"stringUnit": {"state": "translated", "value": "テスト"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "JA", "test_key", "value"})
	test.AssertNoError(t, err)

	var errOutput string
	captureOutput(func() {
		errOutput = captureStderr(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 2) // ExitUsageError
		})
	})

	if !strings.Contains(errOutput, `did you mean "ja"?`) {
		t.Errorf("error should suggest 'ja' for case mismatch 'JA', got: %q", errOutput)
	}
}

func TestSetCommand_Execute_AllowNewLanguageFlagPermitsAddition(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}},
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
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "fr", "--allow-new-language", "test_key", "Valeur"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Successfully set translation") {
		t.Errorf("output should contain success message, got: %q", output)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	loc, exists := xc.Strings["test_key"].Localizations["fr"]
	if !exists {
		t.Fatal("new language 'fr' should have been added with --allow-new-language")
	}
	test.AssertEqual(t, loc.StringUnit.Value, "Valeur")
}

func TestSetCommand_Execute_ExistingLanguageSucceedsWithoutFlag(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"test_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Test"}},
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
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "test_key", "更新"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["test_key"].Localizations["ja"].StringUnit.Value, "更新")
}

func TestSetCommand_Execute_EmptyCatalogAllowsFirstLanguageWithoutFlag(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	// No non-source languages exist yet, so adding "ja" should not require --allow-new-language.
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "first_key", "初めて"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["first_key"].Localizations["ja"].StringUnit.Value, "初めて")
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

func TestSetCommand_Execute_StdinBatch(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--stdin"})
	test.AssertNoError(t, err)

	stdinContent := `{"key": "greeting", "lang": "ja", "value": "こんにちは!"}
{"key": "farewell", "lang": "ja", "value": "さようなら"}
{"key": "farewell", "lang": "en", "value": "Goodbye"}
`

	var output string
	withStdin(t, stdinContent, func() {
		output = captureOutput(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	if !strings.Contains(output, "Summary: 1 created, 2 updated") {
		t.Errorf("output should contain batch summary, got: %q", output)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)

	test.AssertEqual(t, xc.Strings["greeting"].Localizations["ja"].StringUnit.Value, "こんにちは!")

	farewell, exists := xc.Strings["farewell"]
	if !exists {
		t.Fatal("farewell key should have been created by the batch")
	}
	test.AssertEqual(t, farewell.Localizations["ja"].StringUnit.Value, "さようなら")
	test.AssertEqual(t, farewell.Localizations["en"].StringUnit.Value, "Goodbye")
}

func TestSetCommand_Execute_StdinInvalidLineAbortsWholeBatch(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	beforeBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err = flagSet.Parse([]string{"-f", filePath, "--stdin"})
	test.AssertNoError(t, err)

	// Second line is missing the required "lang" field, and the third line is
	// malformed JSON; both should be reported and nothing should be written.
	stdinContent := `{"key": "greeting", "lang": "ja", "value": "Updated"}
{"key": "farewell", "value": "さようなら"}
not valid json
`

	var status subcommands.ExitStatus
	var errOutput string
	withStdin(t, stdinContent, func() {
		errOutput = captureStderr(func() {
			captureOutput(func() {
				status = cmd.Execute(context.Background(), flagSet)
			})
		})
	})
	test.AssertEqual(t, int(status), 2) // ExitUsageError

	if !strings.Contains(errOutput, "line 2") || !strings.Contains(errOutput, "line 3") {
		t.Errorf("error should report both invalid line numbers, got: %q", errOutput)
	}

	afterBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)
	if string(beforeBytes) != string(afterBytes) {
		t.Error("file should not have been modified when the batch contains an invalid line")
	}
}

func TestSetCommand_Execute_StdinRequireExistingReportsLineNumber(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--stdin", "--require-existing"})
	test.AssertNoError(t, err)

	stdinContent := `{"key": "greeting", "lang": "ja", "value": "更新"}
{"key": "does_not_exist", "lang": "ja", "value": "value"}
`

	var status subcommands.ExitStatus
	var errOutput string
	withStdin(t, stdinContent, func() {
		errOutput = captureStderr(func() {
			captureOutput(func() {
				status = cmd.Execute(context.Background(), flagSet)
			})
		})
	})
	test.AssertEqual(t, int(status), 2) // ExitUsageError

	if !strings.Contains(errOutput, "line 2") || !strings.Contains(errOutput, "does_not_exist") {
		t.Errorf("error should report line number and key, got: %q", errOutput)
	}

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["greeting"].Localizations["ja"].StringUnit.Value, "こんにちは")
}

func TestSetCommand_Execute_DryRunDoesNotWriteFile(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	beforeBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err = flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--dry-run", "greeting", "変更後"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("output should be marked as dry-run, got: %q", output)
	}

	afterBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)
	if string(beforeBytes) != string(afterBytes) {
		t.Error("file should not have been modified by --dry-run")
	}
}

func TestSetCommand_Execute_StdinDryRunDoesNotWriteFile(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	beforeBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err = flagSet.Parse([]string{"-f", filePath, "--stdin", "--dry-run"})
	test.AssertNoError(t, err)

	stdinContent := `{"key": "greeting", "lang": "ja", "value": "変更後"}
{"key": "new_key", "lang": "ja", "value": "新規"}
`

	var output string
	withStdin(t, stdinContent, func() {
		output = captureOutput(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	if !strings.Contains(output, "Summary: 1 created, 1 updated") {
		t.Errorf("output should contain batch summary, got: %q", output)
	}

	afterBytes, err := os.ReadFile(filePath)
	test.AssertNoError(t, err)
	if string(beforeBytes) != string(afterBytes) {
		t.Error("file should not have been modified by --stdin --dry-run")
	}
}

func TestSetCommand_Execute_JSONOutput(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--json", "greeting", "こんにちは"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	var parsed setJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v for output: %q", err, output)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(parsed.Results))
	}
	// "greeting" already exists in the fixture (only its "en" localization is
	// set), so adding a "ja" localization to it is an update to the key, not
	// a new key creation.
	test.AssertEqual(t, parsed.Results[0].Key, "greeting")
	test.AssertEqual(t, parsed.Results[0].Lang, "ja")
	test.AssertEqual(t, parsed.Results[0].Action, "updated")
	test.AssertEqual(t, parsed.Summary.Created, 0)
	test.AssertEqual(t, parsed.Summary.Updated, 1)
}

func TestSetCommand_Execute_JSONOutput_CreatedKey(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--lang", "ja", "--json", "new_key", "こんにちは"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	var parsed setJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v for output: %q", err, output)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(parsed.Results))
	}
	test.AssertEqual(t, parsed.Results[0].Action, "created")
	test.AssertEqual(t, parsed.Summary.Created, 1)
	test.AssertEqual(t, parsed.Summary.Updated, 0)
}

func TestSetCommand_Execute_StdinJSONOutput(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--stdin", "--json"})
	test.AssertNoError(t, err)

	stdinContent := `{"key": "greeting", "lang": "ja", "value": "更新"}
{"key": "new_key", "lang": "ja", "value": "新規"}
`

	var output string
	withStdin(t, stdinContent, func() {
		output = captureOutput(func() {
			status := cmd.Execute(context.Background(), flagSet)
			test.AssertEqual(t, int(status), 0)
		})
	})

	var parsed setJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v for output: %q", err, output)
	}
	if len(parsed.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(parsed.Results))
	}
	test.AssertEqual(t, parsed.Summary.Created, 1)
	test.AssertEqual(t, parsed.Summary.Updated, 1)
}

func TestSetCommand_Execute_StdinWithPositionalArgsErrors(t *testing.T) {
	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"--stdin", "key", "value"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 2) // ExitUsageError
}

func TestSetCommand_Execute_StdinNewLanguagePropagatesWithinBatch(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &SetCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--stdin", "--allow-new-language"})
	test.AssertNoError(t, err)

	// "fr" is not yet present in the catalog. The first line introduces it
	// (allowed because --allow-new-language is set); the second line reuses
	// "fr" for a different key and should not be rejected as unknown.
	stdinContent := `{"key": "greeting", "lang": "fr", "value": "Bonjour"}
{"key": "farewell", "lang": "fr", "value": "Au revoir"}
`

	var status subcommands.ExitStatus
	withStdin(t, stdinContent, func() {
		captureOutput(func() {
			status = cmd.Execute(context.Background(), flagSet)
		})
	})
	test.AssertEqual(t, int(status), 0)

	xc, err := xcstrings.Load(filePath)
	test.AssertNoError(t, err)
	test.AssertEqual(t, xc.Strings["greeting"].Localizations["fr"].StringUnit.Value, "Bonjour")
	test.AssertEqual(t, xc.Strings["farewell"].Localizations["fr"].StringUnit.Value, "Au revoir")
}
