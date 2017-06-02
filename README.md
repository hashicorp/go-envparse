# envparse

A minimal Go environment variable parser. It's intended to be used to parse
`.env` style files similar to [godotenv](https://github.com/joho/godotenv) or
[rubydotenv](https://github.com/bkeepers/dotenv), but perform minimal
allocations, handle more complex quoting, and be better tested.

Parsing a line does 2 allocations regardless of line length or complexity.

The parser is just sophisticated enough to handle parsing any JSON strings
properly to allow for cross-language/platform encoding of arbitrarily complex
data.

This is particularly useful if parsing environment variables from a templated
file where the template needs a way to escape newlines, etc:

```
FOO={{ some_template_function | toJSON }}
```

May enocde to something like:

```
FOO="The template value\nmay have included\nsome newlines!\n\ud83d\udd25"
```

Which `Parse()` would return as:

```go
map[string]string{
	"FOO": "The template value\nmay have included\nsome newlines!\n🔥",
}
```

The following common features *are intentionaly missing*:

* Full shell escape sequence support
  * Only JSON escape sequences are supported (see below)
* Variable interpolation
  * Use [Go's os.Expand](https://golang.org/pkg/os/#Expand) on the parsed value
* Anything YAML related
  * No

However, comments, unquoted, single quoted, and double quoted text may all be
used within a single value:

```
SOME_KEY = normal unquoted \text 'plus single quoted\' "\"double quoted " # EOL

# Parses to:
#  Key:   SOME_KEY
#  Value: normal unquoted \text plus single quoted\ "double quoted 
# (Note the trailing space inside the double quote is kept)
```

## Format

* Keys should be of the form: `[A-Za-z_][A-Za-z0-9_]?`
* Values should be valid ASCII or UTF-8.
* Newlines are always treated as delimiters so newlines within values *must* be
  escaped.
* Values may use one of more quoting styles:
  * Unquoted - `FOO=bar baz`
    * No escape sequences
    * Ends at `#`, `"`, `'`, or newline
    * Preceeding and trailing whitespace will be stripped
  * Double Quotes - `FOO="bar baz"`
    * Supports JSON escape sequences: `\uXXXX`, `\r`, `\n`, `\t`, `\\`, and
      `\"`
    * Ends at unescaped `"`
    * No whitespace trimming
  * Single Quotes - `FOO='bar baz'`
    * No escape sequences
    * Ends at `'`
    * No whitespace trimming
