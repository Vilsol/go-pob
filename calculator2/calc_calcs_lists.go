package calculator2

import (
	"fmt"
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

var calcsCombatCharges = []struct{ output, label string }{
	{"PowerCharges", "Power Charges"},
	{"AbsorptionCharges", "Absorption Charges"},
	{"FrenzyCharges", "Frenzy Charges"},
	{"AfflictionCharges", "Affliction Charges"},
	{"EnduranceCharges", "Endurance Charges"},
	{"BrutalCharges", "Brutal Charges"},
	{"SiphoningCharges", "Siphoning Charges"},
	{"ChallengerCharges", "Challenger Charges"},
	{"BlitzCharges", "Blitz Charges"},
}

var calcsCombatFlags = []struct{ flag, label string }{
	{"Fortify", "Fortify"},
	{"Onslaught", "Onslaught"},
	{"UnholyMight", "Unholy Might"},
	{"ChaoticMight", "Chaotic Might"},
	{"Tailwind", "Tailwind"},
	{"Adrenaline", "Adrenaline"},
	{"AlchemistsGenius", "Alchemist's Genius"},
	{"HerEmbrace", "Her Embrace"},
	{"LesserMassiveShrine", "Lesser Massive Shrine"},
	{"LesserBrutalShrine", "Lesser Brutal Shrine"},
	{"DiamondShrine", "Diamond Shrine"},
	{"MassiveShrine", "Massive Shrine"},
}

var calcsMinionCombatFlags = []struct{ flag, label string }{
	{"Fortify", "Fortify"},
	{"Onslaught", "Onslaught"},
	{"UnholyMight", "Unholy Might"},
	{"ChaoticMight", "Chaotic Might"},
	{"Tailwind", "Tailwind"},
	{"DiamondShrine", "Diamond Shrine"},
	{"MassiveShrine", "Massive Shrine"},
}

// calcsBuffLists writes output.BuffList, CombatList and CurseList, and the minion's, and the SkillBuffs and
// SkillDebuffs breakdowns' mod lists.
//
//pob2:block:src/Modules/Calcs.lua:745-938
func calcsBuffLists(env *Environment, multipliersUsed map[string][]mod.Mod) {
	if env.buffs == nil {
		return
	}
	output := env.Player.Output
	var combatList []string
	for _, c := range calcsCombatCharges {
		if output[c.output] > 0 {
			combatList = append(combatList, fmt.Sprintf("%d %s", int64(output[c.output]), c.label))
		}
	}
	if _, ok := multipliersUsed["InspirationCharge"]; ok {
		combatList = append(combatList, fmt.Sprintf("%d Inspiration Charges", int64(output["InspirationCharges"])))
	}
	if output["GhostShrouds"] > 0 {
		combatList = append(combatList, fmt.Sprintf("%d Ghost Shrouds", int64(output["GhostShrouds"])))
	}
	if output["CrabBarriers"] > 0 {
		combatList = append(combatList, fmt.Sprintf("%d Crab Barriers", int64(output["CrabBarriers"])))
	}
	if _, ok := multipliersUsed["BloodCharge"]; ok {
		combatList = append(combatList, fmt.Sprintf("%d Blood Charges", int64(output["BloodCharges"])))
	}
	if _, ok := multipliersUsed["SpiritCharge"]; ok {
		combatList = append(combatList, fmt.Sprintf("%d Spirit Charges", int64(output["SpiritCharges"])))
	}
	if ms := env.Player.MainSkill; ms != nil && ms.BaseSkillModList != nil && ms.BaseSkillModList.Flag(nil, "Cruelty") {
		combatList = append(combatList, "Cruelty")
	}
	for _, f := range calcsCombatFlags {
		if env.ModDB.Flag(nil, f.flag) {
			combatList = append(combatList, f.label)
		}
	}
	buffList := sortedModListKeys(env.buffs.buffs)
	if env.ModDB.Flag(nil, "Elusive") {
		combatList = append(combatList, "Elusive")
	}
	var skillBuffs, skillDebuffs []BreakdownMod
	for _, name := range buffList {
		skillBuffs = appendEvalMods(skillBuffs, env.ModDB, env.buffs.buffs[name].Mods())
	}
	curseList := make([]string, 0, len(env.buffs.debuffs)+len(env.buffs.curseSlots))
	for _, name := range sortedModListKeys(env.buffs.debuffs) {
		if env.Enemy != nil {
			skillDebuffs = appendEvalMods(skillDebuffs, env.Enemy.ModDB, env.buffs.debuffs[name].Mods())
		}
		entry := name
		if stackCount := env.buffs.debuffs[name].Sum(mod.TypeBase, nil, "Multiplier:"+name+"Stack"); stackCount > 0 {
			s := ""
			if stackCount > 1 {
				s = "s"
			}
			entry = name + " (" + luaToString(stackCount) + " stack" + s + ")"
		}
		curseList = append(curseList, entry)
	}
	for _, slot := range env.buffs.curseSlots {
		curseList = append(curseList, slot.name)
		if slot.modList != nil && env.Enemy != nil {
			skillDebuffs = appendEvalMods(skillDebuffs, env.Enemy.ModDB, slot.modList.Mods())
		}
	}
	if bd := env.Player.Breakdown; bd != nil {
		bd.Entries["SkillBuffs"] = &BreakdownEntry{ModList: skillBuffs}
		bd.Entries["SkillDebuffs"] = &BreakdownEntry{ModList: skillDebuffs}
	}
	env.Player.setOutputString("BuffList", strings.Join(buffList, ", "))
	env.Player.setOutputString("CombatList", strings.Join(combatList, ", "))
	env.Player.setOutputString("CurseList", strings.Join(curseList, ", "))

	if env.Minion == nil {
		return
	}
	minion := &env.Minion.Actor
	var minionCombat []string
	for _, c := range []struct{ output, label string }{{"PowerCharges", "Power Charges"}, {"FrenzyCharges", "Frenzy Charges"}, {"EnduranceCharges", "Endurance Charges"}} {
		if minion.Output[c.output] > 0 {
			minionCombat = append(minionCombat, fmt.Sprintf("%d %s", int64(minion.Output[c.output]), c.label))
		}
	}
	for _, f := range calcsMinionCombatFlags {
		if minion.ModDB.Flag(nil, f.flag) {
			minionCombat = append(minionCombat, f.label)
		}
	}
	minionBuffList := sortedModListKeys(env.buffs.minionBuffs)
	if bd := minion.Breakdown; bd != nil {
		var minionSkillBuffs []BreakdownMod
		for _, name := range minionBuffList {
			minionSkillBuffs = appendEvalMods(minionSkillBuffs, minion.ModDB, env.buffs.minionBuffs[name].Mods())
		}
		bd.Entries["SkillBuffs"] = &BreakdownEntry{ModList: minionSkillBuffs}
		bd.Entries["SkillDebuffs"] = &BreakdownEntry{ModList: skillDebuffs}
	}
	minion.setOutputString("BuffList", strings.Join(minionBuffList, ", "))
	minion.setOutputString("CombatList", strings.Join(minionCombat, ", "))
	minion.setOutputString("CurseList", env.Player.OutputStrings["CurseList"])
}
