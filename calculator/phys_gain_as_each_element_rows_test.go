package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPhysicalGainAsEachElementRows pins ModParser.lua:3467 and :3472.
func TestPhysicalGainAsEachElementRows(t *testing.T) {
	wantNames := []string{"PhysicalDamageGainAsLightning", "PhysicalDamageGainAsCold", "PhysicalDamageGainAsFire"}

	assertTriple := func(t *testing.T, e *ModCacheEntry, wantValue float64) []mod.Mod {
		t.Helper()
		if e == nil || e.Extra != "" || len(e.ModList) != 3 {
			t.Fatalf("got %v", e)
		}
		for i, want := range wantNames {
			m := e.ModList[i]
			if m.Name() != want || m.Type() != mod.TypeBase || m.Value().Float() != wantValue {
				t.Errorf("mod %d = %s %s %v, want %s BASE %v", i, m.Name(), m.Type(), m.Value(), want, wantValue)
			}
		}
		return e.ModList
	}

	t.Run("per spirit charge", func(t *testing.T) {
		mods := assertTriple(t, ParseMod("Gain 5% of Physical Damage as Extra Damage of each Element per Spirit Charge", false), 5)
		for _, m := range mods {
			var mt *mod.MultiplierTag
			for _, tg := range m.Tags() {
				if x, ok := tg.(*mod.MultiplierTag); ok {
					mt = x
				}
			}
			if mt == nil || len(mt.VariableList) != 1 || mt.VariableList[0] != "SpiritCharge" {
				t.Errorf("%s Multiplier tag = %+v, want SpiritCharge", m.Name(), mt)
			}
		}
	})

	t.Run("if N influence items are equipped", func(t *testing.T) {
		mods := assertTriple(t, ParseMod("Gain 15% of Physical Damage as Extra Damage of each Element if 6 Shaper Items are Equipped", false), 15)
		for _, m := range mods {
			var mt *mod.MultiplierThresholdTag
			for _, tg := range m.Tags() {
				if x, ok := tg.(*mod.MultiplierThresholdTag); ok {
					mt = x
				}
			}
			if mt == nil || mt.Variable != "ShaperItem" || mt.TagThreshold == nil || *mt.TagThreshold != 6 {
				t.Errorf("%s MultiplierThreshold tag = %+v, want ShaperItem >= 6", m.Name(), mt)
			}
		}
	})
}
