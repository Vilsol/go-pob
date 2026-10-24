package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestTrapAreaOverlap pins the two area-overlap trap funcs and the config chain they hang off.
func TestTrapAreaOverlap(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.29" {
		t.Skipf("SKIP: this test pins 3.29 values but the process loaded %s game data", raw.LatestVersion)
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	cases := []struct {
		build string
		keys  []string
	}{
		{"7312-ladder-happy_1234_5190.", []string{"PulsesPerTrap", "AverageActiveTraps", "OverlapChance"}},
		{"1634-pathfinder-seismic-trap-of-swells.xml", []string{"OverlapChance"}},
	}

	for _, c := range cases {
		build := c.build
		if !strings.HasSuffix(build, ".xml") {
			matches, _ := filepath.Glob(filepath.Join(corpusDirForLoadedData(t), build+"*.xml"))
			if len(matches) != 1 {
				t.Fatalf("REFUSING: %d corpus builds start with %s", len(matches), build)
			}
			build = filepath.Base(matches[0])
		}
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", build)
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

		for _, key := range c.keys {
			want, ok := rec.Output[key]
			if !ok {
				t.Fatalf("REFUSING: PoB emits no %s for %s — wrong fixture", key, build)
			}
			got, ok := env.Player.Output[key]
			switch {
			case !ok:
				t.Errorf("%s: we never write %s; PoB has %g", build, key, want)
			case !oracleClose(got, want, oracleDumpTolerance):
				t.Errorf("%s: %s = %g, PoB %g", build, key, got, want)
			}
		}
	}
}
