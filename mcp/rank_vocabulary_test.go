package mcp

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestBigLossOtherDPS: ranking FullDPS, a main skill down 71.5% is big_loss; ranking MainDPS, Full DPS down by half is.
func TestBigLossOtherDPS(t *testing.T) {
	env := func(out map[string]float64) *calculator.Environment {
		return &calculator.Environment{Player: &calculator.Actor{Output: out}}
	}
	base := env(map[string]float64{"CombinedDPS": 1000, "FullDPS": 3000, "TotalEHP": 100})
	after := env(map[string]float64{"CombinedDPS": 285, "FullDPS": 3300, "TotalEHP": 100})
	if got := bigLosses("FullDPS", base, after, changeColumns(base, after, dpsColumn("FullDPS"))); !slices.Equal(got, []string{"MainDPS -71.5%"}) {
		t.Errorf("ranking FullDPS: %v", got)
	}
	after = env(map[string]float64{"CombinedDPS": 1100, "FullDPS": 1500, "TotalEHP": 100})
	if got := bigLosses(mainDPS, base, after, changeColumns(base, after, mainDPS)); !slices.Equal(got, []string{"FullDPS -50%"}) {
		t.Errorf("ranking MainDPS: %v", got)
	}
}

// TestUnusableOnlyWhatCannotBeUsed: an EHP loss is big_loss, never unusable, in every mode.
func TestUnusableOnlyWhatCannotBeUsed(t *testing.T) {
	rs := []map[string]any{{"name": "ehp", "gain": 90.0, "d": map[string]float64{"dEHP": -30}}}
	markUnusable(rs)
	if rs[0]["unusable"] != nil || rs[0][dirtyKey] != nil {
		t.Fatalf("EHP loss marked unusable: %v", rs[0])
	}
}

// TestPaysForItselfNoWatchedLoss: a saving that gains pays for itself only when usable and losing nothing in any
// watched stat on its basis: not when it loses spell suppression (Crown of the Inward Eye), DPS while TotalEHP ranks,
// or is unusable (Replica Kaom's Heart).
func TestPaysForItselfNoWatchedLoss(t *testing.T) {
	cb := &costBudget{div: 150}
	for _, r := range []map[string]any{
		{"name": "suppression", "gain": 5.0, "cost": -10.0, "flags": []string{"loses: SpellSuppressionChance 100 -> 78"}},
		{"name": "dps", "gain": 5.0, "cost": -10.0, "d": map[string]float64{"dDPS": -500, "dEHP": 5}},
		{"name": "unusable", "gain": 5.0, "cost": -10.0, "unusable": "You do not meet the Strength requirement"},
	} {
		cb.putPerDiv(r, 5, "gain", 100, 100, r["name"] == "dps")
		if r["pays_for_itself"] != nil {
			t.Errorf("%s pays for itself: %v", r["name"], r)
		}
		if r["name"] != "unusable" && len(anyStrings(r["watched_loss"])) == 0 {
			t.Errorf("%s names no watched loss: %v", r["name"], r)
		}
	}
	clean := map[string]any{"gain": 5.0, "cost": -10.0, "d": map[string]float64{"dEHP": 5}}
	cb.putPerDiv(clean, 5, "gain", 100, 100, false)
	if clean["pays_for_itself"] != true {
		t.Errorf("clean saving: %v", clean)
	}
}

func anyStrings(v any) []string {
	s, _ := v.([]string)
	return s
}
