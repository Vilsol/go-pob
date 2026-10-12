package calculator

import (
	"os"
	"testing"
)

// TestPerSecondCostUsesOutputRepeats: The link-skill cost-per-second block at CalcOffence.lua:5792-5833 divides its use speed by the skill's REPEAT COUNT.
func TestPerSecondCostUsesOutputRepeats(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		build string
		stat  string
		why   string
	}{
		{"0046-elementalist-arc", "ManaPerSecondCost", "was 3.0000x — ours read Repeats 1, PoB 3"},
		{"0071-slayer-flicker-strike", "ManaPerSecondCost", "was 3.0000x"},
		{"0085-occultist-voltaxic-burst-3", "ManaPerSecondCost", "was 2.0000x"},
		{"0052-juggernaut-cleave", "ManaPerSecondCost", "was 3.0000x"},
		{"0072-occultist-voltaxic-burst", "ManaPerSecondCost", "was 2.0000x"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			t.Parallel()

			rec, ok := oracle[tc.build+".xml"]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob — an absent build cannot "+
					"fail this test, so it must not pass it either", tc.build)
			}
			want, has := rec.Output[tc.stat]
			if !has || want == 0 {
				t.Fatalf("REFUSING: the oracle carries no nonzero %s for %s, so this case "+
					"asserts nothing (%s)", tc.stat, tc.build, tc.why)
			}

			env := corpusEnvForLoadedData(t, tc.build)
			if env == nil || env.Player == nil {
				t.Fatalf("REFUSING: %s produced no player environment", tc.build)
			}
			got := env.Player.Output[tc.stat]
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("%s = %g, want %g (ratio %.4f) — %s",
					tc.stat, got, want, got/want, tc.why)
			}
		})
	}
}

// TestPerSecondCostFamilyAgreementFloor is the sweep behind the five named builds above.
func TestPerSecondCostFamilyAgreementFloor(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	stats := []string{"ManaPerSecondCost", "LifePerSecondCost", "ESPerSecondCost"}
	graded, agree := 0, 0
	for _, b := range names {
		rec, ok := oracle[b]
		if !ok {
			continue
		}
		var env *Environment
		for _, s := range stats {
			want, has := rec.Output[s]
			if !has || want == 0 {
				continue
			}
			if env == nil {
				if env = corpusEnvForLoadedData(t, b); env == nil || env.Player == nil {
					break
				}
			}
			graded++
			if oracleClose(env.Player.Output[s], want, oracleDumpTolerance) {
				agree++
			}
		}
	}

	if graded == 0 {
		t.Fatal("REFUSING: no build carries a nonzero <resource>PerSecondCost in the oracle, so " +
			"this sweep is not reaching the link-skill cost block at all")
	}
	if agree == 0 {
		t.Fatal("REFUSING: the family agrees on NO build — that reads as the keys having stopped " +
			"being published rather than as a worse port")
	}

	t.Logf("<resource>PerSecondCost: %d graded, %d agree (%.1f%%)",
		graded, agree, 100*float64(agree)/float64(graded))

	const floor = 5608
	if agree < floor {
		t.Errorf("PerSecondCost family agreement fell to %d of %d, floor %d — the use speed at "+
			"calc_offence.go's link-skill cost block is shared by all three resources, so a "+
			"regression here moves the whole family at once", agree, graded, floor)
	}
}
