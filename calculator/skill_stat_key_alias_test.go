package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestPrecisionAuraCritReachesThePlayer pins the Precision crit stat-key skew (act_dex.lua:11584) end to end; go-pob-data
// now emits the key PoB reads, so no alias bridges it.
func TestPrecisionAuraCritReachesThePlayer(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	const name = "15350-juggernaut-sunder.xml"

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}

	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env == nil {
		t.Fatal("REFUSING: the build did not calculate")
	}

	for _, key := range []string{"PreEffectiveCritChance", "CritChance"} {
		want, ok := rec.Output[key]
		if !ok {
			t.Fatalf("REFUSING: the blob has no %s for %s", key, name)
		}
		got := env.Player.Output[key]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s: got %v, PoB %v", key, got, want)
		}
	}
}
