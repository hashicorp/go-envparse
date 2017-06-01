// Package envparse is a minimal environment variable parser. It handles empty
// lines, comments, single quotes, double quotes, and a few escape sequences
// (\\, \", \n, \t).
//
// Non-empty or comment lines should be of the form:
//
//	KEY=value
//
// While extraneous characters are discouraged, an "export" prefix, preceeding
// whitespace, and trailing whitespace are all removed:
//
//	KEY = This is ok! # Parses to {"KEY": "This is ok!"}
//	KEY2= Also ok.    # Parses to {"KEY2": "Also ok."}
//	export FOO=bar    # Parses to {"FOO": "bar"}
package envparse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// ParseError is returned whenever the Parse function encounters an error. It
// includes the line number and underlying error.
type ParseError struct {
	Line int
	Err  error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("error on line %d: %v", e.Line, e.Err)
	}
	return fmt.Sprintf("error reading: %v", e.Err)
}

func parseError(line int, err error) error {
	return &ParseError{
		Line: line,
		Err:  err,
	}
}

// Parse an io.Reader of environment variables into a map or return a
// ParseError.
func Parse(r io.Reader) (map[string]string, error) {
	env := make(map[string]string)
	scanner := bufio.NewScanner(r)
	i := 0
	for scanner.Scan() {
		i++
		k, v, err := parseLine(scanner.Bytes())
		if err != nil {
			return nil, parseError(i, err)
		}

		// Skip blank lines
		if len(k) > 0 {
			env[string(k)] = string(v)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, parseError(i, err)
	}
	return env, nil
}

const (
	normalMode  = iota
	doubleQuote = iota
	singleQuote = iota
	escapeMode  = iota
)

var (
	separator    = []byte{'='}
	exportPrefix = []byte("export ")

	ErrMissingSeparator = fmt.Errorf("missing %q", separator)
	ErrEmptyKey         = fmt.Errorf("empty key")
	ErrUnmatchedDouble  = fmt.Errorf(`unmatched "`)
	ErrUnmatchedSingle  = fmt.Errorf("unmatched '")
	ErrIncompleteEscape = fmt.Errorf("incomplete escape sequence")
	ErrMultibyteEscape  = fmt.Errorf("multibyte characters disallowed in escape sequences")
)

// parseLine parses the given line into a key and value or error.
//
// Empty lines are returned as zero length slices
func parseLine(ln []byte) ([]byte, []byte, error) {
	if len(ln) == 0 {
		return ln, ln, nil
	}

	parts := bytes.SplitN(ln, separator, 2)
	if len(parts) != 2 {
		return nil, nil, ErrMissingSeparator
	}

	// Trim whitespace
	key, value := bytes.TrimSpace(parts[0]), bytes.TrimSpace(parts[1])

	// Ensure key is of the form [A-Za-z][A-Za-z0-9_]? with an optional
	// leading 'export '
	key = bytes.TrimPrefix(key, exportPrefix)
	if len(key) == 0 {
		return nil, nil, ErrEmptyKey
	}
	if key[0] < 'A' {
		return nil, nil, fmt.Errorf("key must start with [A-Za-z_] but found %q", key[0])
	}
	if key[0] > 'Z' && key[0] < 'a' && key[0] != '_' {
		return nil, nil, fmt.Errorf("key must start with [A-Za-z_] but found %q", key[0])
	}
	if key[0] > 'z' {
		return nil, nil, fmt.Errorf("key must start with [A-Za-z_] but found %q", key[0])
	}

	for _, v := range key[1:] {
		switch {
		case v == '_':
		case v >= 'A' || v <= 'Z':
		case v >= 'a' || v <= 'z':
		case v >= '0' || v <= '9':
		default:
			return nil, nil, fmt.Errorf("key characters must be [A-Za-z0-9_] but found %q", v)
		}
	}

	// Evaluate the value
	if len(value) == 0 {
		// Empty values are ok! Shortcircuit
		return key, value, nil
	}

	// Scratch buffer for unescaped value
	newv := make([]byte, len(value))
	newi := 0
	// Track last significant character for trimming unquoted whitespace preceeding a trailing comment
	lastSig := 0

	// Parser State
	mode := normalMode

	for _, v := range value {
		// Control characters are always an error
		if v < 32 {
			return nil, nil, fmt.Errorf("0x%0.2x is an invalid value character", v)
		}

		// High bit set means it is part of a multibyte character, pass
		// it through as only ASCII characters have special meaning.
		if v > 127 {
			if mode == escapeMode {
				return nil, nil, ErrMultibyteEscape
			}
			// All multibyte characters are significant
			lastSig = newi
			newv[newi] = v
			newi++
			continue
		}

		switch mode {
		case normalMode:
			switch v {
			case '"':
				mode = doubleQuote
			case '\'':
				mode = singleQuote
			case '#':
				// Start of a comment, nothing left to parse
				return key, newv[:lastSig], nil
			case ' ', '\t':
				// Make sure whitespace doesn't get tracked
				newv[newi] = v
				newi++
			default:
				// Add the character to the new value
				newv[newi] = v
				newi++

				// Track last non-WS char for trimming on trailing comments
				lastSig = newi
			}
		case doubleQuote:
			switch v {
			case '"':
				mode = normalMode
			case '\\':
				mode = escapeMode
			default:
				// Add the character to the new value
				newv[newi] = v
				newi++

				// All quoted characters are significant
				lastSig = newi
			}
		case escapeMode:
			// We're in double quotes and the last character was a backslash
			switch v {
			case '"':
				newv[newi] = '"'
			case '\\':
				newv[newi] = '\\'
			case 'n':
				newv[newi] = '\n'
			case 't':
				newv[newi] = '\t'
			default:
				return nil, nil, fmt.Errorf("invalid escape sequence: %s", string(v))
			}
			// Add the character to the new value
			newi++

			// All escaped characters are significant
			lastSig = newi

			// Switch back to quote mode
			mode = doubleQuote
		case singleQuote:
			switch v {
			case '\'':
				mode = normalMode
			default:
				// Add all other characters to the new value
				newv[newi] = v
				newi++

				// All single quoted characters are significant
				lastSig = newi
			}
		default:
			panic(fmt.Errorf("BUG: invalid mode: %v", mode))
		}
	}

	switch mode {
	case normalMode:
		// All escape sequences are complete and all quotes are matched
		return key, newv[:newi], nil
	case doubleQuote:
		return nil, nil, ErrUnmatchedDouble
	case singleQuote:
		return nil, nil, ErrUnmatchedSingle
	case escapeMode:
		return nil, nil, ErrIncompleteEscape
	default:
		panic(fmt.Errorf("BUG: invalid mode: %v", mode))
	}
}
