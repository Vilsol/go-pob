package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestBlockIsComputedBeforePrimaryDefences pins the ORDER of CalcDefence.lua:660-771 against :812-818 and :826.
func TestBlockIsComputedBeforePrimaryDefences(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const name = "0245-occultist-winter-orb-3.xml"

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	carriers := len(env.Player.ModDB.Mods["EnergyShieldIncreasedByChanceToBlockSpellDamage"])
	if carriers == 0 {
		t.Fatalf("REFUSING: %s no longer carries EnergyShieldIncreasedByChanceToBlockSpellDamage, so it cannot "+
			"exercise the Block-before-defences ordering", name)
	}

	want := rec.Output["EnergyShield"]
	if want == 0 {
		t.Fatal("REFUSING: PoB reports 0 EnergyShield here; a 0-vs-0 agreement proves nothing")
	}
	if got := env.Player.Output["EnergyShield"]; math.Abs(got-want) > 0.005*math.Abs(want)+1e-9 {
		t.Errorf("%s: EnergyShield = %v, want %v (%d EnergyShieldIncreasedByChanceToBlockSpellDamage carrier(s))",
			name, got, want, carriers)
	}
}
