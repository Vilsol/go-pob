package calculator2

import (
	"math"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// configApply is the arguments every ConfigOptions apply function receives, plus the config set's
// placeholder table, which enemyIsBoss rewrites through SetPlaceholder.
type configApply struct {
	env          *Environment
	modList      *moddb.ModList
	enemyModList *moddb.ModList
	placeholder  map[string]float64
}

type configVar struct {
	name               string
	varType            string
	defaultState       *ConfigValue
	defaultPlaceholder *float64
	apply              func(c *configApply, val ConfigValue)
}

func configNumber(v float64) *ConfigValue { return &ConfigValue{Number: v, IsNumber: true} }
func configString(v string) *ConfigValue  { return &ConfigValue{String: v, IsString: true} }
func configBool(v bool) *ConfigValue      { return &ConfigValue{Boolean: v, IsBoolean: true} }
func configPlaceholder(v float64) *float64 {
	return &v
}

// configVarList is ConfigOptions.lua's varList, in its order, for the options that apply without user
// input: a defaultState, a defaultIndex or a placeholder. The quest options are applied by
// buildConfigModList; options a build has to set are not ported yet.
var configVarList = []configVar{
	//pob2:block:src/Modules/ConfigOptions.lua:117-119
	{name: "multiplierCurrentManaPercentage", varType: "count", defaultState: configNumber(100), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CurrentManaPercentage", mod.TypeBase, math.Max(math.Min(val.Number, 100), 0), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:157-160
	{name: "multiplierCurrentEnergyShield", varType: "count", defaultPlaceholder: configPlaceholder(100), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:UseCurrentEnergyShield", "Config"))
		c.modList.AddMod(newMod("Multiplier:CurrentEnergyShield", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:192-198
	{name: "resourceGainMode", varType: "list", defaultState: configString("AVERAGE"), apply: func(c *configApply, val ConfigValue) {
		switch val.String {
		case "AVERAGE":
			c.modList.AddMod(newFlagMod("Condition:AverageResourceGain", "Config"))
		case "MAX":
			c.modList.AddMod(newFlagMod("Condition:MaxResourceGain", "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:288-290
	{name: "targetBrandedEnemy", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:TargetingBrandedEnemy", "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:320-322
	{name: "ConcPathBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Consecrated Path of Endurance")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:338-342
	{name: "conditionCorruptingCryStages", varType: "count", defaultPlaceholder: configPlaceholder(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:CorruptingCryStageAfterFirst", mod.TypeBase, val.Number-1, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:356-358
	{name: "inDemonForm", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:DemonForm", "Config", mod.StatThreshold("Life", 2)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:359-361
	{name: "demonFormStacks", varType: "count", defaultPlaceholder: configPlaceholder(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DemonFlameStacks", mod.TypeBase, val.Number, "Config", mod.Condition("DemonForm")))
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
	//pob2:block:src/Modules/ConfigOptions.lua:398-400
	{name: "FlickerStrikeBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Flicker Strike").Transfigured(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:401-420
	{name: "elementalConfluxElement", varType: "list", defaultState: configNumber(1), apply: func(c *configApply, val ConfigValue) {
		var cold, fire, lightning float64
		switch val.Number {
		case 1:
			cold, fire, lightning = 3, 3, 3
		case 2:
			lightning = 1
		case 3:
			cold = 1
		case 4:
			fire = 1
		}
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxLightningEffect", mod.TypeBase, lightning, "Config", mod.GlobalEffect()))
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxColdEffect", mod.TypeBase, cold, "Config", mod.GlobalEffect()))
		c.modList.AddMod(newMod("Multiplier:ElementalConfluxFireEffect", mod.TypeBase, fire, "Config", mod.GlobalEffect()))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:458-464
	{name: "SecondsSinceInevitableCrit", varType: "count", defaultState: configNumber(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SecondsSinceInevitableCrit", mod.TypeBase, val.Number, "Config", mod.Condition("Combat")))
		if val.Number < 4 {
			c.modList.AddMod(newFlagMod("Condition:CritRecently", "Config", mod.Condition("Combat")))
			c.modList.AddMod(newFlagMod("Condition:SkillCritRecently", "Config", mod.Condition("Combat")))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:474-476
	{name: "purpleFlameStacks", varType: "count", defaultState: configNumber(10), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:PurpleFlamesCount", mod.TypeBase, math.Min(val.Number, 10), "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:544-549
	{name: "raiseSpectreEnableBuffs", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillType("Buff"), mod.SkillName("Spectre").PartialMatch(true).SummonSkill(true)))
	}},
	{name: "raiseSpectreEnableCurses", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillType("AppliesCurse"), mod.SkillName("Spectre").PartialMatch(true).SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:599-601
	{name: "sigilOfPowerStages", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:SigilOfPowerStage", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:644-649
	{name: "summonCompanionEnableBuffs", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillType("Buff"), mod.SkillName("Companion").PartialMatch(true).SummonSkill(true)))
	}},
	{name: "summonCompanionEnableCurses", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillType("AppliesCurse"), mod.SkillName("Companion").PartialMatch(true).SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:651-659
	{name: "summonElementalRelicEnableAngerAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillId("Anger"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	{name: "summonElementalRelicEnableHatredAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillId("Hatred"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	{name: "summonElementalRelicEnableWrathAura", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(enableSkillData(mod.SkillId("Wrath"), mod.SkillName("Summon Elemental Relic").SummonSkill(true)))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:715-717
	{name: "VigilantStrikeBypassCD", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newMod("CooldownRecovery", mod.TypeOverride, 0, "Config", mod.SkillName("Vigilant Strike")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:770-794
	{name: "enemySizePreset", varType: "list", defaultState: configString("Medium"), apply: func(c *configApply, val ConfigValue) {
		radius := map[string]float64{"Small": 2, "Medium": 3, "Large": 5, "Huge": 11}
		if r, ok := radius[val.String]; ok {
			c.placeholder["enemyRadius"] = r
			c.modList.AddMod(newMod("EnemyRadius", mod.TypeBase, r, "Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:860-862
	{name: "configBossFaceBroken", varType: "count", apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:BossFaceBroken", mod.TypeBase, val.Number, "Config"))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:866-879
	{name: "repeatMode", varType: "list", defaultState: configString("AVERAGE"), apply: func(c *configApply, val ConfigValue) {
		switch val.String {
		case "AVERAGE":
			c.modList.AddMod(newFlagMod("Condition:averageRepeat", "Config"))
			c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:averageRepeat", "Config")}).Source("Config"))
		case "FINAL", "FINAL_DPS":
			c.modList.AddMod(newFlagMod("Condition:alwaysFinalRepeat", "Config"))
			c.modList.AddMod(mod.NewList("MinionModifier", mod.MinionModifier{Mod: newFlagMod("Condition:alwaysFinalRepeat", "Config")}).Source("Config"))
		}
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1032-1034
	{name: "companionInPresence", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.modList.AddMod(newFlagMod("Condition:CompanionInPresence", "Config", mod.Condition("HaveCompanion")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1119-1121
	{name: "multiplierWitheredStackCountSelf", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(15), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WitheredStack", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1281-1283
	{name: "multiplierStunnedRecently", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:StunnedRecently", mod.TypeBase, math.Min(val.Number, 100), "Config", mod.Condition("Combat"), mod.Condition("StunnedRecently")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1421-1423
	{name: "conditionChampionIntimidate", varType: "check", defaultState: configBool(true), apply: func(c *configApply, _ ConfigValue) {
		c.enemyModList.AddMod(newFlagMod("Condition:ChampionIntimidate", "Config", mod.Condition("Combat")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1503-1505
	{name: "multiplierWarcryUsedRecently", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:WarcryUsedRecently", mod.TypeBase, math.Min(val.Number, 100), "Config", mod.Condition("Combat"), mod.Condition("UsedWarcryRecently")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1609-1611
	{name: "GamblesprintMovementSpeed", varType: "list", defaultState: configNumber(20), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("MovementSpeed", mod.TypeIncrease, val.Number, "Config", mod.Condition("Combat"), mod.Condition("HaveGamblesprint")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1639-1641
	{name: "enemyDistance", varType: "count", defaultPlaceholder: configPlaceholder(20), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:enemyDistance", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1657-1662
	{name: "multiplierDifferentAmmoFired", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentAmmoFired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	{name: "multiplierDifferentGrenadeFired", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.modList.AddMod(newMod("Multiplier:DifferentGrenadeFired", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1801-1803
	{name: "ScorchStacks", varType: "countAllowZero", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ScorchStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1819-1821
	{name: "ChillStacks", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ChillStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1859-1861
	{name: "ShockStacks", varType: "count", defaultPlaceholder: configPlaceholder(1), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("Multiplier:ShockStacks", mod.TypeBase, val.Number, "Config", mod.Condition("Effective")))
	}},
	//pob2:block:src/Modules/ConfigOptions.lua:1911-1913
	{name: "enemyCriticalWeaknessStacks", varType: "count", defaultPlaceholder: configPlaceholder(20), apply: func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod("SelfCritChance", mod.TypeBase, math.Max(math.Min(val.Number, 20), 0)/2, "Critical Weakness", mod.Condition("ApplyCriticalWeakness")))
	}},
	{name: "enemyIsBoss", varType: "list", defaultState: configString("Pinnacle"), apply: applyEnemyIsBoss},
	//pob2:block:src/Modules/ConfigOptions.lua:2165-2176
	{name: "enemyLightningResist", varType: "countAllowZero", apply: enemyConfigBase("LightningResist", "EnemyConfig")},
	{name: "enemyColdResist", varType: "countAllowZero", apply: enemyConfigBase("ColdResist", "EnemyConfig")},
	{name: "enemyFireResist", varType: "countAllowZero", apply: enemyConfigBase("FireResist", "EnemyConfig")},
	{name: "enemyChaosResist", varType: "countAllowZero", apply: enemyConfigBase("ChaosResist", "EnemyConfig")},
	//pob2:block:src/Modules/ConfigOptions.lua:2183-2188
	{name: "enemyEvasion", varType: "countAllowZero", apply: enemyConfigBase("Evasion", "Config")},
	{name: "enemyArmour", varType: "countAllowZero", apply: enemyConfigBase("Armour", "Config")},
}

func enableSkillData(tags ...mod.Tag) mod.Mod {
	return mod.NewList("SkillData", mod.SkillData{Key: "enable", Value: 1}).Source("Config").Tag(tags...)
}

func enemyConfigBase(name string, source mod.Source) func(c *configApply, val ConfigValue) {
	return func(c *configApply, val ConfigValue) {
		c.enemyModList.AddMod(newMod(name, mod.TypeBase, val.Number, source))
	}
}

// applyEnemyIsBoss sets the placeholders the enemy options after it read. The Pinnacle and Uber armour
// and evasion placeholders scale by data.bossStats, which the data module does not carry, so the saved
// placeholders stand in for them.
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
	}
}

// configPlaceholders is the active config set's placeholder table after Load: each option's
// defaultPlaceholderState, then the saved Placeholder elements.
//
//pob2:block:src/Classes/ConfigTab.lua:1323-1331
//pob2:block:src/Classes/ConfigTab.lua:907-913
func configPlaceholders(env *Environment) map[string]float64 {
	placeholder := map[string]float64{}
	for _, v := range configVarList {
		if v.defaultPlaceholder != nil {
			placeholder[v.name] = *v.defaultPlaceholder
		}
	}
	placeholders := env.Build.Config.Placeholders
	for _, set := range env.Build.Config.ConfigSets {
		if set.ID == activeConfigSetID(env.Build) {
			placeholders = set.Placeholders
		}
	}
	for _, p := range placeholders {
		if p.Number != nil {
			placeholder[p.Name] = *p.Number
		}
	}
	return placeholder
}

// applyConfigVars is the varList loop of ConfigTab:BuildModList: an option's input (its saved value, else
// its defaultState or defaultIndex value) and, for the number types, its placeholder when the input is
// unset.
//
//pob2:block:src/Classes/ConfigTab.lua:1177-1199
func applyConfigVars(env *Environment, modList, enemyModList *moddb.ModList) {
	c := &configApply{env: env, modList: modList, enemyModList: enemyModList, placeholder: configPlaceholders(env)}
	for _, v := range configVarList {
		input, set := env.ConfigInput[v.name]
		if !set && v.defaultState != nil {
			input, set = *v.defaultState, true
		}
		switch v.varType {
		case "check":
			if set && (!input.IsBoolean || input.Boolean) {
				v.apply(c, ConfigValue{Boolean: true, IsBoolean: true})
			}
		case "count", "integer", "countAllowZero", "float":
			if set && input.IsNumber && (input.Number != 0 || v.varType != "count") {
				v.apply(c, input)
			} else if p, ok := c.placeholder[v.name]; ok && (p != 0 || v.varType != "count") {
				v.apply(c, ConfigValue{Number: p, IsNumber: true})
			}
		case "list", "text":
			if set {
				v.apply(c, input)
			}
		}
	}
}
