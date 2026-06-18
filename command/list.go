package command

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/google/subcommands"
	"xckit/formatter"
)

type ListCommand struct {
	XCStringsCommand
	prefix string
	state  string
}

func (*ListCommand) Name() string {
	return "list"
}

func (*ListCommand) Synopsis() string {
	return "List all keys with translation status"
}

func (*ListCommand) Usage() string {
	return "list [-f file.xcstrings] [--prefix <prefix>] [--state <state>]: List all keys with translation status\n"
}

func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.prefix, "prefix", "", "Filter keys by prefix")
	f.StringVar(&c.state, "state", "", "Filter keys by extractionState (e.g. manual, stale, new)")
}

func (c *ListCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcstrings, err := c.LoadXCStrings()
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	keysToShow := xcstrings.Keys()
	keysToShow = xcstrings.FilterKeysByPrefix(keysToShow, c.prefix)
	if c.state != "" {
		stateSet := make(map[string]bool, len(keysToShow))
		for _, k := range xcstrings.KeysByState(c.state) {
			stateSet[k] = true
		}
		filtered := keysToShow[:0]
		for _, k := range keysToShow {
			if stateSet[k] {
				filtered = append(filtered, k)
			}
		}
		keysToShow = filtered
	}
	sort.Strings(keysToShow)

	if len(keysToShow) == 0 {
		switch {
		case c.prefix != "" && c.state != "":
			fmt.Printf("No keys found with prefix '%s' and state '%s'\n", c.prefix, c.state)
		case c.prefix != "":
			fmt.Printf("No keys found with prefix '%s'\n", c.prefix)
		case c.state != "":
			fmt.Printf("No keys found with state '%s'\n", c.state)
		default:
			fmt.Println("No keys found")
		}
		return subcommands.ExitSuccess
	}

	switch {
	case c.prefix != "" && c.state != "":
		fmt.Printf("Keys with prefix '%s' and state '%s':\n", c.prefix, c.state)
	case c.prefix != "":
		fmt.Printf("Keys with prefix '%s':\n", c.prefix)
	case c.state != "":
		fmt.Printf("Keys with state '%s':\n", c.state)
	default:
		fmt.Println("All keys with translation status:")
	}
	formatter.DisplayKeyDetails(xcstrings, keysToShow)
	return subcommands.ExitSuccess
}
