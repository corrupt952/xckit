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
		if x.IsStale(key) {
			fmt.Printf("\n%s [stale]:\n", key)
		} else {
			fmt.Printf("\n%s:\n", key)
		}
		definition := x.Strings[key]
		for _, lang := range languages {
			if localization, exists := definition.Localizations[lang]; exists {
				if localization.StringUnit != nil {
					state := localization.StringUnit.State
					value := localization.StringUnit.Value
					if value == "" {
						value = "(empty)"
					}
					fmt.Printf("  %s:\n", lang)
					fmt.Printf("    %s - %s\n", state, value)
				} else if localization.Variations != nil {
					fmt.Printf("  %s:\n", lang)
					printVariations(localization.Variations, "    ")
				} else {
					fmt.Printf("  %s: missing\n", lang)
				}
				if localization.Substitutions != nil {
					subNames := make([]string, 0, len(localization.Substitutions))
					for name := range localization.Substitutions {
						subNames = append(subNames, name)
					}
					sort.Strings(subNames)
					for _, name := range subNames {
						sub := localization.Substitutions[name]
						fmt.Printf("    substitutions.%s:\n", name)
						printVariations(&sub.Variations, "      ")
					}
				}
			} else {
				fmt.Printf("  %s: missing\n", lang)
			}
		}
	}
}

// printVariations renders plural and device variations with the given indent prefix.
func printVariations(v *xcstrings.Variations, indent string) {
	if v.Plural != nil {
		categories := make([]string, 0, len(v.Plural))
		for cat := range v.Plural {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			vv := v.Plural[cat]
			if vv == nil {
				continue
			}
			if vv.StringUnit != nil {
				fmt.Printf("%splural.%s: %s - %s\n", indent, cat, vv.StringUnit.State, vv.StringUnit.Value)
			} else if vv.Variations != nil {
				fmt.Printf("%splural.%s:\n", indent, cat)
				printVariations(vv.Variations, indent+"  ")
			}
		}
	}
	if v.Device != nil {
		devices := make([]string, 0, len(v.Device))
		for dev := range v.Device {
			devices = append(devices, dev)
		}
		sort.Strings(devices)
		for _, dev := range devices {
			vv := v.Device[dev]
			if vv == nil {
				continue
			}
			if vv.StringUnit != nil {
				fmt.Printf("%sdevice.%s: %s - %s\n", indent, dev, vv.StringUnit.State, vv.StringUnit.Value)
			} else if vv.Variations != nil {
				fmt.Printf("%sdevice.%s:\n", indent, dev)
				printVariations(vv.Variations, indent+"  ")
			}
		}
	}
}
