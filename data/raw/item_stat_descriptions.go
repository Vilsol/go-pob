package raw

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob-data/loader"
)

// ItemStatDescription is one described entry of Data/StatDescriptions/stat_descriptions.lua: its stats and its forms
// in file order.
type ItemStatDescription struct {
	Stats []string
	Forms []ItemStatDescriptionForm
}

// ItemStatDescriptionForm is one line of a description: its text, a {min, max} limit per stat (a number, "#" for any,
// or "!" and the excluded number) and its special tokens in order.
type ItemStatDescriptionForm struct {
	Text  string
	Limit [][2]any
	Flags []ItemStatDescriptionFlag
}

// ItemStatDescriptionFlag is a special token: V is a number, a string, or true for canonical_line.
type ItemStatDescriptionFlag struct {
	V any
	K string
}

var (
	gggMarkupRe = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggLinkRe   = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggAliasRe  = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
)

// statDescLuaEscapes are the escapes left in a text as the stat description file is loaded: writeLuaTable writes
// every backslash as it is.
var statDescLuaEscapes = strings.NewReplacer(`\n`, "\n", `\b`, "\b", `\\`, `\`)

var loadItemStatDescriptions = sync.OnceValues(func() ([]ItemStatDescription, error) {
	file, err := loader.LoadTranslation(DataContext(context.Background()), LatestVersion, "en", "stat_descriptions", lazyLoader())
	if err != nil {
		return nil, fmt.Errorf("stat descriptions: %w", err)
	}
	return itemStatDescriptionsOf(file), nil
})

// ItemStatDescriptions is Data/StatDescriptions/stat_descriptions.lua's described entries, from go-pob-data's
// stat_descriptions translations.
func ItemStatDescriptions() ([]ItemStatDescription, error) {
	return loadItemStatDescriptions()
}

//pob:block:src/Export/Scripts/statdesc.lua:23-103
func itemStatDescriptionsOf(file *rawdata.TranslationFile) []ItemStatDescription {
	out := make([]ItemStatDescription, 0, len(file.Descriptors))
	for _, d := range file.Descriptors {
		desc := ItemStatDescription{Stats: append(append([]string{}, d.IDs...), d.OffHandIDs...)}
		for _, line := range d.List {
			if strings.Contains(line.Qualifier, "table_only") {
				continue
			}
			form := ItemStatDescriptionForm{Text: statDescLuaEscapes.Replace(escapeGGGString(line.String))}
			for _, c := range line.Conditions {
				form.Limit = append(form.Limit, itemStatLimit(c))
			}
			form.Flags = itemStatFlags(line.Handlers)
			desc.Forms = append(desc.Forms, form)
		}
		out = append(out, desc)
	}
	return out
}

//pob:block:src/Export/Scripts/statdesc.lua:48-69
func itemStatLimit(c rawdata.Condition) [2]any {
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

//pob:block:src/Export/Scripts/statdesc.lua:70-97
func itemStatFlags(handlers []rawdata.IndexHandler) []ItemStatDescriptionFlag {
	var tokens []string
	for _, h := range handlers {
		tokens = append(tokens, h.Name)
		if h.Arg != "" {
			tokens = append(tokens, h.Arg)
		}
	}
	var flags []ItemStatDescriptionFlag
	for i := 0; i < len(tokens); {
		if tokens[i] == "canonical_line" {
			flags = append(flags, ItemStatDescriptionFlag{K: "canonical_line", V: true})
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
		flags = append(flags, ItemStatDescriptionFlag{K: tokens[i], V: v})
		i += 2
	}
	return flags
}

//pob:src/Modules/Common.lua:1110-1116
func escapeGGGString(text string) string {
	text = gggMarkupRe.ReplaceAllString(text, "$1")
	text = gggLinkRe.ReplaceAllString(text, "$1")
	return gggAliasRe.ReplaceAllString(text, "$1")
}
