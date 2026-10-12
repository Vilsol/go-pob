package calculator

import (
	"testing"
)

// TestVileBastionEnergyShieldFromSpellBlock pins CalcDefence.lua:812-817 and ModParser.lua:4805.
func TestVileBastionEnergyShieldFromSpellBlock(t *testing.T) {
	t.Run("the parser row exists", func(t *testing.T) {
		requireGameData(t, "3.29")
		e := ParseMod("Maximum Energy Shield is increased by Chance to Block Spell Damage", false)
		if e == nil {
			t.Fatal("REFUSING: the line does not parse at all")
		}
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, "EnergyShieldIncreasedByChanceToBlockSpellDamage") {
			t.Errorf("produced %v, want the EnergyShieldIncreasedByChanceToBlockSpellDamage flag", got)
		}
	})

	t.Run("the keystone reaches the pool", func(t *testing.T) {
		assertBuildStats(t, "0028-occultist-forbidden-rite-of-soul-sacrifice.xml",
			[]string{"EnergyShield"}, nil)
	})
}
