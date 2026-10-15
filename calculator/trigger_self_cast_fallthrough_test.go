package calculator

import (
	"math"
	"os"
	"testing"
)

func gradeTriggeredBuilds(t *testing.T, builds []string, keys []string) {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		for _, key := range keys {
			if want, ok := rec.Output[key]; !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero %s for %s", key, build)
			}
		}
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for _, key := range keys {
				want := rec.Output[key]
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v (ratio %.5f)", build, key, got, want, got/want)
				}
			}
		})
	}
}

// TestTriggerWithNoActionCooldownStillPublishesARate pins CalcTriggers.lua:583-586.
//
//pob:block:src/Modules/CalcTriggers.lua:588-591
func TestTriggerWithNoActionCooldownStillPublishesARate(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{"0851-assassin-flicker-strike-of-power.xml"},
		[]string{"EffectiveSourceRate", "SkillTriggerRate", "Speed", "TriggerTime"})
}

// TestUniqueItemTriggersFindTheirSource pins CalcTriggers.lua:898-904 and :997-999.
//
//pob:src/Modules/CalcTriggers.lua:903-909,src/Modules/CalcTriggers.lua:1002-1004
func TestUniqueItemTriggersFindTheirSource(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{"0552-occultist-raise-spiders.xml"},
		[]string{"EffectiveSourceRate", "SkillTriggerRate", "Speed", "TriggerTime"})
	gradeTriggeredBuilds(t,
		[]string{"1460-necromancer-flesh-offering.xml"},
		[]string{"EffectiveSourceRate"})
}

// TestBattlemagesCryCapsTheSourceRateByExerts pins CalcTriggers.lua:483-500 — the exerted-hits cap on a Battlemage's Cry source.
//
//pob:block:src/Modules/CalcTriggers.lua:488-505
func TestBattlemagesCryCapsTheSourceRateByExerts(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{
			"10003-ladder-Fnixi_2280.MiehekasMeriMunaMies.xml",
			"10577-ladder-Killaaaa_1225.Uber_Killaaa.xml",
			"12906-ladder-adeukis_6598.allflamebleedjugnut.xml",
			"13289-ladder-d3wolf_4682.PsyUpperManagement.xml",
			"14484-ladder-qualitye_5001.обожеопять.xml",
		},
		[]string{"EffectiveSourceRate", "SkillTriggerRate", "Speed", "TriggerTime"})
}

// TestKineticFluxAndCallThePyreAreTriggered pins CalcTriggers.lua:1147-1150 and :1370-1377.
//
//pob:src/Modules/CalcTriggers.lua:1152-1155,src/Modules/CalcTriggers.lua:1375-1382
func TestKineticFluxAndCallThePyreAreTriggered(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{
			"3402-ladder-apple2062_5053.aawess.xml",
			"0320-chieftain-shield-charge.xml",
			"1552-luminary-shield-charge-5.xml",
		},
		[]string{"SkillTriggerRate", "Speed"})
}

// TestFocusTriggerUsesTheFocusCycle pins CalcTriggers.lua:135-217 — ["focus"]'s customHandler.
//
//pob:src/Modules/CalcTriggers.lua:136-220
func TestFocusTriggerUsesTheFocusCycle(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml"},
		[]string{"TriggerRateCap", "SkillTriggerRate", "Speed", "TriggerTime"})
}

// TestTriggerCraftAndShroudOfTheLightless pins CalcTriggers.lua:1058-1080 and :943-946.
//
//pob:src/Modules/CalcTriggers.lua:948-951,src/Modules/CalcTriggers.lua:1063-1085
func TestTriggerCraftAndShroudOfTheLightless(t *testing.T) {
	gradeTriggeredBuilds(t,
		[]string{
			"12768-ladder-Yildiraykanka_0432.Ejerham.xml",
			"13188-ladder-budgi_7807.SecondAnimendment.xml",
			"14836-ladder-supportTTK_3722.AightTheSecondTry.xml",
			"3047-ladder-Jimbowoe_2431.AllFlameMinionCuck.xml",
			"4173-ladder-BayUlgen_7553.LeblebiKafaliOE.xml",
		},
		[]string{"TriggerRateCap", "SkillTriggerRate", "Speed", "TriggerTime"})
}
