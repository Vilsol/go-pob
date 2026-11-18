package calculator

import "testing"

// TestRuthlessBlowSkipsAMirageWithFewUses: a mirage copy rolls Ruthless only when its stored uses exceed the Ruthless
// count. With The Saviour's Reflection as the main skill, its mirage warriors use the Ruthless-supported Double Strike,
// and the pinned PoB dumps outputMainHand.RuthlessBlowMaxCount = 3 with RuthlessBlowChance = 0.
//
//pob:block:src/Modules/CalcOffence.lua:3012,src/Modules/CalcMirages.lua:46
func TestRuthlessBlowSkipsAMirageWithFewUses(t *testing.T) {
	env := corpusEnvWithEdit(t, "0845-champion-double-strike-of-impaling", `mainSocketGroup="1"`, `mainSocketGroup="11"`)
	if !env.Player.MainSkill.SkillCfg.SkillCond["usedByMirage"] {
		t.Fatal("premise broken: the main skill is not a mirage copy")
	}
	mh := env.Player.OutputTable[OutTableMainHand]
	if got := mh["RuthlessBlowMaxCount"]; got != 3 {
		t.Fatalf("premise broken: RuthlessBlowMaxCount = %v, PoB 3", got)
	}
	if got := mh["RuthlessBlowChance"]; got != 0 {
		t.Errorf("RuthlessBlowChance = %v, PoB 0", got)
	}
	if got := mh["RuthlessBlowHitEffect"]; got != 1 {
		t.Errorf("RuthlessBlowHitEffect = %v, PoB 1", got)
	}
}
