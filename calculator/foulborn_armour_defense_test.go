package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestFoulbornArmourDefenseMods: Foulborn Ancestral Vision — ModParser.lua:3817-3821 and CalcDefence.lua:1561-1567.
func TestFoulbornArmourDefenseMods(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		suppress, percent, pctArmr float64
		wantMax, wantAvg, wantMin  float64
	}{
		{"saturated", 100, 50, 200, 100, 100, 100},
		{"fractional", 1, 50, 200, 100, 50, 0},
		{"no suppression", 0, 50, 200, 100, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moddb.NewModDB()
			db.AddMod(mod.NewFloat("SpellSuppressionChance", mod.TypeBase, tc.suppress).Source("Test"))
			db.AddMod(mod.NewFloat("SpellSuppressionAppliesToChanceToDefendWithArmourPercent",
				mod.TypeMAX, tc.percent).Source("Test"))
			db.AddMod(mod.NewFloat("SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour",
				mod.TypeMAX, tc.pctArmr).Source("Test"))

			got := foulbornArmourDefenseMods(db)
			testza.AssertEqual(t, 3, len(got), "the block emits Max, Average and Min calc arms")
			testza.AssertEqual(t, tc.wantMax, got[0].Value().Float(), "Max calc arm")
			testza.AssertEqual(t, tc.wantAvg, got[1].Value().Float(), "Average calc arm")
			testza.AssertEqual(t, tc.wantMin, got[2].Value().Float(), "Min calc arm")
		})
	}
}

// TestArmourDefenseIsReadAfterItsRuntimeProducers guards the ORDERING half of the fix, which no value assertion in this tree can currently reach.
func TestArmourDefenseIsReadAfterItsRuntimeProducers(t *testing.T) {
	src, err := os.ReadFile("calc_defence.go")
	if err != nil {
		t.Fatalf("read calc_defence.go: %v", err)
	}
	s := string(src)

	const producer = "foulbornArmourDefenseMods(modDB)"
	const consumer = `actor.Output["ArmourDefense"] = modDB.Max(nil, "ArmourDefense") / 100`

	if n := strings.Count(s, producer); n != 1 {
		t.Fatalf("REFUSING: %q appears %d times, want 1 — this guard can no longer locate "+
			"the producer's call site", producer, n)
	}
	if n := strings.Count(s, consumer); n != 1 {
		t.Fatalf("REFUSING: %q appears %d times, want 1 — the pin writes output.ArmourDefense "+
			"exactly once, at :1569", consumer, n)
	}
	if strings.Index(s, consumer) < strings.Index(s, producer) {
		t.Errorf("output.ArmourDefense is read BEFORE the runtime block that adds ArmourDefense " +
			"mods; every runtime producer of it is then a dead branch")
	}
}
