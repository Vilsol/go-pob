package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTriggerAndBuffWorklistRowsParse pins the second half of the parser worklist: five item-granted trigger supports, two buff rows and one no-op tag.
func TestTriggerAndBuffWorklistRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line string
		want string
		aux  string
	}{
		{"Trigger a Socketed Elemental Spell on Block, with a 0.25 second Cooldown", "ExtraSupport", "SupportTriggerElementalSpellOnBlock"},
		{"Trigger a Socketed Spell when you Block, with a 0.25 second Cooldown", "ExtraSupport", "SupportUniqueVirulenceSpellsCastOnBlock"},
		{"Trigger a Socketed Spell on Unarmed Melee Critical Strike, with a 0.25 second Cooldown", "ExtraSupport", "SupportTriggerSpellOnUnarmedMeleeCriticalHit"},
		{"Trigger Socketed Curse Spell when you Cast a Curse Spell, with a 0.25 second Cooldown", "ExtraSupport", "SupportUniqueCastCurseOnCurse"},
		{"20% chance to Trigger Socketed Spell on Kill, with a 0.5 second Cooldown", "ExtraSupport", "SupportTriggerSpellOnKill"},
		{"Eat a Soul when you Hit a Rare or Unique Enemy, no more than once every 0.5 seconds", "Condition:CanHaveSoulEater", ""},
		{"Your Hits Intimidate Enemies for 4 seconds while you are using Pride", "EnemyModifier", "AffectedByPride"},
		{"Gain Arcane Surge for 4 seconds when you create Consecrated Ground while affected by Zealotry", "Condition:ArcaneSurge", "AffectedByZealotry"},
		{"With a Hypnotic Eye Jewel Socketed, gain Arcane Surge on Hit with Spells", "Condition:ArcaneSurge", "HaveHypnoticEyeJewelIn{SlotName}"},
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
		if tc.aux == "" {
			continue
		}
		var got []string
		if found.Value().Type() == mod.ModValueMultiTypeList {
			if es, ok := found.Value().List().(mod.ExtraSupport); ok {
				got = append(got, es.SkillID)
			}
		}
		for _, tag := range found.Tags() {
			if c, ok := tag.(*mod.ConditionTag); ok {
				got = append(got, c.VarList...)
			}
		}
		if !contains(got, tc.aux) {
			t.Errorf("%q -> %q carries %v, want %q", tc.line, tc.want, got, tc.aux)
		}
	}
}
