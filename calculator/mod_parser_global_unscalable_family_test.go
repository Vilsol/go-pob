package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGlobalUnscalableFamilyCarriesTheTag: The `GlobalEffect{effectType = "Global", unscalable = true}` family.
//
//pob:src/Modules/ModParser.lua:4067
func TestGlobalUnscalableFamilyCarriesTheTag(t *testing.T) {
	carries := map[string]string{
		"Unaffected by Temporal Chains":                                                    "ModParser.lua:4067",
		"Cannot be Ignited while at maximum Endurance Charges":                             "ModParser.lua:4840",
		"Unaffected by Curses while affected by Zealotry":                                  "ModParser.lua:4914",
		"Take no Extra Damage from Critical Strikes if you have a Magic Ring in left slot": "ModParser.lua:5851",
		"Unaffected by Temporal Chains while affected by Haste":                            "ModParser.lua:4917",
	}
	lacks := map[string]string{
		"Take no Extra Damage from Critical Strikes if Energy Shield Recharge started Recently":   "ModParser.lua:5853",
		"You take 20% reduced Extra Damage from Critical Strikes while affected by Determination": "ModParser.lua:5855",
	}

	for line, cite := range carries {
		entry := ParseMod(line, false)
		if entry == nil || len(entry.ModList) == 0 {
			t.Errorf("%s (%s): parsed to no mods at all", line, cite)
			continue
		}
		if !anyModHasUnscalableGlobal(entry.ModList) {
			t.Errorf("%s (%s): no mod carries GlobalEffect{Global, unscalable}; the pin's row does, "+
				"so ScaleAddMod scales this mod here and adds it untouched upstream", line, cite)
		}
	}

	for line, cite := range lacks {
		entry := ParseMod(line, false)
		if entry == nil || len(entry.ModList) == 0 {
			t.Errorf("%s (%s): parsed to no mods at all -- a neighbour fixture that stops parsing "+
				"stops being a fixture", line, cite)
			continue
		}
		if anyModHasUnscalableGlobal(entry.ModList) {
			t.Errorf("%s (%s): carries GlobalEffect{Global, unscalable} and the pin's row does NOT "+
				"-- a row above has been widened until it swallowed this neighbour", line, cite)
		}
	}
}

func anyModHasUnscalableGlobal(mods []mod.Mod) bool {
	for _, m := range mods {
		for _, raw := range m.Tags() {
			if tag, ok := raw.(*mod.GlobalEffectTag); ok && tag.UnscalableTag && slices.Contains(tag.GlobalEffectList, "Global") {
				return true
			}
		}
	}
	return false
}
