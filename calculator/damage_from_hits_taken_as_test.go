package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestDamageFromHitsTakenAsIsDistinct covers ModParser.lua:298-341 and CalcDefence.lua:1814-1816.
func TestDamageFromHitsTakenAsIsDistinct(t *testing.T) {
	for _, tc := range []struct {
		line string
		want string
	}{
		{"10% of Physical Damage from Hits taken as Chaos Damage", "PhysicalDamageFromHitsTakenAsChaos"},
		{"25% of Physical Damage from Hits taken as Fire Damage", "PhysicalDamageFromHitsTakenAsFire"},
		{"8% of Elemental Damage from Hits taken as Chaos Damage", "ElementalDamageFromHitsTakenAsChaos"},
		{"15% of Cold Damage from Hits taken as Physical Damage", "ColdDamageFromHitsTakenAsPhysical"},
		{"20% of Physical Damage taken as Fire Damage", "PhysicalDamageTakenAsFire"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra, "%q must parse completely", tc.line)
			testza.AssertEqual(t, 1, len(mods), "%q must produce exactly one mod", tc.line)
			if len(mods) != 1 {
				return
			}
			testza.AssertEqual(t, tc.want, mods[0].Name(), "%q", tc.line)
		})
	}
}

// TestDamageFromHitsTakenAsReachesTheHitTableOnly is the ACTIVATION rung.
func TestDamageFromHitsTakenAsReachesTheHitTableOnly(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, build := range []string{
		"0028-occultist-forbidden-rite-of-soul-sacrifice.xml",
		"0035-elementalist-shield-charge.xml",
	} {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if env.Player.DamageShiftTable == nil || env.Player.DamageOverTimeShiftTable == nil {
				t.Fatal("REFUSING: a shift table is nil; the producer moved")
			}
			var differs int
			for from, row := range env.Player.DamageShiftTable {
				for to, hitPct := range row {
					if from == to {
						continue
					}
					if dotPct := env.Player.DamageOverTimeShiftTable[from][to]; dotPct != hitPct {
						differs++
					}
				}
			}
			if differs == 0 {
				t.Errorf("the hit and damage-over-time shift tables are identical; the " +
					"DamageFromHitsTakenAs family is not reaching the hit table alone " +
					"(CalcDefence.lua:1814-1816)")
			}
		})
	}
}
