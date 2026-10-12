package calculator

import (
	"os"
	"testing"
)

// TestSpellbladeAddsWeaponDamageToSpells covers CalcOffence.lua:562-579.
func TestSpellbladeAddsWeaponDamageToSpells(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0132-elementalist-penance-brand-7.xml",
		"0216-elementalist-hexblast-of-contradiction-2.xml",
		"0237-elementalist-hexblast-of-contradiction-3.xml",
		"0557-elementalist-flame-wall.xml",
		"1038-inquisitor-divine-ire-of-holy-lightning.xml",
		"1179-gladiator-bladefall-of-volleys.xml",
	}

	var seen, agreed int
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, build)
		for _, damageType := range []string{"Physical", "Fire", "Cold", "Lightning", "Chaos"} {
			for _, suffix := range []string{"MinBase", "MaxBase"} {
				key := damageType + suffix
				want, has := rec.Output[key]
				if !has || want == 0 {
					continue
				}
				seen++
				if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
					agreed++
				}
			}
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded Spellblade fixture reports a nonzero base damage rung; this " +
			"test no longer measures the mechanism it was written for")
	}
	t.Logf("Spellblade: %d of %d base damage rungs agree with PoB", agreed, seen)

	const agreementFloor = 44
	if agreed < agreementFloor {
		t.Errorf("Spellblade agrees on %d of %d base damage rungs, below the floor of %d measured when the "+
			"port landed (CalcOffence.lua:562-579)", agreed, seen, agreementFloor)
	}
}
