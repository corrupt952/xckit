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
		"Languages:",
		"ja",
		"Keys",
		"Strings",
		"needs_review",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestStatusCommand_Execute_WithNeedsReview(t *testing.T) {
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
					"ja": {"stringUnit": {"state": "needs_review", "value": "キー2"}}
				}
			},
			"key3": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Key 3"}},
					"ja": {"stringUnit": {"state": "new", "value": ""}}
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
		"1 needs_review",
		"Keys",
		"Strings",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
	}
}

func TestStatusCommand_Execute_WithVariations(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"item_count": {
				"localizations": {
					"ja": {
						"variations": {
							"plural": {
								"one": {"stringUnit": {"state": "translated", "value": "%d アイテム"}},
								"other": {"stringUnit": {"state": "translated", "value": "%d アイテム"}}
							}
						}
					}
				}
			},
			"simple_key": {
				"localizations": {
					"ja": {"stringUnit": {"state": "translated", "value": "シンプル"}}
				}
			},
			"partial_key": {
				"localizations": {
					"ja": {
						"variations": {
							"plural": {
								"one": {"stringUnit": {"state": "translated", "value": "%d 個"}},
								"other": {"stringUnit": {"state": "new", "value": ""}}
							}
						}
					}
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
		"Keys   2/3",
		"Strings   4/5",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got: %q", expected, output)
		}
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
