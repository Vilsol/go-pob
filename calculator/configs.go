package calculator

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type ConfigApplyFunc func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList)

func configFloat(val interface{}) float64 {
	if f, ok := val.(float64); ok {
		return f
	}
	return 0
}

func configString(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

//pob:block:src/Classes/ConfigTab.lua:1084-1087
func configChecked(val interface{}) bool {
	if b, ok := val.(bool); ok {
		return b
	}
	return val != nil
}

//pob:block:src/Classes/ConfigTab.lua:1089-1090
func configCount(val interface{}) (float64, bool) {
	f := configFloat(val)
	return f, f != 0
}

var configDefaults = []string{
	"targetBrandedEnemy",
	"ConcPathBypassCD",
	"FlickerStrikeBypassCD",
	"VigilantStrikeBypassCD",
	"conditionChampionIntimidate",
	"conditionSacrificeMinion",
	"raiseSpectreEnableBuffs",
	"raiseSpectreEnableCurses",
	"summonElementalRelicEnableAngerAura",
	"summonElementalRelicEnableHatredAura",
	"summonElementalRelicEnableWrathAura",
}

//pob:block:src/Classes/ConfigTab.lua:1233-1240
var configCountDefaults = map[string]float64{
	"bloodsoakedBannerStages":      1,
	"conditionCorruptingCryStages": 1,
	"touchedDebuffsCount":          10,
	"maniaDebuffsCount":            15,
}

func configDefaultState(name string, val interface{}) (float64, bool) {
	if b, ok := val.(bool); ok {
		if !b {
			return 0, false
		}
		v, ok := configCountDefaults[name]
		return v, ok
	}
	return configFloat(val), true
}

//pob:block:src/Modules/ConfigOptions.lua:2060
var configIndexDefaults = map[string]string{
	"resourceGainMode":          "AVERAGE",
	"plagueBearerState":         "INC",
	"enemySizePreset":           "Medium",
	"enemyIsBoss":               "Pinnacle",
	"repeatMode":                "AVERAGE",
	"doomBlastSource":           "vixen",
	"GamblesprintMovementSpeed": "30",
}

var goPobOnlyConfigOptions = []string{
	"bannerStages",
	"bonechillEffect",
	"conditionAgainstDamageOverTime",
	"conditionSelfChill",
	"critChanceLucky",
	"enemyCanAvoidPoisonBlindBleed",
	"enemyCanAvoidStatusAilment",
	"enemyHasIncreasedAccuracy",
	"enemyHasLessCurseEffectOnSelf",
	"enemyHasPhysicalReduction",
	"enemyHasResistances",
	"enemyIsHexproof",
	"enemyTakesReducedExtraCritDamage",
	"energyBladeActive",
	"frostboltExposure",
	"playerCannotLeech",
	"playerCannotRegenLifeManaEnergyShield",
	"playerGainsReducedFlaskCharges",
	"playerHasElementalEquilibrium",
	"playerHasLessAreaOfEffect",
	"playerHasLessArmourAndBlock",
	"playerHasLessLifeESRecovery",
	"playerHasMinusMaxResist",
	"playerHasPointBlank",
}

var configOrder = []string{
	"resistancePenalty",
	"bandit",
	"pantheonMajorGod",
	"pantheonMinorGod",
	"detonateDeadCorpseLife",
	"conditionStationary",
	"conditionMoving",
	"conditionInsane",
	"conditionFullLife",
	"conditionLowLife",
	"conditionFullMana",
	"conditionLowMana",
	"conditionFullEnergyShield",
	"conditionLowEnergyShield",
	"conditionHaveEnergyShield",
	"minionsConditionFullLife",
	"minionsConditionLowLife",
	"minionsConditionFullEnergyShield",
	"minionsConditionCreatedRecently",
	"ailmentMode",
	"physMode",
	"lifeRegenMode",
	"EHPUnluckyWorstOf",
	"DisableEHPGainOnBlock",
	"resourceGainMode",
	"armourCalculationMode",
	"warcryMode",
	"pactMode",
	"EVBypass",

	"targetBrandedEnemy",
	"aspectOfTheAvianAviansMight",
	"aspectOfTheAvianAviansFlight",
	"aspectOfTheCatCatsStealth",
	"aspectOfTheCatCatsAgility",
	"overrideCrabBarriers",
	"aspectOfTheSpiderWebStacks",
	"bannerPlanted",
	"bannerStages",
	"bloodsoakedBannerStages",
	"bladestormInBloodstorm",
	"bladestormInSandstorm",
	"bonechillEffect",
	"traumaStacks",
	"ActiveBrands",
	"BrandsAttachedToEnemy",
	"BrandsInLastQuarter",
	"carrionGolemNearbyMinion",
	"closeCombatCombatRush",
	"ConcPathBypassCD",
	"conditionCorruptingCryStages",
	"overrideCruelty",
	"channellingCycloneCheck",
	"darkPactSkeletonLife",
	"manabondMissingUnreservedManaPercentage",
	"divineSentinelPhysAsFire",
	"divineSentinelPhysAsLightning",
	"divineSentinelRegenLife",
	"divineSentinelRegenMana",
	"divineSentinelChaosResistance",
	"divineSentinelSelfCurseEffect",
	"curseOverlaps",
	"deathmarkDeathmarkActive",
	"elementalArmyExposureType",
	"energyBladeActive",
	"embraceMadnessActive",
	"touchedDebuffsCount",
	"feedingFrenzyFeedingFrenzyActive",
	"flameWallAddedDamage",
	"FlickerStrikeBypassCD",
	"frostboltExposure",
	"frostShieldStages",
	"greaterHarbingerOfTimeSlipstream",
	"harbingerOfTimeSlipstream",
	"multiplierHexDoom",
	"heraldOfAgonyVirulenceStack",
	"iceNovaCastOnFrostbolt",
	"infusedChannellingInfusion",
	"innervateInnervation",
	"intensifyIntensity",
	"OverloadedIntensity",
	"meatShieldEnemyNearYou",
	"plagueBearerState",
	"perforateSpikeOverlap",
	"physicalAegisDepleted",
	"prideEffect",
	"sacrificedRageCount",
	"raiseSpectreEnableBuffs",
	"raiseSpectreEnableCurses",
	"raiseSpectreBladeVortexBladeCount",
	"raiseSpectreKaomFireBeamTotemStage",
	"raiseSpectreEnableSummonedUrsaRallyingCry",
	"raiseSpidersSpiderCount",
	"animateWeaponLingeringBlade",
	"sigilOfPowerStages",
	"siphoningTrapAffectedEnemies",
	"configSnipeStages",
	"configResonanceCount",
	"configSpectralTigerCount",
	"configSpectralWolfCount",
	"bloodSandStance",
	"changedStance",
	"shardsConsumed",
	"steelWards",
	"stormRainBeamOverlap",
	"summonElementalRelicEnableAngerAura",
	"summonElementalRelicEnableHatredAura",
	"summonElementalRelicEnableWrathAura",
	"summonHolyRelicEnableHolyRelicBoon",
	"summonLightningGolemEnableWrath",
	"nearbyBleedingEnemies",
	"toxicRainPodOverlap",
	"hoaOverkill",
	"VigilantStrikeBypassCD",
	"voltaxicBurstSpellsQueued",
	"vortexCastOnFrostbolt",
	"ColdSnapBypassCD",
	"multiplierWarcryPower",
	"waveOfConvictionExposureType",
	"MoltenShellDamageMitigated",
	"VaalMoltenShellDamageMitigated",
	"enemySizePreset",
	"enemyRadius",

	"TotalMinionLife",
	"TotalSpectreLife",
	"TotalTotemLife",
	"TotalRadianceSentinelLife",
	"TotalVoidSpawnLife",
	"TotalStoneGolemLife",
	"TotalVaalRejuvenationTotemLife",

	"enemyHasPhysicalReduction",
	"enemyIsHexproof",
	"enemyHasLessCurseEffectOnSelf",
	"enemyCanAvoidPoisonBlindBleed",
	"enemyHasResistances",
	"playerHasElementalEquilibrium",
	"playerCannotLeech",
	"playerGainsReducedFlaskCharges",
	"playerHasMinusMaxResist",
	"playerHasLessAreaOfEffect",
	"enemyCanAvoidStatusAilment",
	"enemyHasIncreasedAccuracy",
	"playerHasLessArmourAndBlock",
	"playerHasPointBlank",
	"playerHasLessLifeESRecovery",
	"playerCannotRegenLifeManaEnergyShield",
	"enemyTakesReducedExtraCritDamage",
	"multiplierSextant",
	"MapPrefix1",
	"MapPrefix2",
	"MapPrefix3",
	"MapPrefix4",
	"MapSuffix1",
	"MapSuffix2",
	"MapSuffix3",
	"MapSuffix4",
	"PvpScaling",
	"playerCursedWithAssassinsMark",
	"playerCursedWithConductivity",
	"playerCursedWithDespair",
	"playerCursedWithElementalWeakness",
	"playerCursedWithEnfeeble",
	"playerCursedWithFlammability",
	"playerCursedWithFrostbite",
	"playerCursedWithPoachersMark",
	"playerCursedWithProjectileWeakness",
	"playerCursedWithPunishment",
	"playerCursedWithTemporalChains",
	"playerCursedWithVulnerability",
	"playerCursedWithWarlordsMark",

	"usePowerCharges",
	"overridePowerCharges",
	"useFrenzyCharges",
	"overrideFrenzyCharges",
	"useEnduranceCharges",
	"overrideEnduranceCharges",
	"useSiphoningCharges",
	"overrideSiphoningCharges",
	"useChallengerCharges",
	"overrideChallengerCharges",
	"useBlitzCharges",
	"overrideBlitzCharges",
	"multiplierGaleForce",
	"multiplierNonInstantSpellCastRecently",
	"overrideInspirationCharges",
	"useGhostShrouds",
	"overrideGhostShrouds",
	"waitForMaxSeals",
	"repeatMode",
	"overrideBloodCharges",
	"minionsUsePowerCharges",
	"minionsUseFrenzyCharges",
	"minionsUseEnduranceCharges",
	"minionsOverridePowerCharges",
	"minionsOverrideFrenzyCharges",
	"minionsOverrideEnduranceCharges",
	"multiplierRampage",
	"conditionFocused",
	"buffLifetap",
	"buffOnslaught",
	"minionBuffOnslaught",
	"buffUnholyMight",
	"minionbuffUnholyMight",
	"buffChaoticMight",
	"minionbuffChaoticMight",
	"conditionOnProfaneGround",
	"minionConditionOnProfaneGround",
	"multiplierSoulEater",
	"multiplierMinionSoulEater",
	"conditionEnemyHaveEnergyShield",
	"arcaneCloakUsedRecentlyCheck",
	"configUnholyResonanceCount",
	"multiplierCritRecently",
	"overrideSpiritInfusion",
	"minesPerThrow",
	"summonReaperConsumeRecently",
	"heraldOfTheHivePressure",
	"conditionSuppressedRecently",
	"buffPhasing",
	"buffFortification",
	"overrideFortification",
	"buffTailwind",
	"bannerValour",
	"barkskinStacks",
	"Unbound",
	"freshMeatBuffs",
	"Disgorged",
	"MomentumStacks",
	"overrideSpiritCharges",
	"conditionOnBrineGround",
	"conditionStunnedRecently",
	"conditionEnergyShieldRechargePastTwoSec",
	"DetonatedMinesRecently",
	"conditionNonPoisonedOnly",
	"overrideIgniteStackPotential",
	"enemyMaxResist",
	"buffHeartstopper",
	"buffArcaneSurge",
	"multiplierChannelling",
	"conditionHitSpellRecently",
	"multiplierLinkedTargets",
	"linkedToMinion",
	"linkedSourceRate",
	"multiplierChilledByYouSeconds",
	"multiplierFrozenByYouSeconds",
	"conditionUsedRubyFlaskRecently",
	"conditionUsedSapphireFlaskRecently",
	"conditionUsedTopazFlaskRecently",
	"conditionUsedAmethystFlaskRecently",
	"buffAdrenaline",
	"buffAlchemistsGenius",
	"buffVaalArcLuckyHits",
	"buffElusive",
	"overrideBuffElusive",
	"buffDivinity",
	"multiplierDefiance",
	"multiplierRage",
	"buffWildSavagery",
	"conditionLeeching",
	"conditionLeechingLife",
	"conditionLeechingEnergyShield",
	"conditionLeechingMana",
	"minionsConditionLeechingEnergyShield",
	"conditionUsingFlask",
	"conditionHaveTotem",
	"conditionSummonedTotemRecently",
	"TotemsSummoned",
	"conditionSummonedGolemInPast8Sec",
	"conditionSummonedGolemInPast10Sec",
	"multiplierNearbyAlly",
	"multiplierNearbyCorpse",
	"multiplierSummonedMinion",
	"multiplierNonVaalSummonedMinion",
	"conditionOnConsecratedGround",
	"conditionOnFungalGround",
	"conditionOnBurningGround",
	"conditionOnChilledGround",
	"conditionOnShockedGround",
	"conditionBlinded",
	"conditionBurning",
	"conditionIgnited",
	"conditionChilled",
	"conditionChilledEffect",
	"conditionSelfChill",
	"conditionFrozen",
	"conditionShocked",
	"conditionPlayerShockEffect",
	"conditionBleeding",
	"conditionPoisoned",
	"multiplierPoisonOnSelf",
	"multiplierWitheredStackCountSelf",
	"conditionAgainstDamageOverTime",
	"multiplierNearbyEnemies",
	"multiplierNearbyRareOrUniqueEnemies",
	"conditionHitRecently",
	"conditionCritRecently",
	"conditionSkillCritRecently",
	"conditionCritWithHeraldSkillRecently",
	"LostNonVaalBuffRecently",
	"conditionNonCritRecently",
	"conditionChannelling",
	"conditionHitRecentlyWithWeapon",
	"conditionKilledRecently",
	"multiplierKilledRecently",
	"conditionKilledLast3Seconds",
	"conditionKilledPoisonedLast2Seconds",
	"conditionTotemsNotSummonedInPastTwoSeconds",
	"conditionTotemsKilledRecently",
	"conditionTotemsHitRecently",
	"conditionTotemsHitSpellRecently",
	"multiplierShockedNonShockedEnemyRecently",
	"conditionUsedBrandRecently",
	"multiplierTotemsKilledRecently",
	"conditionMinionsKilledRecently",
	"conditionMinionsDiedRecently",
	"multiplierMinionsKilledRecently",
	"conditionKilledAffectedByDoT",
	"multiplierShockedEnemyKilledRecently",
	"conditionFrozenEnemyRecently",
	"conditionChilledEnemyRecently",
	"conditionShatteredEnemyRecently",
	"conditionIgnitedEnemyRecently",
	"conditionShockedEnemyRecently",
	"conditionStunnedEnemyRecently",
	"conditionPoisonedEnemyRecently",
	"multiplierPoisonAppliedRecently",
	"multiplierLifeSpentRecently",
	"multiplierManaSpentRecently",
	"conditionBeenHitRecently",
	"multiplierBeenHitRecently",
	"conditionBeenHitByAttackRecently",
	"conditionBeenCritRecently",
	"conditionConsumed12SteelShardsRecently",
	"conditionGainedPowerChargeRecently",
	"conditionGainedFrenzyChargeRecently",
	"conditionBeenSavageHitRecently",
	"conditionHitByFireDamageRecently",
	"conditionHitByColdDamageRecently",
	"conditionHitByLightningDamageRecently",
	"conditionHitBySpellDamageRecently",
	"conditionTakenFireDamageFromEnemyHitRecently",
	"conditionBlockedRecently",
	"conditionBlockedAttackRecently",
	"conditionBlockedSpellRecently",
	"conditionEnergyShieldRechargeRecently",
	"conditionStoppedTakingDamageOverTimeRecently",
	"conditionConvergence",
	"buffPendulum",
	"buffConflux",
	"buffBastionOfHope",
	"buffNgamahuFlamesAdvance",
	"buffHerEmbrace",
	"conditionChampionIntimidate",
	"conditionUsedSkillRecently",
	"multiplierSkillUsedRecently",
	"conditionAttackedRecently",
	"conditionCastSpellRecently",
	"conditionCastLast1Seconds",
	"multiplierCastLast8Seconds",
	"conditionUsedFireSkillRecently",
	"conditionUsedColdSkillRecently",
	"conditionUsedMinionSkillRecently",
	"conditionUsedTravelSkillRecently",
	"conditionUsedDashRecently",
	"conditionUsedMovementSkillRecently",
	"conditionUsedVaalSkillRecently",
	"multiplierUsedVaalSkillInPast8Seconds",
	"conditionLinkedRecently",
	"conditionSoulGainPrevention",
	"conditionSacrificeMinion",
	"conditionUsedWarcryRecently",
	"conditionUsedWarcryInPast8Seconds",
	"multiplierAffectedByWarcryBuffDuration",
	"multiplierMineDetonatedRecently",
	"multiplierTrapTriggeredRecently",
	"conditionThrownTrapOrMineRecently",
	"trapsPerThrow",
	"conditionCursedEnemyRecently",
	"conditionCastMarkRecently",
	"conditionSpawnedCorpseRecently",
	"conditionConsumedCorpseRecently",
	"conditionConsumedCorpseInPast2Sec",
	"multiplierCorpseConsumedRecently",
	"multiplierWarcryUsedRecently",
	"conditionTauntedEnemyRecently",
	"conditionLostEnduranceChargeInPast8Sec",
	"multiplierEnduranceChargesLostRecently",
	"conditionBlockedHitFromUniqueEnemyInPast10Sec",
	"BlockedPast10Sec",
	"conditionImpaledRecently",
	"multiplierImpalesOnEnemy",
	"conditionCausedBleedingRecently",
	"multiplierBleedsOnEnemy",
	"multiplierFragileRegrowth",
	"conditionKilledUniqueEnemy",
	"conditionHaveArborix",
	"conditionHaveAugyre",
	"conditionHaveVulconus",
	"conditionHaveManaStorm",
	"EverlastingSacrifice",
	"buffFanaticism",
	"multiplierPvpTvalueOverride",
	"multiplierPvpDamage",
	"buffAccelerationShrine",
	"buffBrutalShrine",
	"buffDiamondShrine",
	"buffDivineShrine",
	"buffEchoingShrine",
	"buffGloomShrine",
	"buffGreaterFreezingShrine",
	"buffGreaterShockingShrine",
	"buffGreaterSkeletalShrine",
	"buffImpenetrableShrine",
	"buffMassiveShrine",
	"buffReplenishingShrine",
	"buffResistanceShrine",
	"buffResonatingShrine",
	"buffLesserAccelerationShrine",
	"buffLesserBrutalShrine",
	"buffLesserImpenetrableShrine",
	"buffLesserMassiveShrine",
	"buffLesserReplenishingShrine",
	"buffLesserResistanceShrine",

	"critChanceLucky",
	"skillForkCount",
	"skillChainCount",
	"skillPierceCount",
	"meleeDistance",
	"projectileDistance",
	"conditionAtCloseRange",
	"enemyMultiplierEnemyPresenceSeconds",
	"conditionEnemyMoving",
	"conditionEnemyFullLife",
	"conditionEnemyLowLife",
	"conditionEnemyCursed",
	"conditionEnemyBleeding",
	"multiplierRuptureStacks",
	"conditionEnemyPoisoned",
	"multiplierPoisonOnEnemy",
	"multiplierWitheredStackCount",
	"multiplierCorrosionStackCount",
	"multiplierEnsnaredStackCount",
	"conditionEnemyMaimed",
	"conditionEnemyHindered",
	"conditionEnemyBlinded",
	"overrideBuffBlinded",
	"conditionEnemyTaunted",
	"conditionEnemyDebilitated",
	"conditionEnemyMalignantMadness",
	"conditionEnemyBurning",
	"conditionEnemyIgnited",
	"conditionEnemyScorched",
	"conditionScorchedEffect",
	"ScorchStacks",
	"conditionEnemyOnScorchedGround",
	"conditionEnemyChilled",
	"conditionEnemyChilledEffect",
	"conditionEnemyChilledByYourHits",
	"HoarfrostStacks",
	"multiplierBarnacleStacks",
	"conditionEnemyFrozen",
	"conditionEnemyBrittle",
	"conditionBrittleEffect",
	"conditionEnemyOnBrittleGround",
	"conditionEnemyShocked",
	"conditionShockEffect",
	"ShockStacks",
	"conditionEnemyOnShockedGround",
	"conditionEnemySapped",
	"conditionSapEffect",
	"conditionEnemyOnSappedGround",
	"multiplierFreezeShockIgniteOnEnemy",
	"conditionEnemyFireExposure",
	"conditionEnemyColdExposure",
	"conditionEnemyLightningExposure",
	"conditionEnemyIntimidated",
	"conditionEnemyCrushed",
	"conditionNearLinkedTarget",
	"conditionEnemyUnnerved",
	"conditionEnemyCoveredInAsh",
	"conditionEnemyCoveredInFrost",
	"conditionEnemyOnConsecratedGround",
	"conditionEnemyOnProfaneGround",
	"multiplierEnemyAffectedByGraspingVines",
	"conditionEnemyOnFungalGround",
	"conditionEnemyOnBrineGround",
	"conditionEnemyInChillingArea",
	"conditionEnemyInFrostGlobe",
	"enemyConditionHitByFireDamage",
	"enemyConditionHitByColdDamage",
	"enemyConditionHitByLightningDamage",
	"EEIgnoreHitDamage",
	"maniaDebuffsCount",

	"enemyLevel",
	"conditionEnemyRareOrUnique",
	"enemyIsBoss",
	"deliriousPercentage",
	"enemyPhysicalReduction",
	"enemyLightningResist",
	"enemyColdResist",
	"enemyFireResist",
	"enemyChaosResist",
	"enemyEvasion",
	"enemyArmour",
	"presetBossSkills",
	"enemyDamageType",
	"enemySpeed",
	"enemyMultiplierPvpDamage",
	"enemyCritChance",
	"enemyCritDamage",
	"enemyPhysicalDamage",
	"enemyLightningDamage",
	"enemyLightningPen",
	"enemyColdDamage",
	"enemyColdPen",
	"enemyFireDamage",
	"enemyFirePen",
	"enemyChaosDamage",

	"conditionUnbrokenWard",
	"bloodSacramentReservationEHP",
	"BrandsInLastHalf",
	"carrionGolemEqualsChaosGolem",
	"chaosGolemEqualsStoneGolem",
	"cinderflameStacks",
	"stoneGolemEqualsCarrionGolem",
	"inventionMineTrapPlacedDuration",
	"minionPactLife",
	"enemyHitMistyReflection",
	"MomentumSwiftnessStacks",
	"conditionSummonedSpectreInPast8Sec",
	"conditionSummonedZombieInPast8Sec",
	"weepingBlackStacks",
	"multiplierWoCExpiredDuration",
	"absolutionSkillDamageCountedOnce",
	"dominatingBlowSkillDamageCountedOnce",
	"holyStrikeSkillDamageCountedOnce",
	"buffSacrificialZeal",
	"conditionChangedStanceLastSecond",
	"conditionUsingTincture",
	"multiplierPermanentMinion",
	"minionConditionOnBrineGround",
	"conditionOnCausticGround",
	"conditionScorched",
	"conditionBrittle",
	"conditionSapped",
	"conditionCanBeCurseImmune",
	"conditionKilledTauntedEnemyRecently",
	"multiplierIgniteAppliedRecently",
	"conditionWardBrokenPast2Seconds",
	"highestDamageType",
	"multiplierAppliedAilmentsRecently",
	"conditionStunnedWhileCastingRecently",
	"multiplierHitsSuppressedRecently",
	"TriggeredTrapsRecently",
	"conditionRavenousCorpseConsumed",
	"GamblesprintMovementSpeed",
	"conditionHitsAlwaysStun",
	"conditionEnemyStunned",
	"overrideBleedStackPotential",
	"conditionSingleBleed",
	"multiplierCurseExpiredOnEnemy",
	"multiplierCurseDurationExpiredOnEnemy",
	"conditionEnemyExcommunicated",
	"conditionEnemyPacified",
	"conditionEnemyHallowingFlame",
	"multiplierEnemyHallowingFlame",
	"multiplierHallowingFlameStacksRemovedByAlly",
	"conditionHallowingFlameMagnitude",
	"conditionEnemyLifeHigherThanPlayer",
	"enemyInRFOrScorchingRay",
	"conditionBetweenYouAndLinkedTarget",
	"conditionEnemyFireResZero",
	"conditionEnemyColdResZero",
	"conditionEnemyLightningResZero",
	"enemyBlockChance",
	"customMods",
}

//pob:block:src/Classes/ConfigTab.lua:1232-1241
var configDefaultPlaceholders = map[string]float64{
	"sigilOfPowerStages":                          1,
	"configSpectralTigerCount":                    5,
	"multiplierWitheredStackCountSelf":            15,
	"meleeDistance":                               15,
	"projectileDistance":                          40,
	"ScorchStacks":                                1,
	"ShockStacks":                                 1,
	"multiplierEnemyHallowingFlame":               1,
	"multiplierHallowingFlameStacksRemovedByAlly": 1,
	"enemyDamageRollRange":                        70,
	"enemySpeed":                                  700,
	"enemyCritChance":                             5,
	"enemyCritDamage":                             30,
	"enemyPhysicalDamage":                         7,
}

//pob:block:src/Classes/ConfigTab.lua:1089-1092
var configCountAllowZero = map[string]bool{
	"overrideCrabBarriers": true, "aspectOfTheSpiderWebStacks": true, "touchedDebuffsCount": true,
	"sigilOfPowerStages": true, "overrideInspirationCharges": true, "overrideBloodCharges": true,
	"overrideSpiritCharges": true, "overrideSpiritInfusion": true, "multiplierWitheredStackCountSelf": true,
	"multiplierNearbyRareOrUniqueEnemies": true, "multiplierImpalesOnEnemy": true,
	"multiplierHallowingFlameStacksRemovedByAlly": true, "conditionHallowingFlameMagnitude": true,
	"maniaDebuffsCount": true, "enemyLightningResist": true, "enemyColdResist": true,
	"enemyFireResist": true, "enemyChaosResist": true, "enemyBlockChance": true, "enemyEvasion": true,
	"enemyArmour": true, "enemySpeed": true, "enemyCritChance": true, "enemyCritDamage": true,
	"enemyPhysicalDamage": true, "enemyPhysicalOverwhelm": true, "enemyLightningDamage": true,
	"enemyLightningPen": true, "enemyColdDamage": true, "enemyColdPen": true, "enemyFireDamage": true,
	"enemyFirePen": true, "enemyChaosDamage": true,
}

// elseif placeholder[var] and (placeholder[var] ~= 0 or type == "countAllowZero") then
var configPlaceholderVars = []string{
	"enemyPhysicalReduction",
	"enemyLightningResist",
	"enemyColdResist",
	"enemyFireResist",
	"enemyChaosResist",
	"enemyEvasion",
	"enemyArmour",
	"sigilOfPowerStages",
	"configSpectralTigerCount",
	"multiplierWitheredStackCountSelf",
	"ScorchStacks",
	"ShockStacks",
}

//pob:block:src/Classes/ConfigTab.lua:1091-1092
func applyConfigPlaceholders(env *Environment, modList *moddb.ModList, enemyModList *moddb.ModList, suppliedInputs map[string]bool) int {
	dispatched := 0
	for _, name := range configPlaceholderVars {
		if suppliedInputs[name] {
			continue
		}
		if applyConfigPlaceholder(env, name, modList, enemyModList) {
			dispatched++
		}
	}
	return dispatched
}

//pob:block:src/Classes/ConfigTab.lua:1091-1092
func applyConfigPlaceholder(env *Environment, name string, modList *moddb.ModList, enemyModList *moddb.ModList) bool {
	value := env.ConfigPlaceholder[name]
	if value == 0 {
		return false
	}
	config, ok := configurations[name]
	if !ok {
		return false
	}
	config(value, modList, enemyModList)
	return true
}

//pob:block:src/Classes/ConfigTab.lua:1089-1093
func isConfigPlaceholderVar(name string) bool {
	for _, n := range configPlaceholderVars {
		if n == name {
			return true
		}
	}
	return false
}

//pob:block:src/Classes/ConfigTab.lua:1233-1240
func isConfigDefault(name string) bool {
	if _, ok := configCountDefaults[name]; ok {
		return true
	}
	for _, n := range configDefaults {
		if n == name {
			return true
		}
	}
	return false
}

var configurations = map[string]ConfigApplyFunc{
	"raiseSpectreEnableBuffs": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "Enable", Value: 1}).Source("Config").
			Tag(mod.SkillType(string(data.SkillTypeBuff)), mod.SkillName("Raise Spectre").Transfigured(true).SummonSkill(true)))
	},
	"summonElementalRelicEnableAngerAura": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		summonElementalRelicAura(val, modList, "Anger")
	},
	"summonElementalRelicEnableHatredAura": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		summonElementalRelicAura(val, modList, "Hatred")
	},
	"summonElementalRelicEnableWrathAura": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		summonElementalRelicAura(val, modList, "Wrath")
	},
	"raiseSpectreEnableCurses": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "Enable", Value: 1}).Source("Config").
			Tag(mod.SkillType(string(data.SkillTypeHex)), mod.SkillName("Raise Spectre").Transfigured(true).SummonSkill(true)))
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "Enable", Value: 1}).Source("Config").
			Tag(mod.SkillType(string(data.SkillTypeMark)), mod.SkillName("Raise Spectre").Transfigured(true).SummonSkill(true)))
	},
	"usePowerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UsePowerCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overridePowerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("PowerCharges", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"useFrenzyCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseFrenzyCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideFrenzyCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("FrenzyCharges", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"useEnduranceCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseEnduranceCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideEnduranceCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("EnduranceCharges", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"useBlitzCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseBlitzCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideBlitzCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("BlitzCharges", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"detonateDeadCorpseLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "CorpseLife", Value: configFloat(val)}).Source("Config"))
	},
	"conditionStationary": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		out := configFloat(val)
		if v, ok := val.(bool); ok && v {
			// Backwards compatibility with older versions that set this condition as a boolean
			out = 1
		}
		sanitizedValue := math.Max(0, out)
		modList.AddMod(mod.NewFloat("Multiplier:StationarySeconds", mod.TypeBase, sanitizedValue).Source("Config"))
		if sanitizedValue > 0 {
			modList.AddMod(mod.NewFlag("Condition:Stationary", true).Source("Config"))
		}
	},
	"conditionMoving": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Moving", true).Source("Config"))
	},
	"conditionFullLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FullLife", true).Source("Config"))
	},
	"conditionLowLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LowLife", true).Source("Config"))
	},
	"conditionLowMana": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LowMana", true).Source("Config"))
	},
	"conditionFullEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FullEnergyShield", true).Source("Config"))
	},
	"conditionLowEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LowEnergyShield", true).Source("Config"))
	},
	"conditionHaveEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HaveEnergyShield", true).Source("Config"))
	},
	"minionsConditionFullLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:FullLife", true).Source("Config")}).Source("Config"))
	},
	"minionsConditionLowLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:LowLife", true).Source("Config")}).Source("Config"))
	},
	"minionsConditionFullEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:FullEnergyShield", true).Source("Config")}).Source("Config"))
	},
	"minionsConditionCreatedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:MinionsCreatedRecently", true).Source("Config"))
	},
	"lifeRegenMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch val {
		case "AVERAGE":
			modList.AddMod(mod.NewFlag("Condition:LifeRegenBurstAvg", true).Source("Config"))
		case "FULL":
			modList.AddMod(mod.NewFlag("Condition:LifeRegenBurstFull", true).Source("Config"))
		}
	},
	"resourceGainMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch configString(val) {
		case "AVERAGE":
			modList.AddMod(mod.NewFlag("Condition:AverageResourceGain", true).Source("Config"))
		case "MAX":
			modList.AddMod(mod.NewFlag("Condition:MaxResourceGain", true).Source("Config"))
		}
	},
	"armourCalculationMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch val {
		case "MAX":
			modList.AddMod(mod.NewFlag("Condition:ArmourMax", true).Source("Config"))
		case "AVERAGE":
			modList.AddMod(mod.NewFlag("Condition:ArmourAvg", true).Source("Config"))
		}
	},
	"warcryMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "MAX" {
			modList.AddMod(mod.NewFlag("Condition:WarcryMaxHit", true).Source("Config"))
		}
	},
	"EVBypass": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EVBypass", true).Source("Config"))
	},
	"targetBrandedEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TargetingBrandedEnemy", true).Source("Config"))
	},
	"aspectOfTheAvianAviansMight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AviansMightActive", true).Source("Config"))
	},
	"aspectOfTheAvianAviansFlight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AviansFlightActive", true).Source("Config"))
	},
	"aspectOfTheCatCatsStealth": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CatsStealthActive", true).Source("Config"))
	},
	"aspectOfTheCatCatsAgility": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CatsAgilityActive", true).Source("Config"))
	},
	"overrideCrabBarriers": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewFloat("CrabBarriers", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"aspectOfTheSpiderWebStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewList("ExtraSkillMod", mod.ExtraSkillMod{
			Mod: mod.NewFloat("Multiplier:SpiderWebApplyStack", mod.TypeBase, v),
		}).Source("Config").Tag(mod.SkillName("Aspect of the Spider")))
		if v > 0 {
			modList.AddMod(mod.NewFlag("Condition:AspectOfTheSpiderActive", true).Source("Config"))
		}
	},
	"bannerPlanted": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BannerPlanted", true).Source("Config"))
	},
	"bannerStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:BannerStage", mod.TypeBase, math.Min(configFloat(val), 50)).Source("Config"))
	},
	"bladestormInBloodstorm": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BladestormInBloodstorm", true).Source("Config").Tag(mod.SkillName("Bladestorm").Transfigured(true)))
	},
	"bladestormInSandstorm": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BladestormInSandstorm", true).Source("Config").Tag(mod.SkillName("Bladestorm").Transfigured(true)))
	},
	"bonechillEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("BonechillEffect", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("DesiredBonechillEffect", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"traumaStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:TraumaStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"bloodsoakedBannerStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configDefaultState("bloodsoakedBannerStages", val)
		if !ok || v == 0 {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:BloodsoakedBannerStageAfterFirst", mod.TypeBase, math.Min(v-1, 9)).Source("Config").Tag(mod.Condition("Effective")))
	},
	"ActiveBrands": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ConfigActiveBrands", mod.TypeBase, v).Source("Config"))
	},
	"BrandsAttachedToEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ConfigBrandsAttachedToEnemy", mod.TypeBase, v).Source("Config"))
	},
	"BrandsInLastQuarter": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BrandLastQuarter", true).Source("Config"))
		modList.AddMod(mod.NewFlag("Condition:BrandLastHalf", true).Source("Config"))
	},
	"carrionGolemNearbyMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NearbyNonGolemMinion", mod.TypeBase, v).Source("Config"))
	},
	"closeCombatCombatRush": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CombatRushActive", true).Source("Config"))
	},
	"ConcPathBypassCD": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CooldownRecovery", mod.TypeOverride, 0).Source("Config").Tag(mod.SkillName("Consecrated Path of Endurance")))
	},
	"conditionCorruptingCryStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configDefaultState("conditionCorruptingCryStages", val)
		if !ok || v == 0 {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CorruptingCryStageAfterFirst", mod.TypeBase, math.Min(v-1, 9)).Source("Config").Tag(mod.Condition("Effective")))
	},
	"overrideCruelty": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Cruelty", mod.TypeOverride, math.Min(v, 40)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"channellingCycloneCheck": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChannellingCyclone", true).Source("Config"))
	},
	"darkPactSkeletonLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "SkeletonLife", Value: v}).Source("Config").
			Tag(mod.SkillName("Dark Bargain")))
	},
	"manabondMissingUnreservedManaPercentage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{
			Key:   "ManabondMissingUnreservedManaPercentage",
			Value: math.Max(math.Min(v, 100), 0),
		}).Source("Config").Tag(mod.SkillName("Manabond")))
	},

	"divineSentinelPhysAsFire": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelPhysAsFire", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelPhysAsFire", true).Source("Config")}).Source("Config"))
	},
	"divineSentinelPhysAsLightning": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelPhysAsLightning", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelPhysAsLightning", true).Source("Config")}).Source("Config"))
	},
	"divineSentinelRegenLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelRegenLife", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelRegenLife", true).Source("Config")}).Source("Config"))
	},
	"divineSentinelRegenMana": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelRegenMana", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelRegenMana", true).Source("Config")}).Source("Config"))
	},
	"divineSentinelChaosResistance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelChaosResistance", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelChaosResistance", true).Source("Config")}).Source("Config"))
	},
	"divineSentinelSelfCurseEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineSentinelSelfCurseEffect", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:DivineSentinelSelfCurseEffect", true).Source("Config")}).Source("Config"))
	},
	"deathmarkDeathmarkActive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EnemyHasDeathmark", true).Source("Config"))
	},
	"elementalArmyExposureType": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch val {
		case "Fire":
			modList.AddMod(mod.NewFloat("FireExposureChance", mod.TypeBase, 100).Source("Config"))
		case "Cold":
			modList.AddMod(mod.NewFloat("ColdExposureChance", mod.TypeBase, 100).Source("Config"))
		case "Lightning":
			modList.AddMod(mod.NewFloat("LightningExposureChance", mod.TypeBase, 100).Source("Config"))
		}
	},
	"energyBladeActive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EnergyBladeActive", true).Source("Config"))
	},
	//pob:src/Modules/ConfigOptions.lua:423-425
	"embraceMadnessActive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AffectedByGloriousMadness", true).Source("Config"))
	},
	"touchedDebuffsCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configDefaultState("touchedDebuffsCount", val)
		if !ok {
			return
		}
		v = math.Min(v, 10)
		count := strconv.FormatFloat(v, 'f', -1, 64)
		for _, m := range []struct {
			name   string
			value  float64
			source string
		}{
			{"DamageTaken", v * 6, "Eroding Touch"},
			{"ActionSpeed", -v * 6, "Paralysing Touch"},
			{"FlaskChargesGained", -v * 9, "Diluting Touch"},
			{"FlaskEffect", -v * 9, "Diluting Touch"},
			{"LifeRecoveryRate", -v * 9, "Wasting Touch"},
			{"EnergyShieldRecoveryRate", -v * 9, "Wasting Touch"},
		} {
			modList.AddMod(mod.NewFloat(m.name, mod.TypeIncrease, m.value).
				Source(mod.Source(count+" "+m.source+" Stacks")).
				Tag(mod.GlobalEffect("Debuff"), mod.Condition("AffectedByGloriousMadness")))
		}
	},
	"feedingFrenzyFeedingFrenzyActive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FeedingFrenzyActive", true).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 10).Source("Feeding Frenzy")}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, 10).Source("Feeding Frenzy")}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, 10).Source("Feeding Frenzy")}).Source("Config"))
	},
	"flameWallAddedDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FlameWallAddedDamage", true).Source("Config"))
	},
	"frostboltExposure": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("ColdExposureChance", mod.TypeBase, 20).Source("Config"))
	},
	"FlickerStrikeBypassCD": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CooldownRecovery", mod.TypeOverride, 0).Source("Config").Tag(mod.SkillName("Flicker Strike").Transfigured(true)))
	},
	"frostShieldStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:FrostShieldStage", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	"greaterHarbingerOfTimeSlipstream": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GreaterHarbingerOfTime", true).Source("Config"))
	},
	"multiplierHexDoom": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:HexDoomStack", mod.TypeBase, v).Source("Config"))
	},
	"heraldOfAgonyVirulenceStack": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:VirulenceStack", mod.TypeBase, v).Source("Config"))
	},
	"iceNovaCastOnFrostbolt": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastOnFrostbolt", true).Source("Config").Tag(mod.SkillName("Ice Nova of Frostbolts")))
	},
	"infusedChannellingInfusion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:InfusionActive", true).Source("Config"))
	},
	"innervateInnervation": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:InnervationActive", true).Source("Config"))
	},
	"intensifyIntensity": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:Intensity", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	"OverloadedIntensity": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:OverloadedIntensity", mod.TypeBase, math.Min(configFloat(val), 3)).Source("Config"))
	},
	"meatShieldEnemyNearYou": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:MeatShieldEnemyNearYou", true).Source("Config"))
	},
	"plagueBearerState": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch val {
		case "INC":
			modList.AddMod(mod.NewFlag("Condition:PlagueBearerIncubating", true).Source("Config"))
		case "INF":
			modList.AddMod(mod.NewFlag("Condition:PlagueBearerInfecting", true).Source("Config"))
		}
	},
	"perforateSpikeOverlap": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:PerforateSpikeOverlap", mod.TypeBase, v).Source("Config").Tag(mod.SkillName("Perforate").Transfigured(true)))
	},
	"prideEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "MAX" {
			modList.AddMod(mod.NewFlag("Condition:PrideMaxEffect", true).Source("Config"))
		}
	},
	"sacrificedRageCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:RageSacrificedStacks", mod.TypeBase, v).Source("Config"))
	},
	"raiseSpidersSpiderCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:RaisedSpiderConfig", mod.TypeBase, v).Source("Config"))
		modList.AddMod(mod.NewFloat("Multiplier:RaisedSpider", mod.TypeBase, 1).Source("Config").
			Tag(mod.Multiplier("RaisedSpiderConfig").LimitStat("ActiveSpiderLimit")))
	},
	"animateWeaponLingeringBlade": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AnimatingLingeringBlades", true).Source("Config"))
	},
	"sigilOfPowerStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:SigilOfPowerStage", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	"siphoningTrapAffectedEnemies": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnemyAffectedBySiphoningTrap", mod.TypeBase, v).Source("Config"))
		modList.AddMod(mod.NewFlag("Condition:SiphoningTrapSiphoning", true).Source("Config"))
	},
	"configSnipeStages": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SnipeStage", mod.TypeBase, math.Min(v, 6)).Source("Config"))
	},
	"configResonanceCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:ResonanceCount", mod.TypeBase, math.Max(math.Min(configFloat(val), 50), 0)).Source("Config"))
	},
	"configSpectralWolfCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SpectralWolfConfig", mod.TypeBase, v).Source("Config"))
		modList.AddMod(mod.NewFloat("Multiplier:SpectralWolfCount", mod.TypeBase, 1).Source("Config").
			Tag(mod.Multiplier("SpectralWolfConfig").LimitStat("ActiveWolfLimit")))
	},
	"configSpectralTigerCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SpectralTigerConfig", mod.TypeBase, v).Source("Config"))
		modList.AddMod(mod.NewFloat("Multiplier:SpectralTigerCount", mod.TypeBase, 1).Source("Config").
			Tag(mod.Multiplier("SpectralTigerConfig").LimitStat("ActiveTigerLimit")))
	},
	"bloodSandStance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "SAND" {
			modList.AddMod(mod.NewFlag("Condition:SandStance", true).Source("Config"))
		}
	},
	"changedStance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChangedStanceRecently", true).Source("Config"))
	},
	"shardsConsumed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SteelShardConsumed", mod.TypeBase, math.Min(v, 12)).Source("Config"))
	},
	"steelWards": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SteelWardCount", mod.TypeBase, v).Source("Config"))
	},
	"summonHolyRelicEnableHolyRelicBoon": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HolyRelicBoonActive", true).Source("Config"))
	},
	"summonLightningGolemEnableWrath": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "Enable", Value: 1}).Source("Config").
			Tag(mod.SkillId("LightningGolemWrath")))
	},
	"nearbyBleedingEnemies": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NearbyBleedingEnemies", mod.TypeBase, v).Source("Config"))
	},
	"toxicRainPodOverlap": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "podOverlapMultiplier", Value: v}).Source("Config").
			Tag(mod.SkillName("Toxic Rain").Transfigured(true)))
	},
	"VigilantStrikeBypassCD": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CooldownRecovery", mod.TypeOverride, 0).Source("Config").Tag(mod.SkillName("Vigilant Strike")))
	},
	"voltaxicBurstSpellsQueued": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:VoltaxicWaitingStages", mod.TypeBase, v).Source("Config"))
	},
	"vortexCastOnFrostbolt": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastOnFrostbolt", true).Source("Config").Tag(mod.SkillName("Vortex of Projection")))
	},
	"multiplierWarcryPower": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"waveOfConvictionExposureType": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "Fire" {
			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionFireExposureActive", true).Source("Config"))
		} else if val == "Cold" {
			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionColdExposureActive", true).Source("Config"))
		} else if val == "Lightning" {
			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionLightningExposureActive", true).Source("Config"))
		}
	},
	"enemySizePreset": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		radius, ok := map[string]float64{"Small": 2, "Medium": 3, "Large": 5, "Huge": 11}[configString(val)]
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("EnemyRadius", mod.TypeBase, radius).Source("Config"))
	},
	"enemyRadius": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("EnemyRadius", mod.TypeOverride, math.Max(configFloat(val), 1)).Source("Config"))
	},
	"TotalMinionLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalMinionLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalSpectreLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalSpectreLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalTotemLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalTotemLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalRadianceSentinelLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalRadianceSentinelLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalVoidSpawnLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalVoidSpawnLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalStoneGolemLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalStoneGolemLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"TotalVaalRejuvenationTotemLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("TotalVaalRejuvenationTotemLife", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	"enemyHasPhysicalReduction": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	"enemyIsHexproof": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Hexproof", true).Source("Config"))
	},
	"enemyHasLessCurseEffectOnSelf": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -v).Source("Config"))
	},
	"enemyCanAvoidPoisonBlindBleed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		enemyModList.AddMod(mod.NewFloat("AvoidPoison", mod.TypeBase, v).Source("Config"))
		enemyModList.AddMod(mod.NewFloat("AvoidBleed", mod.TypeBase, v).Source("Config"))
	},
	"enemyHasResistances": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		tier, ok := map[interface{}][2]float64{"LOW": {20, 15}, "MID": {30, 20}, "HIGH": {40, 25}}[val]
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, tier[0]).Source("Config"))
		enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, tier[1]).Source("Config"))
	},
	"playerHasElementalEquilibrium": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("Keystone", "Elemental Equilibrium").Source("Config"))
	},
	"playerCannotLeech": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {},
	"playerGainsReducedFlaskCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		modList.AddMod(mod.NewFloat("FlaskChargesGained", mod.TypeIncrease, -v).Source("Config"))
	},
	"playerHasMinusMaxResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, -v).Source("Config"))
		modList.AddMod(mod.NewFloat("ColdResistMax", mod.TypeBase, -v).Source("Config"))
		modList.AddMod(mod.NewFloat("LightningResistMax", mod.TypeBase, -v).Source("Config"))
		modList.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, -v).Source("Config"))
	},
	"playerHasLessAreaOfEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		modList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeMore, -v).Source("Config"))
	},
	"enemyCanAvoidStatusAilment": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		enemyModList.AddMod(mod.NewFloat("AvoidIgnite", mod.TypeBase, v).Source("Config"))
		enemyModList.AddMod(mod.NewFloat("AvoidShock", mod.TypeBase, v).Source("Config"))
		enemyModList.AddMod(mod.NewFloat("AvoidFreeze", mod.TypeBase, v).Source("Config"))
	},
	"enemyHasIncreasedAccuracy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		modList.AddMod(mod.NewFlag("DodgeChanceIsUnlucky", true).Source("Config"))
		enemyModList.AddMod(mod.NewFloat("Accuracy", mod.TypeIncrease, v).Source("Config"))
	},
	"playerHasLessArmourAndBlock": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		tier, ok := map[interface{}][2]float64{"LOW": {20, 20}, "MID": {30, 25}, "HIGH": {40, 30}}[val]
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("BlockChance", mod.TypeIncrease, -tier[0]).Source("Config"))
		modList.AddMod(mod.NewFloat("Armour", mod.TypeMore, -tier[1]).Source("Config"))
	},
	"playerHasPointBlank": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("Keystone", "Point Blank").Source("Config"))
	},
	"playerHasLessLifeESRecovery": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if v == 0 {
			return
		}
		modList.AddMod(mod.NewFloat("LifeRecoveryRate", mod.TypeMore, -v).Source("Config"))
		modList.AddMod(mod.NewFloat("EnergyShieldRecoveryRate", mod.TypeMore, -v).Source("Config"))
	},
	"playerCannotRegenLifeManaEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("NoLifeRegen", true).Source("Config"))
		modList.AddMod(mod.NewFlag("NoEnergyShieldRegen", true).Source("Config"))
		modList.AddMod(mod.NewFlag("NoManaRegen", true).Source("Config"))
	},
	"enemyTakesReducedExtraCritDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("SelfCritMultiplier", mod.TypeIncrease, -v).Source("Config"))
	},
	"multiplierSextant": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:Sextant", mod.TypeBase, math.Min(v, 5)).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:833-841
	//pob:block:src/Modules/ConfigOptions.lua:843-845
	"PvpScaling": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("HasPvpScaling", true).Source("Config"))
	},
	"playerCursedWithAssassinsMark": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "AssassinsMark", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithConductivity": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Conductivity", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithDespair": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Despair", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithElementalWeakness": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "ElementalWeakness", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithEnfeeble": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Enfeeble", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithFlammability": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Flammability", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithFrostbite": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Frostbite", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithPoachersMark": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "PoachersMark", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithProjectileWeakness": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "ProjectileWeakness", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithPunishment": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Punishment", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithTemporalChains": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "TemporalChains", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithVulnerability": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "Vulnerability", Level: int(v), ApplyToPlayer: true}))
	},
	"playerCursedWithWarlordsMark": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("ExtraCurse", mod.ExtraCurse{SkillID: "WarlordsMark", Level: int(v), ApplyToPlayer: true}))
	},
	"useSiphoningCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseSiphoningCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideSiphoningCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("SiphoningCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"useChallengerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseChallengerCharges", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideChallengerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ChallengerCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierGaleForce": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:GaleForce", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.IgnoreCond(), mod.Condition("Combat"), mod.Condition("CanGainGaleForce")))
	},
	"overrideInspirationCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewFloat("InspirationCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"useGhostShrouds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseGhostShrouds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideGhostShrouds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("GhostShrouds", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"waitForMaxSeals": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("UseMaxUnleash", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"repeatMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := val.(string)
		if !ok {
			return
		}
		var name string
		switch v {
		case "AVERAGE":
			name = "Condition:averageRepeat"
		case "FINAL", "FINAL_DPS":
			name = "Condition:alwaysFinalRepeat"
		default:
			return
		}
		modList.AddMod(mod.NewFlag(name, true).Source("Config"))
		modList.AddMod(MOD("MinionModifier", "LIST",
			mod.MinionModifier{Mod: mod.NewFlag(name, true).Source("Config")}))
	},
	"overrideBloodCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewFloat("BloodCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minionsUsePowerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("UsePowerCharges", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"minionsUseFrenzyCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("UseFrenzyCharges", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"minionsUseEnduranceCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("UseEnduranceCharges", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"minionsOverridePowerCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PowerCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"minionsOverrideFrenzyCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("FrenzyCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"minionsOverrideEnduranceCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("EnduranceCharges", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"pactMode": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if configString(val) != "MAX" {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:PactMaxHit", true).Source("Config"))
	},
	"multiplierNonInstantSpellCastRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NonInstantSpellCastRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierRampage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:Rampage", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionFocused": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Focused", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLifetap": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Lifetap", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFloat("FlaskLifeRecovery", mod.TypeIncrease, 20).Source("Lifetap"))
	},
	"buffOnslaught": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Onslaught", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minionBuffOnslaught": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:Onslaught", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"buffUnholyMight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UnholyMight", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:CanWither", true).Source("Unholy Might").Tag(mod.Condition("Combat")))
	},
	"minionbuffUnholyMight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:UnholyMight", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanWither", true).Source("Unholy Might").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"conditionSuppressedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SuppressedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"summonReaperConsumeRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").
			Tag(mod.SkillId("ReaperConsumeMinionForBuff")))
	},
	"heraldOfTheHivePressure": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:OtherworldlyPressure", mod.TypeBase, v).Source("Config"))
	},
	"ColdSnapBypassCD": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CooldownRecovery", mod.TypeOverride, 0).Source("Config").
			Tag(mod.SkillName("Cold Snap").Transfigured(true)))
	},
	"conditionLeechingMana": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LeechingMana", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionOnProfaneGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnProfaneGround", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minionConditionOnProfaneGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:OnProfaneGround", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"multiplierSoulEater": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SoulEaterStack", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierMinionSoulEater": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:SoulEaterStack", mod.TypeBase, v).Source("Config")}).Source("Config"))
	},
	"conditionEnemyHaveEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HaveEnergyShield", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"arcaneCloakUsedRecentlyCheck": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ArcaneCloakUsedRecently", true).Source("Config"))
	},
	"configUnholyResonanceCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:UnholyResonanceCount", mod.TypeBase, max(min(v, 50), 0)).Source("Config"))
	},
	"multiplierCritRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CritRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:SkillCritRecently", 1 <= v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideSpiritInfusion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("SpiritInfusion", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minesPerThrow": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("MineThrowCount", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffChaoticMight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChaoticMight", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minionbuffChaoticMight": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:ChaoticMight", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"buffPhasing": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Phasing", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffFortification": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Fortified", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideFortification": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("FortificationStacks", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffTailwind": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Tailwind", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"bannerValour": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ValourStacks", mod.TypeBase, v).Source("Config").
			Tag(mod.IgnoreCond(), mod.Condition("Combat")))
	},
	"barkskinStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:BarkskinStacks", mod.TypeBase, min(v, 10)).Source("Config"))
		modList.AddMod(mod.NewFloat("Multiplier:MissingBarkskinStacks", mod.TypeBase, max(-v, -10)).Source("Config"))
	},
	"Unbound": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Unbound", true).Source("Config"))
	},
	"freshMeatBuffs": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FreshMeatActive", true).Source("Config"))
	},
	"Disgorged": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Disgorged", true).Source("Config"))
	},
	"MomentumStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:MomentumStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"overrideSpiritCharges": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("SpiritCharges", mod.TypeOverride, configFloat(val)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionOnBrineGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnBrineGround", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, 10).Source("Config").Tag(mod.Condition("OnBrineGround")))
		modList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, 10).Source("Config").Tag(mod.Condition("OnBrineGround")))
	},
	"conditionStunnedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StunnedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionEnergyShieldRechargePastTwoSec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EnergyShieldRechargePastTwoSec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"DetonatedMinesRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DetonatedMinesRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionNonPoisonedOnly": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:NonPoisonedOnly", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"overrideIgniteStackPotential": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("IgniteStackPotentialOverride", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"enemyMaxResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("DoNotChangeMaxResFromConfig", true).Source("EnemyConfig"))
	},
	"buffHeartstopper": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch configString(val) {
		case "HIT":
			modList.AddMod(mod.NewFlag("Condition:HeartstopperHIT", true).Source("Config").Tag(mod.Condition("Combat")))
		case "DOT":
			modList.AddMod(mod.NewFlag("Condition:HeartstopperDOT", true).Source("Config").Tag(mod.Condition("Combat")))
		case "AVERAGE":
			modList.AddMod(mod.NewFlag("Condition:HeartstopperAVERAGE", true).Source("Config").Tag(mod.Condition("Combat")))
		}
	},
	"buffArcaneSurge": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ArcaneSurge", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierChannelling": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ChannellingTime", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Channelling", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHitSpellRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:HitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"linkedToMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LinkedToMinion", true).Source("Config"))
	},
	"curseOverlaps": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CurseOverlaps", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"linkedSourceRate": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("IntuitiveLinkSourceRate", mod.TypeBase, v).Source("Config"))
	},
	"multiplierLinkedTargets": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:LinkedTargets", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1877-1880
	"multiplierChilledByYouSeconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:ChilledByYouSeconds", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		enemyModList.AddMod(mod.NewFlag("Condition:ChilledByYou", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierFrozenByYouSeconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:FrozenByYouSeconds", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		enemyModList.AddMod(mod.NewFlag("Condition:FrozenByYou", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionUsedRubyFlaskRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedRubyFlaskRecently", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("HaveRubyFlask")))
	},
	"conditionUsedSapphireFlaskRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedSapphireFlaskRecently", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("HaveSapphireFlask")))
	},
	"conditionUsedTopazFlaskRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedTopazFlaskRecently", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("HaveTopazFlask")))
	},
	"conditionUsedAmethystFlaskRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedAmethystFlaskRecently", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("HaveAmethystFlask")))
	},
	"buffAdrenaline": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Adrenaline", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffAlchemistsGenius": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AlchemistsGenius", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanHaveAlchemistGenius")))
	},
	"buffVaalArcLuckyHits": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("LuckyHits", true).Source("Config").
			Tag(mod.Condition("Combat", "CanBeLucky"), mod.SkillName("Arc").Transfigured(true)))
	},
	"buffElusive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Elusive", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanBeElusive")))
		modList.AddMod(mod.NewFlag("Elusive", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanBeElusive")))
	},
	"overrideBuffElusive": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ElusiveEffect", mod.TypeOverride, v).Source("Config").Tag(mod.GlobalEffect("Buff")))
	},
	"buffDivinity": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Divinity", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierDefiance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:Defiance", mod.TypeBase, math.Min(v, 10)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierRage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:RageStack", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.IgnoreCond(), mod.Condition("Combat"), mod.Condition("CanGainRage")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1065-1067
	"buffWildSavagery": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:WildSavagery", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionLeeching": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionLeechingLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LeechingLife", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionLeechingEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LeechingEnergyShield", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"minionsConditionLeechingEnergyShield": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:LeechingEnergyShield", true).Source("Config")}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:Leeching", true).Source("Config")}).Source("Config"))
	},
	"conditionUsingFlask": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHaveTotem": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HaveTotem", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSummonedTotemRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SummonedTotemRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"TotemsSummoned": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("TotemsSummoned", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:HaveTotem", v >= 1).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSummonedGolemInPast8Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SummonedGolemInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSummonedGolemInPast10Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SummonedGolemInPast10Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierNearbyAlly": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NearbyAlly", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierNearbyCorpse": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NearbyCorpse", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1128-1130
	"multiplierSummonedMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SummonedMinion", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1131-1133
	"multiplierNonVaalSummonedMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:NonVaalSummonedMinion", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionOnConsecratedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnConsecratedGround", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:OnConsecratedGround", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
	},
	"conditionOnFungalGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnFungalGround", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionOnBurningGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnBurningGround", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBlinded": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Blinded", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBurning": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionIgnited": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Ignited", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionChilled": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionChilledEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ChillVal", mod.TypeOverride, v).Source("Chill").Tag(mod.Condition("Chilled")))
	},
	"conditionSelfChill": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChilledSelf", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionShocked": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionPlayerShockEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ShockVal", mod.TypeOverride, v).Source("Shock").Tag(mod.Condition("Shocked")))
	},
	"conditionBleeding": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierPoisonOnSelf": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:PoisonStack", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionAgainstDamageOverTime": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AgainstDamageOverTime", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierNearbyEnemies": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewFloat("Multiplier:NearbyEnemies", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:OnlyOneNearbyEnemy", v == 1).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierNearbyRareOrUniqueEnemies": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		modList.AddMod(mod.NewFloat("Multiplier:NearbyRareOrUniqueEnemies", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFloat("Multiplier:NearbyEnemies", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:AtMostOneNearbyRareOrUniqueEnemy", v <= 1).Source("Config").Tag(mod.Condition("Combat")))
		enemyModList.AddMod(mod.NewFlag("Condition:NearbyRareOrUniqueEnemy", v >= 1).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionCritRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:SkillCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSkillCritRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SkillCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionCritWithHeraldSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CritWithHeraldSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"LostNonVaalBuffRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LostNonVaalBuffRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionNonCritRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:NonCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionChannelling": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Channelling", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHitRecentlyWithWeapon": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitRecentlyWithWeapon", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:KilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionKilledLast3Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledLast3Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionPoisonedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:PoisonedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionCausedBleedingRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CausedBleedingRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"enemyMultiplierEnemyPresenceSeconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:EnemyPresenceSeconds", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionKilledPoisonedLast2Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledPoisonedLast2Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTotemsNotSummonedInPastTwoSeconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:NoSummonedTotemsInPastTwoSeconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTotemsKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TotemsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTotemsHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TotemsHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTotemsHitSpellRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TotemsHitSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:TotemsHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedBrandRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedBrandRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierTotemsKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledByTotemsRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:TotemsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionMinionsKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:MinionsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionMinionsDiedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:MinionsDiedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierMinionsKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledByMinionsRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:MinionsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionKilledAffectedByDoT": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledAffectedByDotRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierShockedEnemyKilledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ShockedEnemyKilledRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierShockedNonShockedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ShockedNonShockedEnemyRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionFrozenEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FrozenEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionChilledEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChilledEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionShatteredEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ShatteredEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionIgnitedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:IgnitedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionShockedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ShockedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionStunnedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StunnedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierPoisonAppliedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:PoisonAppliedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierLifeSpentRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:LifeSpentRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierManaSpentRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:ManaSpentRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBeenHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierBeenHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:BeenHitRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", 1 <= v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBeenHitByAttackRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BeenHitByAttackRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBeenCritRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BeenCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionConsumed12SteelShardsRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Consumed12SteelShardsRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionGainedPowerChargeRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GainedPowerChargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionGainedFrenzyChargeRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GainedFrenzyChargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBeenSavageHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BeenSavageHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHitBySpellDamageRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitBySpellDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTakenFireDamageFromEnemyHitRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TakenFireDamageFromEnemyHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBlockedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBlockedAttackRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BlockedAttackRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBlockedSpellRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BlockedSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionStoppedTakingDamageOverTimeRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StoppedTakingDamageOverTimeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionConvergence": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Convergence", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanGainConvergence")))
	},
	"buffConflux": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "CHILLING" || val == "ALL" {
			modList.AddMod(mod.NewFlag("Condition:ChillingConflux", true).Source("Config").Tag(mod.Condition("Combat")))
		}
		if val == "SHOCKING" || val == "ALL" {
			modList.AddMod(mod.NewFlag("Condition:ShockingConflux", true).Source("Config").Tag(mod.Condition("Combat")))
		}
		if val == "IGNITING" || val == "ALL" {
			modList.AddMod(mod.NewFlag("Condition:IgnitingConflux", true).Source("Config").Tag(mod.Condition("Combat")))
		}
	},
	"buffBastionOfHope": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BastionOfHopeActive", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffNgamahuFlamesAdvance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:NgamahuFlamesAdvance", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffHerEmbrace": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("HerEmbrace", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanGainHerEmbrace")))
	},
	"conditionUsedSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionPoisoned": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierSkillUsedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:SkillUsedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedTravelSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedTravelSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:src/Modules/ConfigOptions.lua:1499-1501
	"conditionLinkedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LinkedRecently", true).Source("Config"))
	},
	"conditionUsedVaalSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedVaalSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHaveManaStorm": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SacrificeManaForLightning", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierRuptureStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:RuptureStack", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionNearLinkedTarget": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:NearLinkedTarget", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionAttackedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AttackedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionCastSpellRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierCastLast8Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CastLast8Seconds", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedMinionSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedMinionSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedMovementSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierUsedVaalSkillInPast8Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:VaalSkillsUsedInPast8Seconds", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSoulGainPrevention": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SoulGainPrevention", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedWarcryRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedWarcryRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionUsedWarcryInPast8Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierAffectedByWarcryBuffDuration": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:AffectedByWarcryBuffDuration", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierMineDetonatedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:MineDetonatedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierTrapTriggeredRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:TrapTriggeredRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionThrownTrapOrMineRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TrapOrMineThrownRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"trapsPerThrow": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("TrapThrowCount", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionCursedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CursedEnemyRecently", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionSpawnedCorpseRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SpawnedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionConsumedCorpseRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionConsumedCorpseInPast2Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseInPast2Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierCorpseConsumedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CorpseConsumedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierWarcryUsedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:WarcryUsedRecently", mod.TypeBase, math.Min(v, 100)).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedWarcryRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionTauntedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TauntedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionLostEnduranceChargeInPast8Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LostEnduranceChargeInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierEnduranceChargesLostRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnduranceChargesLostRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:LostEnduranceChargeInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionBlockedHitFromUniqueEnemyInPast10Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BlockedHitFromUniqueEnemyInPast10Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionImpaledRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ImpaledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierImpalesOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		enemyModList.AddMod(mod.NewFloat("Multiplier:ImpaleStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierBleedsOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:BleedStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		enemyModList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierFragileRegrowth": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:FragileRegrowthCount", mod.TypeBase, math.Min(v, 10)).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionKilledUniqueEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledUniqueEnemy", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionHaveArborix": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HaveIronReflexes", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Config"))
	},
	"conditionHaveAugyre": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "EleOverload" {
			modList.AddMod(mod.NewFlag("Condition:HaveElementalOverload", true).Source("Config").Tag(mod.Condition("Combat")))
			modList.AddMod(mod.NewList("Keystone", "Elemental Overload").Source("Config"))
		} else if val == "ResTechnique" {
			modList.AddMod(mod.NewFlag("Condition:HaveResoluteTechnique", true).Source("Config").Tag(mod.Condition("Combat")))
			modList.AddMod(mod.NewList("Keystone", "Resolute Technique").Source("Config"))
		}
	},
	"conditionHaveVulconus": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HaveAvatarOfFire", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewList("Keystone", "Avatar of Fire").Source("Config"))
	},
	//pob:src/Modules/ConfigOptions.lua:1676-1679
	"EverlastingSacrifice": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("ElementalResistMax", mod.TypeBase, 5).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("EverlastingSacrifice")))
		modList.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, 5).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("EverlastingSacrifice")))
	},
	"buffFanaticism": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Fanaticism", true).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("CanGainFanaticism")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1686-1691
	"multiplierPvpTvalueOverride": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("MultiplierPvpTvalueOverride", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	"multiplierPvpDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("PvpDamageMultiplier", mod.TypeMore, v-100).Source("Config"))
	},
	"buffAccelerationShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AccelerationShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffBrutalShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BrutalShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffDiamondShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DiamondShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffDivineShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DivineShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffEchoingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EchoingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffGloomShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GloomShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffGreaterFreezingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GreaterFreezingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffGreaterShockingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GreaterShockingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffGreaterSkeletalShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:GreaterSkeletalShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffImpenetrableShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ImpenetrableShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffMassiveShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:MassiveShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffReplenishingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ReplenishingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffResistanceShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ResistanceShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffResonatingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ResonatingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserAccelerationShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserAccelerationShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserBrutalShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserBrutalShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserImpenetrableShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserImpenetrableShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserMassiveShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserMassiveShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserReplenishingShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserReplenishingShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"buffLesserResistanceShrine": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:LesserResistanceShrine", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"critChanceLucky": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("CritChanceLucky", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"skillForkCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ForkedCount", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"skillChainCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("ChainCount", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"skillPierceCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("PiercedCount", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionAtCloseRange": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AtCloseRange", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyMoving": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Moving", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyFullLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:FullLife", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyLowLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:LowLife", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyCursed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Cursed", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyBleeding": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyPoisoned": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierPoisonOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("Multiplier:PoisonStack", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierWitheredStackCountSelf": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:WitheredStack", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierWitheredStackCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("Multiplier:WitheredStack", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierCorrosionStackCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:CorrosionStack", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("Armour", mod.TypeBase, -5000).Source("Corrosion").Tag(mod.Multiplier("CorrosionStack"), mod.ActorCondition("enemy", "CanCorrode")))
		enemyModList.AddMod(mod.NewFloat("Evasion", mod.TypeBase, -1000).Source("Corrosion").Tag(mod.Multiplier("CorrosionStack"), mod.ActorCondition("enemy", "CanCorrode")))
	},
	"multiplierEnsnaredStackCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:EnsnareStackCount", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:Moving", true).Source("Config").Tag(mod.MultiplierThreshold("EnsnareStackCount").Actor("enemy").Threshold(1)))
	},
	"conditionEnemyMaimed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Maimed", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyHindered": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Hindered", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyBlinded": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Blinded", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"overrideBuffBlinded": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("BlindEffect", mod.TypeOverride, v).Source("Config").Tag(mod.GlobalEffect("Buff")))
	},
	"conditionEnemyTaunted": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Taunted", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyDebilitated": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Debilitated", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyMalignantMadness": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:MalignantMadness", true).Source("Config"))
	},
	"conditionEnemyBurning": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyIgnited": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Ignited", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyScorched": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:ScorchedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionScorchedEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("ScorchVal", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.Condition("ScorchedConfig")))
		enemyModList.AddMod(mod.NewFloat("DesiredScorchVal", mod.TypeBase, configFloat(val)).Source("Scorch").Tag(mod.Condition("ScorchedConfig").Neg(true)))
	},
	"ScorchStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:ScorchStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyOnScorchedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:OnScorchedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyChilled": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:ChilledConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1881-1884
	"conditionEnemyChilledEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("ChillVal", mod.TypeBase, configFloat(val)).Source("Chill").Tag(mod.Condition("ChilledConfig")))
		enemyModList.AddMod(mod.NewFloat("DesiredChillVal", mod.TypeBase, configFloat(val)).Source("Chill").Tag(mod.Condition("ChilledConfig").Neg(true)))
	},
	"conditionEnemyChilledByYourHits": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:ChilledByYourHits", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"HoarfrostStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("HoarfrostFreezeDuration", mod.TypeIncrease, v*20).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierBarnacleStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:BarnacleStack", mod.TypeBase, math.Min(v, 10)).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("SelfFreezeDuration", mod.TypeIncrease, 5).Source("Config").Tag(mod.Multiplier("BarnacleStack"), mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("SelfChillDuration", mod.TypeIncrease, 5).Source("Config").Tag(mod.Multiplier("BarnacleStack"), mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageConvertToCold", mod.TypeBase, 5).Source("Config").Tag(mod.Multiplier("BarnacleStack"), mod.Condition("Effective")))
	},
	"conditionEnemyFrozen": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Frozen", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyBrittle": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Brittle", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:BrittleConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionBrittleEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("BrittleVal", mod.TypeBase, configFloat(val)).Source("Config").Tag(mod.Condition("BrittleConfig")))
		enemyModList.AddMod(mod.NewFloat("DesiredBrittleVal", mod.TypeBase, configFloat(val)).Source("Brittle").Tag(mod.Condition("BrittleConfig").Neg(true)))
	},
	"conditionEnemyOnBrittleGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Brittle", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:OnBrittleGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyShocked": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:ShockedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionShockEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("ShockVal", mod.TypeBase, configFloat(val)).Source("Shock").Tag(mod.Condition("ShockedConfig")))
		enemyModList.AddMod(mod.NewFloat("DesiredShockVal", mod.TypeBase, configFloat(val)).Source("Shock").Tag(mod.Condition("ShockedConfig").Neg(true)))
	},
	"ShockStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:ShockStacks", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyOnShockedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:OnShockedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemySapped": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Sapped", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:SappedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionSapEffect": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("SapVal", mod.TypeBase, configFloat(val)).Source("Sap").Tag(mod.Condition("SappedConfig")))
		enemyModList.AddMod(mod.NewFloat("DesiredSapVal", mod.TypeBase, configFloat(val)).Source("Sap").Tag(mod.Condition("SappedConfig").Neg(true)))
	},
	"conditionEnemyOnSappedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Sapped", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFlag("Condition:OnSappedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"multiplierFreezeShockIgniteOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:FreezeShockIgniteOnEnemy", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyFireExposure": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("FireExposure", mod.TypeBase, -10).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyFireExposure")))
	},
	"conditionEnemyColdExposure": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("ColdExposure", mod.TypeBase, -10).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyColdExposure")))
	},
	"conditionEnemyLightningExposure": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("LightningExposure", mod.TypeBase, -10).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyLightningExposure")))
	},
	"conditionChampionIntimidate": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:ChampionIntimidate", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionSacrificeMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SacrificeMinionOnAttack", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	"conditionEnemyIntimidated": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Intimidated", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyCrushed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Crushed", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyUnnerved": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Unnerved", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyCoveredInAsh": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CoveredInAshEffect", mod.TypeBase, 20).Source("Covered in Ash"))
	},
	"conditionEnemyCoveredInFrost": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CoveredInFrostEffect", mod.TypeBase, 20).Source("Covered in Frost"))
	},
	"conditionEnemyOnConsecratedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:OnConsecratedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyOnProfaneGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:OnProfaneGround", true).Source("Config").Tag(mod.Condition("Effective")))
		enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeIncrease, 10).Source("Config").Tag(mod.Condition("OnProfaneGround")))
		enemyModList.AddMod(mod.NewFloat("SelfCritChance", mod.TypeIncrease, 100).Source("Config").Tag(mod.Condition("OnProfaneGround")))
	},
	"multiplierEnemyAffectedByGraspingVines": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:GraspingVinesAffectingEnemy", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	"maniaDebuffsCount": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configDefaultState("maniaDebuffsCount", val)
		if !ok {
			return
		}
		v = math.Min(v, 15)
		source := mod.Source(strconv.FormatFloat(v, 'f', -1, 64) + " Mania Stacks")
		for _, m := range []struct {
			name  string
			value float64
		}{
			{"DamageTaken", v * 4},
			{"ActionSpeed", -v * 2},
			{"LifeRecoveryRate", -v * 10},
			{"EnergyShieldRecoveryRate", -v * 10},
		} {
			enemyModList.AddMod(mod.NewFloat(m.name, mod.TypeIncrease, m.value).Source(source).
				Tag(mod.GlobalEffect("Debuff"), mod.Condition("AfflictedByMania")))
		}
	},
	"conditionEnemyOnFungalGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:OnFungalGround", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyOnBrineGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:OnBrineGround", true).Source("Config").Tag(mod.Condition("Effective")))
		for _, name := range []string{"SelfChillEffect", "SelfBrittleEffect", "SelfShockEffect", "SelfSapEffect"} {
			enemyModList.AddMod(mod.NewFloat(name, mod.TypeIncrease, 30).Source("Config").Tag(mod.Condition("OnBrineGround")))
		}
		for _, name := range []string{"Armour", "Evasion"} {
			enemyModList.AddMod(mod.NewFloat(name, mod.TypeIncrease, -25).Source("Config").Tag(mod.Condition("OnBrineGround")))
		}
	},
	"conditionEnemyInChillingArea": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:InChillingArea", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"conditionEnemyInFrostGlobe": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:EnemyInFrostGlobe", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"enemyConditionHitByFireDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HitByFireDamage", true).Source("Config"))
	},
	"enemyConditionHitByColdDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HitByColdDamage", true).Source("Config"))
	},
	"enemyConditionHitByLightningDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HitByLightningDamage", true).Source("Config"))
	},
	"conditionEnemyRareOrUnique": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	"deliriousPercentage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if val == "20Percent" {
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -16).Source("20% Delirious"))
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 6).Source("20% Delirious"))
		}
		if val == "40Percent" {
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -32).Source("40% Delirious"))
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 12).Source("40% Delirious"))
		}
		if val == "60Percent" {
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -48).Source("60% Delirious"))
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 18).Source("60% Delirious"))
		}
		if val == "80Percent" {
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -64).Source("80% Delirious"))
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 24).Source("80% Delirious"))
		}
		if val == "100Percent" {
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -80).Source("100% Delirious"))
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 30).Source("100% Delirious"))
		}
	},
	"enemyPhysicalReduction": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, configFloat(val)).Source("EnemyConfig"))
	},
	"enemyLightningResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("LightningResist", mod.TypeBase, configFloat(val)).Source("EnemyConfig"))
	},
	"enemyColdResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("ColdResist", mod.TypeBase, configFloat(val)).Source("EnemyConfig"))
	},
	"enemyFireResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("FireResist", mod.TypeBase, configFloat(val)).Source("EnemyConfig"))
	},
	"enemyChaosResist": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, configFloat(val)).Source("EnemyConfig"))
	},
	"enemyEvasion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("Evasion", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	"enemyArmour": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("Armour", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2337-2339
	"enemyMultiplierPvpDamage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("MultiplierPvpDamage", mod.TypeBase, v).Source("Config"))
	},
	"enemyIsBoss": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		name, ok := val.(string)
		if !ok {
			return
		}
		switch normalizeEnemyIsBoss(name) {
		case "Boss":
			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 488).Source("Boss"))
			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))
			modList.AddMod(mod.NewFloat("Multiplier:EnemyPower", mod.TypeBase, 20).Source("Boss"))
		case "Pinnacle":
			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
			enemyModList.AddMod(mod.NewFlag("Condition:PinnacleBoss", true).Source("Config").Tag(mod.Condition("Effective")))
			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 404).Source("Boss"))
			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))
			modList.AddMod(mod.NewFloat("Multiplier:EnemyPower", mod.TypeBase, 20).Source("Boss"))
		case "Uber":
			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
			enemyModList.AddMod(mod.NewFlag("Condition:PinnacleBoss", true).Source("Config").Tag(mod.Condition("Effective")))
			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -70).Source("Boss"))
			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 404).Source("Boss"))
			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))
			modList.AddMod(mod.NewFloat("Multiplier:EnemyPower", mod.TypeBase, 20).Source("Boss"))
		}
	},
	//pob:block:src/Classes/ConfigTab.lua:1113-1128
	"customMods": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		text, ok := val.(string)
		if !ok {
			return
		}
		for _, line := range strings.Split(text, "\n") {
			entry := ParseCustomModLine(line)
			if entry == nil || entry.ModList == nil || entry.Extra != "" {
				continue
			}
			for _, m := range entry.ModList {
				modList.AddMod(mod.SetSource(m.Clone(), "Custom"))
			}
		}
	},
	//pob:block:src/Modules/ConfigOptions.lua:2300-2317
	//pob:block:src/Modules/ConfigOptions.lua:180-182
	"conditionFullMana": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:FullMana", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:195-197
	"conditionUnbrokenWard": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UnbrokenWard", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:316-318
	"bloodSacramentReservationEHP": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BloodSacramentReservationEHP", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:333-335
	"BrandsInLastHalf": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:BrandLastHalf", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:340-342
	"carrionGolemEqualsChaosGolem": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CarrionEqualChaosGolem", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:343-345
	"chaosGolemEqualsStoneGolem": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:ChaosEqualStoneGolem", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:346-348
	"cinderflameStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:CinderflameStacks", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:349-351
	"stoneGolemEqualsCarrionGolem": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StoneEqualCarrionGolem", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:467-469
	"harbingerOfTimeSlipstream": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HarbingerOfTime", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:487-489
	"inventionMineTrapPlacedDuration": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:PlacedDuration", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:524-526
	"minionPactLife": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:MinionLife", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:536-538
	"enemyHitMistyReflection": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:MistyReflection", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:543-545
	"MomentumSwiftnessStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:MomentumStacksRemoved", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:559-561
	"physicalAegisDepleted": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:PhysicalAegisDepleted", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:584-586
	"conditionSummonedSpectreInPast8Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SummonedSpectreInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:609-611
	"conditionSummonedZombieInPast8Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SummonedZombieInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:688-690
	"weepingBlackStacks": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:WeepingBlackStacks", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:716-718
	"conditionInsane": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Insane", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:745-747
	"multiplierWoCExpiredDuration": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:WoCDurationExpired", mod.TypeBase, min(v, 100)).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:748-750
	"absolutionSkillDamageCountedOnce": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:AbsolutionSkillDamageCountedOnce", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:751-753
	"dominatingBlowSkillDamageCountedOnce": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:DominatingBlowSkillDamageCountedOnce", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:754-756
	"holyStrikeSkillDamageCountedOnce": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HolyStrikeSkillDamageCountedOnce", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1019-1021
	"buffSacrificialZeal": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SacrificialZeal", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1040-1042
	"conditionChangedStanceLastSecond": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StanceChangeLastSecond", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1102-1104
	"conditionUsingTincture": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsingTincture", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1134-1136
	"multiplierPermanentMinion": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:PermanentMinion", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1152-1156
	"minionConditionOnBrineGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:OnBrineGround", true).Source("Config").Tag(mod.Condition("Combat"))}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, 10).Source("Config").Tag(mod.Condition("OnBrineGround"))}).Source("Config"))
		modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, 10).Source("Config").Tag(mod.Condition("OnBrineGround"))}).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1157-1159
	"conditionOnCausticGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnCausticGround", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1167-1170
	"conditionOnChilledGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnChilledGround", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1171-1174
	"conditionOnShockedGround": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:OnShockedGround", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1184-1186
	"conditionScorched": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1193-1195
	"conditionFrozen": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Frozen", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1196-1198
	"conditionBrittle": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Brittle", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1205-1207
	"conditionSapped": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:Sapped", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1214-1216
	"conditionCanBeCurseImmune": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("AvoidCurse", mod.TypeBase, 100).Source("Config").Tag(mod.Condition("Combat"), mod.GlobalEffect("Global").Unscalable(true)))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1283-1285
	"conditionKilledTauntedEnemyRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:KilledTauntedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1337-1339
	"multiplierIgniteAppliedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:IgniteAppliedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1361-1363
	"conditionWardBrokenPast2Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:WardBrokenPast2Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1390-1393
	"conditionHitByFireDamageRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitByFireDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1394-1397
	"conditionHitByColdDamageRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitByColdDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1398-1401
	"conditionHitByLightningDamageRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:HitByLightningDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1421-1423
	"conditionEnergyShieldRechargeRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:EnergyShieldRechargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1433-1439
	"buffPendulum": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		switch configString(val) {
		case "AREA":
			modList.AddMod(mod.NewFlag("Condition:PendulumOfDestructionAreaOfEffect", true).Source("Config").Tag(mod.Condition("Combat")))
		case "DAMAGE":
			modList.AddMod(mod.NewFlag("Condition:PendulumOfDestructionElementalDamage", true).Source("Config").Tag(mod.Condition("Combat")))
		}
	},
	//pob:block:src/Modules/ConfigOptions.lua:1451-1456
	"highestDamageType": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if v := configString(val); v != "" && v != "NONE" {
			modList.AddMod(mod.NewFlag("Condition:"+v+"IsHighestDamageType", true).Source("Config"))
			modList.AddMod(mod.NewFlag("IsHighestDamageTypeOVERRIDE", true).Source("Config"))
		}
	},
	//pob:block:src/Modules/ConfigOptions.lua:1496-1498
	"multiplierAppliedAilmentsRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:AppliedAilmentsRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1502-1504
	"conditionStunnedWhileCastingRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:StunnedWhileCastingRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1505-1507
	"conditionCastLast1Seconds": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastLast1Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1514-1517
	"multiplierHitsSuppressedRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:HitsSuppressedRecently", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:SuppressedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1518-1521
	"conditionUsedFireSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedFireSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1522-1525
	"conditionUsedColdSkillRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:UsedColdSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1535-1540
	"conditionUsedDashRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastDashRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedTravelSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1578-1580
	"TriggeredTrapsRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:TriggeredTrapsRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1593-1595
	"conditionCastMarkRecently": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:CastMarkRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1609-1611
	"conditionRavenousCorpseConsumed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:RavenousCorpseConsumed", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1634-1636
	"BlockedPast10Sec": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:BlockedPast10Sec", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Combat")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1673-1675
	"GamblesprintMovementSpeed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v := configFloat(val)
		if text := configString(val); text != "" {
			var err error
			if v, err = strconv.ParseFloat(text, 64); err != nil {
				return
			}
		}
		modList.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, v).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("HaveGamblesprint")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1683-1685
	"conditionHitsAlwaysStun": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFloat("CullPercent", mod.TypeMAX, 10).Source("Config").Tag(mod.Condition("Combat"), mod.Condition("maceMasteryStunCullSpecced")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1783-1785
	"conditionEnemyStunned": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Stunned", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1789-1791
	"overrideBleedStackPotential": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("BleedStackPotentialOverride", mod.TypeOverride, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1792-1794
	"conditionSingleBleed": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:SingleBleed", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1807-1809
	"multiplierCurseExpiredOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:CurseExpired", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1810-1812
	"multiplierCurseDurationExpiredOnEnemy": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:CurseDurationExpired", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1831-1833
	"conditionEnemyExcommunicated": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Excommunicated", true).Source("Config").Tag(mod.ActorCondition("enemy", "CanExcommunicate")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1846-1848
	"conditionEnemyPacified": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:Pacified", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1962-1964
	"conditionEnemyHallowingFlame": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HallowingFlame", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1965-1967
	"multiplierEnemyHallowingFlame": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		enemyModList.AddMod(mod.NewFloat("Multiplier:HallowingFlame", mod.TypeBase, v).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1968-1970
	"multiplierHallowingFlameStacksRemovedByAlly": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("Multiplier:HallowingFlameStacksRemovedByAlly", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:1971-1973
	"conditionHallowingFlameMagnitude": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFloat("HallowingFlameMagnitude", mod.TypeOverride, configFloat(val)).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2019-2021
	"conditionEnemyLifeHigherThanPlayer": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:HigherLifePercentThanPlayer", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2031-2033
	"enemyInRFOrScorchingRay": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		modList.AddMod(mod.NewFlag("Condition:InRFOrScorchingRay", true).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2035-2037
	"conditionBetweenYouAndLinkedTarget": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFlag("Condition:BetweenYouAndLinkedTarget", true).Source("Config").Tag(mod.Condition("Effective")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2038-2040
	"conditionEnemyFireResZero": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("FireResist", mod.TypeOverride, 0).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2041-2043
	"conditionEnemyColdResZero": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("ColdResist", mod.TypeOverride, 0).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2044-2046
	"conditionEnemyLightningResZero": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		if !configChecked(val) {
			return
		}
		enemyModList.AddMod(mod.NewFloat("LightningResist", mod.TypeOverride, 0).Source("Config").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	},
	//pob:block:src/Modules/ConfigOptions.lua:2237-2239
	"enemyBlockChance": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("BlockChance", mod.TypeBase, configFloat(val)).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:591-593
	"raiseSpectreKaomFireBeamTotemStage": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewFloat("Multiplier:KaomFireBeamTotemStage", mod.TypeBase, v).Source("Config"))
	},
	//pob:block:src/Modules/ConfigOptions.lua:659-661
	"stormRainBeamOverlap": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
		v, ok := configCount(val)
		if !ok {
			return
		}
		modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "beamOverlapMultiplier", Value: v}).Source("Config").Tag(mod.SkillName("Storm Rain")))
	},
}

//pob:src/Data/ModMap.lua:16-23,src/Data/ModMap.lua:60-68,src/Data/ModMap.lua:155-162,src/Data/ModMap.lua:204-222,src/Data/ModMap.lua:297-306,src/Data/ModMap.lua:346-359
var mapAffixCheckData = map[string]func(modList *moddb.ModList, enemyModList *moddb.ModList){
	"Hexproof": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFlag("Hexproof", true).Source("Map mod Hexproof"))
	},
	"Unstoppable": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		// MISSING: Monsters cannot be Taunted
		enemyModList.AddMod(MOD("MinimumActionSpeed", "MAX", 100).Source("Map mod Unstoppable"))
	},
	"Conflagrating": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("IgniteChance", mod.TypeBase, 100).Source("Map mod Conflagrating"))
		enemyModList.AddMod(mod.NewFlag("AllDamageIgnites", true).Source("Map mod Conflagrating"))
	},
	"of Balance": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		// Players cannot inflict Exposure
	},
	"of Congealment": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFlag("CannotLeechLifeFromSelf", true).Source("Map mod of Congealment"))
		enemyModList.AddMod(mod.NewFlag("CannotLeechManaFromSelf", true).Source("Map mod of Congealment"))
		enemyModList.AddMod(mod.NewFlag("CannotLeechEnergyShieldFromSelf", true).Source("Map mod of Congealment"))
	},
	"of Stasis": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		modList.AddMod(mod.NewFlag("NoLifeRegen", true).Source("Map mod of Stasis"))
		modList.AddMod(mod.NewFlag("NoEnergyShieldRegen", true).Source("Map mod of Stasis"))
		modList.AddMod(mod.NewFlag("NoManaRegen", true).Source("Map mod of Stasis"))
	},
	"of Blinding": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
	},
	"of Venom": func(modList *moddb.ModList, enemyModList *moddb.ModList) {
		enemyModList.AddMod(mod.NewFloat("PoisonChance", mod.TypeBase, 100).Source("Map mod of Venom"))
	},
}

func rollRanged(lo, hi, rollRange float64) float64 {
	return lo + (hi-lo)*rollRange/100
}

//pob:src/Data/ModMap.lua:7-368
var mapAffixListData = map[string]func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList){
	"Armoured": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{20, 30, 40}
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Armoured"))
	},
	"Hexwarded": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{25, 40, 60}
		enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod Hexwarded"))
	},
	"Resistant": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{20, 15}, {30, 20}, {40, 25}}
		enemyModList.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, values[val-1][0]*mapModEffect).Source("Map mod Resistant"))
		enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, values[val-1][1]*mapModEffect).Source("Map mod Resistant"))
	},
	"Impervious": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{20, 35, 50}
		enemyModList.AddMod(mod.NewFloat("AvoidPoison", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Impervious"))
		enemyModList.AddMod(mod.NewFloat("AvoidImpale", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Impervious"))
		enemyModList.AddMod(mod.NewFloat("AvoidBleed", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Impervious"))
	},
	"Oppressive": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{30, 45, 60}
		enemyModList.AddMod(mod.NewFloat("SpellSuppressionChance", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Oppressive"))
	},
	"Titan's": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{25, 45}, {30, 55}, {35, 70}}
		enemyModList.AddMod(mod.NewFloat("Life", mod.TypeMore, values[val-1][0]*mapModEffect).Source("Map mod Titan's").Tag(mod.Condition("RareOrUnique")))
		enemyModList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, values[val-1][1]*mapModEffect).Source("Map mod Titan's").Tag(mod.Condition("RareOrUnique")))
	},
	"Impaling": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{25, 40, 60}
		enemyModList.AddMod(mod.NewFloat("ImpaleChance", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Impaling").Flag(mod.MFlagAttack))
	},
	"Empowered": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{0, 15, 20}
		enemyModList.AddMod(mod.NewFloat("ElementalAilmentChance", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod Empowered"))
	},
	"Overlord's": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{15, 20}, {20, 25}, {25, 30}}
		enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, values[val-1][0]*mapModEffect).Source("Map mod Overlord's").Tag(mod.Condition("RareOrUnique")))
		enemyModList.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, values[val-1][1]*mapModEffect).Source("Map mod Overlord's").Tag(mod.Condition("RareOrUnique")))
	},
	"Punishing": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {},
	"Mirrored":  func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {},
	"of Drought": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{30, 40, 50}
		modList.AddMod(mod.NewFloat("FlaskChargesGained", mod.TypeIncrease, -values[val-1]*mapModEffect).Source("Map mod of Drought"))
	},
	"of Impotence": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{15, 20, 25}
		modList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod of Impotence"))
	},
	"of Insulation": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{30, 50, 70}
		enemyModList.AddMod(mod.NewFloat("AvoidElementalAilments", mod.TypeBase, values[val-1]*mapModEffect).Source("Map mod of Insulation"))
	},
	"of Miring": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{10, 30}, {15, 40}, {20, 50}}
		modList.AddMod(mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, -values[val-1][0]*mapModEffect).Source("Map mod of Miring"))
		enemyModList.AddMod(mod.NewFloat("Accuracy", mod.TypeIncrease, values[val-1][1]*mapModEffect).Source("Map mod of Miring"))
	},
	"of Rust": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{20, 20}, {30, 25}, {40, 30}}
		modList.AddMod(mod.NewFloat("BlockChance", mod.TypeIncrease, -values[val-1][0]*mapModEffect).Source("Map mod of Rust"))
		modList.AddMod(mod.NewFloat("Armour", mod.TypeMore, -values[val-1][1]*mapModEffect).Source("Map mod of Rust"))
	},
	"of Smothering": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{20, 40, 60}
		modList.AddMod(mod.NewFloat("LifeRecoveryRate", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod of Smothering"))
		modList.AddMod(mod.NewFloat("EnergyShieldRecoveryRate", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod of Smothering"))
	},
	"of Fatigue": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{20, 30, 40}
		modList.AddMod(mod.NewFloat("CooldownRecovery", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod of Fatigue"))
	},
	"of Transience": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {},
	"of Doubt": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{25, 40, 60}
		modList.AddMod(mod.NewFloat("AuraEffect", mod.TypeIncrease, -values[val-1]*mapModEffect).Source("Map mod of Doubt").
			Tag(mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)))
	},
	"of Imprecision": func(val int, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := []float64{15, 20, 25}
		modList.AddMod(mod.NewFloat("Accuracy", mod.TypeMore, -values[val-1]*mapModEffect).Source("Map mod of Imprecision"))
	},
}

//pob:src/Data/ModMap.lua:7-368
var mapAffixCountData = map[string]func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList){
	"Unwavering": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{15, 19}, {20, 24}, {25, 30}}
		enemyModList.AddMod(mod.NewFloat("AvoidStun", mod.TypeBase, 100).Source("Map mod Unwavering"))
		enemyModList.AddMod(mod.NewFloat("Life", mod.TypeMore, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Unwavering"))
	},
	"Fecund": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{20, 29}, {30, 39}, {40, 49}}
		enemyModList.AddMod(mod.NewFloat("Life", mod.TypeMore, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Fecund"))
	},
	"Buffered": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{20, 29}, {30, 39}, {40, 49}}
		enemyModList.AddMod(mod.NewFloat("LifeGainAsEnergyShield", mod.TypeBase, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Buffered"))
	},
	"Savage": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{14, 17}, {18, 21}, {22, 25}}
		enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Savage"))
	},
	"Burning": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{50, 69}, {70, 89}, {90, 110}}
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Burning"))
	},
	"Freezing": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{50, 69}, {70, 89}, {90, 110}}
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Freezing"))
	},
	"Shocking": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{50, 69}, {70, 89}, {90, 110}}
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Shocking"))
	},
	"Profane": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{0, 0}, {21, 25}, {31, 35}}
		enemyModList.AddMod(mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Profane"))
		modList.AddMod(mod.NewFlag("Condition:CanBeWithered", true).Source("Map mod Profane"))
	},
	"Fleet": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{20, 25}, {25, 35}, {35, 45}}
		// attack and cast is the same so applying it once, movespeed does not matter
		enemyModList.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod Fleet"))
	},
	"of Exposure": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{0, 0}, {5, 8}, {9, 12}}
		if values[val-1][1] != 0 {
			roll := rollRanged(values[val-1][0], values[val-1][1], rollRange) * mapModEffect
			modList.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, -roll).Source("Map mod of Exposure"))
			modList.AddMod(mod.NewFloat("ColdResistMax", mod.TypeBase, -roll).Source("Map mod of Exposure"))
			modList.AddMod(mod.NewFloat("LightningResistMax", mod.TypeBase, -roll).Source("Map mod of Exposure"))
			modList.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, -roll).Source("Map mod of Exposure"))
		}
	},
	"of Toughness": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2]float64{{25, 30}, {31, 35}, {36, 40}}
		enemyModList.AddMod(mod.NewFloat("SelfCritMultiplier", mod.TypeIncrease, -rollRanged(values[val-1][0], values[val-1][1], rollRange)*mapModEffect).Source("Map mod of Toughness"))
	},
	"of Deadliness": func(val int, rollRange, mapModEffect float64, modList *moddb.ModList, enemyModList *moddb.ModList) {
		values := [][2][2]float64{{{160, 200}, {30, 35}}, {{260, 300}, {36, 40}}, {{360, 400}, {41, 45}}}
		enemyModList.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, rollRanged(values[val-1][0][0], values[val-1][0][1], rollRange)*mapModEffect).Source("Map mod of Deadliness"))
		enemyModList.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, rollRanged(values[val-1][1][0], values[val-1][1][1], rollRange)*mapModEffect).Source("Map mod of Deadliness"))
	},
}

func mapModTierIndex(build *pob.PathOfBuilding) int {
	switch build.GetStringOption("multiplierMapModTier") {
	case "MED":
		return 2
	case "LOW":
		return 3
	}
	return 1
}

//pob:src/Modules/ConfigOptions.lua:117-130
func mapAffixDropDownFunction(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList, build *pob.PathOfBuilding) {
	v := configString(val)
	if v == "NONE" {
		return
	}
	mapModEffect := 1 + build.GetNumberOption("multiplierMapModEffect")/100
	tier := 4 - mapModTierIndex(build)
	if apply, ok := mapAffixCheckData[v]; ok {
		apply(modList, enemyModList)
	} else if apply, ok := mapAffixListData[v]; ok {
		apply(tier, mapModEffect, modList, enemyModList)
	} else if apply, ok := mapAffixCountData[v]; ok {
		apply(tier, 100, mapModEffect, modList, enemyModList)
	}
}

var configurationsWithBuild = map[string]func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList, build *pob.PathOfBuilding){
	"MapPrefix1": mapAffixDropDownFunction,
	"MapPrefix2": mapAffixDropDownFunction,
	"MapPrefix3": mapAffixDropDownFunction,
	"MapPrefix4": mapAffixDropDownFunction,
	"MapSuffix1": mapAffixDropDownFunction,
	"MapSuffix2": mapAffixDropDownFunction,
	"MapSuffix3": mapAffixDropDownFunction,
	"MapSuffix4": mapAffixDropDownFunction,
	//pob:block:src/Modules/ConfigOptions.lua:2246-2317
	"presetBossSkills": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList, build *pob.PathOfBuilding) {
		preset := configString(val)
		if preset == "" || preset == "None" {
			return
		}

		bossData, known := data.BossSkills[preset]
		if !known {
			return
		}
		isUber := bossSkillIsUber(build, bossData)

		modList.AddMod(mod.NewFlag("BossSkillActive", true).Source("Config"))

		if preset == "Atziri Flameblast" && isUber {
			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 60).Source("Alluring Abyss Map Mod"))
		}
		if bossData.AdditionalStats == nil {
			return
		}
		additionalStats := bossData.AdditionalStats.Base
		if isUber && bossData.AdditionalStats.Uber != nil {
			additionalStats = bossData.AdditionalStats.Uber
		}

		names := make([]string, 0, len(additionalStats))
		for name := range additionalStats {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			stat := additionalStats[name]
			if stat.IsFlag {
				enemyModList.AddMod(mod.NewFlag(name, true).Source("BossSkillAdditionalData"))
				continue
			}
			enemyModList.AddMod(mod.NewFloat(name, mod.TypeBase, stat.Value).Source("BossSkillAdditionalData"))
		}
	},
}

//pob:block:src/Classes/ConfigTab.lua:258-260
func normalizeEnemyIsBoss(val string) string {
	words := strings.Fields(strings.ToLower(val))
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}

	if len(words) == 0 {
		return "Pinnacle"
	}

	switch joined := strings.Join(words, " "); joined {
	case "Uber Atziri":
		return "Boss"
	case "Shaper":
		return "Pinnacle"
	case "Sirus":
		return "Pinnacle"
	default:
		return joined
	}
}

//pob:src/Classes/ConfigTab.lua:481-491
func enemyIsBossLevel(build *pob.PathOfBuilding, placeholder map[string]float64) int {
	level := build.Build.Level
	if input := build.GetNumberOption("enemyLevel"); input > 0 {
		level = int(input)
	} else if saved := placeholder["enemyLevel"]; saved > 0 {
		level = int(saved)
	}
	return max(1, min(data.MaxEnemyLevel, level))
}

func ConfigPlaceholders(build *pob.PathOfBuilding) map[string]float64 {
	return enemyConfigPlaceholders(build)
}

//pob:block:src/Modules/ConfigOptions.lua:1457-1589
func enemyConfigPlaceholders(build *pob.PathOfBuilding) map[string]float64 {
	placeholder := make(map[string]float64, len(configDefaultPlaceholders))
	for name, value := range configDefaultPlaceholders {
		placeholder[name] = value
	}

	for _, saved := range build.Config.Placeholders {
		if saved.Number != nil {
			placeholder[saved.Name] = *saved.Number
		}
	}

	round := func(val float64) float64 {
		return math.Floor(float64(val) + 0.5)
	}

	isBoss := normalizeEnemyIsBoss(build.GetStringOption("enemyIsBoss"))
	if isBoss != "" {
		placeholder["enemySpeed"] = 700
		placeholder["enemyCritChance"] = 5
		placeholder["enemyCritDamage"] = 30
	}

	switch isBoss {
	case "None":
		delete(placeholder, "enemyLightningResist")
		delete(placeholder, "enemyColdResist")
		delete(placeholder, "enemyFireResist")
		delete(placeholder, "enemyChaosResist")

		delete(placeholder, "enemyLevel")
		defaultLevel := enemyIsBossLevel(build, placeholder)

		defaultDamage := round(data.MonsterDamageTable[defaultLevel] * 1.5)
		placeholder["enemyPhysicalDamage"] = defaultDamage
		delete(placeholder, "enemyLightningDamage")
		delete(placeholder, "enemyColdDamage")
		delete(placeholder, "enemyFireDamage")
		delete(placeholder, "enemyChaosDamage")

		delete(placeholder, "enemyLightningPen")
		delete(placeholder, "enemyColdPen")
		delete(placeholder, "enemyFirePen")

		placeholder["enemyArmour"] = data.MonsterArmourTable[defaultLevel]
		placeholder["enemyEvasion"] = data.MonsterEvasionTable[defaultLevel]
	case "Boss":
		defaultEleResist := float64(40)
		placeholder["enemyLightningResist"] = defaultEleResist
		placeholder["enemyColdResist"] = defaultEleResist
		placeholder["enemyFireResist"] = defaultEleResist
		placeholder["enemyChaosResist"] = 25

		delete(placeholder, "enemyLevel")
		defaultLevel := enemyIsBossLevel(build, placeholder)

		defaultDamage := round(data.MonsterDamageTable[defaultLevel] * 1.5 * data.StdBossDPSMult)
		placeholder["enemyPhysicalDamage"] = defaultDamage
		placeholder["enemyLightningDamage"] = defaultDamage
		placeholder["enemyColdDamage"] = defaultDamage
		placeholder["enemyFireDamage"] = defaultDamage
		placeholder["enemyChaosDamage"] = round(defaultDamage / 2.5)

		delete(placeholder, "enemyLightningPen")
		delete(placeholder, "enemyColdPen")
		delete(placeholder, "enemyFirePen")

		placeholder["enemyArmour"] = data.MonsterArmourTable[defaultLevel]
		placeholder["enemyEvasion"] = data.MonsterEvasionTable[defaultLevel]
	case "Pinnacle":
		defaultEleResist := float64(50)
		placeholder["enemyLightningResist"] = defaultEleResist
		placeholder["enemyColdResist"] = defaultEleResist
		placeholder["enemyFireResist"] = defaultEleResist
		placeholder["enemyChaosResist"] = 30

		placeholder["enemyLevel"] = 84
		defaultLevel := max(enemyIsBossLevel(build, placeholder), 84)

		defaultDamage := round(data.MonsterDamageTable[defaultLevel] * 1.5 * data.PinnacleBossDPSMult)
		placeholder["enemyPhysicalDamage"] = defaultDamage
		placeholder["enemyLightningDamage"] = defaultDamage
		placeholder["enemyColdDamage"] = defaultDamage
		placeholder["enemyFireDamage"] = defaultDamage
		placeholder["enemyChaosDamage"] = round(defaultDamage / 2.5)

		placeholder["enemyLightningPen"] = data.PinnacleBossPen
		placeholder["enemyColdPen"] = data.PinnacleBossPen
		placeholder["enemyFirePen"] = data.PinnacleBossPen

		placeholder["enemyArmour"] = round(data.MonsterArmourTable[defaultLevel] * (data.BossStats.PinnacleArmourMean / 100))
		placeholder["enemyEvasion"] = round(data.MonsterEvasionTable[defaultLevel] * (data.BossStats.PinnacleEvasionMean / 100))
	case "Uber":
		defaultEleResist := float64(50)
		placeholder["enemyLightningResist"] = defaultEleResist
		placeholder["enemyColdResist"] = defaultEleResist
		placeholder["enemyFireResist"] = defaultEleResist
		placeholder["enemyChaosResist"] = 30

		placeholder["enemyLevel"] = 85
		defaultLevel := max(enemyIsBossLevel(build, placeholder), 85)

		defaultDamage := round(data.MonsterDamageTable[defaultLevel] * 1.5 * data.UberBossDPSMult)
		placeholder["enemyPhysicalDamage"] = defaultDamage
		placeholder["enemyLightningDamage"] = defaultDamage
		placeholder["enemyColdDamage"] = defaultDamage
		placeholder["enemyFireDamage"] = defaultDamage
		placeholder["enemyChaosDamage"] = round(defaultDamage / 4)

		placeholder["enemyLightningPen"] = data.UberBossPen
		placeholder["enemyColdPen"] = data.UberBossPen
		placeholder["enemyFirePen"] = data.UberBossPen

		placeholder["enemyArmour"] = round(data.MonsterArmourTable[defaultLevel] * (data.BossStats.UberArmourMean / 100))
		placeholder["enemyEvasion"] = round(data.MonsterEvasionTable[defaultLevel] * (data.BossStats.UberEvasionMean / 100))
	}

	applyPresetBossSkills(build, placeholder)

	return placeholder
}

//pob:block:src/Modules/ConfigOptions.lua:1637-1700
func applyPresetBossSkills(build *pob.PathOfBuilding, placeholder map[string]float64) {
	preset := build.GetStringOption("presetBossSkills")
	if preset == "" || preset == "None" {
		return
	}

	bossData, known := data.BossSkills[preset]
	if !known {
		return
	}

	round := func(val float64) float64 {
		return math.Floor(float64(val) + 0.5)
	}

	for _, name := range []string{
		"enemyPhysicalDamage", "enemyLightningDamage", "enemyColdDamage", "enemyFireDamage",
		"enemyChaosDamage",
	} {
		delete(placeholder, name)
	}

	rollRangeMult := placeholder["enemyDamageRollRange"]
	if input, present := build.GetNumberOptionOK("enemyDamageRollRange"); present {
		rollRangeMult = input
	}
	rollRangeMult = min(max(rollRangeMult, 0), 100)

	isUber := bossSkillIsUber(build, bossData)
	damageLevel := enemyIsBossLevel(build, placeholder)
	for damageType, damageMult := range bossData.DamageMultipliers {
		if isUber && bossData.UberDamageMultiplier.Set {
			placeholder["enemy"+damageType+"Damage"] = round(
				data.MonsterDamageTable[damageLevel] * (damageMult[0] + float64(rollRangeMult*damageMult[1])) * bossData.UberDamageMultiplier.Value)
		} else {
			placeholder["enemy"+damageType+"Damage"] = round(
				data.MonsterDamageTable[damageLevel] * (damageMult[0] + float64(rollRangeMult*damageMult[1])))
		}
	}

	for _, name := range []string{
		"enemyPhysicalOverwhelm", "enemyLightningPen", "enemyColdPen", "enemyFirePen",
	} {
		delete(placeholder, name)
	}

	for penType, pen := range bossData.DamagePenetrations {
		name := "enemy" + penType
		if uberPen, ok := bossData.UberDamagePenetrations[penType]; isUber && ok && uberPen.Set {
			placeholder[name] = uberPen.Value
			continue
		}
		if !pen.Set {
			delete(placeholder, name)
			continue
		}
		placeholder[name] = pen.Value
	}

	if isUber && bossData.UberSpeed.Set {
		placeholder["enemySpeed"] = bossData.UberSpeed.Value
	} else if bossData.Speed.Set {
		placeholder["enemySpeed"] = bossData.Speed.Value
	}
	if bossData.CritChance.Set {
		placeholder["enemyCritChance"] = bossData.CritChance.Value
	}
}

//pob:block:src/Modules/ConfigOptions.lua:2249-2252
func bossSkillIsUber(build *pob.PathOfBuilding, bossData data.BossSkill) bool {
	isBoss := normalizeEnemyIsBoss(build.GetStringOption("enemyIsBoss"))
	return isBoss == "Uber" || (bossData.EarlierUber && isBoss == "Pinnacle")
}

func summonElementalRelicAura(val interface{}, modList *moddb.ModList, skillID string) {
	if !configChecked(val) {
		return
	}
	modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "Enable", Value: 1}).Source("Config").
		Tag(mod.SkillId(skillID), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
}

//pob:block:src/Classes/ConfigTab.lua:1106-1129
func applyCustomModifierBlocks(build *pob.PathOfBuilding, modList *moddb.ModList) *moddb.ModList {
	for _, block := range build.GetCustomModifierBlocks() {
		if !block.IsEnabled() || block.Text == "" {
			continue
		}
		source := mod.Source("Custom:" + block.BlockTitle())
		for _, line := range strings.Split(block.Text, "\n") {
			entry := ParseCustomModLine(line)
			if entry == nil || entry.ModList == nil || entry.Extra != "" {
				continue
			}
			for _, m := range entry.ModList {
				modList.AddMod(mod.SetSource(m.Clone(), source))
			}
		}
	}
	return modList
}

var (
	escapeDigitPattern = regexp.MustCompile(`\^\d`)
	escapeHexPattern   = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}`)
)

//pob:src/HeadlessWrapper.lua:75-77
func StripEscapes(text string) string {
	return escapeHexPattern.ReplaceAllString(escapeDigitPattern.ReplaceAllString(text, ""), "")
}

//pob:block:src/Classes/ConfigTab.lua:1114-1115
func ParseCustomModLine(line string) *ModCacheEntry {
	return ParseMod(strings.Trim(StripEscapes(line), " \t\n\v\f\r"), false)
}
