package models

import "strings"

// escapeLikePattern escapes the LIKE wildcards in a search term so it matches
// literally: a term such as "%" or "_" names those characters instead of
// matching every value.
//
// The replacement runs in a single pass, so a backslash inserted for "%" or
// "_" is not escaped again. PostgreSQL treats backslash as the default LIKE
// escape character, so the escaped term needs no ESCAPE clause.
func escapeLikePattern(term string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term)
}
