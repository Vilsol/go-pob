package calculator

import (
	"math"
	"testing"
)

func assertPlayerOutputs(t *testing.T, build string, want map[string]float64) {
	t.Helper()
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, build)
	for k, w := range want {
		if got := env.Player.Output[k]; math.Abs(got-w) > 1e-9*math.Abs(w) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, w)
		}
	}
}

// TestSeismicTrapAltXPreDamage: Seismic Trap of Swells' overlap chance and wave multiplier (act_dex.lua:13627-13667).
func TestSeismicTrapAltXPreDamage(t *testing.T) {
	assertPlayerOutputs(t, "1634-pathfinder-seismic-trap-of-swells.xml",
		map[string]float64{"OverlapChance": 37.345679012346, "SkillDPSMultiplier": 2.6141975308642})
}

// TestFrostBombAltYDurationMultiplier: Frost Bomb of Forthcoming's per-100ms-duration multiplier (act_int.lua:7933-7936).
func TestFrostBombAltYDurationMultiplier(t *testing.T) {
	assertPlayerOutputs(t, "15142-ladder-xdxdgogoxd_9043.мама.xml",
		map[string]float64{"TotalMin": 264007.8336, "TotalMax": 395872.99584})
}

// TestLightningSpireTrapAltYPreDamage: Lightning Spire Trap of Overloading's unscaled targeting radius and active-trap count (act_int.lua:12150-12260).
func TestLightningSpireTrapAltYPreDamage(t *testing.T) {
	assertPlayerOutputs(t, "7312-ladder-happy_1234_5190.天津四.xml", //nolint:gosmopolitan // corpus build name
		map[string]float64{"AverageActiveTraps": 2.2446689113356, "OverlapChance": 76.5625, "AreaOfEffectRadius": 24, "PulsesPerTrap": 8})
}

// TestSlotSupportListsCarryExtraSupports: an item-granted skill takes every support group in its slot.
func TestSlotSupportListsCarryExtraSupports(t *testing.T) {
	assertPlayerOutputs(t, "0786-saboteur-unknown.xml",
		map[string]float64{"AverageHit": 741997.69344, "TotalDPS": 5895616.8729969})
}

// TestCritOverrideIs100NotBaseCrit: only a CritChance override of 100 skips the accuracy scaling; a 100 base crit does not (CalcOffence.lua:3006-3058).
func TestCritOverrideIs100NotBaseCrit(t *testing.T) {
	assertPlayerOutputs(t, "1647-luminary-leap-slam.xml", map[string]float64{"CritChance": 5})
}

// TestSpellDamageAppliesToAttacksSkillScoped: a SkillType-tagged SpellDamageAppliesToAttacks flag is read with skillCfg (CalcOffence.lua:618).
func TestSpellDamageAppliesToAttacksSkillScoped(t *testing.T) {
	assertPlayerOutputs(t, "6984-ladder-chelovek_ryzhiy_0672.incarnationofpenis.xml",
		map[string]float64{"AverageDamage": 184013715.55043, "TotalDPS": 62653631.443796})
}

// TestSkillNameListCarriesTransfiguredGems: the skill-name loop walks every gem, transfigured ones included (ModParser.lua:6042-6050).
func TestSkillNameListCarriesTransfiguredGems(t *testing.T) {
	assertPlayerOutputs(t, "0213-occultist-divine-ire-of-disintegration.xml",
		map[string]float64{"AverageHit": 12277644.827797, "TotalDPS": 9963780.9948664})
}

// TestInflictExposureOnHitRows: the bare "inflict <element> exposure on hit" rows (ModParser.lua:4205-4213).
func TestInflictExposureOnHitRows(t *testing.T) {
	assertPlayerOutputs(t, "0615-luminary-lightning-tendrils-of-escalation.xml",
		map[string]float64{"AverageHit": 16984684.133298, "TotalDPS": 148899064.23525})
}

// TestItemTriggeredGrantedSkillIsTriggered: an item's triggered ExtraSkill marks its gem instance triggered.
func TestItemTriggeredGrantedSkillIsTriggered(t *testing.T) {
	assertPlayerOutputs(t, "4421-ladder-DEM0ON_5479.Zer๏F๏๏๏๏๏๏Mini๏nPlayer.xml", map[string]float64{"ManaCost": 0})
}

// TestMoonbendersWingTriggerConfig: Moonbender's Wing forces a 1s cooldown on its Lightning Warp (CalcTriggers.lua:968-972).
func TestMoonbendersWingTriggerConfig(t *testing.T) {
	assertPlayerOutputs(t, "4421-ladder-DEM0ON_5479.Zer๏F๏๏๏๏๏๏Mini๏nPlayer.xml",
		map[string]float64{"Speed": 0.97751710654936, "Cooldown": 1.023, "TotalDPS": 175.94574780059})
}

// TestFoulbornUniqueItemThreshold: a {mutated} explicit line makes a unique foulborn.
func TestFoulbornUniqueItemThreshold(t *testing.T) {
	assertPlayerOutputs(t, "6151-ladder-SoloT1Witch_1235.AllFlameOfAT.xml",
		map[string]float64{"AverageHit": 1073521.1653519, "TotalDPS": 16031249.402589})
}

// TestPenanceBrandOfDissipationCachedStages: the max-pulse stage count reads the brand's cached HitSpeed and Duration (CalcPerform.lua:2235-2240).
func TestPenanceBrandOfDissipationCachedStages(t *testing.T) {
	assertPlayerOutputs(t, "0408-inquisitor-penance-brand-of-dissipation.xml",
		map[string]float64{"AverageHit": 26137873.501276, "AreaOfEffectRadius": 57})
}

// TestItemExtraSupportIsFromItem: an item ExtraSupport's granted effect is flagged fromItem.
func TestItemExtraSupportIsFromItem(t *testing.T) {
	assertPlayerOutputs(t, "0751-deadeye-ghostly-artillery.xml",
		map[string]float64{"IgniteDPS": 3307.889948136, "CombinedDPS": 368514.12765605})
}
