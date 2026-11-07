package exposition

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

type ModLineStatus string

const (
	ModLineBlank       = ModLineStatus("blank")
	ModLineComment     = ModLineStatus("comment")
	ModLineParsed      = ModLineStatus("parsed")
	ModLineUnsupported = ModLineStatus("unsupported")
)

type ParsedMod struct {
	Name  string
	Type  string
	Value string
	Tags  []string
}

// ModLine is one custom modifier line as BuildModList reads it: parsed when parseMod leaves no remainder.
type ModLine struct {
	Status ModLineStatus
	Extra  string
	Mods   []ParsedMod
}

// CustomBlock is one entry of the active config set's customModsList.
type CustomBlock struct {
	Title   string
	Enabled bool
	Text    string
}

//pob2:block:src/Classes/ConfigTab.lua:1207-1210
func ParseModLines(lines []string) []ModLine {
	out := make([]ModLine, len(lines))
	for i, line := range lines {
		stripped := calculator2.CustomModLine(line)
		mods, extra := calculator2.ParseMod(stripped)
		if mods != nil && extra == "" {
			parsed := make([]ParsedMod, len(mods))
			for j, m := range mods {
				parsed[j] = parsedMod(m)
			}
			out[i] = ModLine{Status: ModLineParsed, Mods: parsed}
			continue
		}
		status := ModLineUnsupported
		switch {
		case stripped == "":
			status = ModLineBlank
		case strings.HasPrefix(stripped, "#"):
			status = ModLineComment
		}
		out[i] = ModLine{Status: status, Extra: extra}
	}
	return out
}

func parsedMod(m mod.Mod) ParsedMod {
	out := ParsedMod{Name: m.Name(), Type: string(m.Type())}
	if v := m.Value(); v != nil {
		switch v.Type() {
		case mod.ModValueMultiTypeFlag:
			out.Value = strconv.FormatBool(v.Flag())
		case mod.ModValueMultiTypeList:
			out.Value = mod.FormatValue(v.List())
		default:
			out.Value = strconv.FormatFloat(v.Float(), 'g', -1, 64)
		}
	}
	for _, tag := range m.Tags() {
		out.Tags = append(out.Tags, string(tag.Type()))
	}
	return out
}

func (c *Calculator) CustomModifierBlocks() []CustomBlock {
	blocks := calculator2.CustomModifierBlocks(c.build)
	out := make([]CustomBlock, len(blocks))
	for i, b := range blocks {
		out[i] = CustomBlock{Title: "Default", Enabled: b.Enabled == nil || *b.Enabled, Text: b.Text}
		if b.Title != nil {
			out[i].Title = *b.Title
		}
	}
	return out
}

func (c *Calculator) SetCustomModifierBlocks(blocks []CustomBlock) {
	out := make([]pob2.CustomModifierBlock, len(blocks))
	for i, b := range blocks {
		out[i] = pob2.CustomModifierBlock{Title: &b.Title, Enabled: &b.Enabled, Text: b.Text}
	}
	calculator2.SetCustomModifierBlocks(c.build, out)
}
