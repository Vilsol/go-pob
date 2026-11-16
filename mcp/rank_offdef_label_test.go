package mcp

import (
	"testing"
)

// TestOffenceDefenceRowsSayWhenTheScoreIsNotEHP: PoB's defence half counts life, armour, energy shield, evasion and
// regeneration (CalcsTab.lua:749-754); a row whose score and EHP move apart says so, one where they agree does not.
func TestOffenceDefenceRowsSayWhenTheScoreIsNotEHP(t *testing.T) {
	env := outputEnv(map[string]float64{"TotalEHP": 10000})
	list := []map[string]any{
		{"name": "regen up, EHP down", "defence_pct": 12.0, "d": map[string]float64{"dEHP": -4300}},
		{"name": "score without EHP", "defence_pct": 31.0, "d": map[string]float64{}},
		{"name": "agree", "defence_pct": 3.0, "d": map[string]float64{"dEHP": 250}},
	}
	ehpNote(list, env)
	for i, want := range []bool{true, true, false} {
		f, _ := list[i]["flags"].([]string)
		if got := len(f) > 0; got != want {
			t.Errorf("%v: flags %v, want flagged %v", list[i]["name"], f, want)
		}
	}
}
