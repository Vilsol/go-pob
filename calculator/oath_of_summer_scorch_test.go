package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOathOfSummerIgniteInsteadScorch pins ModParser.lua:3026, the only producer of `IgniteCanScorch`, and the two CalcOffence.lua arms that read it.
func TestOathOfSummerIgniteInsteadScorch(t *testing.T) {
	e := ParseMod("Hits that would Ignite instead Scorch", false)
	if e == nil || e.Extra != "" {
		t.Fatalf("line did not parse whole: %v", e)
	}
	got := map[string]bool{}
	for _, m := range e.ModList {
		if m.Type() != mod.TypeFlag || !m.Value().Flag() {
			t.Errorf("%s is %s %v, want FLAG true", m.Name(), m.Type(), m.Value())
		}
		got[m.Name()] = true
	}
	for _, want := range []string{"IgniteCanScorch", "CannotIgnite"} {
		if !got[want] {
			t.Errorf("missing %s; got %v", want, got)
		}
	}
	if len(e.ModList) != 2 {
		t.Errorf("got %d mods, want exactly 2: %v", len(e.ModList), e.ModList)
	}
}
