package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestGearArmourUnderIronReflexes pins CalcDefence.lua:914-924.
//
//	if evasionBase > 0 then
//	    if ironReflexes then armour = armour + evasionBase * mod(...)
//	end
func TestGearArmourUnderIronReflexes(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "1499-slayer-sunder-of-earthbreaking.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	if rec.Output["Gear:Armour"] == 0 {
		t.Fatalf("REFUSING: %s publishes no Gear:Armour; the fixture has moved", name)
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range []string{"Gear:Armour", "Gear:Evasion"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want)+0.5 {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}
