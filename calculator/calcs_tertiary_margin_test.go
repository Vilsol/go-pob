package calculator

import (
	"bufio"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// TestCalcsTertiaryMarginBreakdownMatchesPoB grades CalcOffence.lua:446-471, explosive trap's random-radius breakdown, against oracle/calcs_tertiary_margin.lua.
func TestCalcsTertiaryMarginBreakdownMatchesPoB(t *testing.T) {
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_tertiary_margin_expected.jsonl")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		var want struct {
			Build string
			Lines []string
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		if len(want.Lines) == 0 {
			t.Fatalf("REFUSING: %s has no tertiary breakdown in the fixture", want.Build)
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		var got []string
		if e := env.Player.Breakdown.Data["AreaOfEffectRadiusTertiary"]; e != nil {
			got = e.Lines
		}
		if !slices.Equal(got, want.Lines) {
			t.Errorf("%s:\n ours %q\n PoB  %q", want.Build, got, want.Lines)
		}
	}
	if n < 2 {
		t.Fatalf("REFUSING: %d builds in the fixture", n)
	}
}
