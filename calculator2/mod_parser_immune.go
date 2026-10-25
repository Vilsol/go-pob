package calculator2

import (
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

var (
	immuneWordRe = regexp.MustCompile(`\S+`)
	immuneAndRe  = regexp.MustCompile(`^(.*?) and (.*)`)
)

//pob2:block:src/Modules/ModParser.lua:6534-6556
var statusToEffectMap = map[string]string{
	"bleeding":           "Bleed",
	"blinded":            "Blind",
	"chilled":            "Chill",
	"cursed":             "Curse",
	"curses":             "Curse",
	"elemental ailments": "ElementalAilment",
	"frozen":             "Freeze",
	"hindered":           "Hinder",
	"ignited":            "Ignite",
	"light stunned":      "Stun",
	"maimed":             "Maim",
	"poisoned":           "Poison",
	"shocked":            "Shock",
}

//pob2:src/Modules/ModParser.lua:20-28
func combineToUpper(str string) string {
	out := ""
	for _, word := range immuneWordRe.FindAllString(str, -1) {
		out += firstToUpper(word)
	}
	return out
}

// immuneForm is the IMMUNE branch of parseMod; the form carries no value, so every mod is FLAG true.
//
//pob2:src/Modules/ModParser.lua:6874-6925
func immuneForm(line string) (*modNameListType, []string, []float64, bool) {
	effectLine := strings.ToLower(strings.TrimRight(line, " \t\n\r\f\v"))
	numWords := len(immuneWordRe.FindAllString(effectLine, -1))
	multiEffect := strings.Contains(effectLine, " and ")

	getEffectFromStatus := func(statusString string) string {
		if e, ok := statusToEffectMap[strings.ToLower(statusString)]; ok {
			return e
		}
		return statusString
	}
	effectBlacklist := map[string]bool{
		"used manually": true,
	}

	if multiEffect {
		if numWords > 3 {
			m := immuneAndRe.FindStringSubmatch(effectLine)
			if len(immuneWordRe.FindAllString(m[1], -1)) > 2 || len(immuneWordRe.FindAllString(m[2], -1)) > 2 {
				return nil, nil, nil, false
			}
		}
	} else if numWords < 1 || numWords > 2 {
		return nil, nil, nil, false
	} else if effectBlacklist[effectLine] {
		return nil, nil, nil, false
	}

	if multiEffect {
		m := immuneAndRe.FindStringSubmatch(effectLine)
		e1 := combineToUpper(getEffectFromStatus(m[1]))
		e2 := combineToUpper(getEffectFromStatus(m[2]))
		return &modNameListType{names: []string{e1 + "Immune", e2 + "Immune"}}, []string{"FLAG", "FLAG"}, []float64{1, 1}, true
	}
	effect := combineToUpper(getEffectFromStatus(effectLine))
	return &modNameListType{names: []string{effect + "Immune"}}, []string{"FLAG"}, []float64{1}, true
}

// withTagActor is `newMod[1]["actor"] = "enemy"` on a copied mod; tags without an actor field are returned unchanged.
//
//pob2:block:src/Modules/ModParser.lua:7057-7061
func withTagActor(t mod.Tag, actor string) mod.Tag {
	switch tag := t.(type) {
	case *mod.MultiplierTag:
		c := tag.Clone().(*mod.MultiplierTag)
		c.TagActor = actor
		return c
	case *mod.ActorConditionTag:
		c := *tag
		c.Actor = &actor
		return &c
	case *mod.PerStatTag:
		c := *tag
		c.TagActor = actor
		return &c
	case *mod.PercentStatTag:
		c := *tag
		c.TagActor = actor
		return &c
	case *mod.MultiplierThresholdTag:
		c := *tag
		c.TagActor = actor
		return &c
	}
	return t
}
