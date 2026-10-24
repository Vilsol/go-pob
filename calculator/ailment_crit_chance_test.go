package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestAilmentCritChanceIsStackWeighted pins CalcOffence.lua:3364 and :3823.
func TestAilmentCritChanceIsStackWeighted(t *testing.T) {
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
		build  string
		minion bool
		keys   []string
	}{
		{"0653-scion-none-mirror-arrow-of-bombarding-clones.xml", true, []string{"IgniteChance"}},
		{"15671-gladiator-dual-strike-of-ambidexterity.xml", false, []string{"BleedChance"}},
	}

	for _, c := range cases {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", c.build)
		}
		want := rec.Output
		if c.minion {
			want = rec.Minion
		}
		if len(want) == 0 {
			t.Fatalf("REFUSING: %s carries no output table for this actor", c.build)
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.build))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

		got := env.Player.Output
		if c.minion {
			if env.Minion == nil {
				t.Fatalf("REFUSING: we built no minion for %s", c.build)
			}
			got = env.Minion.Output
		}

		for _, key := range c.keys {
			w, ok := want[key]
			if !ok {
				t.Fatalf("REFUSING: PoB emits no %s for %s — wrong fixture", key, c.build)
			}
			if !oracleClose(got[key], w, oracleDumpTolerance) {
				t.Errorf("%s %s = %g, PoB %g", c.build, key, got[key], w)
			}
		}
	}
}
