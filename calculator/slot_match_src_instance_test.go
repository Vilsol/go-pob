package calculator

import (
	"os"
	"testing"
)

// TestSlotMatchSrcInstanceFromItem pins CalcTriggers.lua:33-35's srcInstance.fromItem disjuncts (set at CalcSetup.lua:1392 for every item-granted group).
func TestSlotMatchSrcInstanceFromItem(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1307-occultist-doom-blast-2.xml")
	if name := env.Player.MainSkill.ActiveEffect.GrantedEffect.Name(); name != "Doom Blast" {
		t.Fatalf("REFUSING: main skill is %q, not Doom Blast", name)
	}
	seen := map[string]bool{}
	for _, s := range env.Player.ActiveSkillList {
		switch name := s.ActiveEffect.GrantedEffect.Name(); name {
		case "Temporal Chains", "Punishment":
			seen[name] = true
			if got, want := slotMatch(env, s), name == "Temporal Chains"; got != want {
				t.Errorf("slotMatch(%s) = %v, want %v", name, got, want)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("REFUSING: expected both fixtures in the skill list, saw %v", seen)
	}
}
