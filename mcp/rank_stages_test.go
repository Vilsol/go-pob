package mcp

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// sigilBuild is a Templar casting Ice Nova with Sigil of Power at stages stages in config.
func sigilBuild(t *testing.T, s *Server, stages int) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "level": 70})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Ice Nova"}},
		map[string]any{"op": "add_group", "gems": []string{"Sigil of Power"}},
		map[string]any{"op": "set_main", "group": 1})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "sigilOfPowerStages", "value": stages})})
	return h
}

// TestStageConfigsAreAssumptions: a skill's stage or stack count set in config assumes the skill at that count all the
// time; it is listed with the forced conditions and questioning it halves it.
func TestStageConfigsAreAssumptions(t *testing.T) {
	n := 4.0
	b := &pob.PathOfBuilding{}
	b.Config.Inputs = []pob.Input{{Name: "sigilOfPowerStages", Number: &n}}
	if f := forcedConditions(b); !slices.Contains(f, "sigilOfPowerStages") {
		t.Fatalf("forced %v", f)
	}
	q := questioned(b, []string{"sigilOfPowerStages"}, nil)
	if len(q.Config.Inputs) != 1 || q.Config.Inputs[0].Number == nil || *q.Config.Inputs[0].Number != 2 {
		t.Fatalf("questioned inputs %+v", q.Config.Inputs)
	}
}

// TestRankWorstCaseHalvesStages: worst_case scores a row with Sigil of Power at half its configured stages, as setting
// that stage count and applying the row gives.
func TestRankWorstCaseHalvesStages(t *testing.T) {
	s := testServer(t)
	h := sigilBuild(t, s, 4)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "limit": 3, "worst_case": true})
	if f, _ := res["forced_conditions"].([]any); !slices.Contains(f, any("sigilOfPowerStages")) {
		t.Fatalf("forced_conditions %v", res["forced_conditions"])
	}
	half := sigilBuild(t, s, 2)
	for _, r := range rows(res) {
		want := dryDelta(t, s, "edit_tree", half, mainDPS, map[string]any{"op": "allocate", "node": fmt.Sprint(int64(r["id"].(float64)))})
		if wg := num(r, "worst_gain"); math.Abs(wg-want) > math.Max(1, 0.001*math.Abs(want)) {
			t.Fatalf("%v: worst_gain %v; at 2 stages it gains %v", r["name"], wg, want)
		}
	}
}
