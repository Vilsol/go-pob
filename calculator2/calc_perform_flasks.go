package calculator2

import (
	"math"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// flaskState holds the locals CalcPerform.lua reads once at 1580-1588 and its mergeFlasks closures capture;
// the second mergeFlasks call (2948) still sees these values.
type flaskState struct {
	env                          *Environment
	effectInc, effectIncMagic    float64
	effectIncNonPlayer           float64
	effectIncMagicNonPlayer      float64
	flasksApplyToMinion          bool
	quickSilverAppliesToAllies   bool
	nonUniqueFlasksApplyToMinion bool
	flaskTotalRateInc            float64
	flaskDurInc                  float64
}

//pob2:block:src/Modules/CalcPerform.lua:1580-1588
func newFlaskState(env *Environment) *flaskState {
	modDB := env.ModDB
	fs := &flaskState{
		env:                     env,
		effectInc:               modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "FlaskEffect"),
		effectIncMagic:          modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "MagicUtilityFlaskEffect"),
		effectIncNonPlayer:      modDB.Sum(mod.TypeIncrease, nil, "FlaskEffect"),
		effectIncMagicNonPlayer: modDB.Sum(mod.TypeIncrease, nil, "MagicUtilityFlaskEffect"),
		flaskTotalRateInc:       modDB.Sum(mod.TypeIncrease, nil, "FlaskRecoveryRate"),
		flaskDurInc:             modDB.Sum(mod.TypeIncrease, nil, "FlaskDuration"),
	}
	if env.Minion != nil {
		skillCfg := env.Player.MainSkill.SkillCfg
		fs.flasksApplyToMinion = modDB.Flag(skillCfg, "FlasksApplyToMinion")
		fs.quickSilverAppliesToAllies = modDB.Flag(skillCfg, "QuickSilverAppliesToAllies")
		fs.nonUniqueFlasksApplyToMinion = env.Minion.ModDB.Flag(nil, "ParentNonUniqueFlasksAppliedToYou")
	}
	return fs
}

// Merge flask modifiers
//
//pob2:block:src/Modules/CalcPerform.lua:1605-1648
func (fs *flaskState) calcFlaskRecovery(typ string, item *Item) []mod.Mod {
	modDB := fs.env.ModDB
	out := []mod.Mod{}
	lType := strings.ToLower(typ)

	if item.FlaskData[lType+"EffectNotRemoved"] == 0 && !modDB.Flag(nil, typ+"FlaskEffectNotRemoved") {
		return out
	}

	name := mod.Source(item.Name)
	base := item.FlaskData[lType+"Base"]
	dur := item.FlaskData["duration"]
	instPerc := item.FlaskData["instantPerc"]
	flaskRecInc := modDB.Sum(mod.TypeIncrease, nil, "Flask"+typ+"Recovery")
	flaskRecMore := modDB.More(nil, "Flask"+typ+"Recovery")
	flaskRateInc := modDB.Sum(mod.TypeIncrease, nil, "Flask"+typ+"RecoveryRate")
	flaskTotal := base * (1 - instPerc/100) * (1 + flaskRecInc/100) * flaskRecMore * (1 + fs.flaskDurInc/100)
	flaskDur := dur * (1 + fs.flaskDurInc/100) / (1 + fs.flaskTotalRateInc/100) / (1 + flaskRateInc/100)

	// More life recovery while on low life is not affected by flask effect (verified ingame).
	// Since this will be multiplied by the flask effect value below we have to counteract this by removing the flask effect from the value beforehand.
	// This is also the reason why this value needs a separate multiplier and cannot just be calculated into FlaskLifeRecovery.
	lowLifeFlaskRecMore := modDB.More(nil, "FlaskLifeRecoveryLowLife")
	if lowLifeFlaskRecMore > 1 {
		if lType == "life" {
			flaskTotal *= (lowLifeFlaskRecMore-1)/(1+fs.effectInc/100) + 1
		}
	}

	if !modDB.Flag(nil, typ+"FlaskDoesNotApply") {
		out = append(out, newMod(typ+"Recovery", mod.TypeBase, flaskTotal/flaskDur, name))
	}

	if modDB.Flag(nil, typ+"FlaskAppliesToEnergyShield") {
		out = append(out, newMod("EnergyShieldRecovery", mod.TypeBase, flaskTotal/flaskDur, name))
	}

	if modDB.Flag(nil, typ+"FlaskAppliesToLife") {
		out = append(out, newMod("LifeRecovery", mod.TypeBase, flaskTotal/flaskDur, name))
	}
	wardRecoveryPercent := 0.0
	if typ == "Life" {
		wardRecoveryPercent = modDB.Sum(mod.TypeBase, nil, "LifeFlaskRecoveryAppliesToWard")
	}
	if wardRecoveryPercent > 0 {
		out = append(out, newMod("WardRecovery", mod.TypeBase, flaskTotal/flaskDur*wardRecoveryPercent/100, name))
	}

	return out
}

type flaskBuffTables struct {
	flaskBuffs, flaskBuffsNonPlayer               map[string]*moddb.ModList
	flaskBuffsPerBase, flaskBuffsPerBaseNonPlayer map[string]map[string]*moddb.ModList
}

// buffKey is a unique's title, otherwise its mods' parameters.
//
//pob2:block:src/Modules/CalcPerform.lua:1682-1690
func buffKey(item *Item, modList []mod.Mod) string {
	if item.Rarity == "UNIQUE" || item.Rarity == "RELIC" {
		return item.Title
	}
	var key strings.Builder
	for _, m := range modList {
		key.WriteString(formatModParams(m) + "&")
	}
	return key.String()
}

func (fs *flaskState) appliesToMinion(item *Item, onlyRecovery, checkNonRecoveryFlasksForMinions bool) bool {
	return (!onlyRecovery || checkNonRecoveryFlasksForMinions) && (fs.flasksApplyToMinion || fs.quickSilverAppliesToAllies || (fs.nonUniqueFlasksApplyToMinion && item.Rarity != "UNIQUE" && item.Rarity != "RELIC"))
}

//pob2:block:src/Modules/CalcPerform.lua:1651-1702
func (fs *flaskState) calcFlaskMods(item *Item, baseName string, buffModList, modList []mod.Mod, onlyMinion bool, t *flaskBuffTables, onlyRecovery, checkNonRecoveryFlasksForMinions bool) {
	flaskEffectInc := fs.effectInc + item.FlaskData["effectInc"]
	flaskEffectIncNonPlayer := fs.effectIncNonPlayer + item.FlaskData["effectInc"]
	if item.Rarity == "MAGIC" && !(item.Base.Flask.Life != 0 || item.Base.Flask.Mana != 0) {
		flaskEffectInc += fs.effectIncMagic
		flaskEffectIncNonPlayer += fs.effectIncMagicNonPlayer
	}
	effectMod := 1 + flaskEffectInc/100
	effectModNonPlayer := 1 + flaskEffectIncNonPlayer/100

	// Avert thine eyes, lest they be forever scarred
	// I have no idea how to determine which buff is applied by a given flask,
	// so utility flasks are grouped by base, unique flasks are grouped by name, and magic flasks by their modifiers
	if len(buffModList) > 0 {
		if !onlyMinion {
			srcList := newModList()
			scaleAddList(srcList, buffModList, effectMod, false)
			mergeBuff(srcList.Mods(), t.flaskBuffs, baseName)
			mergeBuff(srcList.Mods(), t.flaskBuffsPerBase[item.BaseName], baseName)
		}
		if fs.appliesToMinion(item, onlyRecovery, checkNonRecoveryFlasksForMinions) {
			srcList := newModList()
			scaleAddList(srcList, buffModList, effectModNonPlayer, false)
			mergeBuff(srcList.Mods(), t.flaskBuffsNonPlayer, baseName)
			mergeBuff(srcList.Mods(), t.flaskBuffsPerBaseNonPlayer[item.BaseName], baseName)
		}
	}

	if len(modList) > 0 {
		srcList := newModList()
		scaleAddList(srcList, modList, effectMod, false)
		key := buffKey(item, modList)
		if !onlyRecovery {
			mergeBuff(srcList.Mods(), t.flaskBuffs, key)
			mergeBuff(srcList.Mods(), t.flaskBuffsPerBase[item.BaseName], key)
		}
		if fs.appliesToMinion(item, onlyRecovery, checkNonRecoveryFlasksForMinions) {
			srcList = newModList()
			scaleAddList(srcList, modList, effectModNonPlayer, false)
			mergeBuff(srcList.Mods(), t.flaskBuffsNonPlayer, key)
			mergeBuff(srcList.Mods(), t.flaskBuffsPerBaseNonPlayer[item.BaseName], key)
		}
	}
}

// orderedBySlot is the items sorted by their earliest active slot, then by id.
//
//pob2:block:src/Modules/CalcPerform.lua:1710-1722
func orderedBySlot(env *Environment, items map[*Item]bool) []*Item {
	ordered := make([]*Item, 0, len(items))
	for item := range items {
		ordered = append(ordered, item)
	}
	sort.SliceStable(ordered, func(a, b int) bool {
		posA, posB := env.ItemSlotIndex[ordered[a]], env.ItemSlotIndex[ordered[b]]
		if posA != posB {
			return posA < posB
		}
		return ordered[a].ID < ordered[b].ID
	})
	return ordered
}

//pob2:block:src/Modules/CalcPerform.lua:1703-1777
func (fs *flaskState) mergeFlasks(flasks map[*Item]bool, onlyRecovery, checkNonRecoveryFlasksForMinions bool) {
	env := fs.env
	modDB := env.ModDB
	t := &flaskBuffTables{
		flaskBuffs:                 map[string]*moddb.ModList{},
		flaskBuffsNonPlayer:        map[string]*moddb.ModList{},
		flaskBuffsPerBase:          map[string]map[string]*moddb.ModList{},
		flaskBuffsPerBaseNonPlayer: map[string]map[string]*moddb.ModList{},
	}
	flaskConditions := map[string]bool{}

	for _, item := range orderedBySlot(env, flasks) {
		if t.flaskBuffsPerBase[item.BaseName] == nil {
			t.flaskBuffsPerBase[item.BaseName] = map[string]*moddb.ModList{}
		}
		if t.flaskBuffsPerBaseNonPlayer[item.BaseName] == nil {
			t.flaskBuffsPerBaseNonPlayer[item.BaseName] = map[string]*moddb.ModList{}
		}
		flaskConditions["UsingFlask"] = true
		flaskConditions["Using"+whitespaceRe.ReplaceAllString(item.BaseName, "")] = true
		if item.Base.Flask.Life != 0 && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
			flaskConditions["UsingLifeFlask"] = true
		}
		if item.Base.Flask.Mana != 0 {
			flaskConditions["UsingManaFlask"] = true
		}

		if onlyRecovery {
			if item.Base.Flask.Life != 0 && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
				fs.calcFlaskMods(item, "LifeFlask", fs.calcFlaskRecovery("Life", item), nil, false, t, onlyRecovery, checkNonRecoveryFlasksForMinions)
			}
			if item.Base.Flask.Mana != 0 {
				fs.calcFlaskMods(item, "ManaFlask", fs.calcFlaskRecovery("Mana", item), nil, false, t, onlyRecovery, checkNonRecoveryFlasksForMinions)
			}
			if checkNonRecoveryFlasksForMinions {
				fs.calcFlaskMods(item, item.BaseName, item.BuffModList, item.ModList, true, t, onlyRecovery, checkNonRecoveryFlasksForMinions)
			}
		} else {
			fs.calcFlaskMods(item, item.BaseName, item.BuffModList, item.ModList, false, t, onlyRecovery, checkNonRecoveryFlasksForMinions)
		}
	}
	if !modDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
		for flaskCond, status := range flaskConditions {
			modDB.Conditions[flaskCond] = status
		}
		for _, k := range sortedModListKeys(t.flaskBuffs) {
			modDB.AddList(t.flaskBuffs[k])
		}
	}
	if env.Minion != nil {
		if fs.flasksApplyToMinion || fs.nonUniqueFlasksApplyToMinion { // nonUniqueAlreadyFiltered
			minionModDB := env.Minion.ModDB
			for flaskCond, status := range flaskConditions {
				minionModDB.Conditions[flaskCond] = status
			}
			for _, k := range sortedModListKeys(t.flaskBuffsNonPlayer) {
				minionModDB.AddList(t.flaskBuffsNonPlayer[k])
			}
		} else { // Not all flasks apply to minions. Check if some flasks need to be selectively applied
			if perBase := t.flaskBuffsPerBaseNonPlayer["Quicksilver Flask"]; fs.quickSilverAppliesToAllies && perBase != nil {
				minionModDB := env.Minion.ModDB
				minionModDB.Conditions["UsingQuicksilverFlask"] = flaskConditions["UsingQuicksilverFlask"]
				minionModDB.Conditions["UsingFlask"] = flaskConditions["UsingFlask"]
				for _, k := range sortedModListKeys(perBase) {
					minionModDB.AddList(perBase[k])
				}
			}
		}
	}
}

type charmState struct {
	env                       *Environment
	effectInc, effectIncMagic float64
	charmLimit                float64
}

//pob2:block:src/Modules/CalcPerform.lua:1790-1820
func (cs *charmState) calcCharmMods(item *Item, baseName string, buffModList, modList []mod.Mod, charmBuffs map[string]*moddb.ModList, charmBuffsPerBase map[string]map[string]*moddb.ModList) {
	charmEffectInc := cs.effectInc + item.CharmData["effectInc"]
	if item.Rarity == "MAGIC" {
		charmEffectInc += cs.effectIncMagic
	}
	effectMod := 1 + charmEffectInc/100

	// same deal as flasks, go look at the comment there
	if len(buffModList) > 0 {
		srcList := newModList()
		scaleAddList(srcList, buffModList, effectMod, false)
		mergeBuff(srcList.Mods(), charmBuffs, baseName)
		mergeBuff(srcList.Mods(), charmBuffsPerBase[item.BaseName], baseName)
	}

	if len(modList) > 0 {
		srcList := newModList()
		scaleAddList(srcList, modList, effectMod, false)
		key := buffKey(item, modList)
		mergeBuff(srcList.Mods(), charmBuffs, key)
		mergeBuff(srcList.Mods(), charmBuffsPerBase[item.BaseName], key)
	}
}

//pob2:block:src/Modules/CalcPerform.lua:1821-1856
func (cs *charmState) mergeCharms(charms map[*Item]bool) {
	env := cs.env
	modDB := env.ModDB
	charmBuffs := map[string]*moddb.ModList{}
	charmConditions := map[string]bool{}
	charmBuffsPerBase := map[string]map[string]*moddb.ModList{}

	for _, item := range orderedBySlot(env, charms) {
		if cs.charmLimit <= 0 {
			break
		}
		cs.charmLimit--
		if charmBuffsPerBase[item.BaseName] == nil {
			charmBuffsPerBase[item.BaseName] = map[string]*moddb.ModList{}
		}
		charmConditions["UsingCharm"] = true
		charmConditions["Using"+whitespaceRe.ReplaceAllString(item.BaseName, "")] = true
		cs.calcCharmMods(item, item.BaseName, item.BuffModList, item.ModList, charmBuffs, charmBuffsPerBase)
	}
	env.Player.Output["EmptyCharms"] = cs.charmLimit
	for charmCond, status := range charmConditions {
		modDB.Conditions[charmCond] = status
	}
	for _, k := range sortedModListKeys(charmBuffs) {
		modDB.AddList(charmBuffs[k])
	}
}

//pob2:block:src/Modules/CalcPerform.lua:1779-1781
func newCharmState(env *Environment) *charmState {
	modDB := env.ModDB
	charmLimit := modDB.Sum(mod.TypeBase, nil, "CharmLimit")
	if o := modDB.Override(nil, "CharmLimit"); o != nil {
		charmLimit = o.Float()
	}
	return &charmState{
		env:            env,
		effectInc:      modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "CharmEffect"),
		effectIncMagic: modDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Actor: "player"}, "MagicCharmEffect"),
		charmLimit:     math.Min(charmLimit, 3),
	}
}
