package envparse

import (
	"strings"
	"testing"
)

func TestParse_OK(t *testing.T) {

}

func TestParse_Err(t *testing.T) {
	cases := []struct {
		name string
		ln   string

		// Either exact err or partial error message should be set
		err     error
		partial string
	}{
		{"MissingEqual", "foo bar", ErrMissingSeparator, ""},
		{"EmptyKey", "=bar", ErrEmptyKey, ""},
		{"EqualOnly", "=", ErrEmptyKey, ""},
		{"InvalidKey", "1abc=x", nil, "key"},
		{"InvalidKey2", "@abc=x", nil, "key"},
		{"InvalidKey3", "a b c=x", nil, "key"},
		{"InvalidKey4", "a\nb=x", nil, "key"},
		{"InvalidValue", "FOO=\x00", nil, "value"},
		{"OpenDoubleQuote", `FOO=" bar`, ErrUnmatchedDouble, ""},
		{"OpenSingleQuote", `FOO=' bar`, ErrUnmatchedSingle, ""},
		{"UnmatchedMix", `FOO=ok '"ok"' \"not ok ''`, ErrUnmatchedDouble, ""},
		{"UnmatchedMix2", `FOO=ok '"ok"' \"not ok '"'`, ErrUnmatchedSingle, ""},
		{"InvalidEscape", `FOO="\a"`, nil, `"a"`},
		{"IncompleteEscape", `FOO="\`, ErrIncompleteEscape, ""},
		{"IncompleteHex", `FOO="\u12"`, ErrIncompleteHex, ""},
		{"InvalidHex", `FOO="\uabcZ"`, nil, `"Z"`},
		{"IncompleteSurrogatePair1", `FOO="abc \uD83D"`, ErrIncompleteSur, ""},
		{"IncompleteSurrogatePair2", `FOO="abc \uD83D \uDE01"`, ErrIncompleteSur, ""},
		{"IncompleteSurrogatePair3", `FOO="abc \uD83DDE01"`, ErrIncompleteSur, ""},
		{"IncompleteSurrogatePair4", `FOO="abc \uD83D\uDE0"`, nil, `"\""`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, v, err := parseLine([]byte(c.ln))
			if err == nil {
				t.Fatalf("err == nil; found: %s=%q", k, v)
			}

			if c.err != nil && c.err != err {
				t.Errorf("expected err=%v but found %v", c.err, err)
			}

			if c.partial != "" && !strings.Contains(err.Error(), c.partial) {
				t.Errorf("expected err to contain %q but found %v", c.partial, err)
			}
		})
	}
}

func TestParseLine_OK(t *testing.T) {
	cases := []struct {
		name string
		ln   string
		k    string
		v    string
	}{
		{"Empty", "", "", ""},
		{"Emptyish", " ", "", ""},
		{"OnlyComment", "# ...", "", ""},
		{"OnlyCommentish", " # ...", "", ""},
		{"EmptyValue", "FoO=", "FoO", ""},
		{"EmptyValueComment", "F=# ...", "F", ""},
		{"EmptyValueSpace", "F_O= ", "F_O", ""},
		{"EmptyValueSpaceComment", "F= # ...", "F", ""},
		{"Simple", "FOO=bar", "FOO", "bar"},
		{"Export", "export FOO=bar", "FOO", "bar"},
		{"Spaces", " FOO = bar baz ", "FOO", "bar baz"},
		{"Tabs", "	FOO	= 	bar 	", "FOO", "bar"},
		{"ExportSpaces", "export FOO = bar", "FOO", "bar"},
		{"ExportAsKey", "export = bar", "export", "bar"},
		{"Nums", "A1B2C3=a1b2c3", "A1B2C3", "a1b2c3"},
		{"Comments", "FOO=bar # ok", "FOO", "bar"},
		{"EmptyComments1", "FOO=#bar#", "FOO", ""},
		{"EmptyComments2", "FOO= # bar ", "FOO", ""},
		{"DoubleQuotes", `FOO="bar#"`, "FOO", "bar#"},
		{"DoubleQuoteNewline", `FOO="bar\n"`, "FOO", "bar\n"},
		{"DoubleQuoteNewlineComment", `FOO="bar\n" # comment`, "FOO", "bar\n"},
		{"DoubleQuoteSpaces", `FOO = " bar\t" `, "FOO", " bar\t"},
		{"SingleQuotes", "FOO='bar#'", "FOO", "bar#"},
		{"SingleQuotesNewline", `FOO='\n' # empty`, "FOO", "\\n"},
		{"SingleQuotesEmpty", "FOO='' # empty", "FOO", ""},
		{"NormalSingleMix", "FOO=normal'single ' ", "FOO", "normalsingle "},
		{"NormalDoubleMix", `FOO= "double\\" normal # "EOL"`, "FOO", "double\\ normal"},
		{"AllModes", `export FOO =  'single\n' \\normal\t "double\"\n " # comment`, "FOO", "single\\n \\\\normal\\t double\"\n "},
		{"UnicodeLiteral", "U1=\U0001F525", "U1", "\U0001F525"},
		{"UnicodeLiteralQuoted", "U2= ' \U0001F525 ' ", "U2", " \U0001F525 "},
		{"EscapedUnicode1byte", `U3="\u2318"`, "U3", "\U00002318"},
		{"EscapedUnicode2byte", `U3="\uD83D\uDE01"`, "U3", "\U0001F601"},
		{"EscapedUnicodeCombined", `U4="\u2318\uD83D\uDE01"`, "U4", "\U00002318\U0001F601"},
		{"README.mdEscapedUnicode", `FOO="The template value\nmay have included\nsome newlines!\n\ud83d\udd25"`, "FOO", "The template value\nmay have included\nsome newlines!\n🔥"},
		{"UnderscoreKey", "_=x' ' ", "_", "x "},
		{"README.md", `SOME_KEY = normal unquoted \text 'plus single quoted\' "\"double quoted " # EOL`, "SOME_KEY", `normal unquoted \text plus single quoted\ "double quoted `},
		{"WindowsNewline", `w="\r\n"`, "w", "\r\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, v, err := parseLine([]byte(c.ln))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if string(k) != c.k {
				t.Errorf("expected key %q but found %q", c.k, string(k))
			}
			if string(v) != c.v {
				t.Errorf("expected value %q but found [%s] - %q", c.v, string(v), string(v))
			}
		})
	}
}

func TestParseLine_Err(t *testing.T) {
}

func BenchmarkParseLine_Simple(b *testing.B) {
	line := []byte("FOO=bar")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k, v, err := parseLine(line)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(k) != 3 {
			b.Fatalf("unexpected key: %q (%d)", k, len(k))
		}
		if len(v) != 3 {
			b.Fatalf("unexpected value: %q (%d)", v, len(v))
		}
	}
}

func BenchmarkParseLine_Complex(b *testing.B) {
	line := []byte(`export FOO = bar"baz'\n'\t " ☃ '#\n\t' "#\uD83D\ude01" # a really # long # comment!!!1111   `)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k, v, err := parseLine(line)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(k) != 3 {
			b.Fatalf("unexpected key: %q (%d)", k, len(k))
		}
		if len(v) != 27 {
			b.Fatalf("unexpected value: %q (%d)", v, len(v))
		}
	}
}
