package command

import (
	"flag"
	"fmt"
	"os"

	"xckit/xcstrings"
)

type XCStringsCommand struct {
	filePath string
}

func (c *XCStringsCommand) SetXCStringsFlags(f *flag.FlagSet) {
	f.StringVar(&c.filePath, "f", "", "Path to the .xcstrings file")
	f.StringVar(&c.filePath, "file", "", "Path to the .xcstrings file")
}

func (c *XCStringsCommand) LoadXCStrings() (*xcstrings.XCStrings, error) {
	path := c.filePath
	if path == "" {
		path = c.findXCStringsFile()
		if path == "" {
			return nil, fmt.Errorf("no .xcstrings file found. Use -f flag to specify the file path")
		}
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	return xcstrings.Load(path)
}

func (c *XCStringsCommand) findXCStringsFile() string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 10 && entry.Name()[len(entry.Name())-10:] == ".xcstrings" {
			return entry.Name()
		}
	}
	return ""
}
