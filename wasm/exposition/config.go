package exposition

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
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

type ModLine struct {
	Status ModLineStatus
	Extra  string
	Mods   []ParsedMod
}

func ParseModLines(lines []string) []ModLine {
	out := make([]ModLine, len(lines))
	for i, line := range lines {
		out[i] = parseModLine(line)
	}
	return out
}

func parseModLine(line string) ModLine {
	stripped := strings.TrimSpace(calculator.StripEscapes(line))
	entry := calculator.ParseCustomModLine(line)
	if entry != nil && entry.ModList != nil && entry.Extra == "" {
		mods := make([]ParsedMod, len(entry.ModList))
		for i, m := range entry.ModList {
			mods[i] = parsedMod(m)
		}
		return ModLine{Status: ModLineParsed, Mods: mods}
	}
	status := ModLineUnsupported
	switch {
	case stripped == "":
		status = ModLineBlank
	case strings.HasPrefix(stripped, "#"):
		status = ModLineComment
	}
	var extra string
	if entry != nil {
		extra = entry.Extra
	}
	return ModLine{Status: status, Extra: extra}
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

func ConfigPlaceholders(build *pob.PathOfBuilding) map[string]float64 {
	return calculator.ConfigPlaceholders(build)
}

func ModBrowserList(search string) []string {
	return calculator.ModBrowserListFuzzy(search)
}

func ModBrowserSupported(line string) bool {
	return calculator.ModBrowserSupported(line)
}

func ModBrowserSources(line string) []string {
	return calculator.ModBrowserSources(line)
}
