package mcp

import (
	"fmt"
	"math"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestRankRowsShowTheFullDPSChange: ranked by MainDPS on a build with Full DPS skills, a row also carries dFullDPS,
// what applying it does to FullDPS.
func TestRankRowsShowTheFullDPSChange(t *testing.T) {
	s := testServer(t)
	h := fullDPSBuild(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": mainDPS, "limit": 3})
	list := rows(res)
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	seen := false
	for _, r := range list {
		want := dryDelta(t, s, "edit_tree", h, "FullDPS", map[string]any{"op": "allocate", "node": fmt.Sprint(int64(r["id"].(float64)))})
		d, _ := r["d"].(map[string]any)
		got, ok := d["dFullDPS"].(float64)
		seen = seen || ok
		if math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
			t.Fatalf("%v: dFullDPS %v, adding it moves FullDPS by %v", r["name"], got, want)
		}
	}
	if !seen {
		t.Fatalf("no row carries dFullDPS: %v", list)
	}
}

func outputEnv(o map[string]float64) *calculator.Environment {
	return &calculator.Environment{Player: &calculator.Actor{Output: o}}
}

// TestFullDPSFlagNamesAnOppositeMove: a MainDPS gain that lowers FullDPS is flagged; one that raises both is not.
func TestFullDPSFlagNamesAnOppositeMove(t *testing.T) {
	base := outputEnv(map[string]float64{"CombinedDPS": 100, "FullDPS": 300})
	if f := fullDPSFlag(base, outputEnv(map[string]float64{"CombinedDPS": 132.6, "FullDPS": 265.5}), mainDPS); len(f) != 1 || f[0] != "full dps: FullDPS -11.5% while MainDPS +32.6%" {
		t.Fatalf("flag %v", f)
	}
	if f := fullDPSFlag(base, outputEnv(map[string]float64{"CombinedDPS": 120, "FullDPS": 330}), mainDPS); len(f) != 0 {
		t.Fatalf("flag %v on a move both ways up", f)
	}
	if f := fullDPSFlag(base, outputEnv(map[string]float64{"CombinedDPS": 80, "FullDPS": 330}), "FullDPS"); len(f) != 0 {
		t.Fatalf("flag %v when ranking by FullDPS", f)
	}
}
