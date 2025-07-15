package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"xckit/helper/test"
)

// buildXckit builds the xckit binary for testing
func buildXckit(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "xckit")

	cmd := exec.Command("go", "build", "-o", binaryPath, "..")
	err := cmd.Run()
	test.AssertNoError(t, err)

	return binaryPath
}

// runXckit runs the xckit binary with given arguments
func runXckit(t *testing.T, binaryPath string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			t.Fatalf("failed to run xckit: %v", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

func TestIntegration_Version(t *testing.T) {
	binaryPath := buildXckit(t)

	stdout, stderr, exitCode := runXckit(t, binaryPath, "version")

	test.AssertEqual(t, exitCode, 0)
	test.AssertEqual(t, stderr, "")

	// Should output version (either "dev" or actual version)
	if !strings.Contains(stdout, "dev") && !strings.Contains(stdout, "0.1.0") {
		t.Errorf("expected version output, got: %q", stdout)
	}
}

func TestIntegration_Help(t *testing.T) {
	binaryPath := buildXckit(t)

	stdout, stderr, exitCode := runXckit(t, binaryPath, "help")

	test.AssertEqual(t, exitCode, 0)
	test.AssertEqual(t, stderr, "")

	// Should contain subcommands
	expectedCommands := []string{"list", "untranslated", "set", "status", "version"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("help output should contain %q, got: %q", cmd, stdout)
		}
	}
}

func TestIntegration_Status(t *testing.T) {
	// Create test xcstrings file
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"hello": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Hello"}},
					"ja": {"stringUnit": {"state": "translated", "value": "こんにちは"}},
					"es": {"stringUnit": {"state": "new", "value": "Hola"}}
				}
			},
			"goodbye": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Goodbye"}}
				}
			}
		},
		"version": "1.0"
	}`

	tmpDir := t.TempDir()
	xcstringsPath := filepath.Join(tmpDir, "test.xcstrings")
	err := os.WriteFile(xcstringsPath, []byte(testContent), 0644)
	test.AssertNoError(t, err)

	binaryPath := buildXckit(t)

	stdout, stderr, exitCode := runXckit(t, binaryPath, "status", "-f", xcstringsPath)

	test.AssertEqual(t, exitCode, 0)
	test.AssertEqual(t, stderr, "")

	// Should contain status information
	expectedContent := []string{
		"Translation Status",
		"Source Language: en",
		"Total Keys: 2",
		"Languages:",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(stdout, expected) {
			t.Errorf("status output should contain %q, got: %q", expected, stdout)
		}
	}
}

func TestIntegration_Untranslated(t *testing.T) {
	testContent := `{
		"sourceLanguage": "en",
		"strings": {
			"translated_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Translated"}},
					"ja": {"stringUnit": {"state": "translated", "value": "翻訳済み"}}
				}
			},
			"untranslated_key": {
				"localizations": {
					"en": {"stringUnit": {"state": "translated", "value": "Untranslated"}},
					"ja": {"stringUnit": {"state": "new", "value": "未翻訳"}}
				}
			}
		},
		"version": "1.0"
	}`

	tmpDir := t.TempDir()
	xcstringsPath := filepath.Join(tmpDir, "test.xcstrings")
	err := os.WriteFile(xcstringsPath, []byte(testContent), 0644)
	test.AssertNoError(t, err)

	binaryPath := buildXckit(t)

	tests := []struct {
		name         string
		args         []string
		expectedKeys []string
	}{
		{
			name:         "untranslated for specific language",
			args:         []string{"untranslated", "-f", xcstringsPath, "--lang", "ja"},
			expectedKeys: []string{"untranslated_key"},
		},
		{
			name:         "untranslated for all languages",
			args:         []string{"untranslated", "-f", xcstringsPath},
			expectedKeys: []string{"untranslated_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runXckit(t, binaryPath, tt.args...)

			test.AssertEqual(t, exitCode, 0)
			test.AssertEqual(t, stderr, "")

			for _, expectedKey := range tt.expectedKeys {
				if !strings.Contains(stdout, expectedKey) {
					t.Errorf("untranslated output should contain %q, got: %q", expectedKey, stdout)
				}
			}
		})
	}
}

func TestIntegration_Set(t *testing.T) {
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

	tmpDir := t.TempDir()
	xcstringsPath := filepath.Join(tmpDir, "test.xcstrings")
	err := os.WriteFile(xcstringsPath, []byte(testContent), 0644)
	test.AssertNoError(t, err)

	binaryPath := buildXckit(t)

	// Set a new translation
	stdout, stderr, exitCode := runXckit(t, binaryPath, "set", "-f", xcstringsPath, "--lang", "ja", "test_key", "テスト")

	test.AssertEqual(t, exitCode, 0)
	test.AssertEqual(t, stderr, "")

	if !strings.Contains(stdout, "Successfully set translation") {
		t.Errorf("set output should contain success message, got: %q", stdout)
	}

	// Verify the translation was set by checking the file
	stdout, stderr, exitCode = runXckit(t, binaryPath, "list", "-f", xcstringsPath)

	test.AssertEqual(t, exitCode, 0)
	test.AssertEqual(t, stderr, "")

	if !strings.Contains(stdout, "テスト") {
		t.Errorf("list output should contain new translation, got: %q", stdout)
	}
}

func TestIntegration_List(t *testing.T) {
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
			}
		},
		"version": "1.0"
	}`

	tmpDir := t.TempDir()
	xcstringsPath := filepath.Join(tmpDir, "test.xcstrings")
	err := os.WriteFile(xcstringsPath, []byte(testContent), 0644)
	test.AssertNoError(t, err)

	binaryPath := buildXckit(t)

	tests := []struct {
		name         string
		args         []string
		expectedKeys []string
	}{
		{
			name:         "list all keys",
			args:         []string{"list", "-f", xcstringsPath},
			expectedKeys: []string{"key1:", "key2:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runXckit(t, binaryPath, tt.args...)

			test.AssertEqual(t, exitCode, 0)
			test.AssertEqual(t, stderr, "")

			for _, expectedKey := range tt.expectedKeys {
				if !strings.Contains(stdout, expectedKey) {
					t.Errorf("list output should contain %q, got: %q", expectedKey, stdout)
				}
			}
		})
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	binaryPath := buildXckit(t)

	tests := []struct {
		name             string
		args             []string
		expectedExitCode int
	}{
		{
			name:             "file not found",
			args:             []string{"status", "-f", "nonexistent.xcstrings"},
			expectedExitCode: 1,
		},
		{
			name:             "set without language",
			args:             []string{"set", "key", "value"},
			expectedExitCode: 2, // Usage error
		},
		{
			name:             "set without enough arguments",
			args:             []string{"set", "--lang", "ja", "key"},
			expectedExitCode: 2, // Usage error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, exitCode := runXckit(t, binaryPath, tt.args...)

			test.AssertEqual(t, exitCode, tt.expectedExitCode)
		})
	}
}
