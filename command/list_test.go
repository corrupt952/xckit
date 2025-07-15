package command

import (
	"context"
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
			expectedKeys:   []string{"key1:", "key2:"},
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

			for _, expectedKey := range tt.expectedKeys {
				if !strings.Contains(output, expectedKey) {
					t.Errorf("output should contain %q, got: %q", expectedKey, output)
				}
			}
		})
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
