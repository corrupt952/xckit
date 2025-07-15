package command

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"xckit/helper/test"
)

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestVersionCommand_Execute(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedOutput string
	}{
		{
			name:           "version set",
			version:        "1.0.0",
			expectedOutput: "1.0.0",
		},
		{
			name:           "version not set",
			version:        "",
			expectedOutput: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original version
			originalVersion := Version
			defer func() {
				Version = originalVersion
			}()

			Version = tt.version

			cmd := &VersionCommand{}

			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			cmd.SetFlags(flagSet)
			err := flagSet.Parse([]string{})
			test.AssertNoError(t, err)

			output := captureOutput(func() {
				status := cmd.Execute(context.Background(), flagSet)
				test.AssertEqual(t, int(status), 0)
			})

			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("expected output to contain %q, got: %q", tt.expectedOutput, output)
			}
		})
	}
}

func TestVersionCommand_Metadata(t *testing.T) {
	cmd := &VersionCommand{}

	test.AssertEqual(t, cmd.Name(), "version")
	test.AssertEqual(t, cmd.Synopsis(), "Print xckit version")

	usage := cmd.Usage()
	if !strings.Contains(usage, "version") {
		t.Errorf("usage should contain 'version', got: %q", usage)
	}
}
