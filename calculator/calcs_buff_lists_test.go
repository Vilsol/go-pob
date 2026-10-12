package calculator

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

const calcsBuffListMismatchCeiling = 5

// TestCalcsBuffListsMatchPoB grades the Calcs tab's buff / combat / curse lines against the CALCS pass of the pinned PoB over every 20th corpus build.
func TestCalcsBuffListsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_buff_lists_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type lists struct{ BuffList, CombatList, CurseList string }
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n, bad := 0, 0
	for sc.Scan() {
		var want struct {
			Build string
			lists
			Minion *lists
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		got := env.Player.OutputStrings
		for k, w := range map[string]string{"BuffList": want.BuffList, "CombatList": want.CombatList, "CurseList": want.CurseList} {
			if got[k] != w {
				bad++
				t.Logf("%s %s:\n got %q\nwant %q", want.Build, k, got[k], w)
			}
		}
		if want.Minion != nil && env.Minion != nil && env.Minion.Actor != nil {
			mg := env.Minion.Actor.OutputStrings
			for k, w := range map[string]string{"BuffList": want.Minion.BuffList, "CombatList": want.Minion.CombatList} {
				if mg[k] != w {
					bad++
					t.Logf("%s minion %s:\n got %q\nwant %q", want.Build, k, mg[k], w)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no oracle rows")
	}
	t.Logf("%d builds, %d lines differ", n, bad)
	if bad > calcsBuffListMismatchCeiling {
		t.Errorf("%d lines differ from PoB, ceiling %d", bad, calcsBuffListMismatchCeiling)
	}
	if bad < calcsBuffListMismatchCeiling {
		t.Errorf("%d lines differ, below the ceiling %d: lower it", bad, calcsBuffListMismatchCeiling)
	}
}
