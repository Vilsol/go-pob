package utils

import (
	"regexp"
	"strings"
)

func Capital(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

var capitalEachRegex = regexp.MustCompile(`(^\w|\s\w)`)

func CapitalEach(s string) string {
	return capitalEachRegex.ReplaceAllStringFunc(s, strings.ToUpper)
}

var (
	gggColourTagRegex  = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggKeywordRegex    = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggKeywordAltRegex = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
)

// Removes GGG string tags used for keyword popups. E.g. "[Critical|Critical Hit]" -> "Critical Hit"
//
//pob:src/Modules/Common.lua:1110-1116
func EscapeGGGString(text string) string {
	line := gggColourTagRegex.ReplaceAllString(text, "${1}")
	line = gggKeywordRegex.ReplaceAllString(line, "${1}")
	return gggKeywordAltRegex.ReplaceAllString(line, "${1}")
}
