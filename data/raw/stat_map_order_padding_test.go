package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestStatMapOrderPaddingRealignsLaterLevels pins skills.lua:339-348.
func TestStatMapOrderPaddingRealignsLaterLevels(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	ge := poe.GrantedEffectByID("SummonSpectralWolf")
	if ge == nil {
		t.Fatal("REFUSING: SummonSpectralWolf is not in the loaded data — wrong fixture")
	}
	cg := GetCalculatedGrantedEffect(ge)

	stats := cg.GetCalculatedStats()
	want := []string{
		"chance_to_cast_on_kill_%_target_self",
		"spectral_wolf_grants_attack_minimum_added_physical_damage",
		"spectral_wolf_grants_attack_maximum_added_physical_damage",
		"base_display_minion_actor_level",
	}
	if len(stats) < len(want) {
		t.Fatalf("REFUSING: only %d stats, want at least %d — wrong fixture", len(stats), len(want))
	}
	for i, id := range want {
		if stats[i] != id {
			t.Fatalf("REFUSING: stat %d is %q, want %q — the fixture's stat order has moved and "+
				"the index assertions below no longer mean anything", i, stats[i], id)
		}
	}

	levels := cg.GetCalculatedLevels()

	first, ok := levels[10]
	if !ok {
		t.Fatalf("REFUSING: no level 10 — the fixture's levels are %d entries", len(levels))
	}
	if len(first.Values) != 4 || first.Values[0] != 100 {
		t.Errorf("level 10 values = %v, want 4 entries starting at the cast chance 100", first.Values)
	}

	later, ok := levels[25]
	if !ok {
		t.Fatalf("REFUSING: no level 25 — the fixture's levels are %d entries", len(levels))
	}
	if len(later.Values) != 4 {
		t.Fatalf("level 25 has %d values (%v), want 4 — the missing stat is not padded, so every "+
			"value is assigned to the stat before it", len(later.Values), later.Values)
	}
	for i, w := range []float64{0, 8, 16, 78} {
		if later.Values[i] != w {
			t.Errorf("level 25 value %d (%s) = %g, want %g",
				i, stats[i], later.Values[i], w)
		}
	}
	if len(later.StatInterpolation) != 4 || later.StatInterpolation[0] != 0 {
		t.Errorf("level 25 statInterpolation = %v, want a leading 0 and 4 entries", later.StatInterpolation)
	}
}
