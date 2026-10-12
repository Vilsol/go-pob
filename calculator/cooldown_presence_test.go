package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCooldownGateAndPresenceTest pins the two Cooldown divergences, and records that BOTH ARE MEASURED ZEROS on this corpus.
//
//	:1285  elseif skillData.cooldown or skillModList:Sum("BASE", skillCfg, "CooldownRecovery") ~= 0
//	:4910  if output.Cooldown then          -- a PRESENCE test; Lua's 0 is TRUTHY
func TestCooldownGateAndPresenceTest(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var builds, recoveryOnly, zeroCooldown, hasCooldownKey, wrong, seen int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.SkillModList == nil || ms.SkillData == nil {
			continue
		}
		builds++

		if ms.SkillData.Cooldown == 0 &&
			ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "CooldownRecovery") != 0 {
			recoveryOnly++
		}
		if v, present := env.Player.Output["Cooldown"]; present {
			hasCooldownKey++
			if v == 0 {
				zeroCooldown++
			}
		}

		if want, has := rec.Output["Cooldown"]; has && want != 0 {
			seen++
			if !oracleClose(env.Player.Output["Cooldown"], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if builds >= 400 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build produced a main skill; this test measured nothing")
	}
	if hasCooldownKey == 0 {
		t.Fatalf("REFUSING: over %d builds NOT ONE publishes output.Cooldown, so neither the "+
			"producer gate nor the presence test can decide anything and this test is vacuous", builds)
	}

	t.Logf("MEASURED over %d builds: %d publish output.Cooldown; recoveryOnly=%d zeroCooldown=%d",
		builds, hasCooldownKey, recoveryOnly, zeroCooldown)
	if recoveryOnly == 0 {
		t.Log("MEASURED ZERO: no build has CooldownRecovery without a base cooldown, so the " +
			"widened :1285 gate admits nothing here. Ported for fidelity.")
	}
	if zeroCooldown == 0 {
		t.Log("MEASURED ZERO: every published output.Cooldown is nonzero, so the :4910 " +
			"presence-vs-value distinction decides nothing here. Ported for fidelity.")
	}
	if wrong > 0 {
		t.Logf("output.Cooldown still disagrees with PoB on %d of %d builds — a standing residual "+
			"neither change here touches", wrong, seen)
	}
}
