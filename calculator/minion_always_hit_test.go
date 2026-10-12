package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMinionAttacksAlwaysHit pins CalcPerform.lua:1141-1145.
func TestMinionAttacksAlwaysHit(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dir, _ := corpusNamesForLoadedData(t)

	for _, name := range []string{
		"0978-luminary-animate-guardian-of-smiting-3.xml",
		"1460-necromancer-flesh-offering.xml",
		"2110-ladder-ChechW_6303.Chechinary.xml",
	} {
		file, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Minion == nil {
			t.Fatalf("REFUSING: %s has no minion actor; this test would pass vacuously", name)
		}
		if !env.Minion.ModDB.Flag(nil, "CannotBeEvaded") {
			t.Errorf("%s: minion lacks CannotBeEvaded", name)
		}
		if got := env.Minion.OutputTable[OutTableMainHand]["HitChance"]; got != 100 {
			t.Errorf("%s: minion MainHand HitChance = %v, want 100 (PoB's minions always hit)", name, got)
		}
	}
}
