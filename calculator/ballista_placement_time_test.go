package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestBallistaPlacementTime pins CalcSetup.lua:59.
func TestBallistaPlacementTime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, tc := range []struct {
		name    string
		build   string
		want    float64
		against string
	}{
		{"ballista", "2598-ladder-Thoughtness_4847.Allflamer_Fusi.xml", 0, "PoB"},
		{"plain-totem-control", "1037-hierophant-rejuvenation-totem.xml", 0, "PoB"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			want := tc.want
			if want == 0 {
				want = rec.Output["TotemPlacementTime"]
			}
			if want == 0 || rec.Output["TotemPlacementTime"] == 0 {
				t.Fatalf("REFUSING: %s publishes no TotemPlacementTime; the fixture has moved", tc.build)
			}
			file, err := os.ReadFile(filepath.Join(corpusDir, tc.build))
			if err != nil {
				t.Fatalf("%v", err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatalf("%v", err)
			}
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
			got := env.Player.Output["TotemPlacementTime"]
			if math.Abs(got-want) > math.Abs(want)*1e-9 {
				t.Errorf("%s: TotemPlacementTime = %v, want %v (%s)", tc.build, got, want, tc.against)
			}
		})
	}
}
