package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestNotWarcriedRecentlyTagIsNegated pins ModParser.lua:1854.
func TestNotWarcriedRecentlyTagIsNegated(t *testing.T) {
	e := ParseMod("100% increased Warcry Speed if you have not Warcried Recently", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("got %v", e)
	}
	m := e.ModList[0]
	if m.Name() != "WarcrySpeed" || m.Type() != mod.TypeIncrease || m.Value().Float() != 100 {
		t.Fatalf("got %s %s %v, want WarcrySpeed INC 100", m.Name(), m.Type(), m.Value())
	}
	var ct *mod.ConditionTag
	for _, tg := range m.Tags() {
		if c, ok := tg.(*mod.ConditionTag); ok {
			ct = c
		}
	}
	if ct == nil || len(ct.VarList) != 1 || ct.VarList[0] != "UsedWarcryRecently" {
		t.Fatalf("Condition tag = %+v, want UsedWarcryRecently", ct)
	}
	if !ct.Negative {
		t.Error("Negative = false, want true")
	}
}
