package utils

import "strings"

func Capital(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
