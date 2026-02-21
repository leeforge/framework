package utils

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func UpperCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	c := cases.Title(language.English)
	s = c.String(s)
	return strings.ReplaceAll(s, " ", "")
}

// LowerCamelCase converts snake_case to lowerCamelCase.
// Example: "created_by_id" -> "createdById"
func LowerCamelCase(s string) string {
	upper := UpperCamelCase(s)
	if len(upper) == 0 {
		return upper
	}
	// Convert first character to lowercase
	return strings.ToLower(upper[:1]) + upper[1:]
}
