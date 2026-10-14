package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// TestCalcsAttributeBreakdownsMatchPoB: breakdown.simple for Str, Dex.
func TestCalcsAttributeBreakdownsMatchPoB(t *testing.T) {
	t.Parallel()
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_attribute_breakdowns_expected.jsonl.gz")
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
	n, lined, wrong := 0, 0, 0
	for sc.Scan() {
		var want struct {
			Build      string
			Attributes json.RawMessage
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		attrs := map[string][]string{}
		if string(want.Attributes) != "[]" {
			if err := json.Unmarshal(want.Attributes, &attrs); err != nil {
				t.Fatal(err)
			}
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		for _, key := range []string{"Str", "Dex", "Int", "Omni"} {
			w := attrs[key]
			if len(w) > 0 {
				lined++
			}
			var got []string
			if e := env.Player.Breakdown.Data[key]; e != nil {
				got = e.Lines
			}
			if !slices.Equal(got, w) {
				wrong++
				t.Errorf("%s %s:\n ours %q\n PoB  %q", want.Build, key, got, w)
			}
		}
	}
	if n < 300 || lined < 250 {
		t.Fatalf("REFUSING: only %d builds / %d breakdowns with lines in the fixture", n, lined)
	}
	t.Logf("%d builds, %d breakdowns with lines, %d differ", n, lined, wrong)
}
