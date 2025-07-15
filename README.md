# xckit

A command-line tool for managing Xcode String Catalogs (.xcstrings files).

## Features

- List all localization keys with their translation status
- Find untranslated keys for specific languages
- Update translations directly from the command line
- View translation progress and statistics
- Support for multiple languages

## Installation

Download the latest release from the [Releases](https://github.com/corrupt952/xckit/releases) page.

### Build from source

```bash
go build -o xckit
```

## Usage

### List all keys

Show all localization keys with their translation status:

```bash
xckit list
```

Filter keys by prefix:

```bash
xckit list --prefix "login"
```

### Find untranslated keys

Find all untranslated keys across all languages:

```bash
xckit untranslated
```

Find untranslated keys for a specific language:

```bash
xckit untranslated --lang ja
```

Filter untranslated keys by prefix:

```bash
xckit untranslated --prefix "error"
xckit untranslated --lang ja --prefix "login"
```

### Set translations

Set a translation for a specific key and language:

```bash
xckit set --lang ja "hello_world" "こんにちは世界"
```

### View translation status

Get an overview of translation progress for all languages:

```bash
xckit status
```

### Specify a custom file

By default, xckit looks for `Localizable.xcstrings` in the current directory. You can specify a different file:

```bash
xckit list -f path/to/your/file.xcstrings
```

## Command Reference

### `list`
Lists all keys with their translation status. Use `--lang` to filter by language.

### `untranslated`
Shows keys that need translation. Use `--lang` to check a specific language.

### `set`
Updates a translation for a specific key and language. Requires `--lang`, key, and value.

### `status`
Displays translation progress summary for all languages.

### `version`
Shows the xckit version.

## Examples

```bash
# Check translation status
xckit status -f MyApp.xcstrings

# Find missing Japanese translations
xckit untranslated --lang ja -f MyApp.xcstrings

# Add a Japanese translation
xckit set --lang ja "welcome_message" "ようこそ" -f MyApp.xcstrings

# List all keys with their current translations
xckit list -f MyApp.xcstrings
```

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

See [LICENSE](LICENSE) file for details.