package calculator

import "testing"

// TestTreeNestedModCarriesTreeSource pins ModTools.lua:217-223 as PassiveSpec.lua:2375 uses it.
func TestTreeNestedModCarriesTreeSource(t *testing.T) {
	for _, name := range []string{"15383-ladder-ArmaMalum_4722.ImHereAsWell.xml", "11745-ladder-Roige_4445.DaddyDommy.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"Spec:LifeInc"}, nil)
		})
	}
}
