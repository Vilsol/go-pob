package calculator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

var calcsCombatCharges = []struct{ output, label string }{
	{"PowerCharges", "Power Charges"},
	{"AbsorptionCharges", "Absorption Charges"},
	{"FrenzyCharges", "Frenzy Charges"},
	{"AfflictionCharges", "Affliction Charges"},
	{"EnduranceCharges", "Endurance Charges"},
	{"BrutalCharges", "Brutal Charges"},
	{"BrineCharges", "Brine Charges"},
	{"SiphoningCharges", "Siphoning Charges"},
	{"ChallengerCharges", "Challenger Charges"},
	{"BlitzCharges", "Blitz Charges"},
}

var calcsCombatFlags = []struct{ flag, label string }{
	{"Fortify", "Fortify"}, {"Onslaught", "Onslaught"}, {"UnholyMight", "Unholy Might"}, {"ChaoticMight", "Chaotic Might"},
	{"Tailwind", "Tailwind"}, {"Adrenaline", "Adrenaline"}, {"AlchemistsGenius", "Alchemist's Genius"}, {"HerEmbrace", "Her Embrace"},
	{"AccelerationShrine", "Acceleration Shrine"}, {"BrutalShrine", "Brutal Shrine"}, {"DiamondShrine", "Diamond Shrine"},
	{"DivineShrine", "Divine Shrine"}, {"EchoingShrine", "Echoing Shrine"}, {"GloomShrine", "Gloom Shrine"},
	{"GreaterFreezingShrine", "Greater Freezing Shrine"}, {"GreaterShockingShrine", "Greater Shocking Shrine"},
	{"GreaterSkeletalShrine", "Greater Skeletal Shrine"}, {"ImpenetrableShrine", "Impenetrable Shrine"},
	{"MassiveShrine", "Massive Shrine"}, {"ReplenishingShrine", "Replenishing Shrine"}, {"ResistanceShrine", "Resistance Shrine"},
	{"ResonatingShrine", "Resonating Shrine"}, {"LesserAccelerationShrine", "Lesser Acceleration Shrine"},
	{"LesserBrutalShrine", "Lesser Brutal Shrine"}, {"LesserImpenetrableShrine", "Lesser Impenetrable Shrine"},
	{"LesserMassiveShrine", "Lesser Massive Shrine"}, {"LesserReplenishingShrine", "Lesser Replenishing Shrine"},
	{"LesserResistanceShrine", "Lesser Resistance Shrine"},
}

var calcsMinionCombatFlags = []struct{ flag, label string }{
	{"Fortify", "Fortify"}, {"Onslaught", "Onslaught"}, {"UnholyMight", "Unholy Might"}, {"ChaoticMight", "Chaotic Might"},
	{"Tailwind", "Tailwind"}, {"DiamondShrine", "Diamond Shrine"}, {"MassiveShrine", "Massive Shrine"},
}

func luaD64(v float64) string {
	return fmt.Sprintf("%d", int64(v))
}

func sortedBuffNames(m map[string]*moddb.ModList) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

//pob:block:src/Modules/Calcs.lua:634-882
func calcsBuffLists(env *Environment, multipliersUsed map[string][]mod.Mod) {
	output := env.Player.Output
	var combatList []string
	for _, c := range calcsCombatCharges {
		if output[c.output] > 0 {
			combatList = append(combatList, luaD64(output[c.output])+" "+c.label)
		}
	}
	if _, ok := multipliersUsed["InspirationCharge"]; ok {
		combatList = append(combatList, luaD64(output["InspirationCharges"])+" Inspiration Charges")
	}
	if output["GhostShrouds"] > 0 {
		combatList = append(combatList, luaD64(output["GhostShrouds"])+" Ghost Shrouds")
	}
	if output["CrabBarriers"] > 0 {
		combatList = append(combatList, luaD64(output["CrabBarriers"])+" Crab Barriers")
	}
	for _, m := range []struct{ used, output, label string }{
		{"BloodCharge", "BloodCharges", "Blood Charges"},
		{"SpiritCharge", "SpiritCharges", "Spirit Charges"},
		{"SpiritInfusion", "SpiritInfusions", "Spirit Infusions"},
	} {
		if _, ok := multipliersUsed[m.used]; ok {
			combatList = append(combatList, luaD64(output[m.output])+" "+m.label)
		}
	}
	if ms := env.Player.MainSkill; ms != nil && ms.BaseSkillModList != nil && ms.BaseSkillModList.Flag(nil, "Cruelty") {
		combatList = append(combatList, "Cruelty")
	}
	for _, f := range calcsCombatFlags {
		if env.ModDB.Flag(nil, f.flag) {
			combatList = append(combatList, f.label)
		}
	}
	buffList := sortedBuffNames(env.Buffs)
	if env.ModDB.Flag(nil, "Elusive") {
		combatList = append(combatList, "Elusive")
	}
	curseList := make([]string, 0, len(env.Debuffs)+len(env.curseSlots))
	for _, name := range sortedBuffNames(env.Debuffs) {
		entry := name
		if stackCount := env.Debuffs[name].Sum(mod.TypeBase, nil, "Multiplier:"+name+"Stack"); stackCount > 0 {
			s := ""
			if stackCount > 1 {
				s = "s"
			}
			entry = name + " (" + luaNumber(stackCount) + " stack" + s + ")"
		}
		curseList = append(curseList, entry)
	}
	for _, slot := range env.curseSlots {
		curseList = append(curseList, slot.name)
	}
	env.Player.breakdownModLists = map[string][]moddb.ModResult{"SkillBuffs": evalModLists(env.ModDB, env.Buffs, buffList)}
	if env.Enemy != nil {
		debuffs := evalModLists(env.Enemy.ModDB, env.Debuffs, sortedBuffNames(env.Debuffs))
		for _, slot := range env.curseSlots {
			if slot.modList != nil {
				debuffs = appendEvalMods(debuffs, env.Enemy.ModDB, slot.modList.Mods())
			}
		}
		env.Player.breakdownModLists["SkillDebuffs"] = debuffs
	}
	env.Player.OutputStrings["BuffList"] = strings.Join(buffList, ", ")
	env.Player.OutputStrings["CombatList"] = strings.Join(combatList, ", ")
	env.Player.OutputStrings["CurseList"] = strings.Join(curseList, ", ")

	if env.Minion == nil || env.Minion.Actor == nil {
		return
	}
	minion := env.Minion.Actor
	var minionCombat []string
	for _, c := range []struct{ output, label string }{{"PowerCharges", "Power Charges"}, {"FrenzyCharges", "Frenzy Charges"}, {"EnduranceCharges", "Endurance Charges"}} {
		if minion.Output[c.output] > 0 {
			minionCombat = append(minionCombat, luaD64(minion.Output[c.output])+" "+c.label)
		}
	}
	for _, f := range calcsMinionCombatFlags {
		if minion.ModDB.Flag(nil, f.flag) {
			minionCombat = append(minionCombat, f.label)
		}
	}
	minionBuffs := sortedBuffNames(env.MinionBuffs)
	minion.breakdownModLists = map[string][]moddb.ModResult{
		"SkillBuffs":   evalModLists(minion.ModDB, env.MinionBuffs, minionBuffs),
		"SkillDebuffs": env.Player.breakdownModLists["SkillDebuffs"],
	}
	minion.OutputStrings["BuffList"] = strings.Join(minionBuffs, ", ")
	minion.OutputStrings["CombatList"] = strings.Join(minionCombat, ", ")
	minion.OutputStrings["CurseList"] = env.Player.OutputStrings["CurseList"]
}

func evalModLists(db *moddb.ModDB, lists map[string]*moddb.ModList, names []string) []moddb.ModResult {
	out := []moddb.ModResult{}
	for _, name := range names {
		if list := lists[name]; list != nil {
			out = appendEvalMods(out, db, list.Mods())
		}
	}
	return out
}

func appendEvalMods(out []moddb.ModResult, db *moddb.ModDB, mods []mod.Mod) []moddb.ModResult {
	for _, m := range mods {
		value := db.EvalMod(m, nil)
		switch {
		case value == nil:
		case value.Type() == mod.ModValueMultiTypeFloat:
			if value.Float() != 0 {
				out = append(out, moddb.ModResult{Value: value.Float(), Mod: m})
			}
		case value.Type() != mod.ModValueMultiTypeFlag || value.Flag():
			out = append(out, moddb.ModResult{Value: 1, Mod: m})
		}
	}
	return out
}
