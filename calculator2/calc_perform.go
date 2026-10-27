package calculator2

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

type legacyEffect struct {
	stat    string
	modType mod.Type
	value   float64
}

// Mageblood Legacies
//
//pob2:src/Modules/CalcPerform.lua:66-143
var legacies = map[string][]legacyEffect{
	"LegacyOfAmethyst":    {{"ChaosResist", mod.TypeBase, 45}},
	"LegacyOfBasalt":      {{"Armour", mod.TypeIncrease, 150}},
	"LegacyOfBismuth":     {{"ElementalResist", mod.TypeBase, 45}},
	"LegacyOfDiamond":     {{"CritChance", mod.TypeIncrease, 75}},
	"LegacyOfGold":        {{"LootRarity", mod.TypeIncrease, 45}},
	"LegacyOfGranite":     {{"Armour", mod.TypeBase, 2000}},
	"LegacyOfJade":        {{"Evasion", mod.TypeBase, 2000}},
	"LegacyOfQuicksilver": {{"MovementSpeed", mod.TypeIncrease, 30}},
	"LegacyOfRuby":        {{"FireResist", mod.TypeBase, 60}, {"FireResistMax", mod.TypeBase, 5}},
	"LegacyOfSapphire":    {{"ColdResist", mod.TypeBase, 60}, {"ColdResistMax", mod.TypeBase, 5}},
	"LegacyOfSilver":      {{"Speed", mod.TypeIncrease, 30}, {"WarcrySpeed", mod.TypeIncrease, 30}, {"TotemPlacementSpeed", mod.TypeIncrease, 30}},
	"LegacyOfStibnite":    {{"Evasion", mod.TypeIncrease, 150}},
	"LegacyOfSulphur":     {{"Damage", mod.TypeIncrease, 60}},
	"LegacyOfTopaz":       {{"LightningResist", mod.TypeBase, 60}, {"LightningResistMax", mod.TypeBase, 5}},
}

type weaponType struct {
	oneHand, melee bool
	flag, label    string
}

//pob2:src/Modules/Data.lua:605-625
var weaponTypeInfo = map[string]weaponType{
	"None":                     {oneHand: true, melee: true, flag: "Unarmed"},
	"Bow":                      {oneHand: false, melee: false, flag: "Bow"},
	"Crossbow":                 {oneHand: false, melee: false, flag: "Crossbow"},
	"Claw":                     {oneHand: true, melee: true, flag: "Claw"},
	"Dagger":                   {oneHand: true, melee: true, flag: "Dagger"},
	"Spear":                    {oneHand: true, melee: true, flag: "Spear"},
	"Flail":                    {oneHand: true, melee: true, flag: "Flail"},
	"Staff":                    {oneHand: false, melee: true, flag: "Staff", label: "Quarterstaff"},
	"Warstaff":                 {oneHand: false, melee: true, flag: "Warstaff"},
	"Wand":                     {oneHand: true, melee: false, flag: "Wand"},
	"One Hand Axe":             {oneHand: true, melee: true, flag: "Axe"},
	"One Hand Mace":            {oneHand: true, melee: true, flag: "Mace"},
	"One Hand Sword":           {oneHand: true, melee: true, flag: "Sword"},
	"Thrusting One Hand Sword": {oneHand: true, melee: true, flag: "Sword", label: "One Hand Sword"},
	"Fishing Rod":              {oneHand: false, melee: true, flag: "Fishing"},
	"Two Hand Axe":             {oneHand: false, melee: true, flag: "Axe"},
	"Two Hand Mace":            {oneHand: false, melee: true, flag: "Mace"},
	"Two Hand Sword":           {oneHand: false, melee: true, flag: "Sword"},
	"Talisman":                 {oneHand: false, melee: true, flag: "Talisman"},
}

//pob2:src/Modules/CalcPerform.lua:147-157
func mergeKeystones(env *Environment) {
	for _, modObj := range env.ModDB.Tabulate(mod.TypeList, nil, "Keystone") {
		name, _ := modObj.Mod.Value().ValueList.(string)
		node := env.Spec.Tree.KeystoneMap[name]
		if env.KeystonesAdded[name] || node == nil {
			continue
		}
		env.KeystonesAdded[name] = true
		src := string(modObj.Mod.GetSource())
		fromTree := src != "" && !strings.Contains(strings.ToLower(src), "tree")
		for _, m := range processStats(env.modSource, node.ID, node.Type, node.Dn, node.Sd).Mods() {
			if fromTree {
				m = mod.SetSource(m, modObj.Mod.GetSource())
			}
			env.ModDB.AddMod(m)
		}
	}
}

// The second pass runs because one attribute can depend on another being higher.
//
//pob2:src/Modules/CalcPerform.lua:233-260
func calculateAttributes(modDB *moddb.ModDB, output map[string]float64, condList map[string]bool) {
	for range 2 {
		for _, stat := range []string{"Str", "Dex", "Int"} {
			output[stat] = math.Max(round(calcLibVal(modDB, stat, nil)), 0)
		}

		stats := []float64{output["Str"], output["Dex"], output["Int"]}
		sort.Float64s(stats)
		output["LowestAttribute"] = stats[0]
		condList["TwoHighestAttributesEqual"] = stats[1] == stats[2]

		condList["DexHigherThanInt"] = output["Dex"] > output["Int"]
		condList["StrHigherThanInt"] = output["Str"] > output["Int"]
		condList["IntHigherThanDex"] = output["Int"] > output["Dex"]
		condList["StrHigherThanDex"] = output["Str"] > output["Dex"]
		condList["IntHigherThanStr"] = output["Int"] > output["Str"]
		condList["DexHigherThanStr"] = output["Dex"] > output["Str"]

		condList["StrHighestAttribute"] = output["Str"] >= output["Dex"] && output["Str"] >= output["Int"]
		condList["IntHighestAttribute"] = output["Int"] >= output["Str"] && output["Int"] >= output["Dex"]
		condList["DexHighestAttribute"] = output["Dex"] >= output["Str"] && output["Dex"] >= output["Int"]
		condList["IntSingleHighestAttribute"] = output["Int"] > output["Str"] && output["Int"] > output["Dex"]
		condList["DexSingleHighestAttribute"] = output["Dex"] > output["Str"] && output["Dex"] > output["Int"]
	}
}

//pob2:src/Modules/CalcPerform.lua:188-199
func hasExposureSource(modDB *moddb.ModDB, element string) bool {
	modName := element + "ExposureChance"
	return modDB.Sum(mod.TypeBase, nil, modName) > 0 || modDB.HasMod(mod.TypeFlag, nil, "InflictExposure")
}

//pob2:src/Modules/CalcPerform.lua:203-232
func calcBuffRadius(actor *Actor, key string) {
	radiusKey, areaKey, modKey := key+"Radius", key+"Area", key+"Mod"
	modDB := actor.ModDB
	baseRadius := modDB.Sum(mod.TypeBase, nil, radiusKey)
	incRadius := modDB.Sum(mod.TypeIncrease, nil, radiusKey)
	moreRadius := modDB.More(nil, radiusKey)
	scaledRadius := baseRadius * (1 + incRadius/100) * moreRadius
	if o := modDB.Override(nil, radiusKey); o != nil {
		scaledRadius = o.Float()
	}
	baseArea := math.Pi * (scaledRadius * scaledRadius)
	incArea := modDB.Sum(mod.TypeIncrease, nil, areaKey)
	moreArea := modDB.More(nil, areaKey)
	scaledArea := baseArea * (1 + incArea/100) * moreArea
	if o := modDB.Override(nil, areaKey); o != nil {
		scaledArea = o.Float()
	}
	finalRadius := math.Floor(math.Sqrt(scaledArea / math.Pi))
	actor.Output[radiusKey] = finalRadius
	actor.Output[radiusKey+"Metres"] = finalRadius / 10
	if scaledRadius/baseRadius != 1 {
		actor.Output[modKey] = scaledRadius / baseRadius
	}
}

// The skill-flag conditions need a main skill, so the mode_combat block runs from its shrine buffs on.
// countsAsAll1H and addWeaponBaseStats are not ported.
//
//pob2:src/Modules/CalcPerform.lua:264-528
func doActorAttribsConditions(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output
	condList := modDB.Conditions

	weapon2, hasWeapon2 := actor.ItemList["Weapon 2"]
	switch {
	case hasWeapon2 && weapon2.Type == "Shield":
		condList["UsingShield"] = true
	case hasWeapon2 && weapon2.Type == "Focus":
		condList["UsingFocus"] = true
	case !hasWeapon2:
		condList["OffHandIsEmpty"] = true
	}
	if actor.WeaponData1Type == "None" {
		condList["Unarmed"] = true
		if _, gloves := actor.ItemList["Gloves"]; !hasWeapon2 && !gloves {
			condList["Unencumbered"] = true
		}
	} else {
		info := weaponTypeInfo[actor.WeaponData1Type]
		condList["Using"+info.flag] = true
		if info.melee {
			condList["UsingMeleeWeapon"] = true
		}
		if info.oneHand {
			condList["UsingOneHandedWeapon"] = true
		} else {
			condList["UsingTwoHandedWeapon"] = true
		}
		weapon1 := actor.ItemList["Weapon 1"]
		if (weapon1 != nil && weapon1.Base != nil && weapon1.Base.Weapon != nil) || (weapon2 != nil && weapon2.Base != nil && weapon2.Base.Weapon != nil) {
			condList["UsingMartialWeapon"] = true
		}
		addWeaponBaseStats(actor)
	}
	for _, slotName := range []string{"Helmet", "Body Armour", "Gloves", "Boots"} {
		if _, ok := actor.ItemList[slotName]; ok {
			condList["Using"+slotName] = true
		}
	}
	if actor.WeaponData2Type != "" && actor.WeaponData2Type != "None" {
		info := weaponTypeInfo[actor.WeaponData2Type]
		condList["Using"+info.flag] = true
		if info.melee {
			condList["UsingMeleeWeapon"] = true
		}
		if info.oneHand {
			condList["UsingOneHandedWeapon"] = true
		} else {
			condList["UsingTwoHandedWeapon"] = true
		}
	}
	if actor.WeaponData1Type != "" && actor.WeaponData2Type != "" {
		condList["DualWielding"] = true
		if actor.WeaponData1Type == "Claw" && actor.WeaponData2Type == "Claw" {
			condList["DualWieldingClaws"] = true
		}
		if actor.WeaponData1Type == "Dagger" && actor.WeaponData2Type == "Dagger" {
			condList["DualWieldingDaggers"] = true
		}
		info1, info2 := weaponTypeInfo[actor.WeaponData1Type], weaponTypeInfo[actor.WeaponData2Type]
		label1, label2 := info1.label, info2.label
		if label1 == "" {
			label1 = actor.WeaponData1Type
		}
		if label2 == "" {
			label2 = actor.WeaponData2Type
		}
		if label1 != label2 && info1.oneHand && info2.oneHand {
			condList["WieldingDifferentWeaponTypes"] = true
		}
	}
	if env.ModeCombat {
		//pob2:block:src/Modules/CalcPerform.lua:449-470
		shrineEffectMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "ShrineBuffEffect")/100
		if modDB.Flag(nil, "LesserMassiveShrine") {
			modDB.AddMod(newMod("Life", mod.TypeIncrease, math.Floor(20*shrineEffectMod), "Lesser Massive Shrine"))
			modDB.AddMod(newMod("AreaOfEffect", mod.TypeIncrease, math.Floor(20*shrineEffectMod), "Lesser Massive Shrine"))
		}
		if modDB.Flag(nil, "LesserBrutalShrine") {
			modDB.AddMod(newMod("Damage", mod.TypeIncrease, math.Floor(20*shrineEffectMod), "Lesser Brutal Shrine"))
			modDB.AddMod(newMod("EnemyStunDuration", mod.TypeIncrease, math.Floor(20*shrineEffectMod), "Lesser Brutal Shrine"))
			modDB.AddMod(newMod("EnemyKnockbackChance", mod.TypeIncrease, 100, "Lesser Brutal Shrine"))
		}
		if modDB.Flag(nil, "DiamondShrine") {
			modDB.AddMod(newMod("CritChance", mod.TypeOverride, 100, "Diamond Shrine"))
		}
		if modDB.Flag(nil, "MassiveShrine") {
			modDB.AddMod(newMod("Life", mod.TypeIncrease, math.Floor(40*shrineEffectMod), "Massive Shrine"))
			modDB.AddMod(newMod("AreaOfEffect", mod.TypeIncrease, math.Floor(40*shrineEffectMod), "Massive Shrine"))
		}
		if modDB.Flag(nil, "Gigantic") {
			modDB.AddMod(newMod("Life", mod.TypeMore, 20, "Gigantic"))
			modDB.AddMod(newMod("Damage", mod.TypeMore, 20, "Gigantic"))
		}
	}
	if env.ModeEffective {
		for _, element := range []string{"Fire", "Cold", "Lightning"} {
			if hasExposureSource(modDB, element) {
				condList["CanApply"+element+"Exposure"] = true
				modDB.AddMod(newFlagMod("Condition:CanApply"+element+"Exposure", "Exposure"))
			}
		}
	}

	calculateAttributes(modDB, output, condList)
	output["TotalAttr"] = output["Str"] + output["Dex"] + output["Int"]

	output["Devotion"] = modDB.Sum(mod.TypeBase, nil, "Devotion")
	output["Tribute"] = modDB.Sum(mod.TypeBase, nil, "Tribute")

	if !modDB.Flag(nil, "NoAttributeBonuses") {
		inherentAttributeMultiplier := 1.0
		if modDB.Flag(nil, "DoubledInherentAttributeBonuses") {
			inherentAttributeMultiplier = 2
		}
		if !modDB.Flag(nil, "NoStrengthAttributeBonuses") {
			if !modDB.Flag(nil, "NoStrBonusToLife") {
				if modDB.Flag(nil, "HalvesLifeFromStrength") {
					modDB.AddMod(newMod("Life", mod.TypeBase, output["Str"]*1*inherentAttributeMultiplier, "Strength"))
				} else {
					modDB.AddMod(newMod("Life", mod.TypeBase, output["Str"]*2*inherentAttributeMultiplier, "Strength"))
				}
			}
		}
		if !modDB.Flag(nil, "NoDexterityAttributeBonuses") {
			if !modDB.Flag(nil, "NoDexBonusToAccuracy") {
				accPerDex := miscConstant("AccuracyPerDexBase")
				if o := modDB.Override(nil, "DexAccBonusOverride"); o != nil {
					accPerDex = o.Float()
				}
				modDB.AddMod(newMod("Accuracy", mod.TypeBase, output["Dex"]*accPerDex*inherentAttributeMultiplier, "Dexterity"))
			}
		}
		if !modDB.Flag(nil, "NoIntelligenceAttributeBonuses") {
			if !modDB.Flag(nil, "NoIntBonusToMana") {
				modDB.AddMod(newMod("Mana", mod.TypeBase, output["Int"]*2*inherentAttributeMultiplier, "Intelligence"))
			}
		}
	}
	calcBuffRadius(actor, "Presence")
	calcBuffRadius(actor, "Surrounded")
	enemyDistance := math.Max(modDB.Sum(mod.TypeBase, nil, "Multiplier:enemyDistance"), 0)
	surroundedMinimum := math.Max(modDB.Sum(mod.TypeBase, nil, "SurroundedMinimum"), 1)
	condList["EnemyInPresence"] = output["PresenceRadius"] >= enemyDistance
	condList["Surrounded"] = surroundedMinimum == 1 && output["SurroundedRadius"] >= enemyDistance
}

//pob2:block:src/Modules/CalcPerform.lua:1490-1537
func addSlotBonusEffectsPerform(env *Environment) {
	modDB := env.ModDB
	for slot, item := range env.Player.ItemList {
		slotEffectMod := modDB.Sum(mod.TypeIncrease, nil, "EffectOfBonusesFrom"+slot) / 100
		if slotEffectMod <= 0 || slot == "Amulet" {
			continue
		}
		if strings.Contains(item.Name, "Kalandra's Touch") {
			if slot == "Ring 2" {
				item = env.Player.ItemList["Ring 1"]
			} else if slot == "Ring 1" {
				item = env.Player.ItemList["Ring 2"]
			}
		}
		if item == nil {
			continue
		}
		groupedMods := map[string]mod.Mod{}
		var groupedModOrder []mod.Mod
		slotEffectSource := mod.Source("Many Sources:^x88FFFF" + luaNum(slotEffectMod*100) + "% " + slot + " Bonus Effect")
		modList := item.ModList
		if modList == nil && len(item.SlotModList) > 1 {
			modList = item.SlotModList[1]
		}
		for _, m := range modList {
			if hasSocketedInTag(m) {
				continue
			}
			if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat && (m.Type() == mod.TypeBase || m.Type() == mod.TypeIncrease) {
				key := formatModParams(m)
				if modCopy, ok := groupedMods[key]; ok {
					modCopy.Value().SetFloat(modCopy.Value().Float() + v.Float())
				} else {
					modCopy = m.Clone().Source(slotEffectSource)
					groupedMods[key] = modCopy
					groupedModOrder = append(groupedModOrder, modCopy)
				}
			} else {
				modDB.ScaleAddMod(m.Clone().Source(slotEffectSource), slotEffectMod, false)
			}
		}
		for _, m := range groupedModOrder {
			modDB.ScaleAddMod(m, slotEffectMod, false)
		}
	}
}

// Tags are compared by mod.FormatValue rather than modLib.formatTags; the key only groups.
//
//pob2:src/Modules/ModTools.lua:265-267
func formatModParams(m mod.Mod) string {
	return m.Name() + "|" + string(m.Type()) + "|" + strconv.Itoa(int(m.Flags())) + "|" + strconv.Itoa(int(m.KeywordFlags())) + "|" + mod.FormatValue(m.Tags())
}

//pob2:block:src/Modules/CalcPerform.lua:1551-1578
func addMagebloodLegacies(env *Environment) {
	modDB := env.ModDB
	if !modDB.Flag(nil, "MagebloodEquipped") {
		return
	}
	legacyCountByName := map[string]float64{}
	for name := range legacies {
		if stacks := modDB.Sum(mod.TypeBase, nil, name); stacks > 0 {
			legacyCountByName[name] = stacks
		}
	}
	if len(legacyCountByName) == 0 {
		return
	}
	totalDuplicates := 0.0
	for _, stacks := range legacyCountByName {
		if stacks > 1 {
			totalDuplicates += stacks - 1
		}
	}
	effectPerDupe := modDB.Sum(mod.TypeIncrease, nil, "MagesLegacyEffect")
	globalEffect := 1 + totalDuplicates*(effectPerDupe/100)
	for name := range legacyCountByName {
		for _, entry := range legacies[name] {
			modDB.AddMod(newMod(entry.stat, entry.modType, math.Floor(globalEffect*entry.value), "Mageblood"))
		}
	}
}

//pob2:src/Modules/CalcPerform.lua:565-579
func applyEnemyModifiers(actor *Actor, clearCache bool) {
	if clearCache || actor.appliedEnemyModifiers == nil {
		actor.appliedEnemyModifiers = map[mod.Mod]bool{}
	}
	cache := actor.appliedEnemyModifiers
	enemyDB := actor.Enemy.ModDB
	for _, value := range actor.ModDB.Tabulate("", nil, "EnemyModifier") {
		v := value.Mod.Value()
		if v == nil || v.Type() != mod.ModValueMultiTypeList {
			continue
		}
		payload, ok := v.List().(mod.EnemyModifier)
		if !ok || payload.Mod == nil || cache[payload.Mod] {
			continue
		}
		source := payload.Mod.GetSource()
		if source == "" {
			source = value.Mod.GetSource()
		}
		enemyDB.AddMod(mod.SetSource(payload.Mod.Clone(), source))
		cache[payload.Mod] = true
	}
}

// Cruelty and Fanaticism read the main skill, so those two are absent; with no main skill, Elusive reads
// SupportedByNightblade as unset.
//
//pob2:src/Modules/CalcPerform.lua:584-857
func doActorMisc(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output
	condList := modDB.Conditions

	if env.ModeCombat {
		if modDB.Sum(mod.TypeBase, nil, "MinimumRage") > modDB.Multipliers["Rage"] {
			modDB.Multipliers["Rage"] = modDB.Sum(mod.TypeBase, nil, "MinimumRage")
		}
		alliedFortify := 0.0
		if modDB.Flag(nil, "YourFortifyEqualToParent") && actor.Parent != nil {
			alliedFortify = actor.Parent.Output["FortificationStacks"]
		}
		if modDB.Sum(mod.TypeBase, nil, "MinimumFortification") > 0 || alliedFortify > 0 {
			condList["Fortified"] = true
		}
		if modDB.Flag(nil, "Fortified") || modDB.Sum(mod.TypeBase, nil, "Multiplier:Fortification") > 0 {
			maxStacks := modDB.Sum(mod.TypeBase, nil, "MaximumFortification")
			if o := modDB.Override(nil, "MaximumFortification"); o != nil {
				maxStacks = o.Float()
			}
			minStacks := math.Min(modDB.Sum(mod.TypeBase, nil, "MinimumFortification"), maxStacks)
			var stacks float64
			if o := modDB.Override(nil, "FortificationStacks"); o != nil {
				stacks = o.Float()
			} else if alliedFortify > 0 {
				stacks = alliedFortify
			} else if minStacks > 0 {
				stacks = minStacks
			} else {
				stacks = maxStacks
			}
			output["FortificationStacks"] = stacks
			if !modDB.Flag(nil, "Condition:NoFortificationMitigation") {
				effectScale := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
				effect := math.Floor(effectScale * stacks)
				modDB.AddMod(newMod("DamageTakenWhenHit", mod.TypeMore, -effect, "Fortification"))
			}
			if stacks >= maxStacks {
				modDB.AddMod(newFlagMod("Condition:HaveMaximumFortification", ""))
			}
			modDB.Multipliers["BuffOnSelf"]++
		}
		if modDB.Flag(nil, "Onslaught") {
			var effect float64
			onslaughtFromFlask := false
			flaskEffectInc := -100.0
			for item := range env.Flasks {
				if strings.Contains(item.BaseName, "Silver Flask") {
					onslaughtFromFlask = true
					curFlaskEffectInc := item.FlaskData["effectInc"] + modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "FlaskEffect")
					if item.Rarity == "MAGIC" {
						curFlaskEffectInc += modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "MagicUtilityFlaskEffect")
					}
					if flaskEffectInc < curFlaskEffectInc/100 {
						flaskEffectInc = curFlaskEffectInc / 100
					}
				}
			}
			onslaughtEffectInc := modDB.Sum(mod.TypeIncrease, nil, "OnslaughtEffect", "BuffEffectOnSelf") / 100
			if onslaughtFromFlask {
				effect = math.Floor(10 * (1 + flaskEffectInc + onslaughtEffectInc))
			} else {
				effect = math.Floor(10 * (1 + onslaughtEffectInc))
			}
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, 2*effect, "Onslaught").Flag(mod.MFlagAttack))
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, 2*effect, "Onslaught").Flag(mod.MFlagCast))
			modDB.AddMod(newMod("WarcrySpeed", mod.TypeIncrease, 2*effect, "Onslaught"))
			modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, effect, "Onslaught"))
		}
		if modDB.Flag(nil, "UnholyMight") {
			effect := 1 + (modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf") / 100)
			modDB.AddMod(newMod("Multiplier:UnholyMightMagnitude", mod.TypeBase, 100, "Unholy Might"))
			modDB.AddMod(newMod("DamageGainAsChaos", mod.TypeBase, 0.3*effect, "Unholy Might", mod.Multiplier("UnholyMightMagnitude")))
		}
		if modDB.Flag(nil, "ChaoticMight") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("PhysicalDamageGainAsChaos", mod.TypeBase, effect, "Chaotic Might"))
		}
		if modDB.Flag(nil, "Adrenaline") {
			effectMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(newMod("Damage", mod.TypeIncrease, math.Floor(100*effectMod), "Adrenaline"))
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, math.Floor(25*effectMod), "Adrenaline").Flag(mod.MFlagAttack))
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, math.Floor(25*effectMod), "Adrenaline").Flag(mod.MFlagCast))
			modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, math.Floor(25*effectMod), "Adrenaline"))
			modDB.AddMod(newMod("PhysicalDamageReduction", mod.TypeBase, math.Floor(10*effectMod), "Adrenaline"))
		}
		if modDB.Flag(nil, "Convergence") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(newMod("ElementalDamage", mod.TypeMore, effect, "Convergence"))
		}
		if modDB.Flag(nil, "HerEmbrace") {
			condList["HerEmbrace"] = true
			modDB.AddMod(newMod("AvoidStun", mod.TypeBase, 100, "Her Embrace"))
			modDB.AddMod(newMod("PhysicalDamageGainAsFire", mod.TypeBase, 123, "Her Embrace").Flag(mod.MFlagSword))
			modDB.AddMod(newMod("AvoidFreeze", mod.TypeBase, 100, "Her Embrace"))
			modDB.AddMod(newMod("AvoidChill", mod.TypeBase, 100, "Her Embrace"))
			modDB.AddMod(newMod("AvoidIgnite", mod.TypeBase, 100, "Her Embrace"))
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, 20, "Her Embrace").Flag(mod.MFlagAttack))
			modDB.AddMod(newMod("Speed", mod.TypeIncrease, 20, "Her Embrace").Flag(mod.MFlagCast))
			modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, 20, "Her Embrace"))
		}
		if modDB.Flag(nil, "Condition:PhantasmalMight") {
			activePhantasmLimit, ok := output["ActivePhantasmLimit"]
			if !ok {
				activePhantasmLimit = 1
			}
			modDB.Multipliers["BuffOnSelf"] += activePhantasmLimit - 1
		}
		if modDB.Flag(nil, "Elusive") {
			skillSource := mod.Source("Skill")
			maxSkillInc := modDB.Max(&moddb.ListCfg{Source: &skillSource}, "ElusiveEffect")
			inc := modDB.Sum(mod.TypeIncrease, nil, "ElusiveEffect", "BuffEffectOnSelf")
			inc += maxSkillInc
			elusiveEffectMod := (1 + inc/100) * modDB.More(nil, "ElusiveEffect", "BuffEffectOnSelf") * 100
			minThreshold := 0.0
			if o := modDB.Override(nil, "ElusiveEffectMinThreshold"); o != nil {
				minThreshold = o.Float()
			}
			output["ElusiveEffectMod"] = (elusiveEffectMod + minThreshold) / 2
			modDB.AddMod(newMod("ElusiveEffect", mod.TypeIncrease, maxSkillInc, "Max Skill Effect"))
			if o := modDB.Override(nil, "ElusiveEffect"); o != nil {
				output["ElusiveEffectMod"] = math.Min(o.Float(), elusiveEffectMod)
			}
			effect := output["ElusiveEffectMod"] / 100
			condList["Elusive"] = true
			modDB.AddMod(newMod("AvoidAllDamageFromHitsChance", mod.TypeBase, math.Floor(15*effect), "Elusive"))
			modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, math.Floor(30*effect), "Elusive"))
		}
		if effect := modDB.Max(nil, "WitherEffectStack"); effect > 0 {
			modDB.AddMod(newFlagMod("Condition:CanWither", "Config"))
			enemyDB.AddMod(newMod("ChaosDamageTaken", mod.TypeIncrease, effect, "Withered", mod.Multiplier("WitheredStack").Limit(10)))
		}
		if modDB.Flag(nil, "Condition:CanInflictIncision") {
			effect := 10 * (1 + modDB.Sum(mod.TypeIncrease, nil, "IncisionEffect")/100)
			enemyDB.AddMod(newMod("SelfBleedChance", mod.TypeBase, effect, "Incision", mod.Multiplier("IncisionStack").Limit(10)))
		}
		if modDB.Flag(nil, "Condition:CanBeWithered") {
			effect := 5 * (100 + modDB.Sum(mod.TypeIncrease, nil, "WitherEffectOnSelf")) / 100 * modDB.More(nil, "WitherEffectOnSelf")
			modDB.AddMod(newMod("ChaosDamageTaken", mod.TypeIncrease, effect, "Withered", mod.Multiplier("WitheredStack").Limit(10)))
		}
		if enemyDB.Flag(nil, "Condition:ArmourFullyBroken") {
			effect := 20 * (1 + modDB.Sum(mod.TypeIncrease, nil, "FullyBrokenArmourEffect")/100)
			if modDB.Flag(nil, "FullyBrokenArmourIncreasesAllDamageFromHits") {
				enemyDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, effect, "Fully Broken Armour").Flag(mod.MFlagHit))
			} else {
				enemyDB.AddMod(newMod("PhysicalDamageTaken", mod.TypeIncrease, effect, "Fully Broken Armour").Flag(mod.MFlagHit))
				if modDB.Flag(nil, "ArmourBreakColdDamageTaken") {
					enemyDB.AddMod(newMod("ColdDamageTaken", mod.TypeIncrease, effect, "Fully Broken Armour").Flag(mod.MFlagHit))
				}
				if modDB.Flag(nil, "ArmourBreakFireDamageTaken") {
					enemyDB.AddMod(newMod("FireDamageTaken", mod.TypeIncrease, effect, "Fully Broken Armour").Flag(mod.MFlagHit))
				}
				if modDB.Flag(nil, "ArmourBreakLightningDamageTaken") {
					enemyDB.AddMod(newMod("LightningDamageTaken", mod.TypeIncrease, effect, "Fully Broken Armour").Flag(mod.MFlagHit))
				}
			}
		}
		if modDB.Flag(nil, "Blind") && !modDB.Flag(nil, "CannotBeBlinded") {
			if !modDB.Flag(nil, "IgnoreBlindHitChance") {
				effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "BlindEffect", "BuffEffectOnSelf")/100
				if o := modDB.Override(nil, "BlindEffect"); o != nil {
					effect = math.Min(o.Float()/100, effect)
				}
				baseBlind := -20.0
				if modDB.Flag(nil, "BlindEffectReversed") {
					baseBlind = 20
				}
				modDB.AddMod(newMod("Accuracy", mod.TypeMore, math.Floor(baseBlind*effect), "Blind"))
				modDB.AddMod(newMod("Evasion", mod.TypeMore, math.Floor(baseBlind*effect), "Blind"))
			}
		}
		if modDB.Flag(nil, "Chill") {
			chillValue := dataNonDamagingAilmentDefault["Chill"]
			if o := modDB.Override(nil, "ChillVal"); o != nil {
				chillValue = o.Float()
			}
			chillSelf := 0.0
			if modDB.Flag(nil, "Condition:ChilledSelf") {
				chillSelf = modDB.Sum(mod.TypeIncrease, nil, "EnemyChillMagnitude") / 100
			}
			totalChillSelfEffect := calcLibMod(modDB, nil, "SelfChillEffect") + chillSelf
			chillMax := gameConstant("ChillMaxEffect")
			if o := modDB.Override(nil, "ChillMax"); o != nil {
				chillMax = o.Float()
			}
			effect := math.Min(math.Max(math.Floor(chillValue*totalChillSelfEffect), 0), chillMax)
			sign := -1.0
			if modDB.Flag(nil, "SelfChillEffectIsReversed") {
				sign = 1
			}
			modDB.AddMod(newMod("ActionSpeed", mod.TypeIncrease, effect*sign, "Chill"))
		}
		if modDB.Flag(nil, "Freeze") {
			effect := math.Max(math.Floor(70*calcLibMod(modDB, nil, "SelfChillEffect")), 0)
			modDB.AddMod(newMod("ActionSpeed", mod.TypeIncrease, -effect, "Freeze"))
		}
		if modDB.Flag(nil, "Unravelling") {
			modDB.AddMod(newFlagMod("ChaosCanFreeze", "", mod.Condition("ColdUnravel")))
			modDB.AddMod(newFlagMod("ChaosCanIgnite", "", mod.Condition("FireUnravel")))
			modDB.AddMod(newFlagMod("ChaosCanShock", "", mod.Condition("LightningUnravel")))
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullLife") && !modDB.Flag(nil, "GhostReaver") {
			condList["Leeching"] = true
			condList["LeechingLife"] = true
		}
		if modDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
			condList["Leeching"] = true
			condList["LeechingEnergyShield"] = true
		}
		if modDB.Flag(nil, "Condition:CanGainRage") || modDB.Sum(mod.TypeBase, nil, "RageRegen") > 0 {
			maxStacks := modDB.Sum(mod.TypeBase, nil, "MaximumRage") * calcLibMod(modDB, nil, "MaximumRage")
			minStacks := math.Min(modDB.Sum(mod.TypeBase, nil, "MinimumRage"), maxStacks)
			rageConfig := modDB.Sum(mod.TypeBase, nil, "Multiplier:RageStack")
			stacks := math.Max(math.Min(rageConfig, maxStacks), math.Max(minStacks, 0))
			if o := modDB.Override(nil, "RageEffect"); o != nil {
				output["RageEffect"] = o.Float()
			} else {
				output["RageEffect"] = math.Floor(stacks * calcLibMod(modDB, nil, "RageEffect"))
			}
			modDB.AddMod(newMod("Multiplier:RageEffect", mod.TypeBase, output["RageEffect"], "Base"))
			output["Rage"] = stacks
			output["MaximumRage"] = maxStacks
			modDB.AddMod(newMod("Multiplier:Rage", mod.TypeBase, output["Rage"], "Base"))
			if modDB.Flag(nil, "Condition:RageSpellDamage") {
				modDB.AddMod(newMod("Damage", mod.TypeMore, output["RageEffect"], "Rage").Flag(mod.MFlagSpell))
			} else {
				modDB.AddMod(newMod("Damage", mod.TypeMore, output["RageEffect"], "Rage").Flag(mod.MFlagAttack))
			}
			if stacks == maxStacks {
				modDB.AddMod(newFlagMod("Condition:HaveMaximumRage", ""))
			}
			output["InherentRageLossDelay"] = 2 + modDB.Sum(mod.TypeBase, nil, "InherentRageLossDelay")
			output["InherentRageLoss"] = 0
			if !modDB.Flag(nil, "InherentRageLossIsPrevented") {
				output["InherentRageLoss"] = 10 * (1 + modDB.Sum(mod.TypeIncrease, nil, "InherentRageLoss")/100)
			}
		}
		if manaBurn := env.ConfigInput["multiplierManaBurnStacks"].Number; manaBurn > 0 {
			maxManaBurn := modDB.Sum(mod.TypeBase, nil, "MaxManaBurnStacks")
			if maxManaBurn == 0 {
				maxManaBurn = 9999
			}
			manaBurnStacks := math.Min(manaBurn, maxManaBurn)
			modDB.AddMod(newMod("Multiplier:ManaBurnStacks", mod.TypeBase, manaBurnStacks, "Config"))
			manaBurnStacks += modDB.Sum(mod.TypeBase, &moddb.ListCfg{Actor: "player"}, "EffectiveManaBurnStacks")
			if modDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") {
				modDB.AddMod(newMod("Multiplier:WeepingWoundsStacks", mod.TypeBase, manaBurnStacks, "Config"))
			} else {
				modDB.AddMod(newMod("Multiplier:EffectiveManaBurnStacks", mod.TypeBase, manaBurnStacks, "Config"))
			}
		}
		if effect := modDB.Sum(mod.TypeBase, nil, "CoveredInAshEffect"); effect > 0 {
			enemyDB.AddMod(newMod("FireDamageTaken", mod.TypeIncrease, math.Min(effect, 20), "Covered in Ash"))
		}
		if effect := modDB.Sum(mod.TypeBase, nil, "CoveredInFrostEffect"); effect > 0 {
			enemyDB.AddMod(newMod("ColdDamageTaken", mod.TypeIncrease, math.Min(effect, 20), "Covered in Frost"))
		}
		if modDB.Flag(nil, "HasMalediction") {
			modDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, 10, "Malediction"))
			modDB.AddMod(newMod("Damage", mod.TypeIncrease, -10, "Malediction"))
		}
		if modDB.Flag(nil, "HasMaddeningPresence") {
			modDB.AddMod(newMod("ActionSpeed", mod.TypeIncrease, -10, "Maddening Presence"))
			modDB.AddMod(newMod("Damage", mod.TypeIncrease, -10, "Maddening Presence"))
		}
		if modDB.Flag(nil, "HasShapersPresence") {
			modDB.AddMod(newMod("BuffExpireFaster", mod.TypeMore, -20, "Shapers Presence"))
		}
		if modDB.Flag(nil, "Condition:CanHaveSoulEater") {
			soulEaterMax := modDB.Sum(mod.TypeBase, nil, "SoulEaterMax")
			if o := modDB.Override(nil, "SoulEaterMax"); o != nil {
				soulEaterMax = o.Float()
			}
			modDB.AddMod(newMod("Multiplier:SoulEater", mod.TypeBase, 1, "Base", mod.Multiplier("SoulEaterStack").Limit(soulEaterMax)))
		}
	}

	//pob2:block:src/Modules/CalcPerform.lua:844-845
	applyEnemyModifiers(actor, false)

	if actor != env.Player && !modDB.Flag(nil, "UnaffectedBySlows") {
		slowEffect := modDB.SumNegativeValues(mod.TypeIncrease, nil, "ActionSpeed")
		slowEffect += modDB.SumNegativeValues(mod.TypeIncrease, nil, "MovementSpeed")
		slowEffect += modDB.SumNegativeValues(mod.TypeIncrease, nil, "TemporalChainsActionSpeed")
		if slowEffect != 0 {
			modDB.AddMod(newFlagMod("Condition:Slowed", ""))
		}
	}
}

//pob2:src/Modules/CalcPerform.lua:1021-1043
func actionSpeedMod(actor *Actor) float64 {
	modDB := actor.ModDB
	minimumActionSpeed := modDB.Max(nil, "MinimumActionSpeed")
	maximumActionSpeedReduction := modDB.Max(nil, "MaximumActionSpeedReduction")
	var actionSpeedSum, tempChainsSum float64
	if modDB.Flag(nil, "UnaffectedBySlows") {
		actionSpeedSum = modDB.SumPositiveValues(mod.TypeIncrease, nil, "ActionSpeed")
		tempChainsSum = modDB.SumPositiveValues(mod.TypeIncrease, nil, "TemporalChainsActionSpeed")
	} else {
		actionSpeedSum = modDB.Sum(mod.TypeIncrease, nil, "ActionSpeed")
		tempChainsSum = modDB.Sum(mod.TypeIncrease, nil, "TemporalChainsActionSpeed")
	}
	speedMod := 1 + (math.Max(-miscConstant("TemporalChainsEffectCap"), tempChainsSum)+actionSpeedSum)/100
	speedMod = math.Max(minimumActionSpeed/100, speedMod)
	if maximumActionSpeedReduction > 0 {
		speedMod = math.Min((100-maximumActionSpeedReduction)/100, speedMod)
	}
	return speedMod
}

// perform is calcs.perform as far as the defence slice's inputs: keystones, attributes and the
// life/mana/spirit pools, plus charges. Minions, flasks, buffs and the rest of the pass are not ported.
//
//pob2:src/Modules/CalcPerform.lua:1193-3320
func perform(env *Environment) {
	env.KeystonesAdded = map[string]bool{}
	mergeKeystones(env)

	env.Player.Output = map[string]float64{}
	env.Enemy.Output = map[string]float64{}

	// countsAsDualWielding and the Facebreaker fields are not ported.
	//pob2:block:src/Modules/CalcSetup.lua:2120-2153
	env.Player.WeaponData1Type = "None"
	weapon1 := env.Player.ItemList["Weapon 1"]
	if weapon1 != nil && weapon1.WeaponData[1] != nil {
		env.Player.WeaponData1Type = weapon1.Base.Type
	}
	if weapon2 := env.Player.ItemList["Weapon 2"]; weapon2 == nil {
		if weapon1 == nil {
			for _, m := range env.ModDB.Mods["Keystone"] {
				if v, _ := m.Value().ValueList.(string); v == "Hollow Palm Technique" {
					env.Player.WeaponData2Type = "None"
					env.ModDB.Conditions["HollowPalm"] = true
					break
				}
			}
		}
	} else if weapon2.WeaponData[2] != nil {
		env.Player.WeaponData2Type = weapon2.Base.Type
	}

	modDB := env.ModDB
	output := env.Player.Output

	//pob2:block:src/Modules/CalcPerform.lua:1249-1256
	if modDB.Flag(nil, "CryWolfMinimumPower") && modDB.Sum(mod.TypeBase, nil, "WarcryPower") < 10 {
		modDB.AddMod(newMod("WarcryPower", mod.TypeOverride, 10, "Minimum Warcry Power from CryWolf"))
	}
	if modDB.Flag(nil, "WarcryInfinitePower") {
		modDB.AddMod(newMod("WarcryPower", mod.TypeOverride, 999999, "Warcries have infinite power"))
	}
	output["WarcryPower"] = overrideOr(modDB, "WarcryPower", modDB.Sum(mod.TypeBase, nil, "WarcryPower"))
	modDB.Multipliers["WarcryPower"] = output["WarcryPower"]

	//pob2:block:src/Modules/CalcPerform.lua:1258-1262
	applyEnemyModifiers(env.Player, true)
	applyEnemyModifiers(env.Enemy, true)

	// Stat sorting category calcs
	//pob2:block:src/Modules/CalcPerform.lua:1450
	output["EffectiveLootRarityMod"] = calcLibMod(modDB, nil, "LootRarity")

	// Special Rarity / Quantity Calc for Bisco's
	//pob2:block:src/Modules/CalcPerform.lua:1458-1461
	lootQuantityNormalEnemies := modDB.Sum(mod.TypeIncrease, nil, "LootQuantityNormalEnemies")
	output["LootQuantityNormalEnemies"] = flagOr(lootQuantityNormalEnemies > 0, lootQuantityNormalEnemies+modDB.Sum(mod.TypeIncrease, nil, "LootQuantity"), 0)
	lootRarityMagicEnemies := modDB.Sum(mod.TypeIncrease, nil, "LootRarityMagicEnemies")
	output["LootRarityMagicEnemies"] = flagOr(lootRarityMagicEnemies > 0, lootRarityMagicEnemies+modDB.Sum(mod.TypeIncrease, nil, "LootRarity"), 0)

	addSlotBonusEffectsPerform(env)
	addMagebloodLegacies(env)

	// flask breakdown; the oracle builds the breakdown, so these always run.
	//pob2:block:src/Modules/CalcPerform.lua:1580
	//pob2:block:src/Modules/CalcPerform.lua:1591-1602
	{
		effectInc := modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "FlaskEffect")
		chargesGenerated := modDB.Sum(mod.TypeBase, nil, "FlaskChargesGenerated")
		utilityChargesGenerated := modDB.Sum(mod.TypeBase, nil, "UtilityFlaskChargesGenerated")
		lifeChargesGenerated := modDB.Sum(mod.TypeBase, nil, "LifeFlaskChargesGenerated")
		manaChargesGenerated := modDB.Sum(mod.TypeBase, nil, "ManaFlaskChargesGenerated")

		output["FlaskEffect"] = effectInc
		output["FlaskChargeGen"] = chargesGenerated
		output["LifeFlaskChargeGen"] = chargesGenerated + lifeChargesGenerated
		output["ManaFlaskChargeGen"] = chargesGenerated + manaChargesGenerated
		output["UtilityFlaskChargeGen"] = chargesGenerated + utilityChargesGenerated
	}

	// No charm is equipped until items exist, so mergeCharms only counts the empty slots.
	//pob2:block:src/Modules/CalcPerform.lua:1779-1787
	charmEffectInc := modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "CharmEffect")
	charmLimit := env.ModDB.Sum(mod.TypeBase, nil, "CharmLimit")
	if o := env.ModDB.Override(nil, "CharmLimit"); o != nil {
		charmLimit = o.Float()
	}
	charmLimit = math.Min(charmLimit, 3)
	output["CharmEffect"] = charmEffectInc
	output["CharmLimit"] = charmLimit
	//pob2:block:src/Modules/CalcPerform.lua:1858-1866
	if env.ModeCombat {
		//pob2:block:src/Modules/CalcPerform.lua:1849
		env.Player.Output["EmptyCharms"] = charmLimit
		mergeKeystones(env)
	}

	//pob2:block:src/Modules/CalcPerform.lua:1869
	doActorAttribsConditions(env, env.Player)

	// Process attribute requirements
	//pob2:block:src/Modules/CalcPerform.lua:1896
	processAttributeRequirements(env, env.requirementsTable)

	//pob2:block:src/Modules/CalcPerform.lua:2952
	doActorCharges(env, env.Player)

	// ExtraCurse and party curses need skill data, and the slot loop needs skills' curses, so only the limits run.
	//pob2:block:src/Modules/CalcPerform.lua:3025-3026
	if env.ModDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
		env.Player.Output["EnemyCurseLimit"] = env.Player.Output["PowerChargesMax"]
	} else {
		env.Player.Output["EnemyCurseLimit"] = env.ModDB.Sum(mod.TypeBase, nil, "EnemyCurseLimit")
	}
	env.Player.Output["EnemyMarkLimit"] = env.ModDB.Sum(mod.TypeBase, nil, "EnemyMarkLimit")

	//pob2:block:src/Modules/CalcPerform.lua:3266-3267
	doActorCharges(env, env.Player)
	doActorMisc(env, env.Player)

	//pob2:block:src/Modules/CalcPerform.lua:3320
	doActorLifeManaSpirit(env.Player)
	applyNonDamagingAilments(env)

	//pob2:block:src/Modules/CalcPerform.lua:3382-3383
	doActorCharges(env, env.Enemy)
	doActorMisc(env, env.Enemy)

	//pob2:block:src/Modules/CalcPerform.lua:3456
	defence(env, env.Player)

	//pob2:block:src/Modules/CalcPerform.lua:3510-3512
	buildDefenceEstimations(env, env.Player)
}
