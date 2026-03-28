package formatter

import (
	"fmt"
	"sort"

	"xckit/xcstrings"
)

func DisplayKeyDetails(x *xcstrings.XCStrings, keys []string) {
	languages := x.Languages()
	sort.Strings(languages)

	for _, key := range keys {
		definition := x.Strings[key]
		if definition.ExtractionState == "stale" {
			fmt.Printf("\n%s [stale]:\n", key)
		} else {
			fmt.Printf("\n%s:\n", key)
		}
		for _, lang := range languages {
			if localization, exists := definition.Localizations[lang]; exists {
				state := localization.StringUnit.State
				value := localization.StringUnit.Value
				if value == "" {
					value = "(empty)"
				}
				fmt.Printf("  %s: %s - %s\n", lang, state, value)
			} else {
				fmt.Printf("  %s: missing\n", lang)
			}
		}
	}
}
