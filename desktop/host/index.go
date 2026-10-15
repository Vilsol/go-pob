package host

import (
	"fmt"
	"regexp"
)

var (
	googleFontLink = regexp.MustCompile(`(?m)^[ \t]*<link href="https://fonts\.googleapis\.com/css2\?[^"]*" rel="stylesheet" />\n`)
	googlePreconn  = regexp.MustCompile(`(?m)^[ \t]*<link rel="preconnect" href="https://fonts\.(googleapis|gstatic)\.com"[^>]*/>\n`)
	manifestLink   = regexp.MustCompile(`(?m)^[ \t]*<link rel="manifest" [^>]*/>\n`)
)

const FontsCSS = "/desktop/fonts/fonts.css"

func DesktopIndex(html []byte) ([]byte, error) {
	fonts := googleFontLink.FindAllIndex(html, -1)
	if len(fonts) == 0 {
		return nil, fmt.Errorf("index.html: no Google Fonts stylesheet link to replace")
	}
	if n := len(manifestLink.FindAllIndex(html, -1)); n != 1 {
		return nil, fmt.Errorf("index.html: %d manifest links, want 1", n)
	}
	first := fonts[0][0]
	out := append([]byte{}, html[:first]...)
	out = append(out, []byte(`    <link href="`+FontsCSS+`" rel="stylesheet" />`+"\n")...)
	out = append(out, html[first:]...)
	out = googleFontLink.ReplaceAll(out, nil)
	out = googlePreconn.ReplaceAll(out, nil)
	out = manifestLink.ReplaceAll(out, nil)
	return out, nil
}
