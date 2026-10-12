package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDroppedTagConjunctRows pins two rows found by a TAG-COUNT differential over ModCache.
func TestDroppedTagConjunctRows(t *testing.T) {
	t.Run("you are blind is gated on not CannotBeBlinded", func(t *testing.T) {
		e := ParseMod("You are Blind", false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "Condition:Blinded" || m.Type() != mod.TypeFlag || !m.Value().Flag() {
			t.Fatalf("got %s %s %v, want Condition:Blinded FLAG true", m.Name(), m.Type(), m.Value())
		}
		var ct *mod.ConditionTag
		for _, tg := range m.Tags() {
			if c, ok := tg.(*mod.ConditionTag); ok {
				ct = c
			}
		}
		if ct == nil || len(ct.VarList) != 1 || ct.VarList[0] != "CannotBeBlinded" {
			t.Fatalf("Condition tag = %+v, want CannotBeBlinded", ct)
		}
		if !ct.Negative {
			t.Error("Negative = false, want true")
		}
	})

	t.Run("culling strike is a global unscalable max", func(t *testing.T) {
		e := ParseMod("Culling Strike", false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "CullPercent" || m.Type() != mod.TypeMAX || m.Value().Float() != 10 {
			t.Fatalf("got %s %s %v, want CullPercent MAX 10", m.Name(), m.Type(), m.Value())
		}
		var ge *mod.GlobalEffectTag
		for _, tg := range m.Tags() {
			if g, ok := tg.(*mod.GlobalEffectTag); ok {
				ge = g
			}
		}
		if ge == nil || len(ge.GlobalEffectList) != 1 || ge.GlobalEffectList[0] != "Global" {
			t.Fatalf("GlobalEffect tag = %+v, want Global", ge)
		}
		if !ge.UnscalableTag {
			t.Error("UnscalableTag = false, want true")
		}
	})
}
