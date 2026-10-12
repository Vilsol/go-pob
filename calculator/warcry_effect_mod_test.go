package calculator

import (
	"math"
	"os"
	"testing"
)

// TestWarcryEffectModIsPublished pins CalcOffence.lua:1185-1191.
//
//	if activeSkill.skillTypes[SkillType.Warcry] then
//		local full_duration = calcSkillDuration(...)
//		local cooldownOverride = skillModList:Override(skillCfg, "CooldownRecovery")
//		local actual_cooldown = cooldownOverride or (skillData.cooldown + Sum "CooldownRecovery")
//		local uptime = env.modDB:Flag(nil, "Condition:WarcryMaxHit") and 1
//		               or m_min(full_duration / actual_cooldown, 1)
//		local unscaledEffect = calcLib.mod(skillModList, skillCfg, "WarcryEffect", "BuffEffect")
//	end
func TestWarcryEffectModIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawBelowOne := false
	for _, build := range []string{
		"0473-luminary-battlemages-cry.xml",
		"1648-luminary-intimidating-cry.xml",
		"1899-ladder-Synetik_7053.Kurael_EnduCry.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["WarcryEffectMod"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero WarcryEffectMod for %s; an unwritten "+
					"key agrees with 0, so this fixture cannot fail", build)
			}
			if want < 1 {
				sawBelowOne = true
			}

			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["WarcryEffectMod"]
			if !present {
				t.Errorf("%s: we never write WarcryEffectMod at all; PoB reports %v", build, want)
				return
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("%s: WarcryEffectMod = %v — the uptime division is unguarded upstream "+
					"too, so a NaN here is a real input problem, not a missing guard", build, got)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: WarcryEffectMod = %v, PoB reports %v", build, got, want)
			}
		})
	}

	if !sawBelowOne {
		t.Errorf("REFUSING: no fixture has WarcryEffectMod < 1 any more, so this test can no " +
			"longer observe the uptime term at all. Replace a fixture.")
	}
}
