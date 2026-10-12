package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const gemSupportPairsMismatchCeiling = 10

// TestGemSupportPairsMatchPoB grades the Skills page's supporting highlight.
func TestGemSupportPairsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/gem_support_pairs_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	builds, groups, nonEmpty, bad := 0, 0, 0, 0
	for sc.Scan() {
		var want struct {
			Build  string
			Groups [][][2]int
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		builds++
		env := BuildOutputWithCache(&Calculator{PoB: loadCorpusBuild(t, want.Build)}, OutputModeMain, NewEnvironmentCache())
		for gi, w := range want.Groups {
			groups++
			if len(w) > 0 {
				nonEmpty++
			}
			got := make([][2]int, 0)
			for _, p := range env.GemSupportPairs(gi) {
				got = append(got, [2]int{p.Support, p.Active})
			}
			if fmt.Sprint(got) != fmt.Sprint(w) && !(len(got) == 0 && len(w) == 0) {
				bad++
				t.Logf("%s group %d: got %v, PoB %v", want.Build, gi, got, w)
			}
		}
	}
	if nonEmpty == 0 {
		t.Fatal("no oracle group has a pair")
	}
	t.Logf("%d builds, %d groups (%d with pairs), %d differ", builds, groups, nonEmpty, bad)
	if bad != gemSupportPairsMismatchCeiling {
		t.Errorf("%d groups differ from PoB, ceiling %d", bad, gemSupportPairsMismatchCeiling)
	}
}
