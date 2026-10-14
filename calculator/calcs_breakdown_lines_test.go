package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
)

const (
	breakdownLinesDifferingKeysCeiling  = 129
	breakdownLinesDifferingPairsCeiling = 490
)

// TestCalcsBreakdownLinesMatchPoB: The lines of every breakdown key.
func TestCalcsBreakdownLinesMatchPoB(t *testing.T) {
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_breakdown_lines_expected.jsonl.gz")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 1<<24), 1<<24)

	type tally struct{ doubled, missing, extra, wrong int }
	census := map[string]*tally{}
	n, pobLined := 0, 0
	for sc.Scan() {
		var want struct {
			Build  string
			Player json.RawMessage
			Minion json.RawMessage
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		for _, actor := range []struct {
			name string
			raw  json.RawMessage
			bd   *Breakdown
		}{{"player", want.Player, env.Player.Breakdown}, {"minion", want.Minion, minionBreakdown(env)}} {
			pob := map[string][]string{}
			if string(actor.raw) != "[]" {
				if err := json.Unmarshal(actor.raw, &pob); err != nil {
					t.Fatal(err)
				}
			}
			ours := map[string][]string{}
			if actor.bd != nil {
				for k, e := range actor.bd.GetData() {
					if e != nil && len(e.Lines) > 0 {
						ours[k] = e.Lines
					}
				}
			}
			keys := map[string]bool{}
			for k := range pob {
				keys[k] = true
				pobLined++
			}
			for k := range ours {
				keys[k] = true
			}
			for k := range keys {
				w, g := pob[k], ours[k]
				if slices.Equal(g, w) {
					continue
				}
				id := actor.name + " " + k
				if census[id] == nil {
					census[id] = &tally{}
				}
				switch {
				case len(g) > 0 && len(g)%2 == 0 && slices.Equal(g[:len(g)/2], g[len(g)/2:]):
					census[id].doubled++
				case len(g) == 0:
					census[id].missing++
				case len(w) == 0:
					census[id].extra++
				default:
					census[id].wrong++
				}
				if bk := os.Getenv("BREAKDOWN_KEY"); bk == id || bk == "*" {
					t.Logf("%s %s:\n ours %q\n PoB  %q", want.Build, id, g, w)
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n < 300 || pobLined < 5000 {
		t.Fatalf("REFUSING: only %d builds / %d PoB breakdowns with lines in the fixture", n, pobLined)
	}
	ids := make([]string, 0, len(census))
	builds := 0
	for id, c := range census {
		ids = append(ids, id)
		builds += c.doubled + c.missing + c.extra + c.wrong
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := census[ids[i]], census[ids[j]]
		sa, sb := a.doubled+a.missing+a.extra+a.wrong, b.doubled+b.missing+b.extra+b.wrong
		if sa != sb {
			return sa > sb
		}
		return ids[i] < ids[j]
	})
	var sb strings.Builder
	fmt.Fprintf(&sb, "%-40s %7s %7s %7s %7s\n", "key", "doubled", "missing", "extra", "wrong")
	for _, id := range ids {
		c := census[id]
		fmt.Fprintf(&sb, "%-40s %7d %7d %7d %7d\n", id, c.doubled, c.missing, c.extra, c.wrong)
	}
	t.Logf("%d builds, %d PoB breakdowns with lines; %d keys differ on %d (key, build) pairs\n%s",
		n, pobLined, len(census), builds, sb.String())
	if len(census) > breakdownLinesDifferingKeysCeiling {
		t.Errorf("%d breakdown keys differ from the pin, ceiling %d", len(census), breakdownLinesDifferingKeysCeiling)
	}
	if builds > breakdownLinesDifferingPairsCeiling {
		t.Errorf("%d (key, build) breakdown pairs differ from the pin, ceiling %d", builds, breakdownLinesDifferingPairsCeiling)
	}
}

func minionBreakdown(env *Environment) *Breakdown {
	if env.Minion == nil {
		return nil
	}
	return env.Minion.Breakdown
}
