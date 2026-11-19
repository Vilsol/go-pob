package calculator

import "testing"

// TestSelfAuraSetsAffectedByAura pins CalcPerform.lua:2501 and 2520: an aura that reaches the player or the minion
// sets AffectedByAura on it.
func TestSelfAuraSetsAffectedByAura(t *testing.T) {
	env := corpusOnlyEnv(t, "0009-trickster-cobra-lash")
	if env.Minion == nil {
		t.Fatalf("PREMISE BROKEN: no minion")
	}
	// PoB's dump sets both.
	if !env.ModDB.Conditions["AffectedByAura"] {
		t.Errorf("player AffectedByAura not set")
	}
	if !env.Minion.ModDB.Conditions["AffectedByAura"] {
		t.Errorf("minion AffectedByAura not set")
	}
}
