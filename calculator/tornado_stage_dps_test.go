package calculator

import (
	"math"
	"os"
	"testing"
)

var tornadoStageBuilds = []string{
	"0222-inquisitor-tornado-of-elemental-turbulence.xml",
	"0511-elementalist-tornado-of-elemental-turbulence.xml",
	"0823-inquisitor-tornado-of-elemental-turbulence-2.xml",
	"1446-elementalist-tornado-of-elemental-turbulence-2.xml",
}

func TestTornadoOfElementalTurbulenceScalesDPSByStages(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawMultiStage := false

	for _, build := range tornadoStageBuilds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, present := rec.Output["TotalDPS"]
		if !present || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero TotalDPS for %s; the case cannot "+
				"discriminate a dropped multiplier from a correct one", build)
		}

		env := corpusEnvForLoadedData(t, build)
		ms := env.Player.MainSkill
		if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil {
			t.Fatalf("REFUSING: %s has no main skill", build)
		}
		if id := ms.ActiveEffect.GrantedEffect.ID(); id != "TornadoAltY" {
			t.Fatalf("REFUSING: %s runs main skill %q, not TornadoAltY; the fixture no longer "+
				"exercises the arm it was chosen for", build, id)
		}
		if ms.SkillData.DpsMultiplier > 1 {
			sawMultiStage = true
		}

		got := env.Player.Output["TotalDPS"]
		if rel := math.Abs(got-want) / math.Abs(want); rel > oracleDumpTolerance {
			t.Errorf("%s: TotalDPS = %v, PoB %v (rel %.4f); ours/PoB = %.4f — the stage "+
				"dpsMultiplier at act_dex.lua:18126 is the factor",
				build, got, want, rel, got/want)
		}
	}

	if !sawMultiStage {
		t.Fatal("REFUSING: every fixture ran at a stage dpsMultiplier of 1, so dropping the " +
			"multiplier entirely would still pass")
	}
}
