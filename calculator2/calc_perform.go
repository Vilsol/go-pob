package calculator2

import (
	"math"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

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

// The weapon-type conditions need data.weaponTypeInfo and the skill-flag conditions need a main skill;
// neither is ported yet, so only an empty weapon slot is recognised and the mode_combat block runs from
// its shrine buffs on.
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
	}
	for _, slotName := range []string{"Helmet", "Body Armour", "Gloves", "Boots"} {
		if _, ok := actor.ItemList[slotName]; ok {
			condList["Using"+slotName] = true
		}
	}
	if actor.WeaponData1Type != "" && actor.WeaponData2Type != "" {
		condList["DualWielding"] = true
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
// life/mana/spirit pools. Minions, flasks, buffs, charges and the rest of the pass are not ported.
//
//pob2:src/Modules/CalcPerform.lua:1193-3320
func perform(env *Environment) {
	env.KeystonesAdded = map[string]bool{}
	mergeKeystones(env)

	env.Player.Output = map[string]float64{}
	env.Enemy.Output = map[string]float64{}

	//pob2:block:src/Modules/CalcSetup.lua:2120
	env.Player.WeaponData1Type = "None"
	if w, ok := env.Player.ItemList["Weapon 1"]; ok {
		env.Player.WeaponData1Type = w.Type
	}

	// No charm is equipped until items exist, so mergeCharms only counts the empty slots.
	//pob2:block:src/Modules/CalcPerform.lua:1781
	charmLimit := env.ModDB.Sum(mod.TypeBase, nil, "CharmLimit")
	if o := env.ModDB.Override(nil, "CharmLimit"); o != nil {
		charmLimit = o.Float()
	}
	charmLimit = math.Min(charmLimit, 3)
	//pob2:block:src/Modules/CalcPerform.lua:1858-1866
	if env.ModeCombat {
		//pob2:block:src/Modules/CalcPerform.lua:1849
		env.Player.Output["EmptyCharms"] = charmLimit
		mergeKeystones(env)
	}

	//pob2:block:src/Modules/CalcPerform.lua:1869
	doActorAttribsConditions(env, env.Player)

	//pob2:block:src/Modules/CalcPerform.lua:3320
	doActorLifeManaSpirit(env.Player)

	//pob2:block:src/Modules/CalcPerform.lua:3456
	defence(env, env.Player)
}
