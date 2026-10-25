package calculator2

import (
	"math"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//pob2:src/Modules/CalcDefence.lua:74-129
func doActorLifeManaSpirit(actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output
	condList := modDB.Conditions

	lowLifePerc := modDB.Sum(mod.TypeBase, nil, "LowLifePercentage")
	if lowLifePerc > 0 {
		output["LowLifePercentage"] = 100.0 * lowLifePerc
	} else {
		output["LowLifePercentage"] = 100.0 * miscConstant("LowPoolThreshold")
	}
	fullLifePerc := modDB.Sum(mod.TypeBase, nil, "FullLifePercentage")
	if fullLifePerc > 0 {
		output["FullLifePercentage"] = 100.0 * fullLifePerc
	} else {
		output["FullLifePercentage"] = 100.0
	}

	chaosInoculation := modDB.Flag(nil, "ChaosInoculation")

	for _, res := range []string{"Life", "Mana", "Spirit"} {
		base := modDB.Sum(mod.TypeBase, nil, res)
		extra := modDB.Sum(mod.TypeBase, nil, "Extra"+res)
		total := modDB.Sum(mod.TypeBase, nil, res+"Total")
		inc := modDB.Sum(mod.TypeIncrease, nil, res)
		more := modDB.More(nil, res)
		conv := math.Min(modDB.Sum(mod.TypeBase, nil, res+"ConvertToEnergyShield", res+"ConvertToArmour", res+"ConvertToEvasion"), 100)
		// output[res.."HasOverride"] is a boolean, which this float output table does not carry.
		if override := modDB.Override(nil, res); override != nil {
			output[res] = override.Float()
			continue
		}
		output[res] = math.Max(round((base*(1-conv/100)+extra)*(1+inc/100)*more+total), 1)
	}
	if chaosInoculation {
		output["Life"] = 1
		condList["FullLife"] = true
	}
	output["LowestOfMaximumLifeAndMaximumMana"] = math.Min(output["Life"], output["Mana"])
}

var isElemental = map[string]bool{"Fire": true, "Cold": true, "Lightning": true}

var resistTypeList = []string{"Fire", "Cold", "Lightning", "Chaos"}

var armourSlots = []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"}

// resistNames is `name, isElemental[elem] and elementalName`: a false name matches no mod.
func resistNames(elem, name, elementalName string) []string {
	if isElemental[elem] {
		return []string{name, elementalName}
	}
	return []string{name}
}

//pob2:src/Modules/CalcDefence.lua:41-47
func monsterHitChance(evasion, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := (1 - (0.95*evasion)/(evasion+4*accuracy)) * 100
	return math.Max(math.Min(round(rawChance), 100), 5)
}

//pob2:src/Modules/CalcDefence.lua:49-55
func deflectChance(deflection, accuracy float64) float64 {
	if deflection < 1 {
		return 0
	}
	chanceToNotDeflect := accuracy/(accuracy+deflection*0.12)*150 - 50
	return math.Max(math.Min(100-round(chanceToNotDeflect), miscConstant("DeflectionChanceCap")), 0)
}

//pob2:src/Classes/Item.lua:2413-2419
func itemArmourDataValue(item *Item, name string, level int) float64 {
	if item == nil || item.ArmourData == nil {
		return 0
	}
	return item.ArmourData[name] + round(item.ArmourData[name+"PerLevel"]*float64(level))
}

//pob2:src/Modules/CalcDefence.lua:134-145
func doActorDarkness(actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output
	inc := modDB.Sum(mod.TypeIncrease, nil, "Darkness")
	base := modDB.Sum(mod.TypeBase, nil, "Darkness")
	output["Darkness"] = base * (1 + inc/100)

	reserved := modDB.Sum(mod.TypeBase, nil, "ReservedDarkness")
	output["ReservedDarkness"] = math.Min(reserved, output["Darkness"])
	output["UnreservedDarkness"] = output["Darkness"] - output["ReservedDarkness"]
}

// No active skill list is built yet, so the per-skill reservation loop reserves nothing.
//
//pob2:src/Modules/CalcDefence.lua:177-360
func doActorLifeManaSpiritReservation(actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output
	condList := modDB.Conditions

	reservedBase := map[string]float64{"Life": 0, "Mana": 0, "Spirit": 0}
	reservedPercent := map[string]float64{"Life": modDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved"), "Mana": 0, "Spirit": 0}
	uncancellable := map[string]float64{
		"Life":   modDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved"),
		"Mana":   modDB.Sum(mod.TypeBase, nil, "ExtraManaReserved"),
		"Spirit": modDB.Sum(mod.TypeBase, nil, "ExtraSpiritReserved"),
	}

	//pob2:block:src/Modules/CalcDefence.lua:338-359
	for _, pool := range []string{"Life", "Mana", "Spirit"} {
		maxPool := output[pool]
		lowPerc := modDB.Sum(mod.TypeBase, nil, "Low"+pool+"Percentage")
		reserved := reservedBase[pool] + math.Ceil(maxPool*reservedPercent[pool]/100)
		uncancellableReservation := uncancellable[pool]
		output[pool+"Reserved"] = math.Min(reserved, maxPool)
		output[pool+"Unreserved"] = maxPool - reserved
		output[pool+"UncancellableReservation"] = math.Min(uncancellableReservation, 0)
		output[pool+"CancellableReservation"] = 100 - uncancellableReservation
		if maxPool > 0 {
			output[pool+"ReservedPercent"] = math.Min(reserved/maxPool*100, 100)
			output[pool+"UnreservedPercent"] = (maxPool - reserved) / maxPool * 100
			threshold := miscConstant("LowPoolThreshold")
			if lowPerc > 0 {
				threshold = lowPerc
			}
			if (maxPool-reserved)/maxPool <= threshold {
				condList["Low"+pool] = true
			}
		}
		for _, value := range modDB.List(nil, "GrantReserved"+pool+"AsAura") {
			t, ok := value.(LuaTable)
			if !ok {
				continue
			}
			m, ok := t["mod"].(mod.Mod)
			if !ok {
				continue
			}
			auraMod := m.Clone()
			if v := auraMod.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
				auraMod.SetValue(mod.NewModValueFloat(math.Floor(v.Float() * math.Min(reserved, maxPool))))
			}
			modDB.AddMod(mod.NewList("ExtraAura", mod.ExtraAura{Mod: auraMod}))
		}
	}
}

// defenceResource is one resourceList entry of the primary-defence block.
type defenceResource struct {
	name           string
	basePerSlot    map[string]float64
	globalBase     float64
	conversionRate map[string]float64
	mods           []string
	modsInc        []string
	modsTotal      []string
	defence        bool

	totalConversion float64
	totalBase       float64
}

// defence is calcs.defence as far as the defence slice: armour-item outputs, resistances, block,
// the pools and their reservation, the primary defences with evade and deflect chance, damage
// reduction caps and movement speed. Suppression, dodge, recovery, leech, regeneration, recoup and
// the EHP section are not ported.
//
//pob2:src/Modules/CalcDefence.lua:790-1938
func defence(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output

	output["ActionSpeedMod"] = actionSpeedMod(actor)

	//pob2:block:src/Modules/CalcDefence.lua:805-857
	for _, slot := range armourSlots {
		item := actor.ItemList[slot]
		if item == nil || item.ArmourData == nil {
			continue
		}
		wardBase := itemArmourDataValue(item, "Ward", actor.Level)
		if wardBase > 0 {
			output["WardOnAllArmourItems"] += wardBase
			if slot == "Body Armour" && modDB.Flag(nil, "DoubleBodyArmourDefence") {
				wardBase *= 2
			}
			output["WardOn"+slot] = wardBase
		}
		energyShieldBase := itemArmourDataValue(item, "EnergyShield", actor.Level)
		if energyShieldBase > 0 {
			output["EnergyShieldOnAllArmourItems"] += energyShieldBase
			if slot == "Body Armour" && modDB.Flag(nil, "DoubleBodyArmourDefence") {
				energyShieldBase *= 2
			}
			output["EnergyShieldOn"+slot] = energyShieldBase
		}
		armourBase := itemArmourDataValue(item, "Armour", actor.Level)
		if armourBase > 0 {
			output["ArmourOnAllArmourItems"] += armourBase
			if slot == "Body Armour" {
				if modDB.Flag(nil, "DoubleBodyArmourDefence") {
					armourBase *= 2
				}
				if modDB.Flag(nil, "Unbreakable") {
					armourBase *= 2
				}
			}
			output["ArmourOn"+slot] = armourBase
		}
		evasionBase := itemArmourDataValue(item, "Evasion", actor.Level)
		if evasionBase > 0 {
			output["EvasionOnAllArmourItems"] += evasionBase
			if slot == "Body Armour" {
				if modDB.Flag(nil, "DoubleBodyArmourDefence") {
					evasionBase *= 2
				}
				if modDB.Flag(nil, "Unbreakable") && modDB.Flag(nil, "IronReflexes") {
					evasionBase *= 2
				}
			}
			output["EvasionOn"+slot] = evasionBase
		}
		output["LowestOfArmourAndEvasionOn"+slot] = math.Min(armourBase, evasionBase)
	}

	output["PhysicalResist"] = 0

	//pob2:block:src/Modules/CalcDefence.lua:864-882
	for _, resFrom := range resistTypeList {
		var maxRes float64
		haveMaxRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum(mod.TypeBase, nil, resFrom+"MaxResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveMaxRes {
					haveMaxRes = true
					for _, m := range modDB.Tabulate(mod.TypeBase, nil, resFrom+"ResistMax") {
						if m.Mod.GetSource() != "Base" {
							maxRes += m.Value
						}
					}
				}
				if maxRes != 0 {
					modDB.AddMod(newMod(resTo+"ResistMax", mod.TypeBase, maxRes*conversionRate, mod.Source(resFrom+" To "+resTo+" Max Resistance Conversion")))
				}
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:884-908
	for _, resFrom := range resistTypeList {
		var res float64
		haveRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum(mod.TypeBase, nil, resFrom+"ResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveRes {
					haveRes = true
					for _, m := range modDB.Tabulate(mod.TypeBase, nil, resFrom+"Resist") {
						if m.Mod.GetSource() != "Base" {
							res += m.Value
						}
					}
				}
				if res != 0 {
					modDB.AddMod(newMod(resTo+"Resist", mod.TypeBase, res*conversionRate, mod.Source(resFrom+" To "+resTo+" Resistance Conversion")))
				}
				for _, m := range modDB.Tabulate(mod.TypeIncrease, nil, resFrom+"Resist") {
					modDB.AddMod(newMod(resTo+"Resist", mod.TypeIncrease, m.Value*conversionRate, m.Mod.GetSource()))
				}
				for _, m := range modDB.Tabulate(mod.TypeMore, nil, resFrom+"Resist") {
					modDB.AddMod(newMod(resTo+"Resist", mod.TypeMore, m.Value*conversionRate, m.Mod.GetSource()))
				}
			}
		}
	}

	maxResistCap := miscConstant("MaxResistCap")
	resistMax := func(elem string) float64 {
		if o := modDB.Override(nil, elem+"ResistMax"); o != nil {
			return o.Float()
		}
		return math.Min(maxResistCap, modDB.Sum(mod.TypeBase, nil, resistNames(elem, elem+"ResistMax", "ElementalResistMax")...))
	}

	//pob2:block:src/Modules/CalcDefence.lua:911-926
	if modDB.Flag(nil, "ElementalResistMaxIsHighestResistMax") {
		highestResistMax := 0.0
		highestResistMaxType := ""
		for _, elem := range resistTypeList {
			rMax := resistMax(elem)
			if rMax > highestResistMax && isElemental[elem] {
				highestResistMax = rMax
				highestResistMaxType = elem
			}
		}
		for _, elem := range resistTypeList {
			if isElemental[elem] {
				modDB.AddMod(newMod(elem+"ResistMax", mod.TypeOverride, highestResistMax, mod.Source(highestResistMaxType+" Melding of the Flesh")))
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:928-988
	for _, elem := range resistTypeList {
		resistFloor := miscConstant("ResistFloor")
		var total, dotTotal, totemTotal, totemMax float64
		haveDotTotal := false
		if o := modDB.Override(nil, "Totem"+elem+"ResistMax"); o != nil {
			totemMax = o.Float()
		} else {
			totemMax = math.Min(maxResistCap, modDB.Sum(mod.TypeBase, nil, resistNames(elem, "Totem"+elem+"ResistMax", "TotemElementalResistMax")...))
		}
		if o := modDB.Override(nil, elem+"Resist"); o != nil {
			total = o.Float()
		} else {
			names := resistNames(elem, elem+"Resist", "ElementalResist")
			base := modDB.Sum(mod.TypeBase, nil, names...)
			inc := math.Max(calcLibMod(modDB, nil, names...), 0)
			total = base * inc
			dotFlags := modFlagDot
			dotKeywordFlags := mod.KeywordFlag(0)
			dotBase := modDB.Sum(mod.TypeBase, &moddb.ListCfg{Flags: &dotFlags, KeywordFlags: &dotKeywordFlags}, names...)
			dotTotal = dotBase * inc
			haveDotTotal = true
		}
		if o := modDB.Override(nil, "Totem"+elem+"Resist"); o != nil {
			totemTotal = o.Float()
		} else {
			names := resistNames(elem, "Totem"+elem+"Resist", "TotemElementalResist")
			base := modDB.Sum(mod.TypeBase, nil, names...)
			totemTotal = base * math.Max(calcLibMod(modDB, nil, names...), 0)
		}

		total = math.Trunc(total)
		output[elem+"ResistTotal"] = total
		var maxRes float64
		if modDB.Flag(nil, "MaxBlockChanceModsApplyMaxResist") {
			blockMaxBonus := modDB.Sum(mod.TypeBase, nil, "BlockChanceMax")
			if modDB.Override(nil, "BlockChanceMax") != nil {
				blockMaxBonus = 0
			}
			if o := modDB.Override(nil, elem+"ResistMax"); o != nil {
				maxRes = o.Float()
			} else {
				maxRes = math.Min(maxResistCap, modDB.Sum(mod.TypeBase, nil, resistNames(elem, elem+"ResistMax", "ElementalResistMax")...)+blockMaxBonus)
			}
		} else {
			maxRes = resistMax(elem)
		}

		if haveDotTotal {
			dotTotal = math.Trunc(dotTotal)
		} else {
			dotTotal = total
		}
		totemTotal = math.Trunc(totemTotal)
		resistFloor = math.Trunc(resistFloor)
		maxRes = math.Trunc(maxRes)
		totemMax = math.Trunc(totemMax)

		final := math.Max(math.Min(total, maxRes), resistFloor)
		dotFinal := math.Max(math.Min(dotTotal, maxRes), resistFloor)
		totemFinal := math.Max(math.Min(totemTotal, totemMax), resistFloor)

		output[elem+"Resist"] = final
		output[elem+"ResistOverCap"] = math.Max(0, total-maxRes)
		output[elem+"ResistOver75"] = math.Max(0, final-75)
		output["Missing"+elem+"Resist"] = math.Max(0, maxRes-final)
		output[elem+"ResistOverTime"] = dotFinal
		output["Totem"+elem+"Resist"] = totemFinal
		output["Totem"+elem+"ResistTotal"] = totemTotal
		output["Totem"+elem+"ResistOverCap"] = math.Max(0, totemTotal-totemMax)
		output["MissingTotem"+elem+"Resist"] = math.Max(0, totemMax-totemFinal)
	}

	//pob2:block:src/Modules/CalcDefence.lua:999-1002
	if actor == env.Player {
		doActorLifeManaSpirit(actor)
		doActorLifeManaSpiritReservation(actor)
	}

	defenceBlock(env, actor)

	//pob2:block:src/Modules/CalcDefence.lua:1137-1199
	for _, conv := range []struct{ flag, target, from string }{
		{"EnergyShieldIncreasedByOvercappedColdRes", "EnergyShield", "ColdResistOverCap"},
		{"EnergyShieldIncreasedByUncappedColdRes", "EnergyShield", "ColdResistTotal"},
		{"ArmourIncreasedByUncappedFireRes", "Armour", "FireResistTotal"},
		{"ArmourIncreasedByOvercappedFireRes", "Armour", "FireResistOverCap"},
		{"EvasionRatingIncreasedByUncappedColdRes", "Evasion", "ColdResistTotal"},
		{"EvasionRatingIncreasedByUncappedLightningRes", "Evasion", "LightningResistTotal"},
		{"EvasionRatingIncreasedByOvercappedColdRes", "Evasion", "ColdResistOverCap"},
		{"EvasionRatingIncreasedByOvercappedLightningRes", "Evasion", "LightningResistOverCap"},
		{"AilmentThresholdIncreasedByUncappedChaosRes", "AilmentThreshold", "ChaosResistTotal"},
	} {
		if modDB.Flag(nil, conv.flag) {
			for _, value := range modDB.Tabulate(mod.TypeFlag, nil, conv.flag) {
				modDB.AddMod(newMod(conv.target, mod.TypeIncrease, output[conv.from], value.Mod.GetSource()))
				break
			}
		}
	}

	primaryDefences(env, actor)

	//pob2:block:src/Modules/CalcDefence.lua:1646-1652
	doActorLifeManaSpirit(actor)
	doActorLifeManaSpiritReservation(actor)
	doActorDarkness(actor)

	//pob2:block:src/Modules/CalcDefence.lua:1909-1923
	damageReductionCap := miscConstant("DamageReductionCap")
	if v := modDB.Max(nil, "DamageReductionMax"); v > 0 {
		output["DamageReductionMax"] = v
	} else {
		output["DamageReductionMax"] = damageReductionCap
	}
	modDB.AddMod(mod.NewFloat("ArmourAppliesToPhysicalDamageTaken", mod.TypeBase, 100))
	for _, damageType := range dmgTypeList {
		typeCap := damageReductionCap
		if v := modDB.Max(nil, damageType+"DamageReductionMax"); v > 0 {
			typeCap = v
		}
		output[damageType+"DamageReductionMax"] = math.Min(typeCap, output["DamageReductionMax"])

		names := []string{damageType + "DamageReduction"}
		if isElemental[damageType] {
			names = append(names, "ElementalDamageReduction")
		}
		base := math.Max(0, modDB.Sum(mod.TypeBase, nil, names...))
		typeMax := math.Min(base, output[damageType+"DamageReductionMax"])
		globalMax := math.Min(typeMax, output[damageType+"DamageReductionMax"])
		output["Base"+damageType+"DamageReduction"] = globalMax

		baseWhenHit := globalMax + modDB.Sum(mod.TypeBase, nil, damageType+"DamageReductionWhenHit")
		typeMaxWhenHit := math.Min(baseWhenHit, output[damageType+"DamageReductionMax"])
		globalMaxWhenHit := math.Min(typeMaxWhenHit, output[damageType+"DamageReductionMax"])
		output["Base"+damageType+"DamageReductionWhenHit"] = globalMaxWhenHit
	}

	// MovementSpeedEqualHighestLinkedPlayers reads actor.partyMembers, which no actor here carries.
	//pob2:block:src/Modules/CalcDefence.lua:1926-1930
	if o := modDB.Override(nil, "MovementSpeed"); o != nil {
		output["MovementSpeedMod"] = o.Float()
	} else {
		output["MovementSpeedMod"] = round((1+modDB.Sum(mod.TypeBase, nil, "MovementSpeed"))*calcLibMod(modDB, nil, "MovementSpeed"), 3)
	}
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		output["MovementSpeedMod"] = math.Max(output["MovementSpeedMod"], 1)
	}
	output["EffectiveMovementSpeedMod"] = round(output["MovementSpeedMod"]*output["ActionSpeedMod"], 3)
}

// The parent and party-member block chances read actor.parent and actor.partyMembers; a player has
// neither, so those branches are not ported.
//
//pob2:block:src/Modules/CalcDefence.lua:1005-1105
func defenceBlock(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output
	blockChanceCap := miscConstant("BlockChanceCap")

	if modDB.Flag(nil, "MaxBlockChanceModsApplyMaxResist") {
		if o := modDB.Override(nil, "BlockChanceMax"); o != nil {
			output["BlockChanceMax"] = o.Float()
		} else {
			output["BlockChanceMax"] = modDB.Sum(mod.TypeBase, nil, "BaseBlockChanceMax")
		}
	} else {
		if o := modDB.Override(nil, "BlockChanceMax"); o != nil {
			output["BlockChanceMax"] = math.Min(o.Float(), blockChanceCap)
		} else {
			output["BlockChanceMax"] = math.Min(modDB.Sum(mod.TypeBase, nil, "BaseBlockChanceMax")+modDB.Sum(mod.TypeBase, nil, "BlockChanceMax"), blockChanceCap)
		}
	}
	output["BlockChanceOverCap"] = 0
	output["SpellBlockChanceOverCap"] = 0
	baseBlockChance := 0.0
	if w := actor.ItemList["Weapon 2"]; w != nil && w.ArmourData != nil {
		baseBlockChance += w.ArmourData["BlockChance"]
	}
	if w := actor.ItemList["Weapon 3"]; w != nil && w.ArmourData != nil {
		baseBlockChance += w.ArmourData["BlockChance"]
	}
	output["ShieldBlockChance"] = baseBlockChance
	if o := modDB.Override(nil, "ReplaceShieldBlock"); o != nil {
		baseBlockChance = o.Float()
	}
	if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		output["BlockChance"] = output["BlockChanceMax"]
	} else {
		inc := modDB.Sum(mod.TypeIncrease, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalBlockChance := round((baseBlockChance + modDB.Sum(mod.TypeBase, nil, "BlockChance")) * (1 + inc/100) * more)
		output["BlockChance"] = math.Min(totalBlockChance, output["BlockChanceMax"])
		output["BlockChanceOverCap"] = math.Max(0, totalBlockChance-output["BlockChanceMax"])
	}
	output["ProjectileBlockChance"] = math.Min(output["BlockChance"]+modDB.Sum(mod.TypeBase, nil, "ProjectileBlockChance")*calcLibMod(modDB, nil, "BlockChance"), output["BlockChanceMax"])
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		output["SpellBlockChanceMax"] = output["BlockChanceMax"]
	} else if o := modDB.Override(nil, "BlockChanceMax"); o != nil {
		output["SpellBlockChanceMax"] = math.Min(o.Float(), blockChanceCap)
	} else {
		output["SpellBlockChanceMax"] = math.Min(modDB.Sum(mod.TypeBase, nil, "BaseSpellBlockChanceMax")+modDB.Sum(mod.TypeBase, nil, "SpellBlockChanceMax"), blockChanceCap)
	}
	switch {
	case modDB.Flag(nil, "MaxSpellBlockIfNotBlockedRecently"):
		output["SpellBlockChance"] = output["SpellBlockChanceMax"]
		output["SpellProjectileBlockChance"] = output["SpellBlockChanceMax"]
	case modDB.Flag(nil, "SpellBlockChanceIsBlockChance"):
		output["SpellBlockChance"] = output["BlockChance"]
		output["SpellProjectileBlockChance"] = output["ProjectileBlockChance"]
		output["SpellBlockChanceOverCap"] = output["BlockChanceOverCap"]
	default:
		inc := modDB.Sum(mod.TypeIncrease, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalSpellBlockChance := round(modDB.Sum(mod.TypeBase, nil, "SpellBlockChance") * (1 + inc/100) * more)
		output["SpellBlockChance"] = math.Min(totalSpellBlockChance, output["SpellBlockChanceMax"])
		output["SpellBlockChanceOverCap"] = math.Max(0, totalSpellBlockChance-output["SpellBlockChanceMax"])
		spellProjectileBlockChance := math.Min(output["SpellBlockChance"]+modDB.Sum(mod.TypeBase, nil, "ProjectileSpellBlockChance")*calcLibMod(modDB, nil, "SpellBlockChance"), output["SpellBlockChanceMax"])
		output["SpellProjectileBlockChance"] = math.Max(math.Max(spellProjectileBlockChance, output["ProjectileBlockChance"]), 0)
	}
	if modDB.Flag(nil, "CannotBlockAttacks") || enemyDB.Flag(nil, "CannotBeBlocked") {
		output["BlockChance"] = 0
		output["ProjectileBlockChance"] = 0
	}
	if modDB.Flag(nil, "CannotBlockSpells") || enemyDB.Flag(nil, "CannotBeBlocked") {
		output["SpellBlockChance"] = 0
		output["SpellProjectileBlockChance"] = 0
	}
	unluckyWorstOf := 1.0
	if v, ok := env.ConfigInput["EHPUnluckyWorstOf"]; ok && v.IsNumber {
		unluckyWorstOf = v.Number
	}
	for _, blockType := range []string{"BlockChance", "ProjectileBlockChance", "SpellBlockChance", "SpellProjectileBlockChance"} {
		if env.ModeEffective {
			output["Effective"+blockType] = math.Max(output[blockType]-enemyDB.Sum(mod.TypeBase, nil, "reduceEnemyBlock"), 0)
		} else {
			output["Effective"+blockType] = output[blockType]
		}
		luck := 1.0
		if env.ModeEffective {
			if modDB.Flag(nil, blockType+"IsLucky") {
				luck *= 2
			}
			if modDB.Flag(nil, blockType+"IsUnlucky") {
				luck /= 2
			}
		}
		luck /= unluckyWorstOf
		for luck != 1 {
			if luck > 1 {
				luck /= 2
				output["Effective"+blockType] = (1 - math.Pow(1-output["Effective"+blockType]/100, 2)) * 100
			} else {
				luck *= 2
				output["Effective"+blockType] = output["Effective"+blockType] / 100 * output["Effective"+blockType]
			}
		}
	}
	output["EffectiveAverageBlockChance"] = (output["EffectiveBlockChance"] + output["EffectiveProjectileBlockChance"] + output["EffectiveSpellBlockChance"] + output["EffectiveSpellProjectileBlockChance"]) / 4
	output["BlockEffect"] = 100 - modDB.Sum(mod.TypeBase, nil, "BlockEffect")
	if output["BlockEffect"] != 0 {
		output["DamageTakenOnBlock"] = 100 - output["BlockEffect"]
	}
}

// The ward and energy-shield-to-ward breakdown rows, and the Gear:* sums' breakdown rows, are UI only.
//
//pob2:block:src/Modules/CalcDefence.lua:1201-1579
func primaryDefences(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output

	doActorLifeManaSpirit(actor)
	var gearWard, gearEnergyShield, gearArmour, gearEvasion float64
	//pob2:block:src/Modules/CalcDefence.lua:1213-1305
	for _, slot := range armourSlots {
		item := actor.ItemList[slot]
		if item == nil || item.ArmourData == nil {
			continue
		}
		wardBase := itemArmourDataValue(item, "Ward", actor.Level)
		if wardBase > 0 {
			if slot == "Body Armour" && modDB.Flag(nil, "DoubleBodyArmourDefence") {
				wardBase *= 2
			}
			gearWard += wardBase
		}
		energyShieldBase := itemArmourDataValue(item, "EnergyShield", actor.Level)
		if energyShieldBase > 0 {
			if slot == "Body Armour" && modDB.Flag(nil, "DoubleBodyArmourDefence") {
				energyShieldBase *= 2
			}
			if modDB.Flag(nil, "EnergyShieldToWard") || !modDB.Flag(nil, "ConvertArmourESToLife") {
				gearEnergyShield += energyShieldBase
			}
		}
		armourBase := itemArmourDataValue(item, "Armour", actor.Level)
		if armourBase > 0 {
			if slot == "Body Armour" {
				if modDB.Flag(nil, "DoubleBodyArmourDefence") {
					armourBase *= 2
				}
				if modDB.Flag(nil, "Unbreakable") {
					armourBase *= 2
				}
				if modDB.Flag(nil, "ConvertBodyArmourArmourEvasionToWard") {
					armourBase *= 1 - modDB.Sum(mod.TypeBase, nil, "BodyArmourArmourEvasionToWardPercent")/100
				}
			}
			gearArmour += armourBase
		}
		evasionBase := itemArmourDataValue(item, "Evasion", actor.Level)
		if evasionBase > 0 {
			if slot == "Body Armour" {
				if modDB.Flag(nil, "DoubleBodyArmourDefence") {
					evasionBase *= 2
				}
				if modDB.Flag(nil, "Unbreakable") && modDB.Flag(nil, "IronReflexes") {
					evasionBase *= 2
				}
				if modDB.Flag(nil, "ConvertBodyArmourArmourEvasionToWard") {
					evasionBase *= 1 - modDB.Sum(mod.TypeBase, nil, "BodyArmourArmourEvasionToWardPercent")/100
				}
			}
			gearEvasion += evasionBase
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1356-1363
	var wardModsInc []string
	if modDB.Flag(nil, "EnergyShieldToWard") {
		wardModsInc = []string{"EnergyShield"}
	}
	newResource := func(name string, mods, modsInc, modsTotal []string, isDefence bool) *defenceResource {
		return &defenceResource{name: name, basePerSlot: map[string]float64{}, conversionRate: map[string]float64{}, mods: mods, modsInc: modsInc, modsTotal: modsTotal, defence: isDefence}
	}
	resourceList := []*defenceResource{
		newResource("Armour", []string{"Armour", "ArmourAndEvasion", "Defences"}, nil, []string{"ArmourTotal"}, true),
		newResource("Evasion", []string{"Evasion", "ArmourAndEvasion", "Defences"}, nil, []string{"EvasionTotal"}, true),
		newResource("EnergyShield", []string{"EnergyShield", "Defences"}, nil, []string{"EnergyShieldTotal"}, true),
		newResource("Life", []string{"Life"}, nil, []string{"LifeTotal"}, false),
		newResource("Mana", []string{"Mana"}, nil, []string{"ManaTotal"}, false),
		newResource("Ward", []string{"Ward", "Defences"}, wardModsInc, []string{"WardTotal"}, true),
	}
	//pob2:block:src/Modules/CalcDefence.lua:1364-1388
	for _, source := range resourceList {
		output[source.name] += 0
		totalConversion := 0.0
		for _, target := range resourceList {
			conversionRate := modDB.Sum(mod.TypeBase, nil, source.name+"ConvertTo"+target.name)
			if modDB.Flag(nil, source.name+"CannotBeConverted") {
				conversionRate = 0
			}
			source.conversionRate[target.name] = math.Min(conversionRate, 100)
			totalConversion += source.conversionRate[target.name]
		}
		// PoB rescales with ipairs over the name-keyed rates, which visits none, so only the total is capped.
		if totalConversion > 100 {
			totalConversion = 100
		}
		source.totalConversion = totalConversion
		for _, slot := range armourSlots {
			source.basePerSlot[slot] = itemArmourDataValue(actor.ItemList[slot], source.name, actor.Level)
			if source.name == "Ward" && slot == "Body Armour" && modDB.Flag(nil, "DoubleBodyArmourDefence") {
				source.basePerSlot[slot] *= 2
			}
		}
		source.totalBase = modDB.Sum(mod.TypeBase, nil, source.modsTotal...)
	}
	//pob2:block:src/Modules/CalcDefence.lua:1389-1443
	for _, source := range resourceList {
		globalBase := modDB.Sum(mod.TypeBase, nil, source.mods...) + source.globalBase
		if o := modDB.Override(nil, source.mods...); o != nil {
			globalBase = o.Float()
		}
		totalBase := source.totalBase
		for _, target := range resourceList {
			if source.name == target.name {
				continue
			}
			gainRate := modDB.Sum(mod.TypeBase, nil, source.name+"GainAs"+target.name)
			rate := source.conversionRate[target.name] + gainRate
			if rate <= 0 {
				continue
			}
			if source.defence {
				for _, slot := range armourSlots {
					if source.basePerSlot[slot] > 0 {
						targetBase := source.basePerSlot[slot] * rate / 100
						if target.defence {
							target.basePerSlot[slot] += targetBase
						} else {
							target.globalBase += targetBase
						}
						source.basePerSlot[slot] = source.basePerSlot[slot] * (100 - source.totalConversion) / 100
					}
				}
				target.globalBase += globalBase * rate / 100
				target.totalBase += totalBase * rate / 100
			} else {
				target.globalBase += round(globalBase * rate / 100)
				target.totalBase += round(totalBase * rate / 100)
			}
		}
		if source.defence {
			source.globalBase = globalBase * (100 - source.totalConversion) / 100
			source.totalBase = totalBase * (100 - source.totalConversion) / 100
		}
	}
	//pob2:block:src/Modules/CalcDefence.lua:1444-1458
	for _, res := range resourceList {
		if res.defence {
			for _, slot := range armourSlots {
				slotCfg := &moddb.ListCfg{SlotName: slot}
				inc, more := calcLibMods(modDB, slotCfg, res.mods...)
				if res.modsInc != nil {
					inc += modDB.Sum(mod.TypeIncrease, slotCfg, res.modsInc...) / 100
				}
				output[res.name] += res.basePerSlot[slot] * inc * more
			}
			inc, more := calcLibMods(modDB, nil, res.mods...)
			if res.modsInc != nil {
				inc += modDB.Sum(mod.TypeIncrease, nil, res.modsInc...) / 100
			}
			output[res.name] += res.globalBase*inc*more + res.totalBase
		} else {
			modDB.AddMod(newMod("Extra"+res.name, mod.TypeBase, res.globalBase, "Conversion"))
			modDB.AddMod(newMod(res.name+"Total", mod.TypeBase, res.totalBase, "Conversion"))
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1460-1487
	if o := modDB.Override(nil, "EnergyShield"); o != nil {
		output["MaximumEnergyShield"] = o.Float()
	} else {
		output["MaximumEnergyShield"] = math.Max(round(output["EnergyShield"]), 0)
	}
	output["EnergyShield"] = output["MaximumEnergyShield"]
	if modDB.Flag(nil, "CannotHaveES") {
		output["EnergyShield"] = 0
	}

	output["Armour"] = math.Max(round(output["Armour"]), 0)
	output["ArmourDefense"] = modDB.Max(nil, "ArmourDefense") / 100
	if output["ArmourDefense"] > 0 {
		output["RawArmourDefense"] = (1 + output["ArmourDefense"]) * 100
	}
	output["Evasion"] = math.Max(round(output["Evasion"]), 0)
	output["MeleeEvasion"] = math.Max(round(output["Evasion"]*calcLibMod(modDB, nil, "MeleeEvasion")), 0)
	output["ProjectileEvasion"] = math.Max(round(output["Evasion"]*calcLibMod(modDB, nil, "ProjectileEvasion")), 0)
	output["SpellEvasion"] = math.Max(round(output["Evasion"]*calcLibMod(modDB, nil, "SpellEvasion")), 0)
	output["SpellProjectileEvasion"] = math.Max(round(output["Evasion"]*calcLibMod(modDB, nil, "SpellProjectileEvasion")), 0)
	output["LowestOfArmourAndEvasion"] = math.Min(output["Armour"], output["Evasion"])
	output["Ward"] = math.Max(round(output["Ward"]), 0)
	output["Gear:Ward"] = gearWard
	output["Gear:EnergyShield"] = gearEnergyShield
	output["Gear:Armour"] = gearArmour
	output["Gear:Evasion"] = gearEvasion
	armourCap := modDB.Flag(nil, "ArmourESRecoveryCap")
	evasionCap := modDB.Flag(nil, "EvasionESRecoveryCap")
	lowES := env.ConfigInput["conditionLowEnergyShield"].Boolean
	cappingES := armourCap && output["Armour"] < output["EnergyShield"] || evasionCap && output["Evasion"] < output["EnergyShield"] || lowES
	if cappingES {
		switch {
		case armourCap && evasionCap:
			output["EnergyShieldRecoveryCap"] = math.Min(output["Armour"], output["Evasion"])
		case armourCap:
			output["EnergyShieldRecoveryCap"] = output["Armour"]
		case evasionCap:
			output["EnergyShieldRecoveryCap"] = output["Evasion"]
		default:
			output["EnergyShieldRecoveryCap"] = output["EnergyShield"]
		}
		if lowES {
			output["EnergyShieldRecoveryCap"] = math.Min(output["EnergyShield"]*miscConstant("LowPoolThreshold"), output["EnergyShieldRecoveryCap"])
		}
	} else {
		output["EnergyShieldRecoveryCap"] = output["EnergyShield"]
	}

	//pob2:block:src/Modules/CalcDefence.lua:1489-1494
	enemyAccuracy := round(calcLibVal(enemyDB, "Accuracy", nil))
	if modDB.Flag(nil, "EnemyAccuracyDistancePenalty") {
		enemyDistance := math.Max(math.Min(modDB.Sum(mod.TypeBase, nil, "Multiplier:enemyDistance"), 120), 20)
		accuracyPenalty := 1 - ((enemyDistance-20)/100)*(1-0.1)
		enemyAccuracy = math.Floor(enemyAccuracy * accuracyPenalty)
	}

	//pob2:block:src/Modules/CalcDefence.lua:1496-1563
	evadeKeys := []string{"EvadeChance", "MeleeEvadeChance", "ProjectileEvadeChance", "SpellEvadeChance", "SpellProjectileEvadeChance"}
	switch {
	case modDB.Flag(nil, "CannotEvade") || enemyDB.Flag(nil, "CannotBeEvaded"):
		for _, k := range evadeKeys {
			output[k] = 0
		}
	case modDB.Flag(nil, "AlwaysEvade"):
		for _, k := range evadeKeys {
			output[k] = 100
		}
	default:
		evadeChance := modDB.Sum(mod.TypeBase, nil, "EvadeChance")
		hitChance := calcLibMod(enemyDB, nil, "HitChance")
		evadeMax := miscConstant("EvadeChanceCap")
		if v := modDB.Max(nil, "EvadeChanceMax"); v > 0 {
			evadeMax = v
		}
		evade := func(evasion float64, names ...string) float64 {
			return math.Max(0, math.Min(evadeMax, (100-(monsterHitChance(evasion, enemyAccuracy)-evadeChance)*hitChance)*calcLibMod(modDB, nil, names...)))
		}
		output["EvadeChance"] = 100 - (monsterHitChance(output["Evasion"], enemyAccuracy)-evadeChance)*hitChance
		output["MeleeEvadeChance"] = evade(output["MeleeEvasion"], "EvadeChance", "MeleeEvadeChance")
		output["ProjectileEvadeChance"] = evade(output["ProjectileEvasion"], "EvadeChance", "ProjectileEvadeChance")
		output["SpellEvadeChance"] = evade(output["SpellEvasion"], "EvadeChance", "SpellEvadeChance")
		output["SpellProjectileEvadeChance"] = evade(output["SpellProjectileEvasion"], "EvadeChance", "ProjectileEvadeChance", "SpellProjectileEvadeChance")
		if !(output["MeleeEvadeChance"] != output["ProjectileEvadeChance"] && output["MeleeEvadeChance"] != output["SpellEvadeChance"] && output["MeleeEvadeChance"] != output["SpellProjectileEvadeChance"]) {
			output["EvadeChance"] = output["MeleeEvadeChance"]
		}
		output["EvadeChance"] = math.Min(output["EvadeChance"], evadeMax)
		if modDB.Flag(nil, "UnluckyEvade") {
			for _, k := range evadeKeys {
				output[k] = output[k] * output[k] / 100
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1565-1571
	output["DeflectionRating"] = modDB.Sum(mod.TypeBase, nil, "DeflectionRating") + (output["Evasion"]*modDB.Sum(mod.TypeBase, nil, "EvasionGainAsDeflection")/100+output["Armour"]*modDB.Sum(mod.TypeBase, nil, "ArmourGainAsDeflection")/100)*calcLibMod(modDB, nil, "DeflectionRating")
	output["DeflectChance"] = deflectChance(output["DeflectionRating"], enemyAccuracy)
	if modDB.Flag(nil, "DeflectIsLucky") {
		notDeflect := 1 - output["DeflectChance"]/100
		output["DeflectChance"] = (1 - notDeflect*notDeflect) * 100
	}
	output["DeflectEffect"] = math.Min(math.Max(miscConstant("DeflectEffect")+modDB.Sum(mod.TypeBase, nil, "DeflectEffect"), 0), 100)
}
