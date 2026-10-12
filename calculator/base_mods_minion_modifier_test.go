package calculator

import (
	"os"
	"testing"
)

// TestBaseModsMinionModifierRows covers the skill-base-mods emitter's MinionModifier arm.
func TestBaseModsMinionModifierRows(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	const name = "1462-necromancer-summon-skeletons-of-mages.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	env := corpusEnvForLoadedData(t, name)
	if env.Minion == nil {
		t.Fatalf("REFUSING: %s builds no minion", name)
	}
	if !env.Minion.ModDB.Flag(nil, "Condition:Adrenaline") {
		t.Errorf("%s: minion carries no Condition:Adrenaline (Fresh Meat baseMods)", name)
	}
	for _, key := range []string{"MovementSpeedMod", "CombinedDPS"} {
		want := rec.Minion[key]
		if want == 0 {
			t.Fatalf("REFUSING: PoB reports 0 for Minion.%s", key)
		}
		if got := env.Minion.Output[key]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s Minion.%s = %g, PoB %g", name, key, got, want)
		}
	}
}
