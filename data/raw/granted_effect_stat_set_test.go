package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestStatSetResolvesToTheEffectsOwnSet pins which GrantedEffectStatSet a granted effect reads its implicit and constant stats from.
func TestStatSetResolvesToTheEffectsOwnSet(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	ge := poe.GrantedEffectByID("TemporalChains")
	if ge == nil {
		t.Fatal("REFUSING: TemporalChains granted effect not found; the probe measured nothing")
	}
	calc := GetCalculatedGrantedEffect(ge)

	constants := calc.GetCalculatedConstantStats()
	for stat, want := range map[string]float64{
		"buff_time_passed_+%_other_than_temporal_chains": -25,
		"curse_effect_+%_final_vs_players":               -50,
	} {
		got, ok := constants[stat]
		if !ok {
			t.Errorf("TemporalChains constantStats is missing %q (have %v)", stat, constants)
			continue
		}
		if got != want {
			t.Errorf("TemporalChains constantStats[%q] = %v, want %v", stat, got, want)
		}
	}

	fireballOnly := []string{
		"base_chance_to_ignite_%",
		"base_is_projectile",
		"quality_display_active_skill_ignite_damage_is_gem",
	}
	if _, ok := constants["base_chance_to_ignite_%"]; ok {
		t.Errorf("TemporalChains carries Fireball's constant base_chance_to_ignite_%%; the stat "+
			"set resolved to slice index 0 rather than to its own (constants: %v)", constants)
	}
	stats := calc.GetCalculatedStats()
	inStats := func(id string) bool {
		for _, s := range stats {
			if s == id {
				return true
			}
		}
		return false
	}
	for _, s := range fireballOnly {
		if inStats(s) {
			t.Errorf("TemporalChains stat list carries Fireball's %q", s)
		}
	}
	if !inStats("base_deal_no_damage") {
		t.Errorf("TemporalChains is missing its own implicit base_deal_no_damage (stats: %v)", stats)
	}

	fb := poe.GrantedEffectByID("Fireball")
	if fb == nil {
		t.Fatal("REFUSING: Fireball granted effect not found; the control cannot run")
	}
	fbCalc := GetCalculatedGrantedEffect(fb)
	if got, ok := fbCalc.GetCalculatedConstantStats()["base_chance_to_ignite_%"]; !ok || got != 25 {
		t.Errorf("CONTROL: Fireball's own constant base_chance_to_ignite_%% = %v (present=%v), want 25 — "+
			"the resolver is not returning the correct set either", got, ok)
	}
}
