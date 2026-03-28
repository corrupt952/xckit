package command

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xckit/helper/test"
	"xckit/xcstrings"
)

func TestExportCommand_Execute_SimpleCSV(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"comment": "A greeting",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 row), got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en,ja:state,ja")
	test.AssertEqual(t, lines[1], "greeting,A greeting,,translated,Hello,translated,こんにちは")
}

func TestExportCommand_Execute_PluralVariations(t *testing.T) {
	testContent := `{
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
					},
					"ja": {
						"variations": {
							"plural": {
								"other": {"stringUnit": {"state": "translated", "value": "%lld個"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en,ja:state,ja")
	test.AssertEqual(t, lines[1], "item_count[plural.one],,,translated,%lld item,,")
	test.AssertEqual(t, lines[2], "item_count[plural.other],,,translated,%lld items,translated,%lld個")
}

func TestExportCommand_Execute_DeviceVariations(t *testing.T) {
	testContent := `{
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

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en")
	test.AssertEqual(t, lines[1], "welcome[device.ipad],,,translated,Welcome iPad")
	test.AssertEqual(t, lines[2], "welcome[device.iphone],,,translated,Welcome iPhone")
}

func TestExportCommand_Execute_NestedVariations(t *testing.T) {
	testContent := `{
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
								},
								"other": {
									"variations": {
										"plural": {
											"one": {"stringUnit": {"state": "translated", "value": "%lld photo"}},
											"other": {"stringUnit": {"state": "translated", "value": "%lld photos"}}
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

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en")

	// Sorted: device.iphone.plural.one, device.iphone.plural.other, device.other.plural.one, device.other.plural.other
	if !strings.Contains(output, "photos[device.iphone.plural.one]") {
		t.Errorf("output should contain nested variation key, got: %q", output)
	}
	if !strings.Contains(output, "%lld photo on iPhone") {
		t.Errorf("output should contain nested variation value, got: %q", output)
	}
}

func TestExportCommand_Execute_Substitutions(t *testing.T) {
	testContent := `{
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
							},
							"folders": {
								"argNum": 2,
								"formatSpecifier": "lld",
								"variations": {
									"plural": {
										"one": {"stringUnit": {"state": "translated", "value": "%arg folder"}},
										"other": {"stringUnit": {"state": "translated", "value": "%arg folders"}}
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

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedSubstrings := []string{
		"file_summary[substitutions.files.plural.one]",
		"file_summary[substitutions.files.plural.other]",
		"file_summary[substitutions.folders.plural.one]",
		"file_summary[substitutions.folders.plural.other]",
		"%arg file",
		"%arg files",
		"%arg folder",
		"%arg folders",
	}
	for _, expected := range expectedSubstrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestExportCommand_Execute_OutputFile(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"hello": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)
	outPath := filepath.Join(t.TempDir(), "output.csv")

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv", "-o", outPath})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 0)

	data, err := os.ReadFile(outPath)
	test.AssertNoError(t, err)

	output := string(data)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en")
	test.AssertEqual(t, lines[1], "hello,,,translated,Hello")
}

func TestExportCommand_Execute_MissingFormat(t *testing.T) {
	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestExportCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{})
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings", "--format", "csv"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1)
}

func TestExportCommand_Execute_ShouldTranslateFalse(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"app_name": {
				"shouldTranslate": false,
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "MyApp"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	if !strings.Contains(output, "false") {
		t.Errorf("output should contain shouldTranslate=false, got: %q", output)
	}
}

func TestExportCommand_Execute_LanguageOrder(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"hello": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"zh": {"stringUnit": {"state": "translated", "value": "你好"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}},
					"es": {"stringUnit": {"state": "translated", "value": "Hola"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Source language (en) first, then es, ja, zh alphabetically
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en,es:state,es,ja:state,ja,zh:state,zh")
}

func TestExportCommand_Execute_CSVQuoting(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"msg": {
				"comment": "Has, comma",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello, World"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	// encoding/csv should quote fields with commas
	if !strings.Contains(output, `"Has, comma"`) {
		t.Errorf("expected quoted comment field, got: %q", output)
	}
	if !strings.Contains(output, `"Hello, World"`) {
		t.Errorf("expected quoted value field, got: %q", output)
	}
}

func TestExportCommand_Execute_MultipleKeysSorted(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"zebra": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Zebra"}}
				}
			},
			"apple": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Apple"}}
				}
			},
			"mango": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Mango"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[1], "apple,") {
		t.Errorf("first data row should be apple, got: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "mango,") {
		t.Errorf("second data row should be mango, got: %q", lines[2])
	}
	if !strings.HasPrefix(lines[3], "zebra,") {
		t.Errorf("third data row should be zebra, got: %q", lines[3])
	}
}

func TestWriteCSV_Unit(t *testing.T) {
	boolFalse := false
	xc := &xcstrings.XCStrings{
		SourceLanguage: "en",
		Strings: map[string]xcstrings.StringDefinition{
			"greeting": {
				Comment: "A greeting",
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "Hello"}},
					"ja": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "こんにちは"}},
				},
			},
			"no_translate": {
				ShouldTranslate: &boolFalse,
				Localizations: map[string]xcstrings.Localization{
					"en": {StringUnit: &xcstrings.StringUnit{State: "translated", Value: "AppName"}},
				},
			},
		},
		Version: "1.0",
	}

	var buf bytes.Buffer
	err := writeCSV(&buf, xc)
	test.AssertNoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}
	test.AssertEqual(t, lines[0], "key,comment,shouldTranslate,en:state,en,ja:state,ja")
	test.AssertEqual(t, lines[1], "greeting,A greeting,,translated,Hello,translated,こんにちは")
	test.AssertEqual(t, lines[2], "no_translate,,false,translated,AppName,,")
}

func TestExportCommand_Execute_EmptyStrings(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ExportCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--format", "csv"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Only header row
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (header only), got %d: %q", len(lines), output)
	}
}
