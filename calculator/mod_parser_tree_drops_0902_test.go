package calculator

import (
	"strings"
	"testing"
)

// TestTreeStatDropBatch0902 pins the parser rows added from the 2026-09-02 tree-stat drop census.
func TestTreeStatDropBatch0902(t *testing.T) {
	cases := []struct {
		line     string
		wantMods int
		wantName string
	}{
		{"10% more Attack speed with Offhand", 1, "Speed"},
		{"40% increased Accuracy Rating while you have at least 1 nearby Ally", 1, "Accuracy"},
		{"15% increased Area of Effect for Attacks while you have at least 1 nearby Ally", 1, "AreaOfEffect"},
		{"+0.4 metres to Melee Strike Range while at least 5 Enemies are Nearby", 2, "MeleeWeaponRangeMetre"},
		{"5% increased Attack Speed per Enemy in Close Range", 1, "Speed"},
		{"Deal 10% more Chaos Damage to enemies which have Energy Shield", 1, "ChaosDamage"},
		{"60% increased Critical Strike Chance against enemies with Lightning Exposure", 1, "CritChance"},
		{"+1% to Cold Damage over Time Multiplier for each 4% Overcapped Cold Resistance", 1, "ColdDotMultiplier"},
		{"25% increased total Power counted by Warcries", 1, "WarcryPower"},
		{"Gain Accuracy Rating equal to your Intelligence", 1, "Accuracy"},
		{"Poisons you inflict on non-Poisoned Enemies deal 300% increased Damage", 1, "Damage"},
		{"Warcries have a minimum of 10 Power", 1, "MinimumWarcryPower"},
		{"15% chance to gain 200 Life on Hit with Attacks", 2, "LifeOnHit"},
		{"3% chance for Hits to deal 300% of Physical Damage as Extra Damage of a random Element", 1, "PhysicalDamageGainAsRandom"},
		{"Hits against you Cannot be Critical Strikes if you've been Stunned Recently", 2, "EnemyModifier"},
		{"Arcane Surge also grants 15% increased Life Regeneration Rate to you", 1, "ArcaneSurgeAlsoLifeRegen"},
		{"Immune to Bleeding if Equipped Helmet has higher Armour than Evasion Rating", 1, "BleedImmune"},
		{"Immune to Poison if Equipped Helmet has higher Evasion Rating than Armour", 1, "PoisonImmune"},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e == nil {
				t.Fatalf("ParseMod returned nil")
			}
			if x := strings.TrimSpace(modExtra(e)); x != "" {
				t.Fatalf("leftover Extra %q -- processModLine discards the WHOLE line for this", x)
			}
			if len(e.ModList) != c.wantMods {
				names := make([]string, len(e.ModList))
				for i, m := range e.ModList {
					names[i] = m.Name()
				}
				t.Fatalf("got %d mods %v, want %d", len(e.ModList), names, c.wantMods)
			}
			found := false
			for _, m := range e.ModList {
				if m.Name() == c.wantName {
					found = true
				}
			}
			if !found {
				t.Fatalf("no mod named %s", c.wantName)
			}
		})
	}
}

// TestWarcryMinimumPowerRowsStayDistinct: The CryWolfMinimumPower row and the MinimumWarcryPower row differ by one article.
func TestWarcryMinimumPowerRowsStayDistinct(t *testing.T) {
	e := ParseMod("Warcries have minimum of 5 Power", false)
	if e == nil || len(e.ModList) != 1 || e.ModList[0].Name() != "CryWolfMinimumPower" {
		t.Fatalf("the articleless line no longer yields CryWolfMinimumPower: %+v", e)
	}
}
