package calculator2

import (
	"math"
	"slices"
	"strings"

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

//pob2:src/Modules/Common.lua:734-741
func floor(val float64, dec ...int) float64 {
	if len(dec) > 0 {
		mult := math.Pow(10, float64(dec[0]))
		return math.Floor(val*mult+0.0001) / mult
	}
	return math.Floor(val)
}

// reservationPool is one pool entry of doActorLifeManaSpiritReservation's per-skill table.
type reservationPool struct {
	baseFlat, basePercent, more, inc, efficiency, efficiencyMore, reservedFlat, reservedPercent float64
}

// skillReservationValue is activeSkill.skillData[key] or activeSkill.activeEffect.grantedEffectLevel[key] or 0.
func skillReservationValue(activeSkill *ActiveSkill, key string) float64 {
	if v, ok := activeSkill.SkillData[key].(float64); ok {
		return v
	}
	if level := activeSkill.ActiveEffect.GrantedEffectLevel; level != nil {
		if v, ok := level.Extra[key]; ok {
			return v
		}
	}
	return 0
}

// PoE2's gem data has no vaalGem, so the vaal branch never runs.
//
//pob2:src/Modules/CalcDefence.lua:149-173
func getActiveSkillCount(activeSkill *ActiveSkill) (float64, bool) {
	switch {
	case activeSkill.SocketGroup == nil:
		return 1, true
	case activeSkill.SocketGroup.HasGroupCount:
		return activeSkill.SocketGroup.GroupCount, true
	default:
		for _, gemData := range activeSkill.SocketGroup.GemList {
			if gemData.GemData != nil {
				if (activeSkill.ActiveEffect.GrantedEffect == gemData.GemData.GrantedEffect && !gemData.GemData.GrantedEffect.Support) || slices.Contains(gemData.GemData.AdditionalGrantedEffects, activeSkill.ActiveEffect.GrantedEffect) {
					return gemData.Count, true
				}
			}
		}
	}
	return 1, true
}

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
	for _, activeSkill := range actor.ActiveSkillList {
		activeInWeaponSet := activeSkill.Actor == actor || truthy(activeSkill.SkillData["reservesInAllWeaponSets"])
		isTotemAndAncestralBond := activeSkill.SkillTypes["SummonsTotem"] && modDB.Flag(nil, "AncestralBond")
		if activeInWeaponSet && ((activeSkill.SkillTypes["HasReservation"] || truthy(activeSkill.SkillData["SupportedByAutoexertion"])) && !activeSkill.SkillTypes["ReservationBecomesCost"] || isTotemAndAncestralBond) {
			skillModList := activeSkill.SkillModList
			skillCfg := activeSkill.SkillCfg
			mult := floor(skillModList.More(skillCfg, "ReservationMultiplier"), 4)
			pool := map[string]*reservationPool{"Mana": {}, "Life": {}, "Spirit": {}}
			// PoB also writes the minion's name into the shared grantedEffect.name for the breakdown; data.skills
			// is shared across builds here, so only the gem instance's nameSpec takes it.
			//pob2:block:src/Modules/CalcDefence.lua:205-216
			if srcInstance := activeSkill.ActiveEffect.SrcInstance; srcInstance != nil && srcInstance.DisplayEffect != nil && activeSkill.Minion != nil {
				if strings.HasPrefix(skillCfg.SkillName, "Companion:") {
					activeSkill.SkillData["spiritReservationPercent"] = activeSkill.Minion.MinionData.CompanionReservation
					srcInstance.NameSpec = "Companion: " + activeSkill.Minion.MinionData.Name // for socket group
				}
				if strings.HasPrefix(skillCfg.SkillName, "Spectre:") {
					activeSkill.SkillData["spiritReservationFlat"] = float64(activeSkill.Minion.MinionData.SpectreReservation)
					srcInstance.NameSpec = "Spectre: " + activeSkill.Minion.MinionData.Name // for socket group
				}
			}
			pool["Mana"].baseFlat = skillReservationValue(activeSkill, "manaReservationFlat")
			pool["Spirit"].baseFlat = skillReservationValue(activeSkill, "spiritReservationFlat")
			pool["Spirit"].baseFlat = pool["Spirit"].baseFlat + skillModList.Sum(mod.TypeBase, skillCfg, "ExtraSpirit")
			grantedEffectLevel := activeSkill.ActiveEffect.GrantedEffectLevel
			if skillModList.Flag(skillCfg, "ManaCostGainAsReservation") && grantedEffectLevel != nil && grantedEffectLevel.Cost != nil {
				pool["Spirit"].baseFlat = skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostBase") + grantedEffectLevel.Cost["Mana"]
			}
			pool["Spirit"].basePercent = skillReservationValue(activeSkill, "spiritReservationPercent")
			pool["Mana"].basePercent = skillReservationValue(activeSkill, "manaReservationPercent")
			pool["Life"].baseFlat = skillReservationValue(activeSkill, "lifeReservationFlat")
			if skillModList.Flag(skillCfg, "LifeCostGainAsReservation") && grantedEffectLevel != nil && grantedEffectLevel.Cost != nil {
				pool["Life"].baseFlat = skillModList.Sum(mod.TypeBase, skillCfg, "LifeCostBase") + grantedEffectLevel.Cost["Life"]
			}
			pool["Life"].basePercent = skillReservationValue(activeSkill, "lifeReservationPercent")
			if srcInstance := activeSkill.ActiveEffect.SrcInstance; activeSkill.SkillTypes["IsBlasphemy"] && srcInstance != nil && srcInstance.SupportEffect != nil && srcInstance.SupportEffect.IsSupporting != nil {
				instances := float64(len(srcInstance.SupportEffect.IsSupporting))
				for name, values := range pool {
					v, _ := activeSkill.SkillData["blasphemyReservationFlat"+name].(float64)
					values.baseFlat = values.baseFlat + v*instances
				}
			}
			if skillModList.Flag(skillCfg, "BloodMagicReserved") {
				pool["Life"].baseFlat = pool["Life"].baseFlat + pool["Mana"].baseFlat
				pool["Mana"].baseFlat = 0
				activeSkill.SkillData["LifeReservationFlatForced"] = activeSkill.SkillData["ManaReservationFlatForced"]
				delete(activeSkill.SkillData, "ManaReservationFlatForced")
				pool["Life"].basePercent = pool["Life"].basePercent + pool["Mana"].basePercent
				pool["Mana"].basePercent = 0
				activeSkill.SkillData["LifeReservationPercentForced"] = activeSkill.SkillData["ManaReservationPercentForced"]
				delete(activeSkill.SkillData, "ManaReservationPercentForced")
			}
			spiritToLifeReservation := skillModList.Sum(mod.TypeBase, skillCfg, "LifeReservePercentPerSpirit")
			if spiritToLifeReservation > 0 {
				pool["Life"].basePercent = pool["Life"].basePercent + pool["Spirit"].baseFlat*spiritToLifeReservation
				pool["Spirit"].baseFlat = 0
				pool["Life"].basePercent = pool["Life"].basePercent + pool["Spirit"].basePercent*spiritToLifeReservation
				pool["Spirit"].basePercent = 0
			}
			for _, name := range []string{"Mana", "Life", "Spirit"} {
				values := pool[name]
				values.more = skillModList.More(skillCfg, name+"Reserved", "Reserved")
				values.inc = skillModList.Sum(mod.TypeIncrease, skillCfg, name+"Reserved", "Reserved")
				values.efficiency = math.Max(skillModList.Sum(mod.TypeIncrease, skillCfg, name+"ReservationEfficiency", "ReservationEfficiency"), -100)
				values.efficiencyMore = skillModList.More(skillCfg, name+"ReservationEfficiency", "ReservationEfficiency")
				// TODO: actor[name.."Efficiency"] feeds ModStore:GetStat's Arcane Cloak case, which is not ported.
				if forced, ok := activeSkill.SkillData[name+"ReservationFlatForced"].(float64); ok {
					values.reservedFlat = forced
				} else {
					baseFlatVal := values.baseFlat * mult
					values.reservedFlat = 0
					if values.more > 0 && values.inc > -100 && baseFlatVal != 0 {
						values.reservedFlat = math.Max(round(baseFlatVal*(100+values.inc)/100*values.more/(1+values.efficiency/100)/values.efficiencyMore, 0), 0)
					}
				}
				if forced, ok := activeSkill.SkillData[name+"ReservationPercentForced"].(float64); ok {
					values.reservedPercent = forced
				} else {
					basePercentVal := values.basePercent * mult
					values.reservedPercent = 0
					if values.more > 0 && values.inc > -100 && basePercentVal != 0 {
						values.reservedPercent = math.Max(round(basePercentVal*(100+values.inc)/100*values.more/(1+values.efficiency/100)/values.efficiencyMore, 2), 0)
					}
				}
				// activeMineCount and activeStageCount are nil in PoB until set; 0 stands for nil here.
				if activeSkill.ActiveMineCount != 0 {
					values.reservedFlat = values.reservedFlat * activeSkill.ActiveMineCount
					values.reservedPercent = values.reservedPercent * activeSkill.ActiveMineCount
				}
				if activeSkill.SkillTypes["MultipleReservation"] || isTotemAndAncestralBond {
					activeSkillCount, _ := getActiveSkillCount(activeSkill)
					minionFreeSpiritCount := skillModList.Sum(mod.TypeBase, skillCfg, "MinionFreeSpiritCount")
					values.reservedFlat = values.reservedFlat * math.Max(activeSkillCount-minionFreeSpiritCount, 0)
				}
				// Blood Sacrament increases reservation per stage channelled
				if skillCfg.SkillName == "Blood Sacrament" && activeSkill.ActiveStageCount != 0 {
					values.reservedFlat = values.reservedFlat * (activeSkill.ActiveStageCount + 1)
					values.reservedPercent = values.reservedPercent * (activeSkill.ActiveStageCount + 1)
				}
				if values.reservedFlat != 0 {
					activeSkill.SkillData[name+"ReservedBase"] = values.reservedFlat
					reservedBase[name] = reservedBase[name] + values.reservedFlat
				}
				if values.reservedPercent != 0 {
					activeSkill.SkillData[name+"ReservedPercent"] = values.reservedPercent
					activeSkill.SkillData[name+"ReservedBase"] = values.reservedFlat + math.Ceil(output[name]*values.reservedPercent/100)
					reservedPercent[name] = reservedPercent[name] + values.reservedPercent
				}
				if skillModList.Flag(skillCfg, "HasUncancellableReservation") {
					uncancellable[name] = uncancellable[name] + values.reservedPercent
				}
			}
		}
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

// Boolean outputs (EnergyShieldRechargeAppliesTo*, specificTypeAvoidance, the *Immunity flags,
// showDebuffExpirationModifier) are not carried by this float output table.
//
//pob2:src/Modules/CalcDefence.lua:790-2051
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
	if actor == env.minionActor() || actor == env.Player {
		doActorLifeManaSpirit(actor)
		doActorLifeManaSpiritReservation(actor)
	}

	defenceBlock(env, actor)

	//pob2:block:src/Modules/CalcDefence.lua:1107-1136
	if modDB.Flag(nil, "ArmourAppliesToEnergyShieldRecharge") {
		multiplier := 1.0
		if v := modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge"); v != 0 {
			multiplier = v / 100
		}
		for _, value := range modDB.Tabulate(mod.TypeIncrease, nil, "Armour", "ArmourAndEvasion", "Defences") {
			m := value.Mod
			modDB.AddMod(newMod("EnergyShieldRecharge", mod.TypeIncrease, math.Floor(m.Value().Float()*multiplier), m.GetSource(), getConvertedModTags(m, multiplier)...).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()))
		}
	}
	for _, conv := range []struct{ flag, from, to string }{
		{"MovementSpeedAppliesToEnergyShieldRecharge", "MovementSpeed", "EnergyShieldRecharge"},
		{"ManaRegenAppliesToEnergyShieldRecharge", "ManaRegen", "EnergyShieldRecharge"},
		{"LifeRegenAppliesToManaRegen", "LifeRegen", "ManaRegen"},
	} {
		if modDB.Flag(nil, conv.flag) {
			for _, value := range modDB.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, conv.from) {
				modDB.AddMod(value.Mod.Clone().SetName(conv.to))
			}
		}
	}

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
	spellSuppressionChance := defenceSuppressionDodge(env, actor)

	//pob2:block:src/Modules/CalcDefence.lua:1646-1652
	doActorLifeManaSpirit(actor)
	doActorLifeManaSpiritReservation(actor)
	doActorDarkness(actor)

	defenceRecovery(actor)

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

	defenceAvoidance(actor, spellSuppressionChance)
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

//pob2:src/Modules/CalcTools.lua:238-253
func getConvertedModTags(m mod.Mod, multiplier float64) []mod.Tag {
	modifiers := make([]mod.Tag, len(m.Tags()))
	for k, value := range m.Tags() {
		switch x := value.(type) {
		case *mod.MultiplierTag:
			c := *x
			if c.TagLimitTotal && c.TagLimit != nil {
				limit := *c.TagLimit * multiplier
				c.TagLimit = &limit
			}
			modifiers[k] = &c
		case *mod.PerStatTag:
			c := *x
			if c.TagLimitTotal && c.TagLimit != nil {
				limit := *c.TagLimit * multiplier
				c.TagLimit = &limit
			}
			modifiers[k] = &c
		default:
			modifiers[k] = value
		}
	}
	return modifiers
}

//pob2:block:src/Modules/Data.lua:409-412
var (
	dataAilmentTypeList             = []string{"Bleed", "Poison", "Ignite", "Chill", "Freeze", "Shock"}
	dataElementalAilmentTypeList    = []string{"Ignite", "Chill", "Freeze", "Shock"}
	dataNonElementalAilmentTypeList = []string{"Bleed", "Poison"}
)

// defenceSuppressionDodge returns the uncapped spell suppression sum, which the ailment-avoidance
// block reads again.
//
//pob2:block:src/Modules/CalcDefence.lua:1581-1630
func defenceSuppressionDodge(env *Environment, actor *Actor) float64 {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output

	spellSuppressionChance := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance")
	totalSpellSuppressionChance := spellSuppressionChance
	if o := modDB.Override(nil, "SpellSuppressionChance"); o != nil {
		totalSpellSuppressionChance = o.Float()
	}

	if modDB.Flag(nil, "ConvertSpellSuppressionToSpellDodge") {
		modDB.AddMod(newMod("SpellDodgeChance", mod.TypeBase, spellSuppressionChance/2, "Acrobatics"))
	}

	output["SpellSuppressionChance"] = math.Min(totalSpellSuppressionChance, miscConstant("SuppressionChanceCap"))
	output["SpellSuppressionEffect"] = math.Max(miscConstant("SuppressionEffect")+modDB.Sum(mod.TypeBase, nil, "SpellSuppressionEffect"), 0)

	if enemyDB.Flag(nil, "CannotBeSuppressed") {
		output["EffectiveSpellSuppressionChance"] = 0
	} else {
		output["EffectiveSpellSuppressionChance"] = output["SpellSuppressionChance"]
	}
	luck := 1.0
	if env.ModeEffective {
		if modDB.Flag(nil, "SpellSuppressionChanceIsLucky") {
			luck *= 2
		}
		if modDB.Flag(nil, "SpellSuppressionChanceIsUnlucky") {
			luck /= 2
		}
	}
	if v, ok := env.ConfigInput["EHPUnluckyWorstOf"]; ok && v.IsNumber {
		luck /= v.Number
	}
	for luck != 1 {
		if luck > 1 {
			luck /= 2
			output["EffectiveSpellSuppressionChance"] = (1 - math.Pow(1-output["EffectiveSpellSuppressionChance"]/100, 2)) * 100
		} else {
			luck *= 2
			output["EffectiveSpellSuppressionChance"] = output["EffectiveSpellSuppressionChance"] / 100 * output["EffectiveSpellSuppressionChance"]
		}
	}

	output["SpellSuppressionChanceOverCap"] = math.Max(0, totalSpellSuppressionChance-miscConstant("SuppressionChanceCap"))

	totalAttackDodgeChance := modDB.Sum(mod.TypeBase, nil, "AttackDodgeChance")
	totalSpellDodgeChance := modDB.Sum(mod.TypeBase, nil, "SpellDodgeChance")
	attackDodgeChanceMax := miscConstant("DodgeChanceCap")
	spellDodgeChanceMax := modDB.Sum(mod.TypeBase, nil, "SpellDodgeChanceMax")
	if o := modDB.Override(nil, "SpellDodgeChanceMax"); o != nil {
		spellDodgeChanceMax = o.Float()
	}
	enemyReduceDodgeChance := enemyDB.Sum(mod.TypeBase, nil, "reduceEnemyDodge")

	cannotBeDodged := enemyDB.Flag(nil, "CannotBeDodged")
	output["AttackDodgeChance"] = math.Min(totalAttackDodgeChance, attackDodgeChanceMax)
	output["EffectiveAttackDodgeChance"] = 0
	if !cannotBeDodged {
		output["EffectiveAttackDodgeChance"] = math.Min(math.Max(totalAttackDodgeChance-enemyReduceDodgeChance, 0), attackDodgeChanceMax)
	}
	output["SpellDodgeChance"] = math.Min(totalSpellDodgeChance, spellDodgeChanceMax)
	output["EffectiveSpellDodgeChance"] = 0
	if !cannotBeDodged {
		output["EffectiveSpellDodgeChance"] = math.Min(math.Max(totalSpellDodgeChance-enemyReduceDodgeChance, 0), spellDodgeChanceMax)
	}
	if env.ModeEffective && modDB.Flag(nil, "DodgeChanceIsUnlucky") {
		output["EffectiveAttackDodgeChance"] = output["EffectiveAttackDodgeChance"] / 100 * output["EffectiveAttackDodgeChance"]
		output["EffectiveSpellDodgeChance"] = output["EffectiveSpellDodgeChance"] / 100 * output["EffectiveSpellDodgeChance"]
	}
	output["AttackDodgeChanceOverCap"] = math.Max(0, totalAttackDodgeChance-attackDodgeChanceMax)
	output["SpellDodgeChanceOverCap"] = math.Max(0, totalSpellDodgeChance-spellDodgeChanceMax)
	return spellSuppressionChance
}

// recoveryResources is the regeneration block's `resources`.
var recoveryResources = []string{"Mana", "Life", "Energy Shield", "Rage", "Ward"}

// The parent and party-member leech-rate branches read actor.parent and actor.partyMembers; a player
// has neither, so those branches are not ported.
//
//pob2:block:src/Modules/CalcDefence.lua:1654-1906
func defenceRecovery(actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output

	//pob2:block:src/Modules/CalcDefence.lua:1655-1666
	if modDB.Conditions["AffectedByArcaneSurge"] || modDB.Flag(nil, "Condition:ArcaneSurge") {
		modDB.Conditions["AffectedByArcaneSurge"] = true
		effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ArcaneSurgeEffect", "BuffEffectOnSelf")/100
		maxOr := func(name string, fallback float64) float64 {
			if v := modDB.Max(nil, name); v != 0 {
				return v
			}
			return fallback
		}
		if modDB.Flag(nil, "ArcaneSurgeLifeRegen") {
			modDB.AddMod(newMod("LifeRegen", mod.TypeMore, maxOr("ArcaneSurgeManaRegen", 20)*effect, "Arcane Surge"))
		} else {
			modDB.AddMod(newMod("ManaRegen", mod.TypeMore, maxOr("ArcaneSurgeManaRegen", 20)*effect, "Arcane Surge"))
		}
		modDB.AddMod(newMod("Speed", mod.TypeIncrease, maxOr("ArcaneSurgeCastSpeed", 15)*effect, "Arcane Surge").Flag(modFlagCast))
		if arcaneSurgeDamage := maxOr("ArcaneSurgeDamage", 0); arcaneSurgeDamage != 0 {
			modDB.AddMod(newMod("Damage", mod.TypeMore, arcaneSurgeDamage*effect, "Arcane Surge").Flag(modFlagSpell))
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1669-1674
	output["LifeRecoveryRateMod"] = 1
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output["LifeRecoveryRateMod"] = calcLibMod(modDB, nil, "LifeRecoveryRate")
	}
	output["ManaRecoveryRateMod"] = calcLibMod(modDB, nil, "ManaRecoveryRate")
	output["EnergyShieldRecoveryRateMod"] = calcLibMod(modDB, nil, "EnergyShieldRecoveryRate")

	//pob2:block:src/Modules/CalcDefence.lua:1677-1702
	output["MaxLifeLeechInstance"] = output["Life"] * calcLibVal(modDB, "MaxLifeLeechInstance", nil) / 100
	output["MaxLifeLeechRatePercent"] = calcLibVal(modDB, "MaxLifeLeechRate", nil)
	output["MaxLifeLeechRate"] = output["Life"] * output["MaxLifeLeechRatePercent"] / 100
	output["MaxEnergyShieldLeechInstance"] = output["EnergyShield"] * calcLibVal(modDB, "MaxEnergyShieldLeechInstance", nil) / 100
	output["MaxEnergyShieldLeechRate"] = output["EnergyShield"] * calcLibVal(modDB, "MaxEnergyShieldLeechRate", nil) / 100
	output["MaxManaLeechInstance"] = output["Mana"] * calcLibVal(modDB, "MaxManaLeechInstance", nil) / 100
	output["MaxManaLeechRate"] = output["Mana"] * calcLibVal(modDB, "MaxManaLeechRate", nil) / 100

	//pob2:block:src/Modules/CalcDefence.lua:1712-1789
	for i, resourceName := range recoveryResources {
		resource := strings.ReplaceAll(resourceName, " ", "")
		pool := output[resource]
		baseRegen := 0.0
		inc := modDB.Sum(mod.TypeIncrease, nil, resource+"Regen", resource+"RecoveryRate")
		more := modDB.More(nil, resource+"Regen", resource+"RecoveryRate")
		regenRate := 0.0
		recoveryRateMod, ok := output[resource+"RecoveryRateMod"]
		if !ok {
			recoveryRateMod = 1
		}
		if modDB.Flag(nil, "No"+resource+"Regen") || modDB.Flag(nil, "CannotGain"+resource) {
			output[resource+"Regen"] = 0
		} else {
			if inc != 0 {
				for _, next := range recoveryResources[i+1:] {
					nextResource := strings.ReplaceAll(next, " ", "")
					if modDB.Flag(nil, resource+"RegenTo"+nextResource+"Regen") {
						modDB.AddMod(newMod(nextResource+"Regen", mod.TypeIncrease, inc, mod.Source(resourceName+" instead applies to "+next)))
						inc = 0
					}
				}
			}
			baseRegen = modDB.Sum(mod.TypeBase, nil, resource+"Regen") + pool*modDB.Sum(mod.TypeBase, nil, resource+"RegenPercent")/100
			regen := baseRegen * (1 + inc/100) * more
			if regen != 0 {
				for _, next := range recoveryResources[i+1:] {
					nextResource := strings.ReplaceAll(next, " ", "")
					if modDB.Flag(nil, resource+"RegenerationRecovers"+nextResource) {
						modDB.AddMod(newMod(nextResource+"Recovery", mod.TypeBase, regen, mod.Source(resourceName+" Regeneration Recovers "+next)))
					}
				}
			}
			regenRate = round(regen, 1)
			output[resource+"Regen"] = regenRate
		}
		output[resource+"RegenInc"] = inc
		baseDegen := modDB.Sum(mod.TypeBase, nil, resource+"Degen") + pool*modDB.Sum(mod.TypeBase, nil, resource+"DegenPercent")/100
		degenRate := 0.0
		if baseDegen > 0 {
			degenRate = baseDegen * calcLibMod(modDB, nil, resource+"Degen")
		}
		output[resource+"Degen"] = degenRate
		recoveryRate := modDB.Sum(mod.TypeBase, nil, resource+"Recovery") * recoveryRateMod
		output[resource+"Recovery"] = recoveryRate
		overflowRecovery := modDB.Sum(mod.TypeBase, nil, "Overflow"+resource+"Recovery")
		regenPart := regenRate
		if modDB.Flag(nil, "UnaffectedBy"+resource+"Regen") {
			regenPart = 0
		}
		output[resource+"RegenRecovery"] = regenPart - degenRate + recoveryRate + overflowRecovery
		if resource == "Life" && modDB.Flag(nil, "ZealotsOath") && output[resource+"RegenRecovery"] > 0 {
			modDB.AddMod(newMod("OverflowEnergyShieldRecovery", mod.TypeBase, output["LifeRegenRecovery"]-recoveryRate, "Zealot's Oath"))
			output[resource+"RegenRecovery"] = 0
		}
		if output[resource+"RegenRecovery"] > 0 {
			modDB.AddMod(newFlagMod("Condition:CanGain"+resource, mod.Source(resourceName+"Regen")))
		}
		output[resource+"RegenPercent"] = 0
		if pool > 0 {
			output[resource+"RegenPercent"] = round(output[resource+"RegenRecovery"]/pool*100, 1)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1792-1850
	rechargeAppliesToLife := modDB.Flag(nil, "EnergyShieldRechargeAppliesToLife") && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech")
	rechargeAppliesToEnergyShield := !(modDB.Flag(nil, "NoEnergyShieldRecharge") || modDB.Flag(nil, "CannotGainEnergyShield") || rechargeAppliesToLife)
	if rechargeAppliesToLife || rechargeAppliesToEnergyShield {
		inc := modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRecharge")
		more := modDB.More(nil, "EnergyShieldRecharge")
		if rechargeAppliesToLife {
			recharge := output["Life"] * miscConstant("EnergyShieldRechargeBase") * (1 + inc/100) * more
			output["LifeRecharge"] = round(recharge*output["LifeRecoveryRateMod"], 1)
		} else {
			recharge := output["EnergyShield"] * miscConstant("EnergyShieldRechargeBase") * (1 + inc/100) * more
			output["EnergyShieldRecharge"] = round(recharge*output["EnergyShieldRecoveryRateMod"], 1)
		}
		rechargeBase := miscConstant("EnergyShieldRechargeDelay") + modDB.Sum(mod.TypeBase, nil, "EnergyShieldRechargeFaster")
		if o := modDB.Override(nil, "EnergyShieldRechargeBase"); o != nil {
			rechargeBase = o.Float()
		}
		output["EnergyShieldRechargeDelay"] = rechargeBase / (1 + modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100)
	} else {
		output["EnergyShieldRecharge"] = 0
	}

	//pob2:block:src/Modules/CalcDefence.lua:1854-1905
	output["anyRecoup"] = 0
	recoupTypeList := []string{"Life", "Mana", "EnergyShield"}
	for _, recoupType := range recoupTypeList {
		recoup := modDB.Sum(mod.TypeBase, nil, recoupType+"Recoup")
		calcRecoup(output, recoup, recoupType, "")
		if modDB.Flag(nil, "Add"+recoupType+"RecoupToEnergyShieldRecoup") {
			source := modDB.Tabulate(mod.TypeFlag, nil, "Add"+recoupType+"RecoupToEnergyShieldRecoup")[0].Mod.GetSource()
			modDB.ReplaceMod(newMod("EnergyShieldRecoup", mod.TypeBase, recoup, source))
		}
	}
	for _, recoupType := range recoupTypeList {
		for _, damageType := range dmgTypeList {
			recoup := modDB.Sum(mod.TypeBase, nil, damageType+recoupType+"Recoup")
			calcRecoup(output, recoup, recoupType, damageType)
			if modDB.Flag(nil, "Add"+recoupType+"RecoupToEnergyShieldRecoup") {
				source := modDB.Tabulate(mod.TypeFlag, nil, "Add"+recoupType+"RecoupToEnergyShieldRecoup")[0].Mod.GetSource()
				modDB.ReplaceMod(newMod(damageType+"EnergyShieldRecoup", mod.TypeBase, recoup, source))
			}
		}
	}
	for _, resource := range recoupTypeList {
		if !modDB.Flag(nil, "No"+resource+"Regen") && !modDB.Flag(nil, "CannotGain"+resource) {
			pseudoRecoup := modDB.Sum(mod.TypeBase, nil, "PhysicalDamageMitigated"+resource+"PseudoRecoup")
			if pseudoRecoup > 0 {
				durationKey := "PhysicalDamageMitigated" + resource + "PseudoRecoupDuration"
				output[durationKey] = modDB.Sum(mod.TypeBase, nil, durationKey)
				if output[durationKey] == 0 {
					output[durationKey] = 4
				}
				inc := modDB.Sum(mod.TypeIncrease, nil, resource+"Regen")
				more := modDB.More(nil, resource+"Regen")
				output["PhysicalDamageMitigated"+resource+"PseudoRecoup"] = pseudoRecoup * (1 + inc/100) * more * output[resource+"RecoveryRateMod"]
				output["anyRecoup"] += output["PhysicalDamageMitigated"+resource+"PseudoRecoup"]
			}
		}
	}
}

//pob2:src/Modules/CalcDefence.lua:774-788
func calcRecoup(output map[string]float64, recoup float64, recoupType, damageType string) {
	output[damageType+recoupType+"Recoup"] = recoup * output[recoupType+"RecoveryRateMod"]
	output["anyRecoup"] += output[damageType+recoupType+"Recoup"]
}

//pob2:block:src/Modules/CalcDefence.lua:1940-2050
func defenceAvoidance(actor *Actor, spellSuppressionChance float64) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output

	if enemyDB.Flag(nil, "Blind") {
		output["BlindEffectMod"] = calcLibMod(enemyDB, nil, "BlindEffect", "BuffEffectOnSelf") * 100
	}

	output["WeaponSwapSpeed"] = modDB.Sum(mod.TypeBase, nil, "WeaponSwapSpeed")
	output["WeaponSwapSpeedMod"] = calcLibMod(modDB, nil, "WeaponSwapSpeed")
	output["WeaponSwapSpeed"] = output["WeaponSwapSpeed"] / output["WeaponSwapSpeedMod"]

	//pob2:block:src/Modules/CalcDefence.lua:1949-1962
	output["LifeOnBlock"] = 0
	output["LifeOnSuppress"] = 0
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output["LifeOnBlock"] = modDB.Sum(mod.TypeBase, nil, "LifeOnBlock")
		output["LifeOnSuppress"] = modDB.Sum(mod.TypeBase, nil, "LifeOnSuppress")
	}
	output["ManaOnBlock"] = modDB.Sum(mod.TypeBase, nil, "ManaOnBlock")
	output["WardOnBlock"] = modDB.Sum(mod.TypeBase, nil, "WardOnBlock")
	output["EnergyShieldOnBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnBlock")
	output["EnergyShieldOnSpellBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnSpellBlock")
	output["EnergyShieldOnSuppress"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnSuppress")

	//pob2:block:src/Modules/CalcDefence.lua:1965-1977
	avoidChanceCap := miscConstant("AvoidChanceCap")
	for _, damageType := range dmgTypeList {
		output["Avoid"+damageType+"DamageChance"] = math.Min(modDB.Sum(mod.TypeBase, nil, "Avoid"+damageType+"DamageChance"), avoidChanceCap)
	}
	output["AvoidProjectilesChance"] = math.Min(modDB.Sum(mod.TypeBase, nil, "AvoidProjectilesChance"), avoidChanceCap)
	output["AvoidAllDamageFromHitsChance"] = math.Min(modDB.Sum(mod.TypeBase, nil, "AvoidAllDamageFromHitsChance"), avoidChanceCap)
	avoidOrImmune := func(immune bool, names ...string) float64 {
		if immune {
			return 100
		}
		return math.Min(modDB.Sum(mod.TypeBase, nil, names...), 100)
	}
	output["BlindAvoidChance"] = avoidOrImmune(modDB.Flag(nil, "BlindImmune"), "AvoidBlind")
	output["ImpaleAvoidChance"] = avoidOrImmune(modDB.Flag(nil, "ImpaleImmune"), "AvoidImpale")

	//pob2:block:src/Modules/CalcDefence.lua:1984-1996
	if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
		spellSuppressionToAilmentPercent := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionAppliesToAilmentAvoidancePercent") / 100
		modDB.AddMod(newMod("AvoidElementalAilments", mod.TypeBase, math.Floor(spellSuppressionToAilmentPercent*spellSuppressionChance), "Ancestral Vision"))
	}
	if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
		if base := modDB.Sum(mod.TypeBase, nil, "AvoidShock"); base != 0 {
			modDB.AddMod(newMod("AvoidShockAppliesToElementalAilments", mod.TypeBase, base, "Stormshroud"))
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:1999-2017
	output["AilmentThreshold"] = calcLibVal(modDB, "AilmentThreshold", nil)
	for _, ailment := range dataNonElementalAilmentTypeList {
		output[ailment+"AvoidChance"] = math.Floor(avoidOrImmune(modDB.Flag(nil, ailment+"Immune"), "Avoid"+ailment, "AvoidAilments"))
	}
	for _, ailment := range dataElementalAilmentTypeList {
		shockAvoidAppliesToAll := modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") && ailment != "Shock"
		if modDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") {
			output[ailment+"AvoidChance"] = 100
			continue
		}
		avoid := modDB.Sum(mod.TypeBase, nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments")
		if shockAvoidAppliesToAll {
			avoid += modDB.Sum(mod.TypeBase, nil, "AvoidShock")
		}
		output[ailment+"AvoidChance"] = math.Floor(math.Min(avoid, 100))
	}
	output["CurseAvoidChance"] = avoidOrImmune(modDB.Flag(nil, "CurseImmune"), "AvoidCurse")
	output["SilenceAvoidChance"] = output["CurseAvoidChance"]
	if modDB.Flag(nil, "SilenceImmune") {
		output["SilenceAvoidChance"] = 100
	}
	output["CritExtraDamageReduction"] = math.Min(modDB.Sum(mod.TypeBase, nil, "ReduceCritExtraDamage"), 100)
	output["LightRadiusMod"] = calcLibMod(modDB, nil, "LightRadius")
	output["CurseEffectOnSelf"] = math.Max(modDB.More(nil, "CurseEffectOnSelf")*(100+modDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")), 0)
	output["ExposureEffectOnSelf"] = modDB.More(nil, "ExposureEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "ExposureEffectOnSelf"))
	output["WitherEffectOnSelf"] = modDB.More(nil, "WitherEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "WitherEffectOnSelf"))

	//pob2:block:src/Modules/CalcDefence.lua:2020-2050
	output["DebuffExpirationRate"] = modDB.Sum(mod.TypeBase, nil, "SelfDebuffExpirationRate")
	output["DebuffExpirationModifier"] = 10000 / (100 + output["DebuffExpirationRate"])
	output["SelfBlindDuration"] = modDB.More(nil, "SelfBlindDuration") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfBlindDuration")) * output["DebuffExpirationModifier"] / 100

	if modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") {
		inc := modDB.Sum(mod.TypeIncrease, nil, "SelfIgniteDuration")
		more := modDB.More(nil, "SelfIgniteDuration")
		if inc != 0 {
			modDB.AddMod(newMod("SelfIgniteDurationToElementalAilments", mod.TypeIncrease, inc, "Firesong"))
		}
		if more != 1 {
			modDB.AddMod(newMod("SelfIgniteDurationToElementalAilments", mod.TypeMore, more, "Firesong"))
		}
	}

	for _, ailment := range dataNonElementalAilmentTypeList {
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration")
		inc := (100 + modDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Duration", "SelfAilmentDuration")) * 100
		output["Self"+ailment+"Duration"] = inc * more / (100 + output["DebuffExpirationRate"] + modDB.Sum(mod.TypeBase, nil, "Self"+ailment+"DebuffExpirationRate"))
	}
	for _, ailment := range dataElementalAilmentTypeList {
		igniteAppliesToAll := modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") && ailment != "Ignite"
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
		inc := 100 + modDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
		if igniteAppliesToAll {
			more *= modDB.More(nil, "SelfIgniteDuration")
			inc += modDB.Sum(mod.TypeIncrease, nil, "SelfIgniteDuration")
		}
		inc *= 100
		output["Self"+ailment+"Duration"] = more * inc / (100 + output["DebuffExpirationRate"] + modDB.Sum(mod.TypeBase, nil, "Self"+ailment+"DebuffExpirationRate"))
	}
	for _, ailment := range dataAilmentTypeList {
		magnitude := calcLibMod(enemyDB, nil, "Enemy"+ailment+"Magnitude", "AilmentMagnitude")
		if modDB.Flag(nil, "Condition:"+ailment+"edSelf") {
			magnitude = calcLibMod(modDB, nil, "Enemy"+ailment+"Magnitude", "AilmentMagnitude")
		}
		output["Self"+ailment+"Effect"] = calcLibMod(modDB, nil, "Self"+ailment+"Effect") * magnitude * 100
	}
}
