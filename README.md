# xckit

CLI tool for managing Xcode String Catalogs (.xcstrings).

---

## Key Features

- List, filter, and inspect translation keys
- Detect untranslated keys with variation-level detail (`--detail` flag)
- Set translations with plural/device variation support (`--plural`, `--device` flags)
- Translation progress tracking with key-level and string-unit-level counting
- CSV export/import for spreadsheet-based translation workflows
- Full support for plural, device, nested, and substitution variations
- `needs_review` and `stale` state recognition
- Stale key management (list, remove, dry-run)
- Atomic file writes for data safety
- Single Go binary — no Xcode required, works on Linux CI

---

## Installation

### go install

```bash
go install github.com/corrupt952/xckit@latest
```

### GitHub Releases

Download a prebuilt binary from the [Releases](https://github.com/corrupt952/xckit/releases) page.

### Build from source

```bash
git clone https://github.com/corrupt952/xckit.git
cd xckit
make build
```

---

## Quick Start

```bash
# View translation progress
xckit status -f MyApp.xcstrings

# List all keys with translation status
xckit list -f MyApp.xcstrings

# Find untranslated keys for Japanese
xckit untranslated --lang ja -f MyApp.xcstrings

# Set a simple translation
xckit set --lang ja "hello_world" "こんにちは世界" -f MyApp.xcstrings

# Set a plural variation
xckit set --lang ja --plural other "item_count" "%lld 個のアイテム" -f MyApp.xcstrings

# Export to CSV for external translation
xckit export --format csv -f MyApp.xcstrings -o translations.csv

# Import translations from CSV
xckit import --format csv -f MyApp.xcstrings translations.csv
```

---

## Commands Reference

| Command        | Description                                              |
|----------------|----------------------------------------------------------|
| `list`         | List all keys with translation status                    |
| `untranslated` | Find keys that need translation                          |
| `set`          | Set a translation, creating the key if missing           |
| `remove`       | Remove a key by name or by extractionState               |
| `status`       | Show translation progress summary per language           |
| `export`       | Export strings to CSV                                    |
| `import`       | Import translations from CSV                             |
| `stale`        | List or remove stale keys                                |
| `lint`         | Statically validate the catalog for inconsistencies      |
| `version`      | Print xckit version                                      |

All commands accept `-f` (or `--file`) to specify the `.xcstrings` file path. When omitted, xckit looks for a `.xcstrings` file in the current directory.

### list

```bash
xckit list [-f file.xcstrings] [--prefix <prefix>] [--state <state>] [--json]
```

Lists all keys with their translation status. Use `--prefix` to filter by key prefix and `--state` to filter by `extractionState` (e.g. `manual`, `stale`, `new`). Each key with a non-empty `extractionState` is annotated in the output (e.g. `mykey [manual]:`). The two filters can be combined.

- `--json`: Print a single JSON document to stdout instead of human-readable text (combinable with `--prefix`/`--state`). Each key includes `key`, `extractionState`, and a `languages` map keyed by language code; each language entry has a `state` (`translated`, `needs_review`, `new`, or `missing`) and either a `value` (plain string) or a `units` array of `{path, state, value}` for plural/device/substitution variations.

### untranslated

```bash
xckit untranslated [-f file.xcstrings] [--lang <language>] [--prefix <prefix>] [--detail] [--json] [--fail-if-any]
```

Shows keys that need translation. Without `--lang`, returns keys with any untranslated language. Use `--detail` to see per-variation-path breakdown (e.g., `key > ja > plural.other`).

- `--json`: Print a single JSON document to stdout instead of human-readable text (combinable with `--lang`/`--prefix`). Always reports at per-variation-path granularity (like `--detail`), as `{"untranslated": [{"key", "language", "path"}, ...]}`.
- `--fail-if-any`: Exit with status 1 if any untranslated string is found (0 otherwise). Works with any output mode, including `--json`, making it suitable for CI and pre-commit gates.

### set

```bash
xckit set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--state <state>] [--comment <text>] [--force] [--allow-new-language] [--require-existing] [--dry-run] [--json] <key> <value>
xckit set [-f file.xcstrings] --stdin [--force] [--allow-new-language] [--require-existing] [--dry-run] [--json]
```

Sets a translation for the given key/language. The key is created when it does not yet exist; existing keys are updated in place.

- `--plural`: Set a plural variation (`zero`, `one`, `two`, `few`, `many`, `other`)
- `--device`: Set a device variation (`iphone`, `ipad`, `mac`, `appletv`, `applewatch`, `applevision`, `other`)
- `--state`: `extractionState` applied only when the key is created (e.g. `manual`). Ignored when the key already exists.
- `--comment`: Set or update the key's translator-facing comment (visible in Xcode and in `export`'s CSV `comment` column). Pass an empty string (`--comment ""`) to clear an existing comment. The comment is a property of the key itself (not per-language), so it is applied together with the value in the same call; it cannot be combined with `--stdin` — use the NDJSON `comment` field instead. Omitting `--comment` entirely leaves any existing comment untouched, including when only the value is being updated.
- `--force`: Suppress the migration warning when converting a plain string to variations
- `--allow-new-language`: Allow adding a language that is not yet present in the catalog
- `--require-existing`: Fail instead of silently creating a new key (protects against typoed keys creating an unintended new entry)
- `--dry-run`: Preview what would be created/updated without writing the file
- `--json`: Print a structured JSON document instead of human-readable text (combinable with `--dry-run`)

Plural and device flags can be combined to set nested variations (e.g., device > plural).

**Migrating a plain string to variations:** setting the first `--plural` or `--device` variation on a key that currently holds a plain translation moves the existing value into the `other` fallback slot (for combined plural+device, `device: other` > `plural: other`), preserving its translation state. Explicitly targeting `other` in the same call overrides the preserved value. Appending further categories to an already-variation key leaves existing values untouched.

**Format specifiers (`%d` vs `%lld`):** Swift string interpolation of an `Int` (e.g. `"\(count) items"`) generates `%lld` in the catalog. When adding plural variations manually, match the specifier your code actually generates — usually `%lld` for `Int` interpolation — otherwise the catalog entry won't line up with what Xcode extracts.

**Language validation:** `--lang` must match a language already present in the catalog (any language used in a key's `localizations`, or the catalog's `sourceLanguage`) unless `--allow-new-language` is passed. This prevents typos (e.g. `--lang jp`) or case mismatches (e.g. `--lang JA` instead of `ja`) from silently creating a bogus new language column. When a match is unrecognized, the error suggests the closest existing language code (`did you mean "ja"?`) when one can be inferred. If the catalog has no languages yet (a brand-new catalog), the first `set` call is never blocked, so seeding the very first translation doesn't require `--allow-new-language`.

**Batch input (`--stdin`):** reads newline-delimited JSON (NDJSON) from stdin, one object per line, and applies every line in a single process run with a single atomic write. This is the efficient way to script many translations at once (e.g. many keys x many languages) instead of spawning `set` once per key/language pair. `--stdin` cannot be combined with the positional `<key> <value>` arguments, and `--lang`/`--plural`/`--device`/`--state` are taken per line instead of as flags. Each line's schema is:

```json
{"key": "greeting", "lang": "ja", "value": "こんにちは", "plural": "other", "device": "iphone", "state": "manual", "comment": "Shown on the login screen"}
```

- `key`, `lang`, `value`: required
- `plural`, `device`, `state`: optional, same meaning as the equivalent single-`set` flags
- `comment`: optional, same meaning as `--comment`. Omitting the field leaves the key's existing comment untouched; an explicit `"comment": ""` clears it.

Example:

```bash
printf '%s\n' \
  '{"key": "greeting", "lang": "ja", "value": "こんにちは"}' \
  '{"key": "greeting", "lang": "fr", "value": "Bonjour"}' \
  '{"key": "item_count", "lang": "ja", "value": "%lldつのアイテム", "plural": "other"}' \
  | xckit set -f Localizable.xcstrings --stdin
```

All lines are parsed and validated (including language validation) before anything is applied — if any single line is malformed or fails validation, nothing is written and every offending line is reported by line number. `--allow-new-language` applies to the whole batch: once a new language is introduced by an earlier line, later lines in the same batch may reuse it without repeating validation failures. The command prints a per-line result plus a final `Summary: N created, M updated` line (or the equivalent `--json` document).

**`--json` output**, for both single and `--stdin` invocations:

```json
{
  "results": [
    {"key": "greeting", "lang": "ja", "action": "created", "commentAction": "set"},
    {"key": "item_count", "lang": "ja", "action": "updated", "path": "plural.other"}
  ],
  "summary": {"created": 1, "updated": 1}
}
```

`action` is `created` when the key itself was newly added to the catalog, and `updated` when an existing key gained or changed a localization. `path` is present only for plural/device variations and describes where within the variation structure the value was written (e.g. `plural.other`, `device.iphone`, `device.iphone.plural.one`); it is omitted for plain string translations. `commentAction` is present only when the call changed the key's comment: `set` when a non-empty comment was written, `cleared` when an empty comment removed an existing one; it is omitted when the comment was left untouched.

### remove

```bash
xckit remove [-f file.xcstrings] [--state <state>] [--dry-run] [<key>]
```

Removes a key from the catalog regardless of its `extractionState`.

- `<key>`: Remove that single key. Errors if the key does not exist.
- `--state <state>`: Remove every key whose `extractionState` matches (e.g. `stale`, `manual`).
- Combining `<key>` and `--state`: removes the named key only if its state matches.
- `--dry-run`: Print the keys that would be removed without modifying the file.

### status

```bash
xckit status [-f file.xcstrings] [--json]
```

Displays translation progress for each language, showing both key-level and string-unit-level completion percentages along with `needs_review` counts. Stale keys are reported separately and excluded from progress calculations.

- `--json`: Print a single JSON document to stdout instead of human-readable text: `{sourceLanguage, totalKeys, staleKeys, activeKeys, languages: [{language, keys: {translated, total, percentage}, strings: {translated, total, percentage}, needsReview}, ...]}`.

### export

```bash
xckit export --format csv [-f file.xcstrings] [-o output.csv]
```

Exports all strings to CSV. Variations are flattened into rows with bracket notation (e.g., `key[plural.other]`, `key[device.iphone.plural.one]`). Substitutions are exported as `key[substitutions.name.plural.other]`. Output goes to stdout when `-o` is omitted.

### import

```bash
xckit import --format csv [-f file.xcstrings] [--dry-run] [--backup] [--on-missing-key skip|error] [--clear-empty] <csv-file>
```

Imports translations from a CSV file produced by `export`.

- `--dry-run`: Preview changes without writing. The summary reports `created / updated / unchanged / cleared / skipped`; cells whose value already matches the catalog are counted as `unchanged` and never written (also when importing for real)
- `--backup`: Create a `.bak` copy before writing
- `--on-missing-key skip|error`: Handle keys present in CSV but missing from the catalog (default: `skip`)
- `--clear-empty`: Remove translations for empty CSV cells

### stale

```bash
xckit stale [-f file.xcstrings] [--remove] [--dry-run]
```

Lists keys with `extractionState: stale`. Use `--remove` to delete them from the catalog, and `--dry-run` to preview without writing.

### lint

```bash
xckit lint [-f file.xcstrings] [--json]
```

Statically validates a catalog for common inconsistencies that Xcode itself doesn't flag.

| Rule | Severity | Description |
| --- | --- | --- |
| `format-specifier` | error | A translation's format specifiers (`%d`, `%@`, `%1$d`, `%#@name@`, ...) don't match the source language. Reordering with explicit positional specifiers (`%1$d` / `%2$d`) is allowed. |
| `plural-missing-other` | error | A plural variation is missing the mandatory `other` category. |
| `empty-key` | error | The catalog contains an empty string (`""`) key. |
| `literal-newline` | warning | A value contains a literal newline character. |
| `language-consistency` | error | Language codes differ only by case (e.g. `ja` and `JA` both present), or a language code appears on a single key while closely resembling a well-established one (likely typo). |
| `substitution-structure` | error | A substitution has `argNum: 0`, an empty `formatSpecifier`, or is never referenced (`%#@name@`) by its host string. |

Exits non-zero if any `error`-level issue is found (warnings alone exit 0), making it suitable for CI. Pass `--json` for a single JSON document: `{"issues": [{"rule", "severity", "key", "language"?, "path"?, "message"}]}`.

---

## Usage Examples

### Export, translate, import workflow

```bash
# 1. Export current translations to CSV
xckit export --format csv -f Localizable.xcstrings -o translations.csv

# 2. Edit translations.csv in a spreadsheet or send to translators

# 3. Preview what will change
xckit import --format csv -f Localizable.xcstrings --dry-run translations.csv

# 4. Import with a backup
xckit import --format csv -f Localizable.xcstrings --backup translations.csv
```

### CI usage

```bash
# Fail the build if any Japanese translations are missing
xckit untranslated --lang ja -f Localizable.xcstrings --fail-if-any

# Fail the build if any translation in any language is missing, and get
# structured output for tooling/AI agents to parse
xckit untranslated -f Localizable.xcstrings --fail-if-any --json

# Check overall progress (machine-readable)
xckit status -f Localizable.xcstrings --json

# Clean up stale keys
xckit stale --remove -f Localizable.xcstrings
```

---

## Why xckit?

xckit is a single Go binary — no Xcode needed, runs on Linux CI.

It covers the full workflow: inspect keys, set translations including plural and device variations, check progress, export to CSV for translators, and import back. Stale and needs_review states are tracked alongside the standard untranslated state.

---

## Development

### Environment setup (Nix)

A [Nix flake](flake.nix) provides Go, gopls, gotools, golangci-lint, and goreleaser.

```bash
# With direnv (recommended)
direnv allow

# Without direnv
nix develop
```

### Running tests

```bash
make test
```

### Building

```bash
make build
```

### Coverage

```bash
make coverage
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
