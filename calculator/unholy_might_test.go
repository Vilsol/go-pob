package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUnholyMightConvertsRatherThanGains pins CalcPerform.lua:720-724.
//
//	if modDB:Flag(nil, "UnholyMight") then
//		local effect = 1 + modDB:Sum("INC", nil, "BuffEffectOnSelf") / 100
//		modDB:NewMod("PhysicalDamageConvertToChaos", "BASE", m_floor(100 * effect), "Unholy Might")
//		modDB:NewMod("Condition:CanWither", "FLAG", true, "Unholy Might")
//	end
func TestUnholyMightConvertsRatherThanGains(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0008-necromancer-vaal-summon-skeletons.xml")
	db := env.Player.ModDB
	if !db.Flag(nil, "UnholyMight") {
		t.Fatal("fixture is useless: this build no longer has UnholyMight, so the branch never runs")
	}

	want := math.Floor(100 * (1 + db.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
	if got := db.Sum(mod.TypeBase, nil, "PhysicalDamageConvertToChaos"); got != want {
		t.Errorf("PhysicalDamageConvertToChaos = %v, want %v — CalcPerform.lua:722 CONVERTS 100%%, "+
			"scaled by BuffEffectOnSelf", got, want)
	}
	if got := db.Sum(mod.TypeBase, nil, "PhysicalDamageGainAsChaos"); got != 0 {
		t.Errorf("PhysicalDamageGainAsChaos = %v, want 0 — Unholy Might no longer grants gain-as; "+
			"leaving it in place double-counts against the conversion", got)
	}
	if !db.Flag(nil, "Condition:CanWither") {
		t.Error("Condition:CanWither is not set — CalcPerform.lua:723 grants it with the buff " +
			"(it has no consumer in this tree yet, and is emitted so it is there when one lands)")
	}
}
