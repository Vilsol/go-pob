package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

// TestCalcsAreaBreakdownsMatchPoB: breakdown.area (CalcBreakdown.lua:95-109) against the pin's CALCS pass over every 40th corpus build.
func TestCalcsAreaBreakdownsMatchPoB(t *testing.T) {
	t.Parallel()
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_area_breakdowns_expected.jsonl.gz")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n, same, seen := 0, 0, 0
	for sc.Scan() {
		var want struct {
			Build string
			Area  json.RawMessage
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		area := map[string][]string{}
		if string(want.Area) != "[]" {
			if err := json.Unmarshal(want.Area, &area); err != nil {
				t.Fatal(err)
			}
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		for key, w := range area {
			seen++
			var got []string
			if e := env.Player.Breakdown.Data[key]; e != nil {
				got = append(slices.Clone(e.Lines), fmt.Sprintf("radius=%s", luaNumber(e.Radius)))
			}
			if slices.Equal(got, w) {
				same++
			} else if testing.Verbose() {
				t.Logf("%s %s:\n ours %q\n PoB  %q", want.Build, key, got, w)
			}
		}
	}
	if n < 300 || seen < 250 {
		t.Fatalf("REFUSING: only %d builds / %d breakdowns in the fixture", n, seen)
	}
	t.Logf("%d of %d radius breakdowns match the pin", same, seen)
	if same < calcsAreaBreakdownsFloor {
		t.Errorf("only %d match; the floor is %d", same, calcsAreaBreakdownsFloor)
	}
}

const calcsAreaBreakdownsFloor = 286
