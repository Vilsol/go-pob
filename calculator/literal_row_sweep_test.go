package calculator

import "testing"

// TestLiteralKeySweepRowsParse pins the second tranche of the literal-key sweep against the pin's specialModList.
func TestLiteralKeySweepRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		count int
		first string
	}{
		{"Physical Damage taken bypasses Energy Shield", 1, "PhysicalEnergyShieldBypass"},
		{"Physical Damage Reduction is Zero", 2, "PhysicalDamageReduction"},
		{"Armour also applies to Chaos Damage taken from Hits", 1, "ArmourAppliesToChaosDamageTaken"},
		{"Immune to Reflected Damage", 1, "ReflectedDamageTaken"},
		{"All Elemental Damage Converted to Chaos Damage", 3, "ColdDamageConvertToChaos"},
		{"You have Consecrated Ground around you while stationary if Strength is your highest Attribute", 1, "Condition:OnConsecratedGround"},
		{"Projectiles cannot continue after colliding with targets", 4, "CannotPierce"},
		{"Bow Attacks have Culling Strike", 1, "CullPercent"},
		{"Nearby Allies have Culling Strike", 1, "ExtraAura"},
		{"Nearby Allies' Action Speed cannot be modified to below base value", 1, "ExtraAura"},
		{"Spell Skills deal no Damage", 1, "DealNoDamage"},
		{"Enemies near Corpses affected by your Curses are Blinded", 1, "EnemyModifier"},
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
			if got := e.ModList[0].Name(); got != tc.first {
				t.Errorf("first mod name = %q, want %q", got, tc.first)
			}
		})
	}

	for _, line := range []string{
		"Armour also applies to Lightning Damage taken from Hits",
		"Elemental Resistances are Zero",
		"Chaos Resistance is Zero",
		"Culling Strike against Burning Enemies",
		"Your Spells have Culling Strike",
	} {
		if e := ParseMod(line, false); e == nil || len(e.ModList) == 0 || e.Extra != "" {
			t.Errorf("control %q: ParseMod produced %v, want mods and no Extra", line, e)
		}
	}
}
