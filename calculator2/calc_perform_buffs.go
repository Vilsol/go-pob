package calculator2

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// buffState is the buffs, guards, debuffs and curses tables combineBuffs fills; notBuff is modList.notBuff.
type buffState struct {
	buffs, guards, debuffs map[string]*moddb.ModList
	notBuff                map[string]bool
	curses, minionCurses   []*performCurse
}

// performCurse is one curses entry: a curse or mark the player or an ally applies.
type performCurse struct {
	name                   string
	modList                *moddb.ModList
	buffModList            *moddb.ModList
	priority               float64
	fromPlayer             bool
	isMark                 bool
	ignoreCurseLimit       bool
	socketedCursesHexLimit bool
}

//pob2:block:src/Modules/Data.lua:341-368
var cursePriority = map[string]float64{
	"Temporal Chains":    1,
	"Enfeeble":           2,
	"Vulnerability":      3,
	"Elemental Weakness": 4,
	"Flammability":       5,
	"Frostbite":          6,
	"Conductivity":       7,
	"Despair":            8,
	"Punishment":         9,
	"Warlord's Mark":     10,
	"Assassin's Mark":    11,
	"Sniper's Mark":      12,
	"Poacher's Mark":     13,
	"SocketPriorityBase": 100,
	"Weapon 1":           1000,
	"Amulet":             2000,
	"Helmet":             3000,
	"Weapon 2":           4000,
	"Body Armour":        5000,
	"Gloves":             6000,
	"Boots":              7000,
	"Ring 1":             8000,
	"Ring 2":             9000,
	"Ring 3":             10000,
	"CurseFromEquipment": 11000,
	"CurseFromAura":      20000,
}

//pob2:src/Modules/CalcPerform.lua:531-561
func determineCursePriority(curseName string, activeSkill *ActiveSkill) float64 {
	source := ""
	slot := ""
	socket := 1.0
	if activeSkill != nil && activeSkill.SocketGroup != nil {
		source = activeSkill.SocketGroup.Source
		slot = activeSkill.SocketGroup.Slot
		for k, v := range activeSkill.SocketGroup.GemList {
			if v.GemData != nil && v.GemData.Data.Name == curseName {
				// We need to enforce a limit of 8 here to avoid collision with data.cursePriority["CurseFromEquipment"]
				socket = math.Min(float64(k+1), 8)
				break
			}
		}
	}
	basePriority := cursePriority[curseName]
	socketPriority := socket * cursePriority["SocketPriorityBase"]
	slotPriority := cursePriority[strings.ReplaceAll(slot, " Swap", "")]
	sourcePriority := 0.0
	if activeSkill != nil && activeSkill.SkillTypes["Aura"] {
		sourcePriority = cursePriority["CurseFromAura"]
	} else if source != "" {
		sourcePriority = cursePriority["CurseFromEquipment"]
	}
	if source != "" && (slotPriority == cursePriority["Ring 2"] || slotPriority == cursePriority["Ring 3"]) {
		// Implicit and explicit curses from rings have equal priority; only curses from socketed skill gems care about which ring slot they're equipped in
		slotPriority = cursePriority["Ring 1"]
	}
	return basePriority + socketPriority + slotPriority + sourcePriority
}

// Merge an instance of a buff, taking the highest value of each modifier
//
//pob2:src/Modules/CalcPerform.lua:42-64
func mergeBuff(src []mod.Mod, destTable map[string]*moddb.ModList, destKey string) {
	if destTable[destKey] == nil {
		destTable[destKey] = newModList()
	}
	dest := destTable[destKey]
	for _, m := range src {
		match := false
		if m.Type() != mod.TypeList {
			for index, destMod := range dest.Mods() {
				if moddb.CompareModParams(m, destMod) {
					if dv, sv := destMod.Value(), m.Value(); dv != nil && sv != nil && dv.Type() == mod.ModValueMultiTypeFloat && sv.Float() > dv.Float() {
						dest.Mods()[index] = m
					}
					match = true
					break
				}
			}
		}
		if !match {
			dest.AddMod(m)
		}
	}
}

// scaleAddList is ModStore:ScaleAddList with its roundToNearest argument.
//
//pob2:src/Classes/ModStore.lua:127-135
func scaleAddList(dest *moddb.ModList, modList []mod.Mod, scale float64, roundToNearest bool) {
	for _, m := range modList {
		if scale == 1 {
			dest.AddMod(m)
		} else {
			dest.ScaleAddMod(m, scale, roundToNearest)
		}
	}
}

// statSetSkillFlags is activeEffect.statSet.skillFlags, which CALCS mode reads from statSetCalcs.
func statSetSkillFlags(activeSkill *ActiveSkill) map[string]bool {
	if activeSkill.ActiveEffect.StatSet == nil || activeSkill.ActiveEffect.StatSet.SkillFlags == nil {
		return map[string]bool{}
	}
	return activeSkill.ActiveEffect.StatSet.SkillFlags
}

// extraEffectModList collects modDB:List(cfg, name) values' mods, summing those with equal parameters.
//
//pob2:block:src/Modules/CalcPerform.lua:2291-2304
func extraEffectModList(modDB *moddb.ModDB, cfg *moddb.ListCfg, name string) []mod.Mod {
	var extraModList []mod.Mod
	for _, value := range modDB.List(cfg, name) {
		var valueMod mod.Mod
		switch v := value.(type) {
		case mod.ExtraAuraEffect:
			valueMod = v.Mod
		case LuaTable:
			valueMod, _ = v["mod"].(mod.Mod)
		}
		if valueMod == nil {
			continue
		}
		add := true
		for i, m := range extraModList {
			if moddb.CompareModParams(m, valueMod) {
				merged := m.Clone()
				merged.SetValue(mod.NewModValueFloat(m.Value().Float() + valueMod.Value().Float()))
				extraModList[i] = merged
				add = false
				break
			}
		}
		if add {
			extraModList = append(extraModList, valueMod.Clone())
		}
	}
	return extraModList
}

// Calculate number of active heralds and auras affecting self
//
//pob2:block:src/Modules/CalcPerform.lua:1994-2008
func countHeraldsAndAuras(env *Environment) {
	if !env.ModeBuffs {
		return
	}
	modDB := env.ModDB
	heraldList := map[string]bool{}
	auraList := map[string]bool{}
	for _, activeSkill := range env.Player.ActiveSkillList {
		skillName := activeSkill.SkillCfg.SkillName
		if activeSkill.SkillTypes["Herald"] && !heraldList[skillName] {
			heraldList[skillName] = true
			modDB.Multipliers["Herald"] = modDB.Multipliers["Herald"] + 1
			modDB.Conditions["AffectedByHerald"] = true
		} else if activeSkill.SkillTypes["Aura"] && !activeSkill.SkillTypes["AuraAffectsEnemies"] && !truthy(activeSkill.SkillData["auraCannotAffectSelf"]) && !auraList[skillName] {
			auraList[skillName] = true
			modDB.Multipliers["AuraAffectingSelf"] = modDB.Multipliers["AuraAffectingSelf"] + 1
		}
	}
}

// combineBuffs gathers the player's skill buffs, guards, debuffs and curses and applies them. env.minion,
// the party tab's exports and allies' buffs (env.partyMembers) are not ported, so their branches are
// absent; warcries need calcSkillDuration and calcSkillCooldown from CalcOffence.
//
//pob2:block:src/Modules/CalcPerform.lua:2034-2050
//pob2:block:src/Modules/CalcPerform.lua:2136-2610
func combineBuffs(env *Environment) *buffState {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	st := &buffState{buffs: map[string]*moddb.ModList{}, guards: map[string]*moddb.ModList{}, debuffs: map[string]*moddb.ModList{}, notBuff: map[string]bool{}}
	buffs, guards, debuffs := st.buffs, st.guards, st.debuffs
	// TODO: Spectre ally, minion and player modifiers (CalcPerform.lua:2049-2097) need data.minions.

	appliedCombustion := false
	for _, activeSkill := range env.Player.ActiveSkillList {
		skillFlags := statSetSkillFlags(activeSkill)
		skillModList := activeSkill.SkillModList
		skillCfg := activeSkill.SkillCfg
		for _, buff := range activeSkill.BuffList {
			buffNameCompressed := strings.ReplaceAll(buff.Name, " ", "")
			if c, _ := skillModList.GetCondition(buff.Cond, skillCfg, false); buff.Cond != "" && !c {
				// Nothing!
				continue
			}
			if c, _ := enemyDB.GetCondition(buff.EnemyCond, nil, false); buff.EnemyCond != "" && !c {
				// Also nothing :/
				continue
			}
			switch buff.Type {
			case "Buff":
				if env.ModeBuffs && (!skillFlags["totem"] || buff.AllowTotemBuff) {
					var buffCfg *moddb.ListCfg
					var modStore moddb.ModStoreFuncs = modDB
					if buff.ActiveSkillBuff {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !buff.ApplyNotPlayer {
						modDB.Conditions["AffectedBy"+buffNameCompressed] = true
						srcList := newModList()
						inc := modStore.Sum(mod.TypeIncrease, buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") + skillModList.Sum(mod.TypeIncrease, buffCfg, buffNameCompressed+"Effect")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf") * calcLibMod(modStore, buffCfg, "Magnitude")
						scaleAddList(srcList, buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods(), buffs, buff.Name)
						if truthy(activeSkill.SkillData["thisIsNotABuff"]) {
							st.notBuff[buff.Name] = true
						}
					}
				}
			case "Guard":
				if env.ModeBuffs && (!skillFlags["totem"] || buff.AllowTotemBuff) {
					var buffCfg *moddb.ListCfg
					var modStore moddb.ModStoreFuncs = modDB
					if buff.ActiveSkillBuff {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !buff.ApplyNotPlayer {
						srcList := newModList()
						inc := modStore.Sum(mod.TypeIncrease, buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf") * calcLibMod(modStore, buffCfg, "Magnitude")
						scaleAddList(srcList, buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods(), guards, buff.Name)
					}
				}
			case "Warcry":
				// TODO: warcry buffs (CalcPerform.lua:2197-2287) need calcSkillDuration and calcSkillCooldown from CalcOffence.
			case "Aura":
				if env.ModeBuffs {
					// Check for extra modifiers to apply to aura skills
					extraAuraModList := extraEffectModList(modDB, skillCfg, "ExtraAuraEffect")
					if !truthy(activeSkill.SkillData["auraCannotAffectSelf"]) {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf") * calcLibMod(skillModList, skillCfg, "Magnitude")
						mult := (1 + inc/100) * more
						modDB.Conditions["AffectedByAura"] = true
						if strings.HasPrefix(buff.Name, "Vaal") {
							modDB.Conditions["AffectedBy"+strings.ReplaceAll(buff.Name[min(5, len(buff.Name)):], " ", "")] = true
						}
						modDB.Conditions["AffectedBy"+buffNameCompressed] = true
						srcList := newModList()
						scaleAddList(srcList, buff.ModList, mult, false)
						scaleAddList(srcList, extraAuraModList, mult, false)
						mergeBuff(srcList.Mods(), buffs, buff.Name)
					}
					//pob2:block:src/Modules/CalcPerform.lua:2348-2383
					if statSetSkillFlags(env.Player.MainSkill)["totem"] && !(modDB.Flag(nil, "SelfAurasCannotAffectAllies") || modDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						mainSkillModList := env.Player.MainSkill.SkillModList
						moddb.SetCondition(mainSkillModList.ModStore, "AffectedBy"+buffNameCompressed, true)
						moddb.SetCondition(mainSkillModList.ModStore, "AffectedByAura", true)
						srcList := newModList()
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
						scale := (1 + inc/100) * more
						scale = math.Max(scale, 0)
						for _, modList := range [][]mod.Mod{extraAuraModList, buff.ModList} {
							for _, m := range modList {
								if m.Name() == "EnergyShield" || m.Name() == "Armour" || m.Name() == "Evasion" || totemResistNameRE.MatchString(m.Name()) {
									totemMod := m.Clone().SetName("Totem" + m.Name())
									if scale != 1 {
										scaleTotemModValue(totemMod, scale)
									}
									srcList.AddMod(totemMod)
								}
							}
						}
						mergeBuff(srcList.Mods(), buffs, "Totem "+buff.Name)
					}
					// check if aura has a debuff added to it, but does not have an auraDebuff
					//pob2:block:src/Modules/CalcPerform.lua:2385-2420
					if env.ModeEffective && len(modDB.List(skillCfg, "ExtraAuraDebuffEffect")) > 0 && !modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
						auraDebuffFound := false
						for _, b := range activeSkill.BuffList {
							if b.Type == "AuraDebuff" {
								auraDebuffFound = true
								break
							}
						}
						if !auraDebuffFound {
							extraAuraModList := extraEffectModList(modDB, skillCfg, "ExtraAuraDebuffEffect")
							newList := newModList()
							for _, m := range extraAuraModList {
								newList.AddMod(m)
							}
							mergeBuff(newList.Mods(), debuffs, buff.Name)
						}
					}
				}
			case "Debuff", "AuraDebuff":
				//pob2:block:src/Modules/CalcPerform.lua:2422-2490
				var stackCount float64
				if buff.StackVar != "" {
					stackCount = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackVar)
					if buff.HasStackLimit {
						stackCount = math.Min(stackCount, buff.StackLimit)
					} else if buff.StackLimitVar != "" {
						stackCount = math.Min(stackCount, skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackLimitVar))
					}
				} else {
					stackCount = 1
					if v, ok := activeSkill.SkillData["stackCount"].(float64); ok {
						stackCount = v
					}
				}
				if env.ModeEffective && stackCount > 0 {
					enemyDB.Conditions["AffectedBy"+buffNameCompressed] = true
					modDB.Conditions["AffectedBy"+buffNameCompressed] = true
					srcList := newModList()
					mult := 1.0
					var extraAuraModList []mod.Mod
					if buff.Type == "AuraDebuff" {
						extraAuraModList = extraEffectModList(modDB, skillCfg, "ExtraAuraDebuffEffect")
						mult = 0
						if !modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
							inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
							mult = (1 + inc/100) * more
						}
					}
					if buff.Type == "Debuff" {
						specificDebuffMult := calcLibMod(skillModList, skillCfg, buffNameCompressed+"Magnitude") // non-skill mods specific to that debuff type
						skillMagnitudeMult := calcLibMod(skillModList, skillCfg, "Magnitude")
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "DebuffEffect")
						more := skillModList.More(skillCfg, "DebuffEffect") * skillMagnitudeMult * specificDebuffMult
						mult = (1 + inc/100) * more
					}
					scaleAddList(srcList, buff.ModList, mult*stackCount, false)
					scaleAddList(srcList, extraAuraModList, mult*stackCount, false)
					if _, ok := activeSkill.SkillData["stackCount"]; ok || buff.StackVar != "" {
						srcList.AddMod(newMod("Multiplier:"+buff.Name+"Stack", mod.TypeBase, stackCount, mod.Source(buff.Name)))
					}
					mergeBuff(srcList.Mods(), debuffs, buff.Name)
				}
			case "Curse", "CurseBuff":
				//pob2:block:src/Modules/CalcPerform.lua:2491-2539
				mark := activeSkill.SkillTypes["Mark"]
				modDB.Conditions["SelfCast"+buffNameCompressed] = !(activeSkill.SkillTypes["Triggered"] || activeSkill.SkillTypes["Aura"])
				if env.ModeEffective && (!enemyDB.Flag(nil, "Hexproof") || modDB.Flag(nil, "CursesIgnoreHexproof") || truthy(activeSkill.SkillData["ignoreCurseLimit"]) || truthy(activeSkill.SkillData["ignoreHexproof"])) || mark {
					curse := &performCurse{
						name:                   buff.Name,
						fromPlayer:             true,
						priority:               determineCursePriority(buff.Name, activeSkill),
						isMark:                 mark,
						ignoreCurseLimit:       (modDB.Flag(activeSkill.SkillCfg, "CursesIgnoreCurseLimit") || truthy(activeSkill.SkillData["ignoreCurseLimit"]) || activeSkill.SkillModList.Flag(nil, "CursesIgnoreCurseLimit")) && !mark,
						socketedCursesHexLimit: modDB.Flag(activeSkill.SkillCfg, "SocketedCursesAdditionalLimit"),
					}
					inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
					if activeSkill.SkillTypes["Aura"] {
						inc = inc + skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect")
					}
					more := skillModList.More(skillCfg, "CurseEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
					// This is non-ideal, but the only More for enemy is the boss effect
					if !curse.isMark {
						more = more * enemyDB.More(nil, "CurseEffectOnSelf")
					}
					mult := 0.0
					if !(modDB.Flag(nil, "SelfAurasOnlyAffectYou") && activeSkill.SkillTypes["Aura"]) { //If your aura only effect you blasphemy does nothing
						mult = (1 + inc/100) * more
					}
					if buff.Type == "Curse" {
						curse.modList = newModList()
						scaleAddList(curse.modList, buff.ModList, mult, true)
					} else {
						// Curse applies a buff; scale by curse effect, then buff effect
						temp := newModList()
						scaleAddList(temp, buff.ModList, mult, true)
						curse.buffModList = newModList()
						buffInc := modDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
						buffMore := modDB.More(skillCfg, "BuffEffectOnSelf")
						scaleAddList(curse.buffModList, temp.Mods(), (1+buffInc/100)*buffMore, true)
					}
					st.curses = append(st.curses, curse)
				}
			case "Link":
				// Links apply only to minions and party members, neither of which is ported.
			}
		}
		//pob2:block:src/Modules/CalcPerform.lua:2588-2610
		if activeSkill.SkillModList.Flag(nil, "Condition:CanWither") {
			effect := math.Floor(5 * (1 + modDB.Sum(mod.TypeIncrease, nil, "WitherEffect")/100))
			modDB.AddMod(newMod("WitherEffectStack", mod.TypeMAX, effect, ""))
		}
		if activeSkill.SkillModList.Flag(nil, "ApplyCriticalWeakness") {
			modDB.AddMod(newFlagMod("ApplyCriticalWeakness", ""))
		}
		if activeSkill.SkillModList.Flag(nil, "CanParry") {
			modDB.AddMod(newFlagMod("CanParry", ""))
		}
		//Handle combustion
		if enemyDB.Flag(nil, "Condition:Ignited") && (activeSkill.SkillTypes["Damage"] || activeSkill.SkillTypes["Attack"]) && !appliedCombustion {
			for _, support := range activeSkill.SupportList {
				if support.GrantedEffect.Name == "Combustion" {
					if !activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "CannotIgnite") {
						value := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "CombustionFireResist")
						enemyDB.AddMod(newMod("FireResist", mod.TypeBase, value, "Combustion", mod.GlobalEffect("Debuff").Name("Combustion"), mod.Condition("Ignited")))
						appliedCombustion = true
					}
					break
				}
			}
		}
		//pob2:block:src/Modules/CalcPerform.lua:2611-2822
		if activeSkill.Minion != nil && activeSkill.Minion.ActiveSkillList != nil {
			combineMinionSkillBuffs(env, st, activeSkill)
		}
	}
	return st
}

// combineMinionSkillBuffs adds the buffs, auras, curses and debuffs of a minion's skills. env.minion,
// the party tab's exports and allies' buffs are not ported, so the branches that read them are absent:
// only the player's buffs, the totem copy, minionCurses and the enemy's debuffs are built.
//
//pob2:block:src/Modules/CalcPerform.lua:2611-2822
func combineMinionSkillBuffs(env *Environment, st *buffState, activeSkill *ActiveSkill) {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	buffs, debuffs := st.buffs, st.debuffs
	castingMinion := activeSkill.Minion
	for _, activeMinionSkill := range castingMinion.ActiveSkillList {
		skillModList := activeMinionSkill.SkillModList
		skillCfg := activeMinionSkill.SkillCfg
		enable := truthy(activeMinionSkill.SkillData["enable"])
		for _, buff := range activeMinionSkill.BuffList {
			buffNameCompressed := strings.ReplaceAll(buff.Name, " ", "")
			switch buff.Type {
			case "Buff":
				if env.ModeBuffs && enable {
					var buffCfg *moddb.ListCfg
					var modStore moddb.ModStoreFuncs = castingMinion.ModDB
					if buff.ActiveSkillBuff {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if buff.ApplyAllies {
						modDB.Conditions["AffectedBy"+buffNameCompressed] = true
						srcList := newModList()
						inc := modStore.Sum(mod.TypeIncrease, buffCfg, "BuffEffect", "BuffEffectOnPlayer") + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnPlayer") * modDB.More(nil, "BuffEffectOnSelf") * calcLibMod(skillModList, buffCfg, "Magnitude")
						scaleAddList(srcList, buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods(), buffs, buff.Name)
						mergeBuff(buff.ModList, buffs, buff.Name)
						if truthy(activeMinionSkill.SkillData["thisIsNotABuff"]) {
							st.notBuff[buff.Name] = true
						}
					}
					// The minion's own buff (CalcPerform.lua:2640-2657) needs env.minion.
				}
			case "Aura":
				if env.ModeBuffs && enable {
					// Check for extra modifiers to apply to aura skills
					extraAuraModList := extraEffectModList(castingMinion.ModDB, skillCfg, "ExtraAuraEffect")
					minionDB := castingMinion.ModDB
					if !(minionDB.Flag(nil, "SelfAurasCannotAffectAllies") || minionDB.Flag(nil, "SelfAurasOnlyAffectYou") || minionDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						if !modDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !modDB.Conditions["AffectedBy"+buffNameCompressed] {
							inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnPlayer", "AuraBuffEffect") + modDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") * modDB.More(skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf") * calcLibMod(skillModList, skillCfg, "Magnitude")
							mult := (1 + inc/100) * more
							modDB.Conditions["AffectedByAura"] = true
							if strings.HasPrefix(buff.Name, "Vaal") {
								modDB.Conditions["AffectedBy"+strings.ReplaceAll(buff.Name[min(5, len(buff.Name)):], " ", "")] = true
							}
							modDB.Conditions["AffectedBy"+buffNameCompressed] = true
							srcList := newModList()
							scaleAddList(srcList, buff.ModList, mult, false)
							scaleAddList(srcList, extraAuraModList, mult, false)
							setSpectreSource(srcList, buff.Name, activeSkill, castingMinion)
							mergeBuff(srcList.Mods(), buffs, buff.Name)
						}
						// The env.minion half (CalcPerform.lua:2695-2709) and buffExports are not ported.
						if statSetSkillFlags(env.Player.MainSkill)["totem"] && !env.Player.MainSkill.SkillModList.ModStore.Conditions["AffectedBy"+buffNameCompressed] {
							mainSkillModList := env.Player.MainSkill.SkillModList
							moddb.SetCondition(mainSkillModList.ModStore, "AffectedBy"+buffNameCompressed, true)
							moddb.SetCondition(mainSkillModList.ModStore, "AffectedByAura", true)
							srcList := newModList()
							inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
							scale := (1 + inc/100) * more
							scale = math.Max(scale, 0)
							for _, modList := range [][]mod.Mod{extraAuraModList, buff.ModList} {
								for _, m := range modList {
									if m.Name() == "EnergyShield" || m.Name() == "Armour" || m.Name() == "Evasion" || totemResistNameRE.MatchString(m.Name()) {
										totemMod := m.Clone().SetName("Totem" + m.Name())
										if scale != 1 {
											scaleTotemModValue(totemMod, scale)
										}
										srcList.AddMod(totemMod)
									}
								}
							}
							setSpectreSource(srcList, "", activeSkill, castingMinion)
							mergeBuff(srcList.Mods(), buffs, "Totem "+buff.Name)
						}
					}
				}
			case "Curse":
				if env.ModeEffective && enable && (!enemyDB.Flag(nil, "Hexproof") || activeMinionSkill.SkillTypes["Mark"]) {
					curse := &performCurse{
						name:     buff.Name,
						priority: determineCursePriority(buff.Name, activeMinionSkill),
					}
					inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
					more := skillModList.More(skillCfg, "CurseEffect") * enemyDB.More(nil, "CurseEffectOnSelf") * calcLibMod(skillModList, skillCfg, "Magnitude")
					curse.modList = newModList()
					scaleAddList(curse.modList, buff.ModList, (1+inc/100)*more, false)
					st.minionCurses = append(st.minionCurses, curse)
				}
			case "Debuff", "AuraDebuff":
				var stackCount float64
				if buff.StackVar != "" {
					stackCount = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackVar)
					if buff.HasStackLimit {
						stackCount = math.Min(stackCount, buff.StackLimit)
					} else if buff.StackLimitVar != "" {
						stackCount = math.Min(stackCount, skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackLimitVar))
					}
				} else {
					stackCount = 1
					if v, ok := activeMinionSkill.SkillData["stackCount"].(float64); ok {
						stackCount = v
					}
				}
				if env.ModeEffective && stackCount > 0 {
					srcList := newModList()
					mult := 1.0
					if buff.Type == "AuraDebuff" {
						mult = 0
						if !skillModList.Flag(nil, "SelfAurasOnlyAffectYou") {
							inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
							mult = (1 + inc/100) * more
							if enemyDB.Conditions["AffectedBy"+buffNameCompressed] {
								mult = 0
							}
						}
					}
					enemyDB.Conditions["AffectedBy"+buffNameCompressed] = true
					if buff.Type == "Debuff" {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "DebuffEffect")
						more := skillModList.More(skillCfg, "DebuffEffect")
						mult = (1 + inc/100) * more
					}
					scaleAddList(srcList, buff.ModList, mult*stackCount, false)
					if v, ok := activeMinionSkill.SkillData["stackCount"]; ok || buff.StackVar != "" {
						// PoB writes skillData.stackCount, nil when only stackVar set it.
						n, _ := v.(float64)
						srcList.AddMod(newMod("Multiplier:"+buff.Name+"Stack", mod.TypeBase, n, mod.Source(buff.Name)))
					}
					mergeBuff(srcList.Mods(), debuffs, buff.Name)
				}
			}
		}
	}
}

// setSpectreSource renames the sources of a spectre's buff mods.
//
//pob2:src/Modules/CalcPerform.lua:1169-1181
func setSpectreSource(modList *moddb.ModList, sourceSkill string, activeSkill *ActiveSkill, castingMinion *Minion) {
	if activeSkill.SkillFlags["spectre"] {
		source := "Spectre:"
		if sourceSkill != "" {
			source = source + sourceSkill + " - " + castingMinion.MinionData.Name
		} else {
			source = source + castingMinion.MinionData.Name
		}
		for _, m := range modList.Mods() {
			m.Source(mod.Source(source))
		}
	}
}

var totemResistNameRE = regexp.MustCompile(`Resist?M?a?x?$`)

// scaleTotemModValue is the totem aura copy's value scaling.
//
//pob2:block:src/Modules/CalcPerform.lua:2371-2377
func scaleTotemModValue(totemMod mod.Mod, scale float64) {
	v := totemMod.Value()
	if v == nil {
		return
	}
	scaleNumber := func(x float64) float64 {
		if math.Floor(x) == x {
			r, _ := math.Modf(round(x*scale, 2))
			return r
		}
		return x * scale
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFloat:
		totemMod.SetValue(mod.NewModValueFloat(scaleNumber(v.Float())))
	}
}

// applyCursesAndBuffs checks for extra curses, fills the curse and mark slots, picks the guard and adds
// every buff, debuff and curse to its actor. Allies' curses and the minion's buffs are absent: neither
// env.minion nor env.partyMembers is ported.
//
//pob2:block:src/Modules/CalcPerform.lua:2957-3181
func applyCursesAndBuffs(env *Environment, st *buffState) {
	buffs, guards, debuffs, curses := st.buffs, st.guards, st.debuffs, st.curses
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	output := env.Player.Output

	// TODO: ExtraCurse (CalcPerform.lua:2957-3000) needs gemIdLookup; the parser keeps the skill name, not its id.

	// Set curse limit
	//pob2:block:src/Modules/CalcPerform.lua:3025-3027
	if modDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
		output["EnemyCurseLimit"] = output["PowerChargesMax"]
	} else {
		output["EnemyCurseLimit"] = modDB.Sum(mod.TypeBase, nil, "EnemyCurseLimit")
	}
	output["EnemyMarkLimit"] = modDB.Sum(mod.TypeBase, nil, "EnemyMarkLimit")

	// Temp different mark and curse slots to handle limits and priorities of both
	var curseSlotList, markSlotList []*performCurse
	for _, curse := range slices.Concat(curses, st.minionCurses) {
		// Calculate curses that ignore hex limit after
		if curse.ignoreCurseLimit || curse.socketedCursesHexLimit {
			continue
		}
		slot := 0
		skipAddingCurse := false
		// Check if we need to disable a certain curse aura.
		for _, activeSkill := range env.Player.ActiveSkillList {
			if len(activeSkill.BuffList) > 0 && curse.name == activeSkill.BuffList[0].Name && activeSkill.SkillTypes["Aura"] {
				if modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
					skipAddingCurse = true
				}
				break
			}
		}
		currentSlots := &curseSlotList
		limit := output["EnemyCurseLimit"]
		if curse.isMark {
			currentSlots = &markSlotList
			limit = output["EnemyMarkLimit"]
		}
		for i := 1; float64(i) <= limit; i++ {
			if i > len(*currentSlots) {
				slot = i
				break
			} else if (*currentSlots)[i-1].name == curse.name {
				if (*currentSlots)[i-1].priority < curse.priority {
					slot = i
				} else {
					slot = 0
				}
				break
			} else if (*currentSlots)[i-1].priority < curse.priority {
				slot = i
			}
		}
		if slot != 0 && !skipAddingCurse {
			if slot > len(*currentSlots) {
				*currentSlots = append(*currentSlots, curse)
			} else {
				(*currentSlots)[slot-1] = curse
			}
		}
	}

	// Merge curse and mark slots as we now process curse ignoring hex limit
	curseSlots := append(append([]*performCurse{}, curseSlotList...), markSlotList...)

	for _, curse := range slices.Concat(curses, st.minionCurses) {
		if curse.ignoreCurseLimit {
			skipAddingCurse := false
			for i := range curseSlots {
				if curseSlots[i].name == curse.name {
					// if curse is higher priority, replace current curse with it, otherwise if same or lower priority skip it entirely
					if curseSlots[i].priority < curse.priority {
						curseSlots[i] = curse
					}
					skipAddingCurse = true
					break
				}
			}
			if !skipAddingCurse {
				curseSlots = append(curseSlots, curse)
			}
		}
		if curse.socketedCursesHexLimit {
			socketedCursesHexLimitValue := modDB.Sum(mod.TypeBase, nil, "SocketedCursesHexLimitValue")
			skipAddingCurse := false
			for i := range curseSlots {
				if curseSlots[i].name == curse.name {
					// if curse is higher priority, replace current curse with it, otherwise if same or lower priority skip it entirely
					if curseSlots[i].priority < curse.priority {
						curseSlots[i] = curse
					}
					skipAddingCurse = true
					break
				}
				if float64(i+1) >= socketedCursesHexLimitValue {
					skipAddingCurse = true
				}
			}
			if !skipAddingCurse {
				curseSlots = append(curseSlots, curse)
			}
		}
	}

	// Process guard buffs
	// PoB walks guards in pairs() order; the names are sorted here.
	//pob2:block:src/Modules/CalcPerform.lua:3117-3140
	type guardSlot struct {
		name    string
		modList *moddb.ModList
	}
	var guardSlots []guardSlot
	nonVaal := false
	guardNames := make([]string, 0, len(guards))
	for name := range guards {
		guardNames = append(guardNames, name)
	}
	sort.Strings(guardNames)
	for _, name := range guardNames {
		modList := guards[name]
		if name == "Vaal Molten Shell" {
			guardSlots = []guardSlot{{name: name, modList: modList}}
			nonVaal = false
			break
		} else if strings.HasPrefix(name, "Vaal") {
			guardSlots = append(guardSlots, guardSlot{name: name, modList: modList})
		} else if !nonVaal {
			guardSlots = append(guardSlots, guardSlot{name: name, modList: modList})
			nonVaal = true
		}
	}
	if nonVaal {
		modDB.Conditions["AffectedByNonVaalGuardSkill"] = true
	}
	for _, guard := range guardSlots {
		modDB.Conditions["AffectedByGuardSkill"] = true
		modDB.Conditions["AffectedBy"+strings.ReplaceAll(guard.name, " ", "")] = true
		mergeBuff(guard.modList.Mods(), buffs, guard.name)
	}

	// Apply buff/debuff modifiers
	//pob2:block:src/Modules/CalcPerform.lua:3142-3181
	for _, name := range sortedModListKeys(buffs) {
		modList := buffs[name]
		modDB.AddList(modList)
		if !st.notBuff[name] {
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
		}
	}
	for _, name := range sortedModListKeys(debuffs) {
		enemyDB.AddList(debuffs[name])
	}
	cursesInCurseSlots := 0
	for _, slot := range curseSlots {
		if !slot.isMark {
			cursesInCurseSlots++
		}
	}
	modDB.Multipliers["CurseOnEnemy"] = float64(cursesInCurseSlots)
	for _, slot := range curseSlots {
		enemyDB.Conditions["Cursed"] = true
		if slot.isMark {
			enemyDB.Conditions["Marked"] = true
		}
		if slot.modList != nil {
			enemyDB.AddList(slot.modList)
		}
		if slot.buffModList != nil {
			modDB.AddList(slot.buffModList)
		}
	}
}

func sortedModListKeys(m map[string]*moddb.ModList) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// applyExtraAuras fixes the configured impale stacks and applies the player's extra auras. Allies' extra
// auras and the minion's copy are absent: neither env.partyMembers nor env.minion is ported.
//
//pob2:block:src/Modules/CalcPerform.lua:3183-3263
func applyExtraAuras(env *Environment) {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB

	// Fix the configured impale stacks on the enemy
	// 		If the config is missing (blank), then use the maximum number of stacks
	//		If the config is larger than the maximum number of stacks, replace it with the correct maximum
	maxImpaleStacks := modDB.Sum(mod.TypeBase, nil, "ImpaleStacksMax") * (1 + modDB.Sum(mod.TypeBase, nil, "ImpaleAdditionalDurationChance")/100)
	if !enemyDB.HasMod(mod.TypeBase, nil, "Multiplier:ImpaleStacks") {
		enemyDB.AddMod(newMod("Multiplier:ImpaleStacks", mod.TypeBase, maxImpaleStacks, "Config", mod.Condition("Combat")))
	} else if enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ImpaleStacks") > maxImpaleStacks {
		enemyDB.ReplaceMod(newMod("Multiplier:ImpaleStacks", mod.TypeBase, maxImpaleStacks, "Config", mod.Condition("Combat")))
	}

	// Check for extra auras
	for _, v := range modDB.List(nil, "ExtraAura") {
		value, ok := v.(mod.ExtraAura)
		if !ok || value.Mod == nil {
			continue
		}
		if !value.OnlyAllies {
			inc := modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			modDB.ScaleAddList(asPerformModList(value.Mod), (1+inc/100)*more)
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
		}
		if !modDB.Flag(nil, "SelfAurasCannotAffectAllies") {
			totemModBlacklist := value.Mod.Name() == "Speed" || value.Mod.Name() == "CritMultiplier" || value.Mod.Name() == "CritChance"
			if statSetSkillFlags(env.Player.MainSkill)["totem"] && !totemModBlacklist {
				totemMod := value.Mod.Clone()
				totemModName := strings.ReplaceAll(totemMod.Name(), "Condition:", "Condition:Totem")
				if !strings.Contains(totemMod.Name(), "Condition:") {
					totemModName = "Totem" + totemMod.Name()
				}
				modDB.AddMod(totemMod.SetName(totemModName))
			}
		}
	}

	// Check for modifiers to apply to actors affected by player auras or curses
	// affectedByAura is an undefined global in PoB-PoE2, so an AffectedByAuraMod would raise there.

	// Merge keystones again to catch any that were added by buffs
	mergeKeystones(env)

	// Special handling for Dancing Dervish
	if modDB.Flag(nil, "DisableWeapons") {
		// TODO: env.player.weaponData1 = unarmedWeaponData (CalcPerform.lua:3256) is not ported.
		modDB.Conditions["Unarmed"] = true
		if env.Player.ItemList["Gloves"] == nil {
			modDB.Conditions["Unencumbered"] = true
		}
	} else if env.WeaponModList1 != nil {
		modDB.AddList(env.WeaponModList1)
	}
}

func asPerformModList(m mod.Mod) *moddb.ModList {
	l := newModList()
	l.AddMod(m)
	return l
}

// nonDamagingAilmentChillDefault is data.nonDamagingAilment.Chill.default.
//
//pob2:block:src/Modules/Data.lua:415
const nonDamagingAilmentChillDefault = 30

// performActiveSkills is perform's banner and per-active-skill pass; it returns how many minion limits
// it counted. output.HasBonechill is a boolean the output map does not hold, and Umbral Well's
// infoMessage is not kept.
//
//pob2:block:src/Modules/CalcPerform.lua:1242-1247
//pob2:block:src/Modules/CalcPerform.lua:1278-1447
func performActiveSkills(env *Environment) int {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	output := env.Player.Output
	mainSkill := env.Player.MainSkill

	// Banners
	if modDB.Flag(nil, "Condition:BannerPlanted") {
		maxValour := modDB.Sum(mod.TypeBase, nil, "MaximumValour")
		stacks := modDB.Sum(mod.TypeBase, nil, "Multiplier:ValourStacks")
		modDB.AddMod(newMod("Multiplier:BannerValour", mod.TypeBase, math.Min(stacks, maxValour), "Base"))
	}

	minionTypeCount, ammoTypeCount, grenadeTypeCount := 0.0, 0.0, 0.0
	minionCount, minionType, ammoType, grenadeType := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	maxPurpleFlameChaosGain := 0.0
	for _, activeSkill := range env.Player.ActiveSkillList {
		skillFlags := statSetSkillFlags(activeSkill)
		skillModList := activeSkill.SkillModList
		grantedEffect := activeSkill.ActiveEffect.GrantedEffect
		if skillFlags["brand"] {
			attachLimit := skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "BrandsAttachedLimit")
			attached := modDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigBrandsAttachedToEnemy")
			activeBrands := modDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigActiveBrands")
			actual := math.Min(attachLimit, attached)
			// Cap the number of active brands by the limit, which is 3 by default
			modDB.Multipliers["ActiveBrand"] = math.Min(activeBrands, modDB.Sum(mod.TypeBase, nil, "ActiveBrandLimit"))
			modDB.Multipliers["BrandsAttachedToEnemy"] = math.Max(actual, modDB.Multipliers["BrandsAttachedToEnemy"])
			enemyDB.Multipliers["BrandsAttached"] = math.Max(actual, enemyDB.Multipliers["BrandsAttached"])
		}
		if skillFlags["totem"] {
			var limit float64
			if o := mainSkill.SkillModList.Override(mainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"); o != nil {
				limit = o.Float()
			} else {
				limit = mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			}
			output["ActiveTotemLimit"] = math.Max(limit, output["ActiveTotemLimit"])
			output["TotemsSummoned"] = overrideOr(modDB, "TotemsSummoned", output["ActiveTotemLimit"])
			enemyDB.Multipliers["TotemsSummoned"] = math.Max(output["TotemsSummoned"], enemyDB.Multipliers["TotemsSummoned"])
		}
		// The actual hexes as opposed to hex related skills all have the curse flag. TotemCastsWhenNotDetached is to remove blasphemy
		// Note that this doesn't work for triggers yet, insufficient support
		// Sum returns a number, which Lua treats as true even at 0.
		if skillFlags["hex"] && skillFlags["curse"] && !activeSkill.SkillTypes["TotemCastsWhenNotDetached"] {
			hexDoom := modDB.Sum(mod.TypeBase, nil, "Multiplier:HexDoomStack")
			maxDoom := skillModList.Sum(mod.TypeBase, nil, "MaxDoom")
			doomEffect := skillModList.More(nil, "DoomEffect")
			// Update the max doom limit
			output["HexDoomLimit"] = math.Max(maxDoom, output["HexDoomLimit"])
			// Update the Hex Doom to apply
			skillModList.AddMod(newMod("CurseEffect", mod.TypeIncrease, math.Min(hexDoom, maxDoom)*doomEffect, "Doom"))
			modDB.Multipliers["HexDoom"] = math.Min(math.Max(hexDoom, modDB.Multipliers["HexDoom"]), output["HexDoomLimit"])
		}
		if grantedEffect.Name == "Vaal Lightning Trap" || grantedEffect.Name == "Shock Ground" {
			// Shock effect applies to shocked ground
			effect := skillModList.Sum(mod.TypeBase, nil, "ShockedGroundEffect") * (1 + skillModList.Sum(mod.TypeIncrease, nil, "EnemyShockMagnitude")/100)
			modDB.AddMod(newMod("ShockOverride", mod.TypeBase, effect, "Shocked Ground", mod.ActorCondition("enemy", "OnShockedGround")))
		}
		if grantedEffect.Name == "Summon Skitterbots" {
			if !skillModList.Flag(nil, "SkitterbotsCannotShock") {
				effect := gameConstant("BaseShockMagnitude") * (1 + skillModList.Sum(mod.TypeIncrease, skillSourceCfg(), "EnemyShockMagnitude")/100)
				modDB.AddMod(newMod("ShockOverride", mod.TypeBase, effect, mod.Source(grantedEffect.Name)))
				enemyDB.AddMod(newFlagMod("Condition:Shocked", mod.Source(grantedEffect.Name)))
			}
			if !skillModList.Flag(nil, "SkitterbotsCannotChill") {
				effect := nonDamagingAilmentChillDefault * (1 + skillModList.Sum(mod.TypeIncrease, skillSourceCfg(), "EnemyChillMagnitude")/100)
				modDB.AddMod(newMod("ChillOverride", mod.TypeBase, effect, mod.Source(grantedEffect.Name)))
				enemyDB.AddMod(newFlagMod("Condition:Chilled", mod.Source(grantedEffect.Name)))
			}
		} else if activeSkill.SkillTypes["ChillingArea"] || (activeSkill.SkillTypes["NonHitChill"] && !skillModList.Flag(nil, "CannotChill")) {
			effect := nonDamagingAilmentChillDefault * calcLibMod(skillModList, activeSkill.SkillCfg, "EnemyChillMagnitude")
			modDB.AddMod(newMod("ChillOverride", mod.TypeBase, effect, mod.Source(grantedEffect.Name)))
			enemyDB.AddMod(newFlagMod("Condition:Chilled", mod.Source(grantedEffect.Name)))
		}
		//pob2:block:src/Modules/CalcPerform.lua:1347-1392
		if minion := activeSkill.Minion; minion != nil && minion.MinionData != nil && minion.MinionData.Limit != "" {
			limitName := minion.MinionData.Limit
			limit := skillModList.Sum(mod.TypeBase, nil, limitName)
			output[limitName] = math.Max(limit, output[limitName])
			if o := modDB.Override(nil, limitName); o != nil {
				limit = math.Floor(o.Float())
			} else {
				limit = math.Floor(calcLibVal(skillModList, limitName, nil) * skillModList.More(activeSkill.SkillCfg, "ActiveMinionLimit"))
			}
			output[limitName] = math.Max(limit, output[limitName])
			if !minionCount[limitName] {
				env.Player.ModDB.AddMod(newMod("Multiplier:SummonedMinion", mod.TypeBase, output[limitName], "Config", mod.Condition("Combat")))
				env.Player.ModDB.AddMod(newMod("Multiplier:MinionPresenceCount", mod.TypeBase, output[limitName], "Config", mod.Condition("Combat")))
				minionCount[limitName] = true
			}
			if modDB.Flag(nil, "UmbralWell") && activeSkill.ActiveEffect.GemData != nil { //todo get buffs from file
				variantID := activeSkill.ActiveEffect.GemData.Data.VariantID
				buffValue := skillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "UmbralWellBuffValue")
				if buffValue > 0 {
					if minion.ModDB != nil {
						minion.ModDB.AddMod(mod.NewFloat("DealNoDamage", mod.TypeFlag, 1).Source("Umbral Well"))
					}
					count, _ := getActiveSkillCount(activeSkill)
					minionName := activeSkill.ActiveEffect.GemData.Data.Name
					source := mod.Source("Umbral Well - " + strconv.FormatFloat(buffValue, 'f', -1, 64) + "% per " + minionName)
					buff := mod.GlobalEffect("Buff")
					value := buffValue * count
					switch variantID {
					// The Lua passes a nil keywordFlags before the tag on the two flagged mods, so createMod keeps no tag.
					case "SkeletalWarrior":
						modDB.AddMod(newMod("Damage", mod.TypeIncrease, value, source).Flag(modFlagAttack))
					case "SkeletalSniper":
						modDB.AddMod(newMod("ProjectileSpeed", mod.TypeIncrease, value, source, buff))
					case "SkeletalCleric":
						modDB.AddMod(newMod("EnergyShieldRecharge", mod.TypeIncrease, value, source, buff))
					case "SkeletalArsonist":
						modDB.AddMod(newMod("AreaOfEffect", mod.TypeIncrease, value, source, buff))
					case "SkeletalStormMage":
						modDB.AddMod(newMod("Damage", mod.TypeIncrease, value, source).Flag(modFlagSpell))
					case "SkeletalFrostMage":
						modDB.AddMod(newMod("EnergyShield", mod.TypeIncrease, value, source, buff))
					case "SkeletalBrute":
						modDB.AddMod(newMod("EnemyHeavyStunBuildup", mod.TypeIncrease, value, source, buff))
					case "SkeletalReaver":
						modDB.AddMod(newMod("Speed", mod.TypeIncrease, value, source, buff))
						modDB.AddMod(newMod("WarcrySpeed", mod.TypeIncrease, value, source, buff))
						modDB.AddMod(newMod("TotemPlacementSpeed", mod.TypeIncrease, value, source, buff))
						modDB.AddMod(newMod("ReloadSpeed", mod.TypeIncrease, value, source, buff))
					}
				}
			}
		}
		if activeSkill.SkillTypes["Minion"] && activeSkill.SkillTypes["Persistent"] && !minionType[grantedEffect.ID] {
			minionTypeCount = minionTypeCount + 1
			minionType[grantedEffect.ID] = true
		}
		modDB.Multipliers["PersistentMinionTypes"] = minionTypeCount
		if activeSkill.SkillTypes["CrossbowAmmoSkill"] && !ammoType[grantedEffect.ID] {
			ammoTypeCount = ammoTypeCount + 1
			ammoType[grantedEffect.ID] = true
		}
		modDB.Multipliers["AmmoTypes"] = ammoTypeCount
		if activeSkill.SkillTypes["Grenade"] && !grenadeType[grantedEffect.ID] {
			grenadeTypeCount = grenadeTypeCount + 1
			grenadeType[grantedEffect.ID] = true
		}
		modDB.Multipliers["GrenadeTypes"] = grenadeTypeCount
		if activeSkill.SkillTypes["CreatesCompanion"] {
			modDB.AddMod(newFlagMod("Condition:HaveCompanion", mod.Source(grantedEffect.Name)))
		}
		if env.ModeBuffs && skillFlags["warcry"] {
			if grantedEffect.Name == "Infernal Cry" && !modDB.Flag(nil, "InfernalActive") {
				env.Player.ModDB.AddMod(newMod("InfernalExtraFireDamage", mod.TypeBase, skillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "InfernalExtraFireDamageMultiplier"), ""))
				modDB.AddMod(newFlagMod("InfernalActive", "")) // Prevents effect from applying multiple times
			}
		}
		if truthy(activeSkill.SkillData["triggeredOnDeath"]) && !skillFlags["minion"] {
			activeSkill.SkillData["triggered"] = true
			for _, value := range skillModList.Tabulate(mod.TypeIncrease, mainSkill.SkillCfg, "TriggeredDamage") {
				skillModList.AddMod(value.Mod.Clone().SetName("Damage"))
			}
			for _, value := range skillModList.Tabulate(mod.TypeMore, mainSkill.SkillCfg, "TriggeredDamage") {
				skillModList.AddMod(value.Mod.Clone().SetName("Damage"))
			}
			// Set trigger time to 1 min in ms ( == 6000 ). Technically any large value would do.
			activeSkill.SkillData["triggerTime"] = 60.0 * 1000
		}
		// Into the Breach: only purple flames get calculated
		if grantedEffect.Name == "Into the Breach" && !modDB.Flag(nil, "BreachFlameOnlyRed") && !modDB.Flag(nil, "BreachFlameOnlyBlue") {
			baseChaosGain := skillModList.Sum(mod.TypeBase, nil, "BreachFlameChaosGain")
			if baseChaosGain > 0 {
				remnantEffectMod := calcLibMod(skillModList, nil, "RemnantEffect")
				doubled := 1.0
				if modDB.Flag(nil, "BreachFlameEffectDoubled") {
					doubled = 2
				}
				chaosGainPerFlame := math.Floor(baseChaosGain * remnantEffectMod * doubled)
				maxPurpleFlameChaosGain = math.Max(maxPurpleFlameChaosGain, chaosGainPerFlame)
			}
		}
	}
	purpleFlameCount := modDB.Sum(mod.TypeBase, nil, "Multiplier:PurpleFlamesCount")
	if purpleFlameCount > 0 && maxPurpleFlameChaosGain > 0 {
		modDB.AddMod(newMod("DamageGainAsChaos", mod.TypeBase, maxPurpleFlameChaosGain*purpleFlameCount, "Into the Breach", mod.GlobalEffect("Buff").Name("Into the Breach")))
	}
	return len(minionCount)
}

func skillSourceCfg() *moddb.ListCfg {
	source := mod.Source("Skill")
	return &moddb.ListCfg{Source: &source}
}

//pob2:block:src/Modules/CalcPerform.lua:2010-2032
func performGroundAndConversions(env *Environment) {
	modDB := env.ModDB
	// Deal with Consecrated Ground
	if modDB.Flag(nil, "Condition:OnConsecratedGround") {
		effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect")/100
		modDB.AddMod(newMod("LifeRegenPercent", mod.TypeBase, 5*effect, "Consecrated Ground"))
		modDB.AddMod(newMod("CurseEffectOnSelf", mod.TypeIncrease, -50*effect, "Consecrated Ground"))
	}

	if modDB.Flag(nil, "ManaAppliesToShockEffect") {
		// Maximum Mana conversion from Lightning Mastery
		multiplier := 1.0
		if v := modDB.Max(nil, "ImprovedManaAppliesToShockEffect"); v != 0 {
			multiplier = v / 100
		}
		for _, value := range modDB.Tabulate(mod.TypeIncrease, nil, "Mana") {
			m := value.Mod
			modDB.AddMod(newMod("EnemyShockMagnitude", mod.TypeIncrease, math.Floor(m.Value().Float()*multiplier), m.GetSource(), getConvertedModTags(m, multiplier)...).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()))
		}
	}
	if modDB.Flag(nil, "ManaRegenAppliesToRageRegen") {
		// Furious Wellspring
		for _, value := range modDB.Tabulate(mod.TypeIncrease, nil, "ManaRegen") {
			m := value.Mod
			modDB.AddMod(newMod("RageRegen", mod.TypeIncrease, m.Value().Float(), m.GetSource(), m.Tags()...).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()))
		}
	}
}
