package calculator2

import (
	"encoding/json"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

// Expected values are the entries of PoB-PoE2's src/Data/ModCache.lua for the same line.

func TestModCacheLines(t *testing.T) {
	mods, extra := ModCache("+5 to Strength")
	testza.AssertEqual(t, "", extra)
	testza.AssertEqual(t, 1, len(mods))
	testza.AssertEqual(t, "Str", mods[0].Name())
	testza.AssertEqual(t, mod.TypeBase, mods[0].Type())
	testza.AssertEqual(t, float64(5), mods[0].Value().Float())

	mods, _ = ModCache("+0.5% to Thorns Critical Hit Chance per 50 Tribute")
	testza.AssertEqual(t, mod.MFlag(32), mods[0].Flags())
	ps := mods[0].Tags()[0].(*mod.PerStatTag)
	testza.AssertEqual(t, []string{"Tribute"}, ps.StatList)
	testza.AssertEqual(t, float64(50), ps.Divide)
	testza.AssertEqual(t, "parent", ps.TagActor)

	mods, _ = ModCache("+1 to maximum number of Summoned Ballista Totems")
	testza.AssertEqual(t, []string{"TotemsAreBallistae"}, mods[0].Tags()[0].(*mod.SkillTypeTag).SkillTypeList)

	mods, extra = ModCache("(2-3)% increased Cast Speed")
	testza.AssertNil(t, mods)
	testza.AssertEqual(t, "(2-3)% increased Cast Speed ", extra)

	mods, extra = ModCache("a line PoB never cached")
	testza.AssertNil(t, mods)
	testza.AssertEqual(t, "a line PoB never cached", extra)
}

func TestModCacheReturnsCopies(t *testing.T) {
	a, _ := ModCache("8% increased maximum Life")
	a[0].Value().SetFloat(99)
	b, _ := ModCache("8% increased maximum Life")
	testza.AssertEqual(t, float64(8), b[0].Value().Float())
}

// Each cached line converts unless its mods carry a tag field or payload the mod package cannot
// express; the ceiling only goes down.
func TestModCacheUnsupportedCeiling(t *testing.T) {
	const ceiling = 11
	bad := 0
	for _, r := range poe2.ModCacheRows {
		if err := ModCacheUnsupported(r.Line); err != nil {
			bad++
			t.Log(r.Line, ": ", err)
		}
	}
	testza.AssertTrue(t, bad <= ceiling, bad)
	if bad < ceiling {
		t.Errorf("only %d cached lines are unsupported now: lower the ceiling to %d", bad, bad)
	}
}

func TestModCacheStringPercent(t *testing.T) {
	mods, _ := ModCache("Attacks with this Weapon have Added Cold Damage equal to 7% to 11% of maximum Mana")
	ps := mods[0].Tags()[0].(*mod.PercentStatTag)
	testza.AssertEqual(t, float64(7), ps.Percent)
}

func TestModCacheStringValue(t *testing.T) {
	mods, _ := ModCache("When taking damage from Hits, 20% of Life Loss is prevented, then 150% of Life Loss prevented this way is Lost over 4 seconds")
	found := false
	for _, m := range mods {
		if m.Name() == "LifeLossLost" {
			found = true
			testza.AssertEqual(t, float64(150), m.Value().Float())
		}
	}
	testza.AssertTrue(t, found)
}

func skillOverlayMods() []string {
	var out []string
	add := func(entries []poe2.StatMapEntry) {
		for _, e := range entries {
			for _, it := range e.Items {
				if it.Group == nil {
					out = append(out, it.Mod)
				}
				out = append(out, it.Group...)
			}
		}
	}
	add(poe2.SkillStatMap)
	for _, s := range poe2.SkillOverlays {
		for _, set := range s.StatSets {
			out = append(out, set.BaseMods...)
			add(set.StatMap)
		}
	}
	return out
}

// Every skill mod PoB-PoE2 writes in its skill files and SkillStatMap converts, a nil value included:
// the stat's value is set when it is merged (CalcActiveSkill.lua mergeStatSet).
func TestCacheModSkillMods(t *testing.T) {
	const ceiling = 11
	mods := skillOverlayMods()
	bad := 0
	for _, s := range mods {
		var r map[string]any
		if err := json.Unmarshal([]byte(s), &r); err != nil {
			t.Fatal(err)
		}
		if _, err := cacheMod(r); err != nil {
			bad++
			t.Log(s, ": ", err)
		}
	}
	t.Logf("%d skill mods, %d do not convert", len(mods), bad)
	testza.AssertTrue(t, bad <= ceiling, bad)
	if bad < ceiling {
		t.Errorf("the ceiling is %d, %d do not convert: lower it", ceiling, bad)
	}
}

// Expected shapes are PoB-PoE2's makeSkillMod / makeSkillDataMod tables (Data.lua 56-71) for the same
// skill-file lines: mod("PresenceArea", "INC", nil), skill("radius", nil), skill("showAverage", true).
func TestCacheModNilValue(t *testing.T) {
	decode := func(s string) mod.Mod {
		var r map[string]any
		if err := json.Unmarshal([]byte(s), &r); err != nil {
			t.Fatal(err)
		}
		m, err := cacheMod(r)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	m := decode(`{"name":"PresenceArea","type":"INC","value":null,"flags":0,"keywordFlags":0,"tags":[]}`)
	testza.AssertEqual(t, mod.TypeIncrease, m.Type())
	testza.AssertEqual(t, float64(0), m.Value().Float())
	testza.AssertEqual(t, mod.ModValueMultiTypeFloat, m.Value().Type())

	m = decode(`{"name":"SkillData","type":"LIST","value":{"key":"radius"},"flags":0,"keywordFlags":0,"tags":[]}`)
	testza.AssertEqual(t, mod.SkillData{Key: "radius"}, m.Value().List())
	m = decode(`{"name":"SkillData","type":"LIST","value":{"key":"decay","merge":"MAX"},"flags":0,"keywordFlags":0,"tags":[]}`)
	testza.AssertEqual(t, mod.SkillData{Key: "decay", Merge: "MAX"}, m.Value().List())
	m = decode(`{"name":"SkillData","type":"LIST","value":{"key":"showAverage","value":true},"flags":0,"keywordFlags":0,"tags":[]}`)
	testza.AssertEqual(t, mod.SkillData{Key: "showAverage", Value: 1}, m.Value().List())
}
