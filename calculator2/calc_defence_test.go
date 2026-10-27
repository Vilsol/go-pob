package calculator2

import (
	"math"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's calcsTab.mainOutput for the same file, loaded headless with
// loadBuildFromXML. It is the tree-only warrior with custom lines that reach every defence-slice stat
// without an item.
func TestDefenceCustomModsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/defence_custom_mods.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Str": 40, "Dex": 17, "Int": 12, "Life": 547, "Mana": 174, "EnergyShield": 78, "Spirit": 70,
		"FireResist": 5, "ColdResist": 78, "LightningResist": -35, "ChaosResist": 17,
		"Armour": 150, "Evasion": 161, "BlockChance": 13, "DeflectChance": 2, "MovementSpeedMod": 1.15,
		"LifeUnreserved": 410, "ManaUnreserved": 174, "ActionSpeedMod": 1, "EvadeChance": 2,
		"DeflectionRating": 182.2, "FireResistTotal": 5, "ChaosResistTotal": 17, "SpellBlockChance": 0,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}

// The expected values are PoB-PoE2's output for the same file, loaded headless: the custom-mods
// warrior above plus recovery, recharge, recoup, avoidance and ailment-on-self lines, and mana regeneration rate applying to recharge rate.
func TestDefenceRecoveryFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/defence_recovery_mods.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"AilmentThreshold": 301.4, "BleedAvoidChance": 25, "ShockAvoidChance": 30, "CurseEffectOnSelf": 85,
		"DebuffExpirationModifier": 100, "EnergyShieldRecharge": 16.8, "EnergyShieldRechargeDelay": 2.8571428571429,
		"EnergyShieldRecoveryRateMod": 1.15, "EnergyShieldRegenInc": 15, "ExposureEffectOnSelf": 100,
		"LifeOnBlock": 25, "LifeRecoup": 16.5, "LifeRecoveryRateMod": 1.1, "LifeRegen": 27.6, "LifeRegenInc": 10,
		"LifeRegenPercent": 5, "LifeRegenRecovery": 27.6, "LightRadiusMod": 1.1, "ManaRecoup": 8,
		"ManaRecoveryRateMod": 1, "ManaRegen": 8.4, "ManaRegenInc": 20, "ManaRegenPercent": 4.8, "ManaRegenRecovery": 8.4,
		"MaxEnergyShieldLeechInstance": 7.8, "MaxEnergyShieldLeechRate": 7.8, "MaxLifeLeechInstance": 54.7,
		"MaxLifeLeechRate": 109.4, "MaxLifeLeechRatePercent": 20, "MaxManaLeechInstance": 17.4, "MaxManaLeechRate": 34.8,
		"SelfBleedDuration": 100, "SelfBleedEffect": 100, "SelfBlindDuration": 100, "SelfChillDuration": 100,
		"SelfChillEffect": 60, "SelfFreezeDuration": 80, "SelfIgniteDuration": 65, "SelfShockEffect": 100,
		"SpellSuppressionEffect": 50, "WeaponSwapSpeedMod": 1, "WitherEffectOnSelf": 100, "anyRecoup": 24.5,
		"SpellSuppressionChance": 0, "AttackDodgeChance": 0, "WardRegen": 0, "RageRegen": 0,
	} {
		got, ok := env.Player.Output[stat]
		if !ok {
			t.Errorf("%s: not written", stat)
			continue
		}
		if math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}

// The expected values are PoB-PoE2's output for the same file, loaded headless: two enabled spirit
// skills under reservation efficiency, a disabled one that reserves nothing, and a life reservation.
func TestDefenceSkillReservationFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/skill_reservation.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Spirit": 350, "SpiritReserved": 54, "SpiritReservedPercent": 15.428571428571, "SpiritUnreserved": 296,
		"SpiritUnreservedPercent": 84.571428571429, "SpiritCancellableReservation": 100,
		"Life": 825, "LifeReserved": 83, "LifeUnreserved": 742, "LifeCancellableReservation": 90,
		"Mana": 298, "ManaReserved": 0, "ManaUnreserved": 298,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}
