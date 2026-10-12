package calculator

import (
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

var weaponLocalTestNames = map[string]bool{
	"Accuracy":                true,
	"ImpaleChance":            true,
	"LifeOnHit":               true,
	"ManaOnHit":               true,
	"PhysicalDamageLifeLeech": true,
	"PhysicalDamageManaLeech": true,
	"PoisonChance":            true,
	"BleedChance":             true,
}

// TestWeaponModsCarryHandCondition grades those mods against headless PoB's own per-item mod list.
func TestWeaponModsCarryHandCondition(t *testing.T) {
	cases, expected := loadItemModListOracle(t)

	checked := 0
	for idx, c := range cases {
		w := expected[idx]
		if !w.IsWeapon || !w.HasBase || w.Error != nil {
			continue
		}

		wantKeys := make([]string, 0, len(w.Mods))
		for _, m := range w.Mods {
			if weaponLocalTestNames[m.Name] {
				wantKeys = append(wantKeys, modKey(m.Name, m.Type, m.Value, m.Flags, m.KeywordFlags, m.Tags))
			}
		}
		if len(wantKeys) == 0 {
			continue
		}
		checked++

		t.Run(c.Name, func(t *testing.T) {
			item := ParseItem(c.Raw)
			if itemHasUnparsedLine(item) {
				t.Skip("blocked on ModParser coverage")
			}

			gotKeys := make([]string, 0, len(item.Mods))
			for _, m := range item.Mods {
				if weaponLocalTestNames[m.Name()] {
					gotKeys = append(gotKeys, goModKey(m))
				}
			}

			sort.Strings(wantKeys)
			sort.Strings(gotKeys)
			testza.AssertEqual(t, wantKeys, gotKeys, "weapon-local mods for %q", c.Name)
		})
	}

	testza.AssertGreater(t, checked, 0, "no oracle weapon carries a weapon-local mod")
}

func handConditionVars(m mod.Mod) []string {
	var out []string
	for _, tag := range m.Tags() {
		if c, ok := tag.(*mod.ConditionTag); ok {
			out = append(out, c.VarList...)
		}
	}
	return out
}

const weaponLocalItem = `Rarity: RARE
Test Blade
Vaal Blade
Quality: 20
Physical Damage: 60-100
+400 to Accuracy Rating
+5 Life gained for each Enemy hit by your Attacks
+3 Mana gained for each Enemy hit by your Attacks
20% chance to Poison on Hit
Melee Critical Strikes cause Bleeding
`

// TestWeaponHandConditionNamesTheSlot asserts the part the type-only oracle cannot see.
func TestWeaponHandConditionNamesTheSlot(t *testing.T) {
	item := ParseItem(weaponLocalItem)
	testza.AssertFalse(t, itemHasUnparsedLine(item), "fixture blocked on ModParser coverage")

	for _, tc := range []struct {
		hand string
		mods []mod.Mod
	}{
		{"MainHandAttack", item.Mods},
		{"OffHandAttack", item.slotModList2},
	} {
		want := map[string][]string{
			"Accuracy":     {tc.hand},
			"LifeOnHit":    nil,
			"ManaOnHit":    nil,
			"PoisonChance": {tc.hand},
			"BleedChance":  {"CriticalStrike", tc.hand},
		}

		seen := map[string]bool{}
		for _, m := range tc.mods {
			vars, ok := want[m.Name()]
			if !ok {
				testza.AssertNotContains(t, handConditionVars(m), tc.hand,
					"%s tagged on non-weapon-local mod %s", tc.hand, m.Name())
				continue
			}
			seen[m.Name()] = true
			testza.AssertEqual(t, vars, handConditionVars(m), "%s: condition vars on %s", tc.hand, m.Name())
		}
		for name := range want {
			testza.AssertTrue(t, seen[name], "%s: %s missing from the slot list", tc.hand, name)
		}
	}
}

// TestWeaponHandConditionDoesNotMutateSharedMods pins the copyTable() at Item.lua:987 that makes the injection safe: PoB tags a per-slot copy.
func TestWeaponHandConditionDoesNotMutateSharedMods(t *testing.T) {
	item := ParseItem(weaponLocalItem)

	for _, m := range item.Mods {
		if !weaponLocalTestNames[m.Name()] || len(handConditionVars(m)) == 0 {
			continue
		}
		for _, other := range item.slotModList2 {
			if m == other {
				t.Errorf("slot lists share the %s mod object", m.Name())
			}
		}
	}
}
