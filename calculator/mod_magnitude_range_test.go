package calculator

import "testing"

const stasisPrisonRaw = `Rarity: UNIQUE
Stasis Prison
Carnal Armour
Implicits: 3
{enchant}{crafted}Quality does not increase Defences
{enchant}{crafted}Grants +1% to Fire Resistance per 2% Quality
{tags:increased}20% of Physical Damage taken as Fire Damage
{unscalable}{range:1}(80-100)% increased Implicit Modifier magnitudes
{tags:defences,evasion,energyshield}{range:0.2}(140-160)% increased Evasion and Energy Shield`

// TestModMagnitudeReadsTheLinesOwnRange pins Item.lua:1251, which re-ranges a magnitude line at `modLine.range or main.defaultItemAffixQuality or 1`.
func TestModMagnitudeReadsTheLinesOwnRange(t *testing.T) {
	requireGameData(t, "3.29")

	takenAsFire := func(raw string) float64 {
		item := &Item{}
		item.ParseRaw(raw)
		total := 0.0
		for _, ml := range item.ImplicitModLines {
			for _, m := range ml.ModList {
				if m.Name() == "PhysicalDamageTakenAsFire" {
					total += m.Value().Float()
				}
			}
		}
		return total
	}

	if got := takenAsFire(stasisPrisonRaw); got != 40 {
		t.Errorf("{range:1} magnitude: PhysicalDamageTakenAsFire = %v, want 40 (20 x (1 + 100%%), Item.lua:1251)", got)
	}

	control := replaceOnce(stasisPrisonRaw, "{range:1}(80-100)%", "{range:0.5}(80-100)%")
	if got := takenAsFire(control); got != 38 {
		t.Errorf("CONTROL {range:0.5}: PhysicalDamageTakenAsFire = %v, want 38 (20 x 1.9)", got)
	}

	noRange := replaceOnce(stasisPrisonRaw, "{unscalable}{range:1}(80-100)%", "{unscalable}(80-100)%")
	if got := takenAsFire(noRange); got != 38 {
		t.Errorf("CONTROL no {range:}: PhysicalDamageTakenAsFire = %v, want 38 (default quality 0.5 -> 20 x 1.9)", got)
	}
}

func replaceOnce(s, old, repl string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + repl + s[i+len(old):]
		}
	}
	panic("replaceOnce: " + old + " not found")
}
