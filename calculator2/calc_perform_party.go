package calculator2

import (
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

// applyAllyBuffs adds the party tab's other effects, auras, aura debuffs, warcries and links. PoB walks each
// list in pairs() order; names are walked sorted here.
//
//pob2:block:src/Modules/CalcPerform.lua:2824-2943
func applyAllyBuffs(env *Environment, st *buffState) {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	buffs, debuffs, minionBuffs := st.buffs, st.debuffs, st.minionBuffs
	allyBuffs := env.partyMembers().Aura
	minionAffected := func() bool {
		return env.Minion != nil && !env.Minion.ModDB.Flag(nil, "HiddenMonster")
	}

	//pob2:block:src/Modules/CalcPerform.lua:2824-2841
	if others := allyBuffs.byType["otherEffects"]; others != nil {
		for _, buffName := range sortedPartyBuffNames(others.buffs) {
			buff := others.buffs[buffName]
			buffNameCompressed := strings.ReplaceAll(buffName, " ", "")
			modDB.Conditions["AffectedBy"+buffNameCompressed] = true
			inc := modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			srcList := newModList()
			scaleAddList(srcList, buff.modList.Mods(), (buff.effectMult+inc)/100*more, false)
			mergeBuff(srcList.Mods(), buffs, buffName)
			if minionAffected() {
				env.Minion.ModDB.Conditions["AffectedBy"+buffNameCompressed] = true
				inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				srcList := newModList()
				scaleAddList(srcList, buff.modList.Mods(), (buff.effectMult+inc)/100*more, false)
				mergeBuff(srcList.Mods(), minionBuffs, buffName)
			}
		}
	}
	//pob2:block:src/Modules/CalcPerform.lua:2842-2882
	if auras := allyBuffs.byType["Aura"]; auras != nil {
		for _, vaal := range []bool{false, true} {
			list := auras.buffs
			if vaal {
				list = auras.vaal
			}
			for _, auraName := range sortedPartyBuffNames(list) {
				aura := list[auraName]
				auraNameCompressed := strings.ReplaceAll(auraName, " ", "")
				if !modDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !modDB.Conditions["AffectedBy"+auraNameCompressed] {
					modDB.Conditions["AffectedByAura"] = true
					if vaal {
						modDB.Conditions["AffectedBy"+strings.ReplaceAll(auraName[min(5, len(auraName)):], " ", "")] = true
					}
					modDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := newModList()
					scaleAddList(srcList, aura.modList.Mods(), aura.effectMult/100, false)
					mergeBuff(srcList.Mods(), buffs, auraName)
				}
				if minionAffected() && !env.Minion.ModDB.Conditions["AffectedBy"+auraNameCompressed] {
					env.Minion.ModDB.Conditions["AffectedByAura"] = true
					env.Minion.ModDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := newModList()
					scaleAddList(srcList, aura.modList.Mods(), aura.effectMult/100, false)
					mergeBuff(srcList.Mods(), minionBuffs, auraName)
				}
			}
		}
	}
	//pob2:block:src/Modules/CalcPerform.lua:2883-2908
	if auraDebuffs := allyBuffs.byType["AuraDebuff"]; auraDebuffs != nil && env.ModeEffective {
		for _, list := range []map[string]*partyBuff{auraDebuffs.buffs, auraDebuffs.vaal} {
			for _, auraName := range sortedPartyBuffNames(list) {
				aura := list[auraName]
				auraNameCompressed := strings.ReplaceAll(auraName, " ", "")
				if !enemyDB.Conditions["AffectedBy"+auraNameCompressed] {
					enemyDB.Conditions["AffectedBy"+auraNameCompressed] = true
					modDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := newModList()
					scaleAddList(srcList, aura.modList.Mods(), aura.effectMult/100, false)
					mergeBuff(srcList.Mods(), debuffs, auraName)
				}
			}
		}
	}
	//pob2:block:src/Modules/CalcPerform.lua:2909-2931
	if warcries := env.partyMembers().Warcry.byType["Warcry"]; warcries != nil {
		for _, warcryName := range sortedPartyBuffNames(warcries.buffs) {
			warcry := warcries.buffs[warcryName]
			warcryNameCompressed := strings.ReplaceAll(warcryName, " ", "")
			scaled := func() []mod.Mod {
				srcList := newModList()
				for _, warcryBuff := range warcry.modList.Mods() {
					bonus, ok := warcry.warcryPowerBonus[warcryBuff]
					if !ok {
						bonus = 1
					}
					scaleAddList(srcList, []mod.Mod{warcryBuff}, warcry.effectMult/100*bonus, false)
				}
				return srcList.Mods()
			}
			if !modDB.Conditions["AffectedBy"+warcryNameCompressed] {
				modDB.Conditions["AffectedByWarcry"] = true
				modDB.Conditions["AffectedBy"+warcryNameCompressed] = true
				mergeBuff(scaled(), buffs, warcryName)
			}
			if minionAffected() && !env.Minion.ModDB.Conditions["AffectedBy"+warcryNameCompressed] {
				env.Minion.ModDB.Conditions["AffectedByWarcry"] = true
				env.Minion.ModDB.Conditions["AffectedBy"+warcryNameCompressed] = true
				mergeBuff(scaled(), minionBuffs, warcryName)
			}
		}
	}
	//pob2:block:src/Modules/CalcPerform.lua:2932-2943
	if links := env.partyMembers().Link.byType["Link"]; links != nil {
		for _, linkName := range sortedPartyBuffNames(links.buffs) {
			link := links.buffs[linkName]
			linkNameCompressed := strings.ReplaceAll(linkName, " ", "")
			if !modDB.Conditions["AffectedBy"+linkNameCompressed] {
				modDB.Conditions["AffectedByLink"] = true
				modDB.Conditions["AffectedBy"+linkNameCompressed] = true
				srcList := newModList()
				scaleAddList(srcList, link.modList.Mods(), link.effectMult/100, false)
				mergeBuff(srcList.Mods(), buffs, linkName)
			}
		}
	}
}

// allyCurses is the party tab's curses, each a curse of priority 0. PoB walks them in pairs() order; names
// are walked sorted here.
//
//pob2:block:src/Modules/CalcPerform.lua:3001-3022
func allyCurses(env *Environment) []*performCurse {
	enemyDB := env.Enemy.ModDB
	allyPartyCurses := env.partyMembers().Curse.byType["Curse"]
	if allyPartyCurses == nil {
		return nil
	}
	var out []*performCurse
	for _, curseName := range sortedPartyBuffNames(allyPartyCurses.buffs) {
		curse := allyPartyCurses.buffs[curseName]
		newCurse := &performCurse{name: curseName, priority: 0, modList: newModList()}
		mult := curse.effectMult / 100
		if curse.isMark {
			newCurse.isMark = true
		} else {
			mult = mult * enemyDB.More(nil, "CurseEffectOnSelf")
		}
		scaleAddList(newCurse.modList, curse.modList.Mods(), mult, true)
		out = append(out, newCurse)
	}
	return out
}

// applyAllyExtraAuras adds the party tab's extra auras to the player and the minion. PoB walks them in pairs()
// order; names are walked sorted here.
//
//pob2:block:src/Modules/CalcPerform.lua:3230-3242
func applyAllyExtraAuras(env *Environment) {
	extra := env.partyMembers().Aura.byType["extraAura"]
	if extra == nil {
		return
	}
	modDB := env.ModDB
	for _, name := range sortedPartyBuffNames(extra.buffs) {
		modList := extra.buffs[name].modList
		inc := modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
		more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
		modDB.ScaleAddList(modList, (1+inc/100)*more)
		if env.Minion != nil && !env.Minion.ModDB.Flag(nil, "HiddenMonster") {
			inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			env.Minion.ModDB.ScaleAddList(modList, (1+inc/100)*more)
		}
	}
}
