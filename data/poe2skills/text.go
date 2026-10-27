package poe2skills

import (
	"regexp"
	"strconv"
	"strings"
)

// luaNumber is a number as the exporter writes it into Data/Skills/*.lua and PoB reads it back:
// tostring and file:write print %.14g.
func luaNumber(v float64) float64 {
	n, err := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 14, 64), 64)
	if err != nil {
		return v
	}
	return n
}

// datFloat is a dat Float cell: a float32 widened to a double.
func datFloat(v float64) float64 {
	return float64(float32(v))
}

var (
	sanitiseAngle  = regexp.MustCompile(`<[^>]*>`)
	sanitiseBullet = regexp.MustCompile("• ?")
	escapeGGGBrace = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	escapeGGGPlain = regexp.MustCompile(`\[([^|\]]+)\]`)
	escapeGGGPiped = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
	csdSuffix      = regexp.MustCompile(`(?s).csd`)
)

var sanitiseReplacer = strings.NewReplacer(
	"‐", "-", "‑", "-", "‒", "-", "–", "-", "—", "-", "―", "-", "−", "-",
)

// sanitiseText is Common.lua's sanitiseText over UTF-8 text.
//
//pob2:src/Modules/Common.lua:321-347
func sanitiseText(text string) string {
	if !hasHighByte(text) {
		return text
	}
	text = sanitiseAngle.ReplaceAllString(text, "")
	text = sanitiseReplacer.Replace(text)
	text = sanitiseBullet.ReplaceAllString(text, "")
	text = strings.NewReplacer("ä", "a", "ö", "o", "í", "i", "ó", "o").Replace(text)
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == 0x96 || c == 0x97:
			b.WriteByte('-')
		case c == 0xe4:
			b.WriteByte('a')
		case c == 0xf6:
			b.WriteByte('o')
		case c >= 0x80:
			b.WriteByte('?')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func hasHighByte(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 || s[i] == '<' {
			return true
		}
	}
	return false
}

// escapeGGGString is Common.lua's escapeGGGString.
//
//pob2:src/Modules/Common.lua:1078-1084
func escapeGGGString(text string) string {
	text = escapeGGGBrace.ReplaceAllString(text, "$1")
	text = escapeGGGPlain.ReplaceAllString(text, "$1")
	return escapeGGGPiped.ReplaceAllString(text, "$1")
}

// cleanAndSplit is skills.lua's cleanAndSplit, less the quote escaping the generated file reads back.
//
//pob2:src/Export/Scripts/skills.lua:16-31
func cleanAndSplit(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	lines := []string{}
	for _, line := range strings.Split(str, "\n") {
		line = strings.Trim(line, " \t\n\v\f\r")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
