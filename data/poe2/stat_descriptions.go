package poe2

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// StatDescription is one described entry of a Data/StatDescriptions file: its stats and its forms in file order.
type StatDescription struct {
	Stats []string
	Forms []StatDescriptionForm
}

// StatDescriptionForm is one line of a description: its text, a {min, max} limit per stat (a number, "#" for any,
// or "!" and the excluded number) and its special tokens in order.
type StatDescriptionForm struct {
	Text  string
	Limit [][2]any
	Flags []StatDescriptionFlag
}

// StatDescriptionFlag is a special token: V is a number, a string, or true for canonical_line.
type StatDescriptionFlag struct {
	V any
	K string
}

var (
	gggMarkupRe    = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggLinkRe      = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggAliasRe     = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
	gggEscapeRe    = regexp.MustCompile(`\\([^nb])`)
	sanitiseFirst  = regexp.MustCompile(`[\x{80}-\x{10FFFF}<]`)
	sanitiseAngles = regexp.MustCompile(`<[^<>]*>`)
)

// luaEscapes are the escapes left in a text as the stat description file is loaded: writeLuaTable writes every
// backslash as it is.
var luaEscapes = strings.NewReplacer(`\n`, "\n", `\b`, "\b", `\\`, `\`)

var loadStatDescriptions = sync.OnceValues(func() ([]StatDescription, error) {
	file, err := loader.LoadTranslation(context.Background(), itemTablesVersion, "en", "stat_descriptions", assetcache.AssetLoaderWrapper{Storage: storage.Get()})
	if err != nil {
		return nil, fmt.Errorf("stat descriptions: %w", err)
	}
	return statDescriptionsOf(file), nil
})

// StatDescriptions is Data/StatDescriptions/stat_descriptions.lua's described entries, from go-pob-data's
// stat_descriptions translations.
func StatDescriptions() ([]StatDescription, error) {
	return loadStatDescriptions()
}

//pob2:block:src/Export/Scripts/statdesc.lua:24-104
func statDescriptionsOf(file *rawdata.TranslationFile) []StatDescription {
	out := make([]StatDescription, 0, len(file.Descriptors))
	for _, d := range file.Descriptors {
		desc := StatDescription{Stats: append(append([]string{}, d.IDs...), d.OffHandIDs...)}
		for _, line := range d.List {
			if strings.Contains(line.Qualifier, "table_only") {
				continue
			}
			form := StatDescriptionForm{Text: luaEscapes.Replace(gggEscapeRe.ReplaceAllString(sanitiseText(escapeGGGString(line.String)), `\n$1`))}
			for _, c := range line.Conditions {
				form.Limit = append(form.Limit, statLimit(c))
			}
			form.Flags = statFlags(line.Handlers)
			desc.Forms = append(desc.Forms, form)
		}
		out = append(out, desc)
	}
	return out
}

//pob2:block:src/Export/Scripts/statdesc.lua:49-70
func statLimit(c rawdata.Condition) [2]any {
	end := func(p *int) any {
		if p == nil {
			return "#"
		}
		return float64(*p)
	}
	if c.Negated && c.Min != nil && c.Max != nil && *c.Min == *c.Max {
		return [2]any{"!", float64(*c.Min)}
	}
	return [2]any{end(c.Min), end(c.Max)}
}

//pob2:block:src/Export/Scripts/statdesc.lua:71-98
func statFlags(handlers []rawdata.IndexHandler) []StatDescriptionFlag {
	var tokens []string
	for _, h := range handlers {
		tokens = append(tokens, h.Name)
		if h.Arg != "" {
			tokens = append(tokens, h.Arg)
		}
	}
	var flags []StatDescriptionFlag
	for i := 0; i < len(tokens); {
		if tokens[i] == "canonical_line" {
			flags = append(flags, StatDescriptionFlag{K: "canonical_line", V: true})
			i++
			continue
		}
		if i+1 >= len(tokens) {
			i++
			continue
		}
		var v any = tokens[i+1]
		if n, err := strconv.ParseFloat(tokens[i+1], 64); err == nil {
			v = n
		}
		flags = append(flags, StatDescriptionFlag{K: tokens[i], V: v})
		i += 2
	}
	return flags
}

//pob2:src/Modules/Common.lua:1078-1084
func escapeGGGString(text string) string {
	text = gggMarkupRe.ReplaceAllString(text, "$1")
	text = gggLinkRe.ReplaceAllString(text, "$1")
	return gggAliasRe.ReplaceAllString(text, "$1")
}

//pob2:src/Modules/Common.lua:321-347
func sanitiseText(text string) string {
	if !sanitiseFirst.MatchString(text) {
		return text
	}
	text = sanitiseAngles.ReplaceAllString(text, "")
	for _, r := range []struct{ from, to string }{
		{"‐", "-"}, {"‑", "-"}, {"‒", "-"}, {"–", "-"}, {"—", "-"}, {"―", "-"}, {"−", "-"},
		{"• ", ""}, {"•", ""}, {"ä", "a"}, {"ö", "o"}, {"í", "i"}, {"ó", "o"},
	} {
		text = strings.ReplaceAll(text, r.from, r.to)
	}
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
