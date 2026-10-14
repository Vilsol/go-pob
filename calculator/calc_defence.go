package calculator

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

//pob:block:src/Modules/CalcDefence.lua:875-885
func enemyConfigNumber(env *Environment, name string) float64 {
	value, _ := enemyConfigNumberSource(env, name)
	return value
}

//pob:block:src/Modules/CalcDefence.lua:875-885
func enemyConfigNumberSource(env *Environment, name string) (float64, bool) {
	if value, ok := env.Build.GetNumberOptionOK(name); ok {
		return value, true
	}
	return env.ConfigPlaceholder[name], false
}

var resistTypeList = []string{
	"Fire",
	"Cold",
	"Lightning",
	"Chaos",
}

var isElemental = map[string]bool{
	"Fire":      true,
	"Cold":      true,
	"Lightning": true,
}

//pob:src/Modules/CalcDefence.lua:66-71
func CalcArmourReductionF(armour float64, raw float64) float64 {
	if armour == 0 && raw == 0 {
		return 0
	}
	return armour / (armour + float64(raw*5)) * 100
}

//pob:src/Modules/CalcDefence.lua:74-76
func CalcArmourReduction(armour float64, raw float64) float64 {
	return math.Round(CalcArmourReductionF(armour, raw))
}

//pob:src/Modules/CalcDefence.lua:83-142
func ApplyDmgTakenConversion(skillModList *moddb.ModList, output map[string]float64, breakdown *Breakdown, sourceType data.DamageType, baseDmg float64) ([]string, float64) {
	damageBreakdown := make([]string, 0)
	totalDamageTaken := float64(0)

	shiftTable := make(map[data.DamageType]float64, 5)
	totalTakenAs := float64(0)
	for _, damageType := range data.DamageType("").Values() {
		eleTakenAs := ""
		eleFromHitsTakenAs := ""
		if isElemental[string(sourceType)] {
			eleTakenAs = string("ElementalDamageTakenAs" + damageType)
			eleFromHitsTakenAs = string("ElementalDamageFromHitsTakenAs" + damageType)
		}
		shiftTable[damageType] = skillModList.Sum(mod.TypeBase, nil,
			string(sourceType+"DamageTakenAs"+damageType),
			string(sourceType+"DamageFromHitsTakenAs"+damageType),
			eleTakenAs, eleFromHitsTakenAs)
		totalTakenAs += shiftTable[damageType]
	}

	for _, damageType := range data.DamageType("").Values() {
		damageTakenAs := shiftTable[damageType] / 100
		if damageType == sourceType {
			damageTakenAs = math.Max(1-totalTakenAs/100, 0)
		}
		if damageTakenAs == 0 {
			continue
		}
		damage := baseDmg * damageTakenAs

		baseTakenInc := skillModList.Sum(mod.TypeIncrease, nil, "DamageTaken", string(damageType+"DamageTaken"), "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
		baseTakenMore := skillModList.More(nil, "DamageTaken", string(damageType+"DamageTaken"), "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
		if isElemental[string(damageType)] {
			baseTakenInc += skillModList.Sum(mod.TypeIncrease, nil, "ElementalDamageTaken", "ElementalDamageTakenWhenHit")
			baseTakenMore *= skillModList.More(nil, "ElementalDamageTaken", "ElementalDamageTakenWhenHit")
		}
		damageTakenMods := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
		reduction := utils.Ternary(skillModList.Flag(nil, string("SelfIgnoreBase"+damageType+"DamageReduction")), 0, output[string("Base"+damageType+"DamageReductionWhenHit")])
		resist := utils.Ternary(skillModList.Flag(nil, string("SelfIgnore"+damageType+"Resistance")), 0, utils.OrF(output[string(damageType+"ResistWhenHit")], output[string(damageType+"Resist")]))

		armourReduct := float64(0)
		resMult := 1 - resist/100

		percentOfArmourApplies := math.Min(utils.Ternary(
			skillModList.Flag(nil, string("ArmourDoesNotApplyTo"+damageType+"DamageTaken")),
			float64(0),
			skillModList.Sum(mod.TypeBase, nil, string("ArmourAppliesTo"+damageType+"DamageTaken")),
		), 100)
		physicalReductionBasedOnWard := damageType == data.DamageTypePhysical && skillModList.Flag(nil, "PhysicalReductionBasedOnWard")
		if percentOfArmourApplies > 0 || physicalReductionBasedOnWard {
			effArmour := (output["Armour"] * percentOfArmourApplies / 100) * (1 + output["ArmourDefense"])
			if physicalReductionBasedOnWard {
				multiplier := float64(0)
				if override := skillModList.Override(nil, "PhysicalReductionBasedOnWardPercent"); override != nil {
					multiplier = override.Float() / 100
				}
				effArmour = output["Ward"] * multiplier
			}
			effDamage := damage * resMult
			if effArmour != 0 && effDamage != 0 {
				armourReduct = CalcArmourReduction(effArmour, effDamage)
			}
			armourReduct = math.Min(output["DamageReductionMax"], armourReduct)
		}
		reductMult := (1 - math.Max(math.Min(output["DamageReductionMax"], armourReduct+reduction), 0)/100) * damageTakenMods
		combinedMult := resMult * reductMult
		finalDamage := float64(damage * combinedMult)
		totalDamageTaken += finalDamage

		if breakdown != nil {
			damageBreakdown = append(damageBreakdown, string(damageType)+" Damage Taken")
			if damageTakenAs != 1 {
				damageBreakdown = append(damageBreakdown, fmt.Sprintf("^8=^7 %d^8 (Base Damage)^7 * %.2f^8 (Damage taken as %s)", int(baseDmg), damageTakenAs, damageType))
			}
			if combinedMult != 1 {
				damageBreakdown = append(damageBreakdown, fmt.Sprintf("^8=^7 %d^8 (%s Damage)^7 * %.4f^8 (Damage taken multi)", int(damage), damageType, combinedMult))
			}
			damageBreakdown = append(damageBreakdown, fmt.Sprintf("^8=^7 %d^8 (%s Damage taken)", int(finalDamage), damageType))
			damageBreakdown = append(damageBreakdown, "")
		}
	}
	return damageBreakdown, totalDamageTaken
}

//pob:src/Modules/CalcDefence.lua:177-181
func calcLifeHitPoolWithLossPrevention(life float64, maxLife float64, lifeLossPrevented float64, lifeLossBelowHalfPrevented float64) float64 {
	halfLife := float64(maxLife * 0.5)
	aboveLow := math.Max(life-halfLife, 0)
	return aboveLow/(1-lifeLossPrevented/100) + math.Min(life, halfLife)/(1-lifeLossBelowHalfPrevented/100)/(1-lifeLossPrevented/100)
}

//pob:src/Modules/CalcDefence.lua:2307-2319
func calcMoMEBPoolF(lifePool float64, momEffect float64, esBypass float64, manaUnreserved float64, energyShieldRecoveryCap float64, energyShieldProtectsMana bool) (float64, float64, float64, float64) {
	mana := math.Max(manaUnreserved, 0)
	maxMoMPool := math.Inf(1)
	if momEffect < 1 {
		maxMoMPool = lifePool/(1-momEffect) - lifePool
	}
	maxManaUsable := math.Floor(math.Min(mana, maxMoMPool))
	maxESUsable := float64(0)
	if energyShieldProtectsMana && esBypass < 1 {
		maxESUsable = math.Floor(math.Min(
			energyShieldRecoveryCap,
			math.Min(
				maxMoMPool*(1-esBypass),
				(lifePool+maxManaUsable)/(1-float64((1-esBypass)*momEffect))-(lifePool+maxManaUsable),
			),
		))
	}
	manaUsed := math.Floor(math.Min(maxMoMPool-maxESUsable, maxManaUsable))
	return lifePool + manaUsed + maxESUsable, maxManaUsable, manaUsed, maxESUsable
}

//pob:src/Modules/CalcDefence.lua:57-63
func CalcHitChance(evasion float64, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := accuracy / (accuracy + math.Pow(evasion/5, 0.9)) * 125
	return math.Max(math.Min(math.Round(rawChance), 100), 5)
}

//pob:src/Modules/CalcDefence.lua:478-504
func DefenceForConditionals(actor *Actor) {
	modDB := actor.ModDB

	for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
		itemData := actor.ItemList[slot]
		if itemData == nil || itemData.ArmourData == nil {
			continue
		}
		armourData := itemData.ArmourData

		if !modDB.Flag(nil, "GainNoWardFrom"+slot) && armourData.Ward > 0 {
			actor.Output["WardOn"+slot] = armourData.Ward
		}
		if !modDB.Flag(nil, "GainNoEnergyShieldFrom"+slot) && armourData.EnergyShield > 0 {
			actor.Output["EnergyShieldOn"+slot] = armourData.EnergyShield
		}
		if !modDB.Flag(nil, "GainNoArmourFrom"+slot) && armourData.Armour > 0 {
			actor.Output["ArmourOn"+slot] = armourData.Armour
		}
		if !modDB.Flag(nil, "GainNoEvasionFrom"+slot) && armourData.Evasion > 0 {
			actor.Output["EvasionOn"+slot] = armourData.Evasion
		}
	}
}

//pob:src/Modules/CalcDefence.lua:32-42
type allyLifePool struct {
	key        string
	life       string
	mitigation string
	redirect   string
	fallback   string
	meleeOnly  bool
	source     string
	limit      string
}

var allyLifePoolList = []allyLifePool{
	{key: "minion", life: "TotalMinionLife", mitigation: "MinionAllyDamageMitigation", redirect: "takenFromMinionBeforeYou", fallback: "Multiplier:MinionLife"},
	{key: "radianceSentinel", life: "TotalRadianceSentinelLife", mitigation: "RadianceSentinelAllyDamageMitigation", redirect: "takenFromRadianceSentinelBeforeYou"},
	{key: "spectres", life: "TotalSpectreLife", mitigation: "SpectreAllyDamageMitigation", redirect: "takenFromSpectresBeforeYou"},
	{key: "totems", life: "TotalTotemLife", mitigation: "TotemAllyDamageMitigation", redirect: "takenFromTotemsBeforeYou"},
	{key: "vaalRejuvenationTotems", life: "TotalVaalRejuvenationTotemLife", mitigation: "VaalRejuvenationTotemAllyDamageMitigation", redirect: "takenFromVaalRejuvenationTotemsBeforeYou"},
	{key: "voidSpawn", life: "TotalVoidSpawnLife", mitigation: "VoidSpawnAllyDamageMitigation", redirect: "takenFromVoidSpawnBeforeYou"},
	{key: "stoneGolem", life: "TotalStoneGolemLife", mitigation: "StoneGolemAllyDamageMitigation", redirect: "takenFromStoneGolemBeforeYou", meleeOnly: true},
	{key: "soulLink", life: "AlliedEnergyShield", mitigation: "SoulLinkMitigation"},
	{key: "frostShield", life: "FrostShieldLife", mitigation: "FrostShieldDamageMitigation"},
}

var minionLifePoolBySkill = map[string]allyLifePool{
	"Summon Sentinel of Radiance":        {key: "radianceSentinel", life: "TotalRadianceSentinelLife", redirect: "takenFromRadianceSentinelBeforeYou", source: "Sentinel of Radiance"},
	"Summon Void Spawn":                  {key: "voidSpawn", life: "TotalVoidSpawnLife", redirect: "takenFromVoidSpawnBeforeYou", source: "Void Spawns", limit: "ActiveVoidSpawnLimit"},
	"Summon Stone Golem of Safeguarding": {key: "stoneGolem", life: "TotalStoneGolemLife", redirect: "takenFromStoneGolemBeforeYou", source: "Stone Golem"},
}

//pob:block:src/Modules/CalcPerform.lua:1233-1238
var allyLifeRedirects = []string{
	"takenFromSpectresBeforeYou", "takenFromMinionBeforeYou",
	"takenFromRadianceSentinelBeforeYou", "takenFromVoidSpawnBeforeYou", "takenFromStoneGolemBeforeYou",
	"takenFromTotemsBeforeYou", "takenFromVaalRejuvenationTotemsBeforeYou",
}

func allyLifePoolByKey(key string) allyLifePool {
	for _, ally := range allyLifePoolList {
		if ally.key == key {
			return allyLifePool{key: ally.key, life: ally.life, redirect: ally.redirect, source: "Minion"}
		}
	}
	return allyLifePool{}
}

func enemyDamageCategory(environment *Environment) string {
	if preset := environment.Build.GetStringOption("presetBossSkills"); preset != "" && preset != "None" {
		if bossData, ok := data.BossSkills[preset]; ok && bossData.DamageType != "" {
			return bossData.DamageType
		}
	}
	if c := environment.Build.GetStringOption("enemyDamageType"); c != "" {
		return c
	}
	return "Average"
}

func CalculateDefence(environment *Environment, actor *Actor) {
	modDB := actor.ModDB

	// Action Speed
	actor.Output["ActionSpeedMod"] = CalcActionSpeedMod(actor)

	calcResistances(actor, modDB)

	//pob:block:src/Modules/CalcDefence.lua:648-653
	if environment.Minion != nil {
		if pct := modDB.Sum(mod.TypeBase, nil, "ResistanceAddedToMinions"); pct > 0 {
			for _, elem := range resistTypeList {
				environment.Minion.Actor.ModDB.AddMod(mod.NewFloat(elem+"Resist", mod.TypeBase,
					math.Floor(actor.Output[elem+"Resist"]*pct/100)).Source("Player"))
			}
		}
	}
	// Formless Inferno
	//pob:block:src/Modules/CalcDefence.lua:655-658
	if environment.Minion != nil && actor == environment.Minion.Actor {
		doActorLifeMana(actor)
		doActorLifeManaReservation(actor, false)
	}

	//pob:src/Modules/CalcDefence.lua:661
	actor.Output["BlockChanceMax"] = min(modDB.Sum(mod.TypeBase, nil, "BlockChanceMax"), data.BlockChanceCap)
	actor.Output["BlockChanceOverCap"] = 0
	actor.Output["SpellBlockChanceOverCap"] = 0
	baseBlockChance := float64(0)

	if actor.ItemList["Weapon 2"] != nil && actor.ItemList["Weapon 2"].ArmourData != nil {
		baseBlockChance = baseBlockChance + actor.ItemList["Weapon 2"].ArmourData.BlockChance
	}
	if actor.ItemList["Weapon 3"] != nil && actor.ItemList["Weapon 3"].ArmourData != nil {
		baseBlockChance = baseBlockChance + actor.ItemList["Weapon 3"].ArmourData.BlockChance
	}
	actor.Output["ShieldBlockChance"] = baseBlockChance
	if _, necroAegis := environment.KeystonesAdded["Necromantic Aegis"]; !necroAegis {
		if o := modDB.Override(nil, "ReplaceShieldBlock"); o != nil {
			baseBlockChance = utils.Or(o, baseBlockChance)
		}
	} else if actor.MinionData != nil {
		if o := environment.Player.ModDB.Override(nil, "ReplaceShieldBlock"); o != nil {
			baseBlockChance = utils.Or(o, baseBlockChance)
		}
	}
	if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		actor.Output["BlockChance"] = actor.Output["BlockChanceMax"]
	} else {
		inc := modDB.Sum(mod.TypeIncrease, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalBlockChance := utils.RoundTo((baseBlockChance+modDB.Sum(mod.TypeBase, nil, "BlockChance"))*(1+inc/100)*more, 0)
		actor.Output["BlockChance"] = min(totalBlockChance, actor.Output["BlockChanceMax"])
		actor.Output["BlockChanceOverCap"] = max(0, totalBlockChance-actor.Output["BlockChanceMax"])
	}
	actor.Output["ProjectileBlockChance"] = min(actor.Output["BlockChance"]+float64(modDB.Sum(mod.TypeBase, nil, "ProjectileBlockChance")*calclib.Mod(modDB, nil, "BlockChance")), actor.Output["BlockChanceMax"])
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		actor.Output["SpellBlockChanceMax"] = actor.Output["BlockChanceMax"]
	} else {
		actor.Output["SpellBlockChanceMax"] = min(modDB.Sum(mod.TypeBase, nil, "SpellBlockChanceMax"), data.BlockChanceCap)
	}
	if modDB.Flag(nil, "MaxSpellBlockIfNotBlockedRecently") {
		actor.Output["SpellBlockChance"] = actor.Output["SpellBlockChanceMax"]
		actor.Output["SpellProjectileBlockChance"] = actor.Output["SpellBlockChanceMax"]
	} else if modDB.Flag(nil, "SpellBlockChanceIsBlockChance") {
		actor.Output["SpellBlockChance"] = actor.Output["BlockChance"]
		actor.Output["SpellProjectileBlockChance"] = actor.Output["ProjectileBlockChance"]
		actor.Output["SpellBlockChanceOverCap"] = actor.Output["BlockChanceOverCap"]
	} else {
		inc := modDB.Sum(mod.TypeIncrease, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalSpellBlockChance := utils.RoundTo(modDB.Sum(mod.TypeBase, nil, "SpellBlockChance")*(1+inc/100)*more, 0)
		actor.Output["SpellBlockChance"] = min(totalSpellBlockChance, actor.Output["SpellBlockChanceMax"])
		actor.Output["SpellBlockChanceOverCap"] = max(0, totalSpellBlockChance-actor.Output["SpellBlockChanceMax"])
		actor.Output["SpellProjectileBlockChance"] = max(min(actor.Output["SpellBlockChance"]+float64(modDB.Sum(mod.TypeBase, nil, "ProjectileSpellBlockChance")*calclib.Mod(modDB, nil, "SpellBlockChance")), actor.Output["SpellBlockChanceMax"]), 0)
	}
	//pob:block:src/Modules/CalcDefence.lua:718-728
	if actor.Breakdown != nil {
		actor.Breakdown.Data["BlockChance"] = &Entry{Lines: []string{
			fmt.Sprintf("Base: %.14g%%", baseBlockChance),
			fmt.Sprintf("Max: %.14g%%", actor.Output["BlockChanceMax"]),
			fmt.Sprintf("Total: %.14g%%", actor.Output["BlockChance"]+actor.Output["BlockChanceOverCap"]),
		}}
		actor.Breakdown.Data["SpellBlockChance"] = &Entry{Lines: []string{
			fmt.Sprintf("Max: %.14g%%", actor.Output["SpellBlockChanceMax"]),
			fmt.Sprintf("Total: %.14g%%", actor.Output["SpellBlockChance"]+actor.Output["SpellBlockChanceOverCap"]),
		}}
	}
	if modDB.Flag(nil, "CannotBlockAttacks") {
		actor.Output["BlockChance"] = 0
		actor.Output["ProjectileBlockChance"] = 0
	}
	if modDB.Flag(nil, "CannotBlockSpells") {
		actor.Output["SpellBlockChance"] = 0
		actor.Output["SpellProjectileBlockChance"] = 0
	}
	for _, blockType := range []string{"BlockChance", "ProjectileBlockChance", "SpellBlockChance", "SpellProjectileBlockChance"} {
		key := "Effective" + blockType
		if environment.ModeEffective {
			actor.Output[key] = max(actor.Output[blockType]-actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "reduceEnemyBlock"), 0)
		} else {
			actor.Output[key] = actor.Output[blockType]
		}

		blockRolls := float64(0)
		if environment.ModeEffective {
			if modDB.Flag(nil, blockType+"IsLucky") {
				blockRolls++
			}
			if modDB.Flag(nil, blockType+"IsUnlucky") {
				blockRolls--
			}
			if modDB.Flag(nil, "ExtremeLuck") {
				blockRolls *= 2
			}
		}
		if worstOf, ok := environment.Build.GetNumberOptionOK("EHPUnluckyWorstOf"); ok && worstOf != 1 {
			blockRolls = -worstOf / 2
		}

		if blockRolls != 0 {
			blockChance := actor.Output[key] / 100
			switch {
			case modDB.Flag(nil, "Unexciting"):
				actor.Output[key] = (float64(3*blockChance*blockChance) - float64(2*blockChance*blockChance*blockChance)) * 100
			case blockRolls > 0:
				actor.Output[key] = (1 - math.Pow(1-blockChance, blockRolls+1)) * 100
			default:
				actor.Output[key] = math.Pow(blockChance, math.Abs(blockRolls)) * actor.Output[key]
			}
		}
	}
	actor.Output["EffectiveAverageBlockChance"] = (actor.Output["EffectiveBlockChance"] +
		actor.Output["EffectiveProjectileBlockChance"] +
		actor.Output["EffectiveSpellBlockChance"] +
		actor.Output["EffectiveSpellProjectileBlockChance"]) / 4
	//pob:block:src/Modules/CalcDefence.lua:768-772
	actor.Output["BlockEffect"] = 100 - modDB.Sum(mod.TypeBase, nil, "BlockEffect")
	if actor.Output["BlockEffect"] != 0 {
		actor.Output["ShowBlockEffect"] = 1
		actor.Output["DamageTakenOnBlock"] = 100 - actor.Output["BlockEffect"]
	}

	if modDB.Flag(nil, "ArmourAppliesToEnergyShieldRecharge") {
		// Armour to ES Recharge conversion from Armour and Energy Shield Mastery
		multiplier := utils.OrF(modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge"), 100) / 100
		for _, value := range modDB.Tabulate("INC", nil, "Armour", "ArmourAndEvasion", "Defences") {
			modDB.AddMod(value.Mod.Clone().SetName("EnergyShieldRecharge").SetValue(mod.NewModValueFloat(math.Floor(value.Mod.Value().Float() * multiplier))).Tag(calclib.GetConvertedModTags(value.Mod, multiplier, false)...))
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:784-825
	for _, conv := range []struct{ flag, name, output string }{
		{"ArmourIncreasedByUncappedFireRes", "Armour", "FireResistTotal"},
		{"ArmourIncreasedByOvercappedFireRes", "Armour", "FireResistOverCap"},
		{"EvasionRatingIncreasedByUncappedColdRes", "Evasion", "ColdResistTotal"},
		{"EvasionRatingIncreasedByOvercappedColdRes", "Evasion", "ColdResistOverCap"},
		{"EnergyShieldIncreasedByChanceToBlockSpellDamage", "EnergyShield", "SpellBlockChance"},
		{"EnergyShieldIncreasedByChaosResistance", "EnergyShield", "ChaosResist"},
	} {
		if !modDB.Flag(nil, conv.flag) {
			continue
		}
		for _, value := range modDB.Tabulate(mod.TypeFlag, nil, conv.flag) {
			modDB.AddMod(mod.NewFloat(conv.name, mod.TypeIncrease, actor.Output[conv.output]).Source(value.Mod.GetSource()))
			break
		}
	}

	// Primary defences: Energy shield, evasion and armour
	{
		ironReflexes := modDB.Flag(nil, "IronReflexes")
		ward := float64(0)
		energyShield := float64(0)
		armour := float64(0)
		evasion := float64(0)
		energyShieldBase := float64(0)
		armourBase := float64(0)
		evasionBase := float64(0)
		wardBase := float64(0)
		gearWard := float64(0)
		gearEnergyShield := float64(0)
		gearArmour := float64(0)
		gearEvasion := float64(0)
		slotCfg := &moddb.ListCfg{}
		for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
			itemData := actor.ItemList[slot]
			var armourData *ArmourData
			if itemData != nil {
				armourData = itemData.ArmourData
			}
			if armourData != nil {
				slotCfg.SlotName = slot
				energyShieldBase = 0
				if !modDB.Flag(nil, "GainNoEnergyShieldFrom"+slot) {
					energyShieldBase = armourData.EnergyShield
				}
				armourBase = 0
				if !modDB.Flag(nil, "GainNoArmourFrom"+slot) {
					armourBase = armourData.Armour
				}
				evasionBase = 0
				if !modDB.Flag(nil, "GainNoEvasionFrom"+slot) && !(modDB.Flag(nil, "GainNoArmourFrom"+slot) && ironReflexes) {
					evasionBase = armourData.Evasion
				}
				wardBase = 0
				if !modDB.Flag(nil, "GainNoWardFrom"+slot) {
					wardBase = armourData.Ward
				}

				if slot == "Body Armour" && modDB.Flag(nil, "ConvertBodyArmourArmourEvasionToWard") {
					conversion := math.Min(modDB.Sum(mod.TypeBase, nil, "BodyArmourArmourEvasionToWardPercent")/100, 1)
					convertedArmour := float64(armourBase * conversion)
					convertedEvasion := float64(evasionBase * conversion)
					armourBase = armourBase - convertedArmour
					evasionBase = evasionBase - convertedEvasion
					wardBase = wardBase + (convertedEvasion + convertedArmour)
				}

				if wardBase > 0 {
					if modDB.Flag(nil, "EnergyShieldToWard") {
						inc := modDB.Sum(mod.TypeIncrease, slotCfg, "Ward", "Defences", "EnergyShield")
						more := modDB.More(slotCfg, "Ward", "Defences")
						ward = ward + float64(wardBase*(1+inc/100)*more)
						gearWard = gearWard + wardBase
						if actor.Breakdown != nil {
							actor.Breakdown.AddSlot("Ward", BSlot{
								Base:   wardBase,
								Inc:    utils.Ternary(inc != 0, fmt.Sprintf(" x %.2f", 1+inc/100), ""),
								More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
								Total:  fmt.Sprintf("%.2f", wardBase*(1+inc/100)*more),
								Source: slot,
								Item:   actor.ItemList[slot],
							})
						}
					} else {
						ward = ward + float64(wardBase*calclib.Mod(modDB, slotCfg, "Ward", "Defences"))
						gearWard = gearWard + wardBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, wardBase, nil, "Ward", "Defences")
						}
					}
				}
				if energyShieldBase > 0 {
					if modDB.Flag(nil, "EnergyShieldToWard") {
						more := modDB.More(slotCfg, "EnergyShield", "Defences")
						energyShield = energyShield + float64(energyShieldBase*more)
						gearEnergyShield = gearEnergyShield + energyShieldBase
						if actor.Breakdown != nil {
							actor.Breakdown.AddSlot("EnergyShield", BSlot{
								Base:   energyShieldBase,
								More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
								Total:  fmt.Sprintf("%.2f", energyShieldBase*more),
								Source: slot,
								Item:   actor.ItemList[slot],
							})
						}
					} else if !modDB.Flag(nil, "ConvertArmourESToLife") {
						energyShield = energyShield + float64(energyShieldBase*calclib.Mod(modDB, slotCfg, "EnergyShield", "Defences", slot+"ESAndArmour"))
						gearEnergyShield = gearEnergyShield + energyShieldBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, energyShieldBase, nil, "EnergyShield", "Defences")
						}
					}
				}
				if armourBase > 0 {
					if slot == "Body Armour" && modDB.Flag(nil, "Unbreakable") {
						armourBase = float64(armourBase * 2)
					}
					armour = armour + float64(armourBase*calclib.Mod(modDB, slotCfg, "Armour", "ArmourAndEvasion", "Defences", slot+"ESAndArmour"))
					gearArmour = gearArmour + armourBase
					if actor.Breakdown != nil {
						actor.Breakdown.Slot(slot, nil, slotCfg, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
					}
				}
				if evasionBase > 0 {
					//pob:block:src/Modules/CalcDefence.lua:914-924
					gearEvasion = gearEvasion + evasionBase
					if actor.Breakdown != nil {
						actor.Breakdown.Slot(slot, nil, slotCfg, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
					}
					if ironReflexes {
						armour = armour + float64(evasionBase*calclib.Mod(modDB, slotCfg, "Armour", "Evasion", "ArmourAndEvasion", "Defences"))
					} else {
						evasion = evasion + float64(evasionBase*calclib.Mod(modDB, slotCfg, "Evasion", "ArmourAndEvasion", "Defences"))
					}
				}
			}
		}
		wardBase = modDB.Sum(mod.TypeBase, nil, "Ward")

		if wardBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				//pob:block:src/Modules/CalcDefence.lua:931-933
				inc := modDB.Sum(mod.TypeIncrease, nil, "Ward", "Defences", "EnergyShield")
				more := modDB.More(nil, "Ward", "Defences")
				ward = ward + float64(wardBase*(1+inc/100)*more)
				if actor.Breakdown != nil {
					actor.Breakdown.AddSlot("Ward", BSlot{
						Base:   wardBase,
						Inc:    utils.Ternary(inc != 0, fmt.Sprintf(" x %.2f", 1+inc/100), ""),
						More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
						Total:  fmt.Sprintf("%.2f", wardBase*(1+inc/100)*more),
						Source: "Global",
						Item:   actor.ItemList["Global"],
					})
				}
			} else {
				ward = ward + float64(wardBase*calclib.Mod(modDB, nil, "Ward", "Defences"))
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Global", nil, nil, wardBase, nil, "Ward", "Defences")
				}
			}
		}
		energyShieldBase = modDB.Sum(mod.TypeBase, nil, "EnergyShield")
		if energyShieldBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				energyShield = energyShield + float64(energyShieldBase*modDB.More(nil, "EnergyShield", "Defences"))
			} else {
				energyShield = energyShield + float64(energyShieldBase*calclib.Mod(modDB, nil, "EnergyShield", "Defences"))
			}
			if actor.Breakdown != nil {
				more := modDB.More(slotCfg, "EnergyShield", "Defences")
				actor.Breakdown.AddSlot("EnergyShield", BSlot{
					Base:   energyShieldBase,
					More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
					Total:  fmt.Sprintf("%.2f", energyShieldBase*more),
					Source: "Global",
					Item:   actor.ItemList["Global"],
				})
			}
		}
		armourBase = modDB.Sum(mod.TypeBase, nil, "Armour", "ArmourAndEvasion")
		if armourBase > 0 {
			armour = armour + float64(armourBase*calclib.Mod(modDB, nil, "Armour", "ArmourAndEvasion", "Defences"))
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Global", nil, nil, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
			}
		}
		evasionBase = modDB.Sum(mod.TypeBase, nil, "Evasion", "ArmourAndEvasion")
		if evasionBase > 0 {
			if ironReflexes {
				armour = armour + float64(evasionBase*calclib.Mod(modDB, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences"))
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Conversion", utils.Ptr("Evasion to Armour"), nil, evasionBase, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
				}
			} else {
				evasion = evasion + float64(evasionBase*calclib.Mod(modDB, nil, "Evasion", "ArmourAndEvasion", "Defences"))
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Global", nil, nil, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
				}
			}
		}
		convManaToArmour := modDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
		if convManaToArmour > 0 {
			armourBase = 2 * modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToArmour / 100
			total := float64(armourBase * calclib.Mod(modDB, nil, "Mana", "Armour", "ArmourAndEvasion", "Defences"))
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Mana to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Mana")
			}
		}
		convManaToES := modDB.Sum(mod.TypeBase, nil, "ManaGainAsEnergyShield")
		if convManaToES > 0 {
			energyShieldBase = modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToES / 100
			energyShield = energyShield + float64(energyShieldBase*calclib.Mod(modDB, nil, "Mana", "EnergyShield", "Defences"))
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Mana to Energy Shield"), nil, energyShieldBase, nil, "EnergyShield", "Defences", "Mana")
			}
		}
		convLifeToArmour := modDB.Sum(mod.TypeBase, nil, "LifeGainAsArmour")
		if convLifeToArmour > 0 {
			armourBase = modDB.Sum(mod.TypeBase, nil, "Life") * convLifeToArmour / 100
			total := float64(0)
			if modDB.Flag(nil, "ChaosInoculation") {
				total = 1
			} else {
				total = armourBase * calclib.Mod(modDB, nil, "Life", "Armour", "ArmourAndEvasion", "Defences")
			}
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Life to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Life")
			}
		}
		convLifeToES := modDB.Sum(mod.TypeBase, nil, "LifeConvertToEnergyShield", "LifeGainAsEnergyShield")
		if convLifeToES > 0 {
			energyShieldBase = modDB.Sum(mod.TypeBase, nil, "Life") * convLifeToES / 100
			total := float64(0)
			if modDB.Flag(nil, "ChaosInoculation") {
				total = 1
			} else {
				total = energyShieldBase * calclib.Mod(modDB, nil, "Life", "EnergyShield", "Defences")
			}
			energyShield = energyShield + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Life to Energy Shield"), nil, energyShieldBase, utils.Ptr(total), "EnergyShield", "Defences", "Life")
			}
		}
		convEvasionToArmour := modDB.Sum(mod.TypeBase, nil, "EvasionGainAsArmour")
		if convEvasionToArmour > 0 {
			armourBase = (modDB.Sum(mod.TypeBase, nil, "Evasion", "ArmourAndEvasion") + gearEvasion) * convEvasionToArmour / 100
			total := float64(armourBase * calclib.Mod(modDB, nil, "Evasion", "Armour", "ArmourAndEvasion", "Defences"))
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Evasion to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Evasion")
			}
		}
		actor.Output["EnergyShield"] = utils.Or(modDB.Override(nil, "EnergyShield"), max(utils.RoundTo(energyShield, 0), 0))
		actor.Output["Armour"] = max(utils.RoundTo(armour, 0), 0)
		actor.Output["Evasion"] = max(utils.RoundTo(evasion, 0), 0)
		actor.Output["MeleeEvasion"] = max(utils.RoundTo(evasion*calclib.Mod(modDB, nil, "MeleeEvasion"), 0), 0)
		actor.Output["ProjectileEvasion"] = max(utils.RoundTo(evasion*calclib.Mod(modDB, nil, "ProjectileEvasion"), 0), 0)
		actor.Output["LowestOfArmourAndEvasion"] = min(actor.Output["Armour"], actor.Output["Evasion"])
		actor.Output["Ward"] = max(math.Floor(ward), 0)
		actor.Output["Gear:Ward"] = gearWard
		actor.Output["Gear:EnergyShield"] = gearEnergyShield
		actor.Output["Gear:Armour"] = gearArmour
		actor.Output["Gear:Evasion"] = gearEvasion

		ArmourESRecoveryCap := modDB.Flag(nil, "ArmourESRecoveryCap")
		EvasionESRecoveryCap := modDB.Flag(nil, "EvasionESRecoveryCap")
		CappingES := (ArmourESRecoveryCap && actor.Output["Armour"] < actor.Output["EnergyShield"]) ||
			(EvasionESRecoveryCap && actor.Output["Evasion"] < actor.Output["EnergyShield"]) ||
			environment.Build.GetBooleanOption("conditionLowEnergyShield")

		actor.Output["CappingES"] = utils.Ternary[float64](CappingES, 1, 0)

		if CappingES {
			actor.Output["EnergyShieldRecoveryCap"] = utils.Ternary(ArmourESRecoveryCap && EvasionESRecoveryCap, min(actor.Output["Armour"], actor.Output["Evasion"]), utils.Ternary(ArmourESRecoveryCap, actor.Output["Armour"], utils.Ternary(EvasionESRecoveryCap, actor.Output["Evasion"], actor.Output["EnergyShield"])))
			actor.Output["EnergyShieldRecoveryCap"] = utils.Ternary(environment.Build.GetBooleanOption("conditionLowEnergyShield"), min(actor.Output["EnergyShield"]*data.LowPoolThreshold, actor.Output["EnergyShieldRecoveryCap"]), actor.Output["EnergyShieldRecoveryCap"])
		} else {
			actor.Output["EnergyShieldRecoveryCap"] = actor.Output["EnergyShield"]
		}

		//pob:block:src/Modules/CalcDefence.lua:1066-1074
		if modDB.Flag(nil, "CannotEvade") || actor.Enemy.ModDB.Flag(nil, "CannotBeEvaded") {
			actor.Output["EvadeChance"] = 0
			actor.Output["MeleeEvadeChance"] = 0
			actor.Output["ProjectileEvadeChance"] = 0
		} else if modDB.Flag(nil, "AlwaysEvade") {
			actor.Output["EvadeChance"] = 100
			actor.Output["MeleeEvadeChance"] = 100
			actor.Output["ProjectileEvadeChance"] = 100
		} else {
			enemyAccuracy := utils.RoundTo(calclib.Val(actor.Enemy.ModDB, "Accuracy"), 0)
			evadeChance := modDB.Sum(mod.TypeBase, nil, "EvadeChance")
			hitChance := calclib.Mod(actor.Enemy.ModDB, nil, "HitChance")
			//pob:block:src/Modules/CalcDefence.lua:1078-1086
			evadeStat := actor.Output["Evasion"]
			meleeEvadeStat := actor.Output["MeleeEvasion"]
			projectileEvadeStat := actor.Output["ProjectileEvasion"]
			if modDB.Flag(nil, "EvadeChanceBasedOnWard") {
				multiplier := float64(0)
				if o := modDB.Override(nil, "EvadeChanceBasedOnWardPercent"); o != nil {
					multiplier = o.Float() / 100
				}
				evadeStat = actor.Output["Ward"] * multiplier
				meleeEvadeStat = evadeStat
				projectileEvadeStat = evadeStat
			}
			actor.Output["EvadeChance"] = 100 - float64((CalcHitChance(evadeStat, enemyAccuracy)-evadeChance)*hitChance)
			actor.Output["MeleeEvadeChance"] = max(0, min(data.EvadeChanceCap, (100-float64((CalcHitChance(meleeEvadeStat, enemyAccuracy)-evadeChance)*hitChance))*calclib.Mod(modDB, nil, "EvadeChance", "MeleeEvadeChance")))
			actor.Output["ProjectileEvadeChance"] = max(0, min(data.EvadeChanceCap, (100-float64((CalcHitChance(projectileEvadeStat, enemyAccuracy)-evadeChance)*hitChance))*calclib.Mod(modDB, nil, "EvadeChance", "ProjectileEvadeChance")))
			if actor.Output["MeleeEvadeChance"] != actor.Output["ProjectileEvadeChance"] {
				actor.Output["splitEvade"] = 1
			} else {
				actor.Output["EvadeChance"] = actor.Output["MeleeEvadeChance"]
				actor.Output["dontSplitEvade"] = 1
			}
			//pob:block:src/Modules/CalcDefence.lua:1097-1115
			if actor.Breakdown != nil {
				enemyLevel := fmt.Sprintf("Enemy level: %d ^8(%s the Configuration tab)", environment.EnemyLevel, utils.Ternary(environment.Build.GetNumberOption("enemyLevel") != 0, "overridden from", "can be overridden in"))
				actor.Breakdown.Data["EvadeChance"] = &Entry{Lines: []string{
					enemyLevel,
					fmt.Sprintf("Average enemy accuracy: %d", int(enemyAccuracy)),
					fmt.Sprintf("Approximate evade chance: %d%%", int(actor.Output["EvadeChance"])),
				}}
				actor.Breakdown.Data["MeleeEvadeChance"] = &Entry{Lines: []string{
					enemyLevel,
					fmt.Sprintf("Average enemy accuracy: %d", int(enemyAccuracy)),
					fmt.Sprintf("Effective Evasion: %d", int(meleeEvadeStat)),
					fmt.Sprintf("Approximate melee evade chance: %d%%", int(actor.Output["MeleeEvadeChance"])),
				}}
				actor.Breakdown.Data["ProjectileEvadeChance"] = &Entry{Lines: []string{
					enemyLevel,
					fmt.Sprintf("Average enemy accuracy: %d", int(enemyAccuracy)),
					fmt.Sprintf("Effective Evasion: %d", int(projectileEvadeStat)),
					fmt.Sprintf("Approximate projectile evade chance: %d%%", int(actor.Output["ProjectileEvadeChance"])),
				}}
			}
		}
	}

	// Acrobatics Spell Suppression to Spell Dodge Chance conversion.
	if modDB.Flag(nil, "ConvertSpellSuppressionToSpellDodge") {
		SpellSuppressionChance := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance")
		modDB.AddMod(mod.NewFloat("SpellDodgeChance", mod.TypeBase, SpellSuppressionChance/2).Source("Acrobatics"))
	}

	totalSpellSuppressionChance := utils.Or(modDB.Override(nil, "SpellSuppressionChance"), modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance"))

	actor.Output["SpellSuppressionChance"] = min(totalSpellSuppressionChance, data.SuppressionChanceCap)
	actor.Output["SpellSuppressionEffect"] = max(data.SuppressionEffect+modDB.Sum(mod.TypeBase, nil, "SpellSuppressionEffect"), 0)

	actor.Output["EffectiveSpellSuppressionChance"] = actor.Output["SpellSuppressionChance"]
	if actor.Enemy.ModDB.Flag(nil, "CannotBeSuppressed") {
		actor.Output["EffectiveSpellSuppressionChance"] = 0
	}
	suppressRolls := float64(0)
	if environment.ModeEffective {
		if modDB.Flag(nil, "SpellSuppressionChanceIsLucky") {
			suppressRolls++
		}
		if modDB.Flag(nil, "SpellSuppressionChanceIsUnlucky") {
			suppressRolls--
		}
		if modDB.Flag(nil, "ExtremeLuck") {
			suppressRolls *= 2
		}
	}
	if worstOf, ok := environment.Build.GetNumberOptionOK("EHPUnluckyWorstOf"); ok && worstOf != 1 {
		suppressRolls = -worstOf / 2
	}
	if suppressRolls != 0 {
		suppressChance := actor.Output["EffectiveSpellSuppressionChance"] / 100
		switch {
		case modDB.Flag(nil, "Unexciting"):
			actor.Output["EffectiveSpellSuppressionChance"] = (float64(3*suppressChance*suppressChance) - float64(2*suppressChance*suppressChance*suppressChance)) * 100
		case suppressRolls > 0:
			actor.Output["EffectiveSpellSuppressionChance"] = (1 - math.Pow(1-suppressChance, suppressRolls+1)) * 100
		default:
			actor.Output["EffectiveSpellSuppressionChance"] = math.Pow(suppressChance, math.Abs(suppressRolls)) * actor.Output["EffectiveSpellSuppressionChance"]
		}
	}

	actor.Output["SpellSuppressionChanceOverCap"] = max(0, totalSpellSuppressionChance-data.SuppressionChanceCap)

	baseDodgeChance := 0
	totalAttackDodgeChance := modDB.Sum(mod.TypeBase, nil, "AttackDodgeChance")
	totalSpellDodgeChance := modDB.Sum(mod.TypeBase, nil, "SpellDodgeChance")
	attackDodgeChanceMax := data.DodgeChanceCap
	spellDodgeChanceMax := utils.Or(modDB.Override(nil, "SpellDodgeChanceMax"), modDB.Sum(mod.TypeBase, nil, "SpellDodgeChanceMax"))

	//pob:block:src/Modules/CalcDefence.lua:1169-1176
	enemyReduceDodgeChance := actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "reduceEnemyDodge")
	cannotBeDodged := actor.Enemy.ModDB.Flag(nil, "CannotBeDodged")

	actor.Output["AttackDodgeChance"] = min(totalAttackDodgeChance, float64(attackDodgeChanceMax))
	actor.Output["EffectiveAttackDodgeChance"] = utils.Ternary(cannotBeDodged, float64(0),
		min(max(totalAttackDodgeChance-enemyReduceDodgeChance, 0), float64(attackDodgeChanceMax)))
	actor.Output["SpellDodgeChance"] = min(totalSpellDodgeChance, spellDodgeChanceMax)
	actor.Output["EffectiveSpellDodgeChance"] = utils.Ternary(cannotBeDodged, float64(0),
		min(max(totalSpellDodgeChance-enemyReduceDodgeChance, 0), spellDodgeChanceMax))
	if environment.ModeEffective && modDB.Flag(nil, "DodgeChanceIsUnlucky") {
		actor.Output["EffectiveAttackDodgeChance"] = actor.Output["EffectiveAttackDodgeChance"] / 100 * actor.Output["EffectiveAttackDodgeChance"]
		actor.Output["EffectiveSpellDodgeChance"] = actor.Output["EffectiveSpellDodgeChance"] / 100 * actor.Output["EffectiveSpellDodgeChance"]
	}
	actor.Output["AttackDodgeChanceOverCap"] = max(0, totalAttackDodgeChance-float64(attackDodgeChanceMax))
	actor.Output["SpellDodgeChanceOverCap"] = max(0, totalSpellDodgeChance-spellDodgeChanceMax)

	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("AttackDodgeChance",
			"Base: "+fmt.Sprint(baseDodgeChance)+"%",
			"Max: "+fmt.Sprint(attackDodgeChanceMax)+"%",
			"Total: "+fmt.Sprint(actor.Output["AttackDodgeChance"]+actor.Output["AttackDodgeChanceOverCap"])+"%",
		)
		actor.Breakdown.AddLine("SpellDodgeChance",
			"Base: "+fmt.Sprint(baseDodgeChance)+"%",
			"Max: "+fmt.Sprint(spellDodgeChanceMax)+"%",
			"Total: "+fmt.Sprint(actor.Output["SpellDodgeChance"]+actor.Output["SpellDodgeChanceOverCap"])+"%",
		)
	}

	// Recovery modifiers
	//pob:src/Modules/CalcDefence.lua:1194-1197
	actor.Output["LifeRecoveryRateMod"] = 1
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		actor.Output["LifeRecoveryRateMod"] = calclib.Mod(modDB, nil, "LifeRecoveryRate")
	}
	actor.Output["ManaRecoveryRateMod"] = calclib.Mod(modDB, nil, "ManaRecoveryRate")
	actor.Output["EnergyShieldRecoveryRateMod"] = calclib.Mod(modDB, nil, "EnergyShieldRecoveryRate")

	actor.Output["MaxLifeLeechInstance"] = actor.Output["Life"] * calclib.Val(modDB, "MaxLifeLeechInstance") / 100
	actor.Output["MaxLifeLeechRatePercent"] = calclib.Val(modDB, "MaxLifeLeechRate")
	actor.Output["MaxLifeLeechRate"] = actor.Output["Life"] * actor.Output["MaxLifeLeechRatePercent"] / 100
	//pob:block:src/Modules/CalcDefence.lua:1210-1216
	if actor.Breakdown != nil {
		actor.Breakdown.Data["MaxLifeLeechRate"] = &Entry{Lines: []string{
			fmt.Sprintf("%d ^8(maximum life)", int(actor.Output["Life"])),
			fmt.Sprintf("x %d%% ^8(percentage of life to maximum leech rate)", int(actor.Output["MaxLifeLeechRatePercent"])),
			fmt.Sprintf("= %.1f", actor.Output["MaxLifeLeechRate"]),
		}}
	}
	actor.Output["MaxEnergyShieldLeechInstance"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechInstance") / 100
	actor.Output["MaxEnergyShieldLeechRate"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechRate") / 100
	//pob:block:src/Modules/CalcDefence.lua:1219-1225
	if actor.Breakdown != nil {
		actor.Breakdown.Data["MaxEnergyShieldLeechRate"] = &Entry{Lines: []string{
			fmt.Sprintf("%d ^8(maximum energy shield)", int(actor.Output["EnergyShield"])),
			fmt.Sprintf("x %d%% ^8(percentage of energy shield to maximum leech rate)", int(calclib.Val(modDB, "MaxEnergyShieldLeechRate"))),
			fmt.Sprintf("= %.1f", actor.Output["MaxEnergyShieldLeechRate"]),
		}}
	}
	actor.Output["MaxManaLeechInstance"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechInstance") / 100
	actor.Output["MaxManaLeechRate"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechRate") / 100
	//pob:block:src/Modules/CalcDefence.lua:1228-1234
	if actor.Breakdown != nil {
		actor.Breakdown.Data["MaxManaLeechRate"] = &Entry{Lines: []string{
			fmt.Sprintf("%d ^8(maximum mana)", int(actor.Output["Mana"])),
			fmt.Sprintf("x %d%% ^8(percentage of mana to maximum leech rate)", int(calclib.Val(modDB, "MaxManaLeechRate"))),
			fmt.Sprintf("= %.1f", actor.Output["MaxManaLeechRate"]),
		}}
	}

	// Regeneration
	//pob:block:src/Modules/CalcDefence.lua:628-705
	regenResources := []struct {
		name    string
		display string
	}{
		{"Mana", "Mana"},
		{"Life", "Life"},
		{"EnergyShield", "Energy Shield"},
		{"Rage", "Rage"},
	}
	for i, resource := range regenResources {
		name := resource.name
		n := &regenNames[i]
		pool := actor.Output[name]
		baseRegen := float64(0)
		inc := modDB.Sum(mod.TypeIncrease, nil, n.regen)
		more := modDB.More(nil, n.regen)
		regen := float64(0)
		regenRate := float64(0)
		recoveryRateMod := utils.OrF(actor.Output[n.recoveryRateMod], 1)

		switch {
		case modDB.Flag(nil, n.noRegen) || modDB.Flag(nil, n.cannotGain):
			actor.Output[n.regen] = 0
		case name == "Life" && modDB.Flag(nil, "ZealotsOath"):
			actor.Output["LifeRegen"] = 0
			lifeBase := modDB.Sum(mod.TypeBase, nil, "LifeRegen")
			if lifeBase > 0 {
				modDB.AddMod(mod.NewFloat("EnergyShieldRegen", mod.TypeBase, lifeBase).Source("Zealot's Oath"))
			}
			lifePercent := modDB.Sum(mod.TypeBase, nil, "LifeRegenPercent")
			if lifePercent > 0 {
				modDB.AddMod(mod.NewFloat("EnergyShieldRegenPercent", mod.TypeBase, lifePercent).Source("Zealot's Oath"))
			}
		default:
			if inc != 0 {
				for j := i + 1; j < len(regenResources); j++ {
					if modDB.Flag(nil, name+"RegenTo"+regenResources[j].name+"Regen") {
						modDB.AddMod(mod.NewFloat(regenResources[j].name+"Regen", mod.TypeIncrease, inc).
							Source(mod.Source(resource.display + " instead applies to " + regenResources[j].display)))
						inc = 0
					}
				}
			}
			baseRegen = modDB.Sum(mod.TypeBase, nil, n.regen) + pool*modDB.Sum(mod.TypeBase, nil, n.regenPercent)/100
			regen = baseRegen * (1 + inc/100) * more
			if regen != 0 {
				for j := i + 1; j < len(regenResources); j++ {
					if modDB.Flag(nil, name+"RegenerationRecovers"+regenResources[j].name) {
						modDB.AddMod(mod.NewFloat(regenResources[j].name+"Recovery", mod.TypeBase, regen).
							Source(mod.Source(resource.display + " Regeneration Recovers " + regenResources[j].display)))
					}
				}
			}
			regenRate = utils.RoundTo(regen*recoveryRateMod, 1)
			actor.Output[n.regen] = regenRate
		}

		actor.Output[n.regenInc] = inc
		baseDegen := modDB.Sum(mod.TypeBase, nil, n.degen) + pool*modDB.Sum(mod.TypeBase, nil, n.degenPercent)/100
		tinctureDegenPercent := modDB.Sum(mod.TypeBase, nil, n.degenPercentTincture)
		baseDegen += math.Max(pool*tinctureDegenPercent/100, tinctureDegenPercent)
		degenRate := float64(0)
		if baseDegen > 0 {
			degenRate = float64(baseDegen * calclib.Mod(modDB, nil, n.degen))
		}
		actor.Output[n.degen] = degenRate
		recoveryRate := float64(modDB.Sum(mod.TypeBase, nil, n.recovery) * recoveryRateMod)
		actor.Output[n.recovery] = recoveryRate

		unaffectedRegen := regenRate
		if modDB.Flag(nil, n.unaffectedBy) {
			unaffectedRegen = 0
		}
		actor.Output[n.regenRecovery] = unaffectedRegen - degenRate + recoveryRate
		if actor.Output[n.regenRecovery] > 0 {
			modDB.AddMod(mod.NewFlag(n.canGain, true).Source(mod.Source(resource.display + "Regen")))
		}
		//	output[resource.."RegenPercent"] = pool > 0 and round(... / pool * 100, 1) or 0
		//pob:block:src/Modules/CalcDefence.lua:1299
		if pool > 0 {
			actor.Output[n.regenPercent] = utils.RoundTo(actor.Output[n.regenRecovery]/pool*100, 1)
		} else {
			actor.Output[n.regenPercent] = 0
		}

		if actor.Breakdown != nil {
			actor.Breakdown.MultiChain(n.regenRecovery, BMultiChain{
				Label: resource.display + " Regeneration:",
				Base:  &BMultiChainItem{"%.1f ^8(base)", baseRegen},
				Total: fmt.Sprintf("= %.1f ^8per second", regenRate),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced)", 1 + inc/100},
					{"%.2f ^8(more/less)", more},
					{"%.2f ^8(recovery rate modifier)", recoveryRateMod},
				},
			})
			if modDB.Flag(nil, n.unaffectedBy) {
				actor.Breakdown.AddLine(n.regenRecovery, "Unaffected by "+resource.display+" Regen")
			}
			if degenRate != 0 {
				actor.Breakdown.AddLine(n.regenRecovery,
					fmt.Sprintf("- %.1f ^8(degen)", degenRate),
					fmt.Sprintf("= %.1f ^8per second", unaffectedRegen-degenRate),
				)
			}
			if recoveryRate != 0 {
				actor.Breakdown.AddLine(n.regenRecovery,
					fmt.Sprintf("+ %.1f ^8(recovery)", recoveryRate),
					fmt.Sprintf("= %.1f ^8per second", actor.Output[n.regenRecovery]),
				)
			}
		}
	}

	// Energy Shield Recharge
	//pob:block:src/Modules/CalcDefence.lua:1324-1383
	appliesToLife := modDB.Flag(nil, "EnergyShieldRechargeAppliesToLife") && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech")
	appliesToEnergyShield := !(modDB.Flag(nil, "NoEnergyShieldRecharge") || modDB.Flag(nil, "CannotGainEnergyShield") || appliesToLife)
	if appliesToLife {
		actor.Output["EnergyShieldRechargeAppliesToLife"] = 1
	}
	if appliesToEnergyShield {
		actor.Output["EnergyShieldRechargeAppliesToEnergyShield"] = 1
	}

	if appliesToLife || appliesToEnergyShield {
		inc := modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRecharge")
		more := modDB.More(nil, "EnergyShieldRecharge")
		base := data.EnergyShieldRechargeBase
		if override := modDB.Override(nil, "EnergyShieldRecharge"); override != nil {
			base = override.Float()
		}
		if appliesToLife {
			recharge := actor.Output["Life"] * base * (1 + inc/100) * more
			actor.Output["LifeRecharge"] = utils.RoundTo(recharge*actor.Output["LifeRecoveryRateMod"], 0)
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain("LifeRecharge", BMultiChain{
					Label: "Recharge rate:",
					Base:  &BMultiChainItem{fmt.Sprintf("%%.1f ^8(%d%%%% per second)", int(base*100)), actor.Output["Life"] * base},
					Total: fmt.Sprintf("= %.1f ^8per second", recharge),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
					},
				})
				actor.Breakdown.MultiChain("LifeRecharge", BMultiChain{
					Label: "Effective Recharge rate:",
					Base:  &BMultiChainItem{"%.1f", recharge},
					Total: fmt.Sprintf("= %.1f ^8per second", actor.Output["LifeRecharge"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(recovery rate modifier)", actor.Output["LifeRecoveryRateMod"]},
					},
				})
			}
		} else {
			recharge := actor.Output["EnergyShield"] * base * (1 + inc/100) * more
			actor.Output["EnergyShieldRecharge"] = utils.RoundTo(recharge*actor.Output["EnergyShieldRecoveryRateMod"], 0)
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain("EnergyShieldRecharge", BMultiChain{
					Label: "Recharge rate:",
					Base:  &BMultiChainItem{fmt.Sprintf("%%.1f ^8(%d%%%% per second)", int(base*100)), actor.Output["EnergyShield"] * base},
					Total: fmt.Sprintf("= %.1f ^8per second", recharge),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
					},
				})
				actor.Breakdown.MultiChain("EnergyShieldRecharge", BMultiChain{
					Label: "Effective Recharge rate:",
					Base:  &BMultiChainItem{"%.1f", recharge},
					Total: fmt.Sprintf("= %.1f ^8per second", actor.Output["EnergyShieldRecharge"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(recovery rate modifier)", actor.Output["EnergyShieldRecoveryRateMod"]},
					},
				})
			}
		}
		actor.Output["EnergyShieldRechargeDelay"] = data.EnergyShieldRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100)
		if actor.Breakdown != nil {
			//pob:block:src/Modules/CalcDefence.lua:1373-1379
			if actor.Output["EnergyShieldRechargeDelay"] != data.EnergyShieldRechargeDelay {
				actor.Breakdown.Data["EnergyShieldRechargeDelay"] = &Entry{Lines: []string{
					fmt.Sprintf("%.2fs ^8(base)", float64(data.EnergyShieldRechargeDelay)),
					fmt.Sprintf("/ %.2f ^8(faster start)", 1+modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100),
					fmt.Sprintf("= %.2fs", actor.Output["EnergyShieldRechargeDelay"]),
				}}
			}
		}
	} else {
		actor.Output["EnergyShieldRecharge"] = 0
	}

	{
		actor.Output["anyRecoup"] = 0
		recoupTypeList := []string{"Life", "Mana", "EnergyShield"}
		//pob:block:src/Modules/CalcDefence.lua:1398-1408
		recoupBreakdown := func(key string, base float64, recoupType string) {
			if actor.Breakdown == nil {
				return
			}
			seconds := 4
			if modDB.Flag(nil, "3Second"+recoupType+"Recoup") || modDB.Flag(nil, "3SecondRecoup") {
				seconds = 3
			}
			if actor.Output[recoupType+"RecoveryRateMod"] != 1 {
				actor.Breakdown.Data[key] = &Entry{Lines: []string{
					fmt.Sprintf("%d%% ^8(base)", int(base)),
					fmt.Sprintf("* %.2f ^8(recovery rate modifier)", actor.Output[recoupType+"RecoveryRateMod"]),
					fmt.Sprintf("= %.1f%% over %d seconds", actor.Output[key], seconds),
				}}
			} else {
				actor.Breakdown.Data[key] = &Entry{Lines: []string{fmt.Sprintf("%d%% over %d seconds", int(actor.Output[key]), seconds)}}
			}
		}
		for _, recoupType := range recoupTypeList {
			baseRecoup := modDB.Sum(mod.TypeBase, nil, recoupType+"Recoup")
			if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
				actor.Output["LifeRecoup"] = 0
				lifeRecoup := modDB.Sum(mod.TypeBase, nil, "LifeRecoup")
				modDB.AddMod(mod.NewFloat("EnergyShieldRecoup", mod.TypeBase, lifeRecoup).Source("Life Recoup Conversion"))
			} else {
				actor.Output[recoupType+"Recoup"] = baseRecoup * actor.Output[recoupType+"RecoveryRateMod"]
				actor.Output["anyRecoup"] += actor.Output[recoupType+"Recoup"]
				recoupBreakdown(recoupType+"Recoup", baseRecoup, recoupType)
			}
			addToEnergyShieldFlag := "Add" + recoupType + "RecoupToEnergyShieldRecoup"
			if modDB.Flag(nil, addToEnergyShieldFlag) {
				if tab := modDB.Tabulate(mod.TypeFlag, nil, addToEnergyShieldFlag); len(tab) > 0 {
					modDB.ReplaceMod(mod.NewFloat("EnergyShieldRecoup", mod.TypeBase, baseRecoup).Source(tab[0].Mod.GetSource()))
				}
			}
		}

		if modDB.Flag(nil, "UsePowerCharges") && modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
			perAbsorption := modDB.Sum(mod.TypeBase, nil, "PerAbsorptionElementalEnergyShieldRecoup")
			for _, damageType := range []string{"Cold", "Fire", "Lightning"} {
				modDB.AddMod(mod.NewFloat(damageType+"EnergyShieldRecoup", mod.TypeBase, perAbsorption).
					Source("Absorption Charges").Tag(mod.Multiplier("AbsorptionCharge")))
			}
		}

		for ri, recoupType := range recoupTypeList {
			for di, dt := range data.DamageType("").Values() {
				damageType := string(dt)
				names := &damageRecoupNames[ri][di]
				recoup := modDB.Sum(mod.TypeBase, nil, names.recoup)
				if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
					actor.Output[damageType+"LifeRecoup"] = 0
					modDB.AddMod(mod.NewFloat(names.energyShield, mod.TypeBase, recoup).Source("Life Recoup Conversion"))
				} else {
					actor.Output[names.recoup] = recoup * actor.Output[recoupType+"RecoveryRateMod"]
					actor.Output["anyRecoup"] += actor.Output[names.recoup]
					//pob:block:src/Modules/CalcDefence.lua:1433-1443
					recoupBreakdown(names.recoup, recoup, recoupType)
				}
				addToEnergyShieldFlag := names.addToEnergyShield
				if modDB.Flag(nil, addToEnergyShieldFlag) {
					if tab := modDB.Tabulate(mod.TypeFlag, nil, addToEnergyShieldFlag); len(tab) > 0 {
						modDB.ReplaceMod(mod.NewFloat(names.energyShield, mod.TypeBase, recoup).Source(tab[0].Mod.GetSource()))
					}
				}
			}
		}

		for _, resource := range recoupTypeList {
			if modDB.Flag(nil, "No"+resource+"Regen") || modDB.Flag(nil, "CannotGain"+resource) {
				continue
			}
			pseudo := modDB.Sum(mod.TypeBase, nil, "PhysicalDamageMitigated"+resource+"PseudoRecoup")
			if pseudo <= 0 {
				continue
			}
			duration := modDB.Sum(mod.TypeBase, nil, "PhysicalDamageMitigated"+resource+"PseudoRecoupDuration")
			if duration == 0 {
				duration = 4
			}
			actor.Output["PhysicalDamageMitigated"+resource+"PseudoRecoupDuration"] = duration
			inc := modDB.Sum(mod.TypeIncrease, nil, resource+"Regen")
			more := modDB.More(nil, resource+"Regen")
			actor.Output["PhysicalDamageMitigated"+resource+"PseudoRecoup"] = pseudo * (1 + inc/100) * more * actor.Output[resource+"RecoveryRateMod"]
			actor.Output["anyRecoup"] += actor.Output["PhysicalDamageMitigated"+resource+"PseudoRecoup"]
			//pob:block:src/Modules/CalcDefence.lua:1465-1477
			if actor.Breakdown != nil {
				key := "PhysicalDamageMitigated" + resource + "PseudoRecoup"
				actor.Breakdown.Data[key] = &Entry{}
				if actor.Output[resource+"RecoveryRateMod"] != 1 || inc != 0 || more != 1 {
					actor.Breakdown.AddLine(key, fmt.Sprintf("%d%% ^8(base)", int(pseudo)))
					if inc != 0 || more != 1 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("* %.2f ^8(regeneration rate modifier)", (1+inc/100)*more))
					}
					if actor.Output[resource+"RecoveryRateMod"] != 1 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("* %.2f ^8(recovery rate modifier)", actor.Output[resource+"RecoveryRateMod"]))
					}
				}
				actor.Breakdown.AddLine(key, fmt.Sprintf("= %.1f%% over %d seconds", actor.Output[key], int(duration)))
			}
		}
	}

	// Ward recharge
	actor.Output["WardRechargeDelay"] = data.WardRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster")/100)
	if actor.Breakdown != nil {
		//pob:block:src/Modules/CalcDefence.lua:1486-1492
		if actor.Output["WardRechargeDelay"] != data.WardRechargeDelay {
			actor.Breakdown.Data["WardRechargeDelay"] = &Entry{Lines: []string{
				fmt.Sprintf("%.2fs ^8(base)", float64(data.WardRechargeDelay)),
				fmt.Sprintf("/ %.2f ^8(faster start)", 1+modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster")/100),
				fmt.Sprintf("= %.2fs", actor.Output["WardRechargeDelay"]),
			}}
		}
	}

	actor.Output["DamageReductionMax"] = data.DamageReductionCap
	DamageReductionMax := actor.ModDB.Override(nil, "DamageReductionMax")
	if DamageReductionMax != nil {
		actor.Output["DamageReductionMax"] = DamageReductionMax.Float()
	}
	modDB.AddMod(mod.NewFloat("ArmourAppliesToPhysicalDamageTaken", mod.TypeBase, 100))

	//pob:block:src/Modules/CalcDefence.lua:136-139
	//pob:block:src/Modules/CalcDefence.lua:1499-1500
	for di, damageType := range data.DamageType("").Values() {
		names := []string{damageReductionNames[di]}
		if damageType == data.DamageTypeFire || damageType == data.DamageTypeCold || damageType == data.DamageTypeLightning {
			names = append(names, "ElementalDamageReduction")
		}

		if override := actor.ModDB.Override(nil, damageReductionNames[di]); override != nil {
			actor.Output[baseDamageReductionNames[di]] = override.Float()
			actor.Output[baseDamageReductionWhenHitNames[di]] = override.Float()
			continue
		}

		actor.Output[baseDamageReductionNames[di]] = math.Min(math.Max(0, actor.ModDB.Sum(mod.TypeBase, nil, names...)), actor.Output["DamageReductionMax"])
		actor.Output[baseDamageReductionWhenHitNames[di]] = math.Min(math.Max(0, actor.Output[baseDamageReductionNames[di]]+actor.ModDB.Sum(mod.TypeBase, nil, damageReductionWhenHitNames[di])), actor.Output["DamageReductionMax"])
	}

	actor.Output["MovementSpeedMod"] = utils.Or(modDB.Override(nil, "MovementSpeed"), calclib.Mod(modDB, nil, "MovementSpeed"))
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
	}
	actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]

	actor.Output["MovementSpeedMod"] = utils.Or(modDB.Override(nil, "MovementSpeed"), calclib.Mod(modDB, nil, "MovementSpeed"))
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
	}
	actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]
	if actor.Breakdown != nil {
		actor.Breakdown.MultiChain("EffectiveMovementSpeedMod", BMultiChain{
			Total: fmt.Sprintf("= %.2f ^8(effective movement speed modifier)", actor.Output["EffectiveMovementSpeedMod"]),
			Items: []BMultiChainItem{
				{"%.2f ^8(movement speed modifier)", actor.Output["MovementSpeedMod"]},
				{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
			},
		})
	}

	if actor.Enemy.ModDB.Flag(nil, "Blind") {
		actor.Output["BlindEffectMod"] = calclib.Mod(actor.Enemy.ModDB, nil, "BlindEffect", "BuffEffectOnSelf") * 100
	}

	// recovery on block, needs to be after primary defences
	//pob:src/Modules/CalcDefence.lua:1522-1528
	actor.Output["LifeOnBlock"] = 0
	actor.Output["LifeOnSuppress"] = 0
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		actor.Output["LifeOnBlock"] = modDB.Sum(mod.TypeBase, nil, "LifeOnBlock")
		actor.Output["LifeOnSuppress"] = modDB.Sum(mod.TypeBase, nil, "LifeOnSuppress")
	}
	actor.Output["ManaOnBlock"] = modDB.Sum(mod.TypeBase, nil, "ManaOnBlock")
	actor.Output["EnergyShieldOnBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnBlock")
	actor.Output["EnergyShieldOnSpellBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnSpellBlock")

	actor.Output["EnergyShieldOnSuppress"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnSuppress")

	// damage avoidances
	specificTypeAvoidance := false
	for di := range data.DamageType("").Values() {
		actor.Output[avoidDamageChanceNames[di]] = min(modDB.Sum(mod.TypeBase, nil, avoidDamageChanceNames[di]), data.AvoidChanceCap)
		if actor.Output[avoidDamageChanceNames[di]] > 0 {
			specificTypeAvoidance = true
		}
	}
	actor.Output["AvoidProjectilesChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidProjectilesChance"), data.AvoidChanceCap)

	// hit avoidance
	actor.Output["AvoidAllDamageFromHitsChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidAllDamageFromHitsChance"), data.AvoidChanceCap)

	// other avoidances etc
	//pob:block:src/Modules/CalcDefence.lua:1548-1549
	if modDB.Flag(nil, "BlindImmune") {
		actor.Output["BlindAvoidChance"] = 100
	} else {
		actor.Output["BlindAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidBlind"), 100)
	}
	if modDB.Flag(nil, "ImpaleImmune") {
		actor.Output["ImpaleAvoidChance"] = 100
	} else {
		actor.Output["ImpaleAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidImpale"), 100)
	}
	//pob:block:src/Modules/CalcDefence.lua:1556-1560
	if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
		modDB.AddMod(mod.NewFloat("AvoidElementalAilments", mod.TypeBase,
			ancestralVisionAvoid(modDB)).Source("Ancestral Vision"))
	}

	//pob:block:src/Modules/CalcDefence.lua:1561-1567
	if modDB.Flag(nil, "SpellSuppressionAppliesToChanceToDefendWithArmour") {
		for _, m := range foulbornArmourDefenseMods(modDB) {
			modDB.AddMod(m)
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:1569-1570
	actor.Output["ArmourDefense"] = modDB.Max(nil, "ArmourDefense") / 100
	actor.Output["RawArmourDefense"] = utils.Ternary(actor.Output["ArmourDefense"] > 0, (1+actor.Output["ArmourDefense"])*100, 0)

	// This is only used for breakdown purposes
	//pob:block:src/Modules/CalcDefence.lua:1572-1578
	if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
		if base := modDB.Sum(mod.TypeBase, nil, "AvoidShock"); base != 0 {
			modDB.AddMod(mod.NewFloat("AvoidShockAppliesToElementalAilments", mod.TypeBase, base).Source("Stormshroud"))
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:1580-1586
	isElementalAilment := make(map[data.Ailment]bool, len(data.ElementalAilment("").Values()))
	for _, ailment := range data.ElementalAilment("").Values() {
		isElementalAilment[ailment] = true
	}
	for ai, ailment := range data.Ailment("").Values() {
		names := []string{avoidAilmentNames[ai], "AvoidAilments"}
		immuneFlags := []string{ailmentImmuneNames[ai]}
		shockAvoidAppliesToAll := float64(0)
		if isElementalAilment[ailment] {
			names = append(names, "AvoidElementalAilments")
			immuneFlags = append(immuneFlags, "ElementalAilmentImmune")
			if ailment != data.AilmentShock && modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
				shockAvoidAppliesToAll = modDB.Sum(mod.TypeBase, nil, "AvoidShock")
			}
		}
		if modDB.Flag(nil, immuneFlags...) {
			actor.Output[ailmentAvoidChanceNames[ai]] = 100
			continue
		}
		actor.Output[ailmentAvoidChanceNames[ai]] = math.Floor(min(modDB.Sum(mod.TypeBase, nil, names...)+shockAvoidAppliesToAll, 100))
	}
	//pob:block:src/Modules/CalcDefence.lua:1588-1589
	if modDB.Flag(nil, "CurseImmune") {
		actor.Output["CurseAvoidChance"] = 100
	} else {
		actor.Output["CurseAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidCurse"), 100)
	}
	if modDB.Flag(nil, "SilenceImmune") {
		actor.Output["SilenceAvoidChance"] = 100
	} else {
		actor.Output["SilenceAvoidChance"] = actor.Output["CurseAvoidChance"]
	}
	actor.Output["CritExtraDamageReduction"] = min(modDB.Sum(mod.TypeBase, nil, "ReduceCritExtraDamage"), 100)
	actor.Output["LightRadiusMod"] = calclib.Mod(modDB, nil, "LightRadius")
	//pob:block:src/Modules/CalcDefence.lua:1592
	actor.Output["LightRadiusInc"] = max(modDB.Sum(mod.TypeIncrease, nil, "LightRadius"), 0)
	if actor.Breakdown != nil {
		actor.Breakdown.Mod("LightRadiusMod", modDB, nil, "LightRadius")
	}
	actor.Output["CurseEffectOnSelf"] = max(modDB.More(nil, "CurseEffectOnSelf")*(100+modDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")), 0)
	//pob:block:src/Modules/CalcDefence.lua:1597-1598
	actor.Output["ExposureEffectOnSelf"] = modDB.More(nil, "ExposureEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "ExposureEffectOnSelf"))
	actor.Output["WitherEffectOnSelf"] = modDB.More(nil, "WitherEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "WitherEffectOnSelf"))

	// Ailment duration on self
	//pob:block:src/Modules/CalcDefence.lua:1600-1628
	actor.Output["DebuffExpirationRate"] = modDB.Sum(mod.TypeBase, nil, "SelfDebuffExpirationRate")
	actor.Output["DebuffExpirationModifier"] = 10000 / (100 + actor.Output["DebuffExpirationRate"])
	actor.Output["SelfBlindDuration"] = modDB.More(nil, "SelfBlindDuration") *
		(100 + modDB.Sum(mod.TypeIncrease, nil, "SelfBlindDuration")) * actor.Output["DebuffExpirationModifier"] / 100

	if modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") {
		inc := modDB.Sum(mod.TypeIncrease, nil, "SelfIgniteDuration")
		more := modDB.More(nil, "SelfIgniteDuration")
		if inc != 0 {
			modDB.AddMod(mod.NewFloat("SelfIgniteDurationToElementalAilments", mod.TypeIncrease, inc).Source("Firesong"))
		}
		if more != 1 {
			modDB.AddMod(mod.NewFloat("SelfIgniteDurationToElementalAilments", mod.TypeMore, more).Source("Firesong"))
		}
	}

	isElementalAilmentForDuration := make(map[data.Ailment]bool, len(data.ElementalAilment("").Values()))
	for _, ailment := range data.ElementalAilment("").Values() {
		isElementalAilmentForDuration[ailment] = true
	}
	igniteAppliesToAll := modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments")
	for ai, ailment := range data.Ailment("").Values() {
		selfDuration := selfAilmentDurationNames[ai]
		divisor := 100 + actor.Output["DebuffExpirationRate"] +
			modDB.Sum(mod.TypeBase, nil, string("Self"+ailment+"DebuffExpirationRate"))
		if !isElementalAilmentForDuration[ailment] {
			//pob:block:src/Modules/CalcDefence.lua:1618-1622
			more := modDB.More(nil, selfDuration, "SelfAilmentDuration")
			inc := (100 + modDB.Sum(mod.TypeIncrease, nil, selfDuration, "SelfAilmentDuration")) * 100
			actor.Output[selfDuration] = inc * more / divisor
			continue
		}
		//pob:block:src/Modules/CalcDefence.lua:1623-1628
		foldIgnite := igniteAppliesToAll && ailment != data.AilmentIgnite
		more := modDB.More(nil, selfDuration, "SelfAilmentDuration", "SelfElementalAilmentDuration")
		if foldIgnite {
			more *= modDB.More(nil, "SelfIgniteDuration")
		}
		incSum := modDB.Sum(mod.TypeIncrease, nil, selfDuration, "SelfAilmentDuration", "SelfElementalAilmentDuration")
		if foldIgnite {
			incSum += modDB.Sum(mod.TypeIncrease, nil, "SelfIgniteDuration")
		}
		actor.Output[selfDuration] = more * (100 + incSum) * 100 / divisor
	}
	//pob:block:src/Modules/CalcDefence.lua:1629-1631
	for ai, ailment := range data.Ailment("").Values() {
		selfName := selfAilmentEffectNames[ai]
		enemyName := enemyAilmentEffectNames[ai]
		self := modDB.More(nil, selfName) * (100 + modDB.Sum(mod.TypeIncrease, nil, selfName))
		enemyFactor := float64(1)
		if modDB.Flag(nil, string("Condition:"+ailment+"edSelf")) {
			enemyFactor = calclib.Mod(modDB, nil, enemyName)
		} else if actor.Enemy != nil {
			enemyFactor = calclib.Mod(actor.Enemy.ModDB, nil, enemyName)
		}
		actor.Output[selfName] = self * enemyFactor
	}

	//pob:block:src/Modules/CalcPerform.lua:3722-3724, src/Modules/CalcPerform.lua:3733-3735
	if environment.skipEHP {
		return
	}

	{
		actor.Output["totalEnemyDamage"] = 0
		actor.Output["totalEnemyDamageIn"] = 0
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("totalEnemyDamage", "Total damage from the enemy")
			actor.Breakdown.AddCol("totalEnemyDamage",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Value", Key: "value"},
				BCol{Label: "Mult", Key: "mult"},
				BCol{Label: "Crit", Key: "crit"},
				BCol{Label: "Final", Key: "final"},
				BCol{Label: "From", Key: "from"},
			)
		}
		//	local damageCategoryConfig = env.configInput.enemyDamageType or "Average"
		//	if damageCategoryConfig ~= "DamageOverTime" then
		//	    output.ConfiguredEvadeChance = output[damageCategoryConfig.."EvadeChance"] or 0
		//	    if worstOf > 1 then ... square it, twice when worstOf == 4 ... end
		//	end
		damageCategoryConfig := enemyDamageCategory(environment)
		//pob:block:src/Modules/CalcDefence.lua:1643-1666
		{
			worstOf := environment.Build.GetNumberOption("EHPUnluckyWorstOf")
			if worstOf == 0 {
				worstOf = 1
			}
			if damageCategoryConfig != "DamageOverTime" {
				actor.Output["AverageEvadeChance"] = (actor.Output["MeleeEvadeChance"] + actor.Output["ProjectileEvadeChance"]) / 4
				actor.Output["ConfiguredEvadeChance"] = actor.Output[damageCategoryConfig+"EvadeChance"]
				if worstOf > 1 {
					actor.Output["ConfiguredEvadeChance"] = actor.Output["ConfiguredEvadeChance"] / 100 * actor.Output["ConfiguredEvadeChance"]
					if worstOf == 4 {
						actor.Output["ConfiguredEvadeChance"] = actor.Output["ConfiguredEvadeChance"] / 100 * actor.Output["ConfiguredEvadeChance"]
					}
				}
			}
		}
		//pob:block:src/Modules/CalcDefence.lua:1671-1677
		if damageCategoryConfig == "DamageOverTime" {
			for _, damageType := range data.DamageType("").Values() {
				actor.Output[string(damageType+"EnemyPen")] = 0
				actor.Output[string(damageType+"EnemyDamageMult")] = calclib.Mod(actor.Enemy.ModDB, nil, "Damage", string(damageType+"Damage"), utils.Ternary(isElemental[string(damageType)], "ElementalDamage", "")) // missing taunt from allies
				actor.Output[string(damageType+"EnemyOverwhelm")] = 0
				actor.Output[string(damageType+"EnemyDamage")] = 0
			}
		} else {
			//	local enemyCritChance = enemyDB:Flag(nil, "NeverCrit") and 0 or enemyDB:Flag(nil, "AlwaysCrit") and 100
			//	  or m_max(m_min((modDB:Override(nil, "enemyCritChance") or env.configInput["enemyCritChance"]
			//	    * (1 + modDB:Sum("INC", nil, "EnemyCritChance") / 100 + enemyDB:Sum("INC", nil, "CritChance") / 100)
			//pob:block:src/Modules/CalcDefence.lua:1694-1697
			enemyCritChance := 0.0
			switch {
			case actor.Enemy.ModDB.Flag(nil, "NeverCrit"):
				enemyCritChance = 0
			case actor.Enemy.ModDB.Flag(nil, "AlwaysCrit"):
				enemyCritChance = 100
			default:
				base := utils.Or(actor.ModDB.Override(nil, "enemyCritChance"), enemyConfigNumber(environment, "enemyCritChance"))
				scaled := base *
					(1 + actor.ModDB.Sum(mod.TypeIncrease, nil, "EnemyCritChance")/100 +
						actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "CritChance")/100) *
					(1 - actor.Output["ConfiguredEvadeChance"]/100)
				enemyCritChance = math.Max(math.Min(scaled, 100), 0)
			}
			actor.Output["EnemyCritChance"] = enemyCritChance
			enemyCritDamage := math.Max(enemyConfigNumber(environment, "enemyCritDamage")+
				actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "CritMultiplier"), 0)
			actor.Output["EnemyCritEffect"] = 1 + float64(enemyCritChance/100*(enemyCritDamage/100)*(1-actor.Output["CritExtraDamageReduction"]/100))
			enemyDamageConversion := make(map[data.DamageType]map[data.DamageType]float64)
			for _, damageType := range data.DamageType("").Values() {
				enemyDamageMult := calclib.Mod(actor.Enemy.ModDB, nil, "Damage", string(damageType+"Damage"), utils.Ternary(isElemental[string(damageType)], "ElementalDamage", "")) // missing taunt from allies
				enemyDamage, fromInput := enemyConfigNumberSource(environment, string("enemy"+damageType+"Damage"))
				sourceStr := utils.Ternary(fromInput, "Config", "Default")
				enemyPen := enemyConfigNumber(environment, string("enemy"+damageType+"Pen"))
				enemyOverwhelm, _ := environment.Build.GetNumberOptionOK(string("enemy" + damageType + "Overwhelm"))

				enemyCfgKeywordFlags := mod.MatchAllMask
				enemyCfg := &moddb.ListCfg{KeywordFlags: &enemyCfgKeywordFlags}
				enemyDamage = enemyDamage + (actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+"Min"))+actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+"Max")))/2

				conversionTotal := float64(0)
				if damageType == data.DamageTypePhysical {
					conversion := make(map[data.DamageType]float64, 5)
					skillConversion := make(map[data.DamageType]float64, 5)
					total, totalSkill := float64(0), float64(0)
					for _, to := range data.DamageType("").Values() {
						skillConversion[to] = actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+"DamageSkillConvertTo"+to))
						conversion[to] = actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+"DamageConvertTo"+to))
						totalSkill += skillConversion[to]
						total += conversion[to]
					}
					switch {
					case totalSkill > 100:
						mult := 100 / totalSkill
						totalSkill *= mult
						total = 0
						for _, to := range data.DamageType("").Values() {
							skillConversion[to] *= mult
							conversion[to] = 0
						}
					case total+totalSkill > 100:
						mult := (100 - totalSkill) / total
						total *= mult
						for _, to := range data.DamageType("").Values() {
							conversion[to] *= mult
						}
					}
					conversionTotal = total + totalSkill
					// Calculate the amount converted/gained as
					for _, to := range data.DamageType("").Values() {
						gainAsPercent := actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+"DamageGainAs"+to)) / 100
						conversionPercent := conversion[to] / 100
						skillConversionPercent := skillConversion[to] / 100
						if skillConversionPercent > 0 && to != data.DamageTypeChaos {
							physBonus := 1 + data.MonsterPhysConversionMultiTable[environment.EnemyLevel]/100
							conversionPercent += float64(skillConversionPercent * physBonus)
						}
						if gainAsPercent > 0 || conversionPercent > 0 {
							if enemyDamageConversion[to] == nil {
								enemyDamageConversion[to] = make(map[data.DamageType]float64, 1)
							}
							enemyDamageConversion[to][damageType] = float64(enemyDamage*gainAsPercent) + float64(enemyDamage*conversionPercent)
						}
					}
				}

				//pob:block:src/Modules/CalcDefence.lua:1763
				enemyOverwhelm = enemyOverwhelm +
					actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "PhysicalOverwhelm") +
					modDB.Sum(mod.TypeBase, nil, "EnemyPhysicalOverwhelm")

				actor.Output[string(damageType+"EnemyPen")] = enemyPen
				actor.Output[string(damageType+"EnemyDamageMult")] = enemyDamageMult
				actor.Output[string(damageType+"EnemyOverwhelm")] = enemyOverwhelm
				actor.Output["totalEnemyDamageIn"] = actor.Output["totalEnemyDamageIn"] + enemyDamage
				actor.Output[string(damageType+"EnemyDamage")] = enemyDamage * (1 - conversionTotal/100) * enemyDamageMult * actor.Output["EnemyCritEffect"]
				conversionExtra := float64(-enemyDamage*enemyDamageMult*actor.Output["EnemyCritEffect"]) + actor.Output[string(damageType+"EnemyDamage")]
				if fromTypes := enemyDamageConversion[damageType]; fromTypes != nil {
					for _, fromType := range data.DamageType("").Values() {
						convertedDamage, ok := fromTypes[fromType]
						if !ok {
							continue
						}
						convertedMult := calclib.Mod(actor.Enemy.ModDB, nil, "Damage", string(damageType+"Damage"), string(fromType+"Damage"),
							utils.Ternary(isElemental[string(damageType)], "ElementalDamage", ""),
							utils.Ternary(isElemental[string(fromType)], "ElementalDamage", "")) // missing taunt from allies
						actor.Output[string(damageType+"EnemyDamage")] += float64(convertedDamage * convertedMult * actor.Output["EnemyCritEffect"])
						conversionExtra += float64(convertedDamage * convertedMult * actor.Output["EnemyCritEffect"])
					}
				}
				actor.Output["totalEnemyDamage"] = actor.Output["totalEnemyDamage"] + actor.Output[string(damageType+"EnemyDamage")]
				if actor.Breakdown != nil {
					convSign := ""
					switch {
					case conversionExtra > 0:
						convSign = "+"
					case conversionExtra < 0:
						convSign = "-"
					}
					//pob:block:src/Modules/CalcDefence.lua:1779-1788
					lines := []string{
						fmt.Sprintf("from %s: %d", sourceStr, int(enemyDamage)),
						fmt.Sprintf("* %.2f (modifiers to enemy damage)", enemyDamageMult),
						fmt.Sprintf("* %.3f (enemy crit effect)", actor.Output["EnemyCritEffect"]),
					}
					if conversionExtra != 0 {
						lines = append(lines, fmt.Sprintf("%s %d (enemy damage conversion)", convSign, int(math.Abs(conversionExtra))))
					}
					lines = append(lines, fmt.Sprintf("= %d", int(actor.Output[string(damageType+"EnemyDamage")])))
					actor.Breakdown.Data[string(damageType+"EnemyDamage")] = &Entry{Lines: lines}
					actor.Breakdown.AddRow("totalEnemyDamage", map[string]string{
						"type":     string(damageType),
						"value":    fmt.Sprintf("%.2f", enemyDamage),
						"mult":     fmt.Sprintf("%.2f", enemyDamageMult),
						"crit":     fmt.Sprintf("%.2f", actor.Output["EnemyCritEffect"]),
						"convMult": fmt.Sprintf("%s%.2f", convSign, math.Abs(conversionExtra)),
						"final":    fmt.Sprintf("%.2f", actor.Output[string(damageType+"EnemyDamage")]),
						"from":     sourceStr,
					})
				}
			}
		}
	}
	{
		actor.DamageShiftTable = make(map[data.DamageType]map[data.DamageType]float64)

		actor.DamageOverTimeShiftTable = make(map[data.DamageType]map[data.DamageType]float64)
		for _, damageType := range data.DamageType("").Values() {
			shiftTable := make(map[data.DamageType]float64)
			dotShiftTable := make(map[data.DamageType]float64)
			destTotal := float64(0)
			dotDestinationTotal := float64(0)
			for _, destType := range data.DamageType("").Values() {
				if destType != damageType {
					dotShiftTable[destType] = modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenAs"+destType), utils.Ternary(isElemental[string(damageType)], string("ElementalDamageTakenAs"+destType), ""))
					dotDestinationTotal = dotDestinationTotal + dotShiftTable[destType]
					shiftTable[destType] = dotShiftTable[destType] +
						modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageFromHitsTakenAs"+destType), utils.Ternary(isElemental[string(damageType)], string("ElementalDamageFromHitsTakenAs"+destType), ""))
					destTotal = destTotal + shiftTable[destType]
				}
			}
			//pob:block:src/Modules/CalcDefence.lua:1820-1823
			shiftTable[damageType] = max(100-destTotal, 0)
			actor.DamageShiftTable[damageType] = shiftTable
			dotShiftTable[damageType] = max(100-dotDestinationTotal, 0)
			actor.DamageOverTimeShiftTable[damageType] = dotShiftTable

			// add same type damage
			actor.Output[string(damageType+"TakenDamage")] = actor.Output[string(damageType+"EnemyDamage")] * actor.DamageShiftTable[damageType][damageType] / 100
			if actor.Breakdown != nil {
				actor.Breakdown.SetLabel(string(damageType+"TakenDamage"), "Taken")
				actor.Breakdown.AddCol(string(damageType+"TakenDamage"),
					BCol{Label: "Type", Key: "type"},
					BCol{Label: "Value", Key: "value"},
				)
				actor.Breakdown.AddRow(string(damageType+"TakenDamage"), map[string]string{
					"type":  string(damageType),
					"value": fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenDamage")]),
				})
			}
		}
		// converted damage types
		for _, damageType := range data.DamageType("").Values() {
			for _, damageConvertedType := range data.DamageType("").Values() {
				if damageType != damageConvertedType {
					damage := actor.Output[string(damageType+"EnemyDamage")] * actor.DamageShiftTable[damageType][damageConvertedType] / 100
					actor.Output[string(damageConvertedType+"TakenDamage")] = actor.Output[string(damageConvertedType+"TakenDamage")] + damage
					if actor.Breakdown != nil && damage > 0 {
						actor.Breakdown.AddRow(string(damageConvertedType+"TakenDamage"), map[string]string{
							"type":  string(damageType),
							"value": fmt.Sprintf("%.2f", damage),
						})
					}
				}
			}
		}
		actor.Output["totalTakenDamage"] = 0
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("totalTakenDamage", "Total damage taken from the enemy after taken as")
			actor.Breakdown.AddCol("totalTakenDamage",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Value", Key: "value"},
			)
		}
		for _, damageType := range data.DamageType("").Values() {
			actor.Output["totalTakenDamage"] = actor.Output["totalTakenDamage"] + actor.Output[string(damageType+"TakenDamage")]
			if actor.Breakdown != nil {
				actor.Breakdown.AddRow("totalTakenDamage", map[string]string{
					"type":  string(damageType),
					"value": fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenDamage")]),
				})
			}
		}
	}
	// Damage taken multipliers/Degen calculations
	actor.Output["AnyTakenReflect"] = 0
	damageCategoryConfig := enemyDamageCategory(environment)
	for _, damageType := range data.DamageType("").Values() {
		baseTakenInc := modDB.Sum(mod.TypeIncrease, nil, "DamageTaken", string(damageType+"DamageTaken"))
		baseTakenMore := modDB.More(nil, "DamageTaken", string(damageType+"DamageTaken"))
		if isElemental[string(damageType)] {
			baseTakenInc = baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTaken")
			baseTakenMore = baseTakenMore * modDB.More(nil, "ElementalDamageTaken")
		}
		{
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
			if isElemental[string(damageType)] {
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenWhenHit")
				takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenWhenHit")
			}
			actor.Output[string(damageType+"TakenHitMult")] = max((1+takenInc/100)*takenMore, 0)

			for _, hitType := range []string{"Attack", "Spell"} {
				baseTakenIncType := takenInc + modDB.Sum(mod.TypeIncrease, nil, hitType+"DamageTaken")
				baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
				actor.Output[hitType+"TakenHitMult"] = max((1+baseTakenIncType/100)*baseTakenMoreType, 0)
				actor.Output[string(damageType)+hitType+"TakenHitMult"] = actor.Output[hitType+"TakenHitMult"]
			}
			{
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ReflectedDamageTaken", string(damageType+"ReflectedDamageTaken"))
				takenMore = takenMore * modDB.More(nil, "ReflectedDamageTaken", string(damageType+"ReflectedDamageTaken"))
				if isElemental[string(damageType)] {
					takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalReflectedDamageTaken")
					takenMore = takenMore * modDB.More(nil, "ElementalReflectedDamageTaken")
				}
				actor.Output[string(damageType+"TakenReflect")] = max((1+takenInc/100)*takenMore, 0)
				if actor.Output[string(damageType+"TakenReflect")] != actor.Output[string(damageType+"TakenHitMult")] {
					actor.Output["AnyTakenReflect"] = 0 // true // this needs a rework as well
				}
			}
		}
		{
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenOverTime", string(damageType+"DamageTakenOverTime"))
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", string(damageType+"DamageTakenOverTime"))
			if isElemental[string(damageType)] {
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenOverTime")
				takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenOverTime")
			}
			resist := actor.Output[string(damageType+"Resist")]
			if overTime, ok := actor.Output[string(damageType+"ResistOverTime")]; ok {
				resist = overTime
			}
			if modDB.Flag(nil, "SelfIgnore"+string(damageType)+"Resistance") {
				resist = 0
			}
			reduction := utils.Ternary(modDB.Flag(nil, string("SelfIgnoreBase"+damageType+"DamageReduction")), 0, actor.Output[string("Base"+damageType+"DamageReduction")])
			actor.Output[string(damageType+"TakenDotMult")] = max((1-resist/100)*(1-reduction/100)*(1+takenInc/100)*takenMore, 0)
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain(string(damageType+"TakenDotMult"), BMultiChain{
					Label: "DoT Multiplier:",
					Total: fmt.Sprintf("= %.2f", actor.Output[string(damageType+"TakenDotMult")]),
					Items: []BMultiChainItem{
						{"%.2f ^8(resistance)", 1 - resist/100},
						{"%.2f ^8(" + strings.ToLower(string(damageType)) + " damage reduction)", 1 - reduction/100},
						{"%.2f ^8(increased/reduced damage taken)", 1 + takenInc/100},
						{"%.2f ^8(more/less damage taken)", takenMore},
					},
				})
			}
		}
	}
	// Incoming hit damage multipliers
	actor.Output["totalTakenHit"] = 0
	if actor.Breakdown != nil {
		actor.Breakdown.SetLabel("totalTakenHit", "Total damage taken after mitigation")
		actor.Breakdown.AddCol("totalTakenHit",
			BCol{Label: "Type", Key: "type"},
			BCol{Label: "Incoming", Key: "incoming"},
			BCol{Label: "Mult", Key: "mult"},
			BCol{Label: "Value", Key: "value"},
		)
	}
	//pob:block:src/Modules/CalcDefence.lua:1956
	impaleFlags, impaleKeywords := mod.MFlag(0), mod.KeywordFlag(0)
	if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" || damageCategoryConfig == "Average" {
		impaleFlags = mod.MFlagAttack
	}
	enemyImpaleChance := actor.Enemy.ModDB.Sum(mod.TypeBase, &moddb.ListCfg{Flags: &impaleFlags, KeywordFlags: &impaleKeywords}, "ImpaleChance") *
		utils.Ternary(damageCategoryConfig == "Average", 0.5, float64(1)) * (1 - actor.Output["ImpaleAvoidChance"])
	for _, damageType := range data.DamageType("").Values() {
		// Calculate incoming damage multiplier
		resist := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageType+"Resistance")), 0, utils.OrF(actor.Output[string(damageType+"ResistWhenHit")], actor.Output[string(damageType+"Resist")]))
		reduction := utils.Ternary(modDB.Flag(nil, string("SelfIgnoreBase"+damageType+"DamageReduction")), 0, actor.Output[string("Base"+damageType+"DamageReductionWhenHit")])
		enemyPen := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageType+"Resistance"), string("EnemyCannotPen"+damageType+"Resistance")), 0, actor.Output[string(damageType+"EnemyPen")])
		enemyOverwhelm := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageType+"DamageReduction")), 0, actor.Output[string(damageType+"EnemyOverwhelm")])
		takenFlat := modDB.Sum(mod.TypeBase, nil, "DamageTaken", string(damageType+"DamageTaken"), "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
		damage := actor.Output[string(damageType+"TakenDamage")]
		if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
			takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", string(damageType+"DamageTakenFromAttacks"), string(damageType)+"DamageTakenFrom"+damageCategoryConfig+"Attacks")
		} else if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
			takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpells", string(damageType+"DamageTakenFromSpells"), string(damageType+"DamageTakenFromSpellProjectiles"))
		} else if damageCategoryConfig == "Average" {
			takenFlat = takenFlat +
				modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", string(damageType+"DamageTakenFromAttacks"))/2 +
				modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenFromProjectileAttacks"))/4 +
				modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpells", string(damageType+"DamageTakenFromSpells"))/2 +
				modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpellProjectiles", string(damageType+"DamageTakenFromSpellProjectiles"))/4
		}
		actor.Output[string(damageType)+"takenFlat"] = takenFlat

		percentOfArmourApplies := min(utils.Ternary(
			modDB.Flag(nil, string("ArmourDoesNotApplyTo"+damageType+"DamageTaken")),
			float64(0),
			modDB.Sum(mod.TypeBase, nil, string("ArmourAppliesTo"+damageType+"DamageTaken")),
		), 100)
		//pob:block:src/Modules/CalcDefence.lua:1964
		impaleDamage := float64(0)
		if enemyImpaleChance > 0 && damageType == data.DamageTypePhysical {
			impaleDamage = damage * data.ImpaleStoredDamageBase
		}
		impaleArmourReduct := float64(0)
		effectiveAppliedArmour := (actor.Output["Armour"] * percentOfArmourApplies / 100) * (1 + actor.Output["ArmourDefense"])
		//pob:block:src/Modules/CalcDefence.lua:1969-1973
		physicalReductionBasedOnWard := damageType == data.DamageTypePhysical && modDB.Flag(nil, "PhysicalReductionBasedOnWard")
		if physicalReductionBasedOnWard {
			multiplier := float64(0)
			if override := modDB.Override(nil, "PhysicalReductionBasedOnWardPercent"); override != nil {
				multiplier = override.Float() / 100
			}
			effectiveAppliedArmour = actor.Output["Ward"] * multiplier
		}
		resMult := 1 - (resist-enemyPen)/100
		armourReduct := float64(0)
		//pob:block:src/Modules/CalcDefence.lua:1985-1998
		if percentOfArmourApplies > 0 || physicalReductionBasedOnWard {
			armourReduct = CalcArmourReduction(effectiveAppliedArmour, damage*resMult)
			armourReduct = min(actor.Output["DamageReductionMax"], armourReduct)
			if impaleDamage > 0 {
				impaleArmourReduct = min(actor.Output["DamageReductionMax"], CalcArmourReduction(effectiveAppliedArmour, impaleDamage*resMult))
			}
		}
		totalReduct := min(actor.Output["DamageReductionMax"], armourReduct+reduction)
		reductMult := 1 - max(min(actor.Output["DamageReductionMax"], totalReduct-enemyOverwhelm), 0)/100
		actor.Output[string(damageType+"DamageReduction")] = 100 - float64(reductMult*100)
		if impaleDamage > 0 {
			impaleDamage = impaleDamage * resMult * (1 - max(min(actor.Output["DamageReductionMax"], min(actor.Output["DamageReductionMax"], impaleArmourReduct+reduction)-enemyOverwhelm), 0)/100)
			impaleDamage = impaleDamage * enemyImpaleChance / 100 * 5 * actor.Output[string(damageType+"TakenReflect")]
		}
		//pob:block:src/Modules/CalcDefence.lua:1999-2021
		if actor.Breakdown != nil {
			actor.Breakdown.Data[string(damageType+"DamageReduction")] = &Entry{}
			if armourReduct != 0 {
				if resMult != 1 {
					actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Enemy Hit Damage After Resistance: %d ^8(total incoming damage)", int(damage*resMult)))
				} else {
					actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Enemy Hit Damage: %d ^8(total incoming damage)", int(damage)))
				}
				if percentOfArmourApplies != 100 && !physicalReductionBasedOnWard {
					actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("%d%% percent of armour applies", int(percentOfArmourApplies)))
				}
				actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Reduction from Armour: %d%%", int(armourReduct)))
			}
			if reduction != 0 {
				actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Base %s Damage Reduction: %d%%", damageType, int(reduction)))
			}
			if enemyOverwhelm != 0 {
				actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Enemy Overwhelm %s Damage: %d%%", damageType, int(enemyOverwhelm)))
			}
			if utils.Ternary(armourReduct != 0, 1, 0)+utils.Ternary(reduction != 0, 1, 0)+utils.Ternary(enemyOverwhelm != 0, 1, 0) >= 2 {
				actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Total %s Damage Reduction: %d%%", damageType, int(100-float64(reductMult*100))))
			}
		}
		takenMult := actor.Output[string(damageType+"TakenHitMult")]
		spellSuppressMult := float64(1)
		if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
			takenMult = actor.Output[string(damageType+"AttackTakenHitMult")]
		} else if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
			takenMult = actor.Output[string(damageType+"SpellTakenHitMult")]
			spellSuppressMult = utils.Ternary(actor.Output["EffectiveSpellSuppressionChance"] == 100, 1-actor.Output["SpellSuppressionEffect"]/100, float64(1))
		} else if damageCategoryConfig == "Average" {
			takenMult = (actor.Output[string(damageType+"SpellTakenHitMult")] + actor.Output[string(damageType+"AttackTakenHitMult")]) / 2
			spellSuppressMult = utils.Ternary(actor.Output["EffectiveSpellSuppressionChance"] == 100, 1-actor.Output["SpellSuppressionEffect"]/100/2, float64(1))
		}
		actor.Output[string(damageType+"EffectiveAppliedArmour")] = effectiveAppliedArmour
		actor.Output[string(damageType+"ResistTakenHitMulti")] = resMult
		afterReductionMulti := takenMult * spellSuppressMult
		actor.Output[string(damageType+"AfterReductionTakenHitMulti")] = afterReductionMulti
		baseMult := resMult * reductMult
		actor.Output[string(damageType+"BaseTakenHitMult")] = baseMult * afterReductionMulti
		takenMultReflect := actor.Output[string(damageType+"TakenReflect")]
		finalReflect := baseMult * takenMultReflect
		actor.Output[string(damageType+"TakenHit")] = float64(max(float64(damage*baseMult)+takenFlat, 0)*takenMult*spellSuppressMult) + impaleDamage
		actor.Output[string(damageType+"TakenHitMult")] = utils.Ternary(damage > 0, actor.Output[string(damageType+"TakenHit")]/damage, 0)
		actor.Output["totalTakenHit"] = actor.Output["totalTakenHit"] + actor.Output[string(damageType+"TakenHit")]
		if actor.Output["AnyTakenReflect"] != 0 {
			actor.Output[string(damageType+"TakenReflectMult")] = finalReflect
		}
		//pob:block:src/Modules/CalcDefence.lua:2047-2095
		if actor.Breakdown != nil {
			key := string(damageType + "TakenHitMult")
			actor.Breakdown.Data[key] = &Entry{}
			if resist != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("Resistance: %.2f", 1-resist/100))
			}
			if enemyPen != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("+ Enemy Pen: %.2f", enemyPen/100))
			}
			if resist != 0 && enemyPen != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("= %.2f", resMult))
			}
			if reduction != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("Base %s Damage Taken: %.2f", damageType, 1-reduction/100))
			}
			if armourReduct != 0 {
				if resMult != 1 {
					actor.Breakdown.AddLine(key, fmt.Sprintf("Enemy Hit Damage After Resistance: %d ^8(total incoming damage)", int(damage*resMult)))
				} else {
					actor.Breakdown.AddLine(key, fmt.Sprintf("Enemy Hit Damage: %d ^8(total incoming damage)", int(damage)))
				}
				if percentOfArmourApplies != 100 && !physicalReductionBasedOnWard {
					actor.Breakdown.AddLine(key, fmt.Sprintf("%d%% percent of armour applies", int(percentOfArmourApplies)))
				}
				actor.Breakdown.AddLine(key, fmt.Sprintf("Reduction from Armour: %.2f", 1-armourReduct/100))
			}
			if enemyOverwhelm != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("+ Enemy Overwhelm %s Damage: %.2f", damageType, enemyOverwhelm/100))
			}
			if reduction != 0 || armourReduct != 0 || enemyOverwhelm != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("= %.2f", reductMult))
			}
			if resMult != 1 && reductMult != 1 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("%.2f x %.2f = %.3f", resMult, reductMult, baseMult))
			}
			if takenFlat != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("+ Flat: %d", int(takenFlat)))
			}
			if takenMult != 1 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("x Taken: %.3f", takenMult))
			}
			if spellSuppressMult != 1 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("x Spell Suppression: %.3f", spellSuppressMult))
			}
			if impaleDamage != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("+ Impale: %.1f", impaleDamage))
			}
			if takenMult != 1 || takenFlat != 0 || spellSuppressMult != 1 || impaleDamage != 0 {
				actor.Breakdown.AddLine(key, fmt.Sprintf("= %.3f", actor.Output[key]))
			}
			actor.Breakdown.AddRow("totalTakenHit", map[string]string{
				"type":     string(damageType),
				"incoming": fmt.Sprintf("%.1f incoming damage", actor.Output[string(damageType+"TakenDamage")]),
				"mult":     fmt.Sprintf("x %.3f damage mult", actor.Output[string(damageType+"TakenHitMult")]),
				"value":    fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenHit")]),
			})
			if actor.Output["AnyTakenReflect"] != 0 {
				actor.Breakdown.AddLine("TakenReflectMult",
					fmt.Sprintf("Resistance: %.3f", 1-resist/100),
				)
				if enemyPen > 0 {
					actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("Enemy Pen: %.2f", enemyPen))
				}
				actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("Taken: %.3f", takenMultReflect))
				actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("= %.3f", finalReflect))
			}
		}
	}
	{
		stunThresholdBase := float64(0)
		stunThresholdSource := ""
		if modDB.Flag(nil, "StunThresholdBasedOnEnergyShieldInsteadOfLife") {
			stunThresholdMult := modDB.Sum(mod.TypeBase, nil, "StunThresholdEnergyShieldPercent")
			stunThresholdBase = actor.Output["EnergyShield"] * stunThresholdMult / 100
			stunThresholdSource = fmt.Sprintf("%.14g%% of Energy Shield", stunThresholdMult)
		} else if modDB.Flag(nil, "StunThresholdBasedOnManaInsteadOfLife") {
			stunThresholdMult := modDB.Sum(mod.TypeBase, nil, "StunThresholdManaPercent")
			stunThresholdBase = actor.Output["Mana"] * stunThresholdMult / 100
			stunThresholdSource = fmt.Sprintf("%.14g%% of Mana", stunThresholdMult)
		} else if modDB.Flag(nil, "ChaosInoculation") {
			stunThresholdBase = modDB.Sum(mod.TypeBase, nil, "Life")
			stunThresholdSource = "Life before Chaos Inoculation"
		} else {
			stunThresholdBase = actor.Output["Life"]
			stunThresholdSource = "Life"
		}
		//pob:block:src/Modules/CalcDefence.lua:2128-2132
		if modDB.Flag(nil, "AddESToStunThreshold") {
			esMult := modDB.Sum(mod.TypeBase, nil, "ESToStunThresholdPercent")
			stunThresholdBase += actor.Output["EnergyShield"] * esMult / 100
			stunThresholdSource = fmt.Sprintf("%s and %.14g%% of Energy Shield", stunThresholdSource, esMult)
		}
		stunThresholdMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunThreshold")/100
		actor.Output["StunThreshold"] = stunThresholdBase * stunThresholdMod

		//pob:block:src/Modules/CalcDefence.lua:1328-1376
		//pob:block:src/Modules/CalcDefence.lua:2136-2139
		notAvoidChance := float64(0)
		if !modDB.Flag(nil, "StunImmune") {
			notAvoidChance = 100 - min(modDB.Sum(mod.TypeBase, nil, "AvoidStun"), 100)
		}
		if actor.Output["EnergyShield"] > actor.Output["totalTakenHit"] && !environment.ModDB.Flag(nil, "EnergyShieldProtectsMana") {
			notAvoidChance = notAvoidChance * 0.5
		}
		actor.Output["StunAvoidChance"] = 100 - notAvoidChance

		//pob:block:src/Modules/CalcDefence.lua:2142-2152
		if actor.Breakdown != nil {
			actor.Breakdown.Data["StunThreshold"] = &Entry{Lines: []string{fmt.Sprintf("%d ^8(base from %s)", int(stunThresholdBase), stunThresholdSource)}}
			if stunThresholdMod != 1 {
				actor.Breakdown.AddLine("StunThreshold", fmt.Sprintf("* %.2f ^8(increased threshold)", stunThresholdMod))
				actor.Breakdown.AddLine("StunThreshold", fmt.Sprintf("= %d", int(actor.Output["StunThreshold"])))
			}
			actor.Breakdown.Data["StunAvoidChance"] = &Entry{Lines: []string{
				colorCustom + "NOTE: Having any energy shield when the hit occurs grants 50% chance to avoid stun.",
				colorCustom + "POB only applies this modifier when ES > Total incoming damage.",
			}}
		}
		//pob:block:src/Modules/CalcDefence.lua:2153-2186
		if actor.Output["StunAvoidChance"] >= 100 {
			actor.Output["StunDuration"] = 0
			actor.Output["BlockDuration"] = 0
			if actor.Breakdown != nil {
				actor.Breakdown.Data["StunDuration"] = &Entry{Lines: []string{"Cannot be Stunned"}}
				actor.Breakdown.Data["BlockDuration"] = &Entry{Lines: []string{"Cannot be Stunned"}}
			}
		} else {
			stunDuration := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunDuration")/100
			baseStunDuration := float64(data.StunBaseDuration)
			stunRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery")/100
			stunAndBlockRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery")/100
			//pob:block:src/Modules/CalcDefence.lua:2165-2166
			actor.Output["StunDuration"] = math.Ceil(baseStunDuration*stunDuration/stunRecovery*data.ServerTickRate) / data.ServerTickRate
			actor.Output["BlockDuration"] = math.Ceil(baseStunDuration*stunDuration/stunAndBlockRecovery*data.ServerTickRate) / data.ServerTickRate
			if actor.Breakdown != nil {
				actor.Breakdown.Data["StunDuration"] = &Entry{Lines: []string{fmt.Sprintf("%.2fs ^8(base)", baseStunDuration)}}
				actor.Breakdown.Data["BlockDuration"] = &Entry{Lines: []string{fmt.Sprintf("%.2fs ^8(base)", baseStunDuration)}}
				if stunDuration != 1 {
					actor.Breakdown.AddLine("StunDuration", fmt.Sprintf("* %.2f ^8(increased duration)", stunDuration))
					actor.Breakdown.AddLine("BlockDuration", fmt.Sprintf("* %.2f ^8(increased duration)", stunDuration))
				}
				if stunRecovery != 1 {
					actor.Breakdown.AddLine("StunDuration", fmt.Sprintf("/ %.2f ^8(increased/reduced recovery)", stunRecovery))
				}
				if stunAndBlockRecovery != 1 {
					actor.Breakdown.AddLine("BlockDuration", fmt.Sprintf("/ %.2f ^8(increased/reduced block recovery)", stunAndBlockRecovery))
				}
				actor.Breakdown.AddLine("StunDuration", "rounded up to nearest server tick", fmt.Sprintf("= %.2fs", actor.Output["StunDuration"]))
				actor.Breakdown.AddLine("BlockDuration", "rounded up to nearest server tick", fmt.Sprintf("= %.2fs", actor.Output["BlockDuration"]))
			}
		}
		actor.Output["InterruptStunAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidInterruptStun", "AvoidInteruptStun"), 100)

		effectiveEnemyDamage := actor.Output["totalTakenHit"] + float64(actor.Output["PhysicalTakenHit"]*0.25)
		if damageCategoryConfig != "Average" {
			effectiveEnemyDamage = effectiveEnemyDamage * (1 + data.StunNotMeleeDamageMult*3) / 4
		} else if damageCategoryConfig != "Melee" {
			effectiveEnemyDamage = effectiveEnemyDamage * data.StunNotMeleeDamageMult
		}
		baseStunChance := min(data.StunBaseMult*effectiveEnemyDamage/actor.Output["StunThreshold"], 100)
		actor.Output["SelfStunChance"] = utils.Ternary(baseStunChance > data.MinStunChanceNeeded, baseStunChance, float64(0)) * notAvoidChance / 100
		//pob:block:src/Modules/CalcDefence.lua:2196-2211
		if actor.Breakdown != nil {
			actor.Breakdown.Data["SelfStunChance"] = &Entry{Lines: []string{
				fmt.Sprintf("%d%% ^8(stun multiplier)", int(data.StunBaseMult)),
				fmt.Sprintf("* %.1f ^8(effective enemy stun damage)", effectiveEnemyDamage),
				fmt.Sprintf("/ %d ^8(stun threshold)", int(actor.Output["StunThreshold"])),
			}}
			if baseStunChance < data.MinStunChanceNeeded {
				actor.Breakdown.AddLine("SelfStunChance", fmt.Sprintf("= %.2f%%", baseStunChance))
				actor.Breakdown.AddLine("SelfStunChance", fmt.Sprintf("Stun Chance has to be more than %d%% to stun.", int(data.MinStunChanceNeeded)))
			} else {
				if notAvoidChance != 100 {
					actor.Breakdown.AddLine("SelfStunChance", fmt.Sprintf("* %.2f ^8(1 - chance to avoid stun)", notAvoidChance/100))
				}
				actor.Breakdown.AddLine("SelfStunChance", fmt.Sprintf("= %.2f%%", actor.Output["SelfStunChance"]))
			}
		}
	}

	// Life Recoverable
	actor.Output["LifeRecoverable"] = actor.Output["LifeUnreserved"]
	if environment.Build.GetBooleanOption("conditionLowLife") {
		lowLifePercentage := actor.Output["LowLifePercentage"]
		if lowLifePercentage == 0 {
			lowLifePercentage = data.LowPoolThreshold
		}
		actor.Output["LifeRecoverable"] = min(actor.Output["Life"]*lowLifePercentage/100, actor.Output["LifeUnreserved"])
		if actor.Output["LifeRecoverable"] < actor.Output["LifeUnreserved"] {
			actor.Output["CappingLife"] = 1
		}
	}
	// Dissolution of the Flesh life pool change
	if modDB.Flag(nil, "DamageInsteadReservesLife") {
		actor.Output["LifeRecoverable"] = (actor.Output["LifeCancellableReservation"] / 100) * actor.Output["Life"]
	}
	actor.Output["LifeRecoverable"] = max(actor.Output["LifeRecoverable"], 1)

	// Prevented life loss taken over 4 seconds (and Petrified Blood)
	{
		recoverable := actor.Output["LifeRecoverable"]
		preventedLifeLoss := min(modDB.Sum(mod.TypeBase, nil, "LifeLossPrevented"), 100)
		actor.Output["preventedLifeLoss"] = preventedLifeLoss
		initialLifeLossBelowHalfPrevented := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
		actor.Output["preventedLifeLossBelowHalf"] = (1 - actor.Output["preventedLifeLoss"]/100) * initialLifeLossBelowHalfPrevented
		portionLife := float64(1)
		if !environment.Build.GetBooleanOption("conditionLowLife") {
			// portion of life that is lowlife
			portionLife = min(actor.Output["Life"]*0.5/recoverable, 1)
			actor.Output["preventedLifeLossTotal"] = actor.Output["preventedLifeLoss"] + float64(actor.Output["preventedLifeLossBelowHalf"]*portionLife)
		} else {
			actor.Output["preventedLifeLossTotal"] = actor.Output["preventedLifeLoss"] + actor.Output["preventedLifeLossBelowHalf"]
		}
		actor.Output["LifeHitPool"] = calcLifeHitPoolWithLossPrevention(recoverable, actor.Output["Life"], preventedLifeLoss, initialLifeLossBelowHalfPrevented)

		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("preventedLifeLossTotal", "Total life protected:")
			if actor.Output["preventedLifeLoss"] != 0 {
				actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("%.2f ^8(portion taken over 4 seconds instead)", actor.Output["preventedLifeLoss"]/100))
			}
			if actor.Output["preventedLifeLossBelowHalf"] != 0 {
				prefix := ""
				if actor.Output["preventedLifeLoss"] != 0 {
					prefix = "+ "
				}
				if portionLife != 1 {
					if actor.Output["preventedLifeLoss"] != 0 {
						actor.Breakdown.AddLine("preventedLifeLossTotal", "")
					}
					actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("%s%.2f ^8(initial portion taken by petrified blood)", prefix, initialLifeLossBelowHalfPrevented/100))
					if actor.Output["preventedLifeLoss"] != 0 {
						actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("* %.2f ^8(portion not already taken over time)", 1-actor.Output["preventedLifeLoss"]/100))
					}
					actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("* %.2f ^8(portion of life on low life)", portionLife))
					actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("= %.2f ^8(final portion taken by petrified blood)", actor.Output["preventedLifeLossBelowHalf"]*portionLife/100))
					actor.Breakdown.AddLine("preventedLifeLossTotal", "")
				} else {
					portionText := "portion"
					if actor.Output["preventedLifeLoss"] != 0 {
						portionText = "initial portion"
					}
					actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("%s%.2f ^8(%s taken by petrified blood)", prefix, initialLifeLossBelowHalfPrevented/100, portionText))
					if actor.Output["preventedLifeLoss"] != 0 {
						actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("* %.2f ^8(portion not already taken over time)", 1-actor.Output["preventedLifeLoss"]/100))
						actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("= %.2f ^8(final portion taken by petrified blood)", actor.Output["preventedLifeLossBelowHalf"]/100))
					}
				}
			}
			actor.Breakdown.AddLine("preventedLifeLossTotal", fmt.Sprintf("%.2f ^8(portion taken from life)", 1-actor.Output["preventedLifeLossTotal"]/100))
		}
	}
	// Energy Shield bypass
	actor.Output["AnyBypass"] = 0
	actor.Output["MinimumBypass"] = 100
	for _, damageType := range data.DamageType("").Values() {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			actor.Output[string(damageType+"EnergyShieldBypass")] = 100
			actor.Output["AnyBypass"] = 1
		} else {
			actor.Output[string(damageType+"EnergyShieldBypass")] = utils.Or(modDB.Override(nil, string(damageType+"EnergyShieldBypass")), modDB.Sum(mod.TypeBase, nil, string(damageType+"EnergyShieldBypass")))
			if actor.Output[string(damageType+"EnergyShieldBypass")] != 0 {
				actor.Output["AnyBypass"] = 1
			}
			if damageType == "Chaos" {
				if !modDB.Flag(nil, "ChaosNotBypassEnergyShield") {
					actor.Output[string(damageType+"EnergyShieldBypass")] = actor.Output[string(damageType+"EnergyShieldBypass")] + 100
				} else {
					actor.Output["AnyBypass"] = 1
				}
			}
		}
		actor.Output[string(damageType+"EnergyShieldBypass")] = max(min(actor.Output[string(damageType+"EnergyShieldBypass")], 100), 0)
		actor.Output["MinimumBypass"] = min(actor.Output["MinimumBypass"], actor.Output[string(damageType+"EnergyShieldBypass")])
	}

	actor.Output["ehpSectionAnySpecificTypes"] = 0
	// Mind over Matter
	actor.Output["OnlySharedMindOverMatter"] = 0
	actor.Output["AnySpecificMindOverMatter"] = 0
	actor.Output["sharedMindOverMatter"] = min(modDB.Sum(mod.TypeBase, nil, "DamageTakenFromManaBeforeLife"), 100)

	calcMoMEBPool := func(lifePool float64, momEffect float64, esBypass float64) (float64, float64, float64, float64) {
		return calcMoMEBPoolF(lifePool, momEffect, esBypass,
			actor.Output["ManaUnreserved"], actor.Output["EnergyShieldRecoveryCap"],
			modDB.Flag(nil, "EnergyShieldProtectsMana"))
	}

	momBreakdown := func(key string, lifePool float64, momEffect float64, maxMana float64, manaUsed float64, esUsed float64, pool float64) {
		if actor.Breakdown == nil {
			return
		}
		lifeProtected := (manaUsed+esUsed)/momEffect - (manaUsed + esUsed)
		actor.Breakdown.AddLine(key,
			"Effective life from Mind over Matter:",
			fmt.Sprintf("%.0f life (%.0f life is protected)", lifePool, lifeProtected),
		)
		if manaUsed == maxMana {
			actor.Breakdown.AddLine(key, fmt.Sprintf("%.0f mana", manaUsed))
		} else {
			actor.Breakdown.AddLine(key, fmt.Sprintf("%.0f mana (%.0f mana shadowed by energy shield)", manaUsed, maxMana-manaUsed))
		}
		esText := ""
		if modDB.Flag(nil, "EnergyShieldProtectsMana") {
			actor.Breakdown.AddLine(key, fmt.Sprintf("%.0f energy shield", esUsed))
			esText = fmt.Sprintf("+ %.0f ", esUsed)
		}
		actor.Breakdown.AddLine(key, fmt.Sprintf("%.0f + %.0f %s= %.0f", lifePool, manaUsed, esText, pool))
	}

	if actor.Output["sharedMindOverMatter"] > 0 {
		MindOverMatter := actor.Output["sharedMindOverMatter"] / 100
		esBypass := actor.Output["MinimumBypass"] / 100

		sharedMoMPool, breakdownMaxMana, breakdownMana, breakdownES := calcMoMEBPool(actor.Output["LifeRecoverable"], MindOverMatter, esBypass)
		actor.Output["sharedManaEffectiveLife"] = sharedMoMPool
		actor.Output["sharedMoMHitPool"], _, _, _ = calcMoMEBPool(actor.Output["LifeHitPool"], MindOverMatter, esBypass)

		momBreakdown("sharedMindOverMatter", actor.Output["LifeRecoverable"], MindOverMatter, breakdownMaxMana, breakdownMana, breakdownES, sharedMoMPool)
	} else {
		actor.Output["sharedManaEffectiveLife"] = actor.Output["LifeRecoverable"]
		actor.Output["sharedMoMHitPool"] = actor.Output["LifeHitPool"]
	}
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"MindOverMatter")] = min(modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenFromManaBeforeLife")), 100-actor.Output["sharedMindOverMatter"])
		if actor.Output[string(damageType+"MindOverMatter")] > 0 || (actor.Output[string(damageType+"EnergyShieldBypass")] > actor.Output["MinimumBypass"] && actor.Output["sharedMindOverMatter"] > 0) {
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnySpecificMindOverMatter"] = 1
			actor.Output["OnlySharedMindOverMatter"] = 0
			MindOverMatter := (actor.Output[string(damageType+"MindOverMatter")] + actor.Output["sharedMindOverMatter"]) / 100
			esBypass := actor.Output[string(damageType+"EnergyShieldBypass")] / 100

			typedMoMPool, breakdownMaxMana, breakdownMana, breakdownES := calcMoMEBPool(actor.Output["LifeRecoverable"], MindOverMatter, esBypass)
			actor.Output[string(damageType+"ManaEffectiveLife")] = typedMoMPool
			actor.Output[string(damageType+"MoMHitPool")], _, _, _ = calcMoMEBPool(actor.Output["LifeHitPool"], MindOverMatter, esBypass)

			momBreakdown(string(damageType+"MindOverMatter"), actor.Output["LifeRecoverable"], MindOverMatter, breakdownMaxMana, breakdownMana, breakdownES, typedMoMPool)
		} else {
			actor.Output[string(damageType+"ManaEffectiveLife")] = actor.Output["sharedManaEffectiveLife"]
			actor.Output[string(damageType+"MoMHitPool")] = actor.Output["sharedMoMHitPool"]
		}
	}
	actor.Output["AnyGuard"] = 0
	actor.Output["sharedGuardAbsorbRate"] = min(modDB.Sum(mod.TypeBase, nil, "GuardAbsorbRate"), 100)
	if actor.Output["sharedGuardAbsorbRate"] > 0 {
		actor.Output["OnlySharedGuard"] = 1
		actor.Output["sharedGuardAbsorb"] = calclib.Val(modDB, "GuardAbsorbLimit")
		lifeProtected := actor.Output["sharedGuardAbsorb"] / (actor.Output["sharedGuardAbsorbRate"] / 100) * (1 - actor.Output["sharedGuardAbsorbRate"]/100)
		//pob:block:src/Modules/CalcDefence.lua:2386-2394
		if actor.Breakdown != nil {
			actor.Breakdown.Data["sharedGuardAbsorb"] = &Entry{Lines: []string{
				"Total life protected:",
				fmt.Sprintf("%d ^8(guard limit)", int(actor.Output["sharedGuardAbsorb"])),
				fmt.Sprintf("/ %.2f ^8(portion taken from guard)", actor.Output["sharedGuardAbsorbRate"]/100),
				fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output["sharedGuardAbsorbRate"]/100),
				fmt.Sprintf("= %d", int(lifeProtected)),
			}}
		}
	}
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"GuardAbsorbRate")] = min(modDB.Sum(mod.TypeBase, nil, string(damageType+"GuardAbsorbRate")), 100)
		if actor.Output[string(damageType+"GuardAbsorbRate")] > 0 {
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnyGuard"] = 1
			actor.Output["OnlySharedGuard"] = 0
			actor.Output[string(damageType+"GuardAbsorb")] = calclib.Val(modDB, string(damageType+"GuardAbsorbLimit"))
			lifeProtected := actor.Output[string(damageType+"GuardAbsorb")] / (actor.Output[string(damageType+"GuardAbsorbRate")] / 100) * (1 - actor.Output[string(damageType+"GuardAbsorbRate")]/100)
			//pob:block:src/Modules/CalcDefence.lua:2404-2412
			if actor.Breakdown != nil {
				actor.Breakdown.Data[string(damageType+"GuardAbsorb")] = &Entry{Lines: []string{
					"Total life protected:",
					fmt.Sprintf("%d ^8(guard limit)", int(actor.Output[string(damageType+"GuardAbsorb")])),
					fmt.Sprintf("/ %.2f ^8(portion taken from guard)", actor.Output[string(damageType+"GuardAbsorbRate")]/100),
					fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output[string(damageType+"GuardAbsorbRate")]/100),
					fmt.Sprintf("= %d", int(lifeProtected)),
				}}
			}
		}
	}
	actor.Output["AnyAegis"] = 0
	actor.Output["sharedAegis"] = modDB.Max(nil, "AegisValue")
	actor.Output["sharedElementalAegis"] = modDB.Max(nil, "ElementalAegisValue")
	if actor.Output["sharedAegis"] > 0 {
		actor.Output["AnyAegis"] = 1
	}
	if actor.Output["sharedElementalAegis"] > 0 {
		actor.Output["ehpSectionAnySpecificTypes"] = 1
		actor.Output["AnyAegis"] = 1
	}
	for _, damageType := range data.DamageType("").Values() {
		aegisValue := modDB.Max(nil, string(damageType+"AegisValue"))
		if aegisValue > 0 {
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnyAegis"] = 1
			actor.Output[string(damageType+"Aegis")] = aegisValue
		} else {
			actor.Output[string(damageType+"Aegis")] = 0
		}
		if isElemental[string(damageType)] {
			actor.Output[string(damageType+"AegisDisplay")] = actor.Output[string(damageType+"Aegis")] + actor.Output["sharedElementalAegis"]
		}
	}
	// frost shield
	{
		actor.Output["FrostShieldLife"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeHealth")
		actor.Output["FrostShieldDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeDamageMitigation")

		lifeProtected := actor.Output["FrostShieldLife"] / (actor.Output["FrostShieldDamageMitigation"] / 100) * (1 - actor.Output["FrostShieldDamageMitigation"]/100)
		//pob:block:src/Modules/CalcDefence.lua:2448-2456
		if actor.Breakdown != nil {
			actor.Breakdown.Data["FrostShieldLife"] = &Entry{Lines: []string{
				"Total life protected:",
				fmt.Sprintf("%d ^8(frost shield limit)", int(actor.Output["FrostShieldLife"])),
				fmt.Sprintf("/ %.2f ^8(portion taken from frost shield)", actor.Output["FrostShieldDamageMitigation"]/100),
				fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output["FrostShieldDamageMitigation"]/100),
				fmt.Sprintf("= %d", int(lifeProtected)),
			}}
		}
	}
	{
		damageCategoryConfig := enemyDamageCategory(environment)
		for _, ally := range allyLifePoolList {
			if ally.redirect == "" {
				continue
			}
			mitigation := modDB.Sum(mod.TypeBase, nil, ally.redirect)
			if ally.meleeOnly {
				switch damageCategoryConfig {
				case "Melee":
				case "Average":
					mitigation /= 4
				default:
					mitigation = 0
				}
			}
			actor.Output[ally.mitigation] = mitigation
			if mitigation != 0 {
				life := modDB.Sum(mod.TypeBase, nil, ally.life)
				if life == 0 && ally.fallback != "" {
					life = modDB.Sum(mod.TypeBase, nil, ally.fallback)
				}
				if o := modDB.Override(nil, ally.life); o != nil {
					life = o.Float()
				}
				actor.Output[ally.life] = life
			}
		}

		for _, ally := range allyLifePoolList {
			if !modDB.Flag(nil, "MinionLifeShares"+ally.life) {
				continue
			}
			specificOverride := modDB.Override(nil, ally.life)
			if modDB.Override(nil, "TotalMinionLife") == nil && specificOverride != nil {
				actor.Output["TotalMinionLife"] = specificOverride.Float()
			} else if actor.Output["TotalMinionLife"] == 0 {
				actor.Output["TotalMinionLife"] = actor.Output[ally.life]
			}
			actor.Output["MinionAllyDamageMitigation"] += actor.Output[ally.mitigation]
			actor.Output[ally.mitigation] = 0
			delete(actor.Output, ally.life)
		}

		//pob:block:src/Modules/CalcDefence.lua:2493-2498
		if actor.Output["TotemAllyDamageMitigation"] > 0 && actor.Output["TotalTotemLife"] == 0 && actor.Output["TotalVaalRejuvenationTotemLife"] > 0 {
			actor.Output["VaalRejuvenationTotemAllyDamageMitigation"] += actor.Output["TotemAllyDamageMitigation"]
			actor.Output["TotemAllyDamageMitigation"] = 0
			delete(actor.Output, "TotalTotemLife")
		}
	}

	// Vaal Arctic Armour
	//pob:block:src/Modules/CalcDefence.lua:2529-2533
	actor.Output["VaalArcticArmourLife"] = modDB.Sum(mod.TypeBase, nil, "VaalArcticArmourMaxHits")
	actor.Output["VaalArcticArmourMitigation"] = math.Min(-modDB.Sum(mod.TypeMore, nil, "VaalArcticArmourMitigation")/100, 1)

	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"ManaEffectiveLife")]
		actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"MoMHitPool")]
		manatext := "Mana"
		if actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
			if modDB.Flag(nil, "EnergyShieldProtectsMana") {
				manatext = manatext + " and non-bypassed Energy Shield"
			} else {
				if actor.Output[string(damageType+"EnergyShieldBypass")] > 0 {
					poolProtected := float64(actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100))
					actor.Output[string(damageType+"TotalPool")] = max(actor.Output[string(damageType+"TotalPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalPool")], poolProtected)/(actor.Output[string(damageType+"EnergyShieldBypass")]/100)
					actor.Output[string(damageType+"TotalHitPool")] = max(actor.Output[string(damageType+"TotalHitPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalHitPool")], poolProtected)/(actor.Output[string(damageType+"EnergyShieldBypass")]/100)
				} else {
					actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"TotalPool")] + actor.Output["EnergyShieldRecoveryCap"]
					actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] + actor.Output["EnergyShieldRecoveryCap"]
				}
			}
		}
		//pob:block:src/Modules/CalcDefence.lua:2554-2565
		if actor.Breakdown != nil {
			actor.Breakdown.Data[string(damageType+"TotalPool")] = &Entry{Lines: []string{
				fmt.Sprintf("Life: %d", int(actor.Output["LifeRecoverable"])),
			}}
			if actor.Output[string(damageType+"ManaEffectiveLife")] != actor.Output["LifeRecoverable"] {
				actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("%s through MoM: %d", manatext, int(actor.Output[string(damageType+"ManaEffectiveLife")]-actor.Output["LifeRecoverable"])))
			}
			if (!modDB.Flag(nil, "EnergyShieldProtectsMana")) && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
				actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("Non-bypassed Energy Shield: %d", int(actor.Output[string(damageType+"TotalPool")]-actor.Output[string(damageType+"ManaEffectiveLife")])))
			}
			actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("TotalPool: %d", int(actor.Output[string(damageType+"TotalPool")])))
		}
	}

	// helper function that iteratively reduces pools until life hits 0 to determine the number of hits it would take with given damage to die
	//pob:block:src/Modules/CalcDefence.lua:2568-2733
	wardAvoidBreakChance := math.Min(modDB.Sum(mod.TypeBase, nil, "WardAvoidBreakChance")/100, 1)
	if modDB.Flag(nil, "Condition:WardNotBreak") {
		wardAvoidBreakChance = 1
	}
	var numberOfHitsToDie func(DamageIn map[string]float64) float64
	numberOfHitsToDie = func(DamageIn map[string]float64) float64 {
		numHits := float64(0)
		if DamageIn["cycles"] == 0 {
			DamageIn["cycles"] = 1
		}

		for _, damageType := range data.DamageType("").Values() {
			numHits = numHits + DamageIn[string(damageType)]
		}
		if numHits == 0 {
			return math.Inf(1)
		} else if wardAvoidBreakChance == 1 && actor.Output["Ward"] > 0 && numHits < actor.Output["Ward"] {
			return math.Inf(1)
		} else {
			numHits = 0
		}

		life := math.Max(actor.Output["LifeRecoverable"], 0)
		mana := math.Max(actor.Output["ManaUnreserved"], 0)
		energyShield := math.Max(actor.Output["EnergyShieldRecoveryCap"], 0)
		ward := math.Max(actor.Output["Ward"], 0)
		wardActiveChance := float64(0)
		if ward > 0 {
			wardActiveChance = 1
		}
		if wardAvoidBreakChance < 1 && DamageIn["cycles"] > 1 {
			ward = 0
			wardActiveChance = 0
		}
		allyRemaining := make(map[string]float64, len(allyLifePoolList))
		for _, ally := range allyLifePoolList {
			if life, mitigation := actor.Output[ally.life], actor.Output[ally.mitigation]; life > 0 && mitigation > 0 {
				allyRemaining[ally.key] = life
			}
		}
		aegis := make(map[string]float64)
		aegis["shared"] = actor.Output["sharedAegis"]
		aegis["sharedElemental"] = actor.Output["sharedElementalAegis"]
		guard := make(map[string]float64)
		guard["shared"] = actor.Output["sharedGuardAbsorb"]
		for _, damageType := range data.DamageType("").Values() {
			aegis[string(damageType)] = actor.Output[string(damageType+"Aegis")]
			guard[string(damageType)] = actor.Output[string(damageType+"GuardAbsorb")]
		}
		if DamageIn["cycles"] != 1 {
			DamageIn["TrackPoolLoss"] = 0
			DamageIn["TrackLifeLossOverTime"] = 0
		}
		DamageIn["WardBypass"] = utils.OrF(DamageIn["WardBypass"], modDB.Sum(mod.TypeBase, nil, "WardBypass"))
		wardBypassBelow := modDB.Sum(mod.TypeBase, nil, "WardBypassBelowPercent") / 100
		lifeLossBelowHalfPrevented := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")

		vaalArcticArmourHitsLeft := actor.Output["VaalArcticArmourLife"]
		if DamageIn["cycles"] > 1 {
			vaalArcticArmourHitsLeft = 0
		}

		itterationMultiplier := float64(1)
		damageTotal := float64(0)
		maxDamage := float64(data.EhpCalcMaxDamage)
		maxIterations := float64(data.EhpCalcMaxIterationsToCalc)
		lifeLossLostOverTime := actor.Output["LifeLossLostOverTime"]
		lifeBelowHalfLossLostOverTime := actor.Output["LifeBelowHalfLossLostOverTime"]
		damageTakenThatCanBeRecouped := make(map[data.DamageType]float64, 5)
		hitOverkill := float64(0)
		for life > 0 && DamageIn["iterations"] < maxIterations {
			DamageIn["iterations"] = DamageIn["iterations"] + 1
			Damage := make(map[data.DamageType]float64)
			damageTotal = 0
			hitOverkill = 0
			//pob:src/Modules/CalcDefence.lua:193-197
			//	local damage = DamageIn[damageType] or 0
			vaalArcticArmourMultiplier := float64(1)
			if vaalArcticArmourHitsLeft > 0 {
				vaalArcticArmourMultiplier = 1 - float64(actor.Output["VaalArcticArmourMitigation"]*
					math.Min(vaalArcticArmourHitsLeft/itterationMultiplier, 1))
			}
			vaalArcticArmourHitsLeft = vaalArcticArmourHitsLeft - itterationMultiplier
			for _, damageType := range data.DamageType("").Values() {
				d := DamageIn[string(damageType)] * itterationMultiplier * vaalArcticArmourMultiplier
				if d > 0 {
					d = math.Ceil(d)
				} else {
					d = 0
				}
				Damage[damageType] = d
				damageTotal = damageTotal + DamageIn[string(damageType)]
			}
			_, missingLifeSet := DamageIn["MissingLifeBeforeEnemyHit"]
			_, missingManaSet := DamageIn["MissingManaBeforeEnemyHit"]
			if (DamageIn["GainWhenHit"] != 0 || missingLifeSet || missingManaSet) &&
				(itterationMultiplier > 1 || DamageIn["cycles"] > 1) {
				gainMult := float64(itterationMultiplier * DamageIn["cycles"])
				if missingLifeSet {
					life = min(life+DamageIn["MissingLifeBeforeEnemyHit"]*(actor.Output["LifeUnreserved"]-life)*(gainMult-1)/100,
						gainMult*actor.Output["LifeRecoverable"])
				}
				if missingManaSet {
					mana = min(mana+DamageIn["MissingManaBeforeEnemyHit"]*(actor.Output["ManaUnreserved"]-mana)*(gainMult-1)/100,
						gainMult*actor.Output["ManaUnreserved"])
				}
				if DamageIn["GainWhenHit"] != 0 {
					life = min(life+float64(DamageIn["LifeWhenHit"]*(gainMult-1)), gainMult*(actor.Output["LifeRecoverable"]))
					mana = min(mana+float64(DamageIn["ManaWhenHit"]*(gainMult-1)), gainMult*(actor.Output["ManaUnreserved"]))
					energyShield = min(energyShield+float64(DamageIn["EnergyShieldWhenHit"]*(gainMult-1)), gainMult*actor.Output["EnergyShieldRecoveryCap"])
				}
			}
			//pob:block:src/Modules/CalcDefence.lua:2661-2666
			if missingLifeSet && life > 0 {
				life = min(life+DamageIn["MissingLifeBeforeEnemyHit"]*(actor.Output["LifeUnreserved"]-life)/100,
					actor.Output["LifeRecoverable"])
			}
			if missingManaSet && mana > 0 {
				mana = min(mana+DamageIn["MissingManaBeforeEnemyHit"]*(actor.Output["ManaUnreserved"]-mana)/100,
					actor.Output["ManaUnreserved"])
			}
			var traceStep *PoolWalkStep
			if environment.TracePoolWalk {
				snap := make(map[data.DamageType]float64, len(Damage))
				for dt, v := range Damage {
					if v != 0 {
						snap[dt] = v
					}
				}
				recoupIn := make(map[data.DamageType]float64, 5)
				for _, dt := range data.DamageType("").Values() {
					if v := damageTakenThatCanBeRecouped[dt]; v != 0 {
						recoupIn[dt] = v
					}
				}
				traceStep = &PoolWalkStep{
					N: len(environment.PoolWalkTrace) + 1, Damage: snap,
					LifeIn: life, ManaIn: mana, EnergyShieldIn: energyShield, WardIn: ward,
					RecoupableIn: recoupIn,
					AegisIn:      maps.Clone(aegis), GuardIn: maps.Clone(guard),
					AllyIn: maps.Clone(allyRemaining),
					Actor:  actor,
				}
			}
			mana = math.Max(mana, 0)
			energyShield = math.Max(energyShield, 0)
			ward = math.Max(ward, 0)
			wardBeforeHit := ward
			drainDamageTotal := float64(0)
			for _, damageType := range data.DamageType("").Values() {
				drainDamageTotal += Damage[damageType]
			}
			//pob:block:src/Modules/CalcDefence.lua:243-261
			lifeHitPoolInitial := calcLifeHitPoolWithLossPrevention(life, actor.Output["Life"],
				actor.Output["preventedLifeLoss"], lifeLossBelowHalfPrevented)
			momPoolsRemaining := make(map[data.DamageType]float64, 5)
			esPoolsRemaining := make(map[data.DamageType]float64, 5)
			for _, damageType := range data.DamageType("").Values() {
				momEffect := min(actor.Output["sharedMindOverMatter"]+actor.Output[string(damageType+"MindOverMatter")], 100) / 100
				maxMoM := math.Inf(1)
				if momEffect < 1 {
					maxMoM = lifeHitPoolInitial/(1-momEffect) - lifeHitPoolInitial
				}
				momPool := mana
				if momEffect < 1 {
					momPool = min(maxMoM, mana)
				}
				lifePlusMoMHitPool := lifeHitPoolInitial + momPool
				esBypass := actor.Output[string(damageType+"EnergyShieldBypass")] / 100
				esPool := energyShield
				if esBypass > 0 {
					esPool = min(lifePlusMoMHitPool/esBypass-lifePlusMoMHitPool, energyShield)
				}
				if modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
					esPoolsRemaining[damageType] = math.Floor(min(min(maxMoM*(1-esBypass), energyShield),
						lifePlusMoMHitPool/(1-float64((1-esBypass)*momEffect))-lifePlusMoMHitPool))
					momPoolsRemaining[damageType] = math.Floor(min(maxMoM-esPoolsRemaining[damageType], momPool))
				} else {
					esPoolsRemaining[damageType] = math.Floor(esPool)
					momPoolsRemaining[damageType] = math.Floor(momPool)
				}
			}
			for _, damageType := range data.DamageType("").Values() {
				if Damage[damageType] > 0 {
					for _, ally := range allyLifePoolList {
						remaining := allyRemaining[ally.key]
						percent := min(actor.Output[ally.mitigation], 100) / 100
						if remaining <= 0 || percent <= 0 {
							continue
						}
						tempDamage := min(Damage[damageType]*percent, remaining)
						allyRemaining[ally.key] = math.Floor(remaining - tempDamage)
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					damageTakenThatCanBeRecouped[damageType] += Damage[damageType]
					if aegis[string(damageType)] > 0 {
						tempDamage := min(Damage[damageType], aegis[string(damageType)])
						aegis[string(damageType)] = aegis[string(damageType)] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if isElemental[string(damageType)] && aegis["sharedElemental"] > 0 {
						tempDamage := min(Damage[damageType], aegis["sharedElemental"])
						aegis["sharedElemental"] = aegis["sharedElemental"] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if aegis["shared"] > 0 {
						tempDamage := min(Damage[damageType], aegis["shared"])
						aegis["shared"] = aegis["shared"] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if guard[string(damageType)] > 0 {
						tempDamage := min(Damage[damageType]*actor.Output[string(damageType+"GuardAbsorbRate")]/100, guard[string(damageType)])
						guard[string(damageType)] = math.Floor(guard[string(damageType)] - tempDamage)
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if guard["shared"] > 0 {
						tempDamage := min(Damage[damageType]*actor.Output["sharedGuardAbsorbRate"]/100, guard["shared"])
						guard["shared"] = math.Floor(guard["shared"] - tempDamage)
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if ward > 0 && (wardBypassBelow == 0 || drainDamageTotal >= wardBeforeHit*wardBypassBelow) {
						tempDamage := min(Damage[damageType]*(1-DamageIn["WardBypass"]/100), ward)
						ward = ward - tempDamage
						tempDamage = float64(tempDamage * wardActiveChance)
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if energyShield > 0 && (!modDB.Flag(nil, "EnergyShieldProtectsMana")) && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
						//pob:block:src/Modules/CalcDefence.lua:319-327
						tempDamage := min(Damage[damageType]*(1-actor.Output[string(damageType+"EnergyShieldBypass")]/100), esPoolsRemaining[damageType])
						for _, dt2 := range data.DamageType("").Values() {
							esPoolsRemaining[dt2] = math.Max(esPoolsRemaining[dt2]-tempDamage, 0)
						}
						energyShield = energyShield - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if (actor.Output["sharedMindOverMatter"] + actor.Output[string(damageType+"MindOverMatter")]) > 0 {
						MoMDamage := math.Ceil(Damage[damageType] * min(actor.Output["sharedMindOverMatter"]+actor.Output[string(damageType+"MindOverMatter")], 100) / 100)
						//pob:block:src/Modules/CalcDefence.lua:328-347
						if modDB.Flag(nil, "EnergyShieldProtectsMana") && energyShield > 0 && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
							tempDamage := math.Ceil(min(MoMDamage*(1-actor.Output[string(damageType+"EnergyShieldBypass")]/100), esPoolsRemaining[damageType]))
							for _, dt2 := range data.DamageType("").Values() {
								esPoolsRemaining[dt2] = math.Floor(math.Max(esPoolsRemaining[dt2]-tempDamage, 0))
							}
							energyShield = energyShield - tempDamage
							MoMDamage = MoMDamage - tempDamage
							tempDamage2 := math.Ceil(min(MoMDamage, momPoolsRemaining[damageType]))
							for _, dt2 := range data.DamageType("").Values() {
								momPoolsRemaining[dt2] = math.Floor(math.Max(momPoolsRemaining[dt2]-tempDamage2, 0))
							}
							mana = mana - tempDamage2
							Damage[damageType] = Damage[damageType] - tempDamage - tempDamage2
						} else {
							tempDamage := math.Ceil(min(MoMDamage, momPoolsRemaining[damageType]))
							for _, dt2 := range data.DamageType("").Values() {
								momPoolsRemaining[dt2] = math.Floor(math.Max(momPoolsRemaining[dt2]-tempDamage, 0))
							}
							mana = mana - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
					}
					if actor.Output["preventedLifeLossTotal"] > 0 {
						halfLife := float64(actor.Output["Life"] * 0.5)
						lifeOverHalfLife := max(life-halfLife, 0)
						preventPercent := actor.Output["preventedLifeLoss"] / 100
						poolAboveLow := lifeOverHalfLife / (1 - preventPercent)
						preventBelowHalfPercent := lifeLossBelowHalfPrevented / 100
						damageThatLifeCanStillTake := poolAboveLow +
							max(min(life, halfLife), 0)/(1-preventBelowHalfPercent)/(1-preventPercent)
						if damageThatLifeCanStillTake < Damage[damageType] {
							hitOverkill = hitOverkill + Damage[damageType] - damageThatLifeCanStillTake
							Damage[damageType] = damageThatLifeCanStillTake
						}
						if actor.Output["preventedLifeLossBelowHalf"] != 0 {
							damageToSplit := min(Damage[damageType], poolAboveLow)
							lostLife := float64(damageToSplit * (1 - preventPercent))
							preventedLoss := float64(damageToSplit * preventPercent)
							Damage[damageType] = Damage[damageType] - damageToSplit
							lifeLossLostOverTime = lifeLossLostOverTime + preventedLoss
							life = life - lostLife
							if life <= halfLife {
								unspecificallyLowLifePreventedDamage := float64(Damage[damageType] * preventPercent)
								lifeLossLostOverTime = lifeLossLostOverTime + unspecificallyLowLifePreventedDamage
								Damage[damageType] = Damage[damageType] - unspecificallyLowLifePreventedDamage
								specificallyLowLifePreventedDamage := float64(Damage[damageType] * preventBelowHalfPercent)
								lifeBelowHalfLossLostOverTime = lifeBelowHalfLossLostOverTime + specificallyLowLifePreventedDamage
								Damage[damageType] = Damage[damageType] - specificallyLowLifePreventedDamage
							}
						} else {
							tempDamage := float64(Damage[damageType] * preventPercent)
							lifeLossLostOverTime = lifeLossLostOverTime + tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						//pob:src/Modules/CalcDefence.lua:385-392
						if life > 0 {
							tempDamage := min(Damage[damageType], life)
							life = life - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						hitOverkill = hitOverkill + Damage[damageType]
					} else {
						life = life - Damage[damageType]
					}
				}
			}
			life = math.Floor(life)
			mana = math.Floor(mana)
			energyShield = math.Floor(energyShield)
			if actor.Output["preventedLifeLossTotal"] > 0 {
				lifeLossLostOverTime = math.Ceil(lifeLossLostOverTime)
				lifeBelowHalfLossLostOverTime = math.Ceil(lifeBelowHalfLossLostOverTime)
			}
			if life > 0 && damageTotal >= maxDamage {
				return math.Inf(1)
			}
			if ward < wardBeforeHit {
				ward = 0
				if wardAvoidBreakChance > 0 {
					ward = wardBeforeHit
				}
				wardActiveChance = wardActiveChance * wardAvoidBreakChance
			}
			ward = math.Floor(ward)
			if traceStep != nil {
				traceStep.LifeOut, traceStep.ManaOut = life, mana
				traceStep.EnergyShieldOut, traceStep.WardOut = energyShield, ward
				recoupOut := make(map[data.DamageType]float64, 5)
				for _, dt := range data.DamageType("").Values() {
					if v := damageTakenThatCanBeRecouped[dt]; v != 0 {
						recoupOut[dt] = v
					}
				}
				traceStep.RecoupableOut = recoupOut
				traceStep.AegisOut = maps.Clone(aegis)
				traceStep.GuardOut = maps.Clone(guard)
				traceStep.AllyOut = maps.Clone(allyRemaining)
				environment.PoolWalkTrace = append(environment.PoolWalkTrace, *traceStep)
			}
			if DamageIn["GainWhenHit"] != 0 && life > 0 {
				life = min(life+DamageIn["LifeWhenHit"], actor.Output["LifeRecoverable"])
				mana = min(mana+DamageIn["ManaWhenHit"], actor.Output["ManaUnreserved"])
				energyShield = min(energyShield+DamageIn["EnergyShieldWhenHit"], actor.Output["EnergyShieldRecoveryCap"])
			}
			itterationMultiplier = 1
			speedUp := float64(data.EhpCalcSpeedUp)
			if DamageIn["LimitEHPSpeedup"] != 0 {
				speedUp = 4
			}
			wardAvoidBreakActive := wardAvoidBreakChance < 1 && wardActiveChance > 0.01
			if DamageIn["cyclesRan"] == 0 && !wardAvoidBreakActive && life > 0 && DamageIn["iterations"] < maxIterations {
				DamageDown := make(map[string]float64)
				for _, damageType := range data.DamageType("").Values() {
					DamageDown[string(damageType)] = DamageIn[string(damageType)] * speedUp
				}
				if DamageIn["GainWhenHit"] != 0 {
					DamageDown["GainWhenHit"] = DamageIn["GainWhenHit"]
					DamageDown["LifeWhenHit"] = DamageIn["LifeWhenHit"]
					DamageDown["ManaWhenHit"] = DamageIn["ManaWhenHit"]
					DamageDown["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"]
				}
				DamageDown["LimitEHPSpeedup"] = DamageIn["LimitEHPSpeedup"]
				DamageDown["cycles"] = DamageIn["cycles"] * speedUp
				DamageDown["iterations"] = DamageIn["iterations"]
				itterationMultiplier = max(float64((numberOfHitsToDie(DamageDown)-1)*speedUp)-1, 1)
				// avoid unnecessary calculations if we know we survive infinite hits
				if math.IsInf(itterationMultiplier, 1) {
					return math.Inf(1)
				}
				DamageIn["iterations"] = DamageDown["iterations"]
				DamageIn["cyclesRan"] = 1
			}
			numHits = numHits + itterationMultiplier
		}
		//pob:block:src/Modules/CalcDefence.lua:2708-2712
		if DamageIn["TrackRecoupable"] != 0 {
			for _, damageType := range data.DamageType("").Values() {
				actor.Output[string(damageType)+"RecoupableDamageTaken"] += damageTakenThatCanBeRecouped[damageType]
			}
		}
		if DamageIn["TrackLifeLossOverTime"] != 0 {
			actor.Output["LifeLossLostOverTime"] += lifeLossLostOverTime
			actor.Output["LifeBelowHalfLossLostOverTime"] += lifeBelowHalfLossLostOverTime
		}
		if DamageIn["cycles"] == 1 {
			if actor.Output["preventedLifeLossTotal"] > 0 {
				if life <= 0 {
					numHits = numHits - math.Ceil(hitOverkill)/damageTotal
				}
			} else if life < 0 {
				numHits = numHits - math.Ceil(-life)/damageTotal
				life = 0
			}
		}
		// Recalculate total hit damage
		damageTotal = 0
		for _, damageType := range data.DamageType("").Values() {
			damageTotal = damageTotal + float64(DamageIn[string(damageType)]*numHits)
		}
		if life >= 0 && damageTotal >= maxDamage {
			return math.Inf(1)
		}
		if math.IsNaN(numHits) {
			return 0
		}
		return max(numHits, 0)
	}
	//pob:block:src/Modules/CalcDefence.lua:2735-2931
	if damageCategoryConfig != "DamageOverTime" {
		// number of damaging hits needed to be taken to die
		{
			DamageIn := make(map[string]float64)
			for _, damageType := range data.DamageType("").Values() {
				DamageIn[string(damageType)] = actor.Output[string(damageType+"TakenHit")]
			}
			if actor.Output["preventedLifeLossTotal"] > 0 {
				DamageIn["LimitEHPSpeedup"] = 1
			}
			actor.Output["NumberOfDamagingHits"] = numberOfHitsToDie(DamageIn)
		}

		{
			DamageIn := make(map[string]float64)
			BlockChance := float64(0)
			suppressChance := float64(0)
			ExtraAvoidChance := float64(0)
			averageAvoidChance := float64(0)
			worstOf := environment.Build.GetNumberOption("EHPUnluckyWorstOf")
			if worstOf == 0 {
				worstOf = 1
			}
			//pob:src/Modules/CalcDefence.lua:2748-2755
			if damageCategoryConfig == "Melee" || damageCategoryConfig == "Untyped" {
				BlockChance = actor.Output["EffectiveBlockChance"] / 100
			} else {
				BlockChance = actor.Output["Effective"+damageCategoryConfig+"BlockChance"] / 100
			}
			if actor.Enemy.ModDB.Flag(nil, "CannotBeBlocked") {
				BlockChance = 0
			}
			blockEffect := 1 - BlockChance*actor.Output["BlockEffect"]/100
			gainOnBlock := !environment.Build.GetBooleanOption("DisableEHPGainOnBlock") &&
				actor.Output["NumberOfDamagingHits"] > 1
			if gainOnBlock {
				DamageIn["LifeWhenHit"] = actor.Output["LifeOnBlock"] * BlockChance
				DamageIn["ManaWhenHit"] = actor.Output["ManaOnBlock"] * BlockChance
				DamageIn["EnergyShieldWhenHit"] = actor.Output["EnergyShieldOnBlock"] * BlockChance
				if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
					DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + float64(actor.Output["EnergyShieldOnSpellBlock"]*BlockChance)
				} else if damageCategoryConfig == "Average" {
					DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + float64(actor.Output["EnergyShieldOnSpellBlock"]/2*BlockChance)
				}
			}
			if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" || damageCategoryConfig == "Average" {
				suppressChance = actor.Output["EffectiveSpellSuppressionChance"] / 100
			}
			// We include suppression in damage reduction if it is 100% otherwise we handle it here.
			suppressionEffect := float64(1)
			if suppressChance < 1 {
				if damageCategoryConfig == "Average" {
					suppressChance = suppressChance / 2
				}
				DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + float64(actor.Output["EnergyShieldOnSuppress"]*suppressChance)
				DamageIn["LifeWhenHit"] = DamageIn["LifeWhenHit"] + float64(actor.Output["LifeOnSuppress"]*suppressChance)
				suppressionEffect = 1 - suppressChance*actor.Output["SpellSuppressionEffect"]/100
			} else {
				averageFactor := utils.Ternary(damageCategoryConfig == "Average", 0.5, float64(1))
				DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + float64(actor.Output["EnergyShieldOnSuppress"]*averageFactor)
				DamageIn["LifeWhenHit"] = DamageIn["LifeWhenHit"] + float64(actor.Output["LifeOnSuppress"]*averageFactor)
			}
			// extra avoid chance
			if damageCategoryConfig == "Projectile" || damageCategoryConfig == "SpellProjectile" {
				ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"]
			} else if damageCategoryConfig == "Average" {
				ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"]/2
			}
			if gainOnBlock {
				DamageIn["MissingLifeBeforeEnemyHit"] = modDB.Sum(mod.TypeBase, nil, "MissingLifeBeforeEnemyHit")
				DamageIn["MissingManaBeforeEnemyHit"] = modDB.Sum(mod.TypeBase, nil, "MissingManaBeforeEnemyHit")
				if DamageIn["LifeWhenHit"] != 0 || DamageIn["ManaWhenHit"] != 0 || DamageIn["EnergyShieldWhenHit"] != 0 ||
					DamageIn["MissingLifeBeforeEnemyHit"] != 0 || DamageIn["MissingManaBeforeEnemyHit"] != 0 {
					DamageIn["GainWhenHit"] = 1
				}
			} else {
				DamageIn["LifeWhenHit"] = 0
				DamageIn["ManaWhenHit"] = 0
				DamageIn["EnergyShieldWhenHit"] = 0
			}
			for _, damageType := range data.DamageType("").Values() {
				if modDB.Flag(nil, "BlockedDamageDoesntBypassES") && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 && damageType != "Chaos" {
					DamageIn[string(damageType+"EnergyShieldBypass")] = actor.Output[string(damageType+"EnergyShieldBypass")] * (1 - BlockChance)
				}
				AvoidChance := float64(0)
				if specificTypeAvoidance {
					AvoidChance = min(actor.Output[string("Avoid"+damageType+"DamageChance")]+ExtraAvoidChance, data.AvoidChanceCap)
					// unlucky config to lower the value of block, dodge, evade etc for ehp
					if worstOf > 1 {
						AvoidChance = AvoidChance / 100 * AvoidChance
						if worstOf == 4 {
							AvoidChance = AvoidChance / 100 * AvoidChance
						}
					}
					averageAvoidChance = averageAvoidChance + AvoidChance
				}
				DamageIn[string(damageType)] = actor.Output[string(damageType+"TakenHit")] * (blockEffect * suppressionEffect * (1 - AvoidChance/100))
			}
			// taken over time degen initialisation
			if actor.Output["preventedLifeLossTotal"] > 0 {
				DamageIn["TrackLifeLossOverTime"] = 1
				actor.Output["LifeLossLostOverTime"] = 0
				actor.Output["LifeBelowHalfLossLostOverTime"] = 0
			}
			averageAvoidChance = averageAvoidChance / 5
			actor.Output["ConfiguredDamageChance"] = 100 * (blockEffect * suppressionEffect * (1 - averageAvoidChance/100))
			if actor.Output["anyRecoup"] > 0 {
				DamageIn["TrackRecoupable"] = 1
				for _, damageType := range data.DamageType("").Values() {
					actor.Output[string(damageType)+"RecoupableDamageTaken"] = 0
				}
			}
			if DamageIn["TrackRecoupable"] != 0 || DamageIn["TrackLifeLossOverTime"] != 0 || DamageIn["GainWhenHit"] != 0 {
				DamageIn["LimitEHPSpeedup"] = 1
			}
			if actor.Output["ConfiguredDamageChance"] != 100 || DamageIn["TrackRecoupable"] != 0 ||
				DamageIn["TrackLifeLossOverTime"] != 0 || DamageIn["GainWhenHit"] != 0 {
				actor.Output["NumberOfMitigatedDamagingHits"] = numberOfHitsToDie(DamageIn)
			} else {
				actor.Output["NumberOfMitigatedDamagingHits"] = actor.Output["NumberOfDamagingHits"]
			}
			//pob:block:src/Modules/CalcDefence.lua:2841-2855
			if actor.Breakdown != nil {
				actor.Breakdown.Data["ConfiguredDamageChance"] = &Entry{Lines: []string{
					fmt.Sprintf("%.2f ^8(chance for block to fail)", 1-BlockChance),
				}}
				if actor.Output["ShowBlockEffect"] != 0 {
					actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.2f ^8(block effect)", actor.Output["BlockEffect"]/100))
				}
				if suppressionEffect != 1 {
					actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.3f ^8(suppression effect)", suppressionEffect))
				}
				if averageAvoidChance > 0 {
					actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.2f ^8(chance for avoidance to fail)", 1-averageAvoidChance/100))
				}
				actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("= %.1f%% ^8(of damage taken from a%s hit)", actor.Output["ConfiguredDamageChance"], utils.Ternary(damageCategoryConfig == "Average" || damageCategoryConfig == "Untyped", "n ", " ")+damageCategoryConfig))
			}
		}
		{
			worstOf := environment.Build.GetNumberOption("EHPUnluckyWorstOf")
			if worstOf == 0 {
				worstOf = 1
			}
			avoidAll := 1 - actor.Output["AvoidAllDamageFromHitsChance"]/100
			avoidProjectiles := 1 - utils.Ternary(specificTypeAvoidance, float64(0), actor.Output["AvoidProjectilesChance"])/100
			actor.Output["MeleeNotHitChance"] = 100 - float64((1-actor.Output["MeleeEvadeChance"]/100)*(1-actor.Output["EffectiveAttackDodgeChance"]/100)*avoidAll*100)
			actor.Output["ProjectileNotHitChance"] = 100 - float64((1-actor.Output["ProjectileEvadeChance"]/100)*(1-actor.Output["EffectiveAttackDodgeChance"]/100)*avoidAll*avoidProjectiles*100)
			actor.Output["SpellNotHitChance"] = 100 - float64((1-actor.Output["EffectiveSpellDodgeChance"]/100)*avoidAll*100)
			actor.Output["SpellProjectileNotHitChance"] = 100 - float64((1-actor.Output["EffectiveSpellDodgeChance"]/100)*avoidAll*avoidProjectiles*100)
			//pob:src/Modules/CalcDefence.lua:1652
			actor.Output["UntypedNotHitChance"] = 100 - float64(avoidAll*100)
			actor.Output["AverageNotHitChance"] = (actor.Output["MeleeNotHitChance"] + actor.Output["ProjectileNotHitChance"] + actor.Output["SpellNotHitChance"] + actor.Output["SpellProjectileNotHitChance"]) / 4
			actor.Output["ConfiguredNotHitChance"] = actor.Output[damageCategoryConfig+"NotHitChance"]
			// unlucky config to lower the value of block, dodge, evade etc for ehp
			if worstOf > 1 {
				actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
				if worstOf == 4 {
					actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
				}
			}
			actor.Output["TotalNumberOfHits"] = actor.Output["NumberOfMitigatedDamagingHits"] / (1 - actor.Output["ConfiguredNotHitChance"]/100)
			//pob:block:src/Modules/CalcDefence.lua:2861-2904
			if actor.Breakdown != nil {
				const key = "ConfiguredNotHitChance"
				actor.Breakdown.Data[key] = &Entry{}
				if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
					if actor.Output[damageCategoryConfig+"EvadeChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("%.2f ^8(chance for evasion to fail)", 1-actor.Output[damageCategoryConfig+"EvadeChance"]/100))
					}
					if actor.Output["AttackDodgeChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("x %.2f ^8(chance for dodge to fail)", 1-actor.Output["AttackDodgeChance"]/100))
					}
					if damageCategoryConfig == "Projectile" && !specificTypeAvoidance && actor.Output["AvoidProjectilesChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("x %.2f ^8(chance for avoidance to fail)", 1-actor.Output["AvoidProjectilesChance"]/100))
					}
				} else if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
					if actor.Output["SpellDodgeChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("%.2f ^8(chance for dodge to fail)", 1-actor.Output["SpellDodgeChance"]/100))
					}
					if damageCategoryConfig == "SpellProjectile" && !specificTypeAvoidance && actor.Output["AvoidProjectilesChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("x %.2f ^8(chance for avoidance to fail)", 1-actor.Output["AvoidProjectilesChance"]/100))
					}
				} else if damageCategoryConfig == "Average" {
					if actor.Output["MeleeEvadeChance"] > 0 || actor.Output["ProjectileEvadeChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("%.2f ^8(chance for evasion to fail, only applies to the attack portion)", 1-(actor.Output["MeleeEvadeChance"]+actor.Output["ProjectileEvadeChance"])/2/100))
					}
					if actor.Output["AttackDodgeChance"] > 0 || actor.Output["SpellDodgeChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("x%.2f ^8(chance for dodge to fail)", 1-(actor.Output["AttackDodgeChance"]+actor.Output["SpellDodgeChance"])/2/100))
					}
					if !specificTypeAvoidance && actor.Output["AvoidProjectilesChance"] > 0 {
						actor.Breakdown.AddLine(key, fmt.Sprintf("x %.2f ^8(chance for avoidance to fail)", 1-actor.Output["AvoidProjectilesChance"]/2/100))
					}
				}
				if actor.Output["AvoidAllDamageFromHitsChance"] > 0 {
					actor.Breakdown.AddLine(key, fmt.Sprintf("x %.2f ^8(chance for damage avoidance to fail)", 1-actor.Output["AvoidAllDamageFromHitsChance"]/100))
				}
				if worstOf > 1 {
					actor.Breakdown.AddLine(key, fmt.Sprintf("unlucky worst of %d", int(worstOf)))
				}
				actor.Breakdown.AddLine(key, fmt.Sprintf("= %d%% ^8(chance to be hit by a%s hit)", int(100-actor.Output["ConfiguredNotHitChance"]), utils.Ternary(damageCategoryConfig == "Average" || damageCategoryConfig == "Untyped", "n ", " ")+damageCategoryConfig))
				actor.Breakdown.Data["TotalNumberOfHits"] = &Entry{Lines: []string{
					fmt.Sprintf("%.2f ^8(Number of mitigated hits)", actor.Output["NumberOfMitigatedDamagingHits"]),
					fmt.Sprintf("/ %.2f ^8(Chance to even be hit)", 1-actor.Output["ConfiguredNotHitChance"]/100),
					fmt.Sprintf("= %.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
				}}
			}
		}

		// effective hit pool
		actor.Output["TotalEHP"] = actor.Output["TotalNumberOfHits"] * actor.Output["totalEnemyDamageIn"]
		//pob:block:src/Modules/CalcDefence.lua:2909-2915
		if actor.Breakdown != nil {
			actor.Breakdown.Data["TotalEHP"] = &Entry{Lines: []string{
				fmt.Sprintf("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
				fmt.Sprintf("x %d ^8(total incoming damage)", int(actor.Output["totalEnemyDamageIn"])),
				fmt.Sprintf("= %d ^8(total damage you can take)", int(actor.Output["TotalEHP"])),
			}}
		}
		// survival time
		//pob:block:src/Modules/CalcDefence.lua:2917-2930
		{
			enemySkillTime, fromInput := enemyConfigNumberSource(environment, "enemySpeed")
			if !fromInput {
				enemySkillTime = utils.OrF(enemySkillTime, 700)
			}
			enemySkillTime = enemySkillTime / (1 + actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "Speed")/100)
			enemyActionSpeed := CalcActionSpeedMod(actor.Enemy)
			enemySkillTime = enemySkillTime / 1000 / enemyActionSpeed
			actor.Output["enemySkillTime"] = enemySkillTime
			actor.Output["EHPSurvivalTime"] = actor.Output["TotalNumberOfHits"] * enemySkillTime
			if actor.Breakdown != nil {
				actor.Breakdown.AddLine("EHPSurvivalTime",
					fmt.Sprintf("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
					fmt.Sprintf("x %.2f ^8enemy attack/cast time", enemySkillTime),
					fmt.Sprintf("= %.2f seconds ^8(total time it would take to die)", actor.Output["EHPSurvivalTime"]),
				)
			}
		}
	}

	if actor.Output["anyRecoup"] > 0 && damageCategoryConfig != "DamageOverTime" {
		totalDamage := float64(0)
		totalElementalDamage := float64(0)
		totalPhysicalDamageMitigated := actor.Output["NumberOfMitigatedDamagingHits"] *
			(actor.Output["PhysicalTakenDamage"] - actor.Output["PhysicalTakenHit"])
		type pseudoExtra struct{ amount, duration float64 }
		extraPseudoRecoup := map[string]pseudoExtra{}

		for _, dt := range data.DamageType("").Values() {
			damageType := string(dt)
			totalDamage += actor.Output[damageType+"RecoupableDamageTaken"]
			if isElemental[damageType] {
				totalElementalDamage += actor.Output[damageType+"RecoupableDamageTaken"]
			}
		}

		recoupTypeList := []string{"Life", "Mana", "EnergyShield"}
		for i, recoupType := range recoupTypeList {
			recoupTime := float64(4)
			if modDB.Flag(nil, "3Second"+recoupType+"Recoup") || modDB.Flag(nil, "3SecondRecoup") {
				recoupTime = 3
			}

			total := float64(actor.Output[recoupType+"Recoup"] / 100 * totalDamage)
			if actor.Output["Elemental"+recoupType+"Recoup"] > 0 && totalElementalDamage > 0 {
				total += float64(actor.Output["Elemental"+recoupType+"Recoup"] / 100 * totalElementalDamage)
			}
			for _, dt := range data.DamageType("").Values() {
				damageType := string(dt)
				if actor.Output[damageType+recoupType+"Recoup"] > 0 && actor.Output[damageType+"RecoupableDamageTaken"] > 0 {
					total += float64(actor.Output[damageType+recoupType+"Recoup"] / 100 * actor.Output[damageType+"RecoupableDamageTaken"])
				}
			}
			actor.Output["Total"+recoupType+"RecoupRecovery"] = total

			pseudo := actor.Output["PhysicalDamageMitigated"+recoupType+"PseudoRecoup"] / 100 * totalPhysicalDamageMitigated
			pseudoDuration := actor.Output["PhysicalDamageMitigated"+recoupType+"PseudoRecoupDuration"]
			if pseudoDuration == 0 {
				pseudoDuration = 4
			}
			if pseudo != 0 {
				for j := i + 1; j < len(recoupTypeList); j++ {
					other := recoupTypeList[j]
					if modDB.Flag(nil, recoupType+"RegenerationRecovers"+other) &&
						!modDB.Flag(nil, "UnaffectedBy"+other+"Regen") &&
						!modDB.Flag(nil, "No"+other+"Regen") &&
						!modDB.Flag(nil, "CannotGain"+other) {
						extraPseudoRecoup[other] = pseudoExtra{
							amount:   pseudo * actor.Output[other+"RecoveryRateMod"] / actor.Output[recoupType+"RecoveryRateMod"],
							duration: pseudoDuration,
						}
					}
				}
			}
			if modDB.Flag(nil, "UnaffectedBy"+recoupType+"Regen") {
				pseudo = 0
			}
			actor.Output["Total"+recoupType+"PseudoRecoup"] = pseudo

			extraMax, extraAvg := float64(0), float64(0)
			if e, ok := extraPseudoRecoup[recoupType]; ok {
				extraMax = e.amount / e.duration
				extraAvg = e.amount / (actor.Output["EHPSurvivalTime"] + e.duration)
			}
			actor.Output[recoupType+"RecoupRecoveryMax"] = total/recoupTime + pseudo/pseudoDuration + extraMax
			actor.Output[recoupType+"RecoupRecoveryAvg"] = total/(actor.Output["EHPSurvivalTime"]+recoupTime) +
				pseudo/(actor.Output["EHPSurvivalTime"]+pseudoDuration) + extraAvg
			//pob:block:src/Modules/CalcDefence.lua:2970-3018
			if actor.Breakdown != nil {
				maxKey, avgKey := recoupType+"RecoupRecoveryMax", recoupType+"RecoupRecoveryAvg"
				var lines []string
				multipleTypes := 0
				if actor.Output[recoupType+"Recoup"] > 0 {
					lines = append(lines,
						fmt.Sprintf("%d ^8(total damage taken during ehp calcs)", int(totalDamage)),
						fmt.Sprintf("* %.2f ^8(percent of damage recouped)", actor.Output[recoupType+"Recoup"]/100),
					)
					multipleTypes++
				}
				if actor.Output["Elemental"+recoupType+"Recoup"] > 0 && totalElementalDamage > 0 {
					if multipleTypes > 0 {
						lines = append(lines, "")
					}
					lines = append(lines,
						fmt.Sprintf("%s%d ^8(total elemental damage taken during ehp calcs)", utils.Ternary(multipleTypes > 0, "+", ""), int(totalDamage)),
						fmt.Sprintf("* %.2f ^8(percent of damage recouped)", actor.Output["Elemental"+recoupType+"Recoup"]/100),
					)
					multipleTypes++
				}
				for _, dt := range data.DamageType("").Values() {
					damageType := string(dt)
					if actor.Output[damageType+recoupType+"Recoup"] > 0 && actor.Output[damageType+"RecoupableDamageTaken"] > 0 {
						if multipleTypes > 0 {
							lines = append(lines, "")
						}
						lines = append(lines,
							fmt.Sprintf("%s%d ^8(total %s damage taken during ehp calcs)", utils.Ternary(multipleTypes > 0, "+", ""), int(totalDamage), damageType),
							fmt.Sprintf("* %.2f ^8(percent of damage recouped)", actor.Output[damageType+recoupType+"Recoup"]/100),
						)
						multipleTypes++
					}
				}
				lines = append(lines, fmt.Sprintf("= %d ^8(total damage recoup amount)", int(total)))
				avg := slices.Clone(lines)
				lines = append(lines, fmt.Sprintf("/ %.2f ^8(over %d seconds)", recoupTime, int(recoupTime)))
				if pseudo > 0 {
					lines = append(lines,
						fmt.Sprintf("+ %.2f ^8(total damage mitigated pseudo recoup amount)", pseudo),
						fmt.Sprintf("/ %.2f ^8(over %d seconds)", pseudoDuration, int(pseudoDuration)),
					)
				}
				if e, ok := extraPseudoRecoup[recoupType]; ok {
					lines = append(lines,
						fmt.Sprintf("+ %.2f ^8(total damage mitigated pseudo recoup amount)", e.amount),
						fmt.Sprintf("/ %.2f ^8(over %d seconds)", e.duration, int(e.duration)),
					)
				}
				lines = append(lines, fmt.Sprintf("= %.2f per second ^8", actor.Output[maxKey]))
				avg = append(avg, fmt.Sprintf("/ %.2f ^8(total time of the recoup (survival time + %d seconds))", actor.Output["EHPSurvivalTime"]+recoupTime, int(recoupTime)))
				if pseudo > 0 {
					avg = append(avg,
						fmt.Sprintf("+ %.2f ^8(total damage mitigated pseudo recoup amount)", pseudo),
						fmt.Sprintf("/ %.2f ^8(total time of the recoup (survival time + %d seconds)", actor.Output["EHPSurvivalTime"]+pseudoDuration, int(pseudoDuration)),
					)
				}
				if e, ok := extraPseudoRecoup[recoupType]; ok {
					avg = append(avg,
						fmt.Sprintf("+ %.2f ^8(total damage mitigated pseudo recoup amount)", e.amount),
						fmt.Sprintf("/ %.2f ^8(total time of the recoup (survival time + %d seconds)", actor.Output["EHPSurvivalTime"]+e.duration, int(e.duration)),
					)
				}
				avg = append(avg, fmt.Sprintf("= %.2f per second ^8", actor.Output[avgKey]))
				actor.Breakdown.Data[maxKey] = &Entry{Lines: lines}
				actor.Breakdown.Data[avgKey] = &Entry{Lines: avg}
			}
		}
	}

	// petrified blood "degen"
	if actor.Output["preventedLifeLossTotal"] > 0 {
		LifeLossBelowHalfLost := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
		lost := actor.Output["LifeLossLostOverTime"] + float64(actor.Output["LifeBelowHalfLossLostOverTime"]*LifeLossBelowHalfLost)
		actor.Output["LifeLossLostMax"] = lost / 4
		actor.Output["LifeLossLostAvg"] = lost / (actor.Output["EHPSurvivalTime"] + 4)
		if actor.Breakdown != nil {
			for _, k := range []string{"LifeLossLostMax", "LifeLossLostAvg"} {
				if actor.Output["LifeLossLostOverTime"] != 0 {
					actor.Breakdown.AddLine(k, fmt.Sprintf("( %.0f ^8(total damage prevented by Progenesis)", actor.Output["LifeLossLostOverTime"]))
				}
				if actor.Output["LifeBelowHalfLossLostOverTime"] != 0 {
					actor.Breakdown.AddLine(k,
						fmt.Sprintf("%s %.0f ^8(total damage prevented by petrified blood)",
							utils.Ternary(actor.Output["LifeLossLostOverTime"] != 0, "+", "("), actor.Output["LifeBelowHalfLossLostOverTime"]),
						fmt.Sprintf("* %.2f ^8(percent of damage taken from petrified blood)", LifeLossBelowHalfLost),
					)
				}
			}
			actor.Breakdown.AddLine("LifeLossLostMax",
				fmt.Sprintf(") / %.2f ^8(over 4 seconds)", float64(4)),
				fmt.Sprintf("= %.2f per second", actor.Output["LifeLossLostMax"]),
			)
			actor.Breakdown.AddLine("LifeLossLostAvg",
				fmt.Sprintf(") / %.2f ^8(total time of the degen (survival time + 4 seconds))", actor.Output["EHPSurvivalTime"]+4),
				fmt.Sprintf("= %.2f per second", actor.Output["LifeLossLostAvg"]),
			)
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:3051-3056
	if actor.Output["LifeRecoupRecoveryAvg"] > 0 || actor.Output["preventedLifeLossTotal"] > 0 {
		actor.Output["netLifeRecoupAndLossLostOverTimeMax"] = actor.Output["LifeRecoupRecoveryMax"] - actor.Output["LifeLossLostMax"]
		actor.Output["netLifeRecoupAndLossLostOverTimeAvg"] = actor.Output["LifeRecoupRecoveryAvg"] - actor.Output["LifeLossLostAvg"]
		if actor.Output["LifeRecoupRecoveryAvg"] > 0 && actor.Output["preventedLifeLossTotal"] > 0 {
			actor.Output["showNetRecoup"] = 1
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:3073-3118
	if environment.Build.GetBooleanOption("PvpScaling") {
		PvpTvalue := actor.Output["enemySkillTime"]
		enemyMultiplierPvpDamage, ok := environment.Build.GetNumberOptionOK("enemyMultiplierPvpDamage")
		if !ok {
			enemyMultiplierPvpDamage = 100
		}
		PvpMultiplier := enemyMultiplierPvpDamage / 100

		PvpNonElemental1 := float64(data.PvpNonElemental1)
		PvpNonElemental2 := float64(data.PvpNonElemental2)
		PvpElemental1 := float64(data.PvpElemental1)
		PvpElemental2 := float64(data.PvpElemental2)

		percentageNonElemental := (actor.Output["PhysicalTakenHit"] + actor.Output["ChaosTakenHit"]) / actor.Output["totalTakenHit"]
		percentageElemental := 1 - percentageNonElemental
		portionNonElemental := float64(math.Pow(actor.Output["totalTakenHit"]/PvpTvalue/PvpNonElemental2, PvpNonElemental1) * PvpTvalue * PvpNonElemental2 * percentageNonElemental)
		portionElemental := float64(math.Pow(actor.Output["totalTakenHit"]/PvpTvalue/PvpElemental2, PvpElemental1) * PvpTvalue * PvpElemental2 * percentageElemental)
		actor.Output["PvPTotalTakenHit"] = (portionNonElemental + portionElemental) * PvpMultiplier

		if actor.Breakdown != nil {
			percentBoth := percentageNonElemental > 0 && percentageElemental > 0
			actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("Pvp Formula is (D/(T*M))^E*T*%s, where D is the damage, T is the time taken,", utils.Ternary(percentBoth, "M*P", "M")))
			actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf(" M is the multiplier%s", utils.Ternary(percentBoth, ", E is the exponent and P is the percentage of that type (ele or non ele)", " and E is the exponent")))
			if percentBoth {
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(M= %.1f for ele and %.1f for non-ele)(E= %.2f for ele and %.2f for non-ele)", PvpElemental2, PvpNonElemental2, PvpElemental1, PvpNonElemental1))
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", actor.Output["totalTakenHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, percentageNonElemental, portionNonElemental))
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", actor.Output["totalTakenHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, percentageElemental, portionElemental))
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(portionNonElemental + portionElemental)%s", utils.Ternary(PvpMultiplier != 1, " * PvP multiplier", " ")))
				if PvpMultiplier != 1 {
					actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(%.1f + %.1f) * %g", portionNonElemental, portionElemental, PvpMultiplier))
				} else {
					actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("%.1f + %.1f", portionNonElemental, portionElemental))
				}
			} else if percentageElemental <= 0 {
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(M= %.1f for non-ele)(E= %.2f for non-ele)", PvpNonElemental2, PvpNonElemental1))
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", actor.Output["totalTakenHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, portionNonElemental))
				if PvpMultiplier != 1 {
					actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("%.1f * %g ^8(portionNonElemental * PvP multiplier)", portionNonElemental, PvpMultiplier))
				}
			} else if percentageNonElemental <= 0 {
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(M= %.1f for ele)(E= %.2f for ele)", PvpElemental2, PvpElemental1))
				actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", actor.Output["totalTakenHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, portionElemental))
				if PvpMultiplier != 1 {
					actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("%.1f * %g ^8(portionElemental * PvP multiplier)", portionElemental, PvpMultiplier))
				}
			}
			actor.Breakdown.AddLine("PvPTotalTakenHit", fmt.Sprintf("= %.1f", actor.Output["PvPTotalTakenHit"]))
		}
	}

	// effective health pool vs dots
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"DotEHP")] = actor.Output[string(damageType+"TotalPool")] / actor.Output[string(damageType+"TakenDotMult")]
		//pob:block:src/Modules/CalcDefence.lua:3330-3336
		if actor.Breakdown != nil {
			actor.Breakdown.Data[string(damageType+"DotEHP")] = &Entry{Lines: []string{
				fmt.Sprintf("Total Pool: %d", int(actor.Output[string(damageType+"TotalPool")])),
				fmt.Sprintf("Dot Damage Taken modifier: %.2f", actor.Output[string(damageType+"TakenDotMult")]),
				fmt.Sprintf("Total Effective Dot Pool: %d", int(actor.Output[string(damageType+"DotEHP")])),
			}}
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:3339-3391
	for _, damageType := range data.DamageType("").Values() {
		baseVal := modDB.Sum(mod.TypeBase, nil, string(damageType+"Degen"))
		if baseVal <= 0 {
			continue
		}
		for _, damageConvertedType := range data.DamageType("").Values() {
			convertPercent := actor.DamageOverTimeShiftTable[damageType][damageConvertedType]
			if convertPercent <= 0 {
				continue
			}
			total := float64(baseVal * (convertPercent / 100) * actor.Output[string(damageConvertedType+"TakenDotMult")])
			actor.Output[string(damageConvertedType+"BuildDegen")] = actor.Output[string(damageConvertedType+"BuildDegen")] + total
			actor.Output["TotalBuildDegen"] = actor.Output["TotalBuildDegen"] + total
			if actor.Breakdown != nil {
				actor.Breakdown.AddCol("TotalBuildDegen",
					BCol{Label: "Base Type", Key: "type"},
					BCol{Label: "Final Type", Key: "type2"},
					BCol{Label: "Base", Key: "base"},
					BCol{Label: "Taken As Percent", Key: "conv"},
					BCol{Label: "Multiplier", Key: "mult"},
					BCol{Label: "Total", Key: "total"},
				)
				actor.Breakdown.AddRow("TotalBuildDegen", map[string]string{
					"type":  string(damageType),
					"type2": string(damageConvertedType),
					"base":  fmt.Sprintf("%.1f", baseVal),
					"conv":  fmt.Sprintf("x %.2f%%", convertPercent),
					"mult":  fmt.Sprintf("x %.2f", actor.Output[string(damageConvertedType+"TakenDotMult")]),
					"total": fmt.Sprintf("%.1f", total),
				})
				actor.Breakdown.AddCol(string(damageConvertedType+"BuildDegen"),
					BCol{Label: "Base Type", Key: "type"},
					BCol{Label: "Base", Key: "base"},
					BCol{Label: "Taken As Percent", Key: "conv"},
					BCol{Label: "Multiplier", Key: "mult"},
					BCol{Label: "Total", Key: "total"},
				)
				actor.Breakdown.AddRow(string(damageConvertedType+"BuildDegen"), map[string]string{
					"type":  string(damageType),
					"base":  fmt.Sprintf("%.1f", baseVal),
					"conv":  fmt.Sprintf("x %.2f%%", convertPercent),
					"mult":  fmt.Sprintf("x %.2f", actor.Output[string(damageConvertedType+"TakenDotMult")]),
					"total": fmt.Sprintf("%.1f", total),
				})
			}
		}
	}
	if _, totalDegenSet := actor.Output["TotalBuildDegen"]; totalDegenSet && actor.Output["TotalBuildDegen"] != 0 {
		actor.Output["NetLifeRegen"] = actor.Output["LifeRegenRecovery"]
		actor.Output["NetManaRegen"] = actor.Output["ManaRegenRecovery"]
		actor.Output["NetEnergyShieldRegen"] = actor.Output["EnergyShieldRegenRecovery"]
		totalLifeDegen := float64(0)
		totalManaDegen := float64(0)
		totalEnergyShieldDegen := float64(0)
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("NetLifeRegen", "Total Life Degen")
			actor.Breakdown.AddCol("NetLifeRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
			actor.Breakdown.SetLabel("NetManaRegen", "Total Mana Degen")
			actor.Breakdown.AddCol("NetManaRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
			actor.Breakdown.SetLabel("NetEnergyShieldRegen", "Total Energy Shield Degen")
			actor.Breakdown.AddCol("NetEnergyShieldRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
		}
		for _, damageType := range data.DamageType("").Values() {
			if _, degenSet := actor.Output[string(damageType+"BuildDegen")]; degenSet {
				energyShieldDegen := float64(0)
				lifeDegen := float64(0)
				manaDegen := float64(0)
				takenFromMana := actor.Output[string(damageType+"MindOverMatter")] + actor.Output["sharedMindOverMatter"]
				if actor.Output["EnergyShieldRegenRecovery"] > 0 {
					if modDB.Flag(nil, "EnergyShieldProtectsMana") {
						lifeDegen = actor.Output[string(damageType+"BuildDegen")] * (1 - takenFromMana/100)
						energyShieldDegen = actor.Output[string(damageType+"BuildDegen")] * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (takenFromMana / 100)
					} else {
						lifeDegen = actor.Output[string(damageType+"BuildDegen")] * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (1 - takenFromMana/100)
						energyShieldDegen = actor.Output[string(damageType+"BuildDegen")] * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100)
					}
					manaDegen = actor.Output[string(damageType+"BuildDegen")] * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (takenFromMana / 100)
				} else {
					lifeDegen = actor.Output[string(damageType+"BuildDegen")] * (1 - takenFromMana/100)
					manaDegen = actor.Output[string(damageType+"BuildDegen")] * (takenFromMana / 100)
				}
				totalLifeDegen = totalLifeDegen + lifeDegen
				totalManaDegen = totalManaDegen + manaDegen
				totalEnergyShieldDegen = totalEnergyShieldDegen + energyShieldDegen
				if actor.Breakdown != nil {
					actor.Breakdown.AddRow("NetLifeRegen", map[string]string{
						"type":  string(damageType),
						"degen": fmt.Sprintf("%.2f", lifeDegen),
					})
					actor.Breakdown.AddRow("NetManaRegen", map[string]string{
						"type":  string(damageType),
						"degen": fmt.Sprintf("%.2f", manaDegen),
					})
					actor.Breakdown.AddRow("NetEnergyShieldRegen", map[string]string{
						"type":  string(damageType),
						"degen": fmt.Sprintf("%.2f", energyShieldDegen),
					})
				}
			}
		}
		actor.Output["NetLifeRegen"] = actor.Output["NetLifeRegen"] - totalLifeDegen
		actor.Output["NetManaRegen"] = actor.Output["NetManaRegen"] - totalManaDegen
		actor.Output["NetEnergyShieldRegen"] = actor.Output["NetEnergyShieldRegen"] - totalEnergyShieldDegen
		actor.Output["TotalNetRegen"] = actor.Output["NetLifeRegen"] + actor.Output["NetManaRegen"] + actor.Output["NetEnergyShieldRegen"]
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("%.1f ^8(total life regen)", actor.Output["LifeRegen"]))
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("- %.1f ^8(total life degen)", totalLifeDegen))
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("= %.1f", actor.Output["NetLifeRegen"]))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("%.1f ^8(total mana regen)", actor.Output["ManaRegen"]))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("- %.1f ^8(total mana degen)", totalManaDegen))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("= %.1f", actor.Output["NetManaRegen"]))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("%.1f ^8(total energy shield regen)", actor.Output["EnergyShieldRegen"]))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("- %.1f ^8(total energy shield degen)", totalEnergyShieldDegen))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("= %.1f", actor.Output["NetEnergyShieldRegen"]))
			actor.Breakdown.AddLine("TotalNetRegen",
				fmt.Sprintf("Net Life Regen: %.1f", actor.Output["NetLifeRegen"]),
				fmt.Sprintf("+ Net Mana Regen: %.1f", actor.Output["NetManaRegen"]),
				fmt.Sprintf("+ Net Energy Shield Regen: %.1f", actor.Output["NetEnergyShieldRegen"]),
				fmt.Sprintf("= Total Net Regen: %.1f", actor.Output["TotalNetRegen"]),
			)
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:3487-3742
	enemyCritAilmentEffect := 1 + float64(actor.Output["EnemyCritChance"]/100*0.5*(1-actor.Output["CritExtraDamageReduction"]/100))
	const enemyCritThreshold = 10.1
	enemyBleedChance := float64(0)
	enemyIgniteChance := float64(0)
	if actor.Output["SelfIgniteEffect"] != 0 && actor.Output["IgniteAvoidChance"] < 100 &&
		actor.Output["SelfIgniteDuration"] != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyIgniteChance = actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "IgniteChance", "ElementalAilmentChance")
		enemyCritAilmentChance := float64(0)
		if !modDB.Flag(nil, "CritsOnYouDontAlwaysApplyElementalAilments") &&
			(actor.Output["EnemyCritChance"] > enemyCritThreshold || enemyIgniteChance > 0) {
			enemyCritAilmentChance = actor.Output["EnemyCritChance"]
		}
		enemyIgniteChance = (enemyCritAilmentChance + float64((1-enemyCritAilmentChance/100)*enemyIgniteChance)) *
			(1 - actor.Output["IgniteAvoidChance"]/100)
	}
	enemyPoisonChance := float64(0)
	if actor.Output["SelfPoisonEffect"] != 0 && actor.Output["PoisonAvoidChance"] < 100 &&
		actor.Output["SelfPoisonDuration"] != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyPoisonChance = actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "PoisonChance") *
			(1 - actor.Output["PoisonAvoidChance"]/100)
	}
	if damageCategoryConfig == "DamageOverTime" || (enemyIgniteChance+enemyPoisonChance+enemyBleedChance) > 0 {
		//pob:src/Modules/CalcDefence.lua:3502
		actor.Output["TotalDegen"] = actor.Output["TotalBuildDegen"]
		addEnemyDegen := func(sourceType data.DamageType, baseVal float64) {
			for _, damageConvertedType := range data.DamageType("").Values() {
				convertPercent := actor.DamageOverTimeShiftTable[sourceType][damageConvertedType]
				if convertPercent <= 0 {
					continue
				}
				total := float64(baseVal * (convertPercent / 100) * actor.Output[string(damageConvertedType+"TakenDotMult")])
				actor.Output[string(damageConvertedType+"EnemyDegen")] += total
				actor.Output["TotalDegen"] += total
			}
		}
		if damageCategoryConfig == "DamageOverTime" {
			//pob:block:src/Modules/CalcDefence.lua:3503-3562
			for _, damageType := range data.DamageType("").Values() {
				baseVal := enemyConfigNumber(environment, "enemy"+string(damageType)+"Damage")
				if baseVal > 0 {
					addEnemyDegen(damageType, baseVal)
				}
			}
		} else {
			//pob:block:src/Modules/CalcDefence.lua:3563-3631
			type enemyAilment struct {
				name        string
				percentBase float64
				damageType  data.DamageType
				sourceTypes []data.DamageType
			}
			ailmentList := make([]enemyAilment, 0, 3)
			if enemyBleedChance > 0 {
				ailmentList = append(ailmentList, enemyAilment{"Bleed", data.BleedPercentBase, data.DamageTypePhysical,
					[]data.DamageType{data.DamageTypePhysical}})
			}
			if enemyIgniteChance > 0 {
				sourceTypes := []data.DamageType{data.DamageTypeFire}
				if actor.Enemy.ModDB.Flag(nil, "AllDamageIgnites") {
					sourceTypes = data.DamageType("").Values()
				}
				ailmentList = append(ailmentList, enemyAilment{"Ignite", data.IgnitePercentBase, data.DamageTypeFire, sourceTypes})
			}
			if enemyPoisonChance > 0 {
				ailmentList = append(ailmentList, enemyAilment{"Poison", data.PoisonPercentBase, data.DamageTypeChaos,
					[]data.DamageType{data.DamageTypePhysical, data.DamageTypeChaos}})
			}
			for _, ailment := range ailmentList {
				baseVal := float64(0)
				for _, damageType := range ailment.sourceTypes {
					baseVal += actor.Output[string(damageType+"TakenDamage")]
				}
				//pob:src/Modules/CalcDefence.lua:3579
				baseVal = baseVal * ailment.percentBase *
					(enemyCritAilmentEffect / actor.Output["EnemyCritEffect"]) *
					actor.Output["Self"+ailment.name+"Effect"] / 100
				if baseVal > 0 {
					addEnemyDegen(ailment.damageType, baseVal)
				}
			}
		}
		if actor.Output["TotalDegen"] == actor.Output["TotalBuildDegen"] {
			delete(actor.Output, "TotalDegen")
		} else {
			//pob:block:src/Modules/CalcDefence.lua:3636-3710
			actor.Output["ComprehensiveNetLifeRegen"] = actor.Output["LifeRegenRecovery"]
			actor.Output["ComprehensiveNetManaRegen"] = actor.Output["ManaRegenRecovery"]
			actor.Output["ComprehensiveNetEnergyShieldRegen"] = actor.Output["EnergyShieldRegenRecovery"]
			comprehensiveLifeDegen := float64(0)
			comprehensiveManaDegen := float64(0)
			comprehensiveEnergyShieldDegen := float64(0)
			for _, damageType := range data.DamageType("").Values() {
				typeDegen := actor.Output[string(damageType+"BuildDegen")] + actor.Output[string(damageType+"EnemyDegen")]
				if typeDegen == 0 {
					continue
				}
				energyShieldDegen := float64(0)
				lifeDegen := float64(0)
				manaDegen := float64(0)
				takenFromMana := actor.Output[string(damageType+"MindOverMatter")] + actor.Output["sharedMindOverMatter"]
				if actor.Output["EnergyShieldRegenRecovery"] > 0 {
					if modDB.Flag(nil, "EnergyShieldProtectsMana") {
						lifeDegen = typeDegen * (1 - takenFromMana/100)
						energyShieldDegen = typeDegen * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (takenFromMana / 100)
					} else {
						lifeDegen = typeDegen * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (1 - takenFromMana/100)
						energyShieldDegen = typeDegen * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100)
					}
					manaDegen = typeDegen * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (takenFromMana / 100)
				} else {
					lifeDegen = typeDegen * (1 - takenFromMana/100)
					manaDegen = typeDegen * (takenFromMana / 100)
				}
				comprehensiveLifeDegen += lifeDegen
				comprehensiveManaDegen += manaDegen
				comprehensiveEnergyShieldDegen += energyShieldDegen
			}
			actor.Output["ComprehensiveNetLifeRegen"] += actor.Output["LifeRecoupRecoveryAvg"] -
				comprehensiveLifeDegen - actor.Output["LifeLossLostAvg"]
			actor.Output["ComprehensiveNetManaRegen"] += actor.Output["ManaRecoupRecoveryAvg"] - comprehensiveManaDegen
			actor.Output["ComprehensiveNetEnergyShieldRegen"] += actor.Output["EnergyShieldRecoupRecoveryAvg"] -
				comprehensiveEnergyShieldDegen
			actor.Output["ComprehensiveTotalNetRegen"] = actor.Output["ComprehensiveNetLifeRegen"] +
				actor.Output["ComprehensiveNetManaRegen"] + actor.Output["ComprehensiveNetEnergyShieldRegen"]
		}
	}
	// maximum hit taken
	// fix total pools, as they aren't used anymore
	//pob:block:src/Modules/CalcDefence.lua:3123-3161
	for _, damageType := range data.DamageType("").Values() {
		wardBypass := modDB.Sum(mod.TypeBase, nil, "WardBypass")
		if wardBypass > 0 {
			poolProtected := float64(actor.Output["Ward"] / (1 - wardBypass/100) * (wardBypass / 100))
			sourcePool := actor.Output[string(damageType+"TotalHitPool")]
			sourcePool = max(sourcePool-poolProtected, 0) + min(sourcePool, poolProtected)/(wardBypass/100)
			actor.Output[string(damageType+"TotalHitPool")] = sourcePool
		} else {
			actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] + actor.Output["Ward"]
		}
		elementalAegis := float64(0)
		if isElemental[string(damageType)] {
			elementalAegis = actor.Output[string(damageType+"AegisDisplay")]
		}
		actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] +
			max(max(actor.Output[string(damageType+"Aegis")], actor.Output["sharedAegis"]), elementalAegis)
		//	local GuardAbsorbRate = output["sharedGuardAbsorbRate"] or 0 + output[damageType.."GuardAbsorbRate"] or 0
		GuardAbsorbRate := actor.Output["sharedGuardAbsorbRate"]
		if GuardAbsorbRate > 0 {
			GuardAbsorb := actor.Output["sharedGuardAbsorb"]
			if GuardAbsorbRate >= 100 {
				actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] + GuardAbsorb
			} else {
				poolProtected := float64(GuardAbsorb / (GuardAbsorbRate / 100) * (1 - GuardAbsorbRate/100))
				actor.Output[string(damageType+"TotalHitPool")] = max(actor.Output[string(damageType+"TotalHitPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalHitPool")], poolProtected)/(1-GuardAbsorbRate/100)
			}
		}
		// Undo the ally pool drains in reverse order to recover the incoming hit.
		for i := len(allyLifePoolList) - 1; i >= 0; i-- {
			ally := allyLifePoolList[i]
			life, mitigation := actor.Output[ally.life], actor.Output[ally.mitigation]
			if life <= 0 || mitigation <= 0 {
				continue
			}
			mitigation = min(mitigation, 100)
			if mitigation == 100 {
				actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] + life
			} else {
				poolProtected := float64(life / (mitigation / 100) * (1 - mitigation/100))
				actor.Output[string(damageType+"TotalHitPool")] = max(actor.Output[string(damageType+"TotalHitPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalHitPool")], poolProtected)/(1-mitigation/100)
			}
		}
	}
	for _, damageType := range data.DamageType("").Values() {
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel(string(damageType+"MaximumHitTaken"), "Maximum Hit Taken (uses lowest value)")
			actor.Breakdown.AddCol(string(damageType+"MaximumHitTaken"),
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "TotalPool", Key: "pool"},
				BCol{Label: "Taken", Key: "taken"},
				BCol{Label: "Final", Key: "final"},
			)
		}
		//pob:block:src/Modules/CalcDefence.lua:3165-3222
		vaalArcticMitigation := actor.Output["VaalArcticArmourMitigation"]
		partMin := math.Inf(1)
		useConversionSmoothing := false
		for _, damageConvertedType := range data.DamageType("").Values() {
			convertPercent := actor.DamageShiftTable[damageType][damageConvertedType]
			takenFlat := actor.Output[string(damageConvertedType)+"takenFlat"]
			if convertPercent > 0 || takenFlat != 0 {
				hitTaken := float64(0)
				effectiveAppliedArmour := actor.Output[string(damageConvertedType+"EffectiveAppliedArmour")]
				damageConvertedMulti := convertPercent / 100
				totalHitPool := actor.Output[string(damageConvertedType+"TotalHitPool")]
				totalTakenMulti := actor.Output[string(damageConvertedType+"AfterReductionTakenHitMulti")] * (1 - vaalArcticMitigation)
				switch {
				case damageConvertedMulti <= 0:
					takenWithoutIncoming := math.Max(takenFlat, 0) * totalTakenMulti
					hitTaken = utils.Ternary(takenWithoutIncoming >= totalHitPool, float64(0), math.Inf(1))
				case effectiveAppliedArmour == 0 && convertPercent == 100:
					totalResistMult := actor.Output[string(damageConvertedType+"ResistTakenHitMulti")]
					drMulti := totalResistMult * (1 - actor.Output[string(damageConvertedType+"DamageReduction")]/100)
					hitTaken = math.Max(totalHitPool/damageConvertedMulti/drMulti-takenFlat, 0) / totalTakenMulti
				default:
					totalResistMult := actor.Output[string(damageConvertedType+"ResistTakenHitMulti")]
					reductionPercent := utils.Ternary(modDB.Flag(nil, string("SelfIgnoreBase"+damageConvertedType+"DamageReduction")), 0, actor.Output[string("Base"+damageConvertedType+"DamageReductionWhenHit")])
					flatDR := reductionPercent / 100
					enemyOverwhelmPercent := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageConvertedType+"DamageReduction")), 0, actor.Output[string(damageConvertedType+"EnemyOverwhelm")])

					poolNetOfFlat := totalHitPool - float64(takenFlat*totalTakenMulti)

					resistXConvert := totalResistMult * damageConvertedMulti
					a := float64(5 * (1 - flatDR + enemyOverwhelmPercent/100) * totalTakenMulti * resistXConvert * resistXConvert)
					b := float64((float64((enemyOverwhelmPercent/100-flatDR)*effectiveAppliedArmour*totalTakenMulti) - float64(5*poolNetOfFlat)) * resistXConvert)
					c := float64(-effectiveAppliedArmour * poolNetOfFlat)

					raw := math.Inf(1)
					if a != 0 {
						raw = (math.Sqrt(math.Max(float64(b*b)-float64(4*a*c), 0)) - b) / (2 * a)
					}

					noDRMaxHit := totalHitPool / damageConvertedMulti / totalResistMult / totalTakenMulti * (1 - takenFlat*totalTakenMulti/totalHitPool)
					maxDRMaxHit := noDRMaxHit / (1 - (actor.Output["DamageReductionMax"]-enemyOverwhelmPercent)/100)
					hitTaken = math.Floor(max(min(raw, maxDRMaxHit), noDRMaxHit))
					useConversionSmoothing = useConversionSmoothing || convertPercent != 100
				}
				partMin = min(partMin, hitTaken)
			}
		}

		//pob:block:src/Modules/CalcDefence.lua:2500-2519
		enemyDamageMult := actor.Output[string(damageType+"EnemyDamageMult")]
		if math.IsInf(partMin, 1) {
			actor.Output[string(damageType+"MaximumHitTaken")] = math.Inf(1)
		} else if useConversionSmoothing {
			//pob:block:src/Modules/CalcDefence.lua:3229-3258
			passIncomingDamage := partMin
			previousOverkill := float64(0)
			havePreviousOverkill := false
			for n := 1; n <= data.MaxHitSmoothingPasses; n++ {
				_, passDamages := TakenHitFromDamage(passIncomingDamage, damageType, actor)
				passPools := ReducePoolsByDamage(nil, passDamages, actor)
				passOverkill := passPools.OverkillDamage - passPools.HitPoolRemaining
				passRatio := float64(0)
				for partType := range passDamages {
					partPool := actor.Output[string(partType+"TotalHitPool")]
					if partPool > 0 {
						passRatio = math.Max(passRatio, (passOverkill+partPool)/partPool)
					}
				}
				if passRatio <= 0 {
					passRatio = 1
				}
				stepSize := float64(1)
				if n > 1 && havePreviousOverkill && previousOverkill != 0 && !math.IsNaN(previousOverkill) {
					stepSize = math.Min(math.Abs((passOverkill-previousOverkill)/previousOverkill), 2)
				}
				stepAdjust := float64(0)
				if stepSize > 1 {
					stepAdjust = -passOverkill / stepSize
				} else if n > 1 {
					stepAdjust = -passOverkill * stepSize
				}
				previousOverkill = passOverkill
				havePreviousOverkill = true
				passIncomingDamage = (passIncomingDamage + stepAdjust) / math.Sqrt(passRatio)
				if passOverkill < 1 && passOverkill > -1 {
					break
				}
			}

			actor.Output[string(damageType+"MaximumHitTaken")] = math.Floor(passIncomingDamage/enemyDamageMult + 0.5)
		} else {
			actor.Output[string(damageType+"MaximumHitTaken")] = math.Round(partMin / enemyDamageMult)
		}
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Total Pool: %.2f", actor.Output[string(damageType+"TotalHitPool")]))
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Taken Mult: %.2f", actor.Output[string(damageType+"TotalHitPool")]/actor.Output[string(damageType+"MaximumHitTaken")]))
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Maximum hit you can take: %.0f", actor.Output[string(damageType+"MaximumHitTaken")]))
		}
	}

	//pob:src/Modules/CalcDefence.lua:3314-3315
	minimum := math.Inf(1)
	secondMinimum := math.Inf(1)
	for _, damageType := range data.DamageType("").Values() {
		hitTaken := actor.Output[string(damageType+"MaximumHitTaken")]
		if hitTaken < minimum {
			secondMinimum = minimum
			minimum = hitTaken
		} else if hitTaken < secondMinimum {
			secondMinimum = hitTaken
		}
	}
	actor.Output["SecondMinimalMaximumHitTaken"] = secondMinimum
}

//pob:block:src/Modules/CalcDefence.lua:1557-1559
//pob:block:src/Modules/CalcDefence.lua:1565-1567
func foulbornArmourDefenseMods(modDB *moddb.ModDB) []mod.Mod {
	percent := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercent")
	percentArmour := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour")
	effective := percent * modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance") / 100
	return []mod.Mod{
		mod.NewFloat("ArmourDefense", mod.TypeMAX, percentArmour-100).
			Source("Chance to Defend from Spell Suppression: Max Calc").Tag(mod.Condition("ArmourMax")),
		mod.NewFloat("ArmourDefense", mod.TypeMAX, min(effective, 1.0)*(percentArmour-100)).
			Source("Chance to Defend from Spell Suppression: Average Calc").Tag(mod.Condition("ArmourAvg")),
		mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(effective), 1.0)*(percentArmour-100)).
			Source("Chance to Defend from Spell Suppression: Min Calc").
			Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)),
	}
}

func ancestralVisionAvoid(modDB *moddb.ModDB) float64 {
	percent := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionAppliesToAilmentAvoidancePercent") / 100
	rawSuppression := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance")
	return math.Floor(percent * rawSuppression)
}

//pob:src/Modules/CalcDefence.lua:507-633
func calcResistances(actor *Actor, modDB *moddb.ModDB) {
	//pob:block:src/Modules/CalcDefence.lua:510-511
	actor.Output["PhysicalResist"] = 0

	// Process Resistance conversion mods.
	//pob:block:src/Modules/CalcDefence.lua:513-558
	for fi, resFrom := range resistTypeList {
		var maxRes float64
		maxResSet := false
		for ti, resTo := range resistTypeList {
			conversionRate := modDB.Sum(mod.TypeBase, nil, resistConvertNames[fi][ti].maxRes) / 100
			if conversionRate == 0 {
				continue
			}
			if !maxResSet {
				for _, m := range modDB.Tabulate(mod.TypeBase, nil, resFrom+"ResistMax") {
					if m.Mod.GetSource() != "Base" {
						maxRes += m.Value
					}
				}
				maxResSet = true
			}
			if maxRes != 0 {
				modDB.AddMod(mod.NewFloat(resTo+"ResistMax", mod.TypeBase, maxRes*conversionRate).
					Source(mod.Source(resFrom + " To " + resTo + " Max Resistance Conversion")))
			}
		}
	}
	for fi, resFrom := range resistTypeList {
		var res float64
		resSet := false
		for ti, resTo := range resistTypeList {
			conversionRate := modDB.Sum(mod.TypeBase, nil, resistConvertNames[fi][ti].res) / 100
			if conversionRate == 0 {
				continue
			}
			if !resSet {
				for _, m := range modDB.Tabulate(mod.TypeBase, nil, resFrom+"Resist") {
					if m.Mod.GetSource() != "Base" {
						res += m.Value
					}
				}
				resSet = true
			}
			if res != 0 {
				modDB.AddMod(mod.NewFloat(resTo+"Resist", mod.TypeBase, res*conversionRate).
					Source(mod.Source(resFrom + " To " + resTo + " Resistance Conversion")))
			}
			for _, m := range modDB.Tabulate(mod.TypeIncrease, nil, resFrom+"Resist") {
				modDB.AddMod(mod.NewFloat(resTo+"Resist", mod.TypeIncrease, m.Value*conversionRate).
					Source(m.Mod.GetSource()))
			}
			for _, m := range modDB.Tabulate(mod.TypeMore, nil, resFrom+"Resist") {
				modDB.AddMod(mod.NewFloat(resTo+"Resist", mod.TypeMore, m.Value*conversionRate).
					Source(m.Mod.GetSource()))
			}
		}
	}

	// Highest Maximum Elemental Resistance for Melding of the Flesh
	if modDB.Flag(nil, "ElementalResistMaxIsHighestResistMax") {
		highestResistMax := float64(0)
		highestResistMaxType := ""
		for _, elem := range resistTypeList {
			resistMax := utils.Or(modDB.Override(nil, elem+"ResistMax"), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, elem+"ResistMax", utils.Ternary(isElemental[elem], "ElementalResistMax", ""))))
			if resistMax > highestResistMax && isElemental[elem] {
				highestResistMax = resistMax
				highestResistMaxType = elem
			}
		}
		for _, elem := range resistTypeList {
			if isElemental[elem] {
				modDB.AddMod(mod.NewFloat(elem+"ResistMax", mod.TypeOverride, highestResistMax).Source(mod.Source(highestResistMaxType + " Melding of the Flesh")))
			}
		}
	}

	for ei, elem := range resistTypeList {
		n := &resistNames[ei]
		Min := float64(data.ResistFloor)
		Max := utils.Or(modDB.Override(nil, n.max), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, n.max, utils.Ternary(isElemental[elem], "ElementalResistMax", ""))))
		totemMax := utils.Or(modDB.Override(nil, n.totemMax), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, n.totemMax, utils.Ternary(isElemental[elem], "TotemElementalResistMax", ""))))
		totalMV := modDB.Override(nil, n.resist)
		totemTotalMV := modDB.Override(nil, n.totem)
		var total, dotTotal, totemTotal float64
		dotSet := false
		if totalMV == nil {
			base := modDB.Sum(mod.TypeBase, nil, n.resist, utils.Ternary(isElemental[elem], "ElementalResist", ""))
			inc := max(calclib.Mod(modDB, nil, n.resist, utils.Ternary(isElemental[elem], "ElementalResist", "")), 0)
			total = base * inc
			//pob:block:src/Modules/CalcDefence.lua:589-590
			dotBase := modDB.Sum(mod.TypeBase, &moddb.ListCfg{
				Flags:        utils.Ptr(mod.MFlagDot),
				KeywordFlags: utils.Ptr(mod.KeywordFlag(0)),
			}, n.resist, utils.Ternary(isElemental[elem], "ElementalResist", ""))
			dotTotal = dotBase * inc
			dotSet = true
		} else {
			total = totalMV.Float()
		}
		if totemTotalMV == nil {
			base := modDB.Sum(mod.TypeBase, nil, n.totem, utils.Ternary(isElemental[elem], "TotemElementalResist", ""))
			totemTotal = base * max(calclib.Mod(modDB, nil, n.totem, utils.Ternary(isElemental[elem], "TotemElementalResist", "")), 0)
		} else {
			totemTotal = totemTotalMV.Float()
		}

		// Fractional resistances are truncated.
		//pob:block:src/Modules/CalcDefence.lua:597-603
		total = math.Trunc(total)
		dotTotal = utils.Ternary(dotSet, math.Trunc(dotTotal), total)
		totemTotal = math.Trunc(totemTotal)
		Min = math.Trunc(Min)
		Max = math.Trunc(Max)
		totemMax = math.Trunc(totemMax)

		final := max(min(total, Max), Min)
		dotFinal := max(min(dotTotal, Max), Min)
		totemFinal := max(min(totemTotal, totemMax), Min)
		actor.Output[n.resist] = final
		actor.Output[n.total] = total
		actor.Output[n.overCap] = max(0, total-Max)
		actor.Output[n.over75] = max(0, final-75)
		actor.Output[n.missing] = max(0, Max-final)
		//pob:block:src/Modules/CalcDefence.lua:615
		actor.Output[n.overTime] = dotFinal
		actor.Output[n.totem] = totemFinal
		actor.Output[n.totemTotal] = totemTotal
		actor.Output[n.totemOverCap] = max(0, totemTotal-totemMax)
		actor.Output[n.missingTotem] = max(0, totemMax-totemFinal)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine(n.resist,
				"Min: "+fmt.Sprint(Min)+"%",
				"Max: "+fmt.Sprint(Max)+"%",
				"Total: "+fmt.Sprint(total)+"%",
			)
			actor.Breakdown.AddLine(n.totem,
				"Min: "+fmt.Sprint(Min)+"%",
				"Max: "+fmt.Sprint(totemMax)+"%",
				"Total: "+fmt.Sprint(totemTotal)+"%",
			)
		}
	}
}

var damageRecoupNames = func() [3][5]struct{ recoup, energyShield, addToEnergyShield string } {
	var out [3][5]struct{ recoup, energyShield, addToEnergyShield string }
	for ri, recoupType := range []string{"Life", "Mana", "EnergyShield"} {
		for di, dt := range data.DamageType("").Values() {
			out[ri][di].recoup = string(dt) + recoupType + "Recoup"
			out[ri][di].energyShield = string(dt) + "EnergyShieldRecoup"
			out[ri][di].addToEnergyShield = "Add" + recoupType + "RecoupToEnergyShieldRecoup"
		}
	}
	return out
}()

var resistNames = func() []struct {
	resist, total, max, overCap, over75, missing, overTime  string
	totem, totemTotal, totemMax, totemOverCap, missingTotem string
} {
	out := make([]struct {
		resist, total, max, overCap, over75, missing, overTime  string
		totem, totemTotal, totemMax, totemOverCap, missingTotem string
	}, len(resistTypeList))
	for i, elem := range resistTypeList {
		n := &out[i]
		n.resist, n.total, n.max = elem+"Resist", elem+"ResistTotal", elem+"ResistMax"
		n.overCap, n.over75, n.missing, n.overTime = elem+"ResistOverCap", elem+"ResistOver75", "Missing"+elem+"Resist", elem+"ResistOverTime"
		n.totem, n.totemTotal, n.totemMax = "Totem"+elem+"Resist", "Totem"+elem+"ResistTotal", "Totem"+elem+"ResistMax"
		n.totemOverCap, n.missingTotem = "Totem"+elem+"ResistOverCap", "MissingTotem"+elem+"Resist"
	}
	return out
}()

func namesOf[T ~string](values []T, prefix, suffix string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = prefix + string(v) + suffix
	}
	return out
}

var (
	damageReductionNames            = namesOf(data.DamageType("").Values(), "", "DamageReduction")
	damageReductionWhenHitNames     = namesOf(data.DamageType("").Values(), "", "DamageReductionWhenHit")
	baseDamageReductionNames        = namesOf(data.DamageType("").Values(), "Base", "DamageReduction")
	baseDamageReductionWhenHitNames = namesOf(data.DamageType("").Values(), "Base", "DamageReductionWhenHit")
	avoidDamageChanceNames          = namesOf(data.DamageType("").Values(), "Avoid", "DamageChance")
	avoidAilmentNames               = namesOf(data.Ailment("").Values(), "Avoid", "")
	ailmentImmuneNames              = namesOf(data.Ailment("").Values(), "", "Immune")
	ailmentAvoidChanceNames         = namesOf(data.Ailment("").Values(), "", "AvoidChance")
	selfAilmentDurationNames        = namesOf(data.Ailment("").Values(), "Self", "Duration")
	selfAilmentEffectNames          = namesOf(data.Ailment("").Values(), "Self", "Effect")
	enemyAilmentEffectNames         = namesOf(data.Ailment("").Values(), "Enemy", "Effect")
)

var regenNames = func() [4]struct {
	regen, regenInc, regenPercent, regenRecovery, recoveryRateMod, recovery, degen, degenPercent, degenPercentTincture string
	noRegen, cannotGain, unaffectedBy, canGain                                                                         string
} {
	var out [4]struct {
		regen, regenInc, regenPercent, regenRecovery, recoveryRateMod, recovery, degen, degenPercent, degenPercentTincture string
		noRegen, cannotGain, unaffectedBy, canGain                                                                         string
	}
	for i, name := range []string{"Mana", "Life", "EnergyShield", "Rage"} {
		n := &out[i]
		n.regen, n.regenInc, n.regenPercent, n.regenRecovery = name+"Regen", name+"RegenInc", name+"RegenPercent", name+"RegenRecovery"
		n.recoveryRateMod, n.recovery = name+"RecoveryRateMod", name+"Recovery"
		n.degen, n.degenPercent, n.degenPercentTincture = name+"Degen", name+"DegenPercent", name+"DegenPercentTincture"
		n.noRegen, n.cannotGain, n.unaffectedBy, n.canGain = "No"+name+"Regen", "CannotGain"+name, "UnaffectedBy"+name+"Regen", "Condition:CanGain"+name
	}
	return out
}()

var resistConvertNames = func() [][]struct{ maxRes, res string } {
	out := make([][]struct{ maxRes, res string }, len(resistTypeList))
	for f, resFrom := range resistTypeList {
		out[f] = make([]struct{ maxRes, res string }, len(resistTypeList))
		for t, resTo := range resistTypeList {
			out[f][t].maxRes, out[f][t].res = resFrom+"MaxResConvertTo"+resTo, resFrom+"ResConvertTo"+resTo
		}
	}
	return out
}()
