package command

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func TestImportCommand_Execute_SimpleCSV(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "needs_review", "value": ""}}
				}
			}
		},
		"version": "1.0"
	}`

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,A greeting,,translated,Hello,translated,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "1 updated") {
		t.Errorf("expected 1 updated, got: %q", output)
	}

	// Verify the file was updated
	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["greeting"].Localizations["ja"]
	test.AssertEqual(t, loc.StringUnit.Value, "こんにちは")
	test.AssertEqual(t, loc.StringUnit.State, "translated")
}

func TestImportCommand_Execute_PluralVariations(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {
			"item_count": {
				"localizations": {
					"en": {
						"variations": {
							"plural": {
								"one": {"stringUnit": {"state": "translated", "value": "%lld item"}},
								"other": {"stringUnit": {"state": "translated", "value": "%lld items"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nitem_count[plural.one],,,translated,%lld item,,1個のアイテム\nitem_count[plural.other],,,translated,%lld items,,アイテム\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "2 updated") {
		t.Errorf("expected 2 updated, got: %q", output)
	}

	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["item_count"].Localizations["ja"]
	if loc.Variations == nil || loc.Variations.Plural == nil {
		t.Fatal("expected plural variations for ja")
	}
	test.AssertEqual(t, loc.Variations.Plural["one"].StringUnit.Value, "1個のアイテム")
	test.AssertEqual(t, loc.Variations.Plural["other"].StringUnit.Value, "アイテム")
}

func TestImportCommand_Execute_DeviceVariations(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {
			"welcome": {
				"localizations": {
					"en": {
						"variations": {
							"device": {
								"ipad": {"stringUnit": {"state": "translated", "value": "Welcome iPad"}},
								"iphone": {"stringUnit": {"state": "translated", "value": "Welcome iPhone"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nwelcome[device.ipad],,,translated,Welcome iPad,,iPadへようこそ\nwelcome[device.iphone],,,translated,Welcome iPhone,,iPhoneへようこそ\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "2 updated") {
		t.Errorf("expected 2 updated, got: %q", output)
	}

	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["welcome"].Localizations["ja"]
	if loc.Variations == nil || loc.Variations.Device == nil {
		t.Fatal("expected device variations for ja")
	}
	test.AssertEqual(t, loc.Variations.Device["ipad"].StringUnit.Value, "iPadへようこそ")
	test.AssertEqual(t, loc.Variations.Device["iphone"].StringUnit.Value, "iPhoneへようこそ")
}

func TestImportCommand_Execute_NestedVariations(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {
			"photos": {
				"localizations": {
					"en": {
						"variations": {
							"device": {
								"iphone": {
									"variations": {
										"plural": {
											"one": {"stringUnit": {"state": "translated", "value": "%lld photo on iPhone"}},
											"other": {"stringUnit": {"state": "translated", "value": "%lld photos on iPhone"}}
										}
									}
								}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nphotos[device.iphone.plural.one],,,translated,%lld photo on iPhone,,iPhoneの写真%lld枚\nphotos[device.iphone.plural.other],,,translated,%lld photos on iPhone,,iPhoneの写真%lld枚\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "2 updated") {
		t.Errorf("expected 2 updated, got: %q", output)
	}

	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["photos"].Localizations["ja"]
	if loc.Variations == nil || loc.Variations.Device == nil {
		t.Fatal("expected device variations for ja")
	}
	dv := loc.Variations.Device["iphone"]
	if dv == nil || dv.Variations == nil || dv.Variations.Plural == nil {
		t.Fatal("expected nested plural variations under device.iphone for ja")
	}
	test.AssertEqual(t, dv.Variations.Plural["one"].StringUnit.Value, "iPhoneの写真%lld枚")
}

func TestImportCommand_Execute_Substitutions(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {
			"file_summary": {
				"localizations": {
					"en": {
						"stringUnit": {"state": "translated", "value": "%#@files@ in %#@folders@"},
						"substitutions": {
							"files": {
								"argNum": 1,
								"formatSpecifier": "lld",
								"variations": {
									"plural": {
										"one": {"stringUnit": {"state": "translated", "value": "%arg file"}},
										"other": {"stringUnit": {"state": "translated", "value": "%arg files"}}
									}
								}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nfile_summary[substitutions.files.plural.one],,,translated,%arg file,,ファイル%arg個\nfile_summary[substitutions.files.plural.other],,,translated,%arg files,,ファイル%arg個\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "2 updated") {
		t.Errorf("expected 2 updated, got: %q", output)
	}

	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["file_summary"].Localizations["ja"]
	if loc.Substitutions == nil {
		t.Fatal("expected substitutions for ja")
	}
	sub := loc.Substitutions["files"]
	if sub.Variations.Plural == nil {
		t.Fatal("expected plural variations in files substitution")
	}
	test.AssertEqual(t, sub.Variations.Plural["one"].StringUnit.Value, "ファイル%arg個")
	test.AssertEqual(t, sub.Variations.Plural["other"].StringUnit.Value, "ファイル%arg個")
}

func TestImportCommand_Execute_DryRun(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Hello,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "--dry-run", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Dry run") {
		t.Errorf("expected dry run output, got: %q", output)
	}

	// Verify file was NOT modified
	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)
	_, exists := xc.Strings["greeting"].Localizations["ja"]
	if exists {
		t.Error("dry run should not modify the file")
	}
}

func TestImportCommand_Execute_Backup(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Hello,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "--backup", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "1 updated") {
		t.Errorf("expected 1 updated, got: %q", output)
	}

	// Verify backup file exists
	bakPath := xcPath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Error("backup file should exist")
	}
}

func TestImportCommand_Execute_OnMissingKeyError(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nnonexistent,,,translated,Hello,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "--on-missing-key", "error", csvPath})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestImportCommand_Execute_OnMissingKeySkip(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\nnonexistent,,,translated,Hello,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "--on-missing-key", "skip", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "1 skipped") {
		t.Errorf("expected 1 skipped, got: %q", output)
	}
}

func TestImportCommand_Execute_ClearEmpty(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Hello,,\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "--clear-empty", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "1 cleared") {
		t.Errorf("expected 1 cleared, got: %q", output)
	}

	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["greeting"].Localizations["ja"]
	if loc.StringUnit != nil {
		t.Error("expected ja stringUnit to be cleared")
	}
}

func TestImportCommand_Execute_EmptyCellSkippedByDefault(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Hello,,\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "0 updated") {
		t.Errorf("expected 0 updated, got: %q", output)
	}

	// Verify ja translation was NOT cleared
	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["greeting"].Localizations["ja"]
	test.AssertEqual(t, loc.StringUnit.Value, "こんにちは")
}

func TestImportCommand_Execute_MissingFormat(t *testing.T) {
	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestImportCommand_Execute_MissingCSVFile(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestImportCommand_Execute_CSVFileNotFound(t *testing.T) {
	xcContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", "nonexistent.csv"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestImportCommand_Execute_SourceLanguageIgnored(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Modified English,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := test.TempFile(t, "translations.csv", csvContent)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	// Verify en was NOT modified
	xc, err := xcstrings.Load(xcPath)
	test.AssertNoError(t, err)

	loc := xc.Strings["greeting"].Localizations["en"]
	test.AssertEqual(t, loc.StringUnit.Value, "Hello")
}

func TestParseKeyBracket(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantSuffix string
	}{
		{"greeting", "greeting", ""},
		{"item_count[plural.one]", "item_count", "plural.one"},
		{"welcome[device.iphone]", "welcome", "device.iphone"},
		{"photos[device.iphone.plural.one]", "photos", "device.iphone.plural.one"},
		{"file_summary[substitutions.files.plural.one]", "file_summary", "substitutions.files.plural.one"},
	}

	for _, tt := range tests {
		key, suffix := parseKeyBracket(tt.input)
		if key != tt.wantKey {
			t.Errorf("parseKeyBracket(%q) key = %q, want %q", tt.input, key, tt.wantKey)
		}
		if suffix != tt.wantSuffix {
			t.Errorf("parseKeyBracket(%q) suffix = %q, want %q", tt.input, suffix, tt.wantSuffix)
		}
	}
}

func TestParseHeader(t *testing.T) {
	header := []string{"key", "comment", "shouldTranslate", "en:state", "en", "ja:state", "ja"}
	cols, err := parseHeader(header)
	test.AssertNoError(t, err)

	if len(cols) != 2 {
		t.Fatalf("expected 2 language columns, got %d", len(cols))
	}
	test.AssertEqual(t, cols[0].lang, "en")
	test.AssertEqual(t, cols[0].stateIdx, 3)
	test.AssertEqual(t, cols[0].valueIdx, 4)
	test.AssertEqual(t, cols[1].lang, "ja")
	test.AssertEqual(t, cols[1].stateIdx, 5)
	test.AssertEqual(t, cols[1].valueIdx, 6)
}

func TestImportCommand_Execute_OutputFile(t *testing.T) {
	xcContent := `{
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

	csvContent := "key,comment,shouldTranslate,en:state,en,ja:state,ja\ngreeting,,,translated,Hello,,こんにちは\n"

	xcPath := test.TempFile(t, "test.xcstrings", xcContent)
	csvPath := filepath.Join(filepath.Dir(xcPath), "translations.csv")
	os.WriteFile(csvPath, []byte(csvContent), 0644)

	cmd := &ImportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", xcPath, "--format", "csv", csvPath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "Imported") {
		t.Errorf("expected Imported output, got: %q", output)
	}
}
