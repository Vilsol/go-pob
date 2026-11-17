package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestImbuedSupportResolvesLikeFindImbuedSupport: SkillsTab.lua:1104-1110 accepts a saved base
// name or an import name that already ends in "Support", and only a support gem.
func TestImbuedSupportResolvesLikeFindImbuedSupport(t *testing.T) {
	for name, want := range map[string]bool{
		"Added Fire Damage":         true,
		"Added Fire Damage Support": true,
		"Fireball":                  false,
	} {
		got := buildImbuedSupportBySlot([]pob.Skill{{Slot: "Helmet", ImbuedSupport: &name}})
		if (len(got["Helmet"]) > 0) != want {
			t.Errorf("%q resolved: %v, want %v", name, len(got["Helmet"]) > 0, want)
		}
	}
}
