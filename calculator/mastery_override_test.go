package calculator

import (
	"math"
	"os"
	"testing"
)

// TestOverriddenMasteryDropsItsSelectedEffect pins PassiveSpec.lua:1393-1400, which is an IF/ELSE and not two independent steps.
//
//	if node.type == "Mastery" and self.masterySelections[id] then
//	    local effect = self.tree.masteryEffects[self.masterySelections[id]]
//	    if effect and self.allocNodes[id] then
//	        if self.hashOverrides and self.hashOverrides[id] then
//	            self:ReplaceNode(node, self.hashOverrides[id])   -- the override WINS
//	        else
func TestOverriddenMasteryDropsItsSelectedEffect(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	const build = "0037-ascendant-fire-trap-of-blasting-2.xml"
	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
	}
	want, ok := rec.Output["Life"]
	if !ok || want == 0 {
		t.Fatalf("REFUSING: PoB reports no nonzero Life for %s", build)
	}

	env := corpusEnvForLoadedData(t, build)
	got := env.Player.Output["Life"]
	if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
		t.Errorf("%s: Life = %v, PoB reports %v (excess %v — the selected mastery effect of a "+
			"node its Runegraft should have replaced)", build, got, want, got-want)
	}
}
