package host

import (
	"fmt"
	"regexp"
)

var manifestLink = regexp.MustCompile(`(?m)^[ \t]*<link rel="manifest" [^>]*/>\n`)

func DesktopIndex(html []byte) ([]byte, error) {
	if n := len(manifestLink.FindAllIndex(html, -1)); n != 1 {
		return nil, fmt.Errorf("index.html: %d manifest links, want 1", n)
	}
	return manifestLink.ReplaceAll(html, nil), nil
}
