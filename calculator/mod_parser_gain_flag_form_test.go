package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGainFlagFormParses pins the three FLAG forms of ModParser.lua:149-153 that were absent here: `^have `, `^gain ` and `^you gain `.
func TestGainFlagFormParses(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line string
		want string
	}{
		{"Gain 20% of Physical Damage as Extra Fire Damage", "PhysicalDamageGainAsFire"},
		{"You gain 5 Mana per Enemy Killed", "ManaOnKill"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
			t.Errorf("CONTROL MOVED: %q no longer parses whole", tc.line)
			continue
		}
		var names []string
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		if !contains(names, tc.want) {
			t.Errorf("CONTROL MOVED: %q -> %v, want %q", tc.line, names, tc.want)
		}
	}

	for _, tc := range []struct {
		line  string
		want  string
		conds []string
	}{
		{"Gain Arcane Surge when you deal a Critical Strike", "Condition:ArcaneSurge", []string{"CritRecently"}},
		{"Gain Arcane Surge on Hit with Spells", "Condition:ArcaneSurge", []string{"HitSpellRecently"}},
		{"You gain Onslaught for 4 seconds on Kill while affected by Haste", "Condition:Onslaught", []string{"KilledRecently", "AffectedByHaste"}},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("DROPPED WHOLE (no mods): %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q, so the line is discarded: %s", extra, tc.line)
			continue
		}
		var found mod.Mod
		for _, m := range e.ModList {
			if m.Name() == tc.want {
				found = m
			}
		}
		if found == nil {
			t.Errorf("no %q produced: %s", tc.want, tc.line)
			continue
		}
		var conds []string
		for _, tag := range found.Tags() {
			if c, ok := tag.(*mod.ConditionTag); ok {
				conds = append(conds, c.VarList...)
			}
		}
		for _, want := range tc.conds {
			if !contains(conds, want) {
				t.Errorf("%q -> %s conditions %v, want %q", tc.line, tc.want, conds, want)
			}
		}
	}
}
