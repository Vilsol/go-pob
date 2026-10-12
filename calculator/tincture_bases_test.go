package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"
)

// TestTinctureBasesResolve pins the ten tincture bases PoB exports, and pins the eight legacy rows OUT.
func TestTinctureBasesResolve(t *testing.T) {
	want := map[string]bool{
		"Prismatic Tincture":   true,
		"Rosethorn Tincture":   true,
		"Ironwood Tincture":    true,
		"Ashbark Tincture":     true,
		"Borealwood Tincture":  true,
		"Fulgurite Tincture":   true,
		"Poisonberry Tincture": true,
		"Blood Sap Tincture":   true,
		"Oakbranch Tincture":   true,
		"Sporebloom Tincture":  true,
	}

	for name := range want {
		base := ItemBaseByName(name)
		testza.AssertNotNil(t, base, "no item base assembled for %q", name)
		if base == nil {
			continue
		}
		testza.AssertEqual(t, "Tincture", base.Type, "type of %q", name)
		testza.AssertNotNil(t, base.Tincture, "base.tincture presence on %q", name)
		testza.AssertEqual(t, 0, base.Req.Level,
			"%q carries a req.level PoB does not export", name)
	}

	got := 0
	for _, base := range ItemBases() {
		if base.Type == "Tincture" {
			got++
		}
	}
	testza.AssertEqual(t, len(want), got, "tincture bases in the index")

	classByKey := make(map[int]*poe.ItemClass, len(poe.ItemClasses))
	for _, c := range poe.ItemClasses {
		classByKey[c.Key] = c
	}

	kept, rejected := map[string]bool{}, map[string]bool{}
	for _, b := range poe.BaseItemTypes {
		c := classByKey[b.ItemClassesKey]
		if c == nil || itemClassTypes[c.Name] != "Tincture" {
			continue
		}
		if tinctureBaseIDRe.MatchString(b.ID) {
			kept[b.Name] = true
		} else {
			rejected[b.ID] = true
		}
	}

	testza.AssertEqual(t, want, kept, "ids kept by tinctureBaseIDRe, by display name")
	testza.AssertEqual(t, map[string]bool{
		"Metadata/Items/Tinctures/TinctureStun":           true,
		"Metadata/Items/Tinctures/TinctureIgnite":         true,
		"Metadata/Items/Tinctures/TinctureShock":          true,
		"Metadata/Items/Tinctures/TincturePoison":         true,
		"Metadata/Items/Tinctures/TinctureCriticalStrike": true,
		"Metadata/Items/Tinctures/TinctureCullingStrike":  true,
		"Metadata/Items/Tinctures/TinctureFreeze":         true,
		"Metadata/Items/Tinctures/TinctureBleed":          true,
	}, rejected, "legacy ids rejected by tinctureBaseIDRe")
}

// TestTinctureBaseOfMatchesPoB grades the tincture tables derived from go-pob-data against the pin's Data/Bases/tincture.lua.
func TestTinctureBaseOfMatchesPoB(t *testing.T) {
	want := map[string]tinctureBase{
		"Prismatic Tincture":   {ManaBurn: 0.7, Cooldown: 8},
		"Rosethorn Tincture":   {ManaBurn: 0.7, Cooldown: 8},
		"Ironwood Tincture":    {ManaBurn: 0.9, Cooldown: 10},
		"Ashbark Tincture":     {ManaBurn: 0.6, Cooldown: 7},
		"Borealwood Tincture":  {ManaBurn: 0.6, Cooldown: 7},
		"Fulgurite Tincture":   {ManaBurn: 0.6, Cooldown: 7},
		"Poisonberry Tincture": {ManaBurn: 0.7, Cooldown: 6},
		"Blood Sap Tincture":   {ManaBurn: 0.7, Cooldown: 6},
		"Oakbranch Tincture":   {ManaBurn: 0.5, Cooldown: 8},
		"Sporebloom Tincture":  {ManaBurn: 0.9, Cooldown: 8},
	}
	for name, w := range want {
		testza.AssertEqual(t, w, tinctureBaseOf(name), "tincture table of %q", name)
	}
}
