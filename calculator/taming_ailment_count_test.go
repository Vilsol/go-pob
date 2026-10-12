package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTamingElementalAilmentExpansion: The Taming's "(30-40)% increased Elemental Damage with Hits and Ailments for each type of Elemental Ailment on Enemy" had no parser row at all.
func TestTamingElementalAilmentExpansion(t *testing.T) {
	t.Parallel()
	requireGameData(t, "3.29")

	entry := ParseMod("40% increased Elemental Damage with Hits and Ailments for each type of Elemental Ailment on Enemy", false)
	if entry == nil {
		t.Fatal("REFUSING: ParseMod returned nil for the line")
	}
	if entry.Extra != "" {
		t.Fatalf("REFUSING: the line did not parse WHOLE — unparsed remainder %q, which discards "+
			"the entire line rather than part of it", entry.Extra)
	}
	mods := entry.ModList
	if len(mods) == 0 {
		t.Fatal("REFUSING: the line parsed to NO mods — a row that matches and emits an empty list " +
			"is worse than a missing row, because every missing-thing instrument then reports it present")
	}

	want := map[string]bool{
		"Frozen": false, "Chilled": false, "Ignited": false, "Shocked": false,
		"Scorched": false, "Brittle": false, "Sapped": false,
	}
	for _, m := range mods {
		if m.Name() != "ElementalDamage" || m.Type() != mod.TypeIncrease {
			t.Errorf("unexpected mod %q %v, want ElementalDamage INC", m.Name(), m.Type())
			continue
		}
		if m.Value().Float() != 40 {
			t.Errorf("ElementalDamage INC value = %v, want 40", m.Value().Float())
		}
		if kw := m.KeywordFlags(); kw&mod.KeywordFlagHit == 0 || kw&mod.KeywordFlagAilment == 0 {
			t.Errorf("mod is missing the Hit|Ailment keyword flags (got %v)", kw)
		}
		var found string
		for _, tg := range m.Tags() {
			ac, ok := tg.(*mod.ActorConditionTag)
			if !ok {
				continue
			}
			for _, v := range ac.VariableList {
				if _, known := want[v]; known {
					found = v
				}
			}
		}
		if found == "" {
			t.Errorf("mod carries no enemy ActorCondition naming an elemental ailment")
			continue
		}
		if want[found] {
			t.Errorf("ailment %q emitted twice", found)
		}
		want[found] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("no mod gated on enemy ActorCondition %q — the expansion is short of the "+
				"seven elemental ailments ModParser.lua:3582-3588 emits", k)
		}
	}
}
