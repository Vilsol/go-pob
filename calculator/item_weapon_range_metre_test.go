package calculator

import (
	"strings"
	"testing"
)

// TestWeaponRangeMetreScalesByTen covers Item.lua:2248's `10 * calcLocal(modList, "WeaponRangeMetre", "BASE", 0)` term, which this port dropped entirely.
func TestWeaponRangeMetreScalesByTen(t *testing.T) {
	requireGameData(t, "3.29")

	const raw = "Rarity: RARE\nProbe Blade\nSiege Axe\n" +
		"Unique ID: 0000\nItem Level: 84\nQuality: 0\nLevelReq: 70\nImplicits: 0\n" +
		"+1 metres to Weapon Range\n"

	item := ParseItem(raw)
	if item.Base == nil || item.Base.Weapon == nil {
		t.Fatalf("REFUSING: fixture resolved no weapon base (baseName %q) -- the test measures nothing", item.BaseName)
	}

	plain := ParseItem(strings.Replace(raw, "+1 metres to Weapon Range\n", "", 1))
	if got, want := plain.WeaponData[1]["range"], float64(plain.Base.Weapon.Range); got != want {
		t.Fatalf("control: unmodified range %v, want the base %v", got, want)
	}

	want := float64(item.Base.Weapon.Range) + 10
	if got := item.WeaponData[1]["range"]; got != want {
		t.Errorf("+1 metres to Weapon Range gave range %v, want %v (Item.lua:2248)", got, want)
	}
}
