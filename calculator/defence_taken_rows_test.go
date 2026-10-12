package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDefensiveTakenRowsParse pins parser rows in the "damage taken" / "defend with armour" families that the pin carries and this tree did not.
func TestDefensiveTakenRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		count int
		first string
		typ   mod.Type
		value float64
	}{
		{"35% less Damage taken while there are at least two Rare or Unique Enemies nearby", 1, "DamageTaken", mod.TypeMore, -35},
		{"1% less Damage taken per 5 Rage, up to a maximum of 30%", 1, "DamageTaken", mod.TypeMore, -1},
		{"100% chance to Defend with 120% of Armour for each time you've been Hit by an Enemy Recently, up to 40%", 3, "ArmourDefense", mod.TypeMAX, 20},
		{"50% chance to Defend with double your Armour for each time you've been Hit by an Enemy Recently, up to 40%", 3, "ArmourDefense", mod.TypeMAX, 100},
		{"Modifiers to Chance to Suppress Spell Damage also apply to Chance to Defend with 130% of Armour at 50% of their value", 3, "SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour", mod.TypeMAX, 130},
		{"25% of Armour also applies to Chaos Damage taken from Hits", 1, "ArmourAppliesToChaosDamageTaken", mod.TypeBase, 25},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("ParseMod(%q) returned nothing", tc.line)
			}
			if e.Extra != "" {
				t.Fatalf("%q left %q unparsed — processModLine discards the whole line", tc.line, e.Extra)
			}
			if len(e.ModList) != tc.count {
				names := make([]string, 0, len(e.ModList))
				for _, m := range e.ModList {
					names = append(names, m.Name())
				}
				t.Fatalf("%q produced %d mods %v, want %d", tc.line, len(e.ModList), names, tc.count)
			}
			m := e.ModList[0]
			if m.Name() != tc.first {
				t.Fatalf("first mod name = %q, want %q", m.Name(), tc.first)
			}
			if m.Type() != tc.typ {
				t.Errorf("first mod type = %q, want %q", m.Type(), tc.typ)
			}
			if v := m.Value(); v.Type() != mod.ModValueMultiTypeFloat {
				t.Errorf("first mod value type = %q, want float", v.Type())
			} else if v.Float() != tc.value {
				t.Errorf("first mod value = %v, want %v", v.Float(), tc.value)
			}
		})
	}
}
