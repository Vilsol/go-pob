package calculator

import (
	"math"
	"os"
	"testing"
)

// TestElementalAegisReachesTheHitPool pins CalcDefence.lua:3135.
//
//	output[damageType.."TotalHitPool"] = output[damageType.."TotalHitPool"]
//		+ m_max(m_max(output[damageType.."Aegis"], output["sharedAegis"]),
//		        isElemental[damageType] and output[damageType.."AegisDisplay"] or 0)
func TestElementalAegisReachesTheHitPool(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0091-elementalist-winter-orb-2.xml"
	env := corpusEnvForLoadedData(t, build)
	out := env.Player.Output

	aegis := out["FireAegisDisplay"]
	if aegis <= 0 {
		t.Fatalf("premise broken: FireAegisDisplay = %v, so this build no longer carries an "+
			"elemental aegis and the assertion below would pass vacuously", aegis)
	}
	for _, k := range []string{"Ward", "FrostShieldLife", "sharedGuardAbsorbRate", "FireAegis", "sharedAegis"} {
		if out[k] != 0 {
			t.Fatalf("premise broken: %s = %v — this fixture was picked because it is 0, so the "+
				"Fire/Physical difference is exactly the elemental aegis", k, out[k])
		}
	}
	if out["FireMoMHitPool"] != out["PhysicalMoMHitPool"] {
		t.Fatalf("premise broken: Fire and Physical MoM hit pools differ (%v vs %v), so their "+
			"difference is no longer the aegis alone", out["FireMoMHitPool"], out["PhysicalMoMHitPool"])
	}

	if got, want := out["FireTotalHitPool"]-out["PhysicalTotalHitPool"], aegis; math.Abs(got-want) > 1e-6 {
		t.Errorf("FireTotalHitPool - PhysicalTotalHitPool = %v, want FireAegisDisplay %v "+
			"(PoB: 8626 - 6376 = 2250)", got, want)
	}

	for _, dead := range []string{"FiresharedAegis", "FiresharedElementalAegis"} {
		if _, ok := out[dead]; ok {
			t.Errorf("%s is now written by something — the old aegis line read this key and got "+
				"0 for free; if it has a producer, re-check which name CalcDefence.lua:3135 means", dead)
		}
	}
}
