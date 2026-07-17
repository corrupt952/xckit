package command

import (
	"context"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"xckit/helper/test"
)

func runLintCommand(t *testing.T, filePath string, args ...string) (string, int) {
	t.Helper()

	cmd := &LintCommand{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(flagSet)
	fullArgs := append([]string{"-f", filePath}, args...)
	err := flagSet.Parse(fullArgs)
	test.AssertNoError(t, err)

	var status int
	output := captureOutput(func() {
		status = int(cmd.Execute(context.Background(), flagSet))
	})
	return output, status
}

func TestLintCommand_Metadata(t *testing.T) {
	cmd := &LintCommand{}
	test.AssertEqual(t, cmd.Name(), "lint")
	test.AssertEqual(t, cmd.Synopsis(), "Statically validate a catalog for common inconsistencies")
	if !strings.Contains(cmd.Usage(), "lint") {
		t.Errorf("usage should contain 'lint', got: %q", cmd.Usage())
	}
}

func TestLintCommand_NoIssues(t *testing.T) {
	for _, fixture := range []string{
		"all_translated.xcstrings",
		"all_untranslated.xcstrings",
		"device_variations.xcstrings",
		"empty.xcstrings",
		"es_only_untranslated.xcstrings",
		"ja_only_untranslated.xcstrings",
		"mixed_states.xcstrings",
		"needs_review.xcstrings",
		"nested_variations.xcstrings",
		"partially_translated.xcstrings",
		"plural_variations.xcstrings",
		"plural_variations_untranslated.xcstrings",
		"simple.xcstrings",
		"stale.xcstrings",
		"substitutions.xcstrings",
		"substitutions_untranslated.xcstrings",
	} {
		t.Run(fixture, func(t *testing.T) {
			output, status := runLintCommand(t, test.FixturePath(fixture))
			if status != 0 {
				t.Errorf("expected exit 0 for fixture %q, got %d; output: %s", fixture, status, output)
			}
			if !strings.Contains(output, "No issues found") {
				t.Errorf("expected 'No issues found' for fixture %q, got: %s", fixture, output)
			}
		})
	}
}

func TestLintCommand_EmptyKey(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Empty key"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	if !strings.Contains(output, "empty-key") {
		t.Errorf("expected output to contain 'empty-key', got: %s", output)
	}
}

func TestLintCommand_FormatSpecifierMismatch(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Complete %d Reminders"}},
					"ja": {"stringUnit": {"state": "translated", "value": "リマインダーを完了"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	if !strings.Contains(output, "format-specifier") {
		t.Errorf("expected output to contain 'format-specifier', got: %s", output)
	}
	if !strings.Contains(output, "missing %d") {
		t.Errorf("expected output to describe the missing %%d, got: %s", output)
	}
}

func TestLintCommand_FormatSpecifierPositionalReorderIsAllowed(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"ratio": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "%1$d of %2$d"}},
					"ja": {"stringUnit": {"state": "translated", "value": "%2$d分の%1$d"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "No issues found") {
		t.Errorf("expected no issues for a legitimate positional reorder, got: %s", output)
	}
}

func TestLintCommand_FormatSpecifierSkipsUntranslatedPlaceholder(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "%d items"}},
					"ja": {"stringUnit": {"state": "new", "value": ""}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 0)
	if strings.Contains(output, "format-specifier") {
		t.Errorf("untranslated placeholder should not be reported as a format-specifier mismatch, got: %s", output)
	}
}

func TestLintCommand_PluralMissingOther(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"count": {
				"localizations": {
					"en": {
						"variations": {
							"plural": {
								"one": {"stringUnit": {"state": "translated", "value": "%lld item"}}
							}
						}
					}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	if !strings.Contains(output, "plural-missing-other") {
		t.Errorf("expected output to contain 'plural-missing-other', got: %s", output)
	}
}

func TestLintCommand_LiteralNewlineIsWarningOnly(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"multiline": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "line one\nline two"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "literal-newline") {
		t.Errorf("expected output to contain 'literal-newline', got: %s", output)
	}
	if !strings.Contains(output, "[warning]") {
		t.Errorf("expected literal-newline to be reported as a warning, got: %s", output)
	}
}

func TestLintCommand_LanguageConsistencyCaseMismatch(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}}
				}
			},
			"farewell": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Bye"}},
					"JA": {"stringUnit": {"state": "translated", "value": "さようなら"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	if !strings.Contains(output, "language-consistency") {
		t.Errorf("expected output to contain 'language-consistency', got: %s", output)
	}
	if !strings.Contains(output, "case mismatch") {
		t.Errorf("expected output to describe a case mismatch, got: %s", output)
	}
}

func TestLintCommand_LanguageConsistencyTypoDoesNotFlagLegitimateSparseLanguage(t *testing.T) {
	// "es" is used on a single key but has no near-identical, well-established
	// language code in the catalog, so it must not be reported.
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}},
					"es": {"stringUnit": {"state": "translated", "value": "Hola"}}
				}
			},
			"farewell": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Bye"}},
					"ja": {"stringUnit": {"state": "translated", "value": "さようなら"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 0)
	if !strings.Contains(output, "No issues found") {
		t.Errorf("expected no issues for a legitimately sparse language, got: %s", output)
	}
}

func TestLintCommand_LanguageConsistencyTypoNearIdenticalCode(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"k1": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "one"}},
					"ja": {"stringUnit": {"state": "translated", "value": "1"}}
				}
			},
			"k2": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "two"}},
					"ja": {"stringUnit": {"state": "translated", "value": "2"}}
				}
			},
			"k3": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "three"}},
					"ja": {"stringUnit": {"state": "translated", "value": "3"}},
					"jp": {"stringUnit": {"state": "translated", "value": "3日"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	if !strings.Contains(output, "language-consistency") || !strings.Contains(output, `"jp"`) {
		t.Errorf("expected output to flag 'jp' as a likely typo of 'ja', got: %s", output)
	}
}

func TestLintCommand_SubstitutionStructure(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"count": {
				"localizations": {
					"en": {
						"stringUnit": {"state": "translated", "value": "no reference here"},
						"substitutions": {
							"count": {
								"argNum": 0,
								"formatSpecifier": "",
								"variations": {
									"plural": {
										"other": {"stringUnit": {"state": "translated", "value": "%arg"}}
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
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 1)
	for _, want := range []string{"substitution-structure", "argNum 0", "empty formatSpecifier", "never references"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got: %s", want, output)
		}
	}
}

func TestLintCommand_JSONOutput(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"greeting": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Complete %d Reminders"}},
					"ja": {"stringUnit": {"state": "translated", "value": "リマインダーを完了"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	output, status := runLintCommand(t, filePath, "--json")
	test.AssertEqual(t, status, 1)

	var doc struct {
		Issues []struct {
			Rule     string `json:"rule"`
			Severity string `json:"severity"`
			Key      string `json:"key"`
			Language string `json:"language"`
			Message  string `json:"message"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(output), &doc); err != nil {
		t.Fatalf("expected valid JSON output, got error %v; output: %s", err, output)
	}
	if len(doc.Issues) != 1 {
		t.Fatalf("expected exactly 1 issue, got %d: %+v", len(doc.Issues), doc.Issues)
	}
	issue := doc.Issues[0]
	test.AssertEqual(t, issue.Rule, "format-specifier")
	test.AssertEqual(t, issue.Severity, "error")
	test.AssertEqual(t, issue.Key, "greeting")
	test.AssertEqual(t, issue.Language, "ja")
}

func TestLintCommand_JSONNoIssues(t *testing.T) {
	filePath := test.FixturePath("all_translated.xcstrings")

	output, status := runLintCommand(t, filePath, "--json")
	test.AssertEqual(t, status, 0)

	var doc struct {
		Issues []interface{} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(output), &doc); err != nil {
		t.Fatalf("expected valid JSON output, got error %v; output: %s", err, output)
	}
	if len(doc.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %+v", len(doc.Issues), doc.Issues)
	}
}

func TestLintCommand_WarningOnlyExitsZero(t *testing.T) {
	content := `{
		"sourceLanguage": "en",
		"strings": {
			"multiline": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "line one\nline two"}}
				}
			}
		},
		"version": "1.0"
	}`
	filePath := test.TempFile(t, "test.xcstrings", content)

	_, status := runLintCommand(t, filePath)
	test.AssertEqual(t, status, 0)
}
