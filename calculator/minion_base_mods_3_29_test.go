package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestMinionBaseModsMatchPoB covers CalcPerform.lua:1146-1160 — the MINION twin of the base-mod block in env.go.
func TestMinionBaseModsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0002-pathfinder-unknown.xml")
	if env.Minion == nil || env.Minion.ModDB == nil {
		t.Fatal("REFUSING: the fixture no longer produces a minion actor")
	}
	db := env.Minion.ModDB

	if n := len(db.Mods["CritDegenMultiplier"]); n != 0 {
		t.Errorf("%d CritDegenMultiplier mod(s) on the minion; the name appears in NO file of the "+
			"pinned PoB and nothing in this tree reads it (CalcPerform.lua:1147)", n)
	}

	var sawDot bool
	for _, m := range db.Mods["DotMultiplier"] {
		if m.GetSource() != mod.SourceBase || m.Type() != mod.TypeBase {
			continue
		}
		sawDot = true
		if got := m.Value().Float(); got != 30 {
			t.Errorf("minion Base DotMultiplier = %v, want 30 "+
				"(monsterConstants critical_ailment_dot_multiplier_+, Misc.lua:166)", got)
		}
		var conditioned bool
		for _, tg := range m.Tags() {
			if c, ok := tg.(*mod.ConditionTag); ok {
				for _, v := range c.VarList {
					if v == "CriticalStrike" {
						conditioned = true
					}
				}
			}
		}
		if !conditioned {
			t.Error("minion Base DotMultiplier carries no Condition:CriticalStrike tag; without it " +
				"every minion DoT gets the crit multiplier (CalcPerform.lua:1147)")
		}
	}
	if !sawDot {
		t.Error("no Base DotMultiplier on the minion; PoB seeds one on all 280 minion dumps " +
			"(CalcPerform.lua:1147)")
	}

	var sawEDR, sawStaleElementalResist bool
	for _, name := range []string{"ElementalDamageReduction", "ElementalResist"} {
		for _, m := range db.Mods[name] {
			if m.GetSource() != mod.SourceBase {
				continue
			}
			var perEndurance bool
			for _, tg := range m.Tags() {
				if mt, ok := tg.(*mod.MultiplierTag); ok {
					for _, v := range mt.VariableList {
						if v == "EnduranceCharge" {
							perEndurance = true
						}
					}
				}
			}
			if !perEndurance {
				continue
			}
			if name == "ElementalDamageReduction" {
				sawEDR = true
				if got := m.Value().Float(); got != float64(data.ElementalDamageReductionPercentPerEnduranceCharge) {
					t.Errorf("minion Base ElementalDamageReduction = %v, want %v", got,
						data.ElementalDamageReductionPercentPerEnduranceCharge)
				}
			} else {
				sawStaleElementalResist = true
			}
		}
	}
	if sawStaleElementalResist {
		t.Error("the minion still grants ElementalResist per endurance charge; PoB grants " +
			"ElementalDamageReduction (CalcPerform.lua:1157) and endurance charges stopped " +
			"granting resistance in the game years ago — this is the exact wrong name the PLAYER " +
			"side was corrected off two batches ago")
	}
	if !sawEDR {
		t.Error("no Base ElementalDamageReduction per endurance charge on the minion " +
			"(CalcPerform.lua:1157)")
	}
}
