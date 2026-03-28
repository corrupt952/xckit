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

		// Build the full list of languages to display: non-source languages
		// from the catalog plus any language in this key's localizations that
		// has variations (which may include the source language).
		langSet := make(map[string]bool)
		for _, l := range languages {
			langSet[l] = true
		}
		for l, loc := range definition.Localizations {
			if loc.Variations != nil || loc.Substitutions != nil {
				langSet[l] = true
			}
		}
		allLangs := make([]string, 0, len(langSet))
		for l := range langSet {
			allLangs = append(allLangs, l)
		}
		sort.Strings(allLangs)

		for _, lang := range allLangs {
			if localization, exists := definition.Localizations[lang]; exists {
				if localization.StringUnit != nil {
					displayStringUnit(lang, "", localization.StringUnit)
				} else if localization.Variations != nil {
					displayVariations(lang, localization.Variations)
				} else {
					fmt.Printf("  %s: (no content)\n", lang)
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
						printSubstitutionVariations(&sub.Variations, "      ")
					}
				}
			} else {
				fmt.Printf("  %s: missing\n", lang)
			}
		}
	}
}

func displayStringUnit(lang, prefix string, su *xcstrings.StringUnit) {
	state := su.State
	value := su.Value
	if value == "" {
		value = "(empty)"
	}
	if prefix != "" {
		fmt.Printf("  %s:\n    %s: %s - %s\n", lang, prefix, state, value)
	} else {
		fmt.Printf("  %s: %s - %s\n", lang, state, value)
	}
}

func displayVariations(lang string, v *xcstrings.Variations) {
	if v.Device != nil {
		displayDeviceVariations(lang, v.Device)
	}
	if v.Plural != nil {
		if v.Device != nil {
			// Language header already printed by displayDeviceVariations
			displayPluralVariations("", "    ", v.Plural)
		} else {
			displayPluralVariations(lang, "", v.Plural)
		}
	}
}

func displayDeviceVariations(lang string, device map[string]*xcstrings.VariationValue) {
	deviceNames := make([]string, 0, len(device))
	for name := range device {
		deviceNames = append(deviceNames, name)
	}
	sort.Strings(deviceNames)

	fmt.Printf("  %s:\n", lang)
	for _, name := range deviceNames {
		vv := device[name]
		if vv == nil {
			continue
		}
		if vv.StringUnit != nil {
			state := vv.StringUnit.State
			value := vv.StringUnit.Value
			if value == "" {
				value = "(empty)"
			}
			fmt.Printf("    device.%s: %s - %s\n", name, state, value)
		} else if vv.Variations != nil && vv.Variations.Plural != nil {
			displayPluralVariations("", fmt.Sprintf("    device.%s", name), vv.Variations.Plural)
		}
	}
}

func displayPluralVariations(lang, prefix string, plural map[xcstrings.PluralCategory]*xcstrings.VariationValue) {
	categories := make([]string, 0, len(plural))
	for cat := range plural {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	if lang != "" {
		fmt.Printf("  %s:\n", lang)
	}
	for _, cat := range categories {
		vv := plural[cat]
		if vv == nil || vv.StringUnit == nil {
			continue
		}
		state := vv.StringUnit.State
		value := vv.StringUnit.Value
		if value == "" {
			value = "(empty)"
		}
		if prefix != "" {
			fmt.Printf("%s.plural.%s: %s - %s\n", prefix, cat, state, value)
		} else {
			fmt.Printf("    plural.%s: %s - %s\n", cat, state, value)
		}
	}
}

// printSubstitutionVariations renders substitution variation leaves with the given indent.
func printSubstitutionVariations(v *xcstrings.Variations, indent string) {
	if v.Plural != nil {
		categories := make([]string, 0, len(v.Plural))
		for cat := range v.Plural {
			categories = append(categories, cat)
		}
		sort.Strings(categories)
		for _, cat := range categories {
			vv := v.Plural[cat]
			if vv == nil || vv.StringUnit == nil {
				continue
			}
			fmt.Printf("%splural.%s: %s - %s\n", indent, cat, vv.StringUnit.State, vv.StringUnit.Value)
		}
	}
}
