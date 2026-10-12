package calculator

import (
	"strings"
	"testing"
)

// TestIncomingHitBlockMatchesThePin pins CalcDefence.lua:1961 and :1976-1984.
func TestIncomingHitBlockMatchesThePin(t *testing.T) {
	t.Run("the four skipped damage-taken rows parse", func(t *testing.T) {
		requireGameData(t, "3.29")

		for text, want := range map[string]string{
			"-150 Physical Damage taken from Projectile Attacks":     "PhysicalDamageTakenFromProjectileAttacks",
			"-150 Physical Damage taken from Projectile Attack Hits": "PhysicalDamageTakenFromProjectileAttacks",
			"-25 Damage taken of each Damage Type from Spell Hits":   "DamageTakenFromSpells",
			"Cold Resistance cannot be penetrated":                   "EnemyCannotPenColdResistance",
		} {
			e := ParseMod(text, false)
			if e == nil || strings.TrimSpace(modExtra(e)) != "" {
				t.Errorf("%q does not parse cleanly (leftover %q)", text, modExtraOf(e))
				continue
			}
			var got []string
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			if !contains(got, want) {
				t.Errorf("%q produced %v, want %s", text, got, want)
			}
		}

		e := ParseMod("-30 Fire, Cold and Lightning Damage taken from Spell Hits", false)
		if e == nil {
			t.Fatal("REFUSING: the three-element row does not parse at all")
		}
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		for _, want := range []string{"FireDamageTakenFromSpells", "ColdDamageTakenFromSpells", "LightningDamageTakenFromSpells"} {
			if !contains(got, want) {
				t.Errorf("the three-element row produced %v, missing %s", got, want)
			}
		}
	})

	t.Run("a projectile-attack reduction reaches the hit taken", func(t *testing.T) {
		assertBuildStats(t, "1294-gladiator-lacerate-of-haemorrhage-6.xml",
			[]string{"PhysicalTakenHit", "PhysicalTakenHitMult", "totalTakenHit"},
			func(t *testing.T, file string) {
				if !strings.Contains(file, "Physical Damage taken from Projectile Attacks") {
					t.Fatalf("REFUSING: the fixture no longer carries the projectile-attack " +
						"reduction, so it cannot exercise the arm under test")
				}
			})
	})
}

func modExtraOf(e *ModCacheEntry) string {
	if e == nil {
		return "<nil>"
	}
	return modExtra(e)
}
