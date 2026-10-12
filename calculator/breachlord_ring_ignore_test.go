package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBreachlordRingIgnoreResistParses pins ModParser.lua:5441.
func TestBreachlordRingIgnoreResistParses(t *testing.T) {
	for _, tc := range []struct{ line, name, varName string }{
		{"Your Hits ignore Enemy Monster Lightning Resistances if all Equipped Rings are Synaptic Rings", "IgnoreLightningResistance", "SynapticRingEquipped"},
		{"Your Hits ignore Enemy Monster Cold Resistances if all Equipped Rings are Cryonic Rings", "IgnoreColdResistance", "CryonicRingEquipped"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("ParseMod(%q) = %v, want 1 mod and no Extra", tc.line, e)
		}
		m := e.ModList[0]
		if m.Name() != tc.name || m.Type() != mod.TypeFlag {
			t.Errorf("%q: got %s %s, want FLAG %s", tc.line, m.Type(), m.Name(), tc.name)
		}
		tags := m.Tags()
		if len(tags) != 1 {
			t.Fatalf("%q: %d tags, want 1 MultiplierThreshold", tc.line, len(tags))
		}
		mt, ok := tags[0].(*mod.MultiplierThresholdTag)
		if !ok || mt.TagThreshold == nil || *mt.TagThreshold != 2 || mt.TagUpper || mt.Variable != tc.varName {
			t.Errorf("%q: tag %#v, want MultiplierThreshold var=%s threshold=2", tc.line, tags[0], tc.varName)
		}
	}

	e := ParseMod("Hits ignore Enemy Monster Chaos Resistance if all Equipped Items are Shaper Items", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "IgnoreChaosResistance" {
		t.Errorf("control :5439: ParseMod = %v", e)
	}
}
