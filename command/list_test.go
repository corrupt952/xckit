package command

import (
	"context"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
)

func TestListCommand_Execute(t *testing.T) {
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
					"en": {"stringUnit": {"state": "translated", "value": "Key 2"}}
				}
			},
			"login.title": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Login"}}
				}
			},
			"login.button": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Sign In"}}
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
			name:           "list all keys",
			args:           []string{},
			expectedKeys:   []string{"key1:", "key2:", "login.title:", "login.button:"},
			expectedStatus: 0,
		},
		{
			name:           "list keys with prefix",
			args:           []string{"--prefix", "login"},
			expectedKeys:   []string{"login.title:", "login.button:"},
			expectedStatus: 0,
		},
		{
			name:           "list keys with non-matching prefix",
			args:           []string{"--prefix", "error"},
			expectedKeys:   []string{},
			expectedStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test.TempFile(t, "test.xcstrings", testContent)

			cmd := &ListCommand{}

			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)
			args := append([]string{"-f", filePath}, tt.args...)
			err := flagSet.Parse(args)
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), tt.expectedStatus)
			})

			if len(tt.expectedKeys) == 0 {
				if strings.Contains(output, "No keys found with prefix") {
					// Expected behavior for non-matching prefix
					return
				}
			}

			for _, expectedKey := range tt.expectedKeys {
				if !strings.Contains(output, expectedKey) {
					t.Errorf("output should contain %q, got: %q", expectedKey, output)
				}
			}
		})
	}
}

func TestListCommand_Execute_StateFilter(t *testing.T) {
	testContent := `{
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
			"stale.key": {
				"extractionState": "stale",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Stale"}}
				}
			}
		},
		"version": "1.0"
	}`

	tests := []struct {
		name         string
		args         []string
		mustContain  []string
		mustNotMatch []string
	}{
		{
			name:         "filter manual",
			args:         []string{"--state", "manual"},
			mustContain:  []string{"manual.key [manual]:", "Keys with state 'manual':"},
			mustNotMatch: []string{"active.key", "stale.key"},
		},
		{
			name:         "filter stale",
			args:         []string{"--state", "stale"},
			mustContain:  []string{"stale.key [stale]:", "Keys with state 'stale':"},
			mustNotMatch: []string{"active.key", "manual.key"},
		},
		{
			name:         "filter empty matches default",
			args:         []string{},
			mustContain:  []string{"active.key:", "manual.key [manual]:", "stale.key [stale]:"},
			mustNotMatch: []string{"active.key [", "active.key [manual]"},
		},
		{
			name:         "filter non-matching state",
			args:         []string{"--state", "new"},
			mustContain:  []string{"No keys found with state 'new'"},
			mustNotMatch: []string{"active.key", "manual.key", "stale.key"},
		},
		{
			name:         "prefix and state combined",
			args:         []string{"--prefix", "manual", "--state", "manual"},
			mustContain:  []string{"manual.key [manual]:", "Keys with prefix 'manual' and state 'manual':"},
			mustNotMatch: []string{"active.key", "stale.key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test.TempFile(t, "test.xcstrings", testContent)

			cmd := &ListCommand{}
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)
			args := append([]string{"-f", filePath}, tt.args...)
			err := flagSet.Parse(args)
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), 0)
			})

			for _, want := range tt.mustContain {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got: %q", want, output)
				}
			}
			for _, unwanted := range tt.mustNotMatch {
				if strings.Contains(output, unwanted) {
					t.Errorf("output should NOT contain %q, got: %q", unwanted, output)
				}
			}
		})
	}
}

func TestListCommand_Execute_PluralVariations(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"%lld items": {
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
								"other": {"stringUnit": {"state": "translated", "value": "%lld個のアイテム"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedStrings := []string{
		"%lld items:",
		"plural.one:",
		"plural.other:",
		"%lld item",
		"%lld items",
		"%lld個のアイテム",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestListCommand_Execute_DeviceVariations(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"welcome_message": {
				"localizations": {
					"en": {
						"variations": {
							"device": {
								"iphone": {"stringUnit": {"state": "translated", "value": "Welcome to our iPhone app!"}},
								"ipad": {"stringUnit": {"state": "translated", "value": "Welcome to our iPad app!"}},
								"other": {"stringUnit": {"state": "translated", "value": "Welcome to our app!"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedStrings := []string{
		"welcome_message:",
		"device.iphone:",
		"device.ipad:",
		"device.other:",
		"Welcome to our iPhone app!",
		"Welcome to our iPad app!",
		"Welcome to our app!",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestListCommand_Execute_Substitutions(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"%lld files in %lld folders": {
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

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	expectedStrings := []string{
		"%lld files in %lld folders:",
		"substitutions.files:",
		"substitutions.folders:",
		"plural.one:",
		"plural.other:",
		"%arg file",
		"%arg files",
		"%arg folder",
		"%arg folders",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestListCommand_Execute_JSON(t *testing.T) {
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
				"extractionState": "manual",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 2"}},
					"ja": {"stringUnit": {"state": "new", "value": ""}}
				}
			},
			"login.title": {
				"localizations": {
					"en": {
						"variations": {
							"plural": {
								"one": {"stringUnit": {"state": "translated", "value": "one"}},
								"other": {"stringUnit": {"state": "translated", "value": "other"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--json"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	var parsed listJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v, output: %q", err, output)
	}

	if len(parsed.Keys) != 3 {
		t.Fatalf("expected 3 keys, got %d: %+v", len(parsed.Keys), parsed.Keys)
	}

	byKey := make(map[string]listJSONKeyEntry, len(parsed.Keys))
	for _, k := range parsed.Keys {
		byKey[k.Key] = k
	}

	key1, ok := byKey["key1"]
	if !ok {
		t.Fatalf("expected key1 in output: %+v", parsed.Keys)
	}
	if key1.ExtractionState != "" {
		t.Errorf("key1 should have no extractionState, got %q", key1.ExtractionState)
	}
	if key1.Languages["en"].State != "translated" || key1.Languages["en"].Value != "Key 1" {
		t.Errorf("key1.en unexpected: %+v", key1.Languages["en"])
	}
	if key1.Languages["ja"].State != "translated" || key1.Languages["ja"].Value != "キー1" {
		t.Errorf("key1.ja unexpected: %+v", key1.Languages["ja"])
	}

	key2, ok := byKey["key2"]
	if !ok {
		t.Fatalf("expected key2 in output: %+v", parsed.Keys)
	}
	if key2.ExtractionState != "manual" {
		t.Errorf("key2 should have extractionState 'manual', got %q", key2.ExtractionState)
	}
	if key2.Languages["ja"].State != "new" {
		t.Errorf("key2.ja should have state 'new', got %+v", key2.Languages["ja"])
	}

	loginTitle, ok := byKey["login.title"]
	if !ok {
		t.Fatalf("expected login.title in output: %+v", parsed.Keys)
	}
	enEntry := loginTitle.Languages["en"]
	if enEntry.State != "translated" {
		t.Errorf("login.title.en should be translated, got %+v", enEntry)
	}
	if len(enEntry.Units) != 2 {
		t.Errorf("login.title.en should have 2 units, got %+v", enEntry.Units)
	}
	if jaEntry := loginTitle.Languages["ja"]; jaEntry.State != "missing" {
		t.Errorf("login.title.ja should be missing, got %+v", jaEntry)
	}
}

func TestListCommand_Execute_JSON_WithFilters(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"login.title": {
				"extractionState": "manual",
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Login"}}
				}
			},
			"other.key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Other"}}
				}
			}
		},
		"version": "1.0"
	}`

	filePath := test.TempFile(t, "test.xcstrings", testContent)

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--prefix", "login", "--state", "manual", "--json"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	var parsed listJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v, output: %q", err, output)
	}
	if len(parsed.Keys) != 1 || parsed.Keys[0].Key != "login.title" {
		t.Errorf("expected only login.title, got %+v", parsed.Keys)
	}
}

func TestListCommand_Execute_JSON_Empty(t *testing.T) {
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

	cmd := &ListCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", filePath, "--prefix", "nonexistent", "--json"})
	test.AssertNoError(t, err)

	output := captureOutput(func() {
		status := cmd.Execute(context.Background(), flagSet)
		test.AssertEqual(t, int(status), 0)
	})

	var parsed listJSONOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON, got error %v, output: %q", err, output)
	}
	if len(parsed.Keys) != 0 {
		t.Errorf("expected no keys, got %+v", parsed.Keys)
	}
}

func TestListCommand_Execute_FileNotFound(t *testing.T) {
	cmd := &ListCommand{}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.SetOutput(&strings.Builder{}) // Suppress error output
	cmd.SetFlags(flagSet)
	err := flagSet.Parse([]string{"-f", "nonexistent.xcstrings"})
	test.AssertNoError(t, err)

	status := cmd.Execute(context.Background(), flagSet)
	test.AssertEqual(t, int(status), 1) // ExitFailure
}
