package calculator

import (
	"os"
	"testing"
)

// TestSplitCountBlock pins CalcOffence.lua:1069-1086, which had no counterpart here at all.
func TestSplitCountBlock(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	for _, name := range []string{
		"0017-deadeye-spectral-throw-of-materialising",
		"0344-assassin-icicle-mine-of-fanning",
	} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name+".xml"]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			want := rec.Output["SplitCount"]
			if want == 0 {
				t.Fatalf("fixture is useless: PoB reports SplitCount 0, so a 0-vs-0 agreement "+
					"would pass against the unported tree (%s)", name)
			}
			env := corpusEnvForLoadedData(t, name)
			for _, k := range []string{"SplitCount", "SplitCountString"} {
				if got := env.Player.Output[k]; got != rec.Output[k] {
					t.Errorf("%s: %s = %v, want %v", name, k, got, rec.Output[k])
				}
			}
		})
	}

	t.Run("a CannotSplit skill publishes no count at all", func(t *testing.T) {
		const name = "0263-assassin-blazing-salvo"
		rec, ok := oracle[name+".xml"]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		if _, published := rec.Output["SplitCount"]; published {
			t.Fatalf("control is useless: PoB now publishes SplitCount for %s, so it no longer "+
				"exercises the CannotSplit arm", name)
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Player.Output["ProjectileCount"] == 0 {
			t.Fatal("control is useless: the build fires no projectiles, so the guard is not " +
				"what is suppressing the count")
		}
		if _, ours := env.Player.Output["SplitCount"]; ours {
			t.Errorf("we published SplitCount = %v on a CannotSplit skill; upstream's :1069 arm "+
				"writes nothing to that key", env.Player.Output["SplitCount"])
		}
	})
}
