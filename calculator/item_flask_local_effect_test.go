package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFlaskEffectIncSumsLocalEffect: Item.lua:2400 sums TWO names into flaskData.effectInc: `FlaskEffect` and `LocalEffect`.
//
//pob:block:src/Classes/Item.lua:2400
func TestFlaskEffectIncSumsLocalEffect(t *testing.T) {
	const flask = "Rarity: RARE\nTest Flask\nDivine Life Flask\n"

	build := func(extra ...mod.Mod) map[string]float64 {
		item := ParseItem(flask)
		if item.Base == nil || item.Base.Flask == nil {
			t.Fatalf("REFUSING: Divine Life Flask resolved no flask base (%v)", item.Base)
		}
		item.FlaskData = map[string]float64{}
		item.buildModListForSlotNum(append([]mod.Mod{}, extra...), 0)
		return item.FlaskData
	}

	t.Run("control: FlaskEffect alone", func(t *testing.T) {
		if got := build(mod.NewFloat("FlaskEffect", mod.TypeIncrease, 20))["effectInc"]; got != 20 {
			t.Fatalf("REFUSING: effectInc = %v with FlaskEffect 20 — the pre-existing term is broken", got)
		}
	})

	t.Run("LocalEffect reaches effectInc", func(t *testing.T) {
		if got := build(mod.NewFloat("LocalEffect", mod.TypeIncrease, 30))["effectInc"]; got != 30 {
			t.Errorf("effectInc = %v, want 30", got)
		}
	})

	t.Run("the two terms are additive", func(t *testing.T) {
		got := build(
			mod.NewFloat("FlaskEffect", mod.TypeIncrease, 20),
			mod.NewFloat("LocalEffect", mod.TypeIncrease, 30),
		)["effectInc"]
		if got != 50 {
			t.Errorf("effectInc = %v, want 50", got)
		}
	})

	t.Run("LocalEffect does not escape as a global", func(t *testing.T) {
		item := ParseItem(flask)
		item.FlaskData = map[string]float64{}
		global := item.buildModListForSlotNum([]mod.Mod{
			mod.NewFloat("LocalEffect", mod.TypeIncrease, 30),
		}, 0)
		if n := countModsNamed(global, "LocalEffect"); n != 0 {
			t.Errorf("LocalEffect survived into the global list %d times; calcLocal must drain it", n)
		}
	})

	t.Run("the bare phrase aliases to LocalEffect and lands in effectInc", func(t *testing.T) {
		entry := ParseMod("30% increased Effect", false)
		if entry == nil || len(entry.ModList) != 1 {
			t.Fatalf("REFUSING: \"30%% increased Effect\" parsed to %v — the row is gone, not renamed", entry)
		}
		if got := entry.ModList[0].Name(); got != "LocalEffect" {
			t.Fatalf("bare \"effect\" aliases to %q, want LocalEffect (ModParser.lua:855)", got)
		}
		if got := build(entry.ModList[0])["effectInc"]; got != 30 {
			t.Errorf("effectInc = %v, want 30 — the parsed mod does not reach the consumer", got)
		}
	})
}
