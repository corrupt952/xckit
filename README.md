# xckit

CLI tool for managing Xcode String Catalogs (.xcstrings).

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
- Single Go binary -- no Xcode required, works on Linux CI

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

## Commands Reference

| Command        | Description                                              |
|----------------|----------------------------------------------------------|
| `list`         | List all keys with translation status                    |
| `untranslated` | Find keys that need translation                          |
| `set`          | Set a translation for a specific key and language        |
| `status`       | Show translation progress summary per language           |
| `export`       | Export strings to CSV                                    |
| `import`       | Import translations from CSV                             |
| `stale`        | List or remove stale keys                                |
| `version`      | Print xckit version                                      |

All commands accept `-f` (or `--file`) to specify the `.xcstrings` file path. When omitted, xckit looks for a `.xcstrings` file in the current directory.

### list

```bash
xckit list [-f file.xcstrings] [--prefix <prefix>]
```

Lists all keys with their translation status. Use `--prefix` to filter by key prefix.

### untranslated

```bash
xckit untranslated [-f file.xcstrings] [--lang <language>] [--prefix <prefix>] [--detail]
```

Shows keys that need translation. Without `--lang`, returns keys with any untranslated language. Use `--detail` to see per-variation-path breakdown (e.g., `key > ja > plural.other`).

### set

```bash
xckit set [-f file.xcstrings] --lang <language> [--plural <category>] [--device <device>] [--force] <key> <value>
```

Sets a translation for a specific key and language.

- `--plural`: Set a plural variation (`zero`, `one`, `two`, `few`, `many`, `other`)
- `--device`: Set a device variation (`iphone`, `ipad`, `mac`, `appletv`, `applewatch`, `applevision`, `other`)
- `--force`: Suppress the migration warning when converting a plain string to variations

Plural and device flags can be combined to set nested variations (e.g., device > plural).

### status

```bash
xckit status [-f file.xcstrings]
```

Displays translation progress for each language, showing both key-level and string-unit-level completion percentages along with `needs_review` counts. Stale keys are reported separately and excluded from progress calculations.

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

- `--dry-run`: Preview changes without writing
- `--backup`: Create a `.bak` copy before writing
- `--on-missing-key skip|error`: Handle keys present in CSV but missing from the catalog (default: `skip`)
- `--clear-empty`: Remove translations for empty CSV cells

### stale

```bash
xckit stale [-f file.xcstrings] [--remove] [--dry-run]
```

Lists keys with `extractionState: stale`. Use `--remove` to delete them from the catalog, and `--dry-run` to preview without writing.

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
xckit untranslated --lang ja -f Localizable.xcstrings && echo "Untranslated keys found" && exit 1

# Check overall progress
xckit status -f Localizable.xcstrings

# Clean up stale keys
xckit stale --remove -f Localizable.xcstrings
```

## Why xckit?

- **Single binary**: Distributed as a standalone Go binary. No Xcode installation needed.
- **CSV round-trip**: Export to CSV, hand off to translators or edit in a spreadsheet, then import back. Variations, substitutions, and nested structures are preserved.
- **Full variation support**: Plural categories, device variations, nested device-plural combinations, and substitutions are first-class citizens.
- **CI-friendly**: Runs on macOS and Linux. Detect untranslated keys, enforce translation coverage, and clean up stale keys in your pipeline.

## Development

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

## License

MIT License. See [LICENSE](LICENSE) for details.
