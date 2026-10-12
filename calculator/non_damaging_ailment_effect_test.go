package calculator

import "testing"

// TestNonDamagingElementalAilmentMoreEffect pins ModParser.lua:3707-3714 (Fury of Nature): "Non-Damaging Elemental Ailments you inflict have 100% more Effect".
func TestNonDamagingElementalAilmentMoreEffect(t *testing.T) {
	for _, name := range []string{"15378-warden-elemental-hit-of-the-spectrum.xml", "1403-pathfinder-eye-of-winter.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"ChillEffectMod", "ShockEffectMod"}, nil)
		})
	}
}
