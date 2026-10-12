package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRepeatFinalRowsParse pins ModParser.lua:5574 and :5673, the two producers that were absent while calc_offence.go:1104-1135 already consumed them.
func TestRepeatFinalRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line     string
		name     string
		modType  mod.Type
		value    float64
		flags    mod.MFlag
		keyword  mod.KeywordFlag
		condNeg  string
		condList []string
	}{
		{
			line: "Final Repeat of Spells has 40% increased Area of Effect",
			name: "RepeatFinalAreaOfEffect", modType: mod.TypeIncrease, value: 40,
			flags: mod.MFlagSpell, keyword: 0,
			condNeg: "CastOnFrostbolt", condList: []string{"averageRepeat", "alwaysFinalRepeat"},
		},
		{
			line: "Final Repeat of Attack Skills deals 60% more Damage",
			name: "RepeatFinalDamage", modType: mod.TypeMore, value: 60,
			flags: 0, keyword: mod.KeywordFlagAttack,
		},
		{
			line: "12% increased Area of Effect", name: "AreaOfEffect",
			modType: mod.TypeIncrease, value: 12,
		},
		{
			line: "40% increased Damage", name: "Damage",
			modType: mod.TypeIncrease, value: 40,
		},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("DROPPED WHOLE: %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q, so the line is discarded: %s", extra, tc.line)
			continue
		}
		var found mod.Mod
		for _, m := range e.ModList {
			if m.Name() == tc.name {
				found = m
			}
		}
		if found == nil {
			names := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				names = append(names, m.Name())
			}
			t.Errorf("%q produced %v, want a %q", tc.line, names, tc.name)
			continue
		}
		if found.Type() != tc.modType {
			t.Errorf("%q -> %s|%s, want %s", tc.line, tc.name, found.Type(), tc.modType)
		}
		if got := found.Value().Float(); got != tc.value {
			t.Errorf("%q -> value %v, want %v", tc.line, got, tc.value)
		}
		if found.Flags() != tc.flags {
			t.Errorf("%q -> ModFlags %d, want %d (ModCache.lua records it)", tc.line, found.Flags(), tc.flags)
		}
		if found.KeywordFlags() != tc.keyword {
			t.Errorf("%q -> KeywordFlags %d, want %d", tc.line, found.KeywordFlags(), tc.keyword)
		}
		if tc.condNeg == "" && tc.condList == nil {
			continue
		}
		var sawNeg, sawList bool
		for _, tag := range found.Tags() {
			c, ok := tag.(*mod.ConditionTag)
			if !ok {
				continue
			}
			if c.Negative && contains(c.VarList, tc.condNeg) {
				sawNeg = true
			}
			if !c.Negative && len(tc.condList) > 0 && contains(c.VarList, tc.condList[0]) &&
				contains(c.VarList, tc.condList[1]) {
				sawList = true
			}
		}
		if !sawNeg {
			t.Errorf("%q is missing the negated %q condition — without it the mod applies on "+
				"Cast on Frostbolt builds, where upstream withholds it", tc.line, tc.condNeg)
		}
		if !sawList {
			t.Errorf("%q is missing the %v condition — without it the re-emitted AreaOfEffect "+
				"applies in every repeat mode", tc.line, tc.condList)
		}
	}
}
