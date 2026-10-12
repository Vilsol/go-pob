package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTriggerOnCritArms pins the two call sites in this tree that matched a "trigger ...
func TestTriggerOnCritArms(t *testing.T) {
	requireGameData(t, "3.29")

	if mods := parsedNames(t, "Trigger Level 20 Summon Spectral Wolf on Critical Strike"); !contains(mods, "ExtraSkillMod") {
		t.Fatalf("REFUSING: an already-ported onCrit sibling no longer emits the ExtraSkillMod")
	}

	for _, tc := range []struct {
		name string
		line string
	}{
		{"chanceWithThisWeapon", "20% chance to Trigger Level 25 Summon Spectral Wolf on Critical Strike with this Weapon"},
		{"commandmentOfInferno", "Trigger Commandment of Inferno on Critical Strike"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all: %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q: %s", extra, tc.line)
			}
			var got []string
			var onCrit bool
			for _, m := range e.ModList {
				got = append(got, m.Name())
				if esm, ok := m.Value().List().(mod.ExtraSkillMod); ok {
					if sd, ok := esm.Mod.Value().List().(mod.SkillData); ok && sd.Key == "triggerOnCrit" {
						onCrit = true
					}
				}
			}
			if !contains(got, "ExtraSkill") {
				t.Fatalf("mods %v, want ExtraSkill: %s", got, tc.line)
			}
			if !onCrit {
				t.Fatalf("mods %v carry no ExtraSkillMod{SkillData triggerOnCrit} — the skill is "+
					"granted and fires on EVERY hit, not only on crits: %s", got, tc.line)
			}
		})
	}
}

func parsedNames(t *testing.T, line string) []string {
	t.Helper()
	e := ParseMod(line, false)
	if e == nil {
		return nil
	}
	var out []string
	for _, m := range e.ModList {
		out = append(out, m.Name())
	}
	return out
}
