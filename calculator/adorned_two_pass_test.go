package calculator

import (
	"math"
	"os"
	"testing"
)

// TestAdornedScalesJewelsInEarlierSockets pins the fact that upstream publishes The Adorned's multiplier in a DIFFERENT LOOP from the one that reads it.
func TestAdornedScalesJewelsInEarlierSockets(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	const build = "2092-ladder-Atechika_2805.Lumichika.xml"
	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
	}
	want, ok := rec.Output["Life"]
	if !ok || want == 0 {
		t.Fatalf("REFUSING: PoB reports no nonzero Life for %s", build)
	}

	env := corpusEnvForLoadedData(t, build)
	got := env.Player.Output["Life"]
	if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
		t.Errorf("%s: Life = %v, PoB reports %v (ratio %.4f) — The Adorned's multiplier is not "+
			"reaching the corrupted jewels socketed at lower node ids", build, got, want, got/want)
	}
}
