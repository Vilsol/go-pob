package mcp

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// setGemCount stores n as the count of gem i in group g, as PoB's skills tab count field does.
func setGemCount(t *testing.T, s *Server, h string, g, i, n int) {
	t.Helper()
	hd, e := s.get(h)
	if e != nil {
		t.Fatal(e.Message)
	}
	b, e := hd.parseRev(hd.cur)
	if e != nil {
		t.Fatal(e.Message)
	}
	(*groupsOf(b))[g-1].Gems[i].Count = &n
	if _, e := s.commit(hd, b); e != nil {
		t.Fatal(e.Message)
	}
}

func fullDPSTotems(t *testing.T, s *Server, count int) string {
	t.Helper()
	h := totemBuild(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 1, "gems": []string{"Freezing Pulse", "Spell Totem"}, "full_dps": true})})
	setGemCount(t, s, h, 1, 0, count)
	return h
}

func buildWarnings(t *testing.T, s *Server, h string) []string {
	t.Helper()
	list := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})["warnings"].([]any)
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.(string))
	}
	return out
}

func totemLimit(t *testing.T, s *Server, h string) int {
	t.Helper()
	return int(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"ActiveTotemLimit"}})["stats"].(map[string]any)["ActiveTotemLimit"].(float64))
}

// TestTotemCountOverTheLimitWarns: PoB multiplies Full DPS by the gem's stored count whatever the totem limit
// (Calcs.lua:253-256); a count above the limit is named in the build's warnings, and a count at the limit is not.
func TestTotemCountOverTheLimitWarns(t *testing.T) {
	s := testServer(t)
	h := fullDPSTotems(t, s, 1)
	limit := totemLimit(t, s, h)
	setGemCount(t, s, h, 1, 0, limit)
	if w := buildWarnings(t, s, h); slices.ContainsFunc(w, func(x string) bool { return strings.Contains(x, "Full DPS counts") }) {
		t.Fatalf("count at the limit %d warns: %v", limit, w)
	}
	setGemCount(t, s, h, 1, 0, limit+2)
	w := buildWarnings(t, s, h)
	if !slices.ContainsFunc(w, func(x string) bool { return strings.Contains(x, "Full DPS counts Freezing Pulse") }) {
		t.Fatalf("count %d over the limit %d does not warn: %v", limit+2, limit, w)
	}
}

// TestTotemCountOverTheLimitWarnsForAnotherSkill: a Full DPS totem skill that is not the main skill is held to the
// limit it has as the main skill.
func TestTotemCountOverTheLimitWarnsForAnotherSkill(t *testing.T) {
	s := testServer(t)
	h := fullDPSBuild(t, s)
	setGemCount(t, s, h, 1, 0, 3)
	f := call(t, s, "new_build", map[string]any{"class": "Templar", "from_handle": h})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": f, "ops": ops(map[string]any{"op": "set_main", "group": 1})})
	want := fmt.Sprintf("Full DPS counts Freezing Pulse 3 times; its active totem limit is %d", totemLimit(t, s, f))
	if w := buildWarnings(t, s, h); !slices.Contains(w, want) {
		t.Fatalf("want %q in %v", want, w)
	}
}

// TestTotemsSummonedConfigOverTheLimitWarns: the summoned-totems config is taken as given (CalcOffence.lua:1434).
func TestTotemsSummonedConfigOverTheLimitWarns(t *testing.T) {
	s := testServer(t)
	h := totemBuild(t, s)
	limit := totemLimit(t, s, h)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "TotemsSummoned", "value": limit + 2})})
	if w := buildWarnings(t, s, h); !slices.ContainsFunc(w, func(x string) bool { return strings.Contains(x, "totems summoned") }) {
		t.Fatalf("summoned config over the limit %d does not warn: %v", limit, w)
	}
}

// TestRankRemoveTotemLimitFlagsAndClampsWorstCase: with the stored count at the totem limit, removing Ancestral Bond
// leaves Full DPS counting a totem the build cannot have; the row is flagged, and its loss and worst_case score it with
// the count held to the new limit, as deallocating it and lowering the count gives.
func TestRankRemoveTotemLimitFlagsAndClampsWorstCase(t *testing.T) {
	s := testServer(t)
	h := fullDPSTotems(t, s, 1)
	limit := totemLimit(t, s, h)
	setGemCount(t, s, h, 1, 0, limit)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "FullDPS", "limit": 25, "worst_case": true, "watch": []string{"FullDPS"}})
	r := removeRow(t, res, 41970)
	if !flagged(r, "warning: Full DPS counts", "Freezing Pulse") {
		t.Fatalf("Ancestral Bond row not flagged: %v", r)
	}
	f := call(t, s, "new_build", map[string]any{"class": "Templar", "from_handle": h})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": f, "ops": ops(map[string]any{"op": "deallocate", "node": "41970"})})
	setGemCount(t, s, f, 1, 0, totemLimit(t, s, f))
	want := stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"FullDPS"}}), "FullDPS") -
		stat(call(t, s, "get_stats", map[string]any{"handle": f, "stats": []string{"FullDPS"}}), "FullDPS")
	if wl, _ := r["worst_loss"].(float64); math.Abs(r["loss"].(float64)-want) > math.Max(1, 0.001*want) || math.Abs(wl-want) > math.Max(1, 0.001*want) {
		t.Fatalf("worst_loss %v, loss %v; deallocating with the count at the new limit loses %v", r["worst_loss"], r["loss"], want)
	}
}

// TestTotemCountWithAnotherTotemSkillWarns: the active totem limit is shared, so a Full DPS count at the limit plus a
// Decoy Totem the build also summons is more totems than it can have out.
func TestTotemCountWithAnotherTotemSkillWarns(t *testing.T) {
	s := testServer(t)
	h := fullDPSTotems(t, s, 1)
	setGemCount(t, s, h, 1, 0, totemLimit(t, s, h))
	if w := buildWarnings(t, s, h); slices.ContainsFunc(w, func(x string) bool { return strings.Contains(x, "totem limit") }) {
		t.Fatalf("count at the limit warns: %v", w)
	}
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Decoy Totem"}})})
	w := buildWarnings(t, s, h)
	if !slices.ContainsFunc(w, func(x string) bool {
		return strings.Contains(x, "Full DPS counts Freezing Pulse") && strings.Contains(x, "Decoy Totem")
	}) {
		t.Fatalf("Decoy Totem beside a count at the limit does not warn: %v", w)
	}
}

// TestRankTotemLimitLossCountsFullDPSAfter: a Full DPS totem counted at its limit keeps PoB's stored count when a
// removal lowers the limit (Calcs.lua:253-256); the row's dFullDPS counts the totems the limit leaves, as dDPS and loss
// do, and says so.
func TestRankTotemLimitLossCountsFullDPSAfter(t *testing.T) {
	s := testServer(t)
	h := fullDPSTotems(t, s, 1)
	l := totemLimit(t, s, h)
	setGemCount(t, s, h, 1, 0, l)
	fullDPS := func(h string) float64 {
		return stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"FullDPS"}}), "FullDPS")
	}
	f := call(t, s, "new_build", map[string]any{"class": "Templar", "from_handle": h})["handle"].(string)
	setGemCount(t, s, f, 1, 0, l-1)
	want := fullDPS(f) + dryDelta(t, s, "edit_tree", f, "FullDPS", map[string]any{"op": "deallocate", "node": "41970"}) - fullDPS(h)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	r := removeRow(t, res, 41970)
	got := num(r["d"], "dFullDPS")
	if want >= 0 || math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
		t.Fatalf("dFullDPS %v, the %d totems the lower limit leaves give %v: %v", got, l-1, want, r)
	}
	if !flagged(r, "totems: ", "dFullDPS counts") {
		t.Fatalf("no word that dFullDPS counts the totems the limit leaves: %v", r["flags"])
	}
}

// TestRankGainCountsTheTotemsLeft: a totem main skill's DPS is per totem, so a change that summons fewer totems gains
// what the totems it leaves deal, as the row's dDPS counts it.
func TestRankGainCountsTheTotemsLeft(t *testing.T) {
	sk := &calculator.ActiveSkill{SkillFlags: map[calculator.SkillFlag]bool{calculator.SkillFlagTotem: true}}
	env := func(dps, totems float64) *calculator.Environment {
		return &calculator.Environment{Player: &calculator.Actor{MainSkill: sk, Output: map[string]float64{"CombinedDPS": dps, "TotemsSummoned": totems}}}
	}
	if g, _, _ := rankGain(calculator.PowerStat{Stat: "CombinedDPS"}, mainDPS, env(100, 4), env(150, 3)); g != 12.5 {
		t.Fatalf("gain %v, 3 totems of 150 against 4 of 100 is 12.5 per totem", g)
	}
}
