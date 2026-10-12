package calculator

import (
	"math"
	"os"
	"testing"
)

// if env.configInput.repeatMode == "FINAL" or skillModList:Flag(nil, "OnlyFinalRepeat") then
// end
var finalRepeatBuilds = []string{
	"2528-ladder-Nijianguoguo_4308.\u795e\u79d8\u7ea2\u8863\u9a9a\u7537.xml",
	"6042-ladder-Seyron_0543.Vichtent_COTA.xml",
	"7347-ladder-hoguras_7830.csqiq.xml",
}

func TestOnlyFinalRepeatDividesDPSByRepeats(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range finalRepeatBuilds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, present := rec.Output["TotalDPS"]
		if !present || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero TotalDPS for %s", build)
		}
		if r := rec.Output["Repeats"]; r <= 1 {
			t.Fatalf("REFUSING: PoB reports Repeats=%v for %s; dividing by it is a no-op and the "+
				"fixture proves nothing", r, build)
		}

		env := corpusEnvForLoadedData(t, build)
		ms := env.Player.MainSkill
		if ms == nil || !ms.SkillModList.Flag(nil, "OnlyFinalRepeat") {
			t.Fatalf("REFUSING: %s no longer carries OnlyFinalRepeat on its main skill; the arm "+
				"this fixture exists for is not reached", build)
		}
		if got := env.Player.Output["Repeats"]; got != rec.Output["Repeats"] {
			t.Fatalf("REFUSING: Repeats disagree on %s (ours %v, PoB %v) — the divisor itself is "+
				"wrong, so a TotalDPS check here would be measuring something else",
				build, got, rec.Output["Repeats"])
		}

		got := env.Player.Output["TotalDPS"]
		if rel := math.Abs(got-want) / math.Abs(want); rel > oracleDumpTolerance {
			t.Errorf("%s: TotalDPS = %v, PoB %v (ours/PoB %.4f, Repeats %v) — "+
				"CalcOffence.lua:2427-2429 divides dpsMultiplier by Repeats",
				build, got, want, got/want, rec.Output["Repeats"])
		}
	}
}
