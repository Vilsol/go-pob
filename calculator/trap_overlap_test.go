package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestTrapAreaOverlap pins the two area-overlap trap funcs and the config chain they hang off.
func TestTrapAreaOverlap(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.20" {
		t.Skipf("SKIP: this test pins 3.20 values but the process loaded %s game data", raw.LatestVersion)
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	cases := []struct {
		build string
		keys  []string
	}{
		{"2181-saboteur-seismic-trap-82.xml", []string{"WavePulseRate", "PulsesPerTrap", "AverageActiveTraps", "OverlapChance"}},
		{"0783-saboteur-seismic-trap-27.xml", []string{"WavePulseRate", "PulsesPerTrap", "AverageActiveTraps", "OverlapChance"}},
	}

	for _, c := range cases {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", c.build)
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.build))
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
				t.Fatalf("REFUSING: PoB emits no %s for %s — wrong fixture", key, c.build)
			}
			got, ok := env.Player.Output[key]
			switch {
			case !ok:
				t.Errorf("%s: we never write %s; PoB has %g", c.build, key, want)
			case !oracleClose(got, want, oracleDumpTolerance):
				t.Errorf("%s: %s = %g, PoB %g", c.build, key, got, want)
			}
		}
	}
}
