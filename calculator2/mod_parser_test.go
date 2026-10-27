package calculator2

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// Expected mods are rows of PoB-PoE2's shipped src/Data/ModCache.lua.
func TestParseModMatchesPoBRows(t *testing.T) {
	type want struct {
		name  string
		typ   mod.Type
		value float64
		flags mod.MFlag
		extra string
	}
	cases := map[string]want{
		"+10% to Chaos Resistance":                               {"ChaosResist", mod.TypeBase, 10, 0, ""},
		"+65 to maximum Life":                                    {"Life", mod.TypeBase, 65, 0, ""},
		"+5% to Block chance":                                    {"BlockChance", mod.TypeBase, 5, 0, ""},
		"10% increased Thorns damage":                            {"Damage", mod.TypeIncrease, 10, ModFlagThorns, ""},
		"+100 to Deflection Rating per 50 missing Energy Shield": {"DeflectionRating", mod.TypeBase, 100, 0, "per 50 missing Energy Shield"},
	}
	for line, w := range cases {
		mods, extra := ParseMod(line)
		if len(mods) != 1 {
			t.Errorf("%q: %d mods, want 1", line, len(mods))
			continue
		}
		m := mods[0]
		if m.Name() != w.name || m.Type() != w.typ || m.Value().Float() != w.value || m.Flags() != w.flags || len(m.Tags()) != 0 {
			t.Errorf("%q: got %s %s %v f%d tags %d, want %s %s %v f%d", line, m.Name(), m.Type(), m.Value().Float(), m.Flags(), len(m.Tags()), w.name, w.typ, w.value, w.flags)
		}
		if strings.TrimSpace(extra) != w.extra {
			t.Errorf("%q: remainder %q, want %q", line, extra, w.extra)
		}
	}

	// PoB caches {nil, line}: no form matches a range text.
	if mods, extra := ParseMod("(2-3)% increased Movement Speed"); mods != nil || strings.TrimSpace(extra) != "(2-3)% increased Movement Speed" {
		t.Errorf("range text: got %d mods, remainder %q", len(mods), extra)
	}
}

// Expected payloads are PoB-PoE2's src/Data/ModCache.lua rows; the scaled value is ScaleAddMod's
// floor for a "level" key (ModStore.lua 99-100) at 200% increased rune effect.
func TestParseModGemPropertyPayload(t *testing.T) {
	cases := map[string]mod.GemProperty{
		"+2 to Level of all Skills with an Intelligence requirement": {Key: "level", Value: 2, KeyOfScaledMod: "value", GemRequirements: map[string]float64{"reqInt": 1}},
		"+2 to Level of all Skills with a Dexterity requirement":     {Key: "level", Value: 2, KeyOfScaledMod: "value", GemRequirements: map[string]float64{"reqDex": 1}},
		"+3 to Level of all Spell Skills":                            {Key: "level", Value: 3, KeyOfScaledMod: "value"},
	}
	for key, w := range cases {
		line := key
		mods, _ := ParseMod(line)
		if cached, _ := ModCache(line); len(cached) == 1 {
			if g, ok := cached[0].Value().ValueList.(mod.GemProperty); !ok || len(g.GemRequirements) != len(w.GemRequirements) || g.KeyOfScaledMod != w.KeyOfScaledMod {
				t.Errorf("%q: cached %+v, want %+v", line, cached[0].Value().ValueList, w)
			}
		} else {
			t.Errorf("%q: the shipped cache gives %d mods, want 1", line, len(cached))
		}
		if len(mods) != 1 {
			t.Fatalf("%q: %d mods, want 1", line, len(mods))
		}
		g, ok := mods[0].Value().ValueList.(mod.GemProperty)
		if !ok {
			t.Fatalf("%q: value %T, want mod.GemProperty", line, mods[0].Value().ValueList)
		}
		if g.Key != w.Key || g.Value != w.Value || g.KeyOfScaledMod != w.KeyOfScaledMod || len(g.GemRequirements) != len(w.GemRequirements) {
			t.Errorf("%q: got %+v, want %+v", line, g, w)
		}
		for k, v := range w.GemRequirements {
			if g.GemRequirements[k] != v {
				t.Errorf("%q: gemRequirements[%s] = %v, want %v", line, k, g.GemRequirements[k], v)
			}
		}
	}

	for name, parse := range map[string]func(string) ([]mod.Mod, string){"parser": ParseMod, "cache": ModCache} {
		mods, _ := parse("+3 to Level of all Spell Skills")
		if len(mods) != 1 {
			t.Fatalf("%s: %d mods, want 1", name, len(mods))
		}
		var scaled []mod.Mod
		scaleAddMod(&scaled, mods[0], 2)
		if g := scaled[0].Value().ValueList.(mod.GemProperty); g.Value != 6 {
			t.Errorf("%s, scaled by 2: value %v, want 6", name, g.Value)
		}
	}
}
