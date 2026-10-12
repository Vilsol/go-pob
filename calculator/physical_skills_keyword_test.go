package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWithPhysicalSkillsCarriesKeyword pins ModParser.lua:1053 `{ keywordFlags = KeywordFlag.Physical }`, which the pin DEFINES (Global.lua:140, 0x10).
func TestWithPhysicalSkillsCarriesKeyword(t *testing.T) {
	for line, want := range map[string]mod.KeywordFlag{
		"2% increased Attack and Cast Speed with Physical Skills": mod.KeywordFlagPhysical,
		"2% increased Attack and Cast Speed with Chaos Skills":    mod.KeywordFlagChaos,
	} {
		e := ParseMod(line, false)
		if e.Extra != "" || len(e.ModList) != 1 {
			t.Errorf("%q: remainder %q, %d mods", line, e.Extra, len(e.ModList))
			continue
		}
		m := e.ModList[0]
		if m.Name() != "Speed" || m.Type() != mod.TypeIncrease || m.KeywordFlags() != want {
			t.Errorf("%q: got %s %s kw=%v, want Speed INC kw=%v", line, m.Name(), m.Type(), m.KeywordFlags(), want)
		}
	}
}
