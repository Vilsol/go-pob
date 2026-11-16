package mcp

import (
	"testing"
	"time"

	"github.com/Vilsol/go-pob/calculator"
)

// TestWorstDAgreesWithWorstGain: where the change provides a questioned condition its worst_gain takes the gain's
// share (Vessel of Vinktar shocking the enemy itself), and worst_d's DPS change says the same.
func TestWorstDAgreesWithWorstGain(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "conditionEnemyShocked", "value": true},
		map[string]any{"op": "set_config", "var": "conditionShockEffect", "value": 30})})
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	env, _ := hd.envAt(hd.cur)
	text, _ := calculator.ItemDBText(calculator.ItemDBUnique, "Vessel of Vinktar, Topaz Flask")
	row := map[string]any{"gain": 1.0, changeKey: flaskChange(text, "Flask 1")}
	newRealism(b, env, mainDPS, time.Now().Add(time.Minute)).annotate([]map[string]any{row}, 1, true)
	w, _ := row["worst_gain"].(float64)
	wd, _ := row["worst_d"].(map[string]float64)
	if _, ok := row["worst_gain"]; !ok || wd["dDPS"] != w {
		t.Fatalf("worst_gain %v, worst_d %v", w, wd)
	}
}

// TestSavingsTieCountsGains: two flasks that save the same to within a few chaos and lose nothing rank by what they
// gain: Atziri's Promise's DPS beats a plain Amethyst Flask that saves 2 chaos more.
func TestSavingsTieCountsGains(t *testing.T) {
	rs := []map[string]any{
		{"name": "plain", "gain": 0.0, "cost": -5998.0},
		{"name": "atziri", "gain": 37.0, "cost": -5996.0},
		{"name": "small", "gain": 50.0, "cost": -100.0},
	}
	(&costBudget{div: 150}).rankSavings(rs, 1000, 100, false)
	if rs[0]["name"] != "atziri" || rs[1]["name"] != "plain" {
		t.Fatalf("order %v", rs)
	}
}
