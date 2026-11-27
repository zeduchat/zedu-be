package models

import (
	"regexp"
	"strings"
	"testing"
)

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain term is unchanged", in: "frontend", want: "frontend"},
		{name: "percent is escaped", in: "%", want: `\%`},
		{name: "underscore is escaped", in: "_", want: `\_`},
		{name: "backslash is escaped", in: `\`, want: `\\`},
		{name: "mixed term escapes every metacharacter", in: `a%b_c\d`, want: `a\%b\_c\\d`},
		{name: "already escaped characters are escaped again", in: `\%`, want: `\\\%`},
		{name: "empty term stays empty", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeLikePattern(tt.in); got != tt.want {
				t.Fatalf("escapeLikePattern(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestEscapeLikePatternMatchesLiterally applies the escaped term the way the
// channel search does, `%<term>%`, using the same matching rules PostgreSQL
// documents for LIKE: "%" is any run of characters, "_" is one character, and
// backslash escapes the next character.
func TestEscapeLikePatternMatchesLiterally(t *testing.T) {
	tests := []struct {
		term  string
		found string
		other string
	}{
		{term: "%", found: "scope-percent%literal", other: "scope-percentXliteral"},
		{term: "_", found: "scope-under_score", other: "scope-underXscore"},
		{term: `\`, found: `scope-back\slash`, other: "scope-backXslash"},
	}

	for _, tt := range tests {
		pattern := likePatternToRegexp("%" + escapeLikePattern(tt.term) + "%")
		re := regexp.MustCompile(pattern)

		if !re.MatchString(tt.found) {
			t.Errorf("search %q does not match literal %q (pattern %q)", tt.term, tt.found, pattern)
		}
		if re.MatchString(tt.other) {
			t.Errorf("search %q matches %q, so the metacharacter is still a wildcard (pattern %q)", tt.term, tt.other, pattern)
		}
	}
}

// likePatternToRegexp converts a PostgreSQL LIKE pattern to an anchored
// regular expression, treating backslash as the escape character.
func likePatternToRegexp(pattern string) string {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
			if i < len(pattern) {
				b.WriteString(regexp.QuoteMeta(string(pattern[i])))
			}
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return b.String()
}
