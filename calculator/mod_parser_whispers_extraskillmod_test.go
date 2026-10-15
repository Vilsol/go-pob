package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionWhispersEmitsExtraSkillMod pins ModParser.lua:4477-4480.
//
//pob:block:src/Modules/ModParser.lua:4523-4526
func TestMinionWhispersEmitsExtraSkillMod(t *testing.T) {
	requireGameData(t, "3.29")

	ctl := ParseMod("Minions' Base Attack Critical Strike Chance is equal to the Critical Strike Chance of your Main Hand Weapon", false)
	if ctl == nil || len(ctl.ModList) != 1 || ctl.ModList[0].Name() != "MinionModifier" {
		t.Fatalf("REFUSING: the neighbouring MinionModifier row (ModParser.lua:4475) no longer parses as one")
	}

	const line = "Minions can hear the whispers for 5 seconds after they deal a Critical Strike"
	e := ParseMod(line, false)
	if e == nil || len(e.ModList) == 0 {
		t.Fatalf("no mods at all: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q: %s", extra, line)
	}
	if len(e.ModList) != 3 {
		t.Fatalf("got %d mods, want 3", len(e.ModList))
	}

	want := []struct {
		name    string
		modType string
		value   float64
		tags    []string
	}{
		{"Speed", "INC", 50, []string{"GlobalEffect/effectType=Global/unscalable=true", "Condition/neg=true/vars=NeverCrit"}},
		{"Damage", "INC", 50, []string{"GlobalEffect/effectType=Global/unscalable=true", "Condition/neg=true/vars=NeverCrit"}},
		{"ChaosDegen", "BASE", 1, []string{"PercentStat/percent=20/stats=Life", "GlobalEffect/effectType=Global/unscalable=true", "Condition/neg=true/vars=NeverCrit"}},
	}

	for i, m := range e.ModList {
		if m.Name() != "ExtraSkillMod" {
			t.Fatalf("mod %d is %q, want ExtraSkillMod — a MinionModifier carries the right NAME to "+
				"every census while reaching a different consumer entirely", i, m.Name())
		}
		esm, ok := m.Value().List().(mod.ExtraSkillMod)
		if !ok {
			t.Fatalf("mod %d payload is %T, want mod.ExtraSkillMod", i, m.Value().List())
		}
		inner := esm.Mod
		w := want[i]
		if inner.Name() != w.name || string(inner.Type()) != w.modType || inner.Value().Float() != w.value {
			t.Fatalf("mod %d inner = %s|%s|%v, want %s|%s|%v",
				i, inner.Name(), inner.Type(), inner.Value().Float(), w.name, w.modType, w.value)
		}
		if f := inner.Flags(); f != 0 {
			t.Fatalf("mod %d inner flags %#x, want 0 — ModCache.lua:10057 records flags=0 on all "+
				"three; a ModFlag here narrows a buff upstream applies to everything", i, f)
		}
		got := collectCanonTags([]mod.Mod{inner})
		for _, wt := range w.tags {
			if !contains(got, wt) {
				t.Fatalf("mod %d (%s) tags %v, want %q", i, w.name, got, wt)
			}
		}
		if len(got) != len(w.tags) {
			t.Fatalf("mod %d (%s) has %d tags %v, want exactly %d — an EXTRA tag over-restricts",
				i, w.name, len(got), got, len(w.tags))
		}
	}
}
