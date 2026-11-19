package calculator

import (
	"math"
	"os"
	"testing"
)

// TestZeroCooldownRecoveryIsNotOne pins CalcTriggers.lua:578: calcLib.mod never returns nil, so a 0 cooldown
// recovery multiplier stays 0 and PoB's trigger cooldown on this edited build is nan.
func TestZeroCooldownRecoveryIsNotOne(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	const anchor = "40% increased Armour while not Ignited, Frozen or Shocked\n"
	env := corpusEnvWithEdit(t, "0604-chieftain-detonate-dead-2", anchor, anchor+"100% reduced Cooldown Recovery Rate\n")
	if name := env.Player.MainSkill.ActiveEffect.GrantedEffect.Name(); name != "Detonate Dead" {
		t.Fatalf("PREMISE BROKEN: main skill %s", name)
	}
	// PoB's dump of this edit: TriggerRateCap and SkillTriggerRate are nan.
	for _, stat := range []string{"TriggerRateCap", "SkillTriggerRate"} {
		if v := env.Player.Output[stat]; !math.IsNaN(v) {
			t.Errorf("%s = %v, PoB nan", stat, v)
		}
	}
}
