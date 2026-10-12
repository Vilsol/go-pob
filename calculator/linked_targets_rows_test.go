package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestLinkedTargetsRowsParse pins ModParser.lua:4319/:4321/:4322.
func TestLinkedTargetsRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line      string
		name      string
		val       float64
		threshold float64
	}{
		{"25% increased Defences while you have at least four Linked targets", "Defences", 25, 4},
		{"10% increased Movement Speed while you have at least two Linked targets", "MovementSpeed", 10, 2},
		{"10% increased Movement Speed while you have a Linked target", "MovementSpeed", 10, 1},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Errorf("%q: want one mod parsed whole, got %+v", tc.line, e)
			continue
		}
		m := e.ModList[0]
		if m.Name() != tc.name || m.Type() != mod.TypeIncrease || m.Value().Float() != tc.val || len(m.Tags()) != 1 {
			t.Errorf("%q: got %s %s %v tags=%d", tc.line, m.Name(), m.Type(), m.Value().Float(), len(m.Tags()))
			continue
		}
		mt, ok := m.Tags()[0].(*mod.MultiplierThresholdTag)
		if !ok || mt.Variable != "LinkedTargets" || mt.TagThreshold == nil || *mt.TagThreshold != tc.threshold {
			t.Errorf("%q: tag %#v, want MultiplierThreshold LinkedTargets >= %v", tc.line, m.Tags()[0], tc.threshold)
		}
	}

	e := ParseMod("Link Skills have 20% increased Buff Effect if you have Linked to a target Recently", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "BuffEffect" || len(e.ModList[0].Tags()) != 2 {
		t.Errorf("link buff effect row: got %+v", e)
	}
}
