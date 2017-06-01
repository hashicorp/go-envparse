# envparse

A minimal Go environment variable parser. It's intended to be used to parse
`.env` style files similar to [godotenv](https://github.com/joho/godotenv) or
[rubydotenv](https://github.com/bkeepers/dotenv), but perform minimal
allocations, handle more complex quoting, and be better tested.

Parsing a line does 2 allocations regardless of line length or complexity.

The parser's behavior has two competing goals:

* Support a minimal number of features
* Support arbitrarily complex combinations of the features it does have

For example the following common features *are intentionaly missing*:

* Full escape sequence support (only `\n`, `\t`, `\\`, and `\"` are supported)
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
