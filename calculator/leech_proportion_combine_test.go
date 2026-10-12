package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestLeechInstantProportionIsCombined pins CalcOffence.lua:3881, :3886 and :3892.
func TestLeechInstantProportionIsCombined(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	fixedByTheCombine := map[string]bool{
		"0068-reliquarian-poisonous-concoction-of-bouncing.xml": true,
	}
	checked, wrong := 0, 0
	for _, name := range names {
		if !fixedByTheCombine[name] {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, ok := rec.Output["ManaLeechInstantProportion"]
		if !ok || want == 0 {
			continue
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			continue
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		}()
		if env == nil {
			continue
		}
		checked++
		if got := env.Player.Output["ManaLeechInstantProportion"]; math.Abs(got-want) > 0.005*math.Abs(want) {
			wrong++
			if wrong <= 3 {
				t.Errorf("%s: ManaLeechInstantProportion = %v, want %v", name, got, want)
			}
		}
	}
	if checked != len(fixedByTheCombine) {
		t.Fatalf("REFUSING: checked %d of %d pinned builds; a fixture has moved and this test "+
			"would pass vacuously", checked, len(fixedByTheCombine))
	}
	if wrong > 0 {
		t.Errorf("ManaLeechInstantProportion wrong on %d of %d builds", wrong, checked)
	}
}
