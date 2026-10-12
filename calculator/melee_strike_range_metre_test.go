package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMeleeStrikeRangeMetre covers ModParser.lua:675-676 and CalcOffence.lua:1125/:1128 — the METRE form of melee strike range.
func TestMeleeStrikeRangeMetre(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, name := range []string{
		"0087-juggernaut-ground-slam.xml",
		"0018-berserker-heavy-strike.xml",
	} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			want := rec.Output["WeaponRange"]
			if want == 0 {
				t.Fatalf("REFUSING: %s publishes no WeaponRange; the fixture has moved", name)
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
			got := env.Player.Output["WeaponRange"]
			if math.Abs(got-want) > 1e-9 {
				t.Errorf("%s: WeaponRange = %v, want %v (PoB)", name, got, want)
			}
		})
	}
}
