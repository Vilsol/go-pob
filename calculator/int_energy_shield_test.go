package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestIntelligenceGrantsEnergyShieldPerTenInt: Intelligence grants INCREASED energy shield at one point per TEN intelligence.
//
//	modDB:NewMod("EnergyShield", "INC", m_floor(output.Int / 10), "Intelligence")
func TestIntelligenceGrantsEnergyShieldPerTenInt(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0009-trickster-cobra-lash.xml"
	env := corpusEnvForLoadedData(t, build)

	intel := env.Player.Output["Int"]
	if intel < 10 {
		t.Fatalf("premise broken: %s has Int %v; below 10 both formulas floor to 0 and the "+
			"case is vacuous", build, intel)
	}
	want := math.Floor(intel / 10)
	if want == math.Floor(intel/5) {
		t.Fatalf("premise broken: Int %v floors identically under /10 and /5; case is vacuous", intel)
	}

	found := 0
	var got float64
	for _, m := range env.Player.ModDB.Mods["EnergyShield"] {
		if m.Type() != mod.TypeIncrease || m.GetSource() != "Intelligence" {
			continue
		}
		found++
		got = m.Value().Float()
	}
	if found != 1 {
		t.Fatalf("expected exactly one Intelligence-sourced EnergyShield INC mod, found %d "+
			"(if 0, the producer at calc_perform.go is gone and this test cannot see the defect)", found)
	}
	if got != want {
		t.Errorf("Intelligence EnergyShield INC = %v, want %v (floor(Int %v / 10)).\n"+
			"  floor(Int/5) would give %v -- the value this ported before the fix",
			got, want, intel, math.Floor(intel/5))
	}
}
