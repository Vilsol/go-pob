package poe2skills

import (
	"testing"

	"github.com/Vilsol/go-pob/data/poe2"
)

// PoB-PoE2's Gems.lua lists no additional effect for any Barbs gem; Data.lua's hard-coded append gives each
// exactly one triggered Barbs effect after the support.
func TestBarbsGemsGrantOneTriggeredEffect(t *testing.T) {
	want := map[string][2]string{
		"Barbs I":   {"SupportBarbsPlayer", "TriggeredBarbsPlayer"},
		"Barbs II":  {"SupportBarbsPlayerTwo", "TriggeredBarbsPlayerTwo"},
		"Barbs III": {"SupportBarbsPlayerThree", "TriggeredBarbsPlayerThree"},
	}
	s := &Skills{byID: map[string]*GrantedEffect{}}
	for _, ids := range want {
		for _, id := range ids {
			s.byID[id] = &GrantedEffect{ID: id}
		}
	}
	seen := 0
	for i := range poe2.Gems {
		gem := &poe2.Gems[i]
		ids, ok := want[gem.Name]
		if !ok {
			continue
		}
		seen++
		g := s.setupGem(gem)
		var got []string
		for _, ge := range g.GrantedEffectList {
			if ge == nil {
				got = append(got, "<nil>")
				continue
			}
			got = append(got, ge.ID)
		}
		if len(got) != 2 || got[0] != ids[0] || got[1] != ids[1] {
			t.Errorf("%s: grantedEffectList = %v, want %v", gem.Name, got, ids)
		}
		if len(g.AdditionalGrantedEffects) != 1 {
			t.Errorf("%s: %d additional granted effects, want 1", gem.Name, len(g.AdditionalGrantedEffects))
		}
	}
	if seen != len(want) {
		t.Fatalf("found %d Barbs gems, want %d", seen, len(want))
	}
}
