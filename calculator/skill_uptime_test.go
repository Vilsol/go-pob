package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSkillUptime pins CalcOffence.lua:1248-1279, the "Skill uptime" block, which go-pob never ported.
func TestSkillUptime(t *testing.T) {
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
		why   string
	}{
		{"1595-trickster-frost-bomb-2.xml", "plain arm: 2.013 / 2.508 = 80.26%"},
		{"0192-inquisitor-molten-shell.xml", "NoCooldownRecoveryInDuration: 3.63 / (3.663 + 3.63) = 49.77%"},
		{"0002-slayer-flicker-strike.xml", "capped at 100%"},
	}

	for _, c := range cases {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", c.build)
		}
		want, ok := rec.Output["DurationUptime"]
		if !ok {
			t.Fatalf("REFUSING: PoB emits no DurationUptime for %s — wrong fixture (%s)", c.build, c.why)
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

		got, ok := env.Player.Output["DurationUptime"]
		switch {
		case !ok:
			t.Errorf("%s: we never write DurationUptime; PoB has %g (%s)", c.build, want, c.why)
		case !oracleClose(got, want, oracleDumpTolerance):
			t.Errorf("%s: DurationUptime = %g, PoB %g (%s)", c.build, got, want, c.why)
		}
	}
}
