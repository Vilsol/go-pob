package calculator2

import (
	"maps"
	"math"
	"slices"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

// configVarList is ConfigOptions.lua's varList, in its order: every option with an apply function. The
// quest options are applied by buildConfigModList.
var configVarList = []configVar{
	//pob2:block:src/Modules/ConfigOptions.lua:114-116
	{name: "detonateDeadCorpseLife", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "corpseLife", Value: val.Number}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:117-119
	{name: "multiplierCurrentManaPercentage", varType: "count", defaultState: configNumber(100), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CurrentManaPercentage", mod.TypeBase, math.Max(math.Min(val.Number, 100), 0), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:120-132
	{name: "conditionStationary", varType: "count", apply: func(c *configApply, val ConfigValue) {
		v := val.Number
		if val.IsBoolean {
			// Backwards compatibility with older versions that set this condition as a boolean
			v = 0
			if val.Boolean {
				v = 1
			}
		}
		sanitizedValue := math.Max(0, v)
		c.modList.AddMod(newMod("Multiplier:StationarySeconds", mod.TypeBase, sanitizedValue, "Config"))
		if sanitizedValue > 0 {
			c.modList.AddMod(newFlagMod("Condition:Stationary", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:133-135
	{name: "conditionMoving", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Moving", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:136-138
	{name: "conditionFullLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FullLife", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:139-141
	{name: "conditionLowLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LowLife", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:142-144
	{name: "conditionFullMana", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FullMana", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:145-147
	{name: "conditionLowMana", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LowMana", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:148-150
	{name: "conditionFullEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FullEnergyShield", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:151-153
	{name: "conditionLowEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LowEnergyShield", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:154-156
	{name: "conditionHaveEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HaveEnergyShield", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:157-160
	{name: "multiplierCurrentEnergyShield", varType: "count", defaultPlaceholder: configPlaceholder(100), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UseCurrentEnergyShield", "Config"))
		c.modList.AddMod(newMod("Multiplier:CurrentEnergyShield", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:161-164
	{name: "conditionLowRunicWard", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LowRunicWard", "Config"))
		c.modList.AddMod(newFlagMod("Condition:MissingRunicWard", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:165-167
	{name: "conditionMissingRunicWard", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:MissingRunicWard", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:168-172
	{name: "conditionNoRunicWard", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:NoRunicWard", "Config"))
		c.modList.AddMod(newFlagMod("Condition:LowRunicWard", "Config"))
		c.modList.AddMod(newFlagMod("Condition:MissingRunicWard", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:173-175
	{name: "minionsConditionFullLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:FullLife", "Config")}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:176-178
	{name: "minionsConditionFullEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:FullEnergyShield", "Config")}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:179-181
	{name: "minionsConditionCreatedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:MinionsCreatedRecently", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:185-191
	{name: "lifeRegenMode", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "AVERAGE" {
			c.modList.AddMod(newFlagMod("Condition:LifeRegenBurstAvg", "Config"))
		} else if val.String == "FULL" {
			c.modList.AddMod(newFlagMod("Condition:LifeRegenBurstFull", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:192-198
	{name: "resourceGainMode", varType: "list", defaultState: configString("AVERAGE"), apply: func(c *configApply, val ConfigValue) {
		if val.String == "AVERAGE" {
			c.modList.AddMod(newFlagMod("Condition:AverageResourceGain", "Config"))
		} else if val.String == "MAX" {
			c.modList.AddMod(newFlagMod("Condition:MaxResourceGain", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:201-207
	{name: "armourCalculationMode", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "MAX" {
			c.modList.AddMod(newFlagMod("Condition:ArmourMax", "Config"))
		} else if val.String == "AVERAGE" {
			c.modList.AddMod(newFlagMod("Condition:ArmourAvg", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:208-212
	{name: "warcryMode", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "MAX" {
			c.modList.AddMod(newFlagMod("Condition:WarcryMaxHit", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:213-215
	{name: "EVBypass", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EVBypass", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:222-224
	{name: "arcaneCloakUsedRecentlyCheck", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ArcaneCloakUsedRecently", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:226-228
	{name: "arcLightningInfused", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ArcLightningInfused", "Config", mod.Condition("CannotConsumeInfusion").Neg(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:230-232
	{name: "eldritchEmpowermentSacrifice", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EldritchEmpowermentSacrifice", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:234-236
	{name: "unwillingOfferingPower", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:UnwillingOfferingPower", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:238-240
	{name: "aspectOfTheAvianAviansMight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AviansMightActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:241-243
	{name: "aspectOfTheAvianAviansFlight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AviansFlightActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:245-247
	{name: "aspectOfTheCatCatsStealth", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CatsStealthActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:248-250
	{name: "aspectOfTheCatCatsAgility", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CatsAgilityActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:252-254
	{name: "overrideCrabBarriers", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("CrabBarriers", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:256-258
	{name: "aspectOfTheSpiderWebStacks", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("ExtraSkillMod", mod.ExtraSkillMod{Mod: mod.NewFloat("Multiplier:SpiderWebApplyStack", mod.TypeBase, val.Number)}).Source("Config").Tag(mod.SkillName("Aspect of the Spider")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:260-262
	{name: "bannerPlanted", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BannerPlanted", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:263-265
	{name: "bannerValour", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ValourStacks", mod.TypeBase, val.Number, "Config", mod.IgnoreCond(), mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:267-269
	{name: "Unbound", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Unbound", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:271-273
	{name: "bladestormInBloodstorm", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BladestormInBloodstorm", "Config", mod.SkillName("Bladestorm").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:274-276
	{name: "bladestormInSandstorm", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BladestormInSandstorm", "Config", mod.SkillName("Bladestorm").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:278-280
	{name: "bloodSacramentReservationEHP", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BloodSacramentReservationEHP", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:282-284
	{name: "ActiveBrands", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ConfigActiveBrands", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:285-287
	{name: "BrandsAttachedToEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ConfigBrandsAttachedToEnemy", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:288-290
	{name: "targetBrandedEnemy", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TargetingBrandedEnemy", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:291-294
	{name: "BrandsInLastQuarter", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BrandLastQuarter", "Config"))
		c.modList.AddMod(newFlagMod("Condition:BrandLastHalf", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:295-297
	{name: "BrandsInLastHalf", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BrandLastHalf", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:299-301
	{name: "carrionGolemNearbyMinion", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyNonGolemMinion", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:302-304
	{name: "carrionGolemEqualsChaosGolem", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CarrionEqualChaosGolem", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:305-307
	{name: "chaosGolemEqualsStoneGolem", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChaosEqualStoneGolem", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:308-310
	{name: "stoneGolemEqualsCarrionGolem", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StoneEqualCarrionGolem", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:312-314
	{name: "clashCondition", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EnemyHigherLifePercent", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:316-318
	{name: "ColdSnapBypassCD", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Cold Snap").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:320-322
	{name: "ConcPathBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Consecrated Path of Endurance")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:324-326
	{name: "conditionAilmentConsumed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AilmentConsumed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:328-330
	{name: "conditionIgniteConsumed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:IgniteConsumed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:331-333
	{name: "conditionFreezeConsumed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FreezeConsumed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:334-336
	{name: "conditionShockConsumed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ShockConsumed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:338-342
	{name: "conditionCorruptingCryStages", varType: "count", defaultPlaceholder: configPlaceholder(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CorruptingCryStageAfterFirst", mod.TypeBase, val.Number-1, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:344-346
	{name: "overrideCruelty", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Cruelty", mod.TypeOverride, math.Min(val.Number, 40), "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:348-350
	{name: "channellingCycloneCheck", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChannellingCyclone", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:352-354
	{name: "darkPactSkeletonLife", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "skeletonLife", Value: val.Number}).Source("Config").Tag(mod.SkillName("Dark Pact")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:356-358
	{name: "inDemonForm", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:DemonForm", "Config", mod.StatThreshold("Life", 2)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:359-361
	{name: "demonFormStacks", varType: "count", defaultPlaceholder: configPlaceholder(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DemonFlameStacks", mod.TypeBase, val.Number, "Config", mod.Condition("DemonForm")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:364-366
	{name: "curseOverlaps", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CurseOverlaps", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:368-370
	{name: "embraceMadnessActive", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AffectedByGloriousMadness", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:372-374
	{name: "eyeOfWinterCold", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EyeOfWinterCold", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:375-377
	{name: "eyeOfWinterFire", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EyeOfWinterFire", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:378-380
	{name: "eyeOfWinterLightning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EyeOfWinterLightning", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:381-389
	{name: "touchedDebuffsCount", varType: "countAllowZero", defaultState: configNumber(10), apply: func(c *configApply, val ConfigValue) {
		v := math.Min(val.Number, 10)
		stacks := func(name string) mod.Source { return mod.Source(luaNum(v) + " " + name + " Stacks") }
		debuff := func() []mod.Tag {
			return []mod.Tag{mod.GlobalEffect("Debuff"), mod.Condition("AffectedByGloriousMadness")}
		}
		c.modList.AddMod(newMod("DamageTaken", mod.TypeIncrease, v*6, stacks("Eroding Touch"), debuff()...))
		c.modList.AddMod(newMod("ActionSpeed", mod.TypeIncrease, -v*6, stacks("Paralysing Touch"), debuff()...))
		c.modList.AddMod(newMod("FlaskChargesGained", mod.TypeIncrease, -v*9, stacks("Diluting Touch"), debuff()...))
		c.modList.AddMod(newMod("FlaskEffect", mod.TypeIncrease, -v*9, stacks("Diluting Touch"), debuff()...))
		c.modList.AddMod(newMod("LifeRecoveryRate", mod.TypeIncrease, -v*9, stacks("Wasting Touch"), debuff()...))
		c.modList.AddMod(newMod("EnergyShieldRecoveryRate", mod.TypeIncrease, -v*9, stacks("Wasting Touch"), debuff()...))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:391-393
	{name: "flameWallAddedDamage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FlameWallAddedDamage", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:394-396
	{name: "flameWallInfused", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FlameWallInfused", "Config", mod.Condition("FlameWallAddedDamage")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:398-400
	{name: "FlickerStrikeBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Flicker Strike").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:401-419
	{name: "elementalConfluxElement", varType: "list", defaultState: configNumber(1), apply: func(c *configApply, val ConfigValue) {
		coldMultiplier := 0.0
		fireMultiplier := 0.0
		lightningMultiplier := 0.0
		if val.Number == 1 {
			coldMultiplier = 3
			fireMultiplier = 3
			lightningMultiplier = 3
		} else if val.Number == 2 {
			lightningMultiplier = 1
		} else if val.Number == 3 {
			coldMultiplier = 1
		} else if val.Number == 4 {
			fireMultiplier = 1
		}
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxLightningEffect", mod.TypeBase, lightningMultiplier, "Config", mod.GlobalEffect()))
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxColdEffect", mod.TypeBase, coldMultiplier, "Config", mod.GlobalEffect()))
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxFireEffect", mod.TypeBase, fireMultiplier, "Config", mod.GlobalEffect()))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:422-424
	{name: "freshMeatBuffs", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FreshMeatActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:426-428
	{name: "frostShieldStages", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:FrostShieldStage", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:430-432
	{name: "greaterHarbingerOfTimeSlipstream", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:GreaterHarbingerOfTime", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:434-436
	{name: "harbingerOfTimeSlipstream", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HarbingerOfTime", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:438-440
	{name: "multiplierHexDoom", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:HexDoomStack", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:442-444
	{name: "heraldOfAgonyVirulenceStack", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:VirulenceStack", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:446-448
	{name: "hoaOverkill", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "hoaOverkill", Value: val.Number}).Source("Config").Tag(mod.SkillName("Herald of Ash")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:450-452
	{name: "iceNovaCastOnFrostbolt", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastOnFrostbolt", "Config", mod.SkillName("Ice Nova of Frostbolts")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:454-456
	{name: "incisionConsumedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:IncisionConsumedRecently", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:458-464
	{name: "SecondsSinceInevitableCrit", varType: "count", defaultState: configNumber(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SecondsSinceInevitableCrit", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		if val.Number < 4 {
			c.modList.AddMod(newFlagMod("Condition:CritRecently", "Config", mod.Condition("Combat")))
			c.modList.AddMod(newFlagMod("Condition:SkillCritRecently", "Config", mod.Condition("Combat")))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:466-468
	{name: "infusedChannellingInfusion", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:InfusionActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:470-472
	{name: "intensifyIntensity", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:Intensity", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:474-476
	{name: "purpleFlameStacks", varType: "count", defaultState: configNumber(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:PurpleFlamesCount", mod.TypeBase, math.Min(val.Number, 10), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:478-480
	{name: "onLeyline", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnLeyline", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:482-484
	{name: "multiplierLinkedTargets", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:LinkedTargets", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:485-487
	{name: "linkedToMinion", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LinkedToMinion", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:488-490
	{name: "linkedSourceRate", varType: "float", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("IntuitiveLinkSourceRate", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:492-494
	{name: "manabondMissingUnreservedManaPercentage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "ManabondMissingUnreservedManaPercentage", Value: math.Max(math.Min(val.Number, 100), 0)}).Source("Config").Tag(mod.SkillName("Manabond")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:496-498
	{name: "meatShieldEnemyNearYou", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:MeatShieldEnemyNearYou", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:500-502
	{name: "momentumDamage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Moved2m", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:504-508
	{name: "stoicismSeconds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsingStoicism", "Config"))
		c.modList.AddMod(newMod("Multiplier:StoicismSeconds", mod.TypeBase, math.Min(math.Max(val.Number, 0), 20), "Config"))
		c.modList.AddMod(newMod("Multiplier:StoicismCap", mod.TypeBase, 20, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:510-512
	{name: "parryActive", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ParryActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:514-520
	{name: "plagueBearerState", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "INC" {
			c.modList.AddMod(newFlagMod("Condition:PlagueBearerIncubating", "Config"))
		} else if val.String == "INF" {
			c.modList.AddMod(newFlagMod("Condition:PlagueBearerInfecting", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:522-524
	{name: "perforateSpikeOverlap", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:PerforateSpikeOverlap", mod.TypeBase, val.Number, "Config", mod.SkillName("Perforate").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:526-528
	{name: "physicalAegisDepleted", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:PhysicalAegisDepleted", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:530-532
	{name: "deathmarkDeathmarkActive", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EnemyHasDeathmark", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:534-538
	{name: "prideEffect", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "MAX" {
			c.modList.AddMod(newFlagMod("Condition:PrideMaxEffect", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:540-542
	{name: "sacrificedRageCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:RageSacrificedStacks", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:544-546
	{name: "raiseSpectreEnableBuffs", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillType("Buff"), mod.SkillName("Spectre").SummonSkill(true).PartialMatch(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:547-549
	{name: "raiseSpectreEnableCurses", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillType("AppliesCurse"), mod.SkillName("Spectre").SummonSkill(true).PartialMatch(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:550-552
	{name: "conditionSummonedSpectreInPast8Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SummonedSpectreInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:553-556
	{name: "raiseSpectreBladeVortexBladeCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "dpsMultiplier", Value: val.Number}).Source("Config").Tag(mod.SkillId("DemonModularBladeVortexSpectre")))
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "dpsMultiplier", Value: val.Number}).Source("Config").Tag(mod.SkillId("GhostPirateBladeVortexSpectre")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:557-559
	{name: "raiseSpectreKaomFireBeamTotemStage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:KaomFireBeamTotemStage", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:560-562
	{name: "raiseSpectreEnableSummonedUrsaRallyingCry", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("DropBearSummonedRallyingCry")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:563-565
	{name: "raiseSpectreEnableSlashingHorrorEnrage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 0}).Source("Config").Tag(mod.SkillId("AzmeriDualStrikeDemonFireEnrage")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:566-568
	{name: "raiseSpectreEnableSanguimancerDemonLowLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 0}).Source("Config").Tag(mod.SkillId("ABTTAzmeriShepherdSpellDamage")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:570-572
	{name: "raiseSpidersSpiderCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:RaisedSpider", mod.TypeBase, math.Min(val.Number, 20), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:574-576
	{name: "conditionSummonedZombieInPast8Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SummonedZombieInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:577-579
	{name: "animateWeaponLingeringBlade", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AnimatingLingeringBlades", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:581-583
	{name: "risingTempestLightning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentElementalSkillUsedRecently", mod.TypeBase, 1, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:584-586
	{name: "risingTempestCold", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentElementalSkillUsedRecently", mod.TypeBase, 1, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:587-589
	{name: "risingTempestFire", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentElementalSkillUsedRecently", mod.TypeBase, 1, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:591-593
	{name: "scavengedPlatingStacks", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ScavengedPlatingStacks", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:595-597
	{name: "ShrapnelBallistaProjectileOverlap", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "ShrapnelBallistaProjectileOverlap", Value: val.Number}).Source("Config").Tag(mod.SkillName("Shrapnel Ballista").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:599-601
	{name: "sigilOfPowerStages", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SigilOfPowerStage", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:603-606
	{name: "siphoningTrapAffectedEnemies", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnemyAffectedBySiphoningTrap", mod.TypeBase, val.Number, "Config"))
		c.modList.AddMod(newFlagMod("Condition:SiphoningTrapSiphoning", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:608-610
	{name: "conditionOldSpear", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SpearOlderThanHalfSecond", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:612-614
	{name: "configSpectralWolfCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SpectralWolfCount", mod.TypeBase, math.Min(val.Number, 10), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:616-618
	{name: "configMistRavenCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:MistRavenCount", mod.TypeBase, math.Min(val.Number, 30), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:620-624
	{name: "bloodSandStance", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "SAND" {
			c.modList.AddMod(newFlagMod("Condition:SandStance", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:625-627
	{name: "changedStance", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChangedStanceRecently", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:629-631
	{name: "shardsConsumed", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SteelShardConsumed", mod.TypeBase, math.Min(val.Number, 12), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:632-634
	{name: "steelWards", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SteelWardCount", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:636-638
	{name: "stormRainBeamOverlap", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "beamOverlapMultiplier", Value: val.Number}).Source("Config").Tag(mod.SkillName("Storm Rain")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:640-642
	{name: "stormRainActiveArrows", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "activeArrowMultiplier", Value: val.Number}).Source("Config").Tag(mod.SkillName("Storm Rain of the Conduit")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:644-646
	{name: "summonCompanionEnableBuffs", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillType("Buff"), mod.SkillName("Companion").SummonSkill(true).PartialMatch(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:647-649
	{name: "summonCompanionEnableCurses", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillType("AppliesCurse"), mod.SkillName("Companion").SummonSkill(true).PartialMatch(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:651-653
	{name: "summonElementalRelicEnableAngerAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("Anger"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:654-656
	{name: "summonElementalRelicEnableHatredAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("Hatred"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:657-659
	{name: "summonElementalRelicEnableWrathAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("Wrath"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:661-663
	{name: "summonHolyRelicEnableHolyRelicBoon", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HolyRelicBoonActive", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:665-667
	{name: "summonLightningGolemEnableWrath", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("LightningGolemWrath")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:669-671
	{name: "summonReaperConsumeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(mod.SkillId("ReaperConsumeMinionForBuff")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:673-675
	{name: "nearbyBleedingEnemies", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyBleedingEnemies", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:677-679
	{name: "tornadoShotSecondaryHitChance", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "tornadoShotSecondaryHitChance", Value: val.Number}).Source("Config").Tag(mod.SkillName("Tornado Shot")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:681-683
	{name: "toxicRainPodOverlap", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "podOverlapMultiplier", Value: val.Number}).Source("Config").Tag(mod.SkillName("Toxic Rain").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:685-687
	{name: "traumaStacks", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:TraumaStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:689-691
	{name: "configResonanceCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ResonanceCount", mod.TypeBase, math.Max(math.Min(val.Number, 300), 0), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:693-695
	{name: "twisterCold", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TwisterCold", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:696-698
	{name: "twisterFire", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TwisterFire", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:699-701
	{name: "twisterLightning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TwisterLightning", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:703-705
	{name: "conditionInsane", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Insane", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:706-713
	{name: "conditionUnravelling", varType: "list", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:"+val.String+"Unravel", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:715-717
	{name: "VigilantStrikeBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Vigilant Strike")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:719-721
	{name: "voltaxicBurstSpellsQueued", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WhirlwindStages", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:723-725
	{name: "vortexCastOnFrostbolt", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastOnFrostbolt", "Config", mod.SkillName("Vortex of Projection")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:727-729
	{name: "multiplierWarcryPower", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("WarcryPower", mod.TypeOverride, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:731-734
	{name: "whirlwindStages", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WhirlwindStageAfterFirst", mod.TypeBase, math.Min(val.Number-1, 3), "Config", mod.Condition("Combat")))
		c.modList.AddMod(newMod("Multiplier:WhirlwindStages", mod.TypeBase, math.Min(val.Number, 4), "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:736-739
	{name: "whirlwindBuffCold", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:WhirlwindBuffCold", "Config", mod.Condition("Combat")))
		c.enemyModList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:740-743
	{name: "whirlwindBuffFire", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:WhirlwindBuffFire", "Config", mod.Condition("Combat")))
		c.enemyModList.AddMod(newFlagMod("Condition:Ignited", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:744-747
	{name: "whirlwindBuffLightning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:WhirlwindBuffLightning", "Config", mod.Condition("Combat")))
		c.enemyModList.AddMod(newFlagMod("Condition:Shocked", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:749-751
	{name: "windDancerStacks", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WindDancerStacks", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:752-754
	{name: "conditionEmpowered", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Empowered", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:755-757
	{name: "multiplierWoCExpiredDuration", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WoCDurationExpired", mod.TypeBase, math.Min(val.Number, 100), "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:758-760
	{name: "absolutionSkillDamageCountedOnce", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AbsolutionSkillDamageCountedOnce", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:762-764
	{name: "MoltenShellDamageMitigated", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "MoltenShellDamageMitigated", Value: val.Number}).Source("Config").Tag(mod.SkillName("Molten Shell")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:766-768
	{name: "VaalMoltenShellDamageMitigated", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "VaalMoltenShellDamageMitigated", Value: val.Number}).Source("Config").Tag(mod.SkillName("Molten Shell")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:770-794
	{name: "enemySizePreset", varType: "list", defaultState: configString("Medium"), apply: func(c *configApply, val ConfigValue) {
		radius := map[string]float64{"Small": 2, "Medium": 3, "Large": 5, "Huge": 11}
		if r, ok := radius[val.String]; ok {
			// SetPlaceholder(r, false) changes only the control's text, not the config set's placeholder table.
			c.modList.AddMod(newMod("EnemyRadius", mod.TypeBase, r, "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:795-797
	{name: "enemyRadius", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("EnemyRadius", mod.TypeOverride, math.Max(val.Number, 1), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:798-800
	{name: "TotalSpectreLife", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotalSpectreLife", mod.TypeOverride, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:801-803
	{name: "TotalTotemLife", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotalTotemLife", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:804-806
	{name: "TotalRadianceSentinelLife", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotalRadianceSentinelLife", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:807-809
	{name: "TotalVaalRejuvenationTotemLife", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotalVaalRejuvenationTotemLife", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:810-812
	{name: "TotalCompanionLife", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotalCompanionLife", mod.TypeOverride, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:815-817
	{name: "usePowerCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsePowerCharges", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:818-820
	{name: "overridePowerCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("PowerCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:821-823
	{name: "useFrenzyCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UseFrenzyCharges", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:824-826
	{name: "overrideFrenzyCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("FrenzyCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:827-829
	{name: "useEnduranceCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UseEnduranceCharges", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:830-832
	{name: "overrideEnduranceCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("EnduranceCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:833-835
	{name: "useSiphoningCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("UseSiphoningCharges", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:836-838
	{name: "overrideSiphoningCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("SiphoningCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:839-841
	{name: "useChallengerCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("UseChallengerCharges", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:842-844
	{name: "overrideChallengerCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ChallengerCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:845-847
	{name: "useBlitzCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("UseBlitzCharges", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:848-850
	{name: "overrideBlitzCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("BlitzCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:851-853
	{name: "overrideInspirationCharges", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("InspirationCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:854-856
	{name: "useGhostShrouds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("UseGhostShrouds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:857-859
	{name: "overrideGhostShrouds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("GhostShrouds", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:860-862
	{name: "configBossFaceBroken", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BossFaceBroken", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:863-865
	{name: "waitForMaxSeals", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("UseMaxUnleash", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:866-879
	{name: "repeatMode", varType: "list", defaultState: configString("AVERAGE"), apply: func(c *configApply, val ConfigValue) {
		if val.String == "AVERAGE" {
			c.modList.AddMod(newFlagMod("Condition:averageRepeat", "Config"))
			c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:averageRepeat", "Config")}))
		} else if val.String == "FINAL" || val.String == "FINAL_DPS" {
			c.modList.AddMod(newFlagMod("Condition:alwaysFinalRepeat", "Config"))
			c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:alwaysFinalRepeat", "Config")}))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:882-884
	{name: "conditionSprinting", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Sprinting", "Config", mod.Condition("Combat"), mod.Condition("CanSprint")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:885-887
	{name: "overrideBloodCharges", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("BloodCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:888-890
	{name: "overrideSpiritCharges", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("SpiritCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:891-893
	{name: "minionsUsePowerCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("UsePowerCharges", "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:894-896
	{name: "minionsUseFrenzyCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("UseFrenzyCharges", "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:897-899
	{name: "minionsUseEnduranceCharges", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("UseEnduranceCharges", "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:900-902
	{name: "minionsOverridePowerCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newMod("PowerCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:903-905
	{name: "minionsOverrideFrenzyCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newMod("FrenzyCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:906-908
	{name: "minionsOverrideEnduranceCharges", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newMod("EnduranceCharges", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:909-911
	{name: "multiplierRampage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:Rampage", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:912-914
	{name: "multiplierSoulEater", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SoulEaterStack", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:915-917
	{name: "conditionFocused", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Focused", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:918-921
	{name: "buffLifetap", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Lifetap", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newMod("FlaskLifeRecovery", mod.TypeIncrease, 20, "Lifetap"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:922-924
	{name: "buffOnslaught", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Onslaught", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:925-927
	{name: "buffArcaneSurge", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ArcaneSurge", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:928-930
	{name: "buffQuicksandHourglass", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:QuicksandHourglass", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:931-933
	{name: "minionBuffOnslaught", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:Onslaught", "Config", mod.Condition("Combat"))}))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:934-936
	{name: "buffUnholyMight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UnholyMight", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:937-939
	{name: "minionbuffUnholyMight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:UnholyMight", "Config", mod.Condition("Combat"))}))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:940-942
	{name: "buffChaoticMight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChaoticMight", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:943-945
	{name: "minionbuffChaoticMight", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:ChaoticMight", "Config", mod.Condition("Combat"))}))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:946-948
	{name: "buffPhasing", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Phasing", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:949-951
	{name: "overrideFortification", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("FortificationStacks", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:952-954
	{name: "multiplierTailwind", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:Tailwind", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:955-957
	{name: "buffAdrenaline", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Adrenaline", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:958-960
	{name: "conditionChangedStanceLastSecond", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StanceChangeLastSecond", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:961-963
	{name: "buffAlchemistsGenius", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AlchemistsGenius", "Config", mod.Condition("Combat"), mod.Condition("CanHaveAlchemistGenius")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:964-966
	{name: "buffVaalArcLuckyHits", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("LuckyHits", "Config", mod.Condition("Combat", "CanBeLucky"), mod.SkillName("Arc").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:967-970
	{name: "buffElusive", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Elusive", "Config", mod.Condition("Combat"), mod.Condition("CanBeElusive")))
		c.modList.AddMod(newFlagMod("Elusive", "Config", mod.Condition("Combat"), mod.Condition("CanBeElusive")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:971-973
	{name: "overrideBuffElusive", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ElusiveEffect", mod.TypeOverride, val.Number, "Config", mod.GlobalEffect("Buff")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:974-976
	{name: "buffDivinity", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Divinity", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:977-979
	{name: "multiplierDefiance", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:Defiance", mod.TypeBase, math.Min(val.Number, 10), "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:980-982
	{name: "multiplierRage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:RageStack", mod.TypeBase, val.Number, "Config", mod.IgnoreCond(), mod.Condition("Combat"), mod.Condition("CanGainRage")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:983-984
	{name: "multiplierMinionRage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newMod("Multiplier:RageStack", mod.TypeBase, val.Number, "Config", mod.IgnoreCond(), mod.Condition("Combat"), mod.Condition("CanGainRage"))}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:985-987
	{name: "multiplierCombo", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ComboStacks", mod.TypeBase, val.Number, "Config", mod.IgnoreCond(), mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:988-990
	{name: "conditionLeeching", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Leeching", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:991-994
	{name: "conditionLeechingLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LeechingLife", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Leeching", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:995-998
	{name: "conditionLeechingEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LeechingEnergyShield", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Leeching", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:999-1002
	{name: "conditionLeechingMana", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LeechingMana", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Leeching", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1003-1006
	{name: "minionsConditionLeechingEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:LeechingEnergyShield", "Config")}).Source("Config"))
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:Leeching", "Config")}).Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1007-1009
	{name: "conditionUsingFlask", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsingFlask", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1010-1012
	{name: "conditionNoLifeFlaskUsesLeft", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:NoLifeFlaskUsesLeft", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1013-1015
	{name: "conditionUsingCharm", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsingCharm", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1016-1018
	{name: "conditionHaveTotem", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HaveTotem", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1019-1021
	{name: "conditionSummonedTotemRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SummonedTotemRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1022-1025
	{name: "TotemsSummoned", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TotemsSummoned", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewFlag("Condition:HaveTotem", val.Number >= 1).Source("Config").Tag(mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1026-1028
	{name: "conditionSummonedGolemInPast8Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SummonedGolemInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1029-1031
	{name: "conditionSummonedGolemInPast10Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SummonedGolemInPast10Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1032-1034
	{name: "companionInPresence", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CompanionInPresence", "Config", mod.Condition("HaveCompanion")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1035-1037
	{name: "multiplierNearbyAlly", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyAlly", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1038-1040
	{name: "multiplierNearbyCorpse", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyCorpse", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1041-1043
	{name: "multiplierSummonedMinion", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SummonedMinion", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1044-1046
	{name: "multiplierMinionsInPresence", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:MinionPresenceCount", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1047-1050
	{name: "conditionOnConsecratedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnConsecratedGround", "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:OnConsecratedGround", "Config", mod.Condition("Combat"))}))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1051-1053
	{name: "conditionOnCausticGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnCausticGround", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1054-1056
	{name: "conditionOnFungalGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnFungalGround", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1057-1060
	{name: "conditionOnBurningGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnBurningGround", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Burning", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1061-1064
	{name: "conditionOnIgnitedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnIgnitedGround", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Ignited", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1065-1068
	{name: "conditionOnChilledGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnChilledGround", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1069-1072
	{name: "conditionOnShockedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OnShockedGround", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Shocked", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1073-1075
	{name: "conditionBlinded", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Blinded", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1076-1078
	{name: "conditionBurning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Burning", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1079-1081
	{name: "conditionIgnited", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Ignited", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1082-1084
	{name: "conditionScorched", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Scorched", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1085-1087
	{name: "conditionChilled", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1088-1090
	{name: "conditionChilledEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ChillVal", mod.TypeOverride, val.Number, "Chill", mod.Condition("Chilled")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1091-1093
	{name: "conditionSelfChill", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChilledSelf", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1094-1096
	{name: "conditionFrozen", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Frozen", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1097-1099
	{name: "conditionBrittle", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Brittle", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1100-1103
	{name: "conditionShocked", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Shocked", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newMod("DamageTaken", mod.TypeIncrease, 15, "Shock", mod.Condition("Shocked")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1104-1106
	{name: "conditionSapped", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Sapped", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1107-1109
	{name: "conditionBleeding", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Bleeding", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1110-1112
	{name: "conditionPoisoned", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Poisoned", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1113-1115
	{name: "conditionCanBeCurseImmune", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("AvoidCurse", mod.TypeBase, 100, "Config", mod.Condition("Combat"), mod.GlobalEffect("Global").Unscalable(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1116-1118
	{name: "multiplierPoisonOnSelf", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:PoisonStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1119-1121
	{name: "multiplierWitheredStackCountSelf", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(15), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WitheredStack", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1122-1125
	{name: "multiplierNearbyEnemies", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyEnemies", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewFlag("Condition:OnlyOneNearbyEnemy", val.Number == 1).Source("Config").Tag(mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1126-1131
	{name: "multiplierNearbyRareOrUniqueEnemies", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NearbyRareOrUniqueEnemies", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newMod("Multiplier:NearbyEnemies", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewFlag("Condition:AtMostOneNearbyRareOrUniqueEnemy", val.Number <= 1).Source("Config").Tag(mod.Condition("Combat")))
		c.enemyModList.AddMod(mod.NewFlag("Condition:NearbyRareOrUniqueEnemy", val.Number >= 1).Source("Config").Tag(mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1132-1134
	{name: "conditionSurrounded", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Surrounded", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1135-1137
	{name: "conditionHitRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1138-1141
	{name: "conditionHitSpellRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitSpellRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:HitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1142-1145
	{name: "conditionHitMeleeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitMeleeRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:HitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1146-1149
	{name: "conditionHitProjectileRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitProjectileRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:HitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1150-1154
	{name: "conditionCritRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CritRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:SkillCritRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:CritInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1155-1157
	{name: "conditionCritInPast8Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CritInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1158-1161
	{name: "conditionSkillCritRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SkillCritRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:CritInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1162-1166
	{name: "conditionCritWithHeraldSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CritWithHeraldSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:SkillCritRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:CritInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1167-1169
	{name: "conditionNonCritRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:NonCritRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1170-1172
	{name: "LostNonVaalBuffRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LostNonVaalBuffRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1173-1175
	{name: "conditionChannelling", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Channelling", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1176-1179
	{name: "multiplierChannelling", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ChannellingTime", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:Channelling", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1180-1182
	{name: "conditionHitRecentlyWithWeapon", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitRecentlyWithWeapon", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1183-1185
	{name: "conditionKilledRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1186-1189
	{name: "multiplierKilledRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnemyKilledRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:KilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1190-1192
	{name: "conditionKilledLast3Seconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledLast3Seconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1193-1195
	{name: "conditionKilledPoisonedLast2Seconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledPoisonedLast2Seconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1196-1198
	{name: "conditionKilledShockedLast3Seconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledShockedLast3Seconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1199-1201
	{name: "conditionKilledTauntedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledTauntedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1202-1204
	{name: "conditionTotemsNotSummonedInPastTwoSeconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:NoSummonedTotemsInPastTwoSeconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1205-1207
	{name: "conditionTotemsKilledRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TotemsKilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1208-1210
	{name: "conditionTotemsHitRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TotemsHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1211-1214
	{name: "conditionTotemsHitSpellRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TotemsHitSpellRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:TotemsHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1215-1217
	{name: "conditionUsedBrandRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedBrandRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1218-1221
	{name: "multiplierTotemsKilledRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnemyKilledByTotemsRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:TotemsKilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1222-1224
	{name: "conditionMinionsKilledRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:MinionsKilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1225-1227
	{name: "conditionMinionsDiedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:MinionsDiedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1228-1231
	{name: "multiplierMinionsKilledRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnemyKilledByMinionsRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:MinionsKilledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1232-1234
	{name: "conditionKilledAffectedByDoT", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledAffectedByDotRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1235-1237
	{name: "multiplierShockedEnemyKilledRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ShockedEnemyKilledRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1238-1240
	{name: "multiplierShockedNonShockedEnemyRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ShockedNonShockedEnemyRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1241-1243
	{name: "conditionFrozenEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:FrozenEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1244-1246
	{name: "conditionChilledEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ChilledEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1247-1249
	{name: "conditionShatteredEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ShatteredEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1250-1252
	{name: "conditionIgnitedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:IgnitedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1253-1255
	{name: "multiplierIgniteAppliedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:IgniteAppliedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1256-1258
	{name: "conditionShockedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ShockedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1259-1261
	{name: "conditionStunnedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StunnedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1262-1264
	{name: "conditionPinnedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:PinnedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1265-1267
	{name: "multiplierPoisonAppliedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:PoisonAppliedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1268-1270
	{name: "multiplierLifeSpentRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:LifeSpentRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1271-1273
	{name: "multiplierLifeSpentPast8s", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:LifeSpentPast8s", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1274-1276
	{name: "multiplierManaSpentRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:ManaSpentRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1277-1280
	{name: "conditionStunnedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StunnedRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1281-1283
	{name: "multiplierStunnedRecently", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:StunnedRecently", mod.TypeBase, math.Min(val.Number, 100), "Config", mod.Condition("Combat"), mod.Condition("StunnedRecently")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1284-1286
	{name: "conditionShapeshifted", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Shapeshifted", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1287-1289
	{name: "conditionShapeshiftToAnimal", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ShapeshiftToAnimal", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1290-1292
	{name: "conditionShapeshiftToHuman", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ShapeshiftToHuman", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1293-1295
	{name: "conditionInfusionConsumedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:InfusionConsumedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1296-1298
	{name: "conditionInDodgeRoll", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:InDodgeRoll", "Config", mod.Condition("Combat"), mod.Condition("CannotDodgeRoll").Neg(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1300-1302
	{name: "conditionBeenHitRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1303-1306
	{name: "multiplierBeenHitRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BeenHitRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewFlag("Condition:BeenHitRecently", 1 <= val.Number).Source("Config").Tag(mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1307-1309
	{name: "conditionBeenHitByAttackRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BeenHitByAttackRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1310-1312
	{name: "conditionBeenCritRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BeenCritRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1313-1315
	{name: "conditionConsumed12SteelShardsRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Consumed12SteelShardsRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1316-1318
	{name: "conditionGainedPowerChargeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:GainedPowerChargeRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1319-1321
	{name: "conditionGainedFrenzyChargeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:GainedFrenzyChargeRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1322-1325
	{name: "conditionBeenSavageHitRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BeenSavageHitRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1326-1329
	{name: "conditionHitByFireDamageRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitByFireDamageRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1330-1333
	{name: "conditionHitByColdDamageRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitByColdDamageRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1334-1337
	{name: "conditionHitByLightningDamageRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitByLightningDamageRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1338-1341
	{name: "conditionHitBySpellDamageRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HitBySpellDamageRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1342-1345
	{name: "conditionTakenFireDamageFromEnemyHitRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TakenFireDamageFromEnemyHitRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BeenHitRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1346-1348
	{name: "conditionBlockedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BlockedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1349-1351
	{name: "conditionActiveBlockedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ActiveBlockedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1352-1355
	{name: "conditionBlockedAttackRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BlockedAttackRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BlockedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1356-1359
	{name: "conditionBlockedSpellRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BlockedSpellRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BlockedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1360-1363
	{name: "conditionParriedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ParriedRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:BlockedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1364-1366
	{name: "conditionDodgeRolledRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:DodgeRolledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1367-1369
	{name: "conditionEnergyShieldRechargeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EnergyShieldRechargeRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1370-1372
	{name: "conditionEnergyShieldRechargePastTwoSec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EnergyShieldRechargePastTwoSec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1373-1375
	{name: "conditionStoppedTakingDamageOverTimeRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StoppedTakingDamageOverTimeRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1376-1378
	{name: "conditionConvergence", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Convergence", "Config", mod.Condition("Combat"), mod.Condition("CanGainConvergence")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1379-1385
	{name: "buffPendulum", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "AREA" {
			c.modList.AddMod(newFlagMod("Condition:PendulumOfDestructionAreaOfEffect", "Config", mod.Condition("Combat")))
		} else if val.String == "DAMAGE" {
			c.modList.AddMod(newFlagMod("Condition:PendulumOfDestructionElementalDamage", "Config", mod.Condition("Combat")))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1386-1396
	{name: "buffConflux", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "CHILLING" || val.String == "ALL" {
			c.modList.AddMod(newFlagMod("Condition:ChillingConflux", "Config", mod.Condition("Combat")))
		}
		if val.String == "SHOCKING" || val.String == "ALL" {
			c.modList.AddMod(newFlagMod("Condition:ShockingConflux", "Config", mod.Condition("Combat")))
		}
		if val.String == "IGNITING" || val.String == "ALL" {
			c.modList.AddMod(newFlagMod("Condition:IgnitingConflux", "Config", mod.Condition("Combat")))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1397-1402
	{name: "highestDamageType", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String != "NONE" {
			c.modList.AddMod(newFlagMod("Condition:"+val.String+"IsHighestDamageType", "Config"))
			c.modList.AddMod(newFlagMod("IsHighestDamageTypeOVERRIDE", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1403-1411
	{name: "buffHeartstopper", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "HIT" {
			c.modList.AddMod(newFlagMod("Condition:HeartstopperHIT", "Config", mod.Condition("Combat")))
		} else if val.String == "DOT" {
			c.modList.AddMod(newFlagMod("Condition:HeartstopperDOT", "Config", mod.Condition("Combat")))
		} else if val.String == "AVERAGE" {
			c.modList.AddMod(newFlagMod("Condition:HeartstopperAVERAGE", "Config", mod.Condition("Combat")))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1412-1414
	{name: "buffBastionOfHope", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BastionOfHopeActive", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1415-1417
	{name: "buffNgamahuFlamesAdvance", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:NgamahuFlamesAdvance", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1418-1420
	{name: "buffHerEmbrace", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("HerEmbrace", "Config", mod.Condition("Combat"), mod.Condition("CanGainHerEmbrace")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1421-1423
	{name: "conditionChampionIntimidate", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:ChampionIntimidate", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1424-1426
	{name: "conditionUsedSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1427-1430
	{name: "multiplierSkillUsedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SkillUsedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1431-1434
	{name: "conditionAttackedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AttackedRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1435-1438
	{name: "conditionCastSpellRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastSpellRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1439-1441
	{name: "multiplierNonInstantSpellCastRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:NonInstantSpellCastRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1442-1444
	{name: "multiplierAppliedAilmentsRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:AppliedAilmentsRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1445-1447
	{name: "conditionLinkedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LinkedRecently", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1448-1450
	{name: "conditionStunnedWhileCastingRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:StunnedWhileCastingRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1451-1453
	{name: "conditionCastLast1Seconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastLast1Seconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1454-1456
	{name: "multiplierCastLast8Seconds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CastLast8Seconds", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1457-1459
	{name: "conditionSuppressedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SuppressedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1460-1463
	{name: "multiplierHitsSuppressedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:HitsSuppressedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:SuppressedRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1464-1467
	{name: "conditionUsedFireSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedFireSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1468-1471
	{name: "conditionUsedColdSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedColdSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1472-1475
	{name: "conditionUsedMinionSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedMinionSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1476-1480
	{name: "conditionUsedTravelSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedTravelSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedMovementSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1481-1486
	{name: "conditionUsedDashRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastDashRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedTravelSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedMovementSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1487-1490
	{name: "conditionUsedMovementSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedMovementSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1491-1494
	{name: "conditionUsedVaalSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedVaalSkillRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1495-1497
	{name: "conditionSoulGainPrevention", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SoulGainPrevention", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1498-1502
	{name: "conditionUsedWarcryRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedWarcryRecently", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedWarcryInPast8Seconds", "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:UsedSkillRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1503-1505
	{name: "multiplierWarcryUsedRecently", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WarcryUsedRecently", mod.TypeBase, math.Min(val.Number, 100), "Config", mod.Condition("Combat"), mod.Condition("UsedWarcryRecently")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1506-1508
	{name: "conditionUsedWarcryInPast8Seconds", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UsedWarcryInPast8Seconds", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1509-1511
	{name: "multiplierAffectedByWarcryBuffDuration", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:AffectedByWarcryBuffDuration", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1512-1514
	{name: "DetonatedMinesRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:DetonatedMinesRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1515-1517
	{name: "multiplierMineDetonatedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:MineDetonatedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1518-1520
	{name: "minesPerThrow", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("MineThrowCount", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1521-1523
	{name: "TriggeredTrapsRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TriggeredTrapsRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1524-1526
	{name: "multiplierTrapTriggeredRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:TrapTriggeredRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1527-1529
	{name: "conditionThrownTrapOrMineRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TrapOrMineThrownRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1530-1532
	{name: "trapsPerThrow", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("TrapThrowCount", mod.TypeOverride, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1533-1535
	{name: "conditionTriggeredSkillRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TriggeredSkillRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1536-1538
	{name: "conditionCursedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CursedEnemyRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1539-1541
	{name: "conditionCastMarkRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CastMarkRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1542-1544
	{name: "conditionSpawnedCorpseRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SpawnedCorpseRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1545-1547
	{name: "conditionConsumedCorpseRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ConsumedCorpseRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1548-1550
	{name: "conditionConsumedCorpseInPast2Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ConsumedCorpseInPast2Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1551-1554
	{name: "multiplierCorpseConsumedRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CorpseConsumedRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:ConsumedCorpseRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1555-1557
	{name: "conditionRavenousCorpseConsumed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:RavenousCorpseConsumed", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1558-1560
	{name: "conditionTauntedEnemyRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TauntedEnemyRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1561-1563
	{name: "conditionReloadedRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ReloadedRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1564-1566
	{name: "conditionLostEnduranceChargeInPast8Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:LostEnduranceChargeInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1567-1570
	{name: "multiplierEnduranceChargesLostRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnduranceChargesLostRecently", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.modList.AddMod(newFlagMod("Condition:LostEnduranceChargeInPast8Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1571-1573
	{name: "conditionBlockedHitFromUniqueEnemyInPast10Sec", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:BlockedHitFromUniqueEnemyInPast10Sec", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1574-1576
	{name: "conditionKilledUniqueEnemy", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:KilledUniqueEnemy", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1577-1579
	{name: "multiplierBlockedPast10Sec", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BlockedPast10Sec", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1580-1582
	{name: "conditionImpaledRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:ImpaledRecently", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1583-1585
	{name: "multiplierImpalesOnEnemy", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ImpaleStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1586-1588
	{name: "multiplierFragileRegrowth", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:FragileRegrowthCount", mod.TypeBase, math.Min(val.Number, 10), "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1589-1592
	{name: "conditionHaveArborix", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HaveIronReflexes", "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1593-1601
	{name: "conditionHaveAugyre", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "EleOverload" {
			c.modList.AddMod(newFlagMod("Condition:HaveElementalOverload", "Config", mod.Condition("Combat")))
			c.modList.AddMod(mod.NewList("Keystone", "Elemental Overload").Source("Config"))
		} else if val.String == "ResTechnique" {
			c.modList.AddMod(newFlagMod("Condition:HaveResoluteTechnique", "Config", mod.Condition("Combat")))
			c.modList.AddMod(mod.NewList("Keystone", "Resolute Technique").Source("Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1602-1605
	{name: "conditionHaveVulconus", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:HaveAvatarOfFire", "Config", mod.Condition("Combat")))
		c.modList.AddMod(mod.NewList("Keystone", "Avatar of Fire").Source("Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1606-1608
	{name: "conditionHaveManaStorm", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:SacrificeManaForLightning", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1609-1611
	{name: "GamblesprintMovementSpeed", varType: "list", defaultState: configNumber(20), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("MovementSpeed", mod.TypeIncrease, val.Number, "Config", mod.Condition("Combat"), mod.Condition("HaveGamblesprint")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1612-1615
	{name: "EverlastingSacrifice", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("ElementalResistMax", mod.TypeBase, 5, "Config", mod.Condition("Combat"), mod.Condition("EverlastingSacrifice")))
		c.modList.AddMod(newMod("ChaosResistMax", mod.TypeBase, 5, "Config", mod.Condition("Combat"), mod.Condition("EverlastingSacrifice")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1616-1618
	{name: "buffFanaticism", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:Fanaticism", "Config", mod.Condition("Combat"), mod.Condition("CanGainFanaticism")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1619-1621
	{name: "conditionHitsAlwaysStun", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AlwaysStunning", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1622-1624
	{name: "conditionHitsAlwaysHeavyStun", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AlwaysHeavyStunning", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1625-1627
	{name: "multiplierPvpTvalueOverride", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("MultiplierPvpTvalueOverride", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1628-1630
	{name: "multiplierPvpDamage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("PvpDamageMultiplier", mod.TypeMore, val.Number-100, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1631-1633
	{name: "reservedDarkness", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ReservedDarkness", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1634-1636
	{name: "multiplierHazardsTriggeredRecently", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:HazardsTriggeredRecently", mod.TypeBase, math.Max(val.Number, 0), "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1639-1641
	{name: "enemyDistance", varType: "count", defaultPlaceholder: configPlaceholder(20), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:enemyDistance", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1642-1644
	{name: "skillForkCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ForkedCount", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1645-1647
	{name: "skillChainCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("ChainCount", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1648-1650
	{name: "skillPierceCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("PiercedCount", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1651-1653
	{name: "multiplierBoltsReloadedPer6Seconds", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BoltsReloadedPastSixSeconds", mod.TypeOverride, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1654-1656
	{name: "multiplierBoltsReloadedPer8Seconds", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BoltsReloadedPastEightSeconds", mod.TypeOverride, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1657-1659
	{name: "multiplierDifferentAmmoFired", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentAmmoFired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1660-1662
	{name: "multiplierDifferentGrenadeFired", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentGrenadeFired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1663-1665
	{name: "conditionAtCloseRange", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:AtCloseRange", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1666-1668
	{name: "enemyMultiplierEnemyPresenceSeconds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:EnemyPresenceSeconds", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1669-1671
	{name: "conditionEnemyExitedPresenceRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:ExitedPresenceRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1672-1674
	{name: "conditionEnemyExitedPresenceRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:EnteredPresenceRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1675-1677
	{name: "conditionEnemyMoving", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Moving", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1678-1680
	{name: "conditionBleedAggravated", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:BleedAggravated", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1681-1683
	{name: "conditionEnemyFullLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:FullLife", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1684-1686
	{name: "conditionEnemyLowLife", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:LowLife", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1687-1691
	{name: "conditionEnemyArmourBroken", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:ArmourFullyBroken", "ArmourBreak", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:ArmourBrokenBelowZeroMax", "ArmourBreak", mod.Condition("Effective"), mod.ActorCondition("enemy", "CanArmourBreakBelowZero")))
		c.enemyModList.AddMod(newMod("Armour", mod.TypeOverride, 0, "ArmourBreak", mod.Condition("ArmourFullyBroken"), mod.GlobalEffect("Debuff").Name("ArmourBreak"), mod.ActorCondition("enemy", "CanArmourBreakBelowZero").Neg(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1692-1698
	{name: "multiplierArmourBreak", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(mod.NewFlag("Condition:ArmourBroken", val.Number > 0).Source("ArmourBreak").Tag(mod.Condition("Effective")))
		c.enemyModList.ReplaceMod(mod.NewFlag("Condition:ArmourFullyBroken", val.Number < 1).Source("ArmourBreak").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "CanArmourBreakBelowZero").Neg(true)))
		c.enemyModList.ReplaceMod(mod.NewFlag("Condition:ArmourBrokenBelowZeroMax", val.Number < 1).Source("ArmourBreak").Tag(mod.Condition("Effective"), mod.ActorCondition("enemy", "CanArmourBreakBelowZero")))
		c.enemyModList.AddMod(newMod("Multiplier:ArmourBreakStack", mod.TypeBase, val.Number, "ArmourBreak", mod.Condition("ArmourBroken")))
		c.enemyModList.AddMod(newMod("Armour", mod.TypeBase, -1, "ArmourBreak", mod.Multiplier("ArmourBreakStack"), mod.GlobalEffect("Debuff").Name("ArmourBreak")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1699-1701
	{name: "conditionEnemyCursed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Cursed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1702-1704
	{name: "conditionEnemyDazed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Dazed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1705-1707
	{name: "conditionEnemyStunned", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Stunned", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1708-1712
	{name: "conditionEnemyHeavyStunned", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:HeavyStunned", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Stunned", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Immobilised", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1713-1715
	{name: "conditionEnemyBleeding", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Bleeding", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1716-1718
	{name: "multiplierBleedsOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:BleedStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1719-1721
	{name: "multiplierRuptureStacks", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:RuptureStack", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1722-1724
	{name: "multiplierIncisionStackCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:IncisionStack", mod.TypeBase, math.Min(val.Number, 10), "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1725-1727
	{name: "multiplierFaerieFireStackCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:FaerieFireStack", mod.TypeBase, math.Min(val.Number, 10), "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1728-1730
	{name: "conditionEnemyPoisoned", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Poisoned", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1731-1733
	{name: "multiplierPoisonOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:PoisonStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1734-1736
	{name: "multiplierCurseExpiredOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:CurseExpired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1737-1739
	{name: "multiplierCurseDurationExpiredOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:CurseDurationExpired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1740-1742
	{name: "multiplierWitheredStackCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:WitheredStack", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1743-1747
	{name: "multiplierCorrosionStackCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:CorrosionStack", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newMod("Armour", mod.TypeBase, -5000, "Corrosion", mod.Multiplier("CorrosionStack"), mod.ActorCondition("enemy", "CanCorrode")))
		c.enemyModList.AddMod(newMod("Evasion", mod.TypeBase, -1000, "Corrosion", mod.Multiplier("CorrosionStack"), mod.ActorCondition("enemy", "CanCorrode")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1748-1751
	{name: "multiplierEnsnaredStackCount", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:EnsnareStackCount", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Moving", "Config", mod.MultiplierThreshold("EnsnareStackCount").Threshold(1).Actor("enemy")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1752-1754
	{name: "conditionEnemyMaimed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Maimed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1755-1757
	{name: "conditionEnemyHindered", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Hindered", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1758-1760
	{name: "conditionEnemyBlinded", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Blinded", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1761-1763
	{name: "overrideBuffBlinded", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("BlindEffect", mod.TypeOverride, val.Number, "Config", mod.GlobalEffect("Buff")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1764-1769
	{name: "conditionEnemyPinned", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Pinned", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Immobilised", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("CannotEvade", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:StunnedRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1770-1772
	{name: "conditionEnemyTaunted", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Taunted", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1773-1775
	{name: "conditionEnemyDebilitated", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Debilitated", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1776-1778
	{name: "conditionEnemyImmobilised", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Immobilised", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1779-1781
	{name: "conditionEnemyPacified", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Pacified", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1782-1784
	{name: "conditionEnemyAffectedByHazardRecently", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:AffectedByHazardRecently", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1785-1787
	{name: "conditionEnemyBurning", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Burning", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1788-1790
	{name: "conditionEnemyIgnited", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Ignited", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1791-1793
	{name: "multiplierIgniteOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:IgniteStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1794-1797
	{name: "conditionEnemyScorched", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Scorched", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:ScorchedConfig", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1798-1800
	{name: "conditionScorchedEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("ScorchVal", mod.TypeBase, val.Number, "Config", mod.Condition("ScorchedConfig")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1801-1803
	{name: "ScorchStacks", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ScorchStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1804-1807
	{name: "conditionEnemyOnScorchedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Scorched", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:OnScorchedGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1808-1811
	{name: "conditionEnemyChilled", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:ChilledConfig", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1812-1815
	{name: "multiplierChilledByYouSeconds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ChilledByYouSeconds", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.enemyModList.AddMod(newFlagMod("Condition:ChilledByYou", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1816-1818
	{name: "conditionEnemyChilledEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("ChillVal", mod.TypeBase, val.Number, "Chill", mod.Condition("ChilledConfig")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1819-1821
	{name: "ChillStacks", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ChillStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1822-1825
	{name: "conditionEnemyChilledByYourHits", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:ChilledByYourHits", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1826-1828
	{name: "HoarfrostStacks", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("HoarfrostFreezeDuration", mod.TypeIncrease, val.Number*20, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1829-1832
	{name: "conditionEnemyFrozen", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Frozen", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Immobilised", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1833-1836
	{name: "multiplierFrozenByYouSeconds", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:FrozenByYouSeconds", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		c.enemyModList.AddMod(newFlagMod("Condition:FrozenByYou", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1837-1840
	{name: "conditionEnemyOnChilledGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:OnChilledGround", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Chilled", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1841-1844
	{name: "conditionEnemyBrittle", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Brittle", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:BrittleConfig", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1845-1847
	{name: "conditionBrittleEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("BrittleVal", mod.TypeBase, val.Number, "Config", mod.Condition("BrittleConfig")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1848-1851
	{name: "conditionEnemyOnBrittleGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Brittle", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:OnBrittleGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1852-1855
	{name: "conditionEnemyShocked", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Shocked", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:ShockedConfig", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1856-1858
	{name: "conditionShockEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("ShockVal", mod.TypeBase, val.Number, "Shock", mod.Condition("ShockedConfig")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1859-1861
	{name: "ShockStacks", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ShockStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1862-1865
	{name: "conditionEnemyElectrocuted", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Electrocuted", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:Immobilised", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1866-1869
	{name: "conditionEnemyOnShockedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Shocked", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:OnShockedGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1870-1873
	{name: "conditionEnemySapped", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Sapped", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:SappedConfig", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1874-1876
	{name: "conditionSapEffect", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("SapVal", mod.TypeBase, val.Number, "Sap", mod.Condition("SappedConfig")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1877-1880
	{name: "conditionEnemyOnSappedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Sapped", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:OnSappedGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1881-1883
	{name: "multiplierFreezeShockIgniteOnEnemy", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:FreezeShockIgniteOnEnemy", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1884-1886
	{name: "conditionEnemyFireExposure", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("FireExposure", mod.TypeBase, 20, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyFireExposure")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1887-1889
	{name: "conditionEnemyColdExposure", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("ColdExposure", mod.TypeBase, 20, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyColdExposure")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1890-1892
	{name: "conditionEnemyLightningExposure", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("LightningExposure", mod.TypeBase, 20, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "CanApplyLightningExposure")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1893-1895
	{name: "conditionEnemyIsolated", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Isolated", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1896-1898
	{name: "conditionEnemyIntimidated", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Intimidated", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1899-1901
	{name: "conditionEnemyCrushed", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Crushed", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1902-1904
	{name: "conditionNearLinkedTarget", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:NearLinkedTarget", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1905-1907
	{name: "conditionEnemyUnnerved", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:Unnerved", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1908-1910
	{name: "conditionEnemyCriticalWeakness", varType: "check", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(mod.NewFlag("Condition:ApplyCriticalWeakness", val.Boolean).Source("Config").Tag(mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1911-1913
	{name: "enemyCriticalWeaknessStacks", varType: "count", defaultPlaceholder: configPlaceholder(20), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("SelfCritChance", mod.TypeBase, math.Max(math.Min(val.Number, 20), 0)/2, "Critical Weakness", mod.Condition("ApplyCriticalWeakness")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1914-1916
	{name: "conditionEnemyCoveredInAsh", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CoveredInAshEffect", mod.TypeBase, 20, "Covered in Ash"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1917-1919
	{name: "conditionEnemyCoveredInFrost", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CoveredInFrostEffect", mod.TypeBase, 20, "Covered in Frost"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1920-1923
	{name: "conditionEnemyHasOpenWeakness", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:EnemyHasOpenWeakness", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1924-1927
	{name: "conditionOpenWeaknessEnemyPresence", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:OpenWeaknessEnemyPresence", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1928-1930
	{name: "conditionEnemyOnConsecratedGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:OnConsecratedGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1931-1933
	{name: "conditionEnemyHaveEnergyShield", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:HaveEnergyShield", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1934-1938
	{name: "conditionEnemyOnProfaneGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:OnProfaneGround", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newMod("CurseEffectOnSelf", mod.TypeIncrease, 10, "Config", mod.Condition("OnProfaneGround")))
		c.enemyModList.AddMod(newMod("SelfCritChance", mod.TypeIncrease, 100, "Config", mod.Condition("OnProfaneGround")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1939-1941
	{name: "multiplierEnemyAffectedByGraspingVines", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:GraspingVinesAffectingEnemy", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1942-1944
	{name: "conditionEnemyOnFungalGround", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:OnFungalGround", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1945-1947
	{name: "conditionEnemyInChillingArea", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:InChillingArea", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1948-1950
	{name: "conditionEnemyInFrostGlobe", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:EnemyInFrostGlobe", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1951-1953
	{name: "enemyConditionHitByFireDamage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:HitByFireDamage", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1954-1956
	{name: "enemyConditionHitByColdDamage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:HitByColdDamage", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1957-1959
	{name: "enemyConditionHitByLightningDamage", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:HitByLightningDamage", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1960-1962
	{name: "enemyInRFOrScorchingRay", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:InRFOrScorchingRay", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1964-1966
	{name: "conditionBetweenYouAndLinkedTarget", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:BetweenYouAndLinkedTarget", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1967-1969
	{name: "conditionEnemyFireResZero", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("FireResist", mod.TypeOverride, 0, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1970-1972
	{name: "conditionEnemyColdResZero", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("ColdResist", mod.TypeOverride, 0, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1973-1975
	{name: "conditionEnemyLightningResZero", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newMod("LightningResist", mod.TypeOverride, 0, "Config", mod.Condition("Effective"), mod.ActorCondition("enemy", "HaveTrickstersSmile")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1979-1981
	{name: "conditionEnemyRareOrUnique", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:RareOrUnique", "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1982-2139
	{name: "enemyIsBoss", varType: "list", defaultState: configString("Pinnacle"), apply: applyEnemyIsBoss},
	//pob2:block:src/Modules/ConfigOptions.lua:2140-2161
	{name: "deliriousPercentage", varType: "list", apply: func(c *configApply, val ConfigValue) {
		if val.String == "20Percent" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -16, "20% Delirious"))
			c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 6, "20% Delirious"))
		}
		if val.String == "40Percent" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -32, "40% Delirious"))
			c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 12, "40% Delirious"))
		}
		if val.String == "60Percent" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -48, "60% Delirious"))
			c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 18, "60% Delirious"))
		}
		if val.String == "80Percent" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -64, "80% Delirious"))
			c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 24, "80% Delirious"))
		}
		if val.String == "100Percent" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -80, "100% Delirious"))
			c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 30, "100% Delirious"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2162-2164
	{name: "enemyPhysicalReduction", varType: "integer", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("PhysicalDamageReduction", mod.TypeBase, val.Number, "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2165-2167
	{name: "enemyLightningResist", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("LightningResist", mod.TypeBase, val.Number, "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2168-2170
	{name: "enemyColdResist", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("ColdResist", mod.TypeBase, val.Number, "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2171-2173
	{name: "enemyFireResist", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("FireResist", mod.TypeBase, val.Number, "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2174-2176
	{name: "enemyChaosResist", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("ChaosResist", mod.TypeBase, val.Number, "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2177-2179
	{name: "enemyMaxResist", varType: "check", apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("DoNotChangeMaxResFromConfig", "EnemyConfig"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2180-2182
	{name: "enemyBlockChance", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("BlockChance", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2183-2185
	{name: "enemyEvasion", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Evasion", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2186-2188
	{name: "enemyArmour", varType: "countAllowZero", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Armour", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:2189-2268
	{name: "presetBossSkills", varType: "list", defaultState: configString("None"), apply: applyPresetBossSkills},
	//pob2:block:src/Modules/ConfigOptions.lua:2280-2282
	{name: "enemyMultiplierPvpDamage", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("MultiplierPvpDamage", mod.TypeBase, val.Number, "Config"))
	}},
}

// applyEnemyIsBoss sets the placeholders the enemy options after it read.
//
//pob2:block:src/Modules/ConfigOptions.lua:1982-2139
func applyEnemyIsBoss(c *configApply, val ConfigValue) {
	unique := func(source mod.Source, name string, modType mod.Type, value float64) {
		c.enemyModList.AddMod(newMod(name, modType, value, source, mod.Condition("Effective")))
	}
	setResists := func(ele float64, set bool) {
		for _, name := range []string{"enemyLightningResist", "enemyColdResist", "enemyFireResist", "enemyChaosResist"} {
			if !set {
				delete(c.placeholder, name)
			} else if name == "enemyChaosResist" {
				c.placeholder[name] = 0
			} else {
				c.placeholder[name] = ele
			}
		}
	}
	setDefences := func(level int) {
		c.placeholder["enemyArmour"] = poe2.MonsterArmourTable[level-1]
		c.placeholder["enemyEvasion"] = poe2.MonsterEvasionTable[level-1]
	}
	commonUnique := func() {
		c.enemyModList.AddMod(newFlagMod("Condition:Unique", "Config", mod.Condition("Effective")))
		c.enemyModList.AddMod(newFlagMod("Condition:RareOrUnique", "Config", mod.Condition("Effective")))
	}
	uniqueMods := func() {
		unique("Unique", "CurseEffectOnSelf", mod.TypeMore, -50)
		unique("Unique", "ExposureEffectOnSelf", mod.TypeMore, -50)
		unique("Unique", "KnockbackDistanceOnSelf", mod.TypeMore, -75)
		unique("Unique", "SlowEffectOnSelf", mod.TypeMore, -50)
		unique("Unique", "MinimumMovementSpeed", mod.TypeBase, 20)
	}
	bossPower := func() {
		c.modList.AddMod(newMod("WarcryPower", mod.TypeBase, 20, "Boss"))
		c.modList.AddMod(newMod("Multiplier:EnemyPower", mod.TypeBase, 20, "Boss"))
	}

	c.placeholder["enemySpeed"] = 700
	c.placeholder["enemyCritChance"] = 5
	c.placeholder["enemyCritDamage"] = poe2.MonsterConstants["base_critical_hit_damage_bonus"]
	switch val.String {
	case "None":
		setResists(0, false)
		setDefences(configEnemyLevel(c.env))
	case "Boss":
		commonUnique()
		uniqueMods()
		unique("Unique", "PoiseThreshold", mod.TypeMore, 500)
		unique("Map Boss", "PoiseThreshold", mod.TypeMore, 213)
		bossPower()
		setResists(30, true)
		setDefences(configEnemyLevel(c.env))
	case "Pinnacle", "Uber":
		commonUnique()
		c.enemyModList.AddMod(newFlagMod("Condition:PinnacleBoss", "Config", mod.Condition("Effective")))
		uniqueMods()
		if val.String == "Uber" {
			c.enemyModList.AddMod(newMod("DamageTaken", mod.TypeMore, -70, "Boss"))
		}
		unique("Unique", "PoiseThreshold", mod.TypeMore, 500)
		unique("Xesht", "PoiseThreshold", mod.TypeMore, 838)
		bossPower()
		setResists(50, true)
		defaultLevel := max(configEnemyLevel(c.env), 82)
		mean := "Pinnacle"
		if val.String == "Uber" {
			mean = "Uber"
		}
		c.placeholder["enemyArmour"] = round(poe2.MonsterArmourTable[defaultLevel-1] * (poe2.BossStats[mean+"ArmourMean"] / 100))
		c.placeholder["enemyEvasion"] = round(poe2.MonsterEvasionTable[defaultLevel-1] * (poe2.BossStats[mean+"EvasionMean"] / 100))
	}
}

// applyPresetBossSkills is presetBossSkills' mods and its enemyDamageType input write; the placeholders it
// sets are only read by the defence estimations (presetBossSkillPlaceholders). The None branch only
// resets a control a non-None preset disabled, which a single load never does.
//
//pob2:block:src/Modules/ConfigOptions.lua:2189-2268
func applyPresetBossSkills(c *configApply, val ConfigValue) {
	if val.String == "None" {
		return
	}
	bossData, name, isUber, ok := presetBossSkill(c.env)
	if !ok {
		return
	}
	if bossData.DamageType != "" {
		c.env.ConfigInput["enemyDamageType"] = ConfigValue{String: bossData.DamageType, IsString: true}
	}

	c.modList.AddMod(newFlagMod("BossSkillActive", "Config"))

	if name == "Atziri Flameblast" && isUber {
		c.enemyModList.AddMod(newMod("Damage", mod.TypeIncrease, 60, "Alluring Abyss Map Mod"))
	}
	if bossData.AdditionalStats != nil {
		additionalStats := bossData.AdditionalStats.Base
		if isUber && bossData.AdditionalStats.Uber != nil {
			additionalStats = bossData.AdditionalStats.Uber
		}
		for _, k := range slices.Sorted(maps.Keys(additionalStats)) {
			if v := additionalStats[k]; v.IsFlag {
				c.enemyModList.AddMod(newFlagMod(k, "BossSkillAdditionalData"))
			} else {
				c.enemyModList.AddMod(newMod(k, mod.TypeBase, v.Value, "BossSkillAdditionalData"))
			}
		}
	}
}
