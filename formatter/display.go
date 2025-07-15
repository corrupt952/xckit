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
		fmt.Printf("\n%s:\n", key)
		definition := x.Strings[key]
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
