package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFungalGroundRowMatchesThePin pins ModParser.lua:5533-5537.
func TestFungalGroundRowMatchesThePin(t *testing.T) {
	e := ParseMod("You have Fungal Ground around you while stationary", false)
	if e == nil {
		t.Fatal("ParseMod returned nil")
	}
	if e.Extra != "" {
		t.Fatalf("leftover Extra %q — the line does not parse whole", e.Extra)
	}
	if len(e.ModList) != 3 {
		t.Fatalf("produced %d mods, want 3 (one ExtraAura + two EnemyModifier)", len(e.ModList))
	}

	type got struct {
		outer, inner string
		value        float64
	}
	seen := make([]got, 0, len(e.ModList))
	for _, m := range e.ModList {
		g := got{outer: m.Name()}
		switch v := m.Value().ValueList.(type) {
		case mod.ExtraAura:
			g.inner = v.Mod.Name()
			g.value = v.Mod.Value().Float()
		case mod.EnemyModifier:
			g.inner = v.Mod.Name()
			g.value = v.Mod.Value().Float()
		default:
			t.Fatalf("unexpected mod payload %T on %q", v, m.Name())
		}
		seen = append(seen, g)
	}

	want := map[got]bool{
		{outer: "ExtraAura", inner: "ChaosResist", value: 25}:          false,
		{outer: "EnemyModifier", inner: "ChaosResist", value: -10}:     false,
		{outer: "EnemyModifier", inner: "ElementalResist", value: -10}: false,
	}
	for _, g := range seen {
		if _, ok := want[g]; !ok {
			t.Errorf("unexpected mod %+v; the pin's row emits only the three below", g)
			continue
		}
		want[g] = true
	}
	for g, found := range want {
		if !found {
			t.Errorf("missing mod %+v", g)
		}
	}
}
