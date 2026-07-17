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
xckit set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--state <state>] [--force] [--allow-new-language] <key> <value>
```

Sets a translation for the given key/language. The key is created when it does not yet exist; existing keys are updated in place.

- `--plural`: Set a plural variation (`zero`, `one`, `two`, `few`, `many`, `other`)
- `--device`: Set a device variation (`iphone`, `ipad`, `mac`, `appletv`, `applewatch`, `applevision`, `other`)
- `--state`: `extractionState` applied only when the key is created (e.g. `manual`). Ignored when the key already exists.
- `--force`: Suppress the migration warning when converting a plain string to variations
- `--allow-new-language`: Allow adding a language that is not yet present in the catalog

Plural and device flags can be combined to set nested variations (e.g., device > plural).

**Language validation:** `--lang` must match a language already present in the catalog (any language used in a key's `localizations`, or the catalog's `sourceLanguage`) unless `--allow-new-language` is passed. This prevents typos (e.g. `--lang jp`) or case mismatches (e.g. `--lang JA` instead of `ja`) from silently creating a bogus new language column. When a match is unrecognized, the error suggests the closest existing language code (`did you mean "ja"?`) when one can be inferred. If the catalog has no languages yet (a brand-new catalog), the first `set` call is never blocked, so seeding the very first translation doesn't require `--allow-new-language`.

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
