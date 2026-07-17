package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"xckit/xcstrings"

	"github.com/google/subcommands"
)

// LintCommand statically validates an .xcstrings catalog for common
// inconsistencies (mismatched format specifiers, missing plural categories,
// empty keys, literal newlines, language-code inconsistencies, and malformed
// substitutions) that Xcode itself does not flag.
type LintCommand struct {
	XCStringsCommand
	jsonOutput bool
}

func (*LintCommand) Name() string {
	return "lint"
}

func (*LintCommand) Synopsis() string {
	return "Statically validate a catalog for common inconsistencies"
}

func (*LintCommand) Usage() string {
	return "lint [-f file.xcstrings] [--json]: Detect format-specifier mismatches, missing plural categories, empty keys, literal newlines, language-code inconsistencies, and malformed substitutions\n"
}

func (c *LintCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.BoolVar(&c.jsonOutput, "json", false, "Output a single JSON document to stdout instead of human-readable text")
}

// lintSeverity is the severity level of a lint issue.
type lintSeverity string

const (
	lintSeverityError   lintSeverity = "error"
	lintSeverityWarning lintSeverity = "warning"
)

// lintIssue is a single detected inconsistency.
type lintIssue struct {
	Rule     string
	Severity lintSeverity
	Key      string
	Language string // empty when not language-specific
	Path     string // empty when not path-specific (e.g. "plural.other", "substitutions.files.plural.one")
	Message  string
}

func (c *LintCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	xcs, err := c.LoadXCStrings()
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	issues := runLint(xcs)

	if c.jsonOutput {
		return c.printJSON(issues)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return subcommands.ExitSuccess
	}

	for _, issue := range issues {
		fmt.Println(formatLintIssue(issue))
	}

	return exitStatusForLintIssues(issues)
}

// exitStatusForLintIssues returns ExitFailure when at least one error-level
// issue is present. A catalog with only warning-level issues exits
// successfully so warnings don't break CI on their own.
func exitStatusForLintIssues(issues []lintIssue) subcommands.ExitStatus {
	for _, issue := range issues {
		if issue.Severity == lintSeverityError {
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}

// formatLintIssue renders a single issue as "rule: key > lang > path: message".
// The language and path segments are omitted when not applicable.
func formatLintIssue(issue lintIssue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s: %s", issue.Severity, issue.Rule, issue.Key)
	if issue.Language != "" {
		fmt.Fprintf(&b, " > %s", issue.Language)
	}
	if issue.Path != "" {
		fmt.Fprintf(&b, " > %s", issue.Path)
	}
	fmt.Fprintf(&b, ": %s", issue.Message)
	return b.String()
}

// lintJSONOutput is the top-level document printed by `lint --json`.
type lintJSONOutput struct {
	Issues []lintJSONIssue `json:"issues"`
}

// lintJSONIssue is a single issue in `lint --json`.
type lintJSONIssue struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Key      string `json:"key"`
	Language string `json:"language,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

func (c *LintCommand) printJSON(issues []lintIssue) subcommands.ExitStatus {
	out := lintJSONOutput{Issues: make([]lintJSONIssue, 0, len(issues))}
	for _, issue := range issues {
		out.Issues = append(out.Issues, lintJSONIssue{
			Rule:     issue.Rule,
			Severity: string(issue.Severity),
			Key:      issue.Key,
			Language: issue.Language,
			Path:     issue.Path,
			Message:  issue.Message,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Println(string(data))
	return exitStatusForLintIssues(issues)
}

// runLint walks the whole catalog and returns every detected issue, sorted
// by key, then language, then path, then rule for deterministic output.
func runLint(xcs *xcstrings.XCStrings) []lintIssue {
	var issues []lintIssue

	for key, def := range xcs.Strings {
		issues = append(issues, lintEmptyKey(key)...)
		issues = append(issues, lintKey(xcs, key, def)...)
	}

	issues = append(issues, lintLanguageConsistency(xcs)...)

	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.Language != b.Language {
			return a.Language < b.Language
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Rule < b.Rule
	})

	return issues
}

// lintEmptyKey flags the empty-string key.
func lintEmptyKey(key string) []lintIssue {
	if key != "" {
		return nil
	}
	return []lintIssue{{
		Rule:     "empty-key",
		Severity: lintSeverityError,
		Key:      key,
		Message:  "catalog contains an empty string key",
	}}
}

// lintKey runs the per-localization rules (plural-missing-other,
// literal-newline, substitution-structure, format-specifier) for a single key.
func lintKey(xcs *xcstrings.XCStrings, key string, def xcstrings.StringDefinition) []lintIssue {
	var issues []lintIssue

	sourceLeaves := map[string]lintLeaf{}
	if srcLoc, ok := def.Localizations[xcs.SourceLanguage]; ok {
		for _, leaf := range collectLintLeaves(srcLoc) {
			sourceLeaves[leaf.Path] = leaf
		}
	}

	for lang, loc := range def.Localizations {
		issues = append(issues, lintPluralMissingOther(key, lang, loc)...)
		issues = append(issues, lintSubstitutionStructure(key, lang, loc)...)

		leaves := collectLintLeaves(loc)
		for _, leaf := range leaves {
			if hasLiteralNewline(leaf.Value) {
				issues = append(issues, lintIssue{
					Rule:     "literal-newline",
					Severity: lintSeverityWarning,
					Key:      key,
					Language: lang,
					Path:     leaf.Path,
					Message:  "value contains a literal newline character",
				})
			}
		}

		if lang == xcs.SourceLanguage {
			continue
		}
		for _, leaf := range leaves {
			// An untranslated placeholder (state != "translated", typically
			// empty) is already reported by the `untranslated` command; don't
			// also report it as a format-specifier mismatch here.
			if leaf.State != "translated" {
				continue
			}
			srcLeaf, ok := sourceLeaves[leaf.Path]
			if !ok {
				continue
			}
			if msg, mismatched := compareFormatSpecifiers(srcLeaf.Value, leaf.Value); mismatched {
				issues = append(issues, lintIssue{
					Rule:     "format-specifier",
					Severity: lintSeverityError,
					Key:      key,
					Language: lang,
					Path:     leaf.Path,
					Message:  msg,
				})
			}
		}
	}

	return issues
}

func hasLiteralNewline(s string) bool {
	return strings.ContainsAny(s, "\n\r")
}

// lintLeaf is a single leaf string unit reachable from a Localization, along
// with the path used to locate the matching leaf in another language.
type lintLeaf struct {
	Path  string
	Value string
	State string
}

// collectLintLeaves gathers every leaf StringUnit reachable from a
// Localization (the top-level unit, plural/device variations at any nesting
// depth, and substitutions) together with a stable path string. The path
// scheme matches xcstrings.UntranslatedDetail.Path, except the top-level
// stringUnit is given the explicit path "stringUnit" instead of "".
func collectLintLeaves(l xcstrings.Localization) []lintLeaf {
	var leaves []lintLeaf
	if l.StringUnit != nil {
		leaves = append(leaves, lintLeaf{Path: "stringUnit", Value: l.StringUnit.Value, State: l.StringUnit.State})
	}
	if l.Variations != nil {
		leaves = append(leaves, collectVariationLeaves(l.Variations, "")...)
	}
	for name, sub := range l.Substitutions {
		leaves = append(leaves, collectVariationLeaves(&sub.Variations, "substitutions."+name)...)
	}
	return leaves
}

func joinLintPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func collectVariationLeaves(v *xcstrings.Variations, prefix string) []lintLeaf {
	var leaves []lintLeaf
	for _, cat := range sortedKeys(v.Plural) {
		vv := v.Plural[cat]
		if vv == nil {
			continue
		}
		path := joinLintPath(prefix, "plural."+cat)
		if vv.StringUnit != nil {
			leaves = append(leaves, lintLeaf{Path: path, Value: vv.StringUnit.Value, State: vv.StringUnit.State})
		}
		if vv.Variations != nil {
			leaves = append(leaves, collectVariationLeaves(vv.Variations, path)...)
		}
	}
	for _, dev := range sortedKeys(v.Device) {
		vv := v.Device[dev]
		if vv == nil {
			continue
		}
		path := joinLintPath(prefix, "device."+dev)
		if vv.StringUnit != nil {
			leaves = append(leaves, lintLeaf{Path: path, Value: vv.StringUnit.Value, State: vv.StringUnit.State})
		}
		if vv.Variations != nil {
			leaves = append(leaves, collectVariationLeaves(vv.Variations, path)...)
		}
	}
	return leaves
}

func sortedKeys(m map[string]*xcstrings.VariationValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// lintPluralMissingOther walks every Plural variation map reachable from a
// Localization (at any nesting depth, including inside substitutions) and
// flags the ones missing the mandatory "other" category.
func lintPluralMissingOther(key, lang string, l xcstrings.Localization) []lintIssue {
	var issues []lintIssue
	if l.Variations != nil {
		issues = append(issues, pluralMissingOtherInVariations(key, lang, l.Variations, "")...)
	}
	for name, sub := range l.Substitutions {
		issues = append(issues, pluralMissingOtherInVariations(key, lang, &sub.Variations, "substitutions."+name)...)
	}
	return issues
}

func pluralMissingOtherInVariations(key, lang string, v *xcstrings.Variations, prefix string) []lintIssue {
	var issues []lintIssue
	if v.Plural != nil {
		if _, ok := v.Plural["other"]; !ok {
			issues = append(issues, lintIssue{
				Rule:     "plural-missing-other",
				Severity: lintSeverityError,
				Key:      key,
				Language: lang,
				Path:     joinLintPath(prefix, "plural"),
				Message:  "plural variation is missing the required 'other' category",
			})
		}
		for _, cat := range sortedKeys(v.Plural) {
			vv := v.Plural[cat]
			if vv != nil && vv.Variations != nil {
				issues = append(issues, pluralMissingOtherInVariations(key, lang, vv.Variations, joinLintPath(prefix, "plural."+cat))...)
			}
		}
	}
	for _, dev := range sortedKeys(v.Device) {
		vv := v.Device[dev]
		if vv != nil && vv.Variations != nil {
			issues = append(issues, pluralMissingOtherInVariations(key, lang, vv.Variations, joinLintPath(prefix, "device."+dev))...)
		}
	}
	return issues
}

// lintSubstitutionStructure flags malformed substitution definitions: a
// missing argNum, an empty formatSpecifier, or a substitution whose name is
// never referenced (as %#@name@) by any top-level string in the same
// localization.
func lintSubstitutionStructure(key, lang string, l xcstrings.Localization) []lintIssue {
	if len(l.Substitutions) == 0 {
		return nil
	}

	var hostValues []lintLeaf
	if l.Variations != nil {
		hostValues = collectVariationLeaves(l.Variations, "")
	}
	var hostText strings.Builder
	if l.StringUnit != nil {
		hostText.WriteString(l.StringUnit.Value)
		hostText.WriteByte('\n')
	}
	for _, leaf := range hostValues {
		hostText.WriteString(leaf.Value)
		hostText.WriteByte('\n')
	}

	var issues []lintIssue
	for _, name := range sortedSubstitutionNames(l.Substitutions) {
		sub := l.Substitutions[name]
		if sub.ArgNum == 0 {
			issues = append(issues, lintIssue{
				Rule:     "substitution-structure",
				Severity: lintSeverityError,
				Key:      key,
				Language: lang,
				Path:     "substitutions." + name,
				Message:  "substitution has argNum 0 (unset or invalid)",
			})
		}
		if strings.TrimSpace(sub.FormatSpecifier) == "" {
			issues = append(issues, lintIssue{
				Rule:     "substitution-structure",
				Severity: lintSeverityError,
				Key:      key,
				Language: lang,
				Path:     "substitutions." + name,
				Message:  "substitution has an empty formatSpecifier",
			})
		}
		if !xcstrings.HostReferencesSubstitution(hostText.String(), name) {
			issues = append(issues, lintIssue{
				Rule:     "substitution-structure",
				Severity: lintSeverityError,
				Key:      key,
				Language: lang,
				Path:     "substitutions." + name,
				Message:  fmt.Sprintf("host string never references %%#@%s@", name),
			})
		}
	}
	return issues
}

func sortedSubstitutionNames(m map[string]xcstrings.Substitution) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// lintLanguageConsistency flags language codes that only differ by case
// (e.g. "ja" and "JA" both present) and language codes that appear on a
// single key in an otherwise multi-key, multi-language catalog -- a common
// symptom of a typo'd language code.
func lintLanguageConsistency(xcs *xcstrings.XCStrings) []lintIssue {
	keyCount := map[string]int{}
	byLower := map[string]map[string]bool{}

	for _, def := range xcs.Strings {
		for lang := range def.Localizations {
			keyCount[lang]++
			lower := strings.ToLower(lang)
			if byLower[lower] == nil {
				byLower[lower] = map[string]bool{}
			}
			byLower[lower][lang] = true
		}
	}

	var issues []lintIssue

	for _, lower := range sortedStringKeys(byLower) {
		variants := byLower[lower]
		if len(variants) <= 1 {
			continue
		}
		variantList := make([]string, 0, len(variants))
		for v := range variants {
			variantList = append(variantList, v)
		}
		sort.Strings(variantList)
		for _, lang := range variantList {
			issues = append(issues, lintIssue{
				Rule:     "language-consistency",
				Severity: lintSeverityError,
				Key:      "*",
				Language: lang,
				Message:  fmt.Sprintf("language code case mismatch: %s also appears as %s", lang, strings.Join(variantList, ", ")),
			})
		}
	}

	// A language code used on exactly one key, when a near-identical code
	// (edit distance 1) is used consistently across many keys, is a strong
	// typo signal (e.g. "jp" once alongside "ja" everywhere else). A rare but
	// intentionally used language (e.g. a single key translated into "es"
	// while no other code resembles it) is not flagged: comparing only
	// against near-identical, well-established codes keeps this from firing
	// on legitimately sparse languages.
	var establishedLangs []string
	for _, lang := range sortedStringIntKeys(keyCount) {
		if lang != xcs.SourceLanguage && keyCount[lang] > 1 {
			establishedLangs = append(establishedLangs, lang)
		}
	}
	for _, lang := range sortedStringIntKeys(keyCount) {
		if lang == xcs.SourceLanguage || keyCount[lang] != 1 {
			continue
		}
		for _, other := range establishedLangs {
			if other == lang {
				continue
			}
			if levenshteinDistance(strings.ToLower(lang), strings.ToLower(other)) == 1 {
				issues = append(issues, lintIssue{
					Rule:     "language-consistency",
					Severity: lintSeverityError,
					Key:      "*",
					Language: lang,
					Message:  fmt.Sprintf("language %q appears on only 1 key and closely resembles %q (used on %d keys); possible typo'd language code", lang, other, keyCount[other]),
				})
				break
			}
		}
	}

	return issues
}

func sortedStringKeys(m map[string]map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringIntKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- format specifier extraction & comparison ---

var (
	lintSubRefRe  = regexp.MustCompile(`%(?:\d+\$)?#@(\w+)@`)
	lintEscapedRe = regexp.MustCompile(`%%`)
	lintArgRe     = regexp.MustCompile(`%(\d+\$)?arg\b`)
	lintStdSpecRe = regexp.MustCompile(`%(\d+\$)?[-+ 0#']*\d*(?:\.\d+)?(hh|h|ll|l|q|L|z|j|t)?([@dioxXucsfeEgGaAp])`)
)

// formatToken is a single parsed format reference within a string: either a
// printf-style conversion (kind is the conversion character, e.g. "d"),
// Apple's substitution "%arg" placeholder (kind "arg"), or a %#@name@
// substitution reference (kind "sub:<name>").
type formatToken struct {
	start    int
	position int // 1-based; explicit via "N$", or auto-assigned by order of appearance
	kind     string
}

// extractFormatTokens parses every format reference out of s. Substitution
// references and %% escapes are masked out before the generic conversion
// regex runs so they can't be mis-parsed as a "%@"/"%%" conversion.
func extractFormatTokens(s string) []formatToken {
	masked := []byte(s)
	var tokens []formatToken

	for _, m := range lintSubRefRe.FindAllStringSubmatchIndex(s, -1) {
		tokens = append(tokens, formatToken{start: m[0], position: -1, kind: "sub:" + s[m[2]:m[3]]})
		maskRange(masked, m[0], m[1])
	}
	for _, m := range lintEscapedRe.FindAllStringIndex(string(masked), -1) {
		maskRange(masked, m[0], m[1])
	}
	for _, m := range lintArgRe.FindAllStringSubmatchIndex(string(masked), -1) {
		tokens = append(tokens, formatToken{start: m[0], position: explicitPosition(s, m[2], m[3]), kind: "arg"})
		maskRange(masked, m[0], m[1])
	}
	for _, m := range lintStdSpecRe.FindAllStringSubmatchIndex(string(masked), -1) {
		conv := s[m[6]:m[7]]
		tokens = append(tokens, formatToken{start: m[0], position: explicitPosition(s, m[2], m[3]), kind: conv})
		maskRange(masked, m[0], m[1])
	}

	sort.Slice(tokens, func(i, j int) bool { return tokens[i].start < tokens[j].start })

	auto := 1
	for i := range tokens {
		if tokens[i].position == 0 {
			tokens[i].position = auto
			auto++
		}
	}

	return tokens
}

// explicitPosition returns the "N$" position captured at s[start:end], or 0
// when the capture group did not participate (no explicit position given).
func explicitPosition(s string, start, end int) int {
	if start < 0 || end < 0 || start >= end {
		return 0
	}
	raw := strings.TrimSuffix(s[start:end], "$")
	n := 0
	for _, r := range raw {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return 0
	}
	return n
}

func maskRange(b []byte, start, end int) {
	for i := start; i < end && i < len(b); i++ {
		b[i] = ' '
	}
}

// compareFormatSpecifiers compares the format references found in source and
// target and reports whether they're inconsistent (missing/extra positional
// argument, an argument whose type changed, or a substitution reference that
// only appears on one side). Positional specifiers (%1$d) may legitimately
// be reordered by a translation, so only the position -> kind mapping is
// compared, not textual order.
func compareFormatSpecifiers(source, target string) (string, bool) {
	srcTokens := extractFormatTokens(source)
	tgtTokens := extractFormatTokens(target)

	srcPos := map[int]string{}
	srcSubs := map[string]bool{}
	for _, t := range srcTokens {
		if t.position == -1 {
			srcSubs[t.kind] = true
			continue
		}
		srcPos[t.position] = t.kind
	}
	tgtPos := map[int]string{}
	tgtSubs := map[string]bool{}
	for _, t := range tgtTokens {
		if t.position == -1 {
			tgtSubs[t.kind] = true
			continue
		}
		tgtPos[t.position] = t.kind
	}

	if len(srcPos) == 0 && len(srcSubs) == 0 && len(tgtPos) == 0 && len(tgtSubs) == 0 {
		return "", false
	}

	var problems []string

	for pos, kind := range srcPos {
		tKind, ok := tgtPos[pos]
		if !ok {
			problems = append(problems, fmt.Sprintf("missing %%%s (argument %d)", kind, pos))
		} else if tKind != kind {
			problems = append(problems, fmt.Sprintf("argument %d changed type: %%%s -> %%%s", pos, kind, tKind))
		}
	}
	for pos, kind := range tgtPos {
		if _, ok := srcPos[pos]; !ok {
			problems = append(problems, fmt.Sprintf("unexpected %%%s (argument %d) not present in source", kind, pos))
		}
	}
	for name := range srcSubs {
		if !tgtSubs[name] {
			problems = append(problems, fmt.Sprintf("missing substitution reference %%#@%s@", strings.TrimPrefix(name, "sub:")))
		}
	}
	for name := range tgtSubs {
		if !srcSubs[name] {
			problems = append(problems, fmt.Sprintf("unexpected substitution reference %%#@%s@ not present in source", strings.TrimPrefix(name, "sub:")))
		}
	}

	if len(problems) == 0 {
		return "", false
	}

	sort.Strings(problems)
	return strings.Join(problems, "; "), true
}
