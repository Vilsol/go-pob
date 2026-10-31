package calculator2

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// grantedEffectLevel is activeEffect.grantedEffectLevel: a copy of grantedEffect.levels[level] with the
// stat set's levels[level] written over it.
type grantedEffectLevel struct {
	Extra             map[string]float64
	Cost              map[string]float64
	Values            []float64
	StatInterpolation []float64
}

// gemPropertyInfo is one effect.gemPropertyInfo entry: a GemProperty mod applyGemMods applied.
type gemPropertyInfo struct {
	Source mod.Source
	Tag    mod.Tag
	Value  mod.GemProperty
}

// supportedGemProperty is one Tabulate row getSourceGemPropertyInfo keeps.
type supportedGemProperty struct {
	Source mod.Source
	Tag    mod.Tag
	Value  mod.SupportedGemProperty
}

// skillBuff is one activeSkill.buffList entry.
type skillBuff struct {
	Type, Name       string
	Cond, EnemyCond  string
	StackVar         string
	StackLimitVar    string
	ModList          []mod.Mod
	StackLimit       float64
	AllowTotemBuff   bool
	ApplyNotPlayer   bool
	ApplyMinions     bool
	ApplyAllies      bool
	ActiveSkillBuff  bool
	HasStackLimit    bool
	HasAllowTotemBuf bool
}

// skillGemCfg is cfg.skillGem: data.gems[id] as ModStore and calcLib read it.
type skillGemCfg struct {
	gem *poe2skills.Gem
}

func (g skillGemCfg) GemName() string          { return g.gem.Data.Name }
func (g skillGemCfg) GemTags() map[string]bool { return gemTags(g.gem.Data) }

func (g skillGemCfg) StatSetBaseFlags() map[string]bool {
	if g.gem.GrantedEffect == nil || len(g.gem.GrantedEffect.StatSets) == 0 {
		return nil
	}
	return g.gem.GrantedEffect.StatSets[0].BaseFlags
}

// skillGrantedEffectCfg is cfg.skillGrantedEffect.
type skillGrantedEffectCfg struct {
	ge *poe2skills.GrantedEffect
}

func (g skillGrantedEffectCfg) ID() string { return g.ge.ID }

func cfgSkillTypes(skillTypes map[string]bool) map[data.SkillType]bool {
	out := make(map[data.SkillType]bool, len(skillTypes))
	for k, v := range skillTypes {
		out[data.SkillType(k)] = v
	}
	return out
}

//pob2:block:src/Data/Global.lua:231-268
var modFlagByName = map[string]mod.MFlag{
	"Attack": ModFlagAttack, "Spell": ModFlagSpell, "Hit": ModFlagHit, "Dot": ModFlagDot, "Cast": ModFlagCast,
	"Thorns": ModFlagThorns, "Melee": ModFlagMelee, "Area": ModFlagArea, "Projectile": ModFlagProjectile,
	"SourceMask": ModFlagSourceMask, "Ailment": ModFlagAilment, "MeleeHit": ModFlagMeleeHit, "Weapon": ModFlagWeapon,
	"Axe": ModFlagAxe, "Bow": ModFlagBow, "Claw": ModFlagClaw, "Dagger": ModFlagDagger, "Mace": ModFlagMace,
	"Staff": ModFlagStaff, "Sword": ModFlagSword, "Wand": ModFlagWand, "Unarmed": ModFlagUnarmed,
	"Fishing": ModFlagFishing, "Crossbow": ModFlagCrossbow, "Flail": ModFlagFlail, "Spear": ModFlagSpear,
	"Warstaff": ModFlagWarstaff, "Talisman": ModFlagTalisman, "WeaponMelee": ModFlagWeaponMelee,
	"WeaponRanged": ModFlagWeaponRanged, "Weapon1H": ModFlagWeapon1H, "Weapon2H": ModFlagWeapon2H,
	"WeaponMask": ModFlagWeaponMask,
}

// skillModTable is one of a skill's mod tables with mergeLevelMod's value written in; the copy is shallow,
// as copyTable(mod, true) is.
//
//pob2:block:src/Modules/CalcActiveSkill.lua:33-44
func skillModTable(m poe2skills.Mod, value float64, source string) poe2skills.Mod {
	newMod := make(poe2skills.Mod, len(m)+1)
	for k, v := range m {
		newMod[k] = v
	}
	if source != "" {
		newMod["source"] = source
	}
	if v, ok := newMod["value"].(map[string]any); ok {
		newValue := make(map[string]any, len(v))
		for k, x := range v {
			newValue[k] = x
		}
		if inner, ok := newValue["mod"].(map[string]any); ok {
			newInner := make(map[string]any, len(inner))
			for k, x := range inner {
				newInner[k] = x
			}
			newInner["value"] = value
			newValue["mod"] = newInner
		} else {
			newValue["value"] = value
		}
		newMod["value"] = newValue
	} else {
		newMod["value"] = value
	}
	return newMod
}

// addSkillMod is modList:AddMod for a skill's mod table. A table the mod package cannot express, or a
// group's nameless tag, adds nothing.
func addSkillMod(modList *moddb.ModList, m poe2skills.Mod) {
	if _, ok := m["name"].(string); !ok {
		return
	}
	converted, err := cacheMod(m)
	if err != nil {
		return
	}
	modList.AddMod(converted)
}

type mergeLevelKey struct {
	mod    unsafe.Pointer
	source string
	value  float64
}

// mergeLevelCache holds the converted mod per skill mod table and value, shared by every build as PoB's is.
var mergeLevelCache sync.Map

// mergeLevelMod merges a level modifier with the given mod list.
//
//pob2:src/Modules/CalcActiveSkill.lua:21-50
func mergeLevelMod(modList *moddb.ModList, m poe2skills.Mod, value float64, source string) {
	if _, ok := m["name"].(string); !ok {
		return
	}
	key := mergeLevelKey{mod: reflect.ValueOf(m).UnsafePointer(), source: source, value: value}
	if cached, ok := mergeLevelCache.Load(key); ok {
		if cached != nil {
			modList.AddMod(cached.(mod.Mod))
		}
		return
	}
	converted, err := cacheMod(skillModTable(m, value, source))
	if err != nil {
		mergeLevelCache.Store(key, nil)
		return
	}
	mergeLevelCache.Store(key, converted)
	modList.AddMod(converted)
}

func skillModTags(m poe2skills.Mod) []map[string]any {
	raw, _ := m["tags"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, t := range raw {
		if tag, ok := t.(map[string]any); ok {
			out = append(out, tag)
		}
	}
	return out
}

// checkForScalarMultiplier allows Multiplier mods to be scaled by sources of the multipliedVariableEffect,
// e.g. var = RemovablePowerCharges, scalar = ConsumedPowerChargeEffect. A group carries no scalar of its
// own and its members carry theirs in their tags, so a group scales by 1.
//
//pob2:src/Modules/CalcActiveSkill.lua:54-67
func checkForScalarMultiplier(m poe2skills.Mod, modList *moddb.ModList) float64 {
	scale := 0.0
	if m != nil {
		for _, config := range skillModTags(m) {
			if scalar, ok := config["scalar"].(string); ok {
				scale = modList.Sum(mod.TypeBase, nil, "Multiplier:"+scalar)
				break
			}
		}
	}
	return 1 + scale/100
}

//pob2:src/Modules/CalcActiveSkill.lua:69-79
func isGlobalEffect(item *poe2skills.StatMapItem) bool {
	mods := item.Group
	if item.Mod != nil {
		mods = []poe2skills.Mod{item.Mod}
	}
	for _, m := range mods {
		if isGlobalEffectMod(m) {
			return true
		}
	}
	return false
}

func isGlobalEffectMod(m poe2skills.Mod) bool {
	for _, tag := range skillModTags(m) {
		if tag["type"] == "GlobalEffect" {
			return true
		}
	}
	return false
}

// statMapScalars is `map.value or statValue * (map.mult or 1) * scalar / (map.div or 1) + (map.base or 0)`.
func statMapScalars(statValue, scalar, value, mult, div, base float64, hasValue, hasMult, hasDiv, hasBase bool) float64 {
	if hasValue {
		return value
	}
	if !hasMult {
		mult = 1
	}
	if !hasDiv {
		div = 1
	}
	if !hasBase {
		base = 0
	}
	return statValue*mult*scalar/div + base
}

//pob2:src/Modules/CalcActiveSkill.lua:80-113
func mergeStatSet(set *poe2skills.StatSet, onlyGlobals bool, skillEffect *GemEffect, grantedEffect *poe2skills.GrantedEffect, env *Environment, extraStats []mod.ExtraSkillStat, modList *moddb.ModList, selectedGlobalStats map[string]bool) {
	stats := buildSkillInstanceStats(skillEffect, grantedEffect, set, env.useAltGemQualityStats)
	for _, stat := range extraStats {
		if v, ok := stat.Value.(float64); ok {
			stats.add(stat.Key, v)
		}
	}
	source := "Skill:" + grantedEffect.ID
	for _, stat := range stats.order {
		statValue := stats.values[stat]
		statMap, ok := set.StatMap(stat)
		if !ok {
			continue
		}
		// Some mods need different scalars for different stats, but the same value.  Putting them in a group allows this
		for i := range statMap.Items {
			modOrGroup := &statMap.Items[i]
			isGlobal := isGlobalEffect(modOrGroup)
			if isGlobal && !onlyGlobals {
				selectedGlobalStats[stat] = true
			}
			if (!onlyGlobals || isGlobal) && !(onlyGlobals && selectedGlobalStats[stat]) {
				scalar := checkForScalarMultiplier(modOrGroup.Mod, modList)
				// Found a mod, since all mods have names
				if modOrGroup.Mod != nil {
					mergeLevelMod(modList, modOrGroup.Mod, statMapScalars(statValue, scalar, statMap.Value, statMap.Mult, statMap.Div, statMap.Base,
						statMap.HasValue, statMap.HasMult, statMap.HasDiv, statMap.HasBase), source)
				} else {
					for _, m := range modOrGroup.Group {
						scalar := checkForScalarMultiplier(m, modList)
						mergeLevelMod(modList, m, statMapScalars(statValue, scalar, modOrGroup.Value, modOrGroup.Mult, modOrGroup.Div, modOrGroup.Base,
							modOrGroup.HasValue, modOrGroup.HasMult, modOrGroup.HasDiv, modOrGroup.HasBase), source)
					}
				}
			}
		}
	}
}

// mergeSkillInstanceMods merges skill effect modifiers with the given mod list. If a stat set is
// provided, it merges it and the global effects from the other stat sets.
//
//pob2:src/Modules/CalcActiveSkill.lua:114-140
func mergeSkillInstanceMods(env *Environment, modList *moddb.ModList, skillEffect *GemEffect, statSet *poe2skills.StatSet, extraStats []mod.ExtraSkillStat) {
	skillEffect.Level = validateGemLevel(skillEffect.GrantedEffect, skillEffect.GemData, skillEffect.Level)
	grantedEffect := skillEffect.GrantedEffect
	// Verify that statSet provided is from skillEffect
	if statSet != nil {
		found := false
		for _, s := range grantedEffect.StatSets {
			if s == statSet {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}
	selectedGlobalStats := map[string]bool{}
	sets := grantedEffect.StatSets
	if statSet != nil {
		sets = []*poe2skills.StatSet{statSet}
	}
	for _, set := range sets {
		mergeStatSet(set, false, skillEffect, grantedEffect, env, extraStats, modList, selectedGlobalStats)
		for _, m := range set.BaseMods {
			addSkillMod(modList, m)
		}
	}
	if statSet != nil {
		for _, set := range grantedEffect.StatSets {
			if set != statSet {
				mergeStatSet(set, true, skillEffect, grantedEffect, env, extraStats, modList, selectedGlobalStats)
				for _, baseMod := range set.BaseMods {
					if isGlobalEffectMod(baseMod) {
						addSkillMod(modList, baseMod)
					}
				}
			}
		}
	}
}

// listWithMods is List zipped with Tabulate: the evaluated value of each LIST mod beside the mod itself.
// Tabulate keeps every non-numeric row, so the two walk the same mods in the same order.
func listWithMods(modList *moddb.ModList, cfg *moddb.ListCfg, name string) ([]any, []mod.Mod) {
	values := modList.List(cfg, name)
	rows := modList.Tabulate(mod.TypeList, cfg, name)
	mods := make([]mod.Mod, len(values))
	if len(rows) == len(values) {
		for i, r := range rows {
			mods[i] = r.Mod
		}
	}
	return values, mods
}

//pob2:src/Modules/CalcActiveSkill.lua:236-263
func getSourceGemPropertyInfo(env *Environment, activeSkill *ActiveSkill) []supportedGemProperty {
	activeEffect := activeSkill.ActiveEffect
	sourceGem := activeEffect.SrcInstance
	if sourceGem == nil || activeEffect.GemData == nil || gemTags(activeEffect.GemData.Data)["support"] {
		return nil
	}

	if env.sourceGemPropertyInfo == nil {
		env.sourceGemPropertyInfo = map[*GemInstance][]supportedGemProperty{}
	}
	if info, ok := env.sourceGemPropertyInfo[sourceGem]; ok {
		return info
	}
	modList := newModList()
	modList.Parent = activeSkill.Actor.ModDB
	supportCount := 0.0
	for _, supportEffect := range activeSkill.SupportList {
		if supportEffect.IsSupporting != nil && supportEffect.IsSupporting[sourceGem] {
			mergeSkillInstanceMods(env, modList, supportEffect, nil, nil)
			if !supportEffect.GrantedEffect.Hidden {
				supportCount++
			}
		}
	}
	modList.AddMod(newMod("Multiplier:SupportCount", mod.TypeBase, supportCount, "Support Count"))
	cfg := &moddb.ListCfg{
		SkillName: activeEffect.GemData.Data.Name,
		GemData:   skillGemCfg{activeEffect.GemData},
		SlotName:  activeSkill.SlotName,
	}
	values, mods := listWithMods(modList, cfg, "SupportedGemProperty")
	info := make([]supportedGemProperty, 0, len(values))
	for i, v := range values {
		var p mod.SupportedGemProperty
		switch x := v.(type) {
		case mod.SupportedGemProperty:
			p = x
		case *mod.SupportedGemProperty:
			p = *x
		default:
			continue
		}
		row := supportedGemProperty{Value: p}
		if m := mods[i]; m != nil {
			row.Source = m.GetSource()
			if tags := m.Tags(); len(tags) > 0 && tags[0].Type() == mod.TypeGemTag {
				row.Tag = tags[0]
			}
		}
		info = append(info, row)
	}
	env.sourceGemPropertyInfo[sourceGem] = info
	return info
}

// weaponDataView is the part of actor.weaponData1/2 getWeaponFlags reads. cannotUseGemTag is not read:
// the parser keeps no value for it.
type weaponDataView struct {
	asThoughUsing map[string]bool
	typ           string
	countsAsAll1H bool
}

func actorWeaponData(actor *Actor, i int) weaponDataView {
	w := weaponDataView{typ: actor.WeaponData1Type, asThoughUsing: actor.WeaponData1AsThoughUsing}
	if i == 2 {
		w = weaponDataView{typ: actor.WeaponData2Type, asThoughUsing: actor.WeaponData2AsThoughUsing}
	}
	slot := "Weapon " + strconv.Itoa(i)
	if item := actor.ItemList[slot]; item != nil && item.WeaponData[i] != nil {
		_, w.countsAsAll1H = item.WeaponData[i]["countsAsAll1H"]
	}
	return w
}

// checkAsThoughWeaponTypes checks for an "asThoughUsing..." weaponTypes match, which is mechanically
// different from "countAs...".
//
//pob2:src/Modules/CalcActiveSkill.lua:300-316
func checkAsThoughWeaponTypes(weaponData weaponDataView, weaponTypes []map[string]bool) bool {
	if weaponData.asThoughUsing == nil {
		return false
	}
	// check if any 'usingKey' for which 'usingValue = true' is also true in weaponTypes
	for usingKey, usingValue := range weaponData.asThoughUsing {
		for _, types := range weaponTypes {
			if usingValue && types[usingKey] {
				return true
			}
		}
	}
	return false
}

// getWeaponFlags gets the weapon flags and info for the given weapon; ok is false where Lua returns nil
// flags, hasInfo where it returns info.
//
//pob2:src/Modules/CalcActiveSkill.lua:318-354
func getWeaponFlags(weaponData weaponDataView, weaponTypes []map[string]bool) (mod.MFlag, bool, weaponType, bool) {
	info, hasInfo := weaponTypeInfo[weaponData.typ]
	if !hasInfo {
		return 0, false, info, false
	}
	for _, types := range weaponTypes {
		if !types[weaponData.typ] &&
			(!weaponData.countsAsAll1H || !(types["Claw"] || types["Dagger"] || types["One Hand Axe"] || types["One Hand Mace"] || types["One Hand Sword"] ||
				types["Spear"])) && !(weaponData.asThoughUsing != nil && checkAsThoughWeaponTypes(weaponData, weaponTypes)) {
			return 0, false, info, true
		}
	}
	flags := modFlagByName[info.flag]
	if weaponData.countsAsAll1H {
		flags = ModFlagAxe | ModFlagClaw | ModFlagDagger | ModFlagMace | ModFlagSword | ModFlagSpear
	}
	if weaponData.typ != "None" {
		flags |= ModFlagWeapon
		if info.oneHand {
			flags |= ModFlagWeapon1H
		} else {
			flags |= ModFlagWeapon2H
		}
		if info.melee {
			flags |= ModFlagWeaponMelee
		} else {
			flags |= ModFlagWeaponRanged
		}
	}
	return flags, true, info, true
}

// totemBase is skillData.totemBase.
type totemBase struct {
	grantedEffect *poe2skills.GrantedEffect
	gemData       *poe2skills.Gem
	skillLevel    int
}

// getTotemBaseStats gets the stats from the totem base skill in case of separate active skills or skills
// that receive totem status via supports. Lua raises an error for a totem skill that neither summons
// totems nor is used by one; that case returns no base here.
//
//pob2:src/Modules/CalcActiveSkill.lua:356-393
func getTotemBaseStats(activeSkill *ActiveSkill) totemBase {
	var base totemBase
	switch {
	case activeSkill.SkillTypes["SummonsTotem"]:
		// Skill that summons totems already has stats on activeEffect
		base.grantedEffect = activeSkill.ActiveEffect.GrantedEffect
		base.gemData = activeSkill.ActiveEffect.GemData
		base.skillLevel = activeSkill.ActiveEffect.Level
	case activeSkill.SkillTypes["UsedByTotem"]:
		if activeSkill.ActiveEffect.GrantedEffect.SkillTypes["UsedByTotem"] {
			// is totem skill by default
			if activeSkill.ActiveEffect.GemData != nil {
				base.grantedEffect = activeSkill.ActiveEffect.GemData.GrantedEffect
			}
			base.gemData = activeSkill.ActiveEffect.GemData
			base.skillLevel = activeSkill.ActiveEffect.Level
		} else if activeSkill.SupportList != nil {
			// skill is receives totem status via support
			for _, support := range activeSkill.SupportList {
				if support.GrantedEffect.AddSkillTypes != nil && !support.Superseded && support.IsSupporting[activeSkill.ActiveEffect.SrcInstance] {
					for _, skillType := range support.GrantedEffect.AddSkillTypes {
						if skillType == "UsedByTotem" {
							if support.GemData != nil {
								base.grantedEffect = support.GemData.GrantedEffect
							}
							base.gemData = support.GemData
							break
						}
					}
				}
				if base.gemData != nil || base.grantedEffect != nil {
					base.skillLevel = support.Level
					break
				}
			}
		}
	}
	return base
}

// applyExtraEmpowerMods applies additional modifiers to skills with the "Empowered" flag: the
// "ExtraEmpowerMod" mods that match the conditions set by the empowering effect. No PoE2 mod line
// produces ExtraEmpowerMod, so only the lookup is ported.
//
//pob2:src/Modules/CalcActiveSkill.lua:395-423
func applyExtraEmpowerMods(activeSkill *ActiveSkill) {
	skillModList := activeSkill.SkillModList
	var empoweredMod mod.Mod
	for _, m := range skillModList.Mods() {
		if m.Name() == "Empowered" {
			empoweredMod = m
			break
		}
	}
	if empoweredMod == nil {
		return
	}
	for _, v := range skillModList.List(activeSkill.SkillCfg, "ExtraEmpowerMod") {
		wrapped, ok := v.(interface{ NestedMod() mod.Mod })
		if !ok {
			continue
		}
		m := wrapped.NestedMod()
		if m.Flags()&empoweredMod.Flags() == m.Flags() && mod.MatchKeywordFlags(empoweredMod.KeywordFlags(), m.KeywordFlags()) {
			newMod := m.Clone()
			newMod.Tag(empoweredMod.Tags()...)
			skillModList.AddMod(newMod)
		}
	}
}

// truthy is Lua truth for a skillData value.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	}
	return true
}

// buildActiveSkillModList builds the list of modifiers for the given active skill. Multipart skills,
// mine and stage counts and totem level are not ported: PoE2's skill data here carries no parts and the
// gem instance no saved counts.
//
//pob2:src/Modules/CalcActiveSkill.lua:425-1133
func buildActiveSkillModList(env *Environment, activeSkill *ActiveSkill) {
	skillTypes := activeSkill.SkillTypes
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect
	activeStatSet := activeEffect.StatSet.StatSet
	skillFlags := activeEffect.StatSet.SkillFlags
	if skillFlags == nil {
		skillFlags = map[string]bool{}
		activeEffect.StatSet.SkillFlags = skillFlags
	}
	activeSkill.SkillFlags = skillFlags
	// Active skills granted by support gems inherit the level of the skill that support applied to.
	supportGrantedInheritedLevel := 0
	if activeEffect.GemData != nil && activeEffect.GemData.GrantedEffect.Support {
		for _, skill := range env.Player.ActiveSkillList {
			effect := skill.ActiveEffect
			if effect != activeEffect && skill.SocketGroup == activeSkill.SocketGroup &&
				!(effect.GemData != nil && effect.GemData.GrantedEffect.Support) {
				// A meta group can contain unrelated active skills. Only inherit from
				// a skill to which this specific support instance was applied.
				for _, supportEffect := range skill.EffectList {
					if supportEffect.GrantedEffect.Support && supportEffect.SrcInstance == activeEffect.SrcInstance {
						supportGrantedInheritedLevel = effect.Level
						activeEffect.Level = effect.Level
						activeSkill.SkillData["inheritsGemLevel"] = true
						break
					}
				}
				if supportGrantedInheritedLevel != 0 {
					break
				}
			}
		}
	}

	// Set mode flags
	if env.ModeBuffs {
		skillFlags["buffs"] = true
	}
	if env.ModeCombat {
		skillFlags["combat"] = true
	}
	if env.ModeEffective {
		skillFlags["effective"] = true
	}

	if (skillTypes["RequiresShield"] || skillFlags["shieldAttack"]) && activeSkill.SummonSkill == nil {
		if weapon2 := activeSkill.Actor.ItemList["Weapon 2"]; weapon2 == nil || weapon2.Type != "Shield" {
			// Skill requires a shield to be equipped
			skillFlags["disable"] = true
			activeSkill.DisableReason = "This skill requires a Shield"
		}
	}

	// initialize weapon flags
	activeSkill.Weapon1Flags = 0
	activeSkill.Weapon2Flags = 0
	// Initialise skill modifier list
	skillModList := newModList()
	skillModList.Parent = activeSkill.Actor.ModDB
	activeSkill.SkillModList = skillModList
	activeSkill.BaseSkillModList = skillModList

	//pob2:block:src/Modules/CalcActiveSkill.lua:514-571
	if skillFlags["shieldAttack"] {
		// Special handling for Spectral Shield Throw
		skillFlags["weapon2Attack"] = true
		activeSkill.Weapon2Flags = 0
	} else {
		// Set weapon flags
		var weaponTypes []map[string]bool
		if activeGrantedEffect.WeaponTypes != nil {
			weaponTypes = append(weaponTypes, activeGrantedEffect.WeaponTypes)
		}
		for _, skillEffect := range activeSkill.EffectList {
			if skillEffect.GrantedEffect.Support && skillEffect.GrantedEffect.WeaponTypes != nil {
				weaponTypes = append(weaponTypes, skillEffect.GrantedEffect.WeaponTypes)
			}
		}
		weapon1Flags, weapon1OK, weapon1Info, weapon1HasInfo := getWeaponFlags(actorWeaponData(activeSkill.Actor, 1), weaponTypes)
		if !weapon1OK && activeSkill.SummonSkill != nil {
			// Minion skills seem to ignore weapon types
			weapon1Info, weapon1HasInfo = weaponTypeInfo["None"], true
			weapon1Flags, weapon1OK = modFlagByName[weapon1Info.flag], true
		}
		if weapon1OK {
			if skillFlags["attack"] || skillFlags["dotFromAttack"] {
				// Concoction skills ignore weapon flags
				if skillFlags["unarmed"] {
					activeSkill.Weapon1Flags = ModFlagUnarmed
				} else {
					activeSkill.Weapon1Flags = weapon1Flags
				}
				skillFlags["weapon1Attack"] = true
				if weapon1Info.melee && skillFlags["melee"] {
					delete(skillFlags, "projectile")
				} else if !weapon1Info.melee && skillFlags["projectile"] {
					delete(skillFlags, "melee")
				}
			}
		} else if (skillTypes["DualWieldOnly"] || skillTypes["MainHandOnly"] || skillFlags["forceMainHand"] || weapon1HasInfo) && activeSkill.SummonSkill == nil {
			// Skill requires a compatible main hand weapon
			skillFlags["disable"] = true
			activeSkill.DisableReason = "Main Hand weapon is not usable with this skill"
		}
		if !skillTypes["MainHandOnly"] && !skillFlags["forceMainHand"] {
			weapon2Flags, weapon2OK, _, weapon2HasInfo := getWeaponFlags(actorWeaponData(activeSkill.Actor, 2), weaponTypes)
			if weapon2OK {
				if skillTypes["DualWieldRequiresDifferentTypes"] && activeSkill.Actor.WeaponData1Type == activeSkill.Actor.WeaponData2Type {
					// Skill requires a different compatible off hand weapon to main hand weapon
					skillFlags["disable"] = true
					if activeSkill.DisableReason == "" {
						activeSkill.DisableReason = "Weapon Types Need to be Different"
					}
				} else if skillFlags["attack"] || skillFlags["dotFromAttack"] {
					if skillFlags["unarmed"] {
						activeSkill.Weapon2Flags = ModFlagUnarmed
					} else {
						activeSkill.Weapon2Flags = weapon2Flags
					}
					skillFlags["weapon2Attack"] = true
				}
			} else if (skillTypes["DualWieldOnly"] || weapon2HasInfo) && activeSkill.SummonSkill == nil {
				// Skill requires a compatible off hand weapon
				skillFlags["disable"] = true
				if activeSkill.DisableReason == "" {
					activeSkill.DisableReason = "Off Hand weapon is not usable with this skill"
				}
			} else if skillFlags["disable"] {
				// Neither weapon is compatible
				if activeSkill.DisableReason == "" {
					activeSkill.DisableReason = "No usable weapon equipped"
				}
			}
		}
		if skillFlags["attack"] {
			if skillFlags["weapon1Attack"] && skillFlags["weapon2Attack"] {
				skillFlags["bothWeaponAttack"] = true
			} else {
				delete(skillFlags, "bothWeaponAttack")
			}
		}
	}

	// Apply stat-map flagged skill flags.
	//pob2:block:src/Modules/CalcActiveSkill.lua:573-579
	if activeStatSet != nil {
		stats := buildSkillInstanceStats(activeEffect, activeGrantedEffect, activeStatSet, env.useAltGemQualityStats)
		for _, stat := range stats.order {
			if entry, ok := poe2.SkillStatMapEntry(stat); ok && stats.values[stat] != 0 && entry.SkillFlag != "" {
				skillFlags[entry.SkillFlag] = true
			}
		}
	}
	// Build skill mod flag set
	//pob2:block:src/Modules/CalcActiveSkill.lua:580-603
	var skillModFlags mod.MFlag
	if skillFlags["hit"] {
		skillModFlags |= ModFlagHit
	}
	switch {
	case skillFlags["attack"] || skillFlags["nonWeaponAttack"]:
		skillModFlags |= ModFlagAttack
	case skillFlags["thorns"]:
		skillModFlags |= ModFlagThorns
	default:
		skillModFlags |= ModFlagCast
		if skillFlags["spell"] {
			skillModFlags |= ModFlagSpell
		}
	}
	if skillFlags["melee"] {
		skillModFlags |= ModFlagMelee
	} else if skillFlags["projectile"] {
		skillModFlags |= ModFlagProjectile
		skillFlags["chaining"] = true
	}
	if skillFlags["area"] {
		skillModFlags |= ModFlagArea
	}

	// Build skill keyword flag set
	//pob2:block:src/Modules/CalcActiveSkill.lua:605-663
	var skillKeywordFlags mod.KeywordFlag
	if skillFlags["hit"] {
		skillKeywordFlags |= KeywordFlagHit
	}
	for _, kw := range []struct {
		skillType string
		flag      mod.KeywordFlag
	}{
		{"Aura", KeywordFlagAura}, {"AppliesCurse", KeywordFlagCurse}, {"Warcry", KeywordFlagWarcry},
		{"Movement", KeywordFlagMovement}, {"Vaal", KeywordFlagVaal}, {"Lightning", KeywordFlagLightning},
		{"Cold", KeywordFlagCold}, {"Fire", KeywordFlagFire}, {"Chaos", KeywordFlagChaos}, {"Physical", KeywordFlagPhysical},
	} {
		if skillTypes[kw.skillType] {
			skillKeywordFlags |= kw.flag
		}
	}
	if skillFlags["weapon1Attack"] && activeSkill.Weapon1Flags&ModFlagBow != 0 {
		skillKeywordFlags |= KeywordFlagBow
	}
	if skillFlags["brand"] {
		skillKeywordFlags |= KeywordFlagBrand
	}
	if skillFlags["arrow"] {
		skillKeywordFlags |= KeywordFlagArrow
	}
	switch {
	case skillFlags["totem"]:
		skillKeywordFlags |= KeywordFlagTotem
	case skillFlags["trap"]:
		skillKeywordFlags |= KeywordFlagTrap
	case skillFlags["mine"]:
		skillKeywordFlags |= KeywordFlagMine
	case !skillTypes["Triggered"]:
		skillFlags["selfCast"] = true
	}
	if skillTypes["Attack"] {
		skillKeywordFlags |= KeywordFlagAttack
	}
	if skillTypes["Spell"] && !skillFlags["cast"] {
		skillKeywordFlags |= KeywordFlagSpell
	}

	// Find totem base stats
	//pob2:block:src/Modules/CalcActiveSkill.lua:665-685
	if skillFlags["totem"] {
		base := getTotemBaseStats(activeSkill)
		if base.grantedEffect != nil && base.gemData != nil {
			activeSkill.SkillData["totemBase"] = base
		}

		// Get skill totem ID for totem skills
		// This is used to calculate totem life
		switch {
		case activeGrantedEffect.HasSkillTotemID:
			activeSkill.SkillTotemID = activeGrantedEffect.SkillTotemID
		case base.grantedEffect != nil && base.gemData != nil && base.grantedEffect.HasSkillTotemID:
			activeSkill.SkillTotemID = base.grantedEffect.SkillTotemID
		case activeGrantedEffect.Color == 2:
			activeSkill.SkillTotemID = 2
		case activeGrantedEffect.Color == 3:
			activeSkill.SkillTotemID = 3
		default:
			activeSkill.SkillTotemID = 1
		}
	}

	// Calculate distance from enemy
	//pob2:block:src/Modules/CalcActiveSkill.lua:687-714
	effectiveRange := configNumberOr(env, enemyConfigPlaceholders(env), "enemyDistance", 20)

	// Build config structure for modifier searches
	cfgFlags := skillModFlags | activeSkill.Weapon1Flags
	cfgKeywordFlags := skillKeywordFlags
	activeSkill.SkillCfg = &moddb.ListCfg{
		Flags:              &cfgFlags,
		KeywordFlags:       &cfgKeywordFlags,
		SkillName:          strings.TrimPrefix(activeGrantedEffect.Name, "Vaal "), // This allows modifiers that target specific skills to also apply to their Vaal counterpart
		SkillGrantedEffect: skillGrantedEffectCfg{activeGrantedEffect},
		SkillPart:          activeSkill.SkillPart,
		SkillTypes:         cfgSkillTypes(activeSkill.SkillTypes),
		SkillCond:          map[string]bool{},
		SkillDist:          effectiveRange,
		SkillDistSet:       env.ModeEffective,
		SlotName:           activeSkill.SlotName,
	}
	if activeSkill.SummonSkill != nil {
		activeSkill.SkillCfg.SummonSkillName = activeSkill.SummonSkill.ActiveEffect.GrantedEffect.Name
	}
	if activeEffect.GemData != nil {
		activeSkill.SkillCfg.GemData = skillGemCfg{activeEffect.GemData}
	}
	if skillFlags["weapon1Attack"] {
		cfg := *activeSkill.SkillCfg
		flags := skillModFlags | activeSkill.Weapon1Flags
		cfg.Flags = &flags
		cfg.SkillCond = map[string]bool{"MainHandAttack": true}
		activeSkill.Weapon1Cfg = &cfg
	}
	if skillFlags["weapon2Attack"] {
		cfg := *activeSkill.SkillCfg
		flags := skillModFlags | activeSkill.Weapon2Flags
		cfg.Flags = &flags
		cfg.SkillCond = map[string]bool{"OffHandAttack": true}
		activeSkill.Weapon2Cfg = &cfg
	}

	//pob2:block:src/Modules/CalcActiveSkill.lua:726-743
	if skillModList.Flag(activeSkill.SkillCfg, "DisableSkill") && !skillModList.Flag(activeSkill.SkillCfg, "EnableSkill") {
		skillFlags["disable"] = true
		activeSkill.DisableReason = "Skills of this type are disabled"
	}

	if skillFlags["disable"] {
		for k := range skillFlags {
			delete(skillFlags, k)
		}
		skillFlags["disable"] = true
		activeEffect.Level = validateGemLevel(activeGrantedEffect, activeEffect.GemData, activeEffect.Level)
		activeEffect.GrantedEffectLevel = newGrantedEffectLevel(activeGrantedEffect, activeStatSet, activeEffect.Level)
		return
	}
	// Add support gem modifiers to skill mod list
	//pob2:block:src/Modules/CalcActiveSkill.lua:744-784
	for _, skillEffect := range activeSkill.EffectList {
		if !skillEffect.GrantedEffect.Support {
			continue
		}
		mergeSkillInstanceMods(env, skillModList, skillEffect, nil, nil)
		modSource := mod.Source(skillEffect.GrantedEffect.ModSource)
		var extra map[string]float64
		if level := skillEffect.GrantedEffect.Levels[skillEffect.Level]; level != nil {
			extra = level.Extra
		}
		if v, ok := extra["manaMultiplier"]; ok {
			skillModList.AddMod(newMod("SupportManaMultiplier", mod.TypeMore, v, modSource))
		}
		if v, ok := extra["reservationMultiplier"]; ok {
			skillModList.AddMod(newMod("ReservationMultiplier", mod.TypeMore, v, modSource))
		}
		if v, ok := extra["manaReservationPercent"]; ok {
			activeSkill.SkillData["manaReservationPercent"] = v
		}
		if v, ok := extra["spiritReservationFlat"]; ok {
			skillModList.AddMod(newMod("ExtraSpirit", mod.TypeBase, v, modSource))
		}
		if v, ok := extra["cooldown"]; ok {
			skillModList.AddMod(newMod("CooldownRecovery", mod.TypeBase, v, modSource))
		}
		// Handle multiple triggers situation and if triggered by a trigger skill save a reference to the trigger.
		match := skillEffect.GrantedEffect.AddSkillTypes != nil && !skillFlags["disable"]
		if match && skillEffect.GrantedEffect.IsTrigger {
			if activeSkill.TriggeredBy != nil {
				skillFlags["disable"] = true
				activeSkill.DisableReason = "This skill is supported by more than one trigger"
			} else {
				activeSkill.TriggeredBy = skillEffect
			}
		}
		if !skillEffect.GrantedEffect.Hidden {
			skillModList.AddMod(newMod("Multiplier:SupportCount", mod.TypeBase, 1, "Support Count"))
		}
		if v, ok := extra["PvPDamageMultiplier"]; ok {
			skillModList.AddMod(newMod("PvpDamageMultiplier", mod.TypeMore, v, modSource))
		}
		if v, ok := extra["storedUses"]; ok {
			activeSkill.SkillData["storedUses"] = v
		}
	}

	// Apply gem/quality modifiers from support gems
	//pob2:block:src/Modules/CalcActiveSkill.lua:786-811
	gemMaxLevel := activeEffect.Level
	gemMaxLevelSource := mod.Source("Max Level")
	switch {
	case supportGrantedInheritedLevel != 0:
		gemMaxLevel = supportGrantedInheritedLevel
		gemMaxLevelSource = "Inherited Max Level"
	case activeEffect.SrcInstance != nil:
		gemMaxLevel = activeEffect.SrcInstance.Level
	}
	skillModList.AddMod(newMod("GemLevel", mod.TypeBase, float64(gemMaxLevel), gemMaxLevelSource))
	gemBaseQuality := activeEffect.Quality
	if activeEffect.SrcInstance != nil {
		gemBaseQuality = activeEffect.SrcInstance.Quality
	}
	skillModList.AddMod(newMod("GemQuality", mod.TypeBase, gemBaseQuality, "Gem Quality"))
	if src := activeEffect.SrcInstance; src != nil && src.Corrupted && !(src.FromItem || src.FromTree || activeGrantedEffect.Overlay.FromItem || env.effectFromItem(activeGrantedEffect) || activeGrantedEffect.Overlay.FromTree) {
		skillModList.AddMod(newMod("GemCorruptionLevel", mod.TypeBase, float64(src.CorruptLevel), "Corruption"))
		activeSkill.SkillCfg.SkillCond["GemCorrupted"] = true
	}
	for _, supportProperty := range getSourceGemPropertyInfo(env, activeSkill) {
		value := supportProperty.Value
		if value.Keyword == "grants_active_skill" {
			switch value.Key {
			case "level":
				activeEffect.Level += value.Value
			case "quality":
				activeEffect.Quality += float64(value.Value)
			default:
				if activeEffect.Values == nil {
					activeEffect.Values = map[string]float64{}
				}
				activeEffect.Values[value.Key] += float64(value.Value)
			}
			m := newMod("GemSupport"+upperFirst(value.Key), mod.TypeBase, float64(value.Value), supportProperty.Source)
			if supportProperty.Tag != nil {
				m.Tag(supportProperty.Tag)
			}
			skillModList.AddMod(m)
		}
	}

	for _, gemProperty := range activeEffect.GemPropertyInfo {
		m := newMod("GemGlobal"+upperFirst(gemProperty.Value.Key), mod.TypeBase, gemProperty.Value.Value, gemProperty.Source)
		if gemProperty.Tag != nil {
			m.Tag(gemProperty.Tag)
		}
		skillModList.AddMod(m)
	}

	// Add active gem modifiers
	//pob2:block:src/Modules/CalcActiveSkill.lua:813-828
	activeEffect.HasActorLevel = false
	var extraStats []mod.ExtraSkillStat
	for _, v := range skillModList.List(activeSkill.SkillCfg, "ExtraSkillStat") {
		switch x := v.(type) {
		case mod.ExtraSkillStat:
			extraStats = append(extraStats, x)
		case *mod.ExtraSkillStat:
			extraStats = append(extraStats, *x)
		}
	}
	if activeStatSet != nil {
		mergeSkillInstanceMods(env, skillModList, activeEffect, activeStatSet, extraStats)
	} else {
		mergeSkillInstanceMods(env, skillModList, activeEffect, nil, extraStats)
	}
	level := newGrantedEffectLevel(activeGrantedEffect, activeStatSet, activeEffect.Level)
	if activeEffect.SrcInstance != nil && activeEffect.SrcInstance.NoReservation {
		for _, resource := range []string{"mana", "life", "spirit"} {
			level.Extra[resource+"ReservationFlat"] = 0
			level.Extra[resource+"ReservationPercent"] = 0
		}
	}
	activeEffect.GrantedEffectLevel = level

	// Add extra modifiers from granted effect level
	//pob2:block:src/Modules/CalcActiveSkill.lua:830-856
	modSource := mod.Source(activeGrantedEffect.ModSource)
	if v, ok := level.Extra["reservationMultiplier"]; ok {
		skillModList.AddMod(newMod("ReservationMultiplier", mod.TypeMore, v, modSource))
	}
	if v, ok := level.Extra["critChance"]; ok {
		activeSkill.SkillData["CritChance"] = v
	} else {
		delete(activeSkill.SkillData, "CritChance")
	}
	if v, ok := level.Extra["damageMultiplier"]; ok {
		skillModList.AddMod(newMod("Damage", mod.TypeMore, v, modSource).Flag(ModFlagAttack))
	}
	for _, key := range []string{"attackTime", "attackSpeedMultiplier", "cooldown", "storedUses", "soulPreventionDuration"} {
		if v, ok := level.Extra[key]; ok {
			activeSkill.SkillData[key] = v
		}
	}
	if v, ok := level.Extra["PvPDamageMultiplier"]; ok {
		skillModList.AddMod(newMod("PvpDamageMultiplier", mod.TypeMore, v, modSource))
	}

	// Add extra modifiers from other sources
	//pob2:block:src/Modules/CalcActiveSkill.lua:858-865
	activeSkill.ExtraSkillModList = nil
	for _, v := range skillModList.List(activeSkill.SkillCfg, "ExtraSkillMod") {
		var m mod.Mod
		switch x := v.(type) {
		case mod.ExtraSkillMod:
			m = x.Mod
		case *mod.ExtraSkillMod:
			m = x.Mod
		}
		if m == nil {
			continue
		}
		skillModList.AddMod(m)
		activeSkill.ExtraSkillModList = append(activeSkill.ExtraSkillModList, m)
	}

	applyExtraEmpowerMods(activeSkill)

	// Determine if it possible to have a stage on this skill based upon skill parts.
	//pob2:block:src/Modules/CalcActiveSkill.lua:890-900
	stageName := strings.Join(strings.Fields(activeGrantedEffect.Name), "")
	if skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:"+stageName+"MaxStages") > 0 {
		skillFlags["multiStage"] = true
		activeSkill.ActiveStageCount = math.Max(1, 1+skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:"+stageName+"MinimumStage"))
		limit := skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:"+stageName+"MaxStages")
		if limit > 0 && activeSkill.ActiveStageCount > 0 {
			skillModList.AddMod(newMod("Multiplier:"+stageName+"Stage", mod.TypeBase, math.Min(limit, activeSkill.ActiveStageCount), "Base"))
			activeSkill.ActiveStageCount--
			skillModList.AddMod(newMod("Multiplier:"+stageName+"StageAfterFirst", mod.TypeBase, math.Min(limit-1, activeSkill.ActiveStageCount), "Base"))
		}
	}

	// Hollow Palm Technique added phys for skills that would use Quarterstaff
	//pob2:block:src/Modules/CalcActiveSkill.lua:906-913
	if activeSkill.Actor.ModDB.Conditions["HollowPalm"] && !(skillModList.Flag(nil, "UseFacebreakerItemDamage") && activeEffect.GrantedEffect.WeaponTypes != nil && activeEffect.GrantedEffect.WeaponTypes["One Hand Mace"]) && ((activeEffect.GrantedEffect.WeaponTypes != nil && activeEffect.GrantedEffect.WeaponTypes["Staff"]) || skillModList.Flag(activeSkill.SkillCfg, "UseHollowPalmDamage")) {
		gemLevel := max(activeEffect.Level, 1)
		if gemLevel <= len(poe2.HollowPalmAddedPhys) {
			phys := poe2.HollowPalmAddedPhys[gemLevel-1]
			// The Lua passes a nil keywordFlags before the Condition tag, so createMod keeps no tag.
			skillModList.AddMod(newMod("PhysicalMin", mod.TypeBase, phys[0], "Hollow Palm Technique").Flag(ModFlagAttack))
			skillModList.AddMod(newMod("PhysicalMax", mod.TypeBase, phys[1], "Hollow Palm Technique").Flag(ModFlagAttack))
		}
	}

	// Extract skill data
	//pob2:block:src/Modules/CalcActiveSkill.lua:915-921
	for _, list := range [][]any{env.ModDB.List(activeSkill.SkillCfg, "SkillData"), skillModList.List(activeSkill.SkillCfg, "SkillData")} {
		for _, v := range list {
			var sd mod.SkillData
			switch x := v.(type) {
			case mod.SkillData:
				sd = x
			case *mod.SkillData:
				sd = *x
			default:
				continue
			}
			activeSkill.SkillData[sd.Key] = sd.SetValue()
		}
	}

	// Create minion
	createMinion(env, activeSkill)

	// Separate global effect modifiers (mods that can affect defensive stats or other skills)
	//pob2:block:src/Modules/CalcActiveSkill.lua:1053-1132
	kept := make([]mod.Mod, 0, len(skillModList.Mods()))
	for _, m := range skillModList.Mods() {
		var effectTag *mod.GlobalEffectTag
		for _, tag := range m.Tags() {
			if g, ok := tag.(*mod.GlobalEffectTag); ok {
				effectTag = g
				break
			}
		}
		if effectTag == nil {
			kept = append(kept, m)
			continue
		}
		if cond, _ := skillModList.GetCondition(effectTag.ModCondTag, activeSkill.SkillCfg, false); effectTag.ModCondTag != "" && !cond {
			continue
		}
		var effectType string
		if len(effectTag.GlobalEffectList) > 0 {
			effectType = effectTag.GlobalEffectList[0]
		}
		if effectType == "" {
			kept = append(kept, m)
			continue
		}
		effectName := effectTag.NameTag
		if effectName == "" {
			effectName = activeGrantedEffect.Name
		}
		if activeSkill.Minion != nil && activeSkill.Minion.MinionData != nil {
			if strings.Contains(effectName, "{0}") {
				effectName = strings.ReplaceAll(effectName, "{0}", activeSkill.Minion.MinionData.Name)
			} else if activeGrantedEffect.Overlay.MinionList != nil && strings.HasPrefix(effectName, "Companion") {
				effectName = "Companion: " + activeSkill.Minion.MinionData.Name
			} else if activeGrantedEffect.Overlay.MinionList != nil && strings.HasPrefix(effectName, "Spectre") {
				effectName = "Spectre: " + activeSkill.Minion.MinionData.Name
			}
		}
		var buff *skillBuff
		for _, skillBuff := range activeSkill.BuffList {
			if skillBuff.Type == effectType && skillBuff.Name == effectName {
				buff = skillBuff
				break
			}
		}
		if buff == nil {
			buff = &skillBuff{
				Type:           effectType,
				Name:           effectName,
				AllowTotemBuff: effectTag.AllowTotemBuffTag,
				Cond:           effectTag.EffectCondTag,
				EnemyCond:      effectTag.EffectEnemyCondTag,
				StackVar:       effectTag.EffectStackVarTag,
				StackLimit:     effectTag.EffectStackLimitTag,
				StackLimitVar:  effectTag.EffectStackLimitVarTag,
				ApplyNotPlayer: effectTag.ApplyNotPlayerTag,
				ApplyMinions:   effectTag.ApplyMinionsTag,
			}
			if m.GetSource() == modSource {
				// Inherit buff configuration from the active skill
				buff.ActiveSkillBuff = true
				buff.ApplyNotPlayer = buff.ApplyNotPlayer || truthy(activeSkill.SkillData["buffNotPlayer"])
				buff.ApplyMinions = buff.ApplyMinions || truthy(activeSkill.SkillData["buffMinions"])
				buff.ApplyAllies = truthy(activeSkill.SkillData["buffAllies"])
				buff.AllowTotemBuff = truthy(activeSkill.SkillData["allowTotemBuff"])
			}
			activeSkill.BuffList = append(activeSkill.BuffList, buff)
		}
		match := false
		for d, destMod := range buff.ModList {
			if moddb.CompareModParams(m, destMod) && (destMod.Type() == mod.TypeBase || destMod.Type() == mod.TypeIncrease) {
				merged := destMod.Clone()
				merged.Value().SetFloat(destMod.Value().Float() + m.Value().Float())
				buff.ModList[d] = merged
				match = true
				break
			}
		}
		if !match {
			buff.ModList = append(buff.ModList, m)
		}
	}
	skillModList.SetMods(kept)

	if len(activeSkill.BuffList) > 0 {
		// Add to auxiliary skill list
		env.AuxSkillList = append(env.AuxSkillList, activeSkill)
	}
}

// newGrantedEffectLevel is copyTable(levels[level]) with statSet.levels[level]'s fields written over it.
func newGrantedEffectLevel(ge *poe2skills.GrantedEffect, statSet *poe2skills.StatSet, level int) *grantedEffectLevel {
	out := &grantedEffectLevel{Extra: map[string]float64{}}
	if row := ge.Levels[level]; row != nil {
		for k, v := range row.Extra {
			out.Extra[k] = v
		}
		if row.Cost != nil {
			out.Cost = make(map[string]float64, len(row.Cost))
			for k, v := range row.Cost {
				out.Cost[k] = v
			}
		}
	}
	if statSet != nil {
		if row := statSet.Levels[level]; row != nil {
			for k, v := range row.Extra {
				out.Extra[k] = v
			}
			if row.HasActorLevel {
				out.Extra["actorLevel"] = row.ActorLevel
			}
			out.Values = row.Values
			out.StatInterpolation = row.StatInterpolation
		}
	}
	return out
}

// upperFirst is key:gsub("^%l", string.upper).
func upperFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:]
}

// setupWeaponData gets the weapon data tables for the equipped weapons. countsAsDualWielding and the
// Facebreaker damage fields are not ported.
//
//pob2:block:src/Modules/CalcSetup.lua:2119-2153
func setupWeaponData(env *Environment) {
	env.Player.WeaponData1Type = "None"
	weapon1 := env.Player.ItemList["Weapon 1"]
	if weapon1 != nil && weapon1.WeaponData[1] != nil {
		env.Player.WeaponData1Type = weapon1.Base.Type
	}
	if weapon2 := env.Player.ItemList["Weapon 2"]; weapon2 == nil {
		// Hollow Palm Technique
		if weapon1 == nil {
			for _, m := range env.ModDB.Mods["Keystone"] {
				if v, _ := m.Value().ValueList.(string); v == "Hollow Palm Technique" {
					env.Player.WeaponData2Type = "None"
					env.Player.WeaponData1AsThoughUsing = map[string]bool{"Staff": true}
					env.Player.WeaponData2AsThoughUsing = map[string]bool{"Staff": true}
					env.ModDB.Conditions["HollowPalm"] = true
					break
				}
			}
		}
		// Facebreaker: Can Attack as though using a One Handed Mace while both of your hand slots are empty
		if weapon1 == nil && env.ModDB.Flag(nil, "CanAttackAsOneHandMaceUnarmed") {
			if env.Player.WeaponData1AsThoughUsing == nil {
				env.Player.WeaponData1AsThoughUsing = map[string]bool{}
			}
			env.Player.WeaponData1AsThoughUsing["One Hand Mace"] = true
		}
	} else if weapon2.WeaponData[2] != nil {
		env.Player.WeaponData2Type = weapon2.Base.Type
	}
}
