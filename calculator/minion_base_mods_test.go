package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestMinionBaseModsMatch3_20 pins the minion base-mod block at CalcPerform.lua:1186-1202 through the values it produces, one build per mod it touches.
func TestMinionBaseModsMatch3_20(t *testing.T) {
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
		why   string
	}{
		{
			build: "1163-occultist-raise-spiders.xml",
			keys:  []string{"PoisonDPS", "PoisonDamage", "TotalPoisonDPS"},
			why:   "Damage MORE -50 Poison, deleted upstream",
		},
		{
			build: "1650-necromancer-summon-raging-spirit-29.xml",
			keys:  []string{"IgniteDPS", "IgniteDamage"},
			why:   "Damage MORE -50 Ignite, deleted upstream",
		},
	}

	for _, c := range cases {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", c.build)
		}
		if len(rec.Minion) == 0 {
			t.Fatalf("REFUSING: %s carries no minionOutput — wrong fixture for %s", c.build, c.why)
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
		if env.Minion == nil {
			t.Fatalf("REFUSING: we built no minion for %s", c.build)
		}

		for _, key := range c.keys {
			want, ok := rec.Minion[key]
			if !ok {
				t.Fatalf("REFUSING: PoB emits no Minion.%s for %s — the fixture cannot see %s",
					key, c.build, c.why)
			}
			got := env.Minion.Output[key]
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("%s Minion.%s = %g, PoB %g (ratio %.4f) — %s",
					c.build, key, got, want, got/want, c.why)
			}
		}
	}
}
