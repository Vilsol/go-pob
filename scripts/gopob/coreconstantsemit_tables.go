package main

var ccnMisc = [][3]string{
	{"ServerTickTime", "ServerTickTime", ""},
	{"BlockChanceCap", "BlockChanceCap", ""},
	{"EnemyPhysicalDamageReductionCap", "EnemyPhysicalDamageReductionCap", ""},
	{"FortifyBaseDuration", "FortifyBaseDuration", ""},
	{"ManaRegenBase", "ManaRegenBase", ""},
	{"MatchingSocketQualityBonus", "MatchingSocketQualityBonus", ""},
	{"PlayerMovementSpeed", "PlayerMovementSpeed", ""},
	{"maxHitSmoothingPasses", "MaxHitSmoothingPasses", ""},
	{"maxStatIncrease", "MaxStatIncrease", ""},
	{"PvpElemental1", "PvpElemental1", ""},
	{"PvpElemental2", "PvpElemental2", ""},
	{"PvpNonElemental1", "PvpNonElemental1", ""},
	{"PvpNonElemental2", "PvpNonElemental2", ""},
	{"TemporalChainsEffectCap", "TemporalChainsEffectCap", "float64"},
	{"DamageReductionCap", "DamageReductionCap", ""},
	{"ResistFloor", "ResistFloor", ""},
	{"MaxResistCap", "MaxResistCap", ""},
	{"EvadeChanceCap", "EvadeChanceCap", ""},
	{"DodgeChanceCap", "DodgeChanceCap", ""},
	{"SuppressionChanceCap", "SuppressionChanceCap", ""},
	{"SuppressionEffect", "SuppressionEffect", ""},
	{"AvoidChanceCap", "AvoidChanceCap", ""},
	{"EnergyShieldRechargeBase", "EnergyShieldRechargeBase", ""},
	{"EnergyShieldRechargeDelay", "EnergyShieldRechargeDelay", ""},
	{"WardRechargeDelay", "WardRechargeDelay", ""},
	{"Transfiguration", "Transfiguration", ""},
	{"EnemyMaxResist", "EnemyMaxResist", ""},
	{"LeechRateBase", "LeechRateBase", ""},
	{"BleedPercentBase", "BleedPercentBase", ""},
	{"BleedDurationBase", "BleedDurationBase", ""},
	{"PoisonPercentBase", "PoisonPercentBase", ""},
	{"PoisonDurationBase", "PoisonDurationBase", ""},
	{"IgnitePercentBase", "IgnitePercentBase", ""},
	{"IgniteDurationBase", "IgniteDurationBase", ""},
	{"ImpaleStoredDamageBase", "ImpaleStoredDamageBase", ""},
	{"BuffExpirationSlowCap", "BuffExpirationSlowCap", ""},
	{"TrapTriggerRadiusBase", "TrapTriggerRadiusBase", ""},
	{"MineDetonationRadiusBase", "MineDetonationRadiusBase", ""},
	{"MineAuraRadiusBase", "MineAuraRadiusBase", ""},
	{"MaxEnemyLevel", "MaxEnemyLevel", ""},
	{"LowPoolThreshold", "LowPoolThreshold", ""},
	{"AccuracyPerDexBase", "AccuracyPerDexBase", "float64"},
	{"BrandAttachmentRangeBase", "BrandAttachmentRangeBase", ""},
	{"ProjectileDistanceCap", "ProjectileDistanceCap", ""},
	{"stdBossDPSMult", "StdBossDPSMult", ""},
	{"pinnacleBossDPSMult", "PinnacleBossDPSMult", ""},
	{"pinnacleBossPen", "PinnacleBossPen", ""},
	{"uberBossDPSMult", "UberBossDPSMult", ""},
	{"uberBossPen", "UberBossPen", ""},
	{"ehpCalcSpeedUp", "EhpCalcSpeedUp", ""},
	{"StunBaseMult", "StunBaseMult", ""},
	{"StunBaseDuration", "StunBaseDuration", ""},
	{"StunNotMeleeDamageMult", "StunNotMeleeDamageMult", ""},
	{"MinStunChanceNeeded", "MinStunChanceNeeded", ""},
	{"ehpCalcMaxDamage", "EhpCalcMaxDamage", ""},
	{"ehpCalcMaxIterationsToCalc", "EhpCalcMaxIterationsToCalc", ""},
	{"DotDpsCap", "DotDpsCap", ""},
}

var ccnMiscRemoved = [][4]string{
	{"IgniteMinDuration", "IgniteMinDuration", "0.3", "REMOVED IN PoB 3.29 -- 0 occurrences in src/. CalcOffence.lua:4904 now computes IgniteDuration with NO minimum clamp. calculator/calc_offence.go:3624 still applies one, so we floor a duration upstream lets fall. Re-port that site and delete this entry; it needs a 3.29 oracle to grade, which is why it is frozen rather than dropped."},
}

var ccnMiscUnported = map[string]string{
	"ServerTickRate":                    "computed in constants.go as a RUNTIME float64 division; data/misc_ulp_test.go pins that form against the const fold",
	"experiencePenaltyMultiplier":       "character XP curve; go-pob computes no experience at all",
	"maxExperiencePenaltyFreeAreaLevel": "character XP curve; go-pob computes no experience at all",
}

var ccnBossStats = []string{"PinnacleArmourMean", "PinnacleEvasionMean", "UberArmourMean", "UberEvasionMean"}

var ccnWeaponClass = [][2]string{
	{"None", "None"},
	{"Bow", "Bow"},
	{"Claw", "Claw"},
	{"Dagger", "Dagger"},
	{"Staff", "Staff"},
	{"Wand", "Wand"},
	{"One Handed Axe", "OneHandAxe"},
	{"One Handed Mace", "OneHandMace"},
	{"One Handed Sword", "OneHandSword"},
	{"Sceptre", "Sceptre"},
	{"Thrusting One Handed Sword", "ThrustingOneHandSword"},
	{"Fishing Rod", "FishingRod"},
	{"Two Handed Axe", "TwoHandAxe"},
	{"Two Handed Mace", "TwoHandMace"},
	{"Two Handed Sword", "TwoHandSword"},
}

var ccnWeaponModFlag = map[string]string{
	"Unarmed": "mod.MFlagUnarmed",
	"Bow":     "mod.MFlagBow",
	"Claw":    "mod.MFlagClaw",
	"Dagger":  "mod.MFlagDagger",
	"Staff":   "mod.MFlagStaff",
	"Wand":    "mod.MFlagWand",
	"Axe":     "mod.MFlagAxe",
	"Mace":    "mod.MFlagMace",
	"Sword":   "mod.MFlagSword",
	"Fishing": "mod.MFlagFishing",
}

var ccnUnarmedClass = []string{"Scion", "Marauder", "Ranger", "Witch", "Duelist", "Templar", "Shadow"}

var ccnAilment = [][2]string{
	{"Chill", "AilmentChill"},
	{"Freeze", "AilmentFreeze"},
	{"Shock", "AilmentShock"},
	{"Scorch", "AilmentScorch"},
	{"Brittle", "AilmentBrittle"},
	{"Sap", "AilmentSap"},
}

var ccnDamageTypeConst = map[string]string{
	"Physical":  "DamageTypePhysical",
	"Lightning": "DamageTypeLightning",
	"Cold":      "DamageTypeCold",
	"Fire":      "DamageTypeFire",
	"Chaos":     "DamageTypeChaos",
}

var ccnMonsterTables = [][2]string{
	{"monsterEvasionTable", "MonsterEvasionTable"},
	{"monsterAccuracyTable", "MonsterAccuracyTable"},
	{"monsterLifeTable", "MonsterLifeTable"},
	{"monsterLifeTable2", "MonsterLifeTable2"},
	{"monsterLifeTable3", "MonsterLifeTable3"},
	{"monsterAllyLifeTable", "MonsterAllyLifeTable"},
	{"monsterDamageTable", "MonsterDamageTable"},
	{"monsterAllyDamageTable", "MonsterAllyDamageTable"},
	{"monsterArmourTable", "MonsterArmourTable"},
	{"monsterPhysConversionMultiTable", "MonsterPhysConversionMultiTable"},
}

var ccnLevelMultTables = [][2]string{
	{"mapLevelLifeMult", "MapLevelLifeMult"},
}

const ccnCharacterConstantKeyCount = 89

const ccnHeader = "package data\n\nimport (\n\t\"github.com/Vilsol/go-pob/mod\"\n\t\"github.com/Vilsol/go-pob/pob\"\n\t\"github.com/Vilsol/go-pob/utils\"\n)\n\n// The core constant tables of PoB's src/Modules/Data.lua.\n//\n// CODE GENERATED BY oracle/core_constants_emit.py FROM PoB {version} (tree {tree}). DO NOT EDIT.\n// Regenerate with `task data-regen -- --only core-constants` after a PoB pin bump.\n//\n// ⚠️ THIS IS THE PoB-DERIVED HALF OF data/constants.go, WHICH IS A SPLIT AND NOT A MIGRATION IN\n// PROGRESS. The other half — the DamageType / Ailment / SkillType / ItemClassName enums, the\n// WeaponTypeInfo / JewelRadiusData / NonDamagingAilmentData type declarations, ServerTickRate's\n// runtime division, and the two ours-only EHP bounds 3.20 replaced — has NO literal PoB\n// counterpart to generate from and stays hand-written next door. Adding a Go-native declaration\n// here, or a PoB table there, is what makes the boundary stop meaning anything.\n//\n// The previous revision of every table below was a hand transcription with no emitter, which is\n// how three of them went stale unnoticed for months: monsterAllyLifeTable (rebalanced at every\n// index from 2), the nonDamagingAilment Brittle row (default 5->2, max 15->6) and\n// WardRechargeDelay (5 vs 4). Each read as a calculator defect and was disproved there first.\n\n"
