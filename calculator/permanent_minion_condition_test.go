package calculator

import (
	"os"
	"testing"
)

// TestPermanentMinionDoesNotSetUsedMinionSkillRecently pins the conjunct dropped from CalcPerform.lua:249.
//
//	if actor.mainSkill.skillFlags.minion and not actor.mainSkill.skillFlags.permanentMinion then
//	end
func TestPermanentMinionDoesNotSetUsedMinionSkillRecently(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0019-champion-herald-of-agony.xml")
	ms := env.Player.MainSkill
	if ms == nil {
		t.Fatal("fixture is useless: no main skill")
	}
	if !ms.SkillFlags[SkillFlagMinion] || !ms.SkillFlags[SkillFlagPermanentMinion] {
		t.Fatalf("fixture is useless: main skill flags minion=%v permanentMinion=%v — this build no "+
			"longer reaches the dropped conjunct",
			ms.SkillFlags[SkillFlagMinion], ms.SkillFlags[SkillFlagPermanentMinion])
	}

	if env.Player.ModDB.Conditions["UsedMinionSkillRecently"] {
		t.Errorf("UsedMinionSkillRecently is set for a PERMANENT minion — CalcPerform.lua:249 " +
			"withholds it via `and not skillFlags.permanentMinion`")
	}
}

// TestNonPermanentMinionStillSetsUsedMinionSkillRecently is the CONTROL and must pass in BOTH states.
func TestNonPermanentMinionStillSetsUsedMinionSkillRecently(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0147-necromancer-animate-weapon-of-ranged-arms.xml")
	ms := env.Player.MainSkill
	if ms == nil || !ms.SkillFlags[SkillFlagMinion] || ms.SkillFlags[SkillFlagPermanentMinion] {
		t.Fatalf("CONTROL IS VACUOUS: this build must be a NON-permanent minion (minion=%v permanentMinion=%v). A skipped control is not a control.", ms.SkillFlags[SkillFlagMinion], ms.SkillFlags[SkillFlagPermanentMinion])
	}
	if !env.Player.ModDB.Conditions["UsedMinionSkillRecently"] {
		t.Error("CONTROL: a non-permanent minion skill must still set UsedMinionSkillRecently")
	}
}
