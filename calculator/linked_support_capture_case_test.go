package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestLinkedSupportMainHandIsCaseInsensitive: PoB's scan() matches against `lineLower` and returns the captures taken from THAT string.
func TestLinkedSupportMainHandIsCaseInsensitive(t *testing.T) {
	mods, extra := parseMod("Socketed Support Gems can also Support Skills from your Main Hand", 0)
	testza.AssertEqual(t, "", extra)
	testza.AssertLen(t, mods, 1)

	linked, ok := mods[0].Value().List().(mod.LinkedSupport)
	testza.AssertTrue(t, ok, "%s is not a LinkedSupport LIST mod", mods[0].Name())
	testza.AssertEqual(t, "Weapon 1", linked.TargetSlotName)
}
