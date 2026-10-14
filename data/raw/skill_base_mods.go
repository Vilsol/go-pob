package raw

import "github.com/Vilsol/go-pob/mod"

// SkillBaseMods is PoB's hand-written per-skill `baseMods` table
// (src/Data/Skills/*.lua), keyed by granted-effect id. PoB applies it verbatim at the end
// of calcs.mergeSkillInstanceMods (src/Modules/CalcActiveSkill.lua:69,
// `modList:AddList(grantedEffect.baseMods)`) for every gem effect — active gems and
// supports alike — and go-pob-data carries none of it, so this table is its only source.
//
// A baseMods entry is either the `skill(key, value)` helper, which
// src/Modules/Data.lua:60 expands to mod("SkillData", "LIST", {key=, value=}), or an
// ordinary mod() with its own type, flags, keywordFlags and tags. Both shapes are here.
// Every mod carries source "Skill:<id>", which src/Modules/Data.lua:64 (processMod) sets
// and which CalcActiveSkill's global-effect pass compares against to decide whether a
// buff inherits the skill's buffNotPlayer/buffMinions/buffAllies/allowTotemBuff config.
//
// Generated from PoB v2.18.1 (493 skills with baseMods, 982 mods) by
// oracle/skill_base_mods.lua + oracle/skill_base_mods_emit.py; 977 mods are emitted,
// 5 are not representable and are omitted:
//   - 3 string-valued skill() entries (triggerSource, corpseMonsterVariety) whose key has
//     no string field on calculator.SkillData
//   - 1 mod.flags holds a tag table (upstream arity bug), so the mod has no tags and a nonsense flag set
//   - 1 skill() entry whose key has no field on calculator.SkillData
//     (chanceToTriggerCounterattackOnBlock), which utils.Set drops as an invalid key rather than storing
//
// SkillPart-tagged mods are ALL carried, for every part and for PoB's `skillPartList`
// form: moddb evaluates the tag (ModStore.lua:529-549 -> moddb/mod_store.go
// evalSkillPartTag) with PoB's drop-when-not-matching semantics, so a mod written for
// part 3 reaches only a build on part 3. The earlier revision of this table kept part 1
// alone and dropped 51 part-gated mods (its header said 55, which was its own tally's
// classification, not a count of mods), because the tag was then a TODO and keeping them
// would have applied every part of a skill at once; that reason is gone. It also flattened
// a `skillPartList` containing 1 down to `mod.SkillPart(1)` — Magma Sigil's showAverage and
// Wild Strike's fire conversion, both {1, 2} — which silently dropped the second part.
//
// One tag is deliberately not emitted rather than omitted with its mod: ChargedDash's
// hitTimeMultiplier carries `{type = "Skill", skillPartList = {1, 2}}`
// (src/Data/Skills/act_dex.lua:1838), and ModStore.lua has no "Skill" branch, so upstream
// the tag gates nothing and the mod always applies. The mod is emitted untagged to match.
var SkillBaseMods = map[string][]mod.Mod{
	// src/Data/Skills/spectre.lua:9185
	"ABTTAzmeriGoddessAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:ABTTAzmeriGoddessAura"),
	},
	// src/Data/Skills/spectre.lua:9984
	"ABTTAzmeriSpiderLeaderAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:ABTTAzmeriSpiderLeaderAura"),
	},
	// src/Data/Skills/act_str.lua:35
	"Absolution": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:Absolution"),
	},
	// src/Data/Skills/act_str.lua:131
	"AbsolutionAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:AbsolutionAltX"),
	},
	// src/Data/Skills/act_dex.lua:40
	"AlchemistsMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:AlchemistsMark"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:AlchemistsMark"),
	},
	// src/Data/Skills/act_str.lua:308
	"AncestralCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:AncestralCry"),
	},
	// src/Data/Skills/act_str.lua:405
	"Anger": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Anger"),
	},
	// src/Data/Skills/act_dex.lua:314
	"AnimateWeaponAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseMainHandWeapon", Value: 1}).Source("Skill:AnimateWeaponAltX"),
	},
	// src/Data/Skills/act_int.lua:357
	"ArcaneCloak": {
		mod.NewFloat("GuardAbsorbLimit", mod.TypeBase, 100).Tag(mod.PercentStatVar("ManaUnreserved", "ArcaneCloakConsumedMana"), mod.GlobalEffect("Guard").Unscalable(true)).Source("Skill:ArcaneCloak"),
		mod.NewFloat("Multiplier:ManaSpentRecently", mod.TypeBase, 100).Tag(mod.PercentStatVar("ManaUnreserved", "ArcaneCloakConsumedMana"), mod.Condition("ArcaneCloakUsedRecently"), mod.GlobalEffect("Buff").Unscalable(true)).Source("Skill:ArcaneCloak"),
	},
	// src/Data/Skills/act_int.lua:729
	"ArmageddonBrand": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 8}).Source("Skill:ArmageddonBrand"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ArmageddonBrand"),
	},
	// src/Data/Skills/act_int.lua:909
	"ArmageddonBrandAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ArmageddonBrandAltY"),
	},
	// src/Data/Skills/act_dex.lua:768
	"ArtilleryBallista": {
		mod.NewFlag("OneShotProj", true).Source("Skill:ArtilleryBallista"),
	},
	// src/Data/Skills/act_dex.lua:853
	"ArtilleryBallistaAltX": {
		mod.NewFlag("OneShotProj", true).Source("Skill:ArtilleryBallistaAltX"),
	},
	// src/Data/Skills/act_dex.lua:939
	"ArtilleryBallistaAltY": {
		mod.NewFlag("OneShotProj", true).Source("Skill:ArtilleryBallistaAltY"),
	},
	// src/Data/Skills/other.lua:368
	"AspectOfTheSpider": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:AspectOfTheSpider"),
		mod.NewList("SkillData", mod.SkillData{Key: "StackCount", Value: 1}).Tag(multiplierLimitVar("SpiderWebApplyStack", "SpiderWebApplyStackMax")).Source("Skill:AspectOfTheSpider"),
	},
	// src/Data/Skills/act_int.lua:1010
	"AssassinsMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:AssassinsMark"),
	},
	// src/Data/Skills/other.lua:1958
	"AtziriUniqueStaffFlameblast": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:AtziriUniqueStaffFlameblast"),
	},
	// src/Data/Skills/other.lua:3988
	"AtziriUniqueStaffStormCall": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:AtziriUniqueStaffStormCall"),
	},
	// src/Data/Skills/spectre.lua:23
	"AxisCasterGlacialCascade": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:AxisCasterGlacialCascade"),
	},
	// src/Data/Skills/spectre.lua:74
	"AxisDoubleStrikeTrigger": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:AxisDoubleStrikeTrigger"),
	},
	// src/Data/Skills/spectre.lua:7809
	"AzmeriAdmiralPrecision": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriAdmiralPrecision"),
	},
	// src/Data/Skills/spectre.lua:8028
	"AzmeriBarrageDemonSpinestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Spine explosion:"}).Source("Skill:AzmeriBarrageDemonSpinestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:AzmeriBarrageDemonSpinestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Spines fall:"}).Source("Skill:AzmeriBarrageDemonSpinestorm"),
	},
	// src/Data/Skills/spectre.lua:9381
	"AzmeriBirdGrace": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriBirdGrace"),
	},
	// src/Data/Skills/spectre.lua:8513
	"AzmeriCasterDemonSpellDamageAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriCasterDemonSpellDamageAura"),
	},
	// src/Data/Skills/spectre.lua:8584
	"AzmeriCycloneDemonDesecratedGroundCyclone": {
		mod.NewFloat("CooldownRecovery", mod.TypeOverride, 0).Source("Skill:AzmeriCycloneDemonDesecratedGroundCyclone"),
	},
	// src/Data/Skills/spectre.lua:11072
	"AzmeriDemonPhysicalDamageAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriDemonPhysicalDamageAura"),
	},
	// src/Data/Skills/spectre.lua:7229
	"AzmeriFirefuryFireResistAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriFirefuryFireResistAura"),
	},
	// src/Data/Skills/spectre.lua:9221
	"AzmeriGoddessDiscipline": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriGoddessDiscipline"),
	},
	// src/Data/Skills/spectre.lua:7025
	"AzmeriHailrakeColdResistAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriHailrakeColdResistAura"),
	},
	// src/Data/Skills/spectre.lua:7344
	"AzmeriHydraHatred": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriHydraHatred"),
	},
	// src/Data/Skills/spectre.lua:7583
	"AzmeriKudukuWrath": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriKudukuWrath"),
	},
	// src/Data/Skills/spectre.lua:9667
	"AzmeriOakVitality": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriOakVitality"),
	},
	// src/Data/Skills/spectre.lua:10408
	"AzmeriOversoulRocksTriggered": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Rock explosion:"}).Source("Skill:AzmeriOversoulRocksTriggered"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:AzmeriOversoulRocksTriggered"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Rocks fall:"}).Source("Skill:AzmeriOversoulRocksTriggered"),
	},
	// src/Data/Skills/spectre.lua:9508
	"AzmeriPhantasmClarity": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriPhantasmClarity"),
	},
	// src/Data/Skills/spectre.lua:7919
	"AzmeriSwordStormCascade": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Sword explosion:"}).Source("Skill:AzmeriSwordStormCascade"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:AzmeriSwordStormCascade"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Swords fall:"}).Source("Skill:AzmeriSwordStormCascade"),
	},
	// src/Data/Skills/spectre.lua:10309
	"AzmeriTigerHaste": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriTigerHaste"),
	},
	// src/Data/Skills/spectre.lua:10370
	"AzmeriTurtleDetermination": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:AzmeriTurtleDetermination"),
	},
	// src/Data/Skills/spectre.lua:10567
	"AzmeriVikingCyclone": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:AzmeriVikingCyclone"),
	},
	// src/Data/Skills/spectre.lua:10009
	"AzmeriZombieCausticGroundWhenHit": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:AzmeriZombieCausticGroundWhenHit"),
		mod.NewFlag("dotIsCausticGround", true).Source("Skill:AzmeriZombieCausticGroundWhenHit"),
	},
	// src/Data/Skills/act_int.lua:1177
	"BallLightning": {
		mod.NewList("SkillData", mod.SkillData{Key: "StrikeInterval", Value: 0.15}).Source("Skill:BallLightning"),
		mod.NewList("SkillData", mod.SkillData{Key: "ProjectileSpeed", Value: 40}).Source("Skill:BallLightning"),
		mod.NewList("SkillData", mod.SkillData{Key: "Duration", Value: 2}).Source("Skill:BallLightning"),
		mod.NewFlag("CannotSplit", true).Source("Skill:BallLightning"),
	},
	// src/Data/Skills/act_int.lua:1296
	"BallLightningAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "StrikeInterval", Value: 0.3}).Tag(mod.SkillPart(2)).Source("Skill:BallLightningAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "StrikeInterval", Value: 0.15}).Tag(mod.SkillPart(3)).Source("Skill:BallLightningAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "ProjectileSpeed", Value: 40}).Source("Skill:BallLightningAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Duration", Value: 2}).Source("Skill:BallLightningAltX"),
		mod.NewFlag("CannotSplit", true).Source("Skill:BallLightningAltX"),
	},
	// src/Data/Skills/act_int.lua:1397
	"BallLightningAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "StrikeInterval", Value: 0.15}).Source("Skill:BallLightningAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Duration", Value: 2}).Source("Skill:BallLightningAltY"),
	},
	// src/Data/Skills/spectre.lua:152
	"BanditChampionBlastRainSpectre": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:BanditChampionBlastRainSpectre"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 4}).Source("Skill:BanditChampionBlastRainSpectre"),
	},
	// src/Data/Skills/spectre.lua:116
	"BanditExplosiveArrowAtAnimationSpeed": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:BanditExplosiveArrowAtAnimationSpeed"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BanditExplosiveArrowAtAnimationSpeed"),
		mod.NewFloat("Multiplier:ExplosiveArrowFuse", mod.TypeBase, 1).Source("Skill:BanditExplosiveArrowAtAnimationSpeed"),
	},
	// src/Data/Skills/act_int.lua:1505
	"Bane": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Bane"),
	},
	// src/Data/Skills/act_int.lua:1678
	"BaneAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:BaneAltX"),
	},
	// src/Data/Skills/act_str.lua:802
	"BattlemagesCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:BattlemagesCry"),
	},
	// src/Data/Skills/act_dex.lua:1384
	"BladeBlast": {
		mod.NewFloat("Multiplier:BladeBlastMaxStages", mod.TypeBase, 50).Source("Skill:BladeBlast"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsBaseMultiplier", Value: 1}).Tag(mod.Multiplier("BladeBlastStage")).Source("Skill:BladeBlast"),
	},
	// src/Data/Skills/act_dex.lua:1485
	"BladeBlastAltX": {
		mod.NewFloat("Multiplier:BladeBlastofUnloadingMaxStages", mod.TypeBase, 10).Source("Skill:BladeBlastAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsBaseMultiplier", Value: 1}).Tag(mod.Multiplier("BladeBlastofUnloadingStage")).Source("Skill:BladeBlastAltX"),
	},
	// src/Data/Skills/act_dex.lua:1938
	"BladeFlurry": {
		mod.NewList("SkillData", mod.SkillData{Key: "NumStages", Value: 1}).Tag(mod.Multiplier("BladeFlurryStage")).Source("Skill:BladeFlurry"),
		mod.NewList("SkillData", mod.SkillData{Key: "StackMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:BladeFlurry"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:BladeFlurry"),
	},
	// src/Data/Skills/act_dex.lua:2055
	"BladeFlurryAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "NumStages", Value: 1}).Tag(mod.Multiplier("BladeFlurryofIncisionStage")).Source("Skill:BladeFlurryAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "StackMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:BladeFlurryAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:BladeFlurryAltX"),
	},
	// src/Data/Skills/act_dex.lua:2248
	"BladeVortex": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:BladeVortex"),
		mod.NewList("SkillData", mod.SkillData{Key: "HitFrequency", Value: 0.6}).Source("Skill:BladeVortex"),
		mod.NewFloat("Multiplier:BladeVortexBlade", mod.TypeBase, 5).Tag(mod.SkillPart(2)).Source("Skill:BladeVortex"),
		mod.NewFloat("Multiplier:BladeVortexBlade", mod.TypeBase, 10).Tag(mod.SkillPart(3)).Source("Skill:BladeVortex"),
		mod.NewFlag("Condition:HaveBladeVortex", true).Source("Skill:BladeVortex"),
	},
	// src/Data/Skills/act_dex.lua:2346
	"BladeVortexAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:BladeVortexAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "HitTimeOverride", Value: 0.6}).Source("Skill:BladeVortexAltX"),
		mod.NewFlag("Condition:HaveBladeVortex", true).Source("Skill:BladeVortexAltX"),
	},
	// src/Data/Skills/act_dex.lua:2539
	"Bladefall": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 44}).Source("Skill:Bladefall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Volley Width:"}).Source("Skill:Bladefall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 12}).Source("Skill:Bladefall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Volley Length:"}).Source("Skill:Bladefall"),
	},
	// src/Data/Skills/act_str.lua:1047
	"Bladestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:Bladestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of initial attack:"}).Source("Skill:Bladestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:Bladestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area of bladestorms:"}).Source("Skill:Bladestorm"),
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Flag(mod.MFlagAttack).Tag(mod.Condition("BloodStance"), mod.SkillPart(2)).Source("Skill:Bladestorm"),
	},
	// src/Data/Skills/act_str.lua:1155
	"BladestormAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:BladestormAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of initial attack:"}).Source("Skill:BladestormAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:BladestormAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area of bladestorms:"}).Source("Skill:BladestormAltX"),
	},
	// src/Data/Skills/act_dex.lua:2903
	"BlastRain": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:BlastRain"),
		mod.NewFlag("OneShotProj", true).Source("Skill:BlastRain"),
	},
	// src/Data/Skills/act_int.lua:1833
	"BlazingSalvo": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:BlazingSalvo"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Minimum Range:"}).Source("Skill:BlazingSalvo"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 22}).Source("Skill:BlazingSalvo"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Maximum Range:"}).Source("Skill:BlazingSalvo"),
		mod.NewFlag("CannotSplit", true).Source("Skill:BlazingSalvo"),
	},
	// src/Data/Skills/act_int.lua:1927
	"Blight": {
		mod.NewFloat("Damage", mod.TypeMore, 100).Tag(mod.Multiplier("BlightStageAfterFirst")).Source("Skill:Blight"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Blight"),
		mod.NewList("SkillData", mod.SkillData{Key: "DebuffSecondary", Value: 1}).Source("Skill:Blight"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:Blight"),
	},
	// src/Data/Skills/act_int.lua:2023
	"BlightAltX": {
		mod.NewFloat("Damage", mod.TypeMore, 100).Tag(mod.Multiplier("BlightofContagionStageAfterFirst")).Source("Skill:BlightAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:BlightAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "DebuffSecondary", Value: 1}).Source("Skill:BlightAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:BlightAltX"),
	},
	// src/Data/Skills/act_int.lua:2119
	"BlightAltY": {
		mod.NewFloat("Damage", mod.TypeMore, 100).Tag(mod.Multiplier("BlightofAtrophyStageAfterFirst")).Source("Skill:BlightAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:BlightAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "DebuffSecondary", Value: 1}).Source("Skill:BlightAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:BlightAltY"),
	},
	// src/Data/Skills/act_dex.lua:3070
	"BlinkArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:BlinkArrow"),
	},
	// src/Data/Skills/act_dex.lua:3156
	"BlinkArrowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:BlinkArrowAltX"),
	},
	// src/Data/Skills/act_dex.lua:3241
	"BlinkArrowAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:BlinkArrowAltY"),
	},
	// src/Data/Skills/other.lua:511
	"BloodOffering": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:BloodOffering"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:BloodOffering"),
	},
	// src/Data/Skills/act_dex.lua:3335
	"BloodRage": {
		mod.NewList("SkillData", mod.SkillData{Key: "ThisIsNotABuff", Value: 1}).Source("Skill:BloodRage"),
	},
	// src/Data/Skills/other.lua:566
	"BloodSacramentUnique": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 5}).Source("Skill:BloodSacramentUnique"),
		mod.NewFloat("Multiplier:BloodSacramentMaxStages", mod.TypeBase, 33).Source("Skill:BloodSacramentUnique"),
		mod.NewFlag("HasUncancellableReservation", true).Tag(mod.Condition("BloodSacramentReservationEHP")).Source("Skill:BloodSacramentUnique"),
	},
	// src/Data/Skills/act_int.lua:2325
	"Bodyswap": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:Bodyswap"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:Bodyswap"),
	},
	// src/Data/Skills/act_int.lua:2448
	"BodyswapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:BodyswapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:BodyswapAltX"),
	},
	// src/Data/Skills/other.lua:614
	"BoneArmour": {
		mod.NewFloat("AvoidBleed", mod.TypeBase, 100).Tag(mod.GlobalEffect("Guard")).Source("Skill:BoneArmour"),
	},
	// src/Data/Skills/minion.lua:1508
	"BoneGolemCascade": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BoneGolemCascade"),
	},
	// src/Data/Skills/minion.lua:1591
	"BoneGolemCascadeEmpowered": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BoneGolemCascadeEmpowered"),
	},
	// src/Data/Skills/minion.lua:1546
	"BoneGolemLeapSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BoneGolemLeapSlam"),
	},
	// src/Data/Skills/act_int.lua:2535
	"BoneOffering": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:BoneOffering"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:BoneOffering"),
	},
	// src/Data/Skills/act_str.lua:1344
	"Boneshatter": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Tag(mod.SkillPart(2)).Source("Skill:Boneshatter"),
		mod.NewFlag("HasTrauma", true).Source("Skill:Boneshatter"),
	},
	// src/Data/Skills/act_str.lua:1457
	"BoneshatterAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Tag(mod.SkillPart(2)).Source("Skill:BoneshatterAltX"),
		mod.NewFlag("HasTrauma", true).Source("Skill:BoneshatterAltX"),
	},
	// src/Data/Skills/act_str.lua:1555
	"BoneshatterAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Tag(mod.SkillPart(2)).Source("Skill:BoneshatterAltY"),
	},
	// src/Data/Skills/act_int.lua:561
	"BrandSupport": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:BrandSupport"),
	},
	// src/Data/Skills/spectre.lua:6543
	"BreachBlizzardSpectre": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:BreachBlizzardSpectre"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:BreachBlizzardSpectre"),
	},
	// src/Data/Skills/spectre.lua:4510
	"BreachLightningOrbsCommander": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BreachLightningOrbsCommander"),
	},
	// src/Data/Skills/spectre.lua:4419
	"BreachLightningWhip": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:BreachLightningWhip"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BreachLightningWhip"),
	},
	// src/Data/Skills/spectre.lua:4482
	"BreachTeamWarp": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:BreachTeamWarp"),
	},
	// src/Data/Skills/other.lua:2365
	"CatsStealthTriggeredIntimidatingCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:CatsStealthTriggeredIntimidatingCry"),
	},
	// src/Data/Skills/act_dex.lua:3643
	"CausticArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:CausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "AoE Explosion:"}).Source("Skill:CausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Caustic Ground:"}).Source("Skill:CausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:CausticArrow"),
		mod.NewFlag("dotIsCausticGround", true).Source("Skill:CausticArrow"),
	},
	// src/Data/Skills/act_dex.lua:3732
	"CausticArrowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "AoE Explosion:"}).Source("Skill:CausticArrowAltX"),
	},
	// src/Data/Skills/act_str.lua:1640
	"ChainHook": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:ChainHook"),
	},
	// src/Data/Skills/act_str.lua:1728
	"ChainHookAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:ChainHookAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Strike area:"}).Source("Skill:ChainHookAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Chain length:"}).Source("Skill:ChainHookAltY"),
	},
	// src/Data/Skills/act_dex.lua:17702
	"ChannelledSnipeSupport": {
		mod.NewFlag("TriggeredBySnipe", true).Source("Skill:ChannelledSnipeSupport"),
	},
	// src/Data/Skills/act_dex.lua:3963
	"ChargedDash": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:ChargedDash"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Start of Dash:"}).Source("Skill:ChargedDash"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 26}).Source("Skill:ChargedDash"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "End of Dash:"}).Source("Skill:ChargedDash"),
		mod.NewList("SkillData", mod.SkillData{Key: "HitTimeMultiplier", Value: 2}).Source("Skill:ChargedDash"),
		mod.NewFloat("Multiplier:ChargedDashMaxStages", mod.TypeBase, 15).Source("Skill:ChargedDash"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(3)).Source("Skill:ChargedDash"),
	},
	// src/Data/Skills/act_dex.lua:4088
	"ChargedDashAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:ChargedDashAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Start of Dash:"}).Source("Skill:ChargedDashAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 26}).Source("Skill:ChargedDashAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "End of Dash:"}).Source("Skill:ChargedDashAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "HitTimeMultiplier", Value: 2}).Source("Skill:ChargedDashAltX"),
		mod.NewFloat("Multiplier:ChargedDashofProjectionMaxStages", mod.TypeBase, 15).Source("Skill:ChargedDashAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(3)).Source("Skill:ChargedDashAltX"),
	},
	// src/Data/Skills/act_int.lua:2673
	"Clarity": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Clarity"),
	},
	// src/Data/Skills/act_str.lua:1814
	"Cleave": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:Cleave"),
	},
	// src/Data/Skills/act_str.lua:1902
	"CleaveAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:CleaveAltX"),
	},
	// src/Data/Skills/act_int.lua:2825
	"ColdSnap": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:ColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Burst:"}).Source("Skill:ColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Initial Chilled Ground:"}).Source("Skill:ColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Final Chilled Ground:"}).Source("Skill:ColdSnap"),
	},
	// src/Data/Skills/act_str.lua:6979
	"Combust": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:Combust"),
	},
	// src/Data/Skills/act_int.lua:3096
	"Conductivity": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Conductivity"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Conductivity"),
	},
	// src/Data/Skills/act_str.lua:2099
	"ConsecratedPath": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 23}).Source("Skill:ConsecratedPath"),
	},
	// src/Data/Skills/act_str.lua:2193
	"ConsecratedPathAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 23}).Source("Skill:ConsecratedPathAltX"),
	},
	// src/Data/Skills/act_int.lua:3174
	"Contagion": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Contagion"),
	},
	// src/Data/Skills/act_int.lua:3248
	"ContagionAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ContagionAltX"),
	},
	// src/Data/Skills/act_int.lua:3322
	"ContagionAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ContagionAltY"),
	},
	// src/Data/Skills/other.lua:925
	"Convocation": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:Convocation"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:Convocation"),
	},
	// src/Data/Skills/act_str.lua:2270
	"CorruptingFever": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:CorruptingFever"),
		mod.NewFlag("dotIsCorruptingBlood", true).Source("Skill:CorruptingFever"),
		mod.NewFloat("Multiplier:CorruptingFeverMaxStages", mod.TypeBase, 10).Source("Skill:CorruptingFever"),
		mod.NewFloat("Damage", mod.TypeMore, 100).Flag(mod.MFlagDot).Tag(mod.Multiplier("CorruptingFeverStageAfterFirst")).Source("Skill:CorruptingFever"),
	},
	// src/Data/Skills/act_int.lua:3480
	"CracklingLance": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:CracklingLance"),
	},
	// src/Data/Skills/act_int.lua:3559
	"CracklingLanceAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:CracklingLanceAltX"),
	},
	// src/Data/Skills/act_int.lua:3634
	"CracklingLanceAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:CracklingLanceAltY"),
	},
	// src/Data/Skills/act_int.lua:3712
	"CreepingFrost": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:CreepingFrost"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Projectile Impact:"}).Source("Skill:CreepingFrost"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "DoT Area:"}).Source("Skill:CreepingFrost"),
	},
	// src/Data/Skills/act_int.lua:3798
	"CreepingFrostAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:CreepingFrostAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Projectile Impact:"}).Source("Skill:CreepingFrostAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "DoT Area:"}).Source("Skill:CreepingFrostAltX"),
	},
	// src/Data/Skills/act_dex.lua:4388
	"Cremation": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:Cremation"),
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:Cremation"),
	},
	// src/Data/Skills/act_dex.lua:4506
	"CremationAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:CremationAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:CremationAltX"),
	},
	// src/Data/Skills/act_dex.lua:4610
	"CremationAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:CremationAltY"),
	},
	// src/Data/Skills/spectre.lua:6778
	"CrucibleIceStormTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Ice explosion:"}).Source("Skill:CrucibleIceStormTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:CrucibleIceStormTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Ice fall:"}).Source("Skill:CrucibleIceStormTrap"),
	},
	// src/Data/Skills/act_dex.lua:4728
	"Cyclone": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:Cyclone"),
	},
	// src/Data/Skills/act_dex.lua:4843
	"CycloneAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:CycloneAltX"),
	},
	// src/Data/Skills/minion.lua:833
	"DancingDervishCycloneChannelled": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusIsWeaponRange", Value: 1}).Source("Skill:DancingDervishCycloneChannelled"),
	},
	// src/Data/Skills/spectre.lua:7366
	"DarkMarionetteExplode": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "FireMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "PhysicalMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "PhysicalMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "LightningMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "LightningMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:DarkMarionetteExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:DarkMarionetteExplode"),
	},
	// src/Data/Skills/spectre.lua:7394
	"DarkMarionetteExplodePerfect": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "FireMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "PhysicalMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "PhysicalMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "LightningMin", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "LightningMax", Value: 1}).Tag(mod.PerStat(20, "Life")).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewFloat("EnemyScorchChance", mod.TypeBase, 100).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:DarkMarionetteExplodePerfect"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:DarkMarionetteExplodePerfect"),
	},
	// src/Data/Skills/act_int.lua:3923
	"DarkPact": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:DarkPact"),
	},
	// src/Data/Skills/act_int.lua:4038
	"DarkPactAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:DarkPactAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:DarkPactAltX"),
	},
	// src/Data/Skills/other.lua:1080
	"DeathAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:DeathAura"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:DeathAura"),
	},
	// src/Data/Skills/other.lua:1110
	"DeathWalk": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Source("Skill:DeathWalk"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:DeathWalk"),
	},
	// src/Data/Skills/other.lua:1188
	"DeathWish": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:DeathWish"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Tag(mod.SkillPart(2)).Source("Skill:DeathWish"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:DeathWish"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:DeathWish"),
	},
	// src/Data/Skills/act_str.lua:2431
	"DecoyTotem": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:DecoyTotem"),
	},
	// src/Data/Skills/act_str.lua:2514
	"DefianceBanner": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:DefianceBanner"),
		mod.NewFloat("AuraEffect", mod.TypeMore, -100).Tag(mod.Condition("BannerPlanted").Neg(true)).Source("Skill:DefianceBanner"),
		mod.NewFlag("Condition:AffectedByPlacedBanner", true).Tag(mod.Condition("BannerPlanted"), mod.GlobalEffect("Buff")).Source("Skill:DefianceBanner"),
	},
	// src/Data/Skills/spectre.lua:6696
	"DelveMonsterEnfeebleOnHit": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:DelveMonsterEnfeebleOnHit"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:DelveMonsterEnfeebleOnHit"),
	},
	// src/Data/Skills/spectre.lua:541
	"DemonModularBladeVortexSpectre": {
		mod.NewList("SkillData", mod.SkillData{Key: "HitTimeOverride", Value: 1}).Source("Skill:DemonModularBladeVortexSpectre"),
	},
	// src/Data/Skills/act_dex.lua:5098
	"Desecrate": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:Desecrate"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:Desecrate"),
	},
	// src/Data/Skills/act_int.lua:4128
	"Despair": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Despair"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Despair"),
	},
	// src/Data/Skills/act_str.lua:2598
	"Determination": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Determination"),
	},
	// src/Data/Skills/act_dex.lua:5192
	"DetonateDead": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:DetonateDead"),
	},
	// src/Data/Skills/act_dex.lua:5268
	"DetonateDeadAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Source("Skill:DetonateDeadAltX"),
	},
	// src/Data/Skills/act_dex.lua:5360
	"DetonateDeadAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:DetonateDeadAltY"),
	},
	// src/Data/Skills/act_int.lua:4452
	"Discipline": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Discipline"),
	},
	// src/Data/Skills/act_int.lua:4634
	"DivineIre": {
		mod.NewFloat("Multiplier:DivineIreMaxStages", mod.TypeBase, 10).Tag(mod.SkillPart(2)).Source("Skill:DivineIre"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 38}).Source("Skill:DivineIre"),
	},
	// src/Data/Skills/act_int.lua:4744
	"DivineIreAltX": {
		mod.NewFloat("Multiplier:DivineIreofHolyLightningMaxStages", mod.TypeBase, 10).Tag(mod.SkillPart(2)).Source("Skill:DivineIreAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 38}).Source("Skill:DivineIreAltX"),
	},
	// src/Data/Skills/act_int.lua:4839
	"DivineIreAltY": {
		mod.NewFloat("Multiplier:DivineIreofDisintegrationMaxStages", mod.TypeBase, 10).Source("Skill:DivineIreAltY"),
	},
	// src/Data/Skills/sup_int.lua:3521
	"DoomBlast": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:DoomBlast"),
	},
	// src/Data/Skills/act_dex.lua:5546
	"DoubleStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:DoubleStrike"),
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, -50).Source("Skill:DoubleStrike"),
	},
	// src/Data/Skills/act_dex.lua:5627
	"DoubleStrikeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:DoubleStrikeAltX"),
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, -50).Source("Skill:DoubleStrikeAltX"),
	},
	// src/Data/Skills/act_dex.lua:5717
	"DoubleStrikeAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:DoubleStrikeAltY"),
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, -50).Source("Skill:DoubleStrikeAltY"),
	},
	// src/Data/Skills/act_str.lua:3219
	"DreadBanner": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:DreadBanner"),
		mod.NewFloat("AuraEffect", mod.TypeMore, -100).Tag(mod.Condition("BannerPlanted").Neg(true)).Source("Skill:DreadBanner"),
		mod.NewFlag("Condition:AffectedByPlacedBanner", true).Tag(mod.Condition("BannerPlanted"), mod.GlobalEffect("Buff")).Source("Skill:DreadBanner"),
	},
	// src/Data/Skills/minion.lua:1447
	"DropBearSummonedRallyingCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:DropBearSummonedRallyingCry"),
	},
	// src/Data/Skills/act_str.lua:3326
	"Earthquake": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:Earthquake"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:Earthquake"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Tag(mod.SkillPart(2)).Source("Skill:Earthquake"),
	},
	// src/Data/Skills/act_str.lua:3431
	"EarthquakeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:EarthquakeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:EarthquakeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Tag(mod.SkillPart(2)).Source("Skill:EarthquakeAltX"),
	},
	// src/Data/Skills/act_str.lua:3645
	"Earthshatter": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:Earthshatter"),
	},
	// src/Data/Skills/act_str.lua:3753
	"EarthshatterAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:EarthshatterAltX"),
	},
	// src/Data/Skills/act_str.lua:3861
	"EarthshatterAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:EarthshatterAltY"),
	},
	// src/Data/Skills/act_dex.lua:6132
	"ElementalHit": {
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Ignited")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Chilled")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Frozen")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Shocked")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Scorched")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Brittle")).Source("Skill:ElementalHit"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Sapped")).Source("Skill:ElementalHit"),
	},
	// src/Data/Skills/act_dex.lua:6279
	"ElementalHitAltX": {
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Ignited")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Chilled")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Frozen")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Shocked")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Scorched")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Brittle")).Source("Skill:ElementalHitAltX"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Sapped")).Source("Skill:ElementalHitAltX"),
	},
	// src/Data/Skills/spectre.lua:579
	"ElementalHitSkeletonKnight": {
		mod.NewFlag("DealNoPhysical", true).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFlag("DealNoChaos", true).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFlag("DealNoCold", true).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFlag("DealNoLightning", true).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("AreaOfEffect", mod.TypeMore, 80).Tag(mod.ActorCondition("enemy", "Ignited")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Ignited")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Chilled")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Frozen")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Shocked")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Scorched")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Brittle")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Multiplier:ElementalHitAilmentOnEnemy", mod.TypeBase, 1).Tag(mod.ActorCondition("enemy", "Sapped")).Source("Skill:ElementalHitSkeletonKnight"),
		mod.NewFloat("Damage", mod.TypeMore, 10).Tag(mod.Multiplier("ElementalHitAilmentOnEnemy")).Source("Skill:ElementalHitSkeletonKnight"),
	},
	// src/Data/Skills/other.lua:1680
	"ElementalWarding": {
		mod.NewFloat("SkillPhysicalDamageConvertToFire", mod.TypeBase, 100).Tag(mod.SkillPart(1)).Source("Skill:ElementalWarding"),
		mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 100).Tag(mod.SkillPart(2)).Source("Skill:ElementalWarding"),
		mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 100).Tag(mod.SkillPart(3)).Source("Skill:ElementalWarding"),
	},
	// src/Data/Skills/act_int.lua:5012
	"ElementalWeakness": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ElementalWeakness"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:ElementalWeakness"),
	},
	// src/Data/Skills/spectre.lua:11489
	"EmpoweringHowlSpectre": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:EmpoweringHowlSpectre"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:EmpoweringHowlSpectre"),
	},
	// src/Data/Skills/spectre.lua:6515
	"EmptyActionSpellWarlordGrandmaster": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:EmptyActionSpellWarlordGrandmaster"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:EmptyActionSpellWarlordGrandmaster"),
	},
	// src/Data/Skills/act_str.lua:3943
	"EnduringCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:EnduringCry"),
	},
	// src/Data/Skills/other.lua:6125
	"EnemyExplode": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:EnemyExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:EnemyExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Source("Skill:EnemyExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "CorpseExplosionLifeMultiplier", Value: 0}).Source("Skill:EnemyExplode"),
		mod.NewList("SkillData", mod.SkillData{Key: "HitChanceIsExplodeChance", Value: 1}).Tag(mod.SkillPart(1)).Source("Skill:EnemyExplode"),
	},
	// src/Data/Skills/act_int.lua:5197
	"Enfeeble": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Enfeeble"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Enfeeble"),
	},
	// src/Data/Skills/other.lua:1760
	"Envy": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Envy"),
	},
	// src/Data/Skills/act_int.lua:5274
	"EssenceDrain": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:EssenceDrain"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:EssenceDrain"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Source("Skill:EssenceDrain"),
	},
	// src/Data/Skills/act_int.lua:5356
	"EssenceDrainAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:EssenceDrainAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:EssenceDrainAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Source("Skill:EssenceDrainAltX"),
	},
	// src/Data/Skills/act_int.lua:5439
	"EssenceDrainAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:EssenceDrainAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:EssenceDrainAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Source("Skill:EssenceDrainAltY"),
	},
	// src/Data/Skills/act_str.lua:4034
	"Eviscerate": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:Eviscerate"),
	},
	// src/Data/Skills/act_dex.lua:6813
	"ExplosiveArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:ExplosiveArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPartList(1, 2)).Source("Skill:ExplosiveArrow"),
		mod.NewFloat("Damage", mod.TypeMore, 100).Tag(mod.SkillPartList(1, 2), mod.Multiplier("ExplosiveArrowStageAfterFirst")).Source("Skill:ExplosiveArrow"),
	},
	// src/Data/Skills/act_dex.lua:7232
	"ExplosiveTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Primary Explosion:"}).Source("Skill:ExplosiveTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Secondary Area:"}).Source("Skill:ExplosiveTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Secondary Explosion:"}).Source("Skill:ExplosiveTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryBaseMargin", Value: 30}).Source("Skill:ExplosiveTrap"),
	},
	// src/Data/Skills/act_dex.lua:7324
	"ExplosiveTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Primary Explosion:"}).Source("Skill:ExplosiveTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Secondary Area:"}).Source("Skill:ExplosiveTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Secondary Explosion:"}).Source("Skill:ExplosiveTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryBaseMargin", Value: 30}).Source("Skill:ExplosiveTrapAltX"),
	},
	// src/Data/Skills/act_str.lua:4122
	"Exsanguinate": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Exsanguinate"),
		mod.NewFloat("Multiplier:ExsanguinateMaxStages", mod.TypeBase, 3).Source("Skill:Exsanguinate"),
		mod.NewFloat("PhysicalDamage", mod.TypeMore, 100).KeywordFlag(mod.KeywordFlagPhysicalDot).Tag(mod.Multiplier("ExsanguinateStageAfterFirst"), mod.Condition("ExsanguinateDebuffIsFireDamage").Neg(true)).Source("Skill:Exsanguinate"),
		mod.NewFloat("FireDamage", mod.TypeMore, 100).Flag(mod.MFlagSpell|mod.MFlagDot).Tag(mod.Multiplier("ExsanguinateStageAfterFirst"), mod.Condition("ExsanguinateDebuffIsFireDamage")).Source("Skill:Exsanguinate"),
	},
	// src/Data/Skills/act_str.lua:4211
	"ExsanguinateAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ExsanguinateAltX"),
		mod.NewFloat("Multiplier:ExsanguinateofTransmissionMaxStages", mod.TypeBase, 3).Source("Skill:ExsanguinateAltX"),
		mod.NewFloat("PhysicalDamage", mod.TypeMore, 100).KeywordFlag(mod.KeywordFlagPhysicalDot).Tag(mod.Multiplier("ExsanguinateofTransmissionStageAfterFirst"), mod.Condition("ExsanguinateDebuffIsFireDamage").Neg(true)).Source("Skill:ExsanguinateAltX"),
		mod.NewFloat("FireDamage", mod.TypeMore, 100).Flag(mod.MFlagSpell|mod.MFlagDot).Tag(mod.Multiplier("ExsanguinateofTransmissionStageAfterFirst"), mod.Condition("ExsanguinateDebuffIsFireDamage")).Source("Skill:ExsanguinateAltX"),
	},
	// src/Data/Skills/spectre.lua:7106
	"FemaleCannibalBossFireStorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fireball explosion:"}).Source("Skill:FemaleCannibalBossFireStorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:FemaleCannibalBossFireStorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Fireballs fall:"}).Source("Skill:FemaleCannibalBossFireStorm"),
	},
	// src/Data/Skills/other.lua:1853
	"FireBurstOnHit": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:FireBurstOnHit"),
	},
	// src/Data/Skills/minion.lua:239
	"FireElementalMeteorSummoned": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:FireElementalMeteorSummoned"),
	},
	// src/Data/Skills/act_dex.lua:7501
	"FireTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:FireTrap"),
		mod.NewFlag("dotIsBurningGround", true).Source("Skill:FireTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:FireTrap"),
	},
	// src/Data/Skills/act_dex.lua:7593
	"FireTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:FireTrapAltX"),
	},
	// src/Data/Skills/act_int.lua:5991
	"Firestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 0}).Tag(mod.SkillPart(1)).Source("Skill:Firestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fireball explosion:"}).Source("Skill:Firestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:Firestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which fireballs fall:"}).Source("Skill:Firestorm"),
	},
	// src/Data/Skills/act_int.lua:6150
	"FirestormAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 22}).Source("Skill:FirestormAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fire projectile:"}).Source("Skill:FirestormAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Target area:"}).Source("Skill:FirestormAltY"),
	},
	// src/Data/Skills/act_int.lua:6329
	"FlameDash": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:FlameDash"),
		mod.NewFlag("dotIsBurningGround", true).Source("Skill:FlameDash"),
	},
	// src/Data/Skills/act_int.lua:6411
	"FlameDashAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:FlameDashAltY"),
		mod.NewFlag("dotIsBurningGround", true).Source("Skill:FlameDashAltY"),
	},
	// src/Data/Skills/act_int.lua:6632
	"FlameSurge": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:FlameSurge"),
	},
	// src/Data/Skills/act_int.lua:6725
	"FlameSurgeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:FlameSurgeAltX"),
	},
	// src/Data/Skills/act_int.lua:6529
	"FlameWall": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Flame Wall Length:"}).Source("Skill:FlameWall"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:FlameWall"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:FlameWall"),
	},
	// src/Data/Skills/act_int.lua:6827
	"Flameblast": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 2}).Source("Skill:Flameblast"),
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, 100).Tag(mod.Multiplier("FlameblastStageAfterFirst")).Source("Skill:Flameblast"),
	},
	// src/Data/Skills/act_int.lua:6932
	"FlameblastAltX": {
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, 100).Tag(mod.Multiplier("FlameblastofCelerityStageAfterFirst")).Source("Skill:FlameblastAltX"),
	},
	// src/Data/Skills/act_int.lua:7038
	"FlameblastAltY": {
		mod.NewFloat("PvpTvalueMultiplier", mod.TypeMore, 100).Tag(mod.Multiplier("FlameblastofContractionStageAfterFirst")).Source("Skill:FlameblastAltY"),
	},
	// src/Data/Skills/act_dex.lua:7728
	"FlamethrowerTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 32}).Source("Skill:FlamethrowerTrap"),
	},
	// src/Data/Skills/act_dex.lua:7831
	"FlamethrowerTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 32}).Source("Skill:FlamethrowerTrapAltX"),
	},
	// src/Data/Skills/act_int.lua:7226
	"Flammability": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Flammability"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Flammability"),
	},
	// src/Data/Skills/act_str.lua:4389
	"FleshAndStone": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:FleshAndStone"),
	},
	// src/Data/Skills/act_int.lua:7312
	"FleshOffering": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:FleshOffering"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:FleshOffering"),
	},
	// src/Data/Skills/act_int.lua:7461
	"ForbiddenRite": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Source("Skill:ForbiddenRite"),
	},
	// src/Data/Skills/act_int.lua:7608
	"ForbiddenRiteAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Source("Skill:ForbiddenRiteAltX"),
	},
	// src/Data/Skills/act_dex.lua:8521
	"FrostBladesAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:FrostBladesAltX"),
	},
	// src/Data/Skills/act_int.lua:7779
	"FrostBomb": {
		mod.NewList("SkillData", mod.SkillData{Key: "DebuffSecondary", Value: 1}).Source("Skill:FrostBomb"),
	},
	// src/Data/Skills/act_int.lua:8049
	"FrostShield": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:FrostShield"),
	},
	// src/Data/Skills/act_int.lua:8218
	"Frostbite": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Frostbite"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Frostbite"),
	},
	// src/Data/Skills/act_int.lua:8304
	"Frostblink": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:Frostblink"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:Frostblink"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of initial explosion:"}).Source("Skill:Frostblink"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 16}).Source("Skill:Frostblink"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area of Chilled Ground:"}).Source("Skill:Frostblink"),
	},
	// src/Data/Skills/act_int.lua:8394
	"FrostblinkAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:FrostblinkAltX"),
	},
	// src/Data/Skills/act_str.lua:4656
	"FrozenSweep": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:FrozenSweep"),
	},
	// src/Data/Skills/act_str.lua:4774
	"FrozenSweepAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:FrozenSweepAltX"),
	},
	// src/Data/Skills/spectre.lua:4776
	"GAHarvestCrabDashSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:GAHarvestCrabDashSlam"),
	},
	// src/Data/Skills/spectre.lua:4920
	"GAHarvestRhexDashSlash": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:GAHarvestRhexDashSlash"),
	},
	// src/Data/Skills/act_dex.lua:8611
	"GalvanicArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:GalvanicArrow"),
	},
	// src/Data/Skills/act_dex.lua:8689
	"GalvanicArrowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:GalvanicArrowAltX"),
	},
	// src/Data/Skills/act_dex.lua:8775
	"GalvanicArrowAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:GalvanicArrowAltY"),
	},
	// src/Data/Skills/act_int.lua:8583
	"GalvanicField": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 19}).Source("Skill:GalvanicField"),
	},
	// src/Data/Skills/act_int.lua:8697
	"GalvanicFieldAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 19}).Source("Skill:GalvanicFieldAltX"),
	},
	// src/Data/Skills/act_str.lua:4851
	"GeneralsCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:GeneralsCry"),
	},
	// src/Data/Skills/spectre.lua:868
	"GhostPirateBladeVortexSpectre": {
		mod.NewList("SkillData", mod.SkillData{Key: "HitTimeOverride", Value: 1}).Source("Skill:GhostPirateBladeVortexSpectre"),
	},
	// src/Data/Skills/act_int.lua:8795
	"GlacialCascade": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:GlacialCascade"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusExtra", Value: 12}).Tag(mod.SkillPart(2)).Source("Skill:GlacialCascade"),
	},
	// src/Data/Skills/act_int.lua:8876
	"GlacialCascadeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:GlacialCascadeAltX"),
	},
	// src/Data/Skills/other.lua:2068
	"GluttonyOfElements": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:GluttonyOfElements"),
		mod.NewFloat("AvoidCurse", mod.TypeBase, 100).Tag(mod.GlobalEffect("Aura").Unscalable(true)).Source("Skill:GluttonyOfElements"),
	},
	// src/Data/Skills/act_dex.lua:8940
	"Grace": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Grace"),
	},
	// src/Data/Skills/act_str.lua:5267
	"GroundSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 35}).Source("Skill:GroundSlam"),
	},
	// src/Data/Skills/act_str.lua:5357
	"GroundSlamAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 35}).Source("Skill:GroundSlamAltX"),
	},
	// src/Data/Skills/minion.lua:1849
	"GuardianSentinelFireAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireDot", Value: 1}).Tag(mod.Multiplier("GuardianSentinelFireAuraBaseDamage"), mod.PercentStat("Life", 1)).Source("Skill:GuardianSentinelFireAura"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:GuardianSentinelFireAura"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 1}).Tag(mod.Multiplier("GuardianSentinelFireAuraRadius")).Source("Skill:GuardianSentinelFireAura"),
	},
	// src/Data/Skills/spectre.lua:1137
	"HalfSkeletonPuncture": {
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Source("Skill:HalfSkeletonPuncture"),
	},
	// src/Data/Skills/spectre.lua:4799
	"HarvestCrabAbyssSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:HarvestCrabAbyssSlam"),
	},
	// src/Data/Skills/spectre.lua:4883
	"HarvestRhexLeapSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:HarvestRhexLeapSlam"),
	},
	// src/Data/Skills/act_dex.lua:9104
	"Haste": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Haste"),
	},
	// src/Data/Skills/act_dex.lua:9272
	"Hatred": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Hatred"),
	},
	// src/Data/Skills/spectre.lua:5241
	"HeistThugRangedExplosiveArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:HeistThugRangedExplosiveArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:HeistThugRangedExplosiveArrow"),
		mod.NewFloat("Damage", mod.TypeMore, 100).Tag(mod.Multiplier("ExplosiveArrowFuse").Base(-100)).Source("Skill:HeistThugRangedExplosiveArrow"),
		mod.NewFloat("Multiplier:ExplosiveArrowFuse", mod.TypeBase, 20).Source("Skill:HeistThugRangedExplosiveArrow"),
	},
	// src/Data/Skills/act_str.lua:5700
	"HeraldOfAsh": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:HeraldOfAsh"),
		mod.NewFlag("dotIsHeraldOfAsh", true).Source("Skill:HeraldOfAsh"),
	},
	// src/Data/Skills/act_int.lua:9411
	"HeraldOfThunder": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 32}).Source("Skill:HeraldOfThunder"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 0}).Source("Skill:HeraldOfThunder"),
	},
	// src/Data/Skills/act_int.lua:9122
	"Hexblast": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:Hexblast"),
	},
	// src/Data/Skills/act_int.lua:9225
	"HexblastAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:HexblastAltX"),
	},
	// src/Data/Skills/act_str.lua:5868
	"HolyFlameTotem": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:HolyFlameTotem"),
	},
	// src/Data/Skills/act_int.lua:9012
	"Hydrosphere": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 23}).Source("Skill:Hydrosphere"),
	},
	// src/Data/Skills/act_str.lua:6545
	"IceCrash": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "First hit:"}).Source("Skill:IceCrash"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Second hit:"}).Source("Skill:IceCrash"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Third hit:"}).Source("Skill:IceCrash"),
	},
	// src/Data/Skills/minion.lua:287
	"IceElementalIceCyclone": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Source("Skill:IceElementalIceCyclone"),
	},
	// src/Data/Skills/act_dex.lua:9558
	"IceShot": {
		mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 40).Tag(mod.SkillPart(2)).Source("Skill:IceShot"),
	},
	// src/Data/Skills/act_dex.lua:9642
	"IceShotAltX": {
		mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 40).Tag(mod.SkillPart(2)).Source("Skill:IceShotAltX"),
	},
	// src/Data/Skills/act_int.lua:9839
	"IceSpear": {
		mod.NewFloat("PierceChance", mod.TypeBase, 100).Tag(mod.SkillPart(1)).Source("Skill:IceSpear"),
	},
	// src/Data/Skills/act_int.lua:9943
	"IceSpearAltX": {
		mod.NewFloat("PierceChance", mod.TypeBase, 100).Tag(mod.SkillPart(1)).Source("Skill:IceSpearAltX"),
	},
	// src/Data/Skills/act_dex.lua:9792
	"IceTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "First Explosion:"}).Source("Skill:IceTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Second Explosion:"}).Source("Skill:IceTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Third Explosion:"}).Source("Skill:IceTrap"),
	},
	// src/Data/Skills/act_dex.lua:9877
	"IceTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "First Explosion:"}).Source("Skill:IceTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Second Explosion:"}).Source("Skill:IceTrapAltX"),
	},
	// src/Data/Skills/other.lua:2262
	"Icestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 22}).Source("Skill:Icestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Ice projectile:"}).Source("Skill:Icestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Target area:"}).Source("Skill:Icestorm"),
	},
	// src/Data/Skills/other.lua:2300
	"IcicleBurst": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:IcicleBurst"),
	},
	// src/Data/Skills/act_int.lua:10337
	"Incinerate": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:Incinerate"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Flame Length:"}).Source("Skill:Incinerate"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:Incinerate"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Flame Width:"}).Source("Skill:Incinerate"),
	},
	// src/Data/Skills/act_int.lua:10475
	"IncinerateAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:IncinerateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Flame Length:"}).Source("Skill:IncinerateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:IncinerateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Flame Width:"}).Source("Skill:IncinerateAltX"),
	},
	// src/Data/Skills/act_int.lua:10593
	"IncinerateAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:IncinerateAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Flame Length:"}).Source("Skill:IncinerateAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:IncinerateAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Flame Width:"}).Source("Skill:IncinerateAltY"),
	},
	// src/Data/Skills/act_str.lua:7094
	"InfernalBlow": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:InfernalBlow"),
		mod.NewFloat("Multiplier:DebuffStack", mod.TypeBase, 5).Tag(mod.SkillPart(3)).Source("Skill:InfernalBlow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Hit area:"}).Tag(mod.SkillPart(1)).Source("Skill:InfernalBlow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Explosion area:"}).Tag(mod.SkillPartList(2, 3)).Source("Skill:InfernalBlow"),
	},
	// src/Data/Skills/act_str.lua:7214
	"InfernalBlowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:InfernalBlowAltX"),
		mod.NewFloat("Multiplier:DebuffStack", mod.TypeBase, 5).Tag(mod.SkillPart(3)).Source("Skill:InfernalBlowAltX"),
	},
	// src/Data/Skills/act_str.lua:6877
	"InfernalCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:InfernalCry"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Warcry area:"}).Source("Skill:InfernalCry"),
	},
	// src/Data/Skills/minion.lua:1827
	"InfernalLegion": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireDot", Value: 1}).Tag(mod.Multiplier("InfernalLegionBaseDamage")).Source("Skill:InfernalLegion"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:InfernalLegion"),
	},
	// src/Data/Skills/act_str.lua:7297
	"IntimidatingCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:IntimidatingCry"),
	},
	// src/Data/Skills/spectre.lua:1386
	"KaomFireBeamTotemSpectre": {
		mod.NewFloat("Damage", mod.TypeMore, 60).Tag(mod.Multiplier("KaomFireBeamTotemStage").Base(-60).Limit(8).Actor("parent")).Source("Skill:KaomFireBeamTotemSpectre"),
	},
	// src/Data/Skills/spectre.lua:3797
	"KitavaSlavedriverFlameWhip": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:KitavaSlavedriverFlameWhip"),
	},
	// src/Data/Skills/act_dex.lua:9978
	"Lacerate": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 38}).Source("Skill:Lacerate"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:Lacerate"),
	},
	// src/Data/Skills/act_dex.lua:10066
	"LacerateAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 38}).Source("Skill:LacerateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:LacerateAltX"),
	},
	// src/Data/Skills/act_dex.lua:10154
	"LacerateAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 38}).Source("Skill:LacerateAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:LacerateAltY"),
	},
	// src/Data/Skills/act_dex.lua:10429
	"LightningArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:LightningArrow"),
	},
	// src/Data/Skills/act_dex.lua:10510
	"LightningArrowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 1}).Tag(mod.Multiplier("LightningArrowofElectrocutionStage")).Source("Skill:LightningArrowAltX"),
	},
	// src/Data/Skills/act_int.lua:12393
	"LightningConduit": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Targeting range:"}).Source("Skill:LightningConduit"),
	},
	// src/Data/Skills/act_int.lua:11828
	"LightningSpireTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:LightningSpireTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Targeting Area:"}).Source("Skill:LightningSpireTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 10}).Source("Skill:LightningSpireTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Impact Area:"}).Source("Skill:LightningSpireTrap"),
	},
	// src/Data/Skills/act_int.lua:12068
	"LightningSpireTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:LightningSpireTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Targeting Area:"}).Source("Skill:LightningSpireTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 10}).Source("Skill:LightningSpireTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Impact Area:"}).Source("Skill:LightningSpireTrapAltX"),
	},
	// src/Data/Skills/act_int.lua:12300
	"LightningSpireTrapAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:LightningSpireTrapAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Targeting Area:"}).Source("Skill:LightningSpireTrapAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 10}).Source("Skill:LightningSpireTrapAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Impact Area:"}).Source("Skill:LightningSpireTrapAltY"),
	},
	// src/Data/Skills/act_int.lua:12566
	"LightningTendrils": {
		mod.NewFloat("DPS", mod.TypeMore, -66.666666666667).Tag(mod.SkillPart(2)).Source("Skill:LightningTendrils"),
		mod.NewFloat("DPS", mod.TypeMore, -33.333333333333).Tag(mod.SkillPart(3)).Source("Skill:LightningTendrils"),
	},
	// src/Data/Skills/act_int.lua:13117
	"LightningWarp": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:LightningWarp"),
	},
	// src/Data/Skills/act_int.lua:13368
	"Malevolence": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Malevolence"),
	},
	// src/Data/Skills/act_int.lua:13546
	"Manabond": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:Manabond"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Circle area:"}).Source("Skill:Manabond"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 23}).Source("Skill:Manabond"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Rectangle area:"}).Source("Skill:Manabond"),
	},
	// src/Data/Skills/other.lua:2567
	"ManifestDancingDervishes": {
		mod.NewFlag("DisableWeapons", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:ManifestDancingDervishes"),
		mod.NewFlag("Condition:Onslaught", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:ManifestDancingDervishes"),
		mod.NewList("SkillData", mod.SkillData{Key: "MinionLevel", Value: 65}).Source("Skill:ManifestDancingDervishes"),
	},
	// src/Data/Skills/minion.lua:1760
	"MinionInstability": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireMin", Value: 1}).Tag(mod.PerStat(3.030303030303, "Life")).Source("Skill:MinionInstability"),
		mod.NewList("SkillData", mod.SkillData{Key: "FireMax", Value: 1}).Tag(mod.PerStat(3.030303030303, "Life")).Source("Skill:MinionInstability"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:MinionInstability"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:MinionInstability"),
	},
	// src/Data/Skills/act_dex.lua:11003
	"MirrorArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:MirrorArrow"),
	},
	// src/Data/Skills/act_dex.lua:11089
	"MirrorArrowAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:MirrorArrowAltX"),
	},
	// src/Data/Skills/act_dex.lua:11174
	"MirrorArrowAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "MinionUseBowAndQuiver", Value: 1}).Source("Skill:MirrorArrowAltY"),
	},
	// src/Data/Skills/other.lua:2600
	"MistyReflection": {
		mod.NewFloat("DamageTaken", mod.TypeIncrease, 30).Tag(mod.Condition("MistyReflection"), mod.GlobalEffect("Debuff")).Source("Skill:MistyReflection"),
		mod.NewFloat("Damage", mod.TypeMore, -30).Tag(mod.Condition("MistyReflection"), mod.GlobalEffect("Debuff")).Source("Skill:MistyReflection"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:MistyReflection"),
	},
	// src/Data/Skills/act_str.lua:7564
	"MoltenShell": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:MoltenShell"),
	},
	// src/Data/Skills/act_str.lua:7831
	"MoltenStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "ProjectileSpeedAppliesToMSAreaOfEffect", Value: 1}).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 9}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Ball area:"}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 2}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Chain Minimum Distance:"}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 25}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Chain Maximum Distance:"}).Tag(mod.SkillPartList(2, 3)).Source("Skill:MoltenStrike"),
		mod.NewFlag("CannotSplit", true).Source("Skill:MoltenStrike"),
	},
	// src/Data/Skills/act_str.lua:8039
	"MoltenStrikeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ProjectileSpeedAppliesToMSAreaOfEffect", Value: 1}).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 9}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Ball area:"}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 2}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Chain Minimum Distance:"}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 25}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Chain Maximum Distance:"}).Tag(mod.SkillPartList(2, 3, 4, 5, 6)).Source("Skill:MoltenStrikeAltX"),
		mod.NewFlag("CannotSplit", true).Source("Skill:MoltenStrikeAltX"),
	},
	// src/Data/Skills/spectre.lua:6614
	"MonsterChanceToTemporalChainsOnHit1": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:MonsterChanceToTemporalChainsOnHit1"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:MonsterChanceToTemporalChainsOnHit1"),
	},
	// src/Data/Skills/spectre.lua:2597
	"MonsterPuncture": {
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Source("Skill:MonsterPuncture"),
	},
	// src/Data/Skills
	"MonsterVulnerabilityOnHit1": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:MonsterVulnerabilityOnHit1"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:MonsterVulnerabilityOnHit1"),
	},
	// src/Data/Skills/act_int.lua:13635
	"OrbOfStorms": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:OrbOfStorms"),
	},
	// src/Data/Skills/act_int.lua:13732
	"OrbOfStormsAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:OrbOfStormsAltX"),
	},
	// src/Data/Skills/other.lua:3350
	"Pacify": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Pacify"),
	},
	// src/Data/Skills/act_int.lua:13822
	"PenanceBrand": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:PenanceBrand"),
		mod.NewFloat("Multiplier:PenanceBrandMaxEnergy", mod.TypeBase, 20).Source("Skill:PenanceBrand"),
	},
	// src/Data/Skills/act_int.lua:13931
	"PenanceBrandAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:PenanceBrandAltX"),
		mod.NewFloat("Multiplier:PenanceBrandofDissipationMaxStages", mod.TypeBase, 20).Tag(mod.SkillPart(1)).Source("Skill:PenanceBrandAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:PenanceBrandAltX"),
	},
	// src/Data/Skills/act_int.lua:14024
	"PenanceBrandAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:PenanceBrandAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:PenanceBrandAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:PenanceBrandAltY"),
	},
	// src/Data/Skills/other.lua:2707
	"PenanceMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:PenanceMark"),
		mod.NewList("SkillData", mod.SkillData{Key: "MinionLevelIsEnemyLevel", Value: 1}).Source("Skill:PenanceMark"),
	},
	// src/Data/Skills/act_str.lua:8260
	"Perforate": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 1}).Tag(multiplierLimitVar("PerforateSpikeOverlap", "PerforateMaxSpikes"), mod.Condition("BloodStance")).Source("Skill:Perforate"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 11}).Tag(mod.Condition("SandStance")).Source("Skill:Perforate"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Tag(mod.Condition("BloodStance")).Source("Skill:Perforate"),
	},
	// src/Data/Skills/act_str.lua:8359
	"PerforateAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 1}).Tag(multiplierLimitVar("PerforateSpikeOverlap", "PerforateofDualityMaxSpikes"), mod.Condition("BloodStance")).Source("Skill:PerforateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 11}).Tag(mod.Condition("SandStance")).Source("Skill:PerforateAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 8}).Tag(mod.Condition("BloodStance")).Source("Skill:PerforateAltX"),
	},
	// src/Data/Skills/act_str.lua:8454
	"PerforateAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 1}).Tag(multiplierLimitVar("PerforateSpikeOverlap", "PerforateofBloodshedMaxSpikes")).Source("Skill:PerforateAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 11}).Source("Skill:PerforateAltY"),
	},
	// src/Data/Skills/act_dex.lua:11256
	"PestilentStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 27}).Source("Skill:PestilentStrike"),
	},
	// src/Data/Skills/act_dex.lua:11413
	"PlagueBearer": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:PlagueBearer"),
	},
	// src/Data/Skills/act_dex.lua:11509
	"PoachersMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:PoachersMark"),
	},
	// src/Data/Skills/act_dex.lua:17537
	"PoisonousConcoctionAltX": {
		mod.NewFloat("DPS", mod.TypeMore, 100).Tag(mod.PerStat(1, "ChainMax")).Source("Skill:PoisonousConcoctionAltX"),
	},
	// src/Data/Skills/act_dex.lua:11590
	"Precision": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Precision"),
	},
	// src/Data/Skills/act_str.lua:8618
	"Pride": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Pride"),
	},
	// src/Data/Skills/sup_int.lua:5438
	"PrismaticBurst": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 13}).Source("Skill:PrismaticBurst"),
	},
	// src/Data/Skills/act_dex.lua:12075
	"Puncture": {
		mod.NewList("SkillData", mod.SkillData{Key: "BleedIsSkillEffect", Value: 1}).Source("Skill:Puncture"),
	},
	// src/Data/Skills/act_dex.lua:12162
	"PunctureAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "BleedIsSkillEffect", Value: 1}).Source("Skill:PunctureAltX"),
	},
	// src/Data/Skills/act_str.lua:8774
	"Punishment": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Punishment"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Punishment"),
		mod.NewFloat("DebilitateChance", mod.TypeBase, 100).Source("Skill:Punishment"),
	},
	// src/Data/Skills/act_int.lua:14380
	"PurifyingFlame": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:PurifyingFlame"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Hit:"}).Source("Skill:PurifyingFlame"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 50}).Source("Skill:PurifyingFlame"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Shockwave:"}).Source("Skill:PurifyingFlame"),
	},
	// src/Data/Skills/act_int.lua:14484
	"PurifyingFlameAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:PurifyingFlameAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Hit:"}).Source("Skill:PurifyingFlameAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 50}).Source("Skill:PurifyingFlameAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Shockwave:"}).Source("Skill:PurifyingFlameAltX"),
	},
	// src/Data/Skills/act_int.lua:14575
	"PurityOfElements": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:PurityOfElements"),
	},
	// src/Data/Skills/act_str.lua:8860
	"PurityOfFire": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:PurityOfFire"),
	},
	// src/Data/Skills/act_dex.lua:12245
	"PurityOfIce": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:PurityOfIce"),
	},
	// src/Data/Skills/act_int.lua:14654
	"PurityOfLightning": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:PurityOfLightning"),
	},
	// src/Data/Skills/act_int.lua:14819
	"PyroclastMine": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:PyroclastMine"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of initial explosion:"}).Source("Skill:PyroclastMine"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 26}).Source("Skill:PyroclastMine"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which projectiles will land:"}).Source("Skill:PyroclastMine"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 12}).Source("Skill:PyroclastMine"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Area of projectile explosions:"}).Source("Skill:PyroclastMine"),
	},
	// src/Data/Skills/act_int.lua:14917
	"PyroclastMineAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:PyroclastMineAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of initial explosion:"}).Source("Skill:PyroclastMineAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 26}).Source("Skill:PyroclastMineAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which projectiles will land:"}).Source("Skill:PyroclastMineAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 12}).Source("Skill:PyroclastMineAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Area of projectile explosions:"}).Source("Skill:PyroclastMineAltX"),
	},
	// src/Data/Skills/act_str.lua:10963
	"QuickGuard": {
		mod.NewFloat("AvoidBleed", mod.TypeBase, 100).Tag(mod.GlobalEffect("Guard")).Source("Skill:QuickGuard"),
	},
	// src/Data/Skills/act_dex.lua:12410
	"RainOfArrows": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:RainOfArrows"),
		mod.NewFlag("OneShotProj", true).Source("Skill:RainOfArrows"),
	},
	// src/Data/Skills/act_dex.lua:12485
	"RainOfArrowsAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:RainOfArrowsAltX"),
		mod.NewFlag("OneShotProj", true).Source("Skill:RainOfArrowsAltX"),
	},
	// src/Data/Skills/act_dex.lua:12560
	"RainOfArrowsAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:RainOfArrowsAltY"),
		mod.NewFlag("OneShotProj", true).Source("Skill:RainOfArrowsAltY"),
	},
	// src/Data/Skills/minion.lua:2090
	"RainOfArrowsCloneShot": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:RainOfArrowsCloneShot"),
		mod.NewFlag("OneShotProj", true).Source("Skill:RainOfArrowsCloneShot"),
	},
	// src/Data/Skills/act_str.lua:9250
	"RallyingCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:RallyingCry"),
	},
	// src/Data/Skills/act_str.lua:11053
	"Reap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 25}).Source("Skill:Reap"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Reap"),
	},
	// src/Data/Skills/act_str.lua:11153
	"ReapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:ReapAltX"),
	},
	// src/Data/Skills/minion.lua:2016
	"ReaperConsumeMinionForBuff": {
		mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, 40)}).Tag(mod.SkillName("Summon Reaper").Transfigured(true), mod.GlobalEffect("Buff")).Source("Skill:ReaperConsumeMinionForBuff"),
		mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, 40)}).Tag(mod.SkillName("Summon Reaper").Transfigured(true), mod.GlobalEffect("Buff")).Source("Skill:ReaperConsumeMinionForBuff"),
		mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 30)}).Tag(mod.SkillName("Summon Reaper").Transfigured(true), mod.GlobalEffect("Buff")).Source("Skill:ReaperConsumeMinionForBuff"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:ReaperConsumeMinionForBuff"),
	},
	// src/Data/Skills/act_dex.lua:12729
	"Reave": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:Reave"),
		mod.NewFloat("Multiplier:ReaveMaxStages", mod.TypeBase, 8).Source("Skill:Reave"),
	},
	// src/Data/Skills/act_dex.lua:12818
	"ReaveAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:ReaveAltX"),
		mod.NewFloat("Multiplier:ReaveofRefractionMaxStages", mod.TypeBase, 8).Source("Skill:ReaveAltX"),
	},
	// src/Data/Skills/act_str.lua:9342
	"RejuvenationTotem": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:RejuvenationTotem"),
	},
	// src/Data/Skills/minion.lua:1022
	"RelicTriggeredNova": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:RelicTriggeredNova"),
	},
	// src/Data/Skills/minion.lua:1805
	"ReplicaSiegebreakerBurningGround": {
		mod.NewList("SkillData", mod.SkillData{Key: "FireDot", Value: 1}).Tag(mod.PercentStatVar("Life", "SiegebreakerBurningGroundPercent")).Source("Skill:ReplicaSiegebreakerBurningGround"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:ReplicaSiegebreakerBurningGround"),
		mod.NewFlag("dotIsBurningGround", true).Source("Skill:ReplicaSiegebreakerBurningGround"),
	},
	// src/Data/Skills/act_int.lua:15460
	"RighteousFire": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:RighteousFire"),
	},
	// src/Data/Skills/act_int.lua:15555
	"RighteousFireAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:RighteousFireAltX"),
	},
	// src/Data/Skills/act_int.lua:13277
	"RollingMagma": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 14}).Source("Skill:RollingMagma"),
		mod.NewFlag("CannotSplit", true).Source("Skill:RollingMagma"),
	},
	// src/Data/Skills/minion.lua:55
	"SandstormChaosElementalSummoned": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:SandstormChaosElementalSummoned"),
	},
	// src/Data/Skills/minion.lua:85
	"SandstormChaosElementalSummonedEmpowered": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:SandstormChaosElementalSummonedEmpowered"),
	},
	// src/Data/Skills/act_int.lua:15750
	"ScorchingRay": {
		mod.NewFlag("Condition:ScorchingRayMaxStages", true).Tag(mod.MultiplierThreshold("ScorchingRayStageAfterFirst").Threshold(7)).Source("Skill:ScorchingRay"),
		mod.NewFlag("DotCanStackAsTotems", true).Source("Skill:ScorchingRay"),
	},
	// src/Data/Skills/act_int.lua:15849
	"ScorchingRayAltX": {
		mod.NewFlag("Condition:ScorchingRayofImmolationMaxStages", true).Tag(mod.MultiplierThreshold("ScorchingRayofImmolationStageAfterFirst").Threshold(7)).Source("Skill:ScorchingRayAltX"),
		mod.NewFlag("DotCanStackAsTotems", true).Source("Skill:ScorchingRayAltX"),
	},
	// src/Data/Skills/act_str.lua:9697
	"SeismicCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 60}).Source("Skill:SeismicCry"),
	},
	// src/Data/Skills/act_dex.lua:13536
	"SeismicTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:SeismicTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Large Burst:"}).Source("Skill:SeismicTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 9}).Source("Skill:SeismicTrap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Small Burst:"}).Source("Skill:SeismicTrap"),
	},
	// src/Data/Skills/act_dex.lua:13694
	"SeismicTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:SeismicTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Large Burst:"}).Source("Skill:SeismicTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 9}).Source("Skill:SeismicTrapAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Small Burst:"}).Source("Skill:SeismicTrapAltX"),
	},
	// src/Data/Skills/other.lua:3598
	"ShaperApparitionBeam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:ShaperApparitionBeam"),
	},
	// src/Data/Skills/other.lua:3633
	"ShaperApparitionProjectiles": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:ShaperApparitionProjectiles"),
	},
	// src/Data/Skills/other.lua:3564
	"ShaperApparitionTeleportSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:ShaperApparitionTeleportSlam"),
	},
	// src/Data/Skills/act_dex.lua:13200
	"ShatteringSteel": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:ShatteringSteel"),
	},
	// src/Data/Skills/act_dex.lua:13303
	"ShatteringSteelAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:ShatteringSteelAltX"),
	},
	// src/Data/Skills/act_str.lua:9783
	"ShieldCharge": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Source("Skill:ShieldCharge"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "End Cone area:"}).Source("Skill:ShieldCharge"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 16}).Source("Skill:ShieldCharge"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "End Circle area:"}).Source("Skill:ShieldCharge"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 8}).Source("Skill:ShieldCharge"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Charging area:"}).Source("Skill:ShieldCharge"),
	},
	// src/Data/Skills/act_str.lua:9885
	"ShieldCrush": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 29}).Source("Skill:ShieldCrush"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:ShieldCrush"),
	},
	// src/Data/Skills/act_str.lua:9979
	"ShieldCrushAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 29}).Source("Skill:ShieldCrushAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:ShieldCrushAltX"),
	},
	// src/Data/Skills/other.lua:3686
	"ShieldShatter": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:ShieldShatter"),
	},
	// src/Data/Skills/act_int.lua:15942
	"ShockNova": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:ShockNova"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(1)).Source("Skill:ShockNova"),
	},
	// src/Data/Skills/act_int.lua:16027
	"ShockNovaAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:ShockNovaAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "DpsMultiplier", Value: 2}).Tag(mod.SkillPart(1)).Source("Skill:ShockNovaAltX"),
	},
	// src/Data/Skills/act_dex.lua:13972
	"SiegeBallista": {
		mod.NewFlag("ProjectileRain", true).Source("Skill:SiegeBallista"),
	},
	// src/Data/Skills/act_dex.lua:14052
	"SiegeBallistaAltX": {
		mod.NewFlag("ProjectileRain", true).Source("Skill:SiegeBallistaAltX"),
	},
	// src/Data/Skills/minion.lua:1783
	"SiegebreakerCausticGround": {
		mod.NewList("SkillData", mod.SkillData{Key: "ChaosDot", Value: 1}).Tag(mod.PercentStatVar("Life", "SiegebreakerCausticGroundPercent")).Source("Skill:SiegebreakerCausticGround"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:SiegebreakerCausticGround"),
		mod.NewFlag("dotIsCausticGround", true).Source("Skill:SiegebreakerCausticGround"),
	},
	// src/Data/Skills/act_int.lua:16128
	"SigilOfPower": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:SigilOfPower"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:SigilOfPower"),
	},
	// src/Data/Skills/act_int.lua:16224
	"SiphoningTrap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SiphoningTrap"),
	},
	// src/Data/Skills/act_int.lua:16306
	"SiphoningTrapAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SiphoningTrapAltX"),
	},
	// src/Data/Skills/spectre.lua:3558
	"SkeletonMassBowProjectile": {
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Source("Skill:SkeletonMassBowProjectile"),
	},
	// src/Data/Skills/spectre.lua:3763
	"SlavedriverFlameWhip": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:SlavedriverFlameWhip"),
	},
	// src/Data/Skills/act_str.lua:10345
	"Smite": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Lightning Strike area:"}).Source("Skill:Smite"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Aura area:"}).Source("Skill:Smite"),
	},
	// src/Data/Skills/act_dex.lua:14224
	"SmokeMine": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:SmokeMine"),
	},
	// src/Data/Skills/act_dex.lua:11671
	"SnipersMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SnipersMark"),
	},
	// src/Data/Skills/act_int.lua:16475
	"Soulrend": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Soulrend"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:Soulrend"),
	},
	// src/Data/Skills/act_int.lua:16556
	"SoulrendAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:SoulrendAltX"),
	},
	// src/Data/Skills/act_int.lua:16634
	"SoulrendAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SoulrendAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:SoulrendAltY"),
	},
	// src/Data/Skills/act_dex.lua:15227
	"SpectralHelix": {
		mod.NewFlag("NoAdditionalProjectiles", true).Source("Skill:SpectralHelix"),
		mod.NewFlag("AdditionalProjectilesAddBouncesInstead", true).Source("Skill:SpectralHelix"),
	},
	// src/Data/Skills/act_dex.lua:15319
	"SpectralHelixAltY": {
		mod.NewFlag("NoAdditionalProjectiles", true).Source("Skill:SpectralHelixAltY"),
		mod.NewFlag("AdditionalProjectilesAddBouncesInstead", true).Source("Skill:SpectralHelixAltY"),
	},
	// src/Data/Skills/act_dex.lua:14602
	"SpectralThrow": {
		mod.NewFloat("PierceChance", mod.TypeBase, 100).Source("Skill:SpectralThrow"),
	},
	// src/Data/Skills/act_dex.lua:14682
	"SpectralThrowAltX": {
		mod.NewFloat("PierceChance", mod.TypeBase, 100).Source("Skill:SpectralThrowAltX"),
	},
	// src/Data/Skills/act_dex.lua:14764
	"SpectralThrowAltY": {
		mod.NewFloat("PierceChance", mod.TypeBase, 100).Source("Skill:SpectralThrowAltY"),
	},
	// src/Data/Skills/other.lua:3812
	"SpiritBurst": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:SpiritBurst"),
	},
	// src/Data/Skills/act_int.lua:17304
	"SpiritOffering": {
		mod.NewList("SkillData", mod.SkillData{Key: "BuffMinions", Value: 1}).Source("Skill:SpiritOffering"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffNotPlayer", Value: 1}).Source("Skill:SpiritOffering"),
	},
	// src/Data/Skills/other.lua:3853
	"StalkingPustule": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Orb explosion:"}).Source("Skill:StalkingPustule"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Corpse explosion:"}).Source("Skill:StalkingPustule"),
	},
	// src/Data/Skills/act_str.lua:10695
	"StaticStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:StaticStrike"),
	},
	// src/Data/Skills/act_str.lua:10790
	"StaticStrikeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:StaticStrikeAltX"),
	},
	// src/Data/Skills/act_int.lua:17387
	"StormBrand": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 9}).Source("Skill:StormBrand"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:StormBrand"),
	},
	// src/Data/Skills/act_int.lua:17484
	"StormBrandAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 9}).Source("Skill:StormBrandAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:StormBrandAltX"),
	},
	// src/Data/Skills/act_int.lua:18063
	"StormBurst": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:StormBurst"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 22}).Source("Skill:StormBurst"),
	},
	// src/Data/Skills/act_int.lua:18172
	"StormBurstAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:StormBurstAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 22}).Source("Skill:StormBurstAltX"),
	},
	// src/Data/Skills/act_int.lua:18261
	"StormCall": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:StormCall"),
	},
	// src/Data/Skills/act_int.lua:18345
	"StormCallAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:StormCallAltX"),
	},
	// src/Data/Skills/act_dex.lua:11770
	"StormRain": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Tag(mod.SkillPart(1)).Source("Skill:StormRain"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Arrow Explosion:"}).Tag(mod.SkillPart(1)).Source("Skill:StormRain"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 12}).Tag(mod.SkillPart(2)).Source("Skill:StormRain"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Beam Width:"}).Tag(mod.SkillPart(2)).Source("Skill:StormRain"),
	},
	// src/Data/Skills/act_int.lua:17698
	"Stormbind": {
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 1).Tag(mod.SkillPart(2)).Source("Skill:Stormbind"),
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 2).Tag(mod.SkillPart(3)).Source("Skill:Stormbind"),
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 3).Tag(mod.SkillPart(4)).Source("Skill:Stormbind"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:Stormbind"),
	},
	// src/Data/Skills/act_int.lua:17880
	"StormbindAltX": {
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 1).Tag(mod.SkillPart(2)).Source("Skill:StormbindAltX"),
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 2).Tag(mod.SkillPart(3)).Source("Skill:StormbindAltX"),
		mod.NewFloat("Multiplier:RuneLevel", mod.TypeBase, 3).Tag(mod.SkillPart(4)).Source("Skill:StormbindAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:StormbindAltX"),
	},
	// src/Data/Skills/act_int.lua:17580
	"StormblastMine": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:StormblastMine"),
	},
	// src/Data/Skills/other.lua:4095
	"SummonBeastialRhoa": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonBeastialRhoa"),
		mod.NewFlag("Condition:HaveBestialMinion", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonBeastialRhoa"),
	},
	// src/Data/Skills/other.lua:4134
	"SummonBeastialSnake": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonBeastialSnake"),
		mod.NewFlag("Condition:HaveBestialMinion", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonBeastialSnake"),
	},
	// src/Data/Skills/other.lua:4174
	"SummonBeastialUrsa": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonBeastialUrsa"),
		mod.NewFlag("Condition:HaveBestialMinion", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonBeastialUrsa"),
	},
	// src/Data/Skills/act_int.lua:18533
	"SummonCarrionGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonCarrionGolem"),
		mod.NewFlag("Condition:HaveCarrionGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonCarrionGolem"),
	},
	// src/Data/Skills/act_int.lua:18616
	"SummonCarrionGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonCarrionGolemAltX"),
		mod.NewFlag("Condition:HaveCarrionGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonCarrionGolemAltX"),
	},
	// src/Data/Skills/act_int.lua:18704
	"SummonCarrionGolemAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonCarrionGolemAltY"),
		mod.NewFlag("Condition:HaveCarrionGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonCarrionGolemAltY"),
	},
	// src/Data/Skills/act_int.lua:18793
	"SummonChaosGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonChaosGolem"),
		mod.NewFlag("Condition:HaveChaosGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonChaosGolem"),
	},
	// src/Data/Skills/act_int.lua:18872
	"SummonChaosGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonChaosGolemAltX"),
		mod.NewFlag("Condition:HaveChaosGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonChaosGolemAltX"),
	},
	// src/Data/Skills/act_int.lua:18956
	"SummonChaosGolemAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonChaosGolemAltY"),
		mod.NewFlag("Condition:HaveChaosGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonChaosGolemAltY"),
		mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanWither", true)}).Source("Skill:SummonChaosGolemAltY"),
	},
	// src/Data/Skills/act_str.lua:11346
	"SummonFlameGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonFlameGolem"),
		mod.NewFlag("Condition:HaveFireGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonFlameGolem"),
	},
	// src/Data/Skills/act_str.lua:11425
	"SummonFlameGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonFlameGolemAltX"),
		mod.NewFlag("Condition:HaveFireGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonFlameGolemAltX"),
	},
	// src/Data/Skills/act_str.lua:11503
	"SummonFlameGolemAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonFlameGolemAltY"),
		mod.NewFlag("Condition:HaveFireGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonFlameGolemAltY"),
	},
	// src/Data/Skills/other.lua:4629
	"SummonHarbingerOfTime": {
		mod.NewFloat("ActionSpeed", mod.TypeIncrease, 10).Tag(mod.GlobalEffect("Buff").Name("Harbinger of Time").ModCond("HarbingerOfTime")).Source("Skill:SummonHarbingerOfTime"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:SummonHarbingerOfTime"),
	},
	// src/Data/Skills/other.lua:4480
	"SummonHarbingerOfTimeAscendancy": {
		mod.NewFloat("ActionSpeed", mod.TypeIncrease, 10).Tag(mod.GlobalEffect("Buff").Name("Harbinger of Time").ModCond("HarbingerOfTime")).Source("Skill:SummonHarbingerOfTimeAscendancy"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:SummonHarbingerOfTimeAscendancy"),
	},
	// src/Data/Skills/other.lua:4407
	"SummonHarbingerOfTimeUber": {
		mod.NewFloat("ActionSpeed", mod.TypeIncrease, 10).Tag(mod.GlobalEffect("Buff").Name("Greater Harbinger of Time").ModCond("GreaterHarbingerOfTime")).Source("Skill:SummonHarbingerOfTimeUber"),
		mod.NewList("SkillData", mod.SkillData{Key: "BuffAllies", Value: 1}).Source("Skill:SummonHarbingerOfTimeUber"),
	},
	// src/Data/Skills/act_dex.lua:15716
	"SummonIceGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonIceGolem"),
		mod.NewFlag("Condition:HaveColdGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonIceGolem"),
	},
	// src/Data/Skills/act_dex.lua:15795
	"SummonIceGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonIceGolemAltX"),
		mod.NewFlag("Condition:HaveColdGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonIceGolemAltX"),
	},
	// src/Data/Skills/act_dex.lua:15873
	"SummonIceGolemAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonIceGolemAltY"),
		mod.NewFlag("Condition:HaveColdGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonIceGolemAltY"),
	},
	// src/Data/Skills/act_int.lua:19210
	"SummonLightningGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonLightningGolem"),
		mod.NewFlag("Condition:HaveLightningGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonLightningGolem"),
	},
	// src/Data/Skills/act_int.lua:19289
	"SummonLightningGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonLightningGolemAltX"),
		mod.NewFlag("Condition:HaveLightningGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonLightningGolemAltX"),
	},
	// src/Data/Skills/act_int.lua:20108
	"SummonSkitterbots": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 30}).Source("Skill:SummonSkitterbots"),
	},
	// src/Data/Skills/act_str.lua:11593
	"SummonStoneGolem": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonStoneGolem"),
		mod.NewFlag("Condition:HavePhysicalGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonStoneGolem"),
	},
	// src/Data/Skills/act_str.lua:11672
	"SummonStoneGolemAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonStoneGolemAltX"),
		mod.NewFlag("Condition:HavePhysicalGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonStoneGolemAltX"),
	},
	// src/Data/Skills/act_str.lua:11750
	"SummonStoneGolemAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "AllowTotemBuff", Value: 1}).Source("Skill:SummonStoneGolemAltY"),
		mod.NewFlag("Condition:HavePhysicalGolem", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SummonStoneGolemAltY"),
	},
	// src/Data/Skills/act_str.lua:11856
	"Sunder": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:Sunder"),
		mod.NewFloat("Multiplier:SunderWaveArea", mod.TypeBase, 4).Tag(mod.SkillPart(2)).Source("Skill:Sunder"),
	},
	// src/Data/Skills/act_str.lua:11961
	"SunderAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:SunderAltX"),
	},
	// src/Data/Skills/act_str.lua:12073
	"SunderAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:SunderAltY"),
	},
	// src/Data/Skills/sup_int.lua:6870
	"SupportAwakenedUnleash": {
		mod.NewFlag("HasSeals", true).Source("Skill:SupportAwakenedUnleash"),
	},
	// src/Data/Skills/sup_str.lua:582
	"SupportBloodsoakedBanner": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SupportBloodsoakedBanner"),
		mod.NewFlag("dotIsCorruptingBlood", true).Source("Skill:SupportBloodsoakedBanner"),
		mod.NewFloat("Damage", mod.TypeMore, 100).KeywordFlag(mod.KeywordFlagPhysicalDot).Tag(mod.Multiplier("BloodsoakedBannerStageAfterFirst")).Source("Skill:SupportBloodsoakedBanner"),
	},
	// src/Data/Skills/sup_int.lua:560
	"SupportBonechill": {
		mod.NewList("SkillData", mod.SkillData{Key: "SupportBonechill", Value: 1}).Source("Skill:SupportBonechill"),
	},
	// src/Data/Skills/act_int.lua:645
	"SupportBrandSupport": {
		mod.NewList("SkillData", mod.SkillData{Key: "TriggeredByBrand", Value: 1}).Source("Skill:SupportBrandSupport"),
	},
	// src/Data/Skills/sup_dex.lua:770
	"SupportCastOnDeath": {
		mod.NewList("SkillData", mod.SkillData{Key: "TriggeredOnDeath", Value: 1}).Source("Skill:SupportCastOnDeath"),
	},
	// src/Data/Skills/sup_dex.lua:1187
	"SupportCloseCombat": {
		mod.NewFlag("Condition:SupportedByCloseCombat", true).Source("Skill:SupportCloseCombat"),
	},
	// src/Data/Skills/sup_str.lua:1459
	"SupportCorruptingCry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:SupportCorruptingCry"),
		mod.NewFlag("dotIsCorruptingBlood", true).Source("Skill:SupportCorruptingCry"),
		mod.NewFloat("Multiplier:CorruptingCryMaxStages", mod.TypeBase, 10).Source("Skill:SupportCorruptingCry"),
		mod.NewFloat("Damage", mod.TypeMore, 100).KeywordFlag(mod.KeywordFlagPhysicalDot).Tag(mod.Multiplier("CorruptingCryStageAfterFirst")).Source("Skill:SupportCorruptingCry"),
	},
	// src/Data/Skills/sup_str.lua:1539
	"SupportCruelty": {
		mod.NewFlag("Cruelty", true).Source("Skill:SupportCruelty"),
	},
	// src/Data/Skills/other.lua:4225
	"SupportCursePillarTriggerCurses": {
		mod.NewList("SkillData", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Source("Skill:SupportCursePillarTriggerCurses"),
	},
	// src/Data/Skills/sup_str.lua:1783
	"SupportDivineSentinel": {
		mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 15).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraPhysAsFire").Unscalable(true), mod.Condition("DivineSentinelPhysAsFire")).Source("Skill:SupportDivineSentinel"),
		mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, 15).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraPhysAsLightning").Unscalable(true), mod.Condition("DivineSentinelPhysAsLightning")).Source("Skill:SupportDivineSentinel"),
		mod.NewFloat("LifeRegenPercent", mod.TypeBase, 1.5).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraLifeRegen").Unscalable(true), mod.Condition("DivineSentinelRegenLife")).Source("Skill:SupportDivineSentinel"),
		mod.NewFloat("ManaRegen", mod.TypeIncrease, 30).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraManaRegen").Unscalable(true), mod.Condition("DivineSentinelRegenMana")).Source("Skill:SupportDivineSentinel"),
		mod.NewFloat("ChaosResist", mod.TypeBase, 23).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraChaosResist").Unscalable(true), mod.Condition("DivineSentinelChaosResistance")).Source("Skill:SupportDivineSentinel"),
		mod.NewFloat("CurseEffectOnSelf", mod.TypeIncrease, -50).Tag(mod.GlobalEffect("Aura").Name("TemplarAuraSelfCurseEffect").Unscalable(true), mod.Condition("DivineSentinelSelfCurseEffect")).Source("Skill:SupportDivineSentinel"),
	},
	// src/Data/Skills/sup_str.lua:2075
	"SupportEternalBlessing": {
		mod.NewList("SkillData", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Source("Skill:SupportEternalBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Source("Skill:SupportEternalBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Source("Skill:SupportEternalBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Source("Skill:SupportEternalBlessing"),
	},
	// src/Data/Skills/sup_int.lua:2724
	"SupportFreshMeat": {
		mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:Adrenaline", true)}).Tag(mod.Condition("FreshMeatActive")).Source("Skill:SupportFreshMeat"),
	},
	// src/Data/Skills/sup_int.lua:6931
	"SupportGreaterUnleash": {
		mod.NewFlag("HasSeals", true).Source("Skill:SupportGreaterUnleash"),
	},
	// src/Data/Skills/sup_str.lua:2822
	"SupportGuardiansBlessing": {
		mod.NewList("SkillData", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillType("Aura")).Source("Skill:SupportGuardiansBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillType("Aura")).Source("Skill:SupportGuardiansBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillType("Aura")).Source("Skill:SupportGuardiansBlessing"),
		mod.NewList("SkillData", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillType("Aura")).Source("Skill:SupportGuardiansBlessing"),
	},
	// src/Data/Skills/sup_dex.lua:3136
	"SupportMomentum": {
		mod.NewFlag("SupportedByMomentum", true).Source("Skill:SupportMomentum"),
	},
	// src/Data/Skills/sup_dex.lua:3349
	"SupportNightblade": {
		mod.NewFlag("SupportedByNightblade", true).Source("Skill:SupportNightblade"),
		mod.NewFlag("Condition:CanBeElusive", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:SupportNightblade"),
	},
	// src/Data/Skills/act_int.lua:17140
	"SupportSpellslinger": {
		mod.NewFlag("Condition:SupportedBySpellslinger", true).Source("Skill:SupportSpellslinger"),
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:SupportSpellslinger"),
	},
	// src/Data/Skills/sup_str.lua:5256
	"SupportTrauma": {
		mod.NewFlag("HasTrauma", true).Source("Skill:SupportTrauma"),
	},
	// src/Data/Skills/sup_int.lua:6788
	"SupportUnleash": {
		mod.NewFlag("HasSeals", true).Source("Skill:SupportUnleash"),
	},
	// src/Data/Skills/act_str.lua:6332
	"Sweep": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:Sweep"),
		mod.NewFlag("CannotBeEvaded", true).Tag(mod.SkillPart(2)).Source("Skill:Sweep"),
	},
	// src/Data/Skills/act_str.lua:6426
	"SweepAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:SweepAltX"),
		mod.NewFlag("CannotBeEvaded", true).Tag(mod.SkillPart(2)).Source("Skill:SweepAltX"),
	},
	// src/Data/Skills/act_str.lua:10879
	"Swordstorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "DoubleHitsWhenDualWielding", Value: 1}).Source("Skill:Swordstorm"),
	},
	// src/Data/Skills/spectre.lua:8959
	"TalismanT1TemporalChains": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:TalismanT1TemporalChains"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:TalismanT1TemporalChains"),
		mod.NewFloat("CooldownRecovery", mod.TypeOverride, 15).Source("Skill:TalismanT1TemporalChains"),
	},
	// src/Data/Skills/spectre.lua:8916
	"TalismanT1Vulnerability": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:TalismanT1Vulnerability"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:TalismanT1Vulnerability"),
		mod.NewFloat("CooldownRecovery", mod.TypeOverride, 15).Source("Skill:TalismanT1Vulnerability"),
	},
	// src/Data/Skills/spectre.lua:8873
	"TalismanT2EnfeebleAura": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:TalismanT2EnfeebleAura"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:TalismanT2EnfeebleAura"),
		mod.NewFloat("CooldownRecovery", mod.TypeOverride, 15).Source("Skill:TalismanT2EnfeebleAura"),
	},
	// src/Data/Skills/act_str.lua:12165
	"TectonicSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:TectonicSlam"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fissure Length:"}).Source("Skill:TectonicSlam"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 7}).Source("Skill:TectonicSlam"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Fissure Width:"}).Source("Skill:TectonicSlam"),
	},
	// src/Data/Skills/act_str.lua:12264
	"TectonicSlamAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:TectonicSlamAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fissure Length:"}).Source("Skill:TectonicSlamAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 7}).Source("Skill:TectonicSlamAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Fissure Width:"}).Source("Skill:TectonicSlamAltX"),
	},
	// src/Data/Skills/act_dex.lua:15964
	"TemporalChains": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:TemporalChains"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:TemporalChains"),
	},
	// src/Data/Skills/act_dex.lua:14941
	"ToxicRain": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:ToxicRain"),
		mod.NewFlag("DotCanStack", true).Source("Skill:ToxicRain"),
		mod.NewFlag("OneShotProj", true).Source("Skill:ToxicRain"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:ToxicRain"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Pod Area:"}).Source("Skill:ToxicRain"),
	},
	// src/Data/Skills/act_dex.lua:15033
	"ToxicRainAltX": {
		mod.NewFlag("OneShotProj", true).Source("Skill:ToxicRainAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:ToxicRainAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Pod Area:"}).Source("Skill:ToxicRainAltX"),
	},
	// src/Data/Skills/act_dex.lua:15127
	"ToxicRainAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:ToxicRainAltY"),
		mod.NewFlag("DotCanStack", true).Source("Skill:ToxicRainAltY"),
		mod.NewFlag("OneShotProj", true).Source("Skill:ToxicRainAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:ToxicRainAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Pod Area:"}).Source("Skill:ToxicRainAltY"),
	},
	// src/Data/Skills/other.lua:5731
	"TriggerSummonedAncestorKiloava": {
		mod.NewFloat("LifeOnBlock", mod.TypeBase, 1).Tag(mod.PercentStatVar("Life", "KiloavaRecoverOnBlock").Floor(), mod.GlobalEffect("Buff").Name("Spirit of Kiloava")).Source("Skill:TriggerSummonedAncestorKiloava"),
		mod.NewFloat("EnergyShieldOnBlock", mod.TypeBase, 1).Tag(mod.PercentStatVar("EnergyShield", "KiloavaRecoverOnBlock").Floor(), mod.GlobalEffect("Buff").Name("Spirit of Kiloava")).Source("Skill:TriggerSummonedAncestorKiloava"),
		mod.NewFloat("ManaOnBlock", mod.TypeBase, 1).Tag(mod.PercentStatVar("Mana", "KiloavaRecoverOnBlock").Floor(), mod.GlobalEffect("Buff").Name("Spirit of Kiloava")).Source("Skill:TriggerSummonedAncestorKiloava"),
	},
	// src/Data/Skills/other.lua:2662
	"TriggeredMoltenStrike": {
		mod.NewFlag("CannotSplit", true).Source("Skill:TriggeredMoltenStrike"),
	},
	// src/Data/Skills/other.lua:3720
	"TriggeredShockedGround": {
		mod.NewFloat("ShockedGroundEffect", mod.TypeBase, 15).Source("Skill:TriggeredShockedGround"),
	},
	// src/Data/Skills/other.lua:1051
	"TriggeredSummonLesserShrine": {
		mod.NewFlag("Condition:CanHaveLesserShrines", true).Tag(mod.GlobalEffect("Buff").Name("Shrines")).Source("Skill:TriggeredSummonLesserShrine"),
	},
	// src/Data/Skills/sup_int.lua:4392
	"TriggeredSupportGreaterKineticInstability": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:TriggeredSupportGreaterKineticInstability"),
	},
	// src/Data/Skills/sup_int.lua:4277
	"TriggeredSupportKineticInstability": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:TriggeredSupportKineticInstability"),
	},
	// src/Data/Skills/sup_dex.lua:4933
	"TriggeredSupportWindburst": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:TriggeredSupportWindburst"),
	},
	// src/Data/Skills/other.lua:5197
	"Twister": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:Twister"),
	},
	// src/Data/Skills/act_int.lua:263
	"VaalArc": {
		mod.NewFlag("Condition:CanBeLucky", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:VaalArc"),
	},
	// src/Data/Skills/act_dex.lua:2448
	"VaalBladeVortex": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 15}).Source("Skill:VaalBladeVortex"),
		mod.NewList("SkillData", mod.SkillData{Key: "VaalBladeVortexBlade", Value: 10}).Source("Skill:VaalBladeVortex"),
	},
	// src/Data/Skills/act_int.lua:2209
	"VaalBlight": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:VaalBlight"),
	},
	// src/Data/Skills/act_dex.lua:3558
	"VaalBurningArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:VaalBurningArrow"),
	},
	// src/Data/Skills/act_dex.lua:3813
	"VaalCausticArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:VaalCausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:VaalCausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "AoE Explosion:"}).Source("Skill:VaalCausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Caustic Ground:"}).Source("Skill:VaalCausticArrow"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VaalCausticArrow"),
		mod.NewFlag("dotIsCausticGround", true).Source("Skill:VaalCausticArrow"),
	},
	// src/Data/Skills/act_int.lua:2750
	"VaalClarity": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:VaalClarity"),
	},
	// src/Data/Skills/act_str.lua:1993
	"VaalCleave": {
		mod.NewFloat("CullPercent", mod.TypeMAX, 10).Tag(mod.SkillName("Cleave").Transfigured(true), mod.GlobalEffect("Buff").Name("Vaal Cleave").Unscalable(true)).Source("Skill:VaalCleave"),
	},
	// src/Data/Skills/act_int.lua:2993
	"VaalColdSnap": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VaalColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Burst:"}).Source("Skill:VaalColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Initial Chilled Ground:"}).Source("Skill:VaalColdSnap"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Final Chilled Ground:"}).Source("Skill:VaalColdSnap"),
	},
	// src/Data/Skills/act_dex.lua:4953
	"VaalCyclone": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:VaalCyclone"),
	},
	// src/Data/Skills/act_dex.lua:5457
	"VaalDetonateDead": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:VaalDetonateDead"),
	},
	// src/Data/Skills/act_int.lua:4531
	"VaalDiscipline": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:VaalDiscipline"),
	},
	// src/Data/Skills/act_str.lua:3534
	"VaalEarthquake": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:VaalEarthquake"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Tag(mod.SkillPart(1)).Source("Skill:VaalEarthquake"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 28}).Tag(mod.SkillPart(2)).Source("Skill:VaalEarthquake"),
	},
	// src/Data/Skills/act_int.lua:6238
	"VaalFirestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VaalFirestorm"),
		mod.NewFlag("dotIsBurningGround", true).Source("Skill:VaalFirestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Area of fireball explosion:"}).Source("Skill:VaalFirestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 20}).Source("Skill:VaalFirestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which fireballs fall:"}).Source("Skill:VaalFirestorm"),
	},
	// src/Data/Skills/act_int.lua:7131
	"VaalFlameblast": {
		mod.NewFloat("Multiplier:VaalFlameblastMaxStages", mod.TypeBase, 15).Source("Skill:VaalFlameblast"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 45}).Source("Skill:VaalFlameblast"),
	},
	// src/Data/Skills/act_dex.lua:8117
	"VaalFlickerStrike": {
		mod.NewFlag("OnlyFinalRepeat", true).Source("Skill:VaalFlickerStrike"),
		mod.NewFlag("FinalRepeatSumsDamage", true).Source("Skill:VaalFlickerStrike"),
	},
	// src/Data/Skills/act_str.lua:5169
	"VaalGlacialHammer": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:VaalGlacialHammer"),
	},
	// src/Data/Skills/act_dex.lua:9018
	"VaalGrace": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:VaalGrace"),
	},
	// src/Data/Skills/act_str.lua:5454
	"VaalGroundSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 39}).Source("Skill:VaalGroundSlam"),
	},
	// src/Data/Skills/act_dex.lua:9189
	"VaalHaste": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:VaalHaste"),
	},
	// src/Data/Skills/act_str.lua:6794
	"VaalImmortalCall": {
		mod.NewFlag("Condition:CannotDie", true).Tag(mod.GlobalEffect("Guard")).Source("Skill:VaalImmortalCall"),
	},
	// src/Data/Skills/spectre.lua:4106
	"VaalIncursionFirestorm": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Fireball explosion:"}).Source("Skill:VaalIncursionFirestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 25}).Source("Skill:VaalIncursionFirestorm"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Area in which Fireballs fall:"}).Source("Skill:VaalIncursionFirestorm"),
	},
	// src/Data/Skills/act_dex.lua:10605
	"VaalLightningArrow": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:VaalLightningArrow"),
	},
	// src/Data/Skills/act_int.lua:13197
	"VaalLightningWarpInstant": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:VaalLightningWarpInstant"),
	},
	// src/Data/Skills/act_str.lua:8155
	"VaalMoltenStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "ProjectileSpeedAppliesToMSAreaOfEffect", Value: 1}).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 9}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Ball area:"}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 2}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Chain Minimum Distance:"}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiary", Value: 25}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusTertiaryLabel", String: "Chain Maximum Distance:"}).Tag(mod.SkillPart(2)).Source("Skill:VaalMoltenStrike"),
		mod.NewFlag("CannotSplit", true).Source("Skill:VaalMoltenStrike"),
	},
	// src/Data/Skills/act_int.lua:14284
	"VaalPowerSiphon": {
		mod.NewFlag("OneShotProj", true).Source("Skill:VaalPowerSiphon"),
	},
	// src/Data/Skills/act_dex.lua:12636
	"VaalRainOfArrows": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 10}).Source("Skill:VaalRainOfArrows"),
		mod.NewFlag("OneShotProj", true).Source("Skill:VaalRainOfArrows"),
	},
	// src/Data/Skills/act_str.lua:11249
	"VaalReap": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 23}).Source("Skill:VaalReap"),
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VaalReap"),
	},
	// src/Data/Skills/act_dex.lua:12909
	"VaalReave": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 12}).Source("Skill:VaalReave"),
		mod.NewFloat("Multiplier:ReaveMaxStages", mod.TypeBase, 4).Source("Skill:VaalReave"),
	},
	// src/Data/Skills/act_str.lua:9427
	"VaalRejuvenationTotem": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:VaalRejuvenationTotem"),
	},
	// src/Data/Skills/act_int.lua:15649
	"VaalRighteousFire": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VaalRighteousFire"),
	},
	// src/Data/Skills/act_str.lua:10567
	"VaalSmite": {
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Lightning Strike area:"}).Source("Skill:VaalSmite"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Aura area:"}).Source("Skill:VaalSmite"),
	},
	// src/Data/Skills/act_int.lua:18429
	"VaalStormCall": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:VaalStormCall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Lightning tether area:"}).Source("Skill:VaalStormCall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondary", Value: 40}).Source("Skill:VaalStormCall"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Final Lightning Strike area:"}).Source("Skill:VaalStormCall"),
	},
	// src/Data/Skills/act_str.lua:12784
	"VaalVolcanicFissure": {
		mod.NewFlag("CannotSplit", true).Source("Skill:VaalVolcanicFissure"),
	},
	// src/Data/Skills/act_dex.lua:16302
	"VenomGyre": {
		mod.NewFlag("Condition:ReturningProjectile", true).Tag(mod.SkillPart(2)).Source("Skill:VenomGyre"),
	},
	// src/Data/Skills/act_dex.lua:16482
	"ViperStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "PoisonIsSkillEffect", Value: 1}).Source("Skill:ViperStrike"),
	},
	// src/Data/Skills/act_dex.lua:16579
	"ViperStrikeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "PoisonIsSkillEffect", Value: 1}).Source("Skill:ViperStrikeAltX"),
	},
	// src/Data/Skills/act_str.lua:12514
	"Vitality": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Vitality"),
	},
	// src/Data/Skills/other.lua:4912
	"VolatileAnomaly": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VolatileAnomaly"),
	},
	// src/Data/Skills/act_dex.lua:16678
	"VolatileDead": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:VolatileDead"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Orb Explosion:"}).Source("Skill:VolatileDead"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Corpse Explosion:"}).Source("Skill:VolatileDead"),
	},
	// src/Data/Skills/act_dex.lua:16781
	"VolatileDeadAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:VolatileDeadAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Orb Explosion:"}).Source("Skill:VolatileDeadAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Corpse Explosion:"}).Source("Skill:VolatileDeadAltX"),
	},
	// src/Data/Skills/act_dex.lua:16885
	"VolatileDeadAltY": {
		mod.NewList("SkillData", mod.SkillData{Key: "ExplodeCorpse", Value: 1}).Tag(mod.SkillPart(2)).Source("Skill:VolatileDeadAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Orb Explosion:"}).Source("Skill:VolatileDeadAltY"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Corpse Explosion:"}).Source("Skill:VolatileDeadAltY"),
	},
	// src/Data/Skills/act_str.lua:12612
	"VolcanicFissure": {
		mod.NewFlag("CannotSplit", true).Source("Skill:VolcanicFissure"),
	},
	// src/Data/Skills/act_int.lua:20463
	"VoltaxicBurst": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:VoltaxicBurst"),
	},
	// src/Data/Skills/act_int.lua:20551
	"Vortex": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:Vortex"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Hit:"}).Source("Skill:Vortex"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Ground Degen:"}).Source("Skill:Vortex"),
	},
	// src/Data/Skills/act_int.lua:20643
	"VortexAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "DotIsArea", Value: 1}).Source("Skill:VortexAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusLabel", String: "Initial Hit:"}).Source("Skill:VortexAltX"),
		mod.NewList("SkillData", mod.SkillData{Key: "RadiusSecondaryLabel", String: "Ground Degen:"}).Source("Skill:VortexAltX"),
	},
	// src/Data/Skills/act_str.lua:12874
	"Vulnerability": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Vulnerability"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 22}).Source("Skill:Vulnerability"),
	},
	// src/Data/Skills/act_str.lua:13047
	"WarBanner": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Source("Skill:WarBanner"),
		mod.NewFloat("AuraEffect", mod.TypeMore, -100).Tag(mod.Condition("BannerPlanted").Neg(true)).Source("Skill:WarBanner"),
		mod.NewFlag("Condition:AffectedByPlacedBanner", true).Tag(mod.Condition("BannerPlanted"), mod.GlobalEffect("Buff")).Source("Skill:WarBanner"),
	},
	// src/Data/Skills/other.lua:5390
	"WardShatter": {
		mod.NewList("SkillData", mod.SkillData{Key: "ShowAverage", Value: 1}).Source("Skill:WardShatter"),
	},
	// src/Data/Skills/act_str.lua:12966
	"WarlordsMark": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:WarlordsMark"),
	},
	// src/Data/Skills/act_dex.lua:17108
	"WildStrike": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Tag(mod.SkillPart(2)).Source("Skill:WildStrike"),
	},
	// src/Data/Skills/act_dex.lua:17249
	"WildStrikeAltX": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 24}).Tag(mod.SkillPart(2)).Source("Skill:WildStrikeAltX"),
	},
	// src/Data/Skills/act_int.lua:21020
	"WinterOrb": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 16}).Source("Skill:WinterOrb"),
	},
	// src/Data/Skills/act_int.lua:21162
	"WintertideBrand": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 20}).Source("Skill:WintertideBrand"),
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:WintertideBrand"),
		mod.NewList("SkillData", mod.SkillData{Key: "DebuffTertiary", Value: 1}).Source("Skill:WintertideBrand"),
	},
	// src/Data/Skills/act_int.lua:21259
	"Wither": {
		mod.NewList("SkillData", mod.SkillData{Key: "Debuff", Value: 1}).Source("Skill:Wither"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:Wither"),
	},
	// src/Data/Skills/act_dex.lua:17333
	"WitheringStep": {
		mod.NewFlag("Condition:CanBeElusive", true).Tag(mod.GlobalEffect("Buff")).Source("Skill:WitheringStep"),
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 26}).Source("Skill:WitheringStep"),
	},
	// src/Data/Skills/act_int.lua:21344
	"Wrath": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Wrath"),
	},
	// src/Data/Skills/act_int.lua:21426
	"Zealotry": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 40}).Source("Skill:Zealotry"),
	},
	// src/Data/Skills/minion.lua:732
	"ZombieSlam": {
		mod.NewList("SkillData", mod.SkillData{Key: "Radius", Value: 18}).Source("Skill:ZombieSlam"),
	},
}

// multiplierLimitVar builds a Multiplier tag limited by another multiplier rather than a
// constant (PoB's `limitVar`). mod.MultiplierTag has the field but no setter.
func multiplierLimitVar(variable string, limitVar string) *mod.MultiplierTag {
	t := mod.Multiplier(variable)
	t.TagLimitVariable = &limitVar
	return t
}

// perStatBase builds a PerStat tag with a flat addend (PoB's `base`). mod.PerStatTag has
// the field but no setter.
func perStatBase(divide float64, base float64, stats ...string) *mod.PerStatTag {
	t := mod.PerStat(divide, stats...)
	t.Base = base
	return t
}
