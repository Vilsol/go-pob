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
