package calculator

import "github.com/Vilsol/go-pob/mod"

type ConfigApplyFunc func(val interface{}, modList *ModList, enemyModList *ModList)

var configurations = map[string]ConfigApplyFunc{
	/*
		   	"detonateDeadCorpseLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewList("SkillData", mod.SkillData{
		   			Key:   "corpseLife",
		   			Value: val,
		   		}).Source("Config"))
		   	},
		   	"conditionStationary": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   	out := float64(0)
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
		   "conditionMoving": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:Moving", true).Source("Config"))
		   	},
		   "conditionInsane": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:Insane", true).Source("Config"))
		   	},
		   "conditionFullLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:FullLife", true).Source("Config"))
		   	},
		   "conditionLowLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:LowLife", true).Source("Config"))
		   	},
		   "conditionFullMana": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:FullMana", true).Source("Config"))
		   	},
		   "conditionLowMana": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:LowMana", true).Source("Config"))
		   	},
		   "conditionFullEnergyShield": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:FullEnergyShield", true).Source("Config"))
		   	},
		   "conditionLowEnergyShield": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:LowEnergyShield", true).Source("Config"))
		   	},
		   "conditionHaveEnergyShield": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:HaveEnergyShield", true).Source("Config"))
		   	},
		   "minionsConditionFullLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewList("MinionModifier", { mod = modLib.createMod("Condition:FullLife", "FLAG", true).Source("Config")) }, "Config")
		   	},
		   "minionsConditionCreatedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:MinionsCreatedRecently", true).Source("Config"))
		   	},
		   "lifeRegenMode": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "AVERAGE" {
		   			modList.AddMod(mod.NewFlag("Condition:LifeRegenBurstAvg", true).Source("Config"))
		   		} else if val == "FULL" {
		   			modList.AddMod(mod.NewFlag("Condition:LifeRegenBurstFull", true).Source("Config"))
		   		}
		   	},
		   "armourCalculationMode": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "MAX" {
		   			modList.AddMod(mod.NewFlag("Condition:ArmourMax", true).Source("Config"))
		   		} else if val == "AVERAGE" {
		   			modList.AddMod(mod.NewFlag("Condition:ArmourAvg", true).Source("Config"))
		   		}
		   	},
		   "warcryMode": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "MAX" {
		   			modList.AddMod(mod.NewFlag("Condition:WarcryMaxHit", true).Source("Config"))
		   		}
		   	},
		   "EVBypass": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:EVBypass", true).Source("Config"))
		   	},
		   "targetBrandedEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:TargetingBrandedEnemy", true).Source("Config"))
		   	},
		   "aspectOfTheAvianAviansMight": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:AviansMightActive", true).Source("Config"))
		   	},
		   "aspectOfTheAvianAviansFlight": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:AviansFlightActive", true).Source("Config"))
		   	},
		   "aspectOfTheCatCatsStealth": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:CatsStealthActive", true).Source("Config"))
		   	},
		   "aspectOfTheCatCatsAgility": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:CatsAgilityActive", true).Source("Config"))
		   	},
		   "overrideCrabBarriers": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("CrabBarriers", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "aspectOfTheSpiderWebStacks": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraSkillMod", "LIST", { mod = modLib.createMod("Multiplier:SpiderWebApplyStack", "BASE", val) }, "Config", { type = "SkillName", skillName = "Aspect of the Spider" })
		   	},
		   "bannerPlanted": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:BannerPlanted", true).Source("Config"))
		   	},
		   "bannerStages": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:BannerStage", "BASE", math.Min(val, 50), "Config")
		   	},
		   "bladestormInBloodstorm": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Condition:BladestormInBloodstorm", "FLAG", true, "Config", { type = "SkillName", skillName = "Bladestorm" })
		   	},
		   "bladestormInSandstorm": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Condition:BladestormInSandstorm", "FLAG", true, "Config", { type = "SkillName", skillName = "Bladestorm" })
		   	},
		   "bonechillEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		enemyModList.AddMod(mod.NewFloat("BonechillEffect", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
		   		enemyModList.AddMod(mod.NewFloat("DesiredBonechillEffect", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
		   	},
		   "boneshatterTraumaStacks": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:TraumaStacks", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "ActiveBrands": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:ConfigActiveBrands", mod.TypeBase, val).Source("Config"))
		   	},
		   "BrandsAttachedToEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:ConfigBrandsAttachedToEnemy", mod.TypeBase, val).Source("Config"))
		   	},
		   "BrandsInLastQuarter": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:BrandLastQuarter", true).Source("Config"))
		   	},
		   "carrionGolemNearbyMinion": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:NearbyNonGolemMinion", mod.TypeBase, val).Source("Config"))
		   	},
		   "closeCombatCombatRush": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:CombatRushActive", true).Source("Config"))
		   	},
		   "overrideCruelty": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Cruelty", "OVERRIDE", math.Min(val, 40), "Config", { type = "Condition", var = "Combat" })
		   	},
		   "channellingCycloneCheck": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:ChannellingCyclone", true).Source("Config"))
		   	},
		   "darkPactSkeletonLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "skeletonLife", value = val }, "Config", { type = "SkillName", skillName = "Dark Pact" })
		   	},
		   "deathmarkDeathmarkActive": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:EnemyHasDeathmark", true).Source("Config"))
		   	},
		   "elementalArmyExposureType": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "Fire" {
		   			modList.AddMod(mod.NewFloat("FireExposureChance", mod.TypeBase, 100).Source("Config"))
		   		} else if val == "Cold" {
		   			modList.AddMod(mod.NewFloat("ColdExposureChance", mod.TypeBase, 100).Source("Config"))
		   		} else if val == "Lightning" {
		   			modList.AddMod(mod.NewFloat("LightningExposureChance", mod.TypeBase, 100).Source("Config"))
		   		}
		   	},
		   "energyBladeActive": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:EnergyBladeActive", true).Source("Config"))
		   	},
		   "embraceMadnessActive": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:AffectedByGloriousMadness", true).Source("Config"))
		   	},
		   "feedingFrenzyFeedingFrenzyActive": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:FeedingFrenzyActive", true).Source("Config"))
		   		modList.AddMod(mod.NewList("MinionModifier", { mod = modLib.createMod("Damage", "MORE", 10).Source("Feeding Frenzy")) }, "Config")
		   		modList.AddMod(mod.NewList("MinionModifier", { mod = modLib.createMod("MovementSpeed", "INC", 10).Source("Feeding Frenzy")) }, "Config")
		   		modList.AddMod(mod.NewList("MinionModifier", { mod = modLib.createMod("Speed", "INC", 10).Source("Feeding Frenzy")) }, "Config")
		   	},
		   "flameWallAddedDamage": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:FlameWallAddedDamage", true).Source("Config"))
		   	},
		   "frostboltExposure": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("ColdExposureChance", mod.TypeBase, 20).Source("Config"))
		   	},
		   "frostShieldStages": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:FrostShieldStage", mod.TypeBase, val).Source("Config"))
		   	},
		   "greaterHarbingerOfTimeSlipstream": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:GreaterHarbingerOfTime", true).Source("Config"))
		   	},
		   "harbingerOfTimeSlipstream": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:HarbingerOfTime", true).Source("Config"))
		   	},
		   "multiplierHexDoom": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:HexDoomStack", mod.TypeBase, val).Source("Config"))
		   	},
		   "heraldOfAgonyVirulenceStack": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:VirulenceStack", mod.TypeBase, val).Source("Config"))
		   	},
		   "iceNovaCastOnFrostbolt": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Condition:CastOnFrostbolt", "FLAG", true, "Config", { type = "SkillName", skillName = "Ice Nova" })
		   	},
		   "infusedChannellingInfusion": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:InfusionActive", true).Source("Config"))
		   	},
		   "innervateInnervation": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:InnervationActive", true).Source("Config"))
		   	},
		   "intensifyIntensity": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:Intensity", mod.TypeBase, val).Source("Config"))
		   	},
		   "meatShieldEnemyNearYou": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:MeatShieldEnemyNearYou", true).Source("Config"))
		   	},
		   "plagueBearerState": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "INC" {
		   			modList.AddMod(mod.NewFlag("Condition:PlagueBearerIncubating", true).Source("Config"))
		   		} else if val == "INF" {
		   			modList.AddMod(mod.NewFlag("Condition:PlagueBearerInfecting", true).Source("Config"))
		   		}
		   	},
		   "perforateSpikeOverlap": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:PerforateSpikeOverlap", "BASE", val, "Config", { type = "SkillName", skillName = "Perforate" })
		   	},
		   "physicalAegisDepleted": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:PhysicalAegisDepleted", true).Source("Config"))
		   	},
		   "prideEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "MAX" {
		   			modList.AddMod(mod.NewFlag("Condition:PrideMaxEffect", true).Source("Config"))
		   		}
		   	},
		   "sacrificedRageCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:RageSacrificed", mod.TypeBase, val).Source("Config"))
		   	},
		   "raiseSpectreEnableBuffs": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "enable", value = true }, "Config", { type = "SkillType", skillType = SkillType.Buff }, { type = "SkillName", skillName = "Raise Spectre", summonSkill = true })
		   	},
		   "raiseSpectreEnableCurses": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "enable", value = true }, "Config", { type = "SkillType", skillType = SkillType.Hex }, { type = "SkillName", skillName = "Raise Spectre", summonSkill = true })
		   		modList.AddMod("SkillData", "LIST", { key = "enable", value = true }, "Config", { type = "SkillType", skillType = SkillType.Mark }, { type = "SkillName", skillName = "Raise Spectre", summonSkill = true })
		   	},
		   "raiseSpectreBladeVortexBladeCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "dpsMultiplier", value = val }, "Config", { type = "SkillId", skillId = "DemonModularBladeVortexSpectre" })
		   		modList.AddMod("SkillData", "LIST", { key = "dpsMultiplier", value = val }, "Config", { type = "SkillId", skillId = "GhostPirateBladeVortexSpectre" })
		   	},
		   "raiseSpectreKaomFireBeamTotemStage": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:KaomFireBeamTotemStage", mod.TypeBase, val).Source("Config"))
		   	},
		   "raiseSpectreEnableSummonedUrsaRallyingCry": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "enable", value = true }, "Config", { type = "SkillId", skillId = "DropBearSummonedRallyingCry" })
		   	},
		   "raiseSpidersSpiderCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:RaisedSpider", "BASE", math.Min(val, 20), "Config")
		   	},
		   "animateWeaponLingeringBlade": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:AnimatingLingeringBlades", true).Source("Config"))
		   	},
		   "sigilOfPowerStages": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:SigilOfPowerStage", mod.TypeBase, val).Source("Config"))
		   	},
		   "siphoningTrapAffectedEnemies": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:EnemyAffectedBySiphoningTrap", mod.TypeBase, val).Source("Config"))
		   		modList.AddMod(mod.NewFlag("Condition:SiphoningTrapSiphoning", true).Source("Config"))
		   	},
		   "configSnipeStages": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:SnipeStage", "BASE", math.Min(val, 6), "Config")
		   	},
		   "configResonanceCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:ResonanceCount", "BASE", m_max(math.Min(val, 50), 0), "Config")
		   	},
		   "configSpectralWolfCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:SpectralWolfCount", "BASE", math.Min(val, 10), "Config")
		   	},
		   "bloodSandStance": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "SAND" {
		   			modList.AddMod(mod.NewFlag("Condition:SandStance", true).Source("Config"))
		   		}
		   	},
		   "changedStance": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:ChangedStanceRecently", true).Source("Config"))
		   	},
		   "shardsConsumed": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:SteelShardConsumed", "BASE", math.Min(val, 12), "Config")
		   	},
		   "steelWards": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:SteelWardCount", mod.TypeBase, val).Source("Config"))
		   	},
		   "stormRainBeamOverlap": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "beamOverlapMultiplier", value = val }, "Config", { type = "SkillName", skillName = "Storm Rain" })
		   	},
		   "summonHolyRelicEnableHolyRelicBoon": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:HolyRelicBoonActive", true).Source("Config"))
		   	},
		   "summonLightningGolemEnableWrath": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "enable", value = true }, "Config", { type = "SkillId", skillId = "LightningGolemWrath" })
		   	},
		   "nearbyBleedingEnemies": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:NearbyBleedingEnemies", "BASE", val, "Config" )
		   	},
		   "toxicRainPodOverlap": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "podOverlapMultiplier", value = val }, "Config", { type = "SkillName", skillName = "Toxic Rain" })
		   	},
		   "hoaOverkill": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "hoaOverkill", value = val }, "Config", { type = "SkillName", skillName = "Herald of Ash" })
		   	},
		   "voltaxicBurstSpellsQueued": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:VoltaxicWaitingStages", mod.TypeBase, val).Source("Config"))
		   	},
		   "vortexCastOnFrostbolt": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Condition:CastOnFrostbolt", "FLAG", true, "Config", { type = "SkillName", skillName = "Vortex" })
		   	},
		   "ColdSnapBypassCD": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("CooldownRecovery", "OVERRIDE", 0, "Config", { type = "SkillName", skillName = "Cold Snap" })
		   	},
		   "multiplierWarcryPower": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeOverride, val).Source("Config"))
		   	},
		   "waveOfConvictionExposureType": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val == "Fire" {
		   			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionFireExposureActive", true).Source("Config"))
		   		} else if val == "Cold" {
		   			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionColdExposureActive", true).Source("Config"))
		   		} else if val == "Lightning" {
		   			modList.AddMod(mod.NewFlag("Condition:WaveOfConvictionLightningExposureActive", true).Source("Config"))
		   		}
		   	},
		   "MoltenShellDamageMitigated": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "MoltenShellDamageMitigated", value = val }, "Config", { type = "SkillName", skillName = "Molten Shell" })
		   	},
		   "VaalMoltenShellDamageMitigated": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("SkillData", "LIST", { key = "VaalMoltenShellDamageMitigated", value = val }, "Config", { type = "SkillName", skillName = "Molten Shell" })
		   	},
		   "enemyHasPhysicalReduction": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		enemyModList.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, val).Source("Config"))
		   	},
		   "enemyIsHexproof": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		enemymodList.AddMod(mod.NewFlag("Hexproof", true).Source("Config"))
		   	},
		   "enemyHasLessCurseEffectOnSelf": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -val).Source("Config"))
		   		}
		   	},
		   "enemyCanAvoidPoisonBlindBleed": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			enemyModList.AddMod(mod.NewFloat("AvoidPoison", mod.TypeBase, val).Source("Config"))
		   			enemyModList.AddMod(mod.NewFloat("AvoidBleed", mod.TypeBase, val).Source("Config"))
		   		}
		   	},
		   "enemyHasResistances": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		//map := { ["LOW"] = {20,15}, ["MID"] = {30,20}, ["HIGH"] = {40,25} }
		   		//if map[val] {
		   		//	enemyModList.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, map[val][1]).Source("Config"))
		   		//	enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, map[val][2]).Source("Config"))
		   		//}
		   	},
		   "playerHasElementalEquilibrium": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewList("Keystone", "Elemental Equilibrium").Source("Config"))
		   	},
		   "playerCannotLeech": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		enemymodList.AddMod(mod.NewFlag("CannotLeechLifeFromSelf", true).Source("Config"))
		   		enemymodList.AddMod(mod.NewFlag("CannotLeechManaFromSelf", true).Source("Config"))
		   	},
		   "playerGainsReducedFlaskCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			modList.AddMod(mod.NewFloat("FlaskChargesGained", mod.TypeIncrease, -val).Source("Config"))
		   		}
		   	},
		   "playerHasMinusMaxResist": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			modList.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, -val).Source("Config"))
		   			modList.AddMod(mod.NewFloat("ColdResistMax", mod.TypeBase, -val).Source("Config"))
		   			modList.AddMod(mod.NewFloat("LightningResistMax", mod.TypeBase, -val).Source("Config"))
		   			modList.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, -val).Source("Config"))
		   		}
		   	},
		   "playerHasLessAreaOfEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			modList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeMore, -val).Source("Config"))
		   		}
		   	},
		   "enemyCanAvoidStatusAilment": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			enemyModList.AddMod(mod.NewFloat("AvoidIgnite", mod.TypeBase, val).Source("Config"))
		   			enemyModList.AddMod(mod.NewFloat("AvoidShock", mod.TypeBase, val).Source("Config"))
		   			enemyModList.AddMod(mod.NewFloat("AvoidFreeze", mod.TypeBase, val).Source("Config"))
		   		}
		   	},
		   "enemyHasIncreasedAccuracy": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			modList.AddMod(mod.NewFlag("DodgeChanceIsUnlucky", true).Source("Config"))
		   			enemyModList.AddMod(mod.NewFloat("Accuracy", mod.TypeIncrease, val).Source("Config"))
		   		}
		   	},
		   "playerHasLessArmourAndBlock": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		//map := { ["LOW"] = {20,20}, ["MID"] = {30,25}, ["HIGH"] = {40,30} }
		   		//if map[val] {
		   		//	modList.AddMod(mod.NewFloat("BlockChance", mod.TypeIncrease, -map[val][1]).Source("Config"))
		   		//	modList.AddMod(mod.NewFloat("Armour", mod.TypeMore, -map[val][2]).Source("Config"))
		   		//}
		   	},
		   "playerHasPointBlank": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewList("Keystone", "Point Blank").Source("Config"))
		   	},
		   "playerHasLessLifeESRecovery": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			modList.AddMod(mod.NewFloat("LifeRecoveryRate", mod.TypeMore, -val).Source("Config"))
		   			modList.AddMod(mod.NewFloat("EnergyShieldRecoveryRate", mod.TypeMore, -val).Source("Config"))
		   		}
		   	},
		   "playerCannotRegenLifeManaEnergyShield": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("NoLifeRegen", true).Source("Config"))
		   		modList.AddMod(mod.NewFlag("NoEnergyShieldRegen", true).Source("Config"))
		   		modList.AddMod(mod.NewFlag("NoManaRegen", true).Source("Config"))
		   	},
		   "enemyTakesReducedExtraCritDamage": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		if val != 0 {
		   			enemyModList.AddMod(mod.NewFloat("SelfCritMultiplier", mod.TypeIncrease, -val).Source("Config"))
		   		}
		   	},
		   "multiplierSextant": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:Sextant", "BASE", math.Min(val, 5), "Config")
		   	},
		   "playerCursedWithAssassinsMark": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "AssassinsMark", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithConductivity": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Conductivity", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithDespair": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Despair", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithElementalWeakness": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "ElementalWeakness", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWi{feeble": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Enfeeble", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithFlammability": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Flammability", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithFrostbite": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Frostbite", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithPoachersMark": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "PoachersMark", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithProjectileWeakness": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "ProjectileWeakness", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithPunishment": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Punishment", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithTemporalChains": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "TemporalChains", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithVulnerability": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "Vulnerability", level = val, applyToPlayer = true })
		   	},
		   "playerCursedWithWarlordsMark": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("ExtraCurse", "LIST", { skillId = "WarlordsMark", level = val, applyToPlayer = true })
		   	},
		   "usePowerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UsePowerCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overridePowerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("PowerCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "useFrenzyCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseFrenzyCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideFrenzyCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("FrenzyCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "use}uranceCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Use}uranceCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "override}uranceCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("}uranceCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "useSiphoningCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseSiphoningCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideSiphoningCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("SiphoningCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "useChallengerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseChallengerCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideChallengerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("ChallengerCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "useBlitzCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseBlitzCharges", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideBlitzCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("BlitzCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "multiplierGaleForce": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("Multiplier:GaleForce", "BASE", val, "Config", { type = "IgnoreCond" }, { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanGainGaleForce" })
		   	},
		   "overrideInspirationCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("InspirationCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "useGhostShrouds": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseGhostShrouds", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideGhostShrouds": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("GhostShrouds", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "waitForMaxSeals": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("UseMaxUnleash", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "overrideBloodCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("BloodCharges", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "minionsUsePowerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("UsePowerCharges", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "minionsUseFrenzyCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("UseFrenzyCharges", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "minionsUse}uranceCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("Use}uranceCharges", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "minionsOverridePowerCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("PowerCharges", "OVERRIDE", val, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "minionsOverrideFrenzyCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("FrenzyCharges", "OVERRIDE", val, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "minionsOverride}uranceCharges": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("}uranceCharges", "OVERRIDE", val, "Config", { type = "Condition", var = "Combat" }) }, "Config")
		   	},
		   "multiplierRampage": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFloat("Multiplier:Rampage", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "conditionFocused": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:Focused", true).Source("Config").Tag(mod.Condition("Combat")))
		   	},
		   "buffLifetap": func(val interface{}, modList *ModList, enemyModList *ModList) {
		   		modList.AddMod(mod.NewFlag("Condition:Lifetap", true).Source("Config").Tag(mod.Condition("Combat")))
		   		modList.AddMod(mod.NewFloat("FlaskLifeRecovery", mod.TypeIncrease, 20).Source("Lifetap"))
		   	},
	*/
	"buffOnslaught": func(val interface{}, modList *ModList, enemyModList *ModList) {
		modList.AddMod(mod.NewFlag("Condition:Onslaught", true).Source("Config").Tag(mod.Condition("Combat")))
	},
	/*
	   "minionBuffOnslaught": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("Condition:Onslaught", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) })
	   	},
	   "buffUnholyMight": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UnholyMight", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "minionbuffUnholyMight": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("Condition:UnholyMight", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) })
	   	},
	   "buffPhasing": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Phasing", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffFortification": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Fortified", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "overrideFortification": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("FortificationStacks", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffTailwind": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Tailwind", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffAdrenaline": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Adrenaline", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffAlchemistsGenius": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Condition:AlchemistsGenius", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanHaveAlchemistGenius" })
	   	},
	   "buffVaalArcLuckyHits": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("LuckyHits", "FLAG", true, "Config", { type = "Condition", varList = { "Combat", "CanBeLucky" } }, { type = "SkillName", skillNameList = { "Arc", "Vaal Arc" } })
	   	},
	   "buffElusive": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Condition:Elusive", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanBeElusive" })
	   		modList.AddMod("Elusive", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanBeElusive" })
	   	},
	   "overrideBuffElusive": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("ElusiveEffect", "OVERRIDE", val, "Config", {type = "GlobalEffect", effectType = "Buff" })
	   	},
	   "buffDivinity": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Divinity", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierDefiance": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Multiplier:Defiance", "BASE", math.Min(val, 10), "Config", { type = "Condition", var = "Combat" })
	   	},
	   "multiplierRage": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Multiplier:RageStack", "BASE", val, "Config", { type = "IgnoreCond" }, { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanGainRage" })
	   	},
	   "conditionLeeching": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionLeechingLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:LeechingLife", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionLeechingEnergyShield": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:LeechingEnergyShield", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionLeechingMana": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:LeechingMana", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Leeching", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsingFlask": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHaveTotem": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HaveTotem", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSummonedTotemRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SummonedTotemRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "TotemsSummoned": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("TotemsSummoned", mod.TypeOverride, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:HaveTotem", val >= 1).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSummonedGolemInPast8Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SummonedGolemInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSummonedGolemInPast10Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SummonedGolemInPast10Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierNearbyAlly": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:NearbyAlly", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierNearbyCorpse": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:NearbyCorpse", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierSummonedMinion": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:SummonedMinion", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionOnConsecratedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:OnConsecratedGround", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("Condition:OnConsecratedGround", "FLAG", true, "Config", { type = "Condition", var = "Combat" }) })
	   	},
	   "conditionOnFungalGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:OnFungalGround", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionOnBurningGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:OnBurningGround", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionOnChilledGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:OnChilledGround", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionOnShockedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:OnShockedGround", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBlinded": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Blinded", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBurning": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionIgnited": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Ignited", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionChilled": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionChilledEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("ChillVal", mod.TypeOverride, val).Source("Chill").Tag(mod.Condition("Chilled")))
	   	},
	   "conditionSelfChill": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ChilledSelf", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionFrozen": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Frozen", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionShocked": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFloat("DamageTaken", mod.TypeBase, 15).Source("Shock").Tag(mod.Condition("Shocked")))
	   	},
	   "conditionBleeding": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionPoisoned": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierPoisonOnSelf": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:PoisonStack", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionAgainstDamageOverTime": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:AgainstDamageOverTime", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierNearbyEnemies": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:NearbyEnemies", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:OnlyOneNearbyEnemy", val == 1).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierNearbyRareOrUniqueEnemies": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:NearbyRareOrUniqueEnemies", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFloat("Multiplier:NearbyEnemies", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:AtMostOneNearbyRareOrUniqueEnemy", val <= 1).Source("Config").Tag(mod.Condition("Combat")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:NearbyRareOrUniqueEnemy", val >= 1).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionCritRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:SkillCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSkillCritRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SkillCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionCritWithHeraldSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CritWithHeraldSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "LostNonVaalBuffRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:LostNonVaalBuffRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionNonCritRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:NonCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionChannelling": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Channelling", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitRecentlyWithWeapon": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitRecentlyWithWeapon", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:KilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:KilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionKilledLast3Seconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:KilledLast3Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionKilledPosionedLast2Seconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:KilledPosionedLast2Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionTotemsNotSummonedInPastTwoSeconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:NoSummonedTotemsInPastTwoSeconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionTotemsKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:TotemsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedBrandRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedBrandRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierTotemsKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledByTotemsRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:TotemsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionMinionsKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:MinionsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionMinionsDiedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:MinionsDiedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierMinionsKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:EnemyKilledByMinionsRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:MinionsKilledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionKilledAffectedByDoT": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:KilledAffectedByDotRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierShockedEnemyKilledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:ShockedEnemyKilledRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionFrozenEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:FrozenEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionChilledEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ChilledEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionShatteredEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ShatteredEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionIgnitedEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:IgnitedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionShockedEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ShockedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionStunnedEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:StunnedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierPoisonAppliedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:PoisonAppliedRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierLifeSpentRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:LifeSpentRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierManaSpentRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:ManaSpentRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBeenHitRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierBeenHitRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:BeenHitRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", 1 <= val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBeenHitByAttackRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitByAttackRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBeenCritRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BeenCritRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionConsumed12SteelShardsRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Consumed12SteelShardsRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionGainedPowerChargeRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:GainedPowerChargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionGainedFrenzyChargeRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:GainedFrenzyChargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBeenSavageHitRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BeenSavageHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitByFireDamageRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitByFireDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitByColdDamageRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitByColdDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitByLightningDamageRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitByLightningDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHitBySpellDamageRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HitBySpellDamageRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionTakenFireDamageFromEnemyHitRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:TakenFireDamageFromEnemyHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBlockedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBlockedAttackRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BlockedAttackRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBlockedSpellRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BlockedSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:BlockedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionEnergyShieldRechargeRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:EnergyShieldRechargeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionStoppedTakingDamageOverTimeRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:StoppedTakingDamageOverTimeRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionConvergence": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Condition:Convergence", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanGainConvergence" })
	   	},
	   "buffP}ulum": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		if val == "AREA" {
	   			modList.AddMod(mod.NewFlag("Condition:P}ulumOfDestructionAreaOfEffect", true).Source("Config").Tag(mod.Condition("Combat")))
	   		} else if val == "DAMAGE" {
	   			modList.AddMod(mod.NewFlag("Condition:P}ulumOfDestructionElementalDamage", true).Source("Config").Tag(mod.Condition("Combat")))
	   		}
	   	},
	   "buffConflux": func(val interface{}, modList *ModList, enemyModList *ModList) {
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
	   "buffBastionOfHope": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BastionOfHopeActive", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffNgamahuFlamesAdvance": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:NgamahuFlamesAdvance", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffHerEmbrace": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("HerEmbrace", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanGainHerEmbrace" })
	   	},
	   "conditionUsedSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierSkillUsedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:SkillUsedRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionAttackedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:AttackedRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionCastSpellRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CastSpellRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionCastLast1Seconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CastLast1Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierCastLast8Seconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:CastLast8Seconds", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedFireSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedFireSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedColdSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedColdSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedMinionSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedMinionSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedTravelSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedTravelSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedDashRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CastDashRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedTravelSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedMovementSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedMovementSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedVaalSkillRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedVaalSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSoulGainPrevention": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SoulGainPrevention", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedWarcryRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedWarcryRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionUsedWarcryInPast8Seconds": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierMineDetonatedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:MineDetonatedRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierTrapTriggeredRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:TrapTriggeredRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionThrownTrapOrMineRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:TrapOrMineThrownRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionCursedEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CursedEnemyRecently", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionCastMarkRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:CastMarkRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionSpawnedCorpseRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SpawnedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionConsumedCorpseRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionConsumedCorpseInPast2Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseInPast2Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierCorpseConsumedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:CorpseConsumedRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:ConsumedCorpseRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierWarcryUsedRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Multiplier:WarcryUsedRecently", "BASE", math.Min(val, 100), "Config", { type = "Condition", var = "Combat" })
	   		modList.AddMod(mod.NewFlag("Condition:UsedWarcryRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedWarcryInPast8Seconds", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:UsedSkillRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionTauntedEnemyRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:TauntedEnemyRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionLost}uranceChargeInPast8Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:Lost}uranceChargeInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplier}uranceChargesLostRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:}uranceChargesLostRecently", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewFlag("Condition:Lost}uranceChargeInPast8Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionBlockedHitFromUniqueEnemyInPast10Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:BlockedHitFromUniqueEnemyInPast10Sec", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "BlockedPast10Sec": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:BlockedPast10Sec", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionImpaledRecently": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:ImpaledRecently", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierImpalesOnEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:ImpaleStacks", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "multiplierBleedsOnEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:BleedStacks", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Combat")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierFragileRegrowth": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Multiplier:FragileRegrowthCount", "BASE", math.Min(val,10), "Config", { type = "Condition", var = "Combat" })
	   	},
	   "conditionKilledUniqueEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:KilledUniqueEnemy", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "conditionHaveArborix": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HaveIronReflexes", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Config"))
	   	},
	   "conditionHaveAugyre": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		if val == "EleOverload" {
	   			modList.AddMod(mod.NewFlag("Condition:HaveElementalOverload", true).Source("Config").Tag(mod.Condition("Combat")))
	   			modList.AddMod(mod.NewList("Keystone", "Elemental Overload").Source("Config"))
	   		} else if val == "ResTechnique" {
	   			modList.AddMod(mod.NewFlag("Condition:HaveResoluteTechnique", true).Source("Config").Tag(mod.Condition("Combat")))
	   			modList.AddMod(mod.NewList("Keystone", "Resolute Technique").Source("Config"))
	   		}
	   	},
	   "conditionHaveVulconus": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:HaveAvatarOfFire", true).Source("Config").Tag(mod.Condition("Combat")))
	   		modList.AddMod(mod.NewList("Keystone", "Avatar of Fire").Source("Config"))
	   	},
	   "conditionHaveManaStorm": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:SacrificeManaForLightning", true).Source("Config").Tag(mod.Condition("Combat")))
	   	},
	   "buffFanaticism": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod("Condition:Fanaticism", "FLAG", true, "Config", { type = "Condition", var = "Combat" }, { type = "Condition", var = "CanGainFanaticism" })
	   	},
	   "critChanceLucky": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("CritChanceLucky", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "skillForkCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("ForkedCount", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "skillChainCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("ChainCount", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "skillPierceCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("PiercedCount", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionAtCloseRange": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFlag("Condition:AtCloseRange", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyMoving": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Moving", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyFullLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:FullLife", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyLowLife": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:LowLife", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyCursed": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Cursed", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyBleeding": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Bleeding", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierRuptureStacks": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:RuptureStack", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod("DamageTaken", "MORE", 25, "Rupture", nil, KeywordFlag.Bleed, { type = "Multiplier", var = "RuptureStack", limit = 3 }, { type = "ActorCondition", actor = "enemy", var = "CanInflictRupture" })
	   		enemyModList.AddMod("BleedExpireRate", "MORE", 25, "Rupture", nil, KeywordFlag.Bleed, { type = "Multiplier", var = "RuptureStack", limit = 3 }, { type = "ActorCondition", actor = "enemy", var = "CanInflictRupture" })
	   	},
	   "conditionEnemyPoisoned": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierPoisonOnEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:PoisonStack", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierWitheredStackCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:WitheredStack", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierCorrosionStackCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("Multiplier:CorrosionStack", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod("Armour", "BASE", -5000, "Corrosion", { type = "Multiplier", var = "CorrosionStack" }, { type = "ActorCondition", actor = "enemy", var = "CanCorrode" })
	   		enemyModList.AddMod("Evasion", "BASE", -1000, "Corrosion", { type = "Multiplier", var = "CorrosionStack" }, { type = "ActorCondition", actor = "enemy", var = "CanCorrode" })
	   	},
	   "multiplierEnsnaredStackCount": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:EnsnareStackCount", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod("Condition:Moving", "FLAG", true, "Config", { type = "MultiplierThreshold", actor = "enemy", var = "EnsnareStackCount", threshold = 1 })
	   	},
	   "conditionEnemyMaimed": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Maimed", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyHindered": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Hindered", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyBlinded": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Blinded", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "overrideBuffBlinded": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod("BlindEffect", "OVERRIDE", val, "Config", {type = "GlobalEffect", effectType = "Buff" })
	   	},
	   "conditionEnemyTaunted": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Taunted", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyBurning": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Burning", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyIgnited": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Ignited", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyScorched": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:ScorchedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionScorchedEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("ScorchVal", mod.TypeBase, val).Source("Config").Tag(mod.Condition("ScorchedConfig")))
	   		enemyModList.AddMod("DesiredScorchVal", "BASE", val, "Brittle", { type = "Condition", var = "ScorchedConfig", neg = true })
	   	},
	   "conditionEnemyOnScorchedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnScorchedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyChilled": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyChilledEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("ChillVal", mod.TypeOverride, val).Source("Chill").Tag(mod.Condition("Chilled")))
	   	},
	   "conditionEnemyChilledByYourHits": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:ChilledByYourHits", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyFrozen": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Frozen", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyBrittle": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Brittle", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:BrittleConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionBrittleEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("BrittleVal", mod.TypeBase, val).Source("Config").Tag(mod.Condition("BrittleConfig")))
	   		enemyModList.AddMod("DesiredBrittleVal", "BASE", val, "Brittle", { type = "Condition", var = "BrittleConfig", neg = true })
	   	},
	   "conditionEnemyOnBrittleGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Brittle", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnBrittleGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyShocked": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:ShockedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionShockEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod("ShockVal", "BASE", val, "Shock", { type = "Condition", var = "ShockedConfig" })
	   		enemyModList.AddMod("DesiredShockVal", "BASE", val, "Shock", { type = "Condition", var = "ShockedConfig", neg = true })
	   	},
	   "conditionEnemyOnShockedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnShockedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemySapped": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Sapped", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:SappedConfig", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionSapEffect": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   	enemyModList.AddMod(mod.NewFloat("SapVal"))
	   		enemyModList.AddMod("SapVal", "BASE", val, "Sap", { type = "Condition", var = "SappedConfig" })
	   		enemyModList.AddMod("DesiredSapVal", "BASE", val, "Sap", { type = "Condition", var = "SappedConfig", neg = true })
	   	},
	   "conditionEnemyOnSappedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Sapped", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnSappedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "multiplierFreezeShockIgniteOnEnemy": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:FreezeShockIgniteOnEnemy", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyFireExposure": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod("FireExposure", "BASE", -10, "Config", { type = "Condition", var = "Effective" }, { type = "ActorCondition", actor = "enemy", var = "CanApplyFireExposure" })
	   	},
	   "conditionEnemyColdExposure": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod("ColdExposure", "BASE", -10, "Config", { type = "Condition", var = "Effective" }, { type = "ActorCondition", actor = "enemy", var = "CanApplyColdExposure" })
	   	},
	   "conditionEnemyLightningExposure": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod("LightningExposure", "BASE", -10, "Config", { type = "Condition", var = "Effective" }, { type = "ActorCondition", actor = "enemy", var = "CanApplyLightningExposure" })
	   	},
	   "conditionEnemyIntimidated": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Intimidated", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyCrushed": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Crushed", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionNearLinkedTarget": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:NearLinkedTarget", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyUnnerved": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:Unnerved", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyCoveredInAsh": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("CoveredInAshEffect", mod.TypeBase, 20).Source("Covered in Ash"))
	   	},
	   "conditionEnemyCoveredInFrost": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("CoveredInFrostEffect", mod.TypeBase, 20).Source("Covered in Frost"))
	   	},
	   "conditionEnemyOnConsecratedGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnConsecratedGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyOnProfaneGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnProfaneGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   		enemyModList.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, -10).Source("Config").Tag(mod.Condition("OnProfaneGround")))
	   		enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, -10).Source("Config").Tag(mod.Condition("OnProfaneGround")))
	   		modList.AddMod("CritChance", "BASE", 1, "Config", { type = "ActorCondition", actor = "enemy", var = "OnProfaneGround" })
	   		modList.AddMod("MinionModifier", "LIST", { mod = modLib.createMod("CritChance", "BASE", 1, "Config", { type = "ActorCondition", actor = "enemy", var = "OnProfaneGround" }) })
	   	},
	   "multiplierEnemyAffectedByGraspingVines": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		modList.AddMod(mod.NewFloat("Multiplier:GraspingVinesAffectingEnemy", mod.TypeBase, val).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyOnFungalGround": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:OnFungalGround", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyInChillingArea": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:InChillingArea", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "conditionEnemyInFrostGlobe": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:EnemyInFrostGlobe", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   "enemyConditionHitByFireDamage": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemymodList.AddMod(mod.NewFlag("Condition:HitByFireDamage", true).Source("Config"))
	   	},
	   "enemyConditionHitByColdDamage": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemymodList.AddMod(mod.NewFlag("Condition:HitByColdDamage", true).Source("Config"))
	   	},
	   "enemyConditionHitByLightningDamage": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemymodList.AddMod(mod.NewFlag("Condition:HitByLightningDamage", true).Source("Config"))
	   	},
	   "conditionEnemyRareOrUnique": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
	   	},
	   	"enemyIsBoss": func(val interface{}, modList *ModList, enemyModList *ModList, build) {
	   		//these defaults are here so that the placeholder gets reset correctly
	   		build.configTab.varControls['enemySpeed']:SetPlaceholder(700, true)
	   		build.configTab.varControls['enemyCritChance']:SetPlaceholder(5, true)
	   		build.configTab.varControls['enemyCritDamage']:SetPlaceholder(30, true)
	   		if val == "None" {
	   			defaultResist := ""
	   			build.configTab.varControls['enemyLightningResist']:SetPlaceholder(defaultResist, true)
	   			build.configTab.varControls['enemyColdResist']:SetPlaceholder(defaultResist, true)
	   			build.configTab.varControls['enemyFireResist']:SetPlaceholder(defaultResist, true)
	   			build.configTab.varControls['enemyChaosResist']:SetPlaceholder(defaultResist, true)

	   			defaultLevel := 66
	   			if build.calcsTab.mainEnv {
	   				defaultLevel = build.calcsTab.mainEnv.enemyLevel
	   			}

	   			defaultDamage := round(data.monsterDamageTable[defaultLevel] * 1.5)
	   			build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyLightningDamage']:SetPlaceholder("", true)
	   			build.configTab.varControls['enemyColdDamage']:SetPlaceholder("", true)
	   			build.configTab.varControls['enemyFireDamage']:SetPlaceholder("", true)
	   			build.configTab.varControls['enemyChaosDamage']:SetPlaceholder("", true)

	   			defaultPen := ""
	   			build.configTab.varControls['enemyLightningPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(defaultPen, true)
	   		} else if val == "Boss" {
	   			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
	   			enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -33).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 488).Source("Boss"))
	   			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))

	   			defaultEleResist := 40
	   			build.configTab.varControls['enemyLightningResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyColdResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyFireResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyChaosResist']:SetPlaceholder(25, true)

	   			defaultLevel := 83
	   			build.configTab.varControls['enemyLevel']:SetPlaceholder(defaultLevel, true)
	   			if build.calcsTab.mainEnv {
	   				defaultLevel = build.calcsTab.mainEnv.enemyLevel
	   			}

	   			defaultDamage := round(data.monsterDamageTable[defaultLevel] * 1.5  * data.misc.stdBossDPSMult)
	   			build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyLightningDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyColdDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyFireDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyChaosDamage']:SetPlaceholder(defaultDamage / 4, true)

	   			defaultPen := ""
	   			build.configTab.varControls['enemyLightningPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(defaultPen, true)
	   		} else if val == "Pinnacle" {
	   			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
	   			enemyModList.AddMod(mod.NewFlag("Condition:PinnacleBoss", true).Source("Config").Tag(mod.Condition("Effective")))
	   			enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -66).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("Armour", mod.TypeMore, 33).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 404).Source("Boss"))
	   			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))

	   			defaultEleResist := 50
	   			build.configTab.varControls['enemyLightningResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyColdResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyFireResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyChaosResist']:SetPlaceholder(30, true)

	   			defaultLevel := 84
	   			build.configTab.varControls['enemyLevel']:SetPlaceholder(defaultLevel, true)
	   			if build.calcsTab.mainEnv {
	   				defaultLevel = build.calcsTab.mainEnv.enemyLevel
	   			}

	   			defaultDamage := round(data.monsterDamageTable[defaultLevel] * 1.5  * data.misc.pinnacleBossDPSMult)
	   			build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyLightningDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyColdDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyFireDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyChaosDamage']:SetPlaceholder(defaultDamage / 4, true)

	   			build.configTab.varControls['enemyLightningPen']:SetPlaceholder(data.misc.pinnacleBossPen, true)
	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(data.misc.pinnacleBossPen, true)
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(data.misc.pinnacleBossPen, true)
	   		} else if val == "Uber" {
	   			enemyModList.AddMod(mod.NewFlag("Condition:RareOrUnique", true).Source("Config").Tag(mod.Condition("Effective")))
	   			enemyModList.AddMod(mod.NewFlag("Condition:PinnacleBoss", true).Source("Config").Tag(mod.Condition("Effective")))
	   			enemyModList.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -66).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("Armour", mod.TypeMore, 100).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -70).Source("Boss"))
	   			enemyModList.AddMod(mod.NewFloat("AilmentThreshold", mod.TypeMore, 404).Source("Boss"))
	   			modList.AddMod(mod.NewFloat("WarcryPower", mod.TypeBase, 20).Source("Boss"))

	   			defaultEleResist := 50
	   			build.configTab.varControls['enemyLightningResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyColdResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyFireResist']:SetPlaceholder(defaultEleResist, true)
	   			build.configTab.varControls['enemyChaosResist']:SetPlaceholder(30, true)

	   			defaultLevel := 85
	   			build.configTab.varControls['enemyLevel']:SetPlaceholder(defaultLevel, true)
	   			if build.calcsTab.mainEnv {
	   				defaultLevel = build.calcsTab.mainEnv.enemyLevel
	   			}

	   			defaultDamage := round(data.monsterDamageTable[defaultLevel] * 1.5  * data.misc.uberBossDPSMult)
	   			build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyLightningDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyColdDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyFireDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyChaosDamage']:SetPlaceholder(defaultDamage / 4, true)

	   			build.configTab.varControls['enemyLightningPen']:SetPlaceholder(data.misc.uberBossPen, true)
	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(data.misc.uberBossPen, true)
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(data.misc.uberBossPen, true)
	   		}
	   	 },
	   "deliriousPercentage": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		if val == "20Percent" {
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -19.2).Source("20% Delirious"))
	   			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 6).Source("20% Delirious"))
	   		}
	   		if val == "40Percent" {
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -38.4).Source("40% Delirious"))
	   			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 12).Source("40% Delirious"))
	   		}
	   		if val == "60Percent" {
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -57.6).Source("60% Delirious"))
	   			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 18).Source("60% Delirious"))
	   		}
	   		if val == "80Percent" {
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -76.8).Source("80% Delirious"))
	   			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 24).Source("80% Delirious"))
	   		}
	   		if val == "100Percent" {
	   			enemyModList.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -96).Source("100% Delirious"))
	   			enemyModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 30).Source("100% Delirious"))
	   		}
	   	},
	   "enemyPhysicalReduction": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, val).Source("Config"))
	   	},
	   "enemyLightningResist": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("LightningResist", mod.TypeBase, val).Source("Config"))
	   	},
	   "enemyColdResist": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("ColdResist", mod.TypeBase, val).Source("Config"))
	   	},
	   "enemyFireResist": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("FireResist", mod.TypeBase, val).Source("Config"))
	   	},
	   "enemyChaosResist": func(val interface{}, modList *ModList, enemyModList *ModList) {
	   		enemyModList.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, val).Source("Config"))
	   	},
	   	"presetBossSkills": func(val interface{}, modList *ModList, enemyModList *ModList, build) {
	   		//reset to empty
	   		if !(val == "None") {
	   			defaultDamage := ""
	   			build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyLightningDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyColdDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyFireDamage']:SetPlaceholder(defaultDamage, true)
	   			build.configTab.varControls['enemyChaosDamage']:SetPlaceholder(defaultDamage, true)

	   			defaultPen := ""
	   			build.configTab.varControls['enemyLightningPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(defaultPen, true)
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(defaultPen, true)
	   		else
	   			build.configTab.varControls['enemyDamageType'].enabled = true
	   		}

	   		if val == "Uber Atziri Flameblast" {
	   			if build.calcsTab.mainEnv {
	   				build.configTab.varControls['enemyFireDamage']:SetPlaceholder(round(data.monsterDamageTable[build.calcsTab.mainEnv.enemyLevel] * 3.48 * 10.9), true)
	   				build.configTab.varControls['enemyDamageType']:SelByValue("Spell", "val")
	   				build.configTab.varControls['enemyDamageType'].enabled = false
	   				build.configTab.input['enemyDamageType'] = "Spell"
	   			}
	   			build.configTab.varControls['enemyFirePen']:SetPlaceholder(10, true)

	   			build.configTab.varControls['enemySpeed']:SetPlaceholder(25000, true)
	   			build.configTab.varControls['enemyCritChance']:SetPlaceholder(0, true)
	   		} else if val == "Shaper Ball" {
	   			if build.calcsTab.mainEnv {
	   				build.configTab.varControls['enemyColdDamage']:SetPlaceholder(round(data.monsterDamageTable[build.calcsTab.mainEnv.enemyLevel] * 9.17), true)
	   			}

	   			build.configTab.varControls['enemyColdPen']:SetPlaceholder(25, true)
	   			build.configTab.varControls['enemySpeed']:SetPlaceholder(1400, true)
	   			build.configTab.varControls['enemyDamageType'].enabled = false
	   			build.configTab.varControls['enemyDamageType']:SelByValue("SpellProjectile", "val")
	   			build.configTab.input['enemyDamageType'] = "SpellProjectile"
	   		} else if val == "Shaper Slam" {
	   			if build.calcsTab.mainEnv {
	   				build.configTab.varControls['enemyPhysicalDamage']:SetPlaceholder(round(data.monsterDamageTable[build.calcsTab.mainEnv.enemyLevel] * 15.2), true)
	   			}
	   			build.configTab.varControls['enemyDamageType'].enabled = false
	   			build.configTab.varControls['enemyDamageType']:SelByValue("Melee", "val")
	   			build.configTab.input['enemyDamageType'] = "Melee"

	   			build.configTab.varControls['enemySpeed']:SetPlaceholder(3510, true)
	   		} else if val == "Maven Memory Game" {
	   			if build.calcsTab.mainEnv {
	   				defaultEleDamage := round(data.monsterDamageTable[build.calcsTab.mainEnv.enemyLevel] * 24.69)
	   				build.configTab.varControls['enemyLightningDamage']:SetPlaceholder(defaultEleDamage, true)
	   				build.configTab.varControls['enemyColdDamage']:SetPlaceholder(defaultEleDamage, true)
	   				build.configTab.varControls['enemyFireDamage']:SetPlaceholder(defaultEleDamage, true)
	   			}
	   			build.configTab.varControls['enemyDamageType'].enabled = false
	   			build.configTab.varControls['enemyDamageType']:SelByValue("Melee", "val")
	   			build.configTab.input['enemyDamageType'] = "Melee"
	   		}
	   	},
	*/
}
