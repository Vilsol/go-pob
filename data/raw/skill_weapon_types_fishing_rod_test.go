package raw

import (
	"slices"
	"testing"
)

// TestSkillWeaponTypesMapFishingRod: The skills export maps the FishingRod item class.
func TestSkillWeaponTypesMapFishingRod(t *testing.T) {
	if got := SkillWeaponTypes["ChainHookAltX"]; !slices.Equal(got, []string{"Fishing Rod"}) {
		t.Errorf("ChainHookAltX weapon types %q, want the pin's [Fishing Rod]", got)
	}
}
