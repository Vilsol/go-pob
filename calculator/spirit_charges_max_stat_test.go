package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMaximumSpiritChargesStatName pins ModParser.lua:529.
func TestMaximumSpiritChargesStatName(t *testing.T) {
	e := ParseMod("+1 to Maximum Spirit Charges per Abyss Jewel affecting you", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("got %v", e)
	}
	m := e.ModList[0]
	if m.Name() != "SpiritChargesMax" || m.Type() != mod.TypeBase || m.Value().Float() != 1 {
		t.Fatalf("got %s %s %v, want SpiritChargesMax BASE 1", m.Name(), m.Type(), m.Value())
	}
	var mt *mod.MultiplierTag
	for _, tg := range m.Tags() {
		if x, ok := tg.(*mod.MultiplierTag); ok {
			mt = x
		}
	}
	if mt == nil || len(mt.VariableList) != 1 {
		t.Fatalf("Multiplier tag = %+v, want one variable", mt)
	}
}
