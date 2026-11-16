package mcp

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// basisRows are a saving that gains in the best case and loses in the worst case, where TotalEHP falls too, and a
// smaller saving that gains in both.
func basisRows() []map[string]any {
	return []map[string]any{
		{"name": "drawback", "gain": 1.0, "cost": -300.0, "d": map[string]float64{"dEHP": 10},
			"worst_gain": -10.0, "worst_d": map[string]float64{"dEHP": -20}, worstBaseKey: map[string]float64{"stat": 50, "TotalEHP": 80}},
		{"name": "clean", "gain": 5.0, "cost": -100.0, "d": map[string]float64{"dEHP": 5},
			"worst_gain": 2.0, "worst_d": map[string]float64{"dEHP": 1}, worstBaseKey: map[string]float64{"stat": 50, "TotalEHP": 80}},
	}
}

// TestSavingsLossOneBasis: loss_pct counts the worst-case stat and the worst-case TotalEHP, each against the worst-case
// build, so a drawback that only gains in the best case no longer ranks first at loss 0.
func TestSavingsLossOneBasis(t *testing.T) {
	rs := basisRows()
	(&costBudget{div: 150}).rankSavings(rs, 100, 100, false)
	if rs[0]["name"] != "clean" || rs[1]["loss_pct"] != 45.0 || rs[1]["loss_of"] != "worst_gain" {
		t.Fatalf("savings %v", rs)
	}
}

// TestPaysForItselfOneBasis: a saving pays for itself only when it loses nothing net on the savings basis; one that
// does shows its loss and savings_per_pct, and a downgrade's gain per divine is not taken at a 1 chaos cost.
func TestPaysForItselfOneBasis(t *testing.T) {
	cb := &costBudget{div: 150}
	rs := basisRows()
	for _, r := range rs {
		score, of := rowScore(r)
		cb.putPerDiv(r, score, of, 100, 100, false)
	}
	if rs[0]["pays_for_itself"] != nil || rs[0]["gain_per_div"] != nil || rs[0]["loss_pct"] != 45.0 || rs[0]["savings_per_pct"] != round(300.0/45) {
		t.Errorf("drawback %v", rs[0])
	}
	if rs[1]["pays_for_itself"] != true {
		t.Errorf("clean %v", rs[1])
	}
	down := map[string]any{"gain": -10.0, "cost": -100.0, "downgrade": true}
	cb.putPerDiv(down, -10, "gain", 100, 100, false)
	if down["gain_per_div"] != nil || down["loss_pct"] != 10.0 || down["savings_per_pct"] != 10.0 {
		t.Errorf("downgrade %v", down)
	}
	sortPerDiv(rs)
	if rs[0]["name"] != "clean" {
		t.Errorf("order %v", rs)
	}
}

// TestBigLosses: a watched stat other than the ranked one down a fifth or more, or a resistance down 20 points or more,
// is named; the ranked stat and smaller drops are not.
func TestBigLosses(t *testing.T) {
	env := func(out map[string]float64) *calculator.Environment {
		return &calculator.Environment{Player: &calculator.Actor{Output: out}}
	}
	base := env(map[string]float64{"CombinedDPS": 1000, "TotalEHP": 100, "Life": 50, "ChaosResist": 51, "FireResist": 75})
	after := env(map[string]float64{"CombinedDPS": 100, "TotalEHP": 68, "Life": 45, "ChaosResist": 17, "FireResist": 60})
	got := bigLosses(mainDPS, base, after, changeColumns(base, after, mainDPS))
	if want := []string{"TotalEHP -32%", "ChaosResist 51 -> 17"}; !slices.Equal(got, want) {
		t.Fatalf("bigLosses %v, want %v", got, want)
	}
	rs := []map[string]any{{"name": "a", "big_loss": got}, {"name": "b"}}
	markBigLoss(rs)
	if rs[0][dirtyKey] == nil || rs[1][dirtyKey] != nil {
		t.Errorf("marks %v", rs)
	}
}

// TestSavingsLossNoOffset: a flask that trades a tenth of the worst-case stat for a tenth more worst-case TotalEHP
// loses that tenth; the TotalEHP it adds does not hide it, so a smaller saving that loses nothing ranks first.
func TestSavingsLossNoOffset(t *testing.T) {
	wb := map[string]float64{"stat": 100, "TotalEHP": 100}
	rs := []map[string]any{
		{"name": "trade", "gain": -40.0, "cost": -300.0, "worst_gain": -10.8, "worst_d": map[string]float64{"dEHP": 11.5}, worstBaseKey: wb},
		{"name": "clean", "gain": 5.0, "cost": -100.0, "worst_gain": 2.0, "worst_d": map[string]float64{"dEHP": 1}, worstBaseKey: wb},
	}
	(&costBudget{div: 150}).rankSavings(rs, 400, 100, false)
	if rs[0]["name"] != "clean" || rs[1]["loss_pct"] != 10.8 {
		t.Fatalf("savings %v", rs)
	}
}
