package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestLeechInstanceCapAppliesBeforeTheInstantSplit pins CalcOffence.lua:3581-3591.
func TestLeechInstanceCapAppliesBeforeTheInstantSplit(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, name := range []string{"0149-warden-frost-blades-2.xml", "0093-ascendant-frostblink.xml"} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		if rec.MainHand["ManaLeechInstant"] == 0 {
			t.Fatalf("REFUSING: %s publishes no MainHand ManaLeechInstant; the fixture has moved", name)
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
		mh := env.Player.OutputTable[OutTableMainHand]
		for _, k := range []string{"ManaLeechInstant"} {
			want, got := rec.MainHand[k], mh[k]
			if math.Abs(got-want) > 0.005*math.Abs(want) {
				t.Errorf("%s: MainHand %s = %v, want %v", name, k, got, want)
			}
		}
	}
}
