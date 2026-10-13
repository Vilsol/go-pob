package calculator

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

var skillFlagIDAliases = map[string]string{
	"PlayerMelee": "Melee",
}

//pob:block:src/Data/Skills/act_dex.lua:11581
var skillStatKeyAliases = map[string]string{
	"critical_strike_chance_+%": "skill_buff_grants_critical_strike_chance_+%",
}

func skillStatKeyForPoB(id string, key string) string {
	alias, aliased := skillStatKeyAliases[key]
	if !aliased {
		return key
	}
	perSkill := raw2.SkillStatMaps[id]
	if perSkill == nil || perSkill[key] != nil || perSkill[alias] == nil {
		return key
	}
	return alias
}

//pob:block:src/Modules/CalcActiveSkill.lua:245-246
func skillPartsFor(id string) []raw2.SkillPart {
	if alias, aliased := skillFlagIDAliases[id]; aliased {
		id = alias
	}

	return raw2.SkillParts[id]
}

//pob:block:src/Modules/CalcActiveSkill.lua:102-104
func SkillBaseFlagsFor(id string) (map[SkillFlag]bool, bool) {
	if alias, aliased := skillFlagIDAliases[id]; aliased {
		id = alias
	}

	flags, ok := raw2.SkillBaseFlags[id]
	out := make(map[SkillFlag]bool, len(flags)+1)
	for _, flag := range flags {
		out[SkillFlag(flag)] = true
	}
	return out, ok
}

// It will determine the base flag set, and check which of the support gems can support this skill
//
//pob:src/Modules/CalcActiveSkill.lua:82-125
func CreateActiveSkill(activeEffect *GemEffect, supportList []*GemEffect, actor *Actor, socketGroup *pob.Skill, summonSkill *ActiveSkill) *ActiveSkill {
	activeSkill := &ActiveSkill{
		ActiveEffect: activeEffect,
		SupportList:  supportList,
		SkillData:    &SkillData{},
		Actor:        actor,
		SocketGroup:  socketGroup,
		SummonSkill:  summonSkill,
	}

	activeGrantedEffect := activeEffect.GrantedEffect

	activeSkill.SkillTypes = utils.CopyMap(activeEffect.GrantedEffect.SkillTypes)

	// Initialise skill types
	//pob:src/Modules/CalcActiveSkill.lua:97-99
	if as := activeGrantedEffect.Raw.GetActiveSkill(); as != nil && as.MinionActiveSkillTypes != nil {
		activeSkill.MinionSkillTypes = make(map[data.SkillType]bool)

		for _, minionType := range as.MinionActiveSkillTypes {
			activeSkill.MinionSkillTypes[data.SkillType(poe.ActiveSkillTypes[minionType].ID)] = true
		}
	}

	// Initialise skill flag set ('attack', 'projectile', etc)
	activeSkill.SkillFlags, _ = SkillBaseFlagsFor(activeGrantedEffect.ID())
	activeSkill.SkillFlags[SkillFlagHit] = activeSkill.SkillFlags[SkillFlagHit] || activeSkill.SkillTypes[data.SkillTypeAttack] || activeSkill.SkillTypes[data.SkillTypeDamage] || activeSkill.SkillTypes[data.SkillTypeProjectile]

	activeSkill.EffectList = make([]*GemEffect, 1)
	activeSkill.EffectList[0] = activeEffect

	for _, supportEffect := range supportList {
		// Pass 1: Add skill types from compatible supports
		if CalcCanGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill) {
			if override, ok := raw2.SkillAddSkillTypes[supportEffect.GrantedEffect.ID()]; ok {
				for _, skillType := range override {
					activeSkill.SkillTypes[data.SkillType(skillType)] = true
				}
			} else {
				for _, skillType := range supportEffect.GrantedEffect.Raw.AddTypes {
					activeSkill.SkillTypes[data.SkillType(poe.ActiveSkillTypes[skillType].ID)] = true
				}
			}
		}
	}

	for _, supportEffect := range supportList {
		// Pass 2: Add all compatible supports
		if CalcCanGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill) {
			activeSkill.EffectList = append(activeSkill.EffectList, supportEffect)
			if supportEffect.IsSupporting != nil && activeEffect.SrcInstance != nil {
				supportEffect.IsSupporting[activeEffect.SrcInstance] = true
			}
			if summonSkill == nil {
				// Support skill adds flags to supported skills (eg. Remote Mine adds 'mine')
				for _, flag := range raw2.SkillAddFlags[supportEffect.GrantedEffect.ID()] {
					activeSkill.SkillFlags[SkillFlag(flag)] = true
				}
			}
		}
	}

	return activeSkill
}

//pob:block:src/Classes/SkillsTab.lua:1161-1165
func emptyTriggeredGrantedCost(activeEffect *GemEffect) {
	if !activeEffect.CostEmptied || activeEffect.GrantedEffectLevel == nil {
		return
	}
	lvl := *activeEffect.GrantedEffectLevel
	lvl.Cost = map[string]int{}
	activeEffect.GrantedEffectLevel = &lvl
}

//pob:block:src/Export/Scripts/skills.lua:173-178
func grantedEffectIsTrigger(grantedEffect *GrantedEffect) bool {
	if grantedEffect == nil || grantedEffect.Raw == nil {
		return false
	}

	if override, ok := raw2.SkillAddSkillTypes[grantedEffect.ID()]; ok {
		for _, skillType := range override {
			if data.SkillType(skillType) == data.SkillTypeTriggered {
				return true
			}
		}
		return false
	}

	for _, skillType := range grantedEffect.Raw.AddTypes {
		if data.SkillType(poe.ActiveSkillTypes[skillType].ID) == data.SkillTypeTriggered {
			return true
		}
	}

	return false
}

type skillInstanceStat struct {
	name    string
	value   float64
	statMap *raw2.StatMap
}

type skillInstanceStatsKey struct {
	raw        *poe.GrantedEffect
	level      int
	quality    int
	qualityID  string
	actorLevel int
	hasActor   bool
}

var skillInstanceStatsCache sync.Map

func skillInstanceStats(skillEffect *GemEffect, grantedEffect *GrantedEffect) []skillInstanceStat {
	key := skillInstanceStatsKey{raw: grantedEffect.Raw, level: skillEffect.Level, quality: skillEffect.Quality, qualityID: skillEffect.QualityID}
	if skillEffect.ActorLevel != nil {
		key.actorLevel, key.hasActor = *skillEffect.ActorLevel, true
	}
	if cached, ok := skillInstanceStatsCache.Load(key); ok {
		return cached.([]skillInstanceStat)
	}
	stats := CalcBuildSkillInstanceStats(skillEffect, grantedEffect)
	out := make([]skillInstanceStat, 0, len(stats))
	for stat, value := range stats {
		out = append(out, skillInstanceStat{name: stat, value: value})
	}
	slices.SortFunc(out, func(a, b skillInstanceStat) int { return strings.Compare(a.name, b.name) })
	calculated := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw)
	for i := range out {
		out[i].statMap = calculated.GetStatMapEntry(skillStatKeyForPoB(grantedEffect.Raw.ID, out[i].name))
	}
	cached, _ := skillInstanceStatsCache.LoadOrStore(key, out)
	return cached.([]skillInstanceStat)
}

//pob:src/Modules/CalcActiveSkill.lua:52-70
func CalcMergeSkillInstanceMods(env *Environment, modList *moddb.ModList, skillEffect *GemEffect, extraStats []interface{}) {
	CalcValidateGemLevel(skillEffect)

	grantedEffect := skillEffect.GrantedEffect

	if len(extraStats) > 0 {
		stats := CalcBuildSkillInstanceStats(skillEffect, grantedEffect)
		for _, stat := range extraStats {
			statCast := stat.(mod.ExtraSkillStat)
			stats[statCast.Key] = stats[statCast.Key] + utils.Number[float64](statCast.Value)
		}

		statNames := make([]string, 0, len(stats))
		for stat := range stats {
			statNames = append(statNames, stat)
		}
		slices.Sort(statNames)
		for _, stat := range statNames {
			statValue := stats[stat]
			statMap := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw).
				GetStatMapEntry(skillStatKeyForPoB(grantedEffect.Raw.ID, stat))
			if statMap != nil {
				mergeStatMapEntry(modList, statMap, statValue)
			}
		}
	} else {
		for _, stat := range skillInstanceStats(skillEffect, grantedEffect) {
			if stat.statMap != nil {
				mergeStatMapEntry(modList, stat.statMap, stat.value)
			}
		}
	}

	id := grantedEffect.ID()
	if alias, aliased := skillFlagIDAliases[id]; aliased {
		id = alias
	}
	for _, m := range raw2.SkillBaseMods[id] {
		modList.AddMod(m)
	}
}

//pob:block:src/Modules/CalcActiveSkill.lua:61-76
func mergeStatMapEntry(modList *moddb.ModList, statMap *raw2.StatMap, statValue float64) {
	for _, m := range statMap.Mods {
		mergeLevelMod(modList, m, statMapScalar(statMap, statValue))
	}
	for _, grp := range statMap.Groups {
		for _, m := range grp.Mods {
			mergeLevelMod(modList, m, statMapScalar(grp, statValue))
		}
	}
}

//pob:block:src/Modules/CalcActiveSkill.lua:68
func statMapScalar(m *raw2.StatMap, statValue float64) float64 {
	return utils.UnwrapOrF(m.Value, statValue)*utils.UnwrapOrF(m.Mult, 1)/utils.UnwrapOrF(m.Div, 1) + utils.UnwrapOrF(m.Base, 0)
}

//pob:src/Modules/CalcActiveSkill.lua:21-49
func mergeLevelMod(modList *moddb.ModList, m mod.Mod, value float64) {
	newMod := m.Clone()

	switch newMod.Value().Type() {
	case mod.ModValueMultiTypeFloat:
		newMod.Value().SetFloat(value)
	case mod.ModValueMultiTypeList:
		switch inner := newMod.Value().ValueList.(type) {
		case mod.SkillData:
			inner.Value = value
			newMod.Value().ValueList = inner
		case mod.MinionModifier:
			nested := inner.Mod.Clone()
			nested.Value().SetFloat(value)
			newMod.Value().ValueList = mod.MinionModifier{Mod: nested}
		case mod.PactDamage:
			nested := inner.Mod.Clone()
			nested.Value().SetFloat(value)
			newMod.Value().ValueList = mod.PactDamage{Mod: nested}
		case mod.SupportedGemProperty:
			inner.Value = int(value)
			newMod.Value().ValueList = inner
		}
	}

	modList.AddMod(newMod)
}

//pob:block:src/Modules/CalcActiveSkill.lua:407
func activeSkillCfgName(grantedEffect *GrantedEffect) string {
	as := grantedEffect.Raw.GetActiveSkill()
	if as == nil {
		return ""
	}

	return strings.TrimPrefix(as.DisplayedName, "Vaal ")
}

//pob:block:src/Modules/CalcActiveSkill.lua:530-537
func stageMultiplierName(grantedEffect *GrantedEffect) string {
	as := grantedEffect.Raw.GetActiveSkill()
	if as == nil {
		return ""
	}

	return strings.Join(strings.Fields(as.DisplayedName), "")
}

//pob:block:src/Modules/Build.lua:969-974
func spectreList(env *Environment) []string {
	list := make([]string, 0, len(env.Build.Build.Spectres))
	for _, spectre := range env.Build.Build.Spectres {
		if spectre.ID != "" && raw2.Minions[spectre.ID] != nil {
			list = append(list, spectre.ID)
		}
	}
	return list
}

//pob:block:src/Modules/CalcActiveSkill.lua:557,src/Modules/CalcActiveSkill.lua:563
func gemPropertyMod(prefix string, key string, value float64, src mod.Mod) mod.Mod {
	name := prefix + strings.ToUpper(key[:1]) + key[1:]
	out := mod.NewFloat(name, mod.TypeBase, value)
	if src == nil {
		return out
	}
	out = out.Source(src.GetSource())
	if tags := src.Tags(); len(tags) > 0 {
		out = out.Tag(tags[0])
	}
	return out
}

//pob:src/Modules/CalcActiveSkill.lua:178-677
func CalcBuildActiveSkillModList(env *Environment, activeSkill *ActiveSkill) {
	skillTypes := activeSkill.SkillTypes
	skillFlags := activeSkill.SkillFlags
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect

	if env.ModeBuffs {
		skillFlags[SkillFlagBuffs] = true
	}
	if env.ModeCombat {
		skillFlags[SkillFlagCombat] = true
	}
	if env.ModeEffective {
		skillFlags[SkillFlagEffective] = true
	}

	// Handle multipart skills
	activeGemParts := activeGrantedEffect.Parts
	if activeGemParts != nil {
		selected := 0
		if activeEffect.SrcInstance != nil {
			selected = activeEffect.SrcInstance.SkillPart
			if env.Mode == OutputModeCalcs && activeSkill == env.Player.MainSkill {
				selected = activeEffect.SrcInstance.SkillPartCalcs
			}
		}

		if selected < 1 {
			selected = 1
		}
		if selected > len(activeGemParts) {
			selected = len(activeGemParts)
		}
		activeSkill.SkillPart = selected

		part := activeGemParts[selected-1]
		for k, v := range part.Flags {
			if v {
				skillFlags[SkillFlag(k)] = true
			} else {
				delete(skillFlags, SkillFlag(k))
			}
		}
		activeSkill.SkillPartName = part.Name
		skillFlags[SkillFlagMultiPart] = len(activeGemParts) > 1
	}

	if (skillTypes[data.SkillTypeRequiresShield] || skillFlags[SkillFlagShieldAttack]) && activeSkill.SummonSkill == nil && (activeSkill.Actor.ItemList["Weapon 2"] == nil || activeSkill.Actor.ItemList["Weapon 2"].Type != "Shield") {
		// Skill requires a shield to be equipped
		skillFlags[SkillFlagDisable] = true
		activeSkill.DisableReason = "This skill requires a Shield"
	}

	if skillFlags[SkillFlagShieldAttack] {
		// Special handling for Spectral Shield Throw
		skillFlags[SkillFlagWeapon2Attack] = true
		activeSkill.Weapon2Flags = 0
	} else {
		// Set weapon flags

		//pob:src/Modules/CalcActiveSkill.lua:281-286

		weaponTypes := [][]data.ItemClassName{activeGrantedEffect.WeaponTypes()}
		for _, skillEffect := range activeSkill.EffectList {
			if skillEffect.GrantedEffect.Raw.IsSupport && skillEffect.GrantedEffect.WeaponTypes() != nil {
				weaponTypes = append(weaponTypes, skillEffect.GrantedEffect.WeaponTypes())
			}
		}

		weapon1Flags, weapon1Info := getWeaponFlags(env, activeSkill.Actor.WeaponData1, weaponTypes)
		if weapon1Flags == 0 && activeSkill.SummonSkill != nil {
			// Minion skills seem to ignore weapon types
			weapon1Flags = data.WeaponTypes[data.None].ModFlag
			weapon1Info = data.WeaponTypes[data.None]
		}

		if weapon1Flags != 0 {
			if skillFlags[SkillFlagAttack] || skillFlags[SkillFlagDotFromAttack] {
				activeSkill.Weapon1Flags = weapon1Flags
				skillFlags[SkillFlagWeapon1Attack] = true
				if weapon1Info.Melee && skillFlags[SkillFlagMelee] {
					delete(skillFlags, SkillFlagProjectile)
				} else if !weapon1Info.Melee && skillFlags[SkillFlagProjectile] {
					delete(skillFlags, SkillFlagMelee)
				}
			}
		} else if (skillTypes[data.SkillTypeDualWieldOnly] || skillTypes[data.SkillTypeMainHandOnly] ||
			skillFlags[SkillFlagForceMainHand] || weapon1Info != nil) && activeSkill.SummonSkill == nil {
			// Skill requires a compatible main hand weapon
			skillFlags[SkillFlagDisable] = true
			activeSkill.DisableReason = "Main Hand weapon is not usable with this skill"
		}

		if utils.MissingOrFalse(skillTypes, data.SkillTypeMainHandOnly) && utils.MissingOrFalse(skillFlags, SkillFlagForceMainHand) {
			weapon2Flags, weapon2Info := getWeaponFlags(env, activeSkill.Actor.WeaponData2, weaponTypes)
			if weapon2Flags != 0 {
				//pob:src/Modules/CalcActiveSkill.lua:323
				if skillFlags[SkillFlagAttack] || skillFlags[SkillFlagDotFromAttack] {
					activeSkill.Weapon2Flags = weapon2Flags
					skillFlags[SkillFlagWeapon2Attack] = true
				}
			} else if (skillTypes[data.SkillTypeDualWieldOnly] || weapon2Info != nil) && activeSkill.SummonSkill == nil {
				// Skill requires a compatible off hand weapon
				skillFlags[SkillFlagDisable] = true
				//pob:src/Modules/CalcActiveSkill.lua:330
				if activeSkill.DisableReason == "" {
					activeSkill.DisableReason = "Off Hand weapon is not usable with this skill"
				}
			} else if skillFlags[SkillFlagDisable] {
				// Neither weapon is compatible
				activeSkill.DisableReason = "No usable weapon equipped"
			}
		}

		if skillFlags[SkillFlagAttack] {
			skillFlags[SkillFlagBothWeaponAttack] = skillFlags[SkillFlagWeapon1Attack] && skillFlags[SkillFlagWeapon2Attack]
		}
	}

	// Apply stat-map flagged skill flags.
	//pob:block:src/Modules/CalcActiveSkill.lua:341-347
	if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.GrantedEffect != nil &&
		activeSkill.ActiveEffect.GrantedEffect.Raw != nil {
		grantedEffect := activeSkill.ActiveEffect.GrantedEffect
		CalcValidateGemLevel(activeSkill.ActiveEffect)
		for _, stat := range skillInstanceStats(activeSkill.ActiveEffect, grantedEffect) {
			if stat.value == 0 {
				continue
			}
			if stat.statMap != nil && stat.statMap.SkillFlag != "" {
				skillFlags[SkillFlag(stat.statMap.SkillFlag)] = true
			}
		}
	}

	// Build skill mod flag set
	skillModFlags := mod.MFlag(0)

	if utils.HasTrue(skillFlags, SkillFlagHit) {
		skillModFlags |= mod.MFlagHit
	}

	if utils.HasTrue(skillFlags, SkillFlagAttack) {
		skillModFlags |= mod.MFlagAttack
	} else {
		skillModFlags |= mod.MFlagCast
		if utils.HasTrue(skillFlags, SkillFlagSpell) {
			skillModFlags |= mod.MFlagSpell
		}
	}

	if utils.HasTrue(skillFlags, SkillFlagMelee) {
		skillModFlags |= mod.MFlagMelee
	} else if utils.HasTrue(skillFlags, SkillFlagProjectile) {
		skillModFlags |= mod.MFlagProjectile
		skillFlags[SkillFlagChaining] = true
	}

	if utils.HasTrue(skillFlags, SkillFlagArea) {
		skillModFlags |= mod.MFlagArea
	}

	// Build skill keyword flag set
	skillKeywordFlags := mod.KeywordFlag(0)

	if skillFlags[SkillFlagHit] {
		skillKeywordFlags |= mod.KeywordFlagHit
	}

	if skillTypes[data.SkillTypeAura] {
		skillKeywordFlags |= mod.KeywordFlagAura
	}

	//pob:src/Modules/CalcActiveSkill.lua:379-381
	if skillTypes[data.SkillTypeAppliesCurse] {
		skillKeywordFlags |= mod.KeywordFlagCurse
	}

	if skillTypes[data.SkillTypeWarcry] {
		skillKeywordFlags |= mod.KeywordFlagWarcry
	}

	if skillTypes[data.SkillTypeMovement] {
		skillKeywordFlags |= mod.KeywordFlagMovement
	}

	if skillTypes[data.SkillTypeVaal] {
		skillKeywordFlags |= mod.KeywordFlagVaal
	}

	if skillTypes[data.SkillTypeLightning] {
		skillKeywordFlags |= mod.KeywordFlagLightning
	}

	if skillTypes[data.SkillTypeCold] {
		skillKeywordFlags |= mod.KeywordFlagCold
	}

	if skillTypes[data.SkillTypeFire] {
		skillKeywordFlags |= mod.KeywordFlagFire
	}

	if skillTypes[data.SkillTypeChaos] {
		skillKeywordFlags |= mod.KeywordFlagChaos
	}

	if skillTypes[data.SkillTypePhysical] {
		skillKeywordFlags |= mod.KeywordFlagPhysical
	}

	if skillFlags[SkillFlagWeapon1Attack] && activeSkill.Weapon1Flags&mod.MFlagBow != 0 {
		skillKeywordFlags |= mod.KeywordFlagBow
	}

	if skillFlags[SkillFlagBrand] {
		skillKeywordFlags |= mod.KeywordFlagBrand
	}

	if skillFlags[SkillFlagArrow] {
		skillKeywordFlags |= mod.KeywordFlagArrow
	}

	if skillFlags[SkillFlagTotem] {
		skillKeywordFlags |= mod.KeywordFlagTotem
	} else if skillFlags[SkillFlagTrap] {
		skillKeywordFlags |= mod.KeywordFlagTrap
	} else if skillFlags[SkillFlagMine] {
		skillKeywordFlags |= mod.KeywordFlagMine
	} else if !skillTypes[data.SkillTypeTriggered] {
		skillFlags[SkillFlagSelfCast] = true
	}

	if skillTypes[data.SkillTypeAttack] {
		skillKeywordFlags |= mod.KeywordFlagAttack
	}

	if skillTypes[data.SkillTypeSpell] && !skillFlags[SkillFlagCast] {
		skillKeywordFlags |= mod.KeywordFlagSpell
	}

	// Get skill totem ID for totem skills
	// This is used to calculate totem life
	if skillFlags[SkillFlagTotem] {
		activeSkill.SkillTotemId = activeGrantedEffect.Raw.GetActiveSkill().SkillTotemID
		if activeSkill.SkillTotemId == 0 {
			if activeGrantedEffect.Raw.Attribute == 2 {
				activeSkill.SkillTotemId = 2
			} else if activeGrantedEffect.Raw.Attribute == 3 {
				activeSkill.SkillTotemId = 3
			} else {
				activeSkill.SkillTotemId = 1
			}
		}
	}

	// Calculate Distance for meleeDistance or projectileDistance (for melee proximity, e.g. Impact)
	effectiveRange := float64(0)
	if skillFlags[SkillFlagMelee] {
		effectiveRange = skillDistanceOption(env.Build, "meleeDistance", 15)
	} else {
		effectiveRange = skillDistanceOption(env.Build, "projectileDistance", 40)
	}

	summonSkillName := ""
	if activeSkill.SummonSkill != nil {
		if as := activeSkill.SummonSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil {
			summonSkillName = as.DisplayedName
		}
	}

	//pob:src/Modules/CalcActiveSkill.lua:455
	weaponCfgFlags := activeSkill.Weapon1Flags
	if weaponCfgFlags == 0 {
		weaponCfgFlags = activeSkill.Weapon2Flags
	}

	activeSkill.SkillCfg = &moddb.ListCfg{
		Flags:              utils.Ptr(skillModFlags | weaponCfgFlags),
		KeywordFlags:       utils.Ptr(skillKeywordFlags),
		SkillCond:          make(map[string]bool),
		SkillName:          activeSkillCfgName(activeGrantedEffect), // This allows modifiers that target specific skills to also apply to their Vaal counterpart
		SummonSkillName:    summonSkillName,
		SkillGem:           activeEffect.GemData,
		SkillGrantedEffect: activeGrantedEffect,
		SkillPart:          activeSkill.SkillPart,
		SkillTypes:         activeSkill.SkillTypes,
		SkillDist:          utils.Ternary(env.ModeEffective, effectiveRange, 0),
		SkillDistSet:       env.ModeEffective,
		SlotName:           activeSkill.SlotName,
	}
	//pob:block:src/Modules/CalcActiveSkill.lua:465-467
	if gemCfg := activeSkill.ActiveEffect.gemCfg; gemCfg != nil {
		if activeSkill.SkillCfg.SlotName == "" {
			activeSkill.SkillCfg.SlotName = gemCfg.SlotName
		}
		activeSkill.SkillCfg.SocketColor = gemCfg.SocketColor
		activeSkill.SkillCfg.SocketNum = gemCfg.SocketNum
	}

	// Build config structure for modifier searches.
	weaponCfg := func(cond string, weaponFlags mod.MFlag) *moddb.ListCfg {
		cfg := *activeSkill.SkillCfg
		cfg.Flags = utils.Ptr(skillModFlags | weaponFlags)
		cfg.SkillCond = utils.CopyMap(activeSkill.SkillCfg.SkillCond)
		cfg.SkillCond[cond] = true
		return &cfg
	}

	if skillFlags[SkillFlagWeapon1Attack] {
		activeSkill.Weapon1Cfg = weaponCfg("MainHandAttack", activeSkill.Weapon1Flags)
	}

	if skillFlags[SkillFlagWeapon2Attack] {
		activeSkill.Weapon2Cfg = weaponCfg("OffHandAttack", activeSkill.Weapon2Flags)
	}

	// Initialise skill modifier list
	skillModList := moddb.NewModList()
	skillModList.Parent = activeSkill.Actor.ModDB
	activeSkill.SkillModList = skillModList
	activeSkill.BaseSkillModList = skillModList

	//pob:src/Modules/CalcActiveSkill.lua:486-491
	if activeSkill.Actor != nil && activeSkill.Actor.MinionData != nil && activeSkill.Actor.MinionData.DamageFixup != 0 {
		fixup := activeSkill.Actor.MinionData.DamageFixup
		skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, -100*fixup).Source("Damage Fixup").Flag(mod.MFlagAttack))
		skillModList.AddMod(mod.NewFloat("Speed", mod.TypeMore, 100*fixup).Source("Damage Fixup").Flag(mod.MFlagAttack))
	}

	if skillModList.Flag(activeSkill.SkillCfg, "DisableSkill") && !skillModList.Flag(activeSkill.SkillCfg, "EnableSkill") {
		skillFlags[SkillFlagDisable] = true
		activeSkill.DisableReason = "Skills of this type are disabled"
	}

	if skillFlags[SkillFlagDisable] {
		for k := range skillFlags {
			delete(skillFlags, k)
		}
		skillFlags[SkillFlagDisable] = true
		CalcValidateGemLevel(activeEffect)
		activeEffect.GrantedEffectLevel = raw2.GetCalculatedGrantedEffect(activeGrantedEffect.Raw).GetCalculatedLevels()[activeEffect.Level]
		emptyTriggeredGrantedCost(activeEffect)
		return
	}

	// Add support gem modifiers to skill mod list
	for _, skillEffect := range activeSkill.EffectList {
		if skillEffect.GrantedEffect.Raw.IsSupport {
			CalcMergeSkillInstanceMods(env, skillModList, skillEffect, nil)
			level := raw2.GetCalculatedGrantedEffect(skillEffect.GrantedEffect.Raw).GetCalculatedLevels()[skillEffect.Level]
			if level.ManaMultiplier != nil {
				skillModList.AddMod(mod.NewFloat("SupportManaMultiplier", mod.TypeMore, *level.ManaMultiplier).Source(mod.Source("Skill:" + skillEffect.GrantedEffect.Raw.ID)))
			}
			if level.ManaReservationPercent != nil {
				activeSkill.SkillData.ManaReservationPercent = utils.Ptr(*level.ManaReservationPercent)
			}
			if level.StoredUses != nil {
				activeSkill.SkillData.StoredUses = utils.Ptr(*level.StoredUses)
			}
			//pob:src/Modules/CalcActiveSkill.lua:535-537
			if level.SkillUseStorage != nil && activeSkill.SkillData.StoredUses == nil {
				activeSkill.SkillData.StoredUses = utils.Ptr(*level.SkillUseStorage)
			}

			//pob:src/Modules/CalcActiveSkill.lua:519-528
			if !activeSkill.SkillFlags[SkillFlagDisable] && grantedEffectIsTrigger(skillEffect.GrantedEffect) {
				if activeSkill.TriggeredBy != nil {
					activeSkill.SkillFlags[SkillFlagDisable] = true
					activeSkill.DisableReason = "This skill is supported by more than one trigger"
				} else {
					activeSkill.TriggeredBy = skillEffect
				}
			}
			//pob:block:src/Modules/CalcActiveSkill.lua:529-531
			if level.PvPDamageMultiplier != nil {
				skillModList.AddMod(mod.NewFloat("PvpDamageMultiplier", mod.TypeMore, *level.PvPDamageMultiplier).Source(mod.Source("Skill:" + skillEffect.GrantedEffect.Raw.ID)))
			}
		}
	}

	// Apply gem/quality modifiers from support gems.
	//pob:src/Modules/CalcActiveSkill.lua:549-559
	for _, supportProperty := range skillModList.Tabulate(mod.TypeList, activeSkill.SkillCfg, "SupportedGemProperty") {
		castValue := supportProperty.Mod.Value().List().(mod.SupportedGemProperty)
		if castValue.Keyword == "grants_active_skill" && activeSkill.ActiveEffect.GemData != nil &&
			!gemIsSupport(activeSkill.ActiveEffect.GemData) {
			switch castValue.Key {
			case "level":
				activeEffect.Level = activeEffect.Level + castValue.Value
			case "quality":
				activeEffect.Quality = activeEffect.Quality + castValue.Value
				//pob:block:src/Modules/CalcActiveSkill.lua:552-555
				activeEffect.SupportQuality += castValue.Value
			default:
				panic("unknown key: " + castValue.Key)
			}
			//pob:block:src/Modules/CalcActiveSkill.lua:557
			skillModList.AddMod(gemPropertyMod("GemSupport", castValue.Key, float64(castValue.Value), supportProperty.Mod))
		}
	}

	// Add active gem modifiers

	//pob:block:src/Modules/CalcActiveSkill.lua:567
	if activeSkill.Actor != nil && activeSkill.Actor.MinionData != nil {
		actorLevel := activeSkill.Actor.Level
		activeEffect.ActorLevel = &actorLevel
	}

	CalcMergeSkillInstanceMods(env, skillModList, activeEffect, skillModList.List(activeSkill.SkillCfg, "ExtraSkillStat"))
	activeEffect.GrantedEffectLevel = raw2.GetCalculatedGrantedEffect(activeGrantedEffect.Raw).GetCalculatedLevels()[activeEffect.Level]
	emptyTriggeredGrantedCost(activeEffect)

	// Add extra modifiers from granted effect level
	level := activeEffect.GrantedEffectLevel
	if level.CritChance != nil {
		activeSkill.SkillData.CritChance = *level.CritChance
	}

	//pob:src/Modules/CalcActiveSkill.lua:574-576

	if level.AttackTime != nil {
		activeSkill.SkillData.AttackTime = *level.AttackTime
	}

	//pob:src/Modules/CalcActiveSkill.lua:580-581
	if level.AttackSpeedMultiplier != nil {
		activeSkill.SkillData.AttackSpeedMultiplier = *level.AttackSpeedMultiplier
	}

	if level.Cooldown != nil {
		activeSkill.SkillData.Cooldown = *level.Cooldown
	}

	if level.StoredUses != nil {
		activeSkill.SkillData.StoredUses = utils.Ptr(*level.StoredUses)
	}
	//pob:src/Modules/CalcActiveSkill.lua:589-591
	if level.SkillUseStorage != nil && activeSkill.SkillData.StoredUses == nil {
		activeSkill.SkillData.StoredUses = utils.Ptr(*level.SkillUseStorage)
	}

	//pob:block:src/Modules/CalcActiveSkill.lua:592-594
	if level.SoulPreventionDuration != nil {
		activeSkill.SkillData.SoulPreventionDuration = *level.SoulPreventionDuration
	}
	//pob:block:src/Modules/CalcActiveSkill.lua:595-597
	if level.PvPDamageMultiplier != nil {
		skillModList.AddMod(mod.NewFloat("PvpDamageMultiplier", mod.TypeMore, *level.PvPDamageMultiplier).Source(mod.Source("Skill:" + activeGrantedEffect.Raw.ID)))
	}

	// Apply gem/quality modifiers from support gems
	//pob:block:src/Modules/CalcActiveSkill.lua:542-543
	gemLevel, gemQuality := activeEffect.Level, activeEffect.Quality
	if activeEffect.SrcInstance != nil {
		gemLevel, gemQuality = activeEffect.SrcInstance.Level, activeEffect.SrcInstance.Quality
	}
	skillModList.AddMod(mod.NewFloat("GemLevel", mod.TypeBase, float64(gemLevel)).Source("Max Level"))
	skillModList.AddMod(mod.NewFloat("GemQuality", mod.TypeBase, float64(gemQuality)).Source("Max Quality"))
	//pob:block:src/Modules/CalcActiveSkill.lua:545-548
	if activeEffect.SocketQuality != 0 {
		skillModList.AddMod(mod.NewFloat("GemSocketQuality", mod.TypeBase,
			float64(activeEffect.SocketQuality)).Source("Socket Quality"))
	}
	//pob:block:src/Modules/CalcActiveSkill.lua:561-564
	for _, gemProperty := range activeEffect.gemPropertyInfo {
		skillModList.AddMod(gemPropertyMod("GemItem", gemProperty.Key, gemProperty.Value, gemProperty.mod))
	}

	// Add extra modifiers from other sources
	activeSkill.ExtraSkillModList = nil
	for _, value := range utils.CastSlice[mod.ExtraSkillMod](skillModList.List(activeSkill.SkillCfg, "ExtraSkillMod")) {
		skillModList.AddMod(value.Mod)
		activeSkill.ExtraSkillModList = append(activeSkill.ExtraSkillModList, value.Mod)
	}

	// Find totem level
	if skillFlags[SkillFlagTotem] {
		activeSkill.SkillData.TotemLevel = activeEffect.GrantedEffectLevel.LevelRequirement
	}

	// Add active mine multiplier
	//pob:block:src/Modules/CalcActiveSkill.lua:566-572
	if skillFlags[SkillFlagMine] && activeEffect.SrcInstance != nil {
		var mineCount *int
		if env.Mode == OutputModeCalcs {
			mineCount = activeEffect.SrcInstance.SkillMineCountCalcs
		} else {
			mineCount = activeEffect.SrcInstance.SkillMineCount
		}
		if mineCount != nil {
			activeSkill.ActiveMineCount = utils.Ptr(float64(*mineCount))
		}
		if activeSkill.ActiveMineCount != nil && *activeSkill.ActiveMineCount > 0 {
			skillModList.AddMod(mod.NewFloat("Multiplier:ActiveMineCount", mod.TypeBase, *activeSkill.ActiveMineCount).Source("Base"))
			env.EnemyModDB.Multipliers["ActiveMineCount"] = max(*activeSkill.ActiveMineCount, env.EnemyModDB.Multipliers["ActiveMineCount"])
		}
	}

	//pob:block:src/Modules/CalcActiveSkill.lua:635-644
	if stageSkillName := stageMultiplierName(activeGrantedEffect); stageSkillName != "" {
		limit := skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:"+stageSkillName+"MaxStages")
		activeSkill.SkillData.StagesMax = limit
		if limit > 0 {
			skillFlags[SkillFlagMultiStage] = true

			stageCount := float64(0)
			if activeEffect.SrcInstance != nil {
				if env.Mode == OutputModeCalcs {
					stageCount = float64(activeEffect.SrcInstance.SkillStageCountCalcs)
				} else {
					stageCount = float64(activeEffect.SrcInstance.SkillStageCount)
				}
			}
			if stageCount == 0 {
				stageCount = limit
			}

			activeSkill.ActiveStageCount = max(stageCount,
				1+skillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:"+stageSkillName+"MinimumStage"))

			if activeSkill.ActiveStageCount > 0 {
				skillModList.AddMod(mod.NewFloat("Multiplier:"+stageSkillName+"Stage", mod.TypeBase,
					min(limit, activeSkill.ActiveStageCount)).Source("Base"))
				activeSkill.ActiveStageCount--
				skillModList.AddMod(mod.NewFloat("Multiplier:"+stageSkillName+"StageAfterFirst", mod.TypeBase,
					min(limit-1, activeSkill.ActiveStageCount)).Source("Base"))
			}
		}
	}

	// Extract skill data
	for _, value := range utils.CastSlice[mod.SkillData](env.ModDB.List(activeSkill.SkillCfg, "SkillData")) {
		utils.Set(activeSkill.SkillData, value.Key, value.Value)
	}
	for _, value := range utils.CastSlice[mod.SkillData](skillModList.List(activeSkill.SkillCfg, "SkillData")) {
		utils.Set(activeSkill.SkillData, value.Key, value.Value)
	}

	// Create minion
	//pob:src/Modules/CalcActiveSkill.lua:554-573
	var isSpectre bool
	baseMinionList, hasMinionList := raw2.SkillMinionLists[activeGrantedEffect.Raw.ID]
	if hasMinionList && len(baseMinionList) == 0 {
		baseMinionList = spectreList(env)
		isSpectre = true
	}
	minionList := append([]string(nil), baseMinionList...)
	minionSupportLevel := map[string]int{}
	for _, skillEffect := range activeSkill.EffectList {
		if skillEffect.GrantedEffect == nil || !skillEffect.GrantedEffect.Raw.IsSupport {
			continue
		}
		for _, minionType := range raw2.SupportAddMinionLists[skillEffect.GrantedEffect.ID()] {
			if slices.Contains(minionList, minionType) {
				continue
			}
			if lvl := raw2.GetCalculatedGrantedEffect(skillEffect.GrantedEffect.Raw).GetCalculatedLevels()[skillEffect.Level]; lvl != nil {
				minionSupportLevel[minionType] = lvl.LevelRequirement
			}
			minionList = append(minionList, minionType)
		}
	}
	activeSkill.MinionList = minionList

	//pob:src/Modules/CalcActiveSkill.lua:574-595
	if len(minionList) > 0 && activeSkill.Actor.MinionData == nil {
		minionType := minionList[0]
		if activeEffect.SrcInstance != nil {
			//pob:block:src/Modules/CalcActiveSkill.lua:683-692
			selected := activeEffect.SrcInstance.SkillMinion
			if env.Mode == OutputModeCalcs && env.Player != nil && activeSkill == env.Player.MainSkill {
				selected = ""
				if activeEffect.SrcInstance.SkillMinionCalcs != nil {
					selected = *activeEffect.SrcInstance.SkillMinionCalcs
				}
			}
			if index := slices.Index(minionList, selected); index >= 0 {
				minionType = minionList[index]
			}
		}

		if minionData := raw2.Minions[minionType]; minionData != nil {
			minion := &ActorMinion{
				Actor: &Actor{
					Parent:     env.Player,
					Enemy:      env.Enemy,
					MinionData: minionData,
					ItemList:   make(map[string]*ItemData),
				},
				Type: minionType,
				//pob:src/Modules/CalcActiveSkill.lua:717
				LifeTable: data.MonsterAllyLifeTable,
				Uses:      raw2.SkillMinionUses[activeGrantedEffect.Raw.ID],
			}
			//pob:src/Modules/CalcActiveSkill.lua:699-717
			if minionData.Hostile {
				minion.Parent = env.Enemy
				minion.Enemy = env.Player
				minion.LifeTable = data.MonsterLifeTable
			} else {
				minion.LifeTable = minionLifeTable(minion.MinionData.LifeScaling, isSpectre)
			}
			activeSkill.Minion = minion
			skillFlags[SkillFlagHaveMinion] = true

			//	minionLevelIsPlayerLevel and m_min(characterLevel, minionLevelIsPlayerLevel)
			level := activeEffect.GrantedEffectLevel.LevelRequirement
			switch {
			case activeSkill.SkillData.MinionLevelIsEnemyLevel:
				level = env.EnemyLevel
			case activeSkill.SkillData.MinionLevelIsPlayerLevel != 0:
				level = min(env.Player.Level, int(activeSkill.SkillData.MinionLevelIsPlayerLevel))
			case minionSupportLevel[minionType] != 0:
				level = minionSupportLevel[minionType]
			case activeSkill.SkillData.MinionLevel != 0:
				level = int(activeSkill.SkillData.MinionLevel)
			}
			// fix minion level between 1 and 100
			minion.Level = min(max(level, 1), 100)

			attackTime := minionData.AttackTime
			//pob:src/Modules/CalcActiveSkill.lua:720-721
			damageTable := data.MonsterAllyDamageTable
			if isSpectre || minionData.Hostile {
				damageTable = data.MonsterDamageTable
			}
			damage := damageTable[minion.Level] * minionData.Damage
			//pob:src/Modules/CalcActiveSkill.lua:722-724
			if !minionData.BaseDamageIgnoresAttackSpeed {
				damage *= attackTime
			}

			//pob:block:src/Modules/CalcActiveSkill.lua:554-564
			if raw2.SkillMinionHasItemSet[activeGrantedEffect.Raw.ID] && activeEffect.SrcInstance != nil {
				itemSetID := activeEffect.SrcInstance.SkillMinionItemSet
				//pob:block:src/Modules/CalcActiveSkill.lua:726-730
				if env.Mode == OutputModeCalcs && env.Player != nil && activeSkill == env.Player.MainSkill {
					itemSetID = 0
					if activeEffect.SrcInstance.SkillMinionItemSetCalcs != nil {
						itemSetID = *activeEffect.SrcInstance.SkillMinionItemSetCalcs
					}
				}
				minion.ItemSet = minionItemSet(env.Build.Items, itemSetID)
			}

			//pob:src/Modules/CalcActiveSkill.lua:741-744
			switch {
			case activeSkill.SkillData.MinionUseBowAndQuiver && env.Player.WeaponData1 != nil && env.Player.WeaponData1.Type == "Bow",
				activeSkill.SkillData.MinionUseMainHandWeapon && env.Player.WeaponData1 != nil:
				minion.WeaponData1 = env.Player.WeaponData1
			case env.TheIronMass != nil && minionType == "RaisedSkeleton":
				minion.WeaponData1 = env.Player.WeaponData1
			default:
				weaponType := data.ItemClassName(minionData.WeaponType1)
				if weaponType == "" {
					weaponType = "None"
				}
				minion.WeaponData1 = &SkillData{
					Type:        weaponType,
					AttackRate:  1 / attackTime,
					CritChance:  5,
					PhysicalMin: utils.RoundTo(damage*(1-minionData.DamageSpread), 0),
					PhysicalMax: utils.RoundTo(damage*(1+minionData.DamageSpread), 0),
					Range:       minionData.AttackRange,
				}
			}
			minion.WeaponData2 = &SkillData{}

			//pob:block:src/Modules/CalcActiveSkill.lua:756-777
			if minion.Uses != nil {
				if minion.Uses["Weapon 1"] {
					if minion.ItemSet != nil {
						item := peekItemSetSlotItem(env, minion.ItemSet, weaponSetSlotName(minion.ItemSet, "Weapon 1"))
						if item != nil && item.WeaponData != nil {
							minion.WeaponData1 = newWeaponSkillData(data.ItemClassName(item.Type), item.WeaponData[1])
						}
					} else {
						minion.WeaponData1 = env.Player.WeaponData1
					}
				}
				if minion.Uses["Weapon 2"] {
					if minion.ItemSet != nil {
						item := peekItemSetSlotItem(env, minion.ItemSet, weaponSetSlotName(minion.ItemSet, "Weapon 2"))
						if item != nil && item.WeaponData != nil {
							minion.WeaponData2 = newWeaponSkillData(data.ItemClassName(item.Type), item.WeaponData[2])
						}
					} else {
						minion.WeaponData2 = env.Player.WeaponData2
					}
				}
			}
			//pob:block:src/Modules/CalcActiveSkill.lua:778-783
			if !isSpectre && skillModList.Flag(activeSkill.SkillCfg, "NonSpectreMinionsUseParentMainHandAttackTime") &&
				env.Player.WeaponData1 != nil && env.Player.WeaponData1.AttackRate > 0 && minion.WeaponData1 != nil {
				weaponData1 := *minion.WeaponData1
				weaponData1.AttackRate = env.Player.WeaponData1.AttackRate
				minion.WeaponData1 = &weaponData1
			}
		}
	}
	// Separate global effect modifiers (mods that can affect defensive stats or other skills)
	activeGrantedEffectModSource := mod.Source("Skill:" + activeGrantedEffect.Raw.ID)
	effectNameDefault := activeGrantedEffect.Name()

	remaining := make([]mod.Mod, 0, len(skillModList.Mods()))
	for _, skillMod := range skillModList.Mods() {
		var effectType, effectName string
		var effectTag *mod.GlobalEffectTag
		for _, tag := range skillMod.Tags() {
			if get, ok := tag.(*mod.GlobalEffectTag); ok {
				if len(get.GlobalEffectList) > 0 {
					effectType = get.GlobalEffectList[0]
				}
				effectName = get.NameTag
				if effectName == "" {
					effectName = effectNameDefault
				}
				effectTag = get
				break
			}
		}
		if effectTag != nil && effectTag.ModCondTag != "" {
			if cond, _ := skillModList.GetCondition(effectTag.ModCondTag, activeSkill.SkillCfg, false); !cond {
				continue
			}
		}
		if effectType == "" {
			remaining = append(remaining, skillMod)
			continue
		}

		var buff *Buff
		for _, skillBuff := range activeSkill.BuffList {
			if skillBuff.Type == effectType && skillBuff.Name == effectName {
				buff = skillBuff
				break
			}
		}
		if buff == nil {
			buff = &Buff{
				Type:              effectType,
				Name:              effectName,
				Cond:              effectTag.EffectCondTag,
				EnemyCond:         effectTag.EffectEnemyCondTag,
				StackVar:          effectTag.EffectStackVarTag,
				StackLimit:        int(effectTag.EffectStackLimitTag),
				StackLimitVar:     effectTag.EffectStackLimitVarTag,
				ApplyNotPlayer:    effectTag.ApplyNotPlayerTag,
				ApplyMinions:      effectTag.ApplyMinionsTag,
				AllowTotemBuff:    effectTag.AllowTotemBuffTag,
				ModList:           moddb.NewModList(),
				UnscalableModList: moddb.NewModList(),
			}
			if skillMod.GetSource() == activeGrantedEffectModSource {
				// Inherit buff configuration from the active skill
				buff.ActiveSkillBuff = true
				buff.ApplyNotPlayer = buff.ApplyNotPlayer || activeSkill.SkillData.BuffNotPlayer
				buff.ApplyMinions = buff.ApplyMinions || activeSkill.SkillData.BuffMinions
				buff.ApplyAllies = activeSkill.SkillData.BuffAllies
				buff.AllowTotemBuff = activeSkill.SkillData.AllowTotemBuff
			}
			activeSkill.BuffList = append(activeSkill.BuffList, buff)
		}

		destList := buff.ModList
		if effectTag.UnscalableTag {
			destList = buff.UnscalableModList
		}
		match := false
		for _, destMod := range destList.Mods() {
			if moddb.CompareModParams(skillMod, destMod) && (destMod.Type() == mod.TypeBase || destMod.Type() == mod.TypeIncrease) {
				destMod.Value().SetFloat(destMod.Value().Float() + skillMod.Value().Float())
				match = true
				break
			}
		}
		if !match {
			destList.AddMod(skillMod)
		}
	}
	skillModList.SetMods(remaining)

	if len(activeSkill.BuffList) > 0 {
		// Add to auxiliary skill list
		env.AuxSkillList = append(env.AuxSkillList, activeSkill)
	}
}

//pob:src/Modules/CalcActiveSkill.lua:192-223
func getWeaponFlags(env *Environment, weaponData *SkillData, weaponTypes [][]data.ItemClassName) (mod.MFlag, *data.WeaponTypeInfo) {
	if weaponData.Type == "" {
		return 0, nil
	}

	info := data.WeaponTypes[weaponData.Type]

	if info == nil {
		return 0, nil
	}

	for _, types := range weaponTypes {
		if len(types) == 0 || slices.Contains(types, weaponData.Type) {
			continue
		}

		if weaponData.CountsAsAll1H && (slices.Contains(types, data.Claw) ||
			slices.Contains(types, data.Dagger) ||
			slices.Contains(types, data.OneHandAxe) ||
			slices.Contains(types, data.OneHandMace) ||
			slices.Contains(types, data.OneHandSword)) {
			continue
		}

		return 0, info
	}

	flags := info.ModFlag
	if weaponData.CountsAsAll1H {
		flags = mod.MFlagAxe | mod.MFlagClaw | mod.MFlagDagger | mod.MFlagMace | mod.MFlagSword
	}

	if weaponData.Type != "None" {
		flags |= mod.MFlagWeapon
		if info.OneHand {
			flags |= mod.MFlagWeapon1H
		} else {
			flags |= mod.MFlagWeapon2H
		}

		if info.Melee {
			flags |= mod.MFlagWeaponMelee
		} else {
			flags |= mod.MFlagWeaponRanged
		}
	}

	return flags, info
}

var pobOnlyGrantedEffects = map[string]pobOnlySkill{
	"InfernalLegion": {SkillTypes: []data.SkillType{
		data.SkillTypeDamageOverTime, data.SkillTypeCausesBurning,
	}},
	"MinionInstability": {SkillTypes: []data.SkillType{
		data.SkillTypeDamage, data.SkillTypeArea,
	}},
	"DelveMonsterEnfeebleOnHit": {
		SkillTypes: []data.SkillType{
			data.SkillTypeSpell, data.SkillTypeArea, data.SkillTypeDuration,
			data.SkillTypeTrappable, data.SkillTypeTotemable, data.SkillTypeMineable,
			data.SkillTypeMulticastable, data.SkillTypeTriggerable, data.SkillTypeCascadable,
			data.SkillTypeAppliesCurse, data.SkillTypeCanRapidFire, data.SkillTypeAreaSpell,
			data.SkillTypeInstantNoRepeatWhenHeld, data.SkillTypeInstantShiftAttackForLeftMouse,
			data.SkillTypeHex,
		},
		CastTimeMS: 500,
	},
	"EnemyExplode": {
		SkillTypes:           []data.SkillType{data.SkillTypeDamage, data.SkillTypeArea},
		CastTimeDeclaredZero: true,
	},
	"HarvestNessaCrabScreechDebuff": {SkillTypes: []data.SkillType{
		data.SkillTypeSpell, data.SkillTypeArea, data.SkillTypeDuration,
		data.SkillTypeTriggerable, data.SkillTypeAreaSpell,
	}},
}

type pobOnlySkill struct {
	SkillTypes           []data.SkillType
	CastTimeMS           int
	CastTimeDeclaredZero bool
}

var pobOnlySkillShells = sync.OnceValue(func() map[string]*poe.GrantedEffect {
	out := make(map[string]*poe.GrantedEffect, len(pobOnlyGrantedEffects))
	for id, spec := range pobOnlyGrantedEffects {
		out[id] = &poe.GrantedEffect{GrantedEffect: raw.GrantedEffect{
			ID:       id,
			CastTime: spec.CastTimeMS,
			Key:      pobOnlySkillKey,
		}}
	}
	return out
})

const pobOnlySkillKey = -1

func pobOnlySkillEffect(id string) *poe.GrantedEffect {
	return pobOnlySkillShells()[id]
}

func PoBSkillDeclaresNoCastTime(grantedEffect *poe.GrantedEffect) bool {
	if grantedEffect == nil || grantedEffect.Key != pobOnlySkillKey {
		return false
	}
	spec, ok := pobOnlyGrantedEffects[grantedEffect.ID]
	return ok && spec.CastTimeMS == 0 && !spec.CastTimeDeclaredZero
}

func pobOnlySkillTypes(grantedEffect *poe.GrantedEffect) map[data.SkillType]bool {
	if grantedEffect == nil || grantedEffect.Key != pobOnlySkillKey {
		return nil
	}
	declared, ok := pobOnlyGrantedEffects[grantedEffect.ID]
	if !ok {
		return nil
	}
	out := make(map[data.SkillType]bool, len(declared.SkillTypes))
	for _, t := range declared.SkillTypes {
		out[t] = true
	}
	return out
}

var droppedMinionSkills sync.Map

func DroppedMinionSkillIDs() []string {
	out := make([]string, 0, 8)
	droppedMinionSkills.Range(func(k, _ any) bool {
		out = append(out, k.(string))
		return true
	})
	slices.Sort(out)
	return out
}

//pob:src/Modules/CalcActiveSkill.lua:867-919
func CalcCreateMinionSkills(env *Environment, activeSkill *ActiveSkill) {
	activeEffect := activeSkill.ActiveEffect
	minion := activeSkill.Minion
	minionData := minion.MinionData

	minion.ActiveSkillList = nil
	skillIDList := make([]string, 0, len(minionData.SkillList))
	//pob:block:src/Modules/CalcActiveSkill.lua:875-879
	for _, skillID := range minionData.SkillList {
		if _, inPoBData := SkillBaseFlagsFor(skillID); inPoBData {
			skillIDList = append(skillIDList, skillID)
		}
	}
	for _, skill := range utils.CastSlice[mod.ExtraMinionSkill](activeSkill.SkillModList.List(activeSkill.SkillCfg, "ExtraMinionSkill")) {
		skillIDList = append(skillIDList, skill.SkillID)
	}
	if len(skillIDList) == 0 {
		// Not ideal, but let's avoid horrible crashes if a spectre has no skills for some reason
		skillIDList = append(skillIDList, "Melee")
	}

	for _, skillID := range skillIDList {
		lookupID := skillID
		for goID, pobID := range skillFlagIDAliases {
			if pobID == skillID {
				lookupID = goID
				break
			}
		}
		grantedEffect := poe.GrantedEffectByID(lookupID)
		if grantedEffect == nil {
			grantedEffect = pobOnlySkillEffect(skillID)
			if grantedEffect == nil {
				if _, seen := droppedMinionSkills.LoadOrStore(skillID, struct{}{}); !seen {
					slog.Warn(
						"minion skill dropped: no granted effect and no PoB-only shell",
						slog.String("skillID", skillID),
						slog.String("lookupID", lookupID),
					)
				}
				continue
			}
		}

		skillTypes := grantedEffectSkillTypes(grantedEffect)
		for t := range pobOnlySkillTypes(grantedEffect) {
			skillTypes[t] = true
		}
		for _, t := range raw2.SkillHiddenLevels[grantedEffect.ID].SkillTypes {
			skillTypes[data.SkillType(t)] = true
		}

		minionEffect := &GemEffect{
			GrantedEffect: &GrantedEffect{
				Raw:        grantedEffect,
				Parts:      skillPartsFor(grantedEffect.ID),
				SkillTypes: skillTypes,
				Funcs:      skillFuncsFor(grantedEffect.ID),
			},
			Level:   1,
			Quality: 0,
		}
		levels := raw2.GetCalculatedGrantedEffect(grantedEffect).GetCalculatedLevels()
		if len(levels) > 1 {
			for level := 1; level <= len(levels); level++ {
				levelData, ok := levels[level]
				if !ok || levelData.LevelRequirement > minion.Level {
					break
				}
				minionEffect.Level = level
			}
		}

		minionSkill := CreateActiveSkill(minionEffect, activeSkill.SupportList, minion.Actor, nil, activeSkill)
		CalcBuildActiveSkillModList(env, minionSkill)
		minionSkill.SkillFlags[SkillFlagMinion] = true
		minionSkill.SkillFlags[SkillFlagMinionSkill] = true
		minionSkill.SkillFlags[SkillFlagHaveMinion] = true
		minionSkill.SkillFlags[SkillFlagSpectre] = activeSkill.SkillFlags[SkillFlagSpectre]
		minionSkill.SkillData.DamageEffectiveness = 1 + activeSkill.SkillData.MinionDamageEffectiveness/100
		minion.ActiveSkillList = append(minion.ActiveSkillList, minionSkill)
	}

	//pob:src/Modules/CalcActiveSkill.lua:913-921
	skillIndex := 1
	if env.Mode == OutputModeCalcs {
		if activeEffect.SrcInstance != nil && activeEffect.SrcInstance.SkillMinionSkillCalcs != nil {
			skillIndex = *activeEffect.SrcInstance.SkillMinionSkillCalcs
		}
	} else if activeEffect.SrcInstance != nil && activeEffect.SrcInstance.SkillMinionSkill != 0 {
		skillIndex = activeEffect.SrcInstance.SkillMinionSkill
	}
	skillIndex = max(min(skillIndex, len(minion.ActiveSkillList)), 1)
	if len(minion.ActiveSkillList) > 0 {
		minion.MainSkill = minion.ActiveSkillList[skillIndex-1]
	}
}

//pob:block:src/Modules/CalcActiveSkill.lua:732-735
func minionItemSet(items pob.Items, id int) *pob.ItemSet {
	if len(items.ItemSets) == 0 {
		return nil
	}
	for i, set := range items.ItemSets {
		if setID, err := strconv.Atoi(set.ID); err == nil && setID == id {
			return &items.ItemSets[i]
		}
	}
	return &items.ItemSets[0]
}

//pob:block:src/Modules/CalcActiveSkill.lua:759
func weaponSetSlotName(set *pob.ItemSet, slotName string) string {
	if set != nil && set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet {
		return slotName + " Swap"
	}
	return slotName
}

//pob:block:src/Modules/CalcActiveSkill.lua:759
func itemSetSlotItem(env *Environment, set *pob.ItemSet, slotName string) *Item {
	return peekItemSetSlotItem(env, set, slotName).Clone()
}

func peekItemSetSlotItem(env *Environment, set *pob.ItemSet, slotName string) *Item {
	if set == nil {
		return nil
	}
	for _, slot := range set.Slots {
		if slot.Name != slotName || slot.ItemID == 0 {
			continue
		}
		for _, it := range env.Build.Items.Items {
			if it.ID == slot.ItemID {
				return peekItemCached(env.Cache, it.Text)
			}
		}
		return nil
	}
	return nil
}

//pob:block:src/Modules/CalcActiveSkill.lua:717
func minionLifeTable(lifeScaling string, isSpectre bool) []float64 {
	switch lifeScaling {
	case "AltLife1":
		return data.MonsterLifeTable2
	case "AltLife2":
		return data.MonsterLifeTable3
	}
	if isSpectre {
		return data.MonsterLifeTable
	}
	return data.MonsterAllyLifeTable
}

//pob:block:src/Modules/CalcActiveSkill.lua:446-450
func skillDistanceOption(b *pob.PathOfBuilding, name string, declaredDefault float64) float64 {
	if v, ok := b.GetNumberOptionOK(name); ok {
		return v
	}
	for _, p := range b.Config.Placeholders {
		if p.Name == name && p.Number != nil {
			return *p.Number
		}
	}
	return declaredDefault
}
