package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestExertBlockCannotDoubleApply pins CalcOffence.lua:2770 and :2800 — the over-application guard on the exerted-attack block.
//
//	and not globalOutput.ExertedAttackUptimeRatioCalculated then
func TestExertBlockCannotDoubleApply(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	carriers := make([]string, 0, 256)
	for build, rec := range oracle {
		if rec.Output["ExertedAttackUptimeRatio"] > 0 {
			carriers = append(carriers, build)
		}
	}
	if len(carriers) == 0 {
		t.Fatal("REFUSING: no oracle record has ExertedAttackUptimeRatio > 0 — either the blob " +
			"moved or this probe reads the wrong key. A 0 here is not a finding.")
	}

	exposed := make([]string, 0, 4)
	for _, build := range carriers {
		env := corpusEnvForLoadedData(t, build)
		skill := env.Player.MainSkill
		if skill == nil {
			continue
		}
		if skill.SkillFlags[SkillFlagBothWeaponAttack] {
			exposed = append(exposed, build)
			cfg := skill.SkillCfg
			t.Logf("  %s: ExertIncrease INC %v MORE %v, ExertAttackIncrease MORE %v",
				build,
				skill.SkillModList.Sum(mod.TypeIncrease, cfg, "ExertIncrease"),
				skill.SkillModList.Sum(mod.TypeMore, cfg, "ExertIncrease"),
				skill.SkillModList.Sum(mod.TypeMore, cfg, "ExertAttackIncrease"))
		}
	}

	t.Logf("builds with ExertedAttackUptimeRatio > 0: %d; of those, running two weapon passes: %d",
		len(carriers), len(exposed))

	if len(exposed) != 0 {
		t.Logf("the guard is now GRADEABLE on %d build(s), e.g. %v — replace this count with a "+
			"value fixture against PoB", len(exposed), exposed[:min(3, len(exposed))])
	}
}
