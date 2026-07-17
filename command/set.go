package command

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"xckit/xcstrings"

	"github.com/google/subcommands"
)

type SetCommand struct {
	XCStringsCommand
	language         string
	plural           string
	device           string
	state            string
	comment          string
	substitution     string
	argNum           int
	formatSpecifier  string
	force            bool
	allowNewLanguage bool
	stdin            bool
	requireExisting  bool
	dryRun           bool
	jsonOutput       bool
}

func (*SetCommand) Name() string {
	return "set"
}

func (*SetCommand) Synopsis() string {
	return "Set translation for a specific key and language"
}

func (*SetCommand) Usage() string {
	return "set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--state <state>] [--comment <text>] [--substitution <name> --plural <category> [--arg-num <n>] [--format-specifier <spec>]] [--force] [--require-existing] [--dry-run] [--json] <key> <value>: Set translation, creating the key if it does not yet exist\n" +
		"set [-f file.xcstrings] --stdin [--allow-new-language] [--force] [--require-existing] [--dry-run] [--json]: Apply a batch of translations from NDJSON on stdin, one object per line: {\"key\", \"lang\", \"value\", \"plural\"?, \"device\"?, \"state\"?, \"comment\"?, \"substitution\"?, \"argNum\"?, \"formatSpecifier\"?}\n" +
		"  --comment sets or updates the key's translator-facing comment (pass an empty string to clear it); it is applied together with the value and cannot be combined with --stdin. In NDJSON rows, an omitted \"comment\" field leaves the existing comment untouched, while an explicit \"comment\": \"\" clears it.\n" +
		"  --substitution <name> writes <value> into substitutions.<name>.variations.plural.<category> instead of the top-level translation; it requires --plural and does not support --device. If <name> already exists for --lang, the value is written directly. If it exists for another language of the same key, its argNum/formatSpecifier are copied when creating it for --lang. Otherwise --arg-num and --format-specifier are required, and the key's host string for --lang must already contain a %#@<name>@ reference (set it first with a plain `set`). The key must already exist.\n"
}

func (c *SetCommand) SetFlags(f *flag.FlagSet) {
	c.SetXCStringsFlags(f)
	f.StringVar(&c.language, "lang", "", "Target language code (e.g., ja, fr, de)")
	f.StringVar(&c.plural, "plural", "", "Plural category (zero, one, two, few, many, other)")
	f.StringVar(&c.device, "device", "", "Device variation (iphone, ipad, mac, appletv, applewatch, applevision, other)")
	f.StringVar(&c.state, "state", "", "extractionState applied when the key is newly created (e.g. manual). Ignored when the key already exists.")
	f.StringVar(&c.comment, "comment", "", "Set or update the key's translator-facing comment; pass an empty string to clear it. Cannot be combined with --stdin (use the NDJSON \"comment\" field instead).")
	f.StringVar(&c.substitution, "substitution", "", "Substitution name; writes the value into substitutions.<name>.variations.plural.<category>. Requires --plural; does not support --device.")
	f.IntVar(&c.argNum, "arg-num", 0, "1-based printf argument index, required together with --format-specifier when --substitution names a substitution not yet defined for any language of the key")
	f.StringVar(&c.formatSpecifier, "format-specifier", "", "printf format specifier (e.g. lld), required together with --arg-num when --substitution names a substitution not yet defined for any language of the key")
	f.BoolVar(&c.force, "force", false, "Suppress migration warning when converting plain stringUnit to variations")
	f.BoolVar(&c.allowNewLanguage, "allow-new-language", false, "Allow adding a language that is not yet present in the catalog")
	f.BoolVar(&c.stdin, "stdin", false, "Read NDJSON lines from stdin and apply them in a single batch instead of taking <key> <value> arguments")
	f.BoolVar(&c.requireExisting, "require-existing", false, "Fail instead of creating a new key when the key does not already exist")
	f.BoolVar(&c.dryRun, "dry-run", false, "Preview the changes that would be applied without writing the file")
	f.BoolVar(&c.jsonOutput, "json", false, "Output a structured JSON document describing the applied changes")
}

// setResult describes the outcome of applying a single translation.
type setResult struct {
	Key           string
	Lang          string
	Action        string // "created" or "updated"
	Path          string // variation path (e.g. "plural.other", "device.iphone"), empty for a plain stringUnit
	CommentAction string // "set", "cleared", or "" when the comment was left untouched
}

// setJSONResult is a single entry of `set --json` / `set --stdin --json` output.
type setJSONResult struct {
	Key           string `json:"key"`
	Lang          string `json:"lang"`
	Action        string `json:"action"`
	Path          string `json:"path,omitempty"`
	CommentAction string `json:"commentAction,omitempty"`
}

// setJSONSummary is the created/updated tally of `set --json` output.
type setJSONSummary struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

// setJSONOutput is the top-level document printed by `set --json`.
type setJSONOutput struct {
	Results []setJSONResult `json:"results"`
	Summary setJSONSummary  `json:"summary"`
}

func (c *SetCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	commentSet := false
	f.Visit(func(fl *flag.Flag) {
		if fl.Name == "comment" {
			commentSet = true
		}
	})

	if c.stdin {
		if commentSet {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: --comment cannot be combined with --stdin; use the \"comment\" field in NDJSON rows instead\n")
			return subcommands.ExitUsageError
		}
		return c.executeStdin(f)
	}

	if c.language == "" {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: --lang flag is required\n")
		_, _ = fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	if f.NArg() < 2 {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: key and value arguments are required\n")
		_, _ = fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	if c.plural != "" && !slices.Contains(xcstrings.ValidPluralCategories, c.plural) {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: invalid plural category '%s' (valid: zero, one, two, few, many, other)\n", c.plural)
		return subcommands.ExitUsageError
	}

	if c.device != "" && !slices.Contains(xcstrings.ValidDeviceCategories, c.device) {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: invalid device '%s' (valid: iphone, ipad, mac, appletv, applewatch, applevision, other)\n", c.device)
		return subcommands.ExitUsageError
	}

	if err := validateSubstitutionFlags(c.substitution, c.plural, c.device, c.argNum, c.formatSpecifier); err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitUsageError
	}

	key := f.Arg(0)
	value := f.Arg(1)

	xcs, err := c.LoadXCStrings()
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := c.validateLanguage(xcs); err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitUsageError
	}

	if c.requireExisting {
		if _, exists := xcs.Strings[key]; !exists {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: key '%s' does not exist (--require-existing)\n", key)
			return subcommands.ExitUsageError
		}
	}

	var comment *string
	if commentSet {
		comment = &c.comment
	}

	result, migrated, err := applySetTranslation(xcs, key, c.language, value, c.plural, c.device, c.state, comment, c.substitution, c.argNum, c.formatSpecifier)
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	if migrated && !c.force {
		fmt.Fprintf(os.Stderr, "Warning: existing plain stringUnit for key '%s' in language '%s' was migrated to variations; the original value was preserved under the 'other' fallback\n", key, c.language)
	}

	filePath := c.filePath
	if filePath == "" {
		filePath = c.findXCStringsFile()
	}

	if !c.dryRun {
		if err := xcs.SaveToFile(filePath); err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error saving file: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	if c.jsonOutput {
		out := setJSONOutput{
			Results: []setJSONResult{setJSONResult(result)},
			Summary: summarizeSetResults([]setResult{result}),
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		fmt.Println(string(data))
		return subcommands.ExitSuccess
	}

	prefix := ""
	if c.dryRun {
		prefix = "[dry-run] "
	}
	fmt.Printf("%s%s\n", prefix, setResultMessage(result))
	return subcommands.ExitSuccess
}

// setStdinRow is the NDJSON schema accepted by `set --stdin`, one object per line.
// Comment is a pointer so that an omitted field (nil) can be distinguished
// from an explicit empty string, which clears the key's existing comment.
type setStdinRow struct {
	Key             string  `json:"key"`
	Lang            string  `json:"lang"`
	Value           string  `json:"value"`
	Plural          string  `json:"plural,omitempty"`
	Device          string  `json:"device,omitempty"`
	State           string  `json:"state,omitempty"`
	Comment         *string `json:"comment,omitempty"`
	Substitution    string  `json:"substitution,omitempty"`
	ArgNum          int     `json:"argNum,omitempty"`
	FormatSpecifier string  `json:"formatSpecifier,omitempty"`
}

// stdinRowEntry pairs a parsed row with its 1-based line number for error reporting.
type stdinRowEntry struct {
	line int
	row  setStdinRow
}

func (c *SetCommand) executeStdin(f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() > 0 {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: --stdin cannot be combined with positional <key> <value> arguments\n")
		_, _ = fmt.Fprint(flag.CommandLine.Output(), c.Usage())
		return subcommands.ExitUsageError
	}

	entries, parseErrs := parseStdinRows(os.Stdin)
	if len(entries) == 0 && len(parseErrs) == 0 {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: no input lines received on stdin\n")
		return subcommands.ExitUsageError
	}

	xcs, err := c.LoadXCStrings()
	if err != nil {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	// Languages already present in the catalog (or the source language). This
	// set grows as the batch is validated so that a new language introduced by
	// an earlier line (with --allow-new-language) is accepted by later lines
	// referencing the same language, without requiring every single line to
	// pass --allow-new-language independently.
	knownLangs := append(append([]string{}, xcs.Languages()...), xcs.SourceLanguage)

	// Field-level and cross-line validation runs even when some lines failed
	// to parse as JSON, so that every problem in the batch is reported
	// together in one pass rather than requiring several fix-and-retry
	// round-trips.
	validationErrs := append([]string{}, parseErrs...)
	for _, entry := range entries {
		row := entry.row

		if row.Key == "" {
			validationErrs = append(validationErrs, fmt.Sprintf("line %d: missing required field 'key'", entry.line))
			continue
		}
		if row.Lang == "" {
			validationErrs = append(validationErrs, fmt.Sprintf("line %d: missing required field 'lang'", entry.line))
			continue
		}
		if row.Plural != "" && !slices.Contains(xcstrings.ValidPluralCategories, row.Plural) {
			validationErrs = append(validationErrs, fmt.Sprintf("line %d: invalid plural category '%s' (valid: zero, one, two, few, many, other)", entry.line, row.Plural))
			continue
		}
		if row.Device != "" && !slices.Contains(xcstrings.ValidDeviceCategories, row.Device) {
			validationErrs = append(validationErrs, fmt.Sprintf("line %d: invalid device '%s' (valid: iphone, ipad, mac, appletv, applewatch, applevision, other)", entry.line, row.Device))
			continue
		}

		if err := validateSubstitutionFlags(row.Substitution, row.Plural, row.Device, row.ArgNum, row.FormatSpecifier); err != nil {
			validationErrs = append(validationErrs, fmt.Sprintf("line %d: %v", entry.line, err))
			continue
		}

		if len(knownLangs) > 0 && !slices.Contains(knownLangs, row.Lang) {
			if c.allowNewLanguage {
				knownLangs = append(knownLangs, row.Lang)
			} else if suggestion := caseInsensitiveLanguageMatch(row.Lang, knownLangs); suggestion != "" {
				validationErrs = append(validationErrs, fmt.Sprintf("line %d: unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", entry.line, row.Lang, suggestion))
				continue
			} else if suggestion := nearestLanguageMatch(row.Lang, knownLangs); suggestion != "" {
				validationErrs = append(validationErrs, fmt.Sprintf("line %d: unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", entry.line, row.Lang, suggestion))
				continue
			} else {
				validationErrs = append(validationErrs, fmt.Sprintf("line %d: unknown language '%s' is not present in the catalog. Use --allow-new-language to add a new language", entry.line, row.Lang))
				continue
			}
		}

		if c.requireExisting {
			if _, exists := xcs.Strings[row.Key]; !exists {
				validationErrs = append(validationErrs, fmt.Sprintf("line %d: key '%s' does not exist (--require-existing)", entry.line, row.Key))
				continue
			}
		}
	}

	if len(validationErrs) > 0 {
		for _, e := range validationErrs {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %s\n", e)
		}
		return subcommands.ExitUsageError
	}

	results := make([]setResult, 0, len(entries))
	for _, entry := range entries {
		row := entry.row
		result, migrated, err := applySetTranslation(xcs, row.Key, row.Lang, row.Value, row.Plural, row.Device, row.State, row.Comment, row.Substitution, row.ArgNum, row.FormatSpecifier)
		if err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: line %d: %v\n", entry.line, err)
			return subcommands.ExitFailure
		}
		if migrated && !c.force {
			fmt.Fprintf(os.Stderr, "Warning: existing plain stringUnit for key '%s' in language '%s' was migrated to variations; the original value was preserved under the 'other' fallback\n", row.Key, row.Lang)
		}
		results = append(results, result)
	}

	filePath := c.filePath
	if filePath == "" {
		filePath = c.findXCStringsFile()
	}

	if !c.dryRun {
		if err := xcs.SaveToFile(filePath); err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error saving file: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	summary := summarizeSetResults(results)

	if c.jsonOutput {
		out := setJSONOutput{Results: make([]setJSONResult, 0, len(results)), Summary: summary}
		for _, r := range results {
			out.Results = append(out.Results, setJSONResult(r))
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
			return subcommands.ExitFailure
		}
		fmt.Println(string(data))
		return subcommands.ExitSuccess
	}

	prefix := ""
	if c.dryRun {
		prefix = "[dry-run] "
	}
	for _, r := range results {
		fmt.Printf("%s%s\n", prefix, setResultMessage(r))
	}
	fmt.Printf("%sSummary: %d created, %d updated\n", prefix, summary.Created, summary.Updated)
	return subcommands.ExitSuccess
}

// parseStdinRows reads NDJSON lines (one JSON object per line, blank lines
// skipped) from r and returns them alongside their 1-based line numbers. All
// lines are parsed before returning so that every malformed line is reported
// together rather than stopping at the first failure.
func parseStdinRows(r *os.File) ([]stdinRowEntry, []string) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var entries []stdinRowEntry
	var errs []string
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row setStdinRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			errs = append(errs, fmt.Sprintf("line %d: invalid JSON: %v", lineNum, err))
			continue
		}
		entries = append(entries, stdinRowEntry{line: lineNum, row: row})
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Sprintf("failed reading stdin: %v", err))
	}
	return entries, errs
}

// applySetTranslation applies a single key/lang/value translation (optionally
// within a plural/device variation, or within a named substitution's plural
// variation) to xcs, mirroring the plain-vs-variation-vs-substitution
// branching used by the single-shot `set` command. When comment is non-nil,
// the key's translator-facing comment is also set to *comment (an empty
// string clears it); comment == nil leaves any existing comment untouched.
// It returns the resulting setResult, whether an existing plain stringUnit
// was migrated to variations, and any error from the underlying xcstrings
// API.
func applySetTranslation(xcs *xcstrings.XCStrings, key, lang, value, plural, device, state string, comment *string, substitution string, argNum int, formatSpecifier string) (setResult, bool, error) {
	var result setResult
	var migrated bool

	switch {
	case substitution != "":
		opts := xcstrings.VariationOptions{Plural: plural}
		created, err := xcs.SetSubstitutionTranslation(key, lang, substitution, value, opts, argNum, formatSpecifier)
		if err != nil {
			return setResult{}, false, err
		}
		action := "updated"
		if created {
			action = "created"
		}
		result = setResult{Key: key, Lang: lang, Action: action, Path: substitutionPath(substitution, plural)}
	case plural != "" || device != "":
		opts := xcstrings.VariationOptions{Plural: plural, Device: device}
		m, created, err := xcs.SetVariationTranslation(key, lang, value, opts, state)
		if err != nil {
			return setResult{}, false, err
		}
		action := "updated"
		if created {
			action = "created"
		}
		result = setResult{Key: key, Lang: lang, Action: action, Path: variationPath(plural, device)}
		migrated = m
	default:
		created, err := xcs.SetTranslation(key, lang, value, state)
		if err != nil {
			return setResult{}, false, err
		}
		action := "updated"
		if created {
			action = "created"
		}
		result = setResult{Key: key, Lang: lang, Action: action}
	}

	if comment != nil {
		if err := xcs.SetComment(key, *comment); err != nil {
			return setResult{}, false, err
		}
		if *comment == "" {
			result.CommentAction = "cleared"
		} else {
			result.CommentAction = "set"
		}
	}

	return result, migrated, nil
}

// commentActionSuffix renders a short human-readable suffix describing a
// comment change alongside the translation result, or "" when the comment
// was left untouched.
func commentActionSuffix(action string) string {
	switch action {
	case "set":
		return " (comment set)"
	case "cleared":
		return " (comment cleared)"
	default:
		return ""
	}
}

// setResultMessage renders the human-readable line printed for a single
// setResult. "created" means different things depending on where the
// translation was written: for a plain or plural/device translation it means
// the key itself was newly added, while for a substitution it means the
// named substitution was newly defined for that language (the key always
// already existed, since SetSubstitutionTranslation never creates one) --
// so the wording is chosen accordingly rather than always claiming the key
// was created.
func setResultMessage(r setResult) string {
	commentSuffix := commentActionSuffix(r.CommentAction)
	isSubstitution := strings.HasPrefix(r.Path, "substitutions.")
	switch {
	case r.Action == "created" && isSubstitution:
		return fmt.Sprintf("Successfully created substitution and set translation for key '%s' in language '%s'%s", r.Key, r.Lang, commentSuffix)
	case r.Action == "created":
		return fmt.Sprintf("Successfully created key '%s' with translation for language '%s'%s", r.Key, r.Lang, commentSuffix)
	default:
		return fmt.Sprintf("Successfully set translation for key '%s' in language '%s'%s", r.Key, r.Lang, commentSuffix)
	}
}

// variationPath renders a short, human-readable path describing where within
// the localization's variation structure a translation was written.
func variationPath(plural, device string) string {
	switch {
	case device != "" && plural != "":
		return fmt.Sprintf("device.%s.plural.%s", device, plural)
	case device != "":
		return fmt.Sprintf("device.%s", device)
	case plural != "":
		return fmt.Sprintf("plural.%s", plural)
	default:
		return ""
	}
}

// substitutionPath renders the variation path reported for a translation
// written into a named substitution's plural variation.
func substitutionPath(substitution, plural string) string {
	return fmt.Sprintf("substitutions.%s.plural.%s", substitution, plural)
}

// validateSubstitutionFlags enforces the constraints `set --substitution`
// operates under: it requires --plural, does not support --device, and
// --arg-num/--format-specifier are only meaningful alongside it.
func validateSubstitutionFlags(substitution, plural, device string, argNum int, formatSpecifier string) error {
	if substitution != "" {
		if plural == "" {
			return fmt.Errorf("--substitution requires --plural")
		}
		if device != "" {
			return fmt.Errorf("--substitution does not support --device")
		}
		return nil
	}
	if argNum != 0 || formatSpecifier != "" {
		return fmt.Errorf("--arg-num and --format-specifier require --substitution")
	}
	return nil
}

// summarizeSetResults tallies created/updated counts across a batch of results.
func summarizeSetResults(results []setResult) setJSONSummary {
	var s setJSONSummary
	for _, r := range results {
		if r.Action == "created" {
			s.Created++
		} else {
			s.Updated++
		}
	}
	return s
}

// validateLanguage ensures c.language refers to a language already present in
// the catalog (or the catalog's source language), unless the catalog has no
// languages yet (nothing to compare against, so the first language addition
// is never blocked) or --allow-new-language was explicitly passed.
func (c *SetCommand) validateLanguage(xcs *xcstrings.XCStrings) error {
	existing := xcs.Languages()
	if len(existing) == 0 {
		// Nothing to validate against yet; do not block initial catalog setup.
		return nil
	}

	candidates := append(existing, xcs.SourceLanguage)
	if slices.Contains(candidates, c.language) {
		return nil
	}

	if c.allowNewLanguage {
		return nil
	}

	if suggestion := caseInsensitiveLanguageMatch(c.language, candidates); suggestion != "" {
		return fmt.Errorf("unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", c.language, suggestion)
	}

	if suggestion := nearestLanguageMatch(c.language, candidates); suggestion != "" {
		return fmt.Errorf("unknown language '%s' (did you mean %q?). Use --allow-new-language to add a new language", c.language, suggestion)
	}

	return fmt.Errorf("unknown language '%s' is not present in the catalog. Use --allow-new-language to add a new language", c.language)
}

// caseInsensitiveLanguageMatch returns the candidate that matches input
// case-insensitively (e.g. "JA" matching existing "ja"), or "" if none.
func caseInsensitiveLanguageMatch(input string, candidates []string) string {
	for _, candidate := range candidates {
		if candidate != input && strings.EqualFold(candidate, input) {
			return candidate
		}
	}
	return ""
}

// nearestLanguageMatch returns the candidate closest to input by simple edit
// distance, useful for catching typos like "jp" -> "ja". Returns "" if no
// candidate is close enough to be a plausible suggestion.
func nearestLanguageMatch(input string, candidates []string) string {
	lowerInput := strings.ToLower(input)
	best := ""
	bestDistance := -1
	for _, candidate := range candidates {
		lowerCandidate := strings.ToLower(candidate)
		distance := levenshteinDistance(lowerInput, lowerCandidate)
		threshold := 1
		if len(lowerInput) > 3 {
			threshold = 2
		}
		if distance <= threshold && (bestDistance == -1 || distance < bestDistance) {
			bestDistance = distance
			best = candidate
		}
	}
	return best
}

// levenshteinDistance computes the classic edit distance between two strings.
func levenshteinDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[len(br)]
}
