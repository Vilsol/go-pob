package poe2skills

import (
	"math"
	"slices"
	"strconv"
	"strings"

	game "github.com/Vilsol/go-pob-data/poe2"

	"github.com/Vilsol/go-pob/data/poe2"
)

//pob2:block:src/Export/Scripts/skills.lua:8-10
var fullNameGems = map[string]bool{
	"Metadata/Items/Gems/SupportGemBarrage": true,
}

//pob2:block:src/Export/Scripts/skills.lua:33-51
var weaponClassMap = map[string]string{
	"Claw":                     "Claw",
	"Dagger":                   "Dagger",
	"One Hand Sword":           "One Hand Sword",
	"Thrusting One Hand Sword": "Thrusting One Hand Sword",
	"One Hand Axe":             "One Hand Axe",
	"One Hand Mace":            "One Hand Mace",
	"Bow":                      "Bow",
	"Crossbow":                 "Crossbow",
	"Fishing Rod":              "Fishing Rod",
	"Warstaff":                 "Staff",
	"Two Hand Sword":           "Two Hand Sword",
	"Two Hand Axe":             "Two Hand Axe",
	"Two Hand Mace":            "Two Hand Mace",
	"Unarmed":                  "None",
	"Flail":                    "Flail",
	"Spear":                    "Spear",
	"Talisman":                 "Talisman",
}

// exporter is the state skills.lua's directives share while they read one skill.
type exporter struct {
	t             *game.SkillTables
	flavourText   []game.FlavourText
	gemEffectByID map[string]*game.GemEffect
	ge            *GrantedEffect
	granted       *game.GrantedEffect
	baseStatRow   []*game.GrantedEffectStatSetPerLevel
	baseStatSet   *statSetRow
	levelsByIndex []*Level
	displayName   string
	setIndex      int
}

// statSetRow is a copied GrantedEffectStatSets row: later sets concatenate the first set's lists into it.
type statSetRow struct {
	label                          *int
	implicitStats                  []int
	constantStats                  []int
	constantStatsValues            []int
	removeStats                    []int
	baseEffectiveness              float64
	incrementalEffectiveness       float64
	damageIncrementalEffectiveness float64
	useSetAttackMulti              bool
}

// statRow is a copied GrantedEffectStatSetsPerLevel row. baseStats is nil until a later set builds it;
// the first set's rows have no such column, so PoB stops on a first set that removes a stat.
type statRow struct {
	baseResolvedValues    []float64
	floatStats            []int
	floatStatsValues      []float64
	statInterpolations    []int
	interpolationBases    []int
	additionalStats       []int
	additionalStatsValues []float64
	additionalFlags       []int
	baseStats             []int
	actorLevel            float64
	gemLevel              int
	attackCritChance      int
	offhandCritChance     int
	baseMultiplier        int
}

func ints(v []int) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x)
	}
	return out
}

func copyStatRow(r *game.GrantedEffectStatSetPerLevel) *statRow {
	floats := make([]float64, len(r.FloatStatsValues))
	for i, v := range r.FloatStatsValues {
		floats[i] = datFloat(v)
	}
	return &statRow{
		baseResolvedValues:    ints(r.BaseResolvedValues),
		floatStats:            slices.Clone(r.FloatStats),
		floatStatsValues:      floats,
		statInterpolations:    slices.Clone(r.StatInterpolations),
		interpolationBases:    slices.Clone(r.InterpolationBases),
		additionalStats:       slices.Clone(r.AdditionalStats),
		additionalStatsValues: ints(r.AdditionalStatsValues),
		additionalFlags:       slices.Clone(r.AdditionalFlags),
		actorLevel:            datFloat(r.ActorLevel),
		gemLevel:              r.GemLevel,
		attackCritChance:      r.SpellCritChance,
		offhandCritChance:     r.AttackCritChance,
		baseMultiplier:        r.BaseMultiplier,
	}
}

func newExporter(t *game.SkillTables, flavourText []game.FlavourText) *exporter {
	e := &exporter{t: t, flavourText: flavourText, gemEffectByID: make(map[string]*game.GemEffect, len(t.GemEffects))}
	for i := range t.GemEffects {
		if _, ok := e.gemEffectByID[t.GemEffects[i].ID]; !ok {
			e.gemEffectByID[t.GemEffects[i].ID] = &t.GemEffects[i]
		}
	}
	return e
}

// mapAST is a skill type's name, "Unknown<row>" when ActiveSkillType has no such row; PoB reads either
// through the SkillType enum, so a name Global.lua lacks is nil.
//
//pob2:src/Export/Scripts/skills.lua:12-14
func (e *exporter) mapAST(key int) (string, bool) {
	if key < 0 || key >= len(e.t.ActiveSkillTypes) {
		return "", false
	}
	name := e.t.ActiveSkillTypes[key].ID
	_, ok := poe2.SkillType[name]
	return name, ok
}

func (e *exporter) mapASTList(keys []int) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if name, ok := e.mapAST(k); ok {
			out = append(out, name)
		}
	}
	return out
}

func (e *exporter) statID(key int) string {
	return e.t.Stats[key].ID
}

// exportSkill is one #skill ... #skillEnd block of a skill template, read back as Data.lua loads it.
func (e *exporter) exportSkill(o *poe2.SkillOverlay) *GrantedEffect {
	e.ge = &GrantedEffect{ID: o.ID, Overlay: o}
	e.granted = nil
	e.baseStatRow = nil
	e.baseStatSet = nil
	e.levelsByIndex = nil
	e.setIndex = 1
	if o.NoTemplate {
		e.noTemplate(o)
		return e.ge
	}
	e.skill(o)
	for i := range o.StatSets {
		e.set(&o.StatSets[i])
	}
	return e.ge
}

// skill is the #skill directive.
//
//pob2:src/Export/Scripts/skills.lua:126-473
func (e *exporter) skill(o *poe2.SkillOverlay) {
	ge := e.ge
	t := e.t
	grantedID, displayName := o.ID, o.ID
	if o.DisplayName != "" {
		displayName = o.DisplayName
	}
	granted := t.GrantedEffectByID(grantedID)
	if granted == nil {
		return
	}
	e.granted = granted
	var activeSkill *game.ActiveSkill
	if granted.ActiveSkill != nil {
		activeSkill = &t.ActiveSkills[*granted.ActiveSkill]
	}
	gemEffect := t.GemEffect(granted.Key)
	var skillGem *game.SkillGem
	gemColor := 0
	trueGemName := ""
	if gemEffect != nil {
		skillGem = t.SkillGem(gemEffect.Key)
		if skillGem != nil {
			switch {
			case skillGem.StrengthRequirementPercent >= 50:
				gemColor = 1
			case skillGem.IntelligenceRequirementPercent >= 50:
				gemColor = 3
			case skillGem.DexterityRequirementPercent >= 50:
				gemColor = 2
			default:
				gemColor = 4
			}
			if named := e.gemEffectByID[gemEffect.ID]; named != nil && named.Name != "" {
				trueGemName = named.Name
			}
		}
	}
	gemLevels := len(t.PerLevel(granted.Key))
	gemPath := skillGem != nil && !o.Hidden
	if gemPath {
		baseItem := t.BaseItemTypes[skillGem.BaseItemType]
		if granted.IsSupport {
			if fullNameGems[baseItem.ID] {
				e.displayName = baseItem.Name
			} else {
				e.displayName = strings.ReplaceAll(baseItem.Name, " Support", "")
			}
			e.displayName = sanitiseText(e.displayName)
			ge.Name = e.displayName
			if gemEffect.SupportText != "" {
				ge.Description = escapeGGGString(strings.ReplaceAll(gemEffect.SupportText, "\r", ""))
			}
			gemLevels = 1
		} else {
			e.displayName = activeSkill.DisplayedName
			if trueGemName != "" {
				e.displayName = trueGemName
			}
			ge.Name = sanitiseText(e.displayName)
			ge.BaseTypeName, ge.HasBaseTypeName = activeSkill.DisplayedName, true
		}
	} else {
		if displayName == grantedID && !granted.IsSupport {
			switch {
			case gemEffect != nil && trueGemName != "":
				displayName = trueGemName
			case activeSkill != nil:
				displayName = activeSkill.DisplayedName
			}
		}
		e.displayName = sanitiseText(displayName)
		ge.Name = displayName
		ge.Hidden = true
	}
	if activeSkill != nil {
		ge.Icon, ge.HasIcon = activeSkill.IconDDSFile, true
	}
	// #noGem is cleared before the colour and the level progression read it.
	if skillGem != nil {
		ge.Color = gemColor
	}

	perLevel := t.PerLevel(granted.Key)
	statsPerLevel := t.LevelStatRows(granted.Key, gemLevels)
	var gemLevelProgression []*game.ItemExperiencePerLevel
	if skillGem != nil {
		gemLevelProgression = t.GemLevelProgression(skillGem.Key)
	}
	ge.Levels = map[int]*Level{}
	nextGemLevelReqValue := 0.0
	e.baseStatRow = make([]*game.GrantedEffectStatSetPerLevel, gemLevels)
	for indx := range gemLevels {
		if indx >= len(perLevel) {
			break
		}
		levelRow := perLevel[indx]
		var sr *game.GrantedEffectStatSetPerLevel
		if indx < len(statsPerLevel) {
			sr = statsPerLevel[indx]
		}
		e.baseStatRow[indx] = sr
		level := &Level{Level: levelRow.Level, Extra: map[string]float64{}, Cost: map[string]float64{}}
		req := 0.0
		if indx < len(gemLevelProgression) {
			req = float64(gemLevelProgression[indx].Level)
		}
		level.Extra["levelRequirement"] = math.Max(req, nextGemLevelReqValue)
		nextGemLevelReqValue = level.Extra["levelRequirement"]
		for i, cost := range granted.CostTypes {
			if i < len(levelRow.CostAmounts) {
				level.Cost[t.CostTypes[cost].ID] = float64(levelRow.CostAmounts[i])
			}
		}
		if levelRow.Reservation != 0 {
			level.Extra["spiritReservationFlat"] = float64(levelRow.Reservation)
		}
		if levelRow.EffectOnPlayer != 100 {
			level.Extra["reservationMultiplier"] = float64(levelRow.EffectOnPlayer - 100)
		}
		if levelRow.CostMultiplier != 100 {
			level.Extra["manaMultiplier"] = float64(levelRow.CostMultiplier - 100)
		}
		if levelRow.AttackSpeedMultiplier != 0 {
			level.Extra["attackSpeedMultiplier"] = float64(levelRow.AttackSpeedMultiplier)
		}
		if levelRow.AttackTime != 0 {
			level.Extra["attackTime"] = float64(levelRow.AttackTime)
		}
		if levelRow.Cooldown != 0 {
			level.Extra["cooldown"] = luaNumber(float64(levelRow.Cooldown) / 1000)
		}
		if levelRow.PvPDamageMultiplier != 0 {
			level.Extra["PvPDamageMultiplier"] = float64(levelRow.PvPDamageMultiplier)
		}
		if levelRow.StoredUses != 0 {
			level.Extra["storedUses"] = float64(levelRow.StoredUses)
		}
		if sr != nil && sr.SpellCritChance != 0 {
			level.Extra["critChance"] = luaNumber(float64(sr.SpellCritChance) / 100)
		}
		if sr != nil && sr.AttackCritChance != 0 {
			level.Extra["critChance"] = luaNumber(float64(sr.AttackCritChance) / 100)
		}
		if sr != nil && sr.BaseMultiplier != 0 {
			level.Extra["baseMultiplier"] = luaNumber(float64(sr.BaseMultiplier)/10000 + 1)
		}
		if levelRow.VaalSouls != 0 {
			level.Cost["Soul"] = float64(levelRow.VaalSouls)
		}
		if levelRow.VaalStoredUses != 0 {
			level.Extra["vaalStoredUses"] = float64(levelRow.VaalStoredUses)
		}
		if levelRow.SoulGainPreventionDuration != 0 {
			level.Extra["soulPreventionDuration"] = luaNumber(float64(levelRow.SoulGainPreventionDuration) / 1000)
		}
		ge.Levels[level.Level] = level
		e.levelsByIndex = append(e.levelsByIndex, level)
	}
	if !(skillGem != nil && granted.IsSupport) {
		ge.QualityStats = []QualityStat{}
		ge.AltQualityStats = []QualityStat{}
		if q := t.QualityStats(granted.Key); q != nil {
			for i, stat := range q.Stats {
				ge.QualityStats = append(ge.QualityStats, QualityStat{Stat: e.statID(stat), Value: luaNumber(float64(q.StatsValuesPermille[i]) / 1000), StatSets: ints(q.ApplyToStatSets)})
			}
			for i, stat := range q.AltStats {
				ge.AltQualityStats = append(ge.AltQualityStats, QualityStat{Stat: e.statID(stat), Value: luaNumber(float64(q.AltStatValuesPermille[i]) / 1000), StatSets: ints(q.AltApplyToStatSets)})
			}
		}
	}
	if granted.IsSupport {
		ge.Support = true
		ge.RequireSkillTypes = e.mapASTList(granted.AllowedActiveSkillTypes)
		ge.AddSkillTypes = e.mapASTList(granted.AddedActiveSkillTypes)
		for _, k := range granted.AddedActiveSkillTypes {
			if name, _ := e.mapAST(k); name == "Triggered" {
				ge.IsTrigger = true
			}
		}
		ge.ExcludeSkillTypes = e.mapASTList(granted.ExcludedActiveSkillTypes)
		if skillGem != nil {
			if supportGem := t.SupportGem(skillGem.Key); supportGem != nil {
				for _, f := range supportGem.Family {
					ge.GemFamily = append(ge.GemFamily, t.SupportGemFamilies[f].ID)
				}
				if supportGem.IsLineage && supportGem.FlavourText != nil {
					ge.IsLineage = true
					ge.FlavourText = cleanAndSplit(e.flavourText[*supportGem.FlavourText].Text)
				}
			}
		}
		ge.SupportGemsOnly = granted.SupportsGemsOnly
		ge.IgnoreMinionTypes = granted.IgnoreMinionTypes
	} else {
		if activeSkill.Description != "" {
			ge.Description = escapeGGGString(strings.ReplaceAll(activeSkill.Description, "\r", ""))
		}
		ge.SkillTypes = map[string]bool{}
		for _, k := range activeSkill.ActiveSkillTypes {
			if name, ok := e.mapAST(k); ok {
				ge.SkillTypes[name] = true
			}
		}
		for _, name := range o.AddSkillTypes {
			if _, ok := poe2.SkillType[name]; ok {
				ge.SkillTypes[name] = true
			}
		}
		if len(activeSkill.MinionActiveSkillTypes) > 0 {
			ge.MinionSkillTypes = map[string]bool{}
			for _, k := range activeSkill.MinionActiveSkillTypes {
				if name, ok := e.mapAST(k); ok {
					ge.MinionSkillTypes[name] = true
				}
			}
		}
		for _, class := range t.WeaponClasses(*granted.ActiveSkill) {
			if wt, ok := weaponClassMap[class.ID]; ok {
				if ge.WeaponTypes == nil {
					ge.WeaponTypes = map[string]bool{}
				}
				ge.WeaponTypes[wt] = true
			}
		}
		if activeSkill.SkillTotemID < 25 {
			ge.SkillTotemID, ge.HasSkillTotemID = activeSkill.SkillTotemID, true
		}
		ge.CastTime, ge.HasCastTime = luaNumber(float64(granted.CastTime)/1000), true
		ge.CannotBeSupported = granted.CannotBeSupported
	}
}

// set is the #set directive and the #mods directive that closes it.
//
//pob2:src/Export/Scripts/skills.lua:487-780
func (e *exporter) set(so *poe2.StatSetOverlay) {
	t := e.t
	original := t.StatSetByID(so.ID)
	if original == nil {
		return
	}
	g := &statSetRow{
		label:                          original.Label,
		implicitStats:                  slices.Clone(original.ImplicitStats),
		constantStats:                  slices.Clone(original.ConstantStats),
		constantStatsValues:            slices.Clone(original.ConstantStatsValues),
		removeStats:                    slices.Clone(original.IgnoredStats),
		baseEffectiveness:              datFloat(original.BaseEffectiveness),
		incrementalEffectiveness:       datFloat(original.IncrementalEffectiveness),
		damageIncrementalEffectiveness: datFloat(original.DamageIncrementalEffectiveness),
		useSetAttackMulti:              original.UseSetAttackMulti,
	}
	perLevelRows := t.StatSetPerLevel(original.Key)
	statsPerLevel := make([]*statRow, len(perLevelRows))
	for i, r := range perLevelRows {
		statsPerLevel[i] = copyStatRow(r)
	}
	label := e.displayName
	if g.label != nil {
		label = t.GrantedEffectLabels[*g.label].Text
	}
	label = sanitiseText(label)
	set := &StatSet{effect: e.ge, Overlay: so, Levels: map[int]*StatSetLevel{}}
	e.ge.StatSets = append(e.ge.StatSets, set)
	removeStats := make([]string, len(g.removeStats))
	for k, v := range g.removeStats {
		removeStats[k] = e.statID(v)
	}

	if e.setIndex == 1 {
		e.baseStatSet = g
	} else {
		base := e.baseStatSet
		g.implicitStats = concat(base.implicitStats, g.implicitStats)
		g.constantStats = concat(base.constantStats, g.constantStats)
		g.constantStatsValues = concat(base.constantStatsValues, g.constantStatsValues)
		if g.baseEffectiveness == 1 {
			g.baseEffectiveness = base.baseEffectiveness
		}
		if g.incrementalEffectiveness == 0 {
			g.incrementalEffectiveness = base.incrementalEffectiveness
		}
		if g.damageIncrementalEffectiveness == 0 {
			g.damageIncrementalEffectiveness = base.damageIncrementalEffectiveness
		}
	}

	statMap := map[string]bool{}
	var statMapOrder []string
	var cannotGrantToMinion []string
	addStat := func(key int, first bool) {
		id := e.statID(key)
		set.Stats = append(set.Stats, id)
		statMap[id] = true
		if first {
			statMapOrder = append(statMapOrder, id)
			if t.Stats[key].CannotGrantToMinion && !slices.Contains(cannotGrantToMinion, id) {
				cannotGrantToMinion = append(cannotGrantToMinion, id)
			}
		}
	}

	for indx, sr := range statsPerLevel {
		var baseStatRow *game.GrantedEffectStatSetPerLevel
		if indx < len(e.baseStatRow) {
			baseStatRow = e.baseStatRow[indx]
		}
		level := &StatSetLevel{Extra: map[string]float64{}, ActorLevel: 1, HasActorLevel: true}
		level.Level = sr.gemLevel
		if e.setIndex != 1 && baseStatRow != nil {
			if sr.attackCritChance != 0 {
				level.Extra["critChance"] = luaNumber(float64(baseStatRow.SpellCritChance+sr.attackCritChance) / 100)
			}
			if sr.offhandCritChance != 0 {
				level.Extra["critChance"] = luaNumber(float64(baseStatRow.AttackCritChance+sr.offhandCritChance) / 100)
			}
		}
		if e.setIndex != 1 && sr.baseMultiplier != 0 {
			level.Extra["baseMultiplier"] = luaNumber(float64(sr.baseMultiplier)/10000 + 1)
		}
		if e.setIndex != 1 {
			var b *statRow
			if baseStatRow != nil {
				b = copyStatRow(baseStatRow)
			} else {
				b = &statRow{}
			}
			base := e.baseStatSet
			sr.baseResolvedValues = concat(b.baseResolvedValues, sr.baseResolvedValues)
			sr.floatStats = concat(b.floatStats, sr.floatStats)
			sr.floatStatsValues = concat(b.floatStatsValues, sr.floatStatsValues)
			sr.statInterpolations = concat(b.statInterpolations, sr.statInterpolations)
			sr.interpolationBases = concat(b.interpolationBases, sr.interpolationBases)
			sr.additionalStats = concat(b.additionalStats, sr.additionalStats)
			sr.additionalStatsValues = concat(b.additionalStatsValues, sr.additionalStatsValues)
			sr.baseStats = concat(concat(concat(base.implicitStats, base.constantStats), b.floatStats), b.additionalStats)
		}
		si := luaArr{}
		for i, v := range sr.statInterpolations {
			si[i+1] = float64(v)
		}
		level.ActorLevel = luaNumber(sr.actorLevel)
		// Removing while iterating skips the stat that moves into the removed slot, as PoB's pairs does.
		tempRemoveStats := slices.Clone(removeStats)
		for i := 0; i < len(removeStats); i++ {
			remove := false
			for _, stat := range sr.baseStats {
				if e.statID(stat) == removeStats[i] {
					remove = true
				}
			}
			if !remove {
				tempRemoveStats = slices.Delete(tempRemoveStats, i, i+1)
				removeStats = slices.Delete(removeStats, i, i+1)
			}
		}
		takeRemoved := func(id string) bool {
			for k, v := range tempRemoveStats {
				if v == id {
					tempRemoveStats = slices.Delete(tempRemoveStats, k, k+1)
					return true
				}
			}
			return false
		}
		statMapOrderIndex := 1
		addMissing := func(id string) {
			for statMapOrderIndex < len(statMapOrder) && statMapOrder[statMapOrderIndex-1] != id {
				level.Values = append(level.Values, 0)
				if si.border() < len(statMapOrder) {
					si.insertAt(statMapOrderIndex, 0)
				}
				statMapOrderIndex++
			}
		}
		orderAt := func(i int) string {
			if i-1 < len(statMapOrder) {
				return statMapOrder[i-1]
			}
			return ""
		}
		for i, key := range sr.floatStats {
			id := e.statID(key)
			if takeRemoved(id) && i < len(sr.baseResolvedValues) {
				sr.baseResolvedValues[i] = 0
			}
			if !statMap[id] || indx == 0 {
				addStat(key, indx == 0)
			} else if orderAt(statMapOrderIndex) != id {
				addMissing(id)
			}
			if len(statsPerLevel) > 5 {
				if i < len(sr.baseResolvedValues) {
					level.Values = append(level.Values, sr.baseResolvedValues[i])
				}
				if e.setIndex != 1 {
					si[len(level.Values)] = 1
				} else {
					si[statMapOrderIndex] = 1
				}
			} else {
				div := 0.00001
				if i < len(sr.interpolationBases) {
					div = math.Max(datFloat(t.EffectivenessCostConstants[sr.interpolationBases[i]].Multiplier), 0.00001)
				}
				if i < len(sr.floatStatsValues) {
					level.Values = append(level.Values, luaNumber(sr.floatStatsValues[i]/div))
				}
			}
			statMapOrderIndex++
		}
		for i, key := range sr.additionalStats {
			id := e.statID(key)
			if takeRemoved(id) && i < len(sr.additionalStatsValues) {
				sr.additionalStatsValues[i] = 0
			}
			if !statMap[id] || indx == 0 {
				addStat(key, indx == 0)
			} else if orderAt(statMapOrderIndex) != id {
				addMissing(id)
			}
			if i < len(sr.additionalStatsValues) {
				level.Values = append(level.Values, sr.additionalStatsValues[i])
			}
			si[statMapOrderIndex] = 1
			statMapOrderIndex++
		}
		for _, key := range sr.additionalFlags {
			id := e.statID(key)
			if takeRemoved(id) {
				continue
			}
			if !statMap[id] {
				set.Stats = append(set.Stats, id)
				statMap[id] = true
				if t.Stats[key].CannotGrantToMinion && !slices.Contains(cannotGrantToMinion, id) {
					cannotGrantToMinion = append(cannotGrantToMinion, id)
				}
			}
		}
		if len(si) > 0 {
			level.StatInterpolation = si.list()
		}
		set.Levels[level.Level] = level
	}
	takeSetRemoved := func(id string) bool {
		for k, v := range removeStats {
			if v == id {
				removeStats = slices.Delete(removeStats, k, k+1)
				return true
			}
		}
		return false
	}
	for _, key := range g.implicitStats {
		id := e.statID(key)
		if !takeSetRemoved(id) && !statMap[id] {
			set.Stats = append(set.Stats, id)
			statMap[id] = true
		}
	}
	for i, key := range g.constantStats {
		id := e.statID(key)
		if !takeSetRemoved(id) && i < len(g.constantStatsValues) {
			set.ConstantStats = append(set.ConstantStats, ConstantStat{Stat: id, Value: float64(g.constantStatsValues[i])})
		}
	}
	set.NotMinionStat = cannotGrantToMinion

	set.Label, set.HasLabel = label, true
	if g.baseEffectiveness != 1 {
		set.BaseEffectiveness, set.HasBaseEffectiveness = luaNumber(g.baseEffectiveness), true
	}
	if g.incrementalEffectiveness != 0 {
		set.IncrementalEffectiveness, set.HasIncrementalEffectiveness = luaNumber(g.incrementalEffectiveness), true
	}
	if g.damageIncrementalEffectiveness != 0 {
		set.DamageIncrementalEffectiveness, set.HasDamageIncrementalEffect = luaNumber(g.damageIncrementalEffectiveness), true
	}
	if e.granted.IsSupport {
		set.StatDescriptionScope = "gem_stat_descriptions"
		if adding := t.GemEffectsAdding(e.granted.Key); len(adding) > 0 {
			for _, tag := range adding[0].GemTags {
				if t.GemTags[tag].ID == "meta" {
					set.StatDescriptionScope = "meta_gem_stat_descriptions"
				}
			}
		}
	} else {
		scope := t.ActiveSkills[*e.granted.ActiveSkill].StatDescription
		scope = strings.TrimPrefix(scope, "Data/StatDescriptions/")
		scope = strings.ReplaceAll(scope, "specific_skill_stat_descriptions/", "")
		scope = strings.ReplaceAll(scope, "statset_0", "statset_"+strconv.Itoa(e.setIndex-1))
		scope = strings.TrimSuffix(scope, "/")
		scope = strings.ReplaceAll(scope, "/", "_")
		scope = csdSuffix.ReplaceAllString(scope, "")
		set.StatDescriptionScope = scope
	}
	set.HasStatDescriptionScope = true
	//pob2:block:src/Export/Scripts/skills.lua:839-884
	set.HasStats = true
	if set.Stats == nil {
		set.Stats = []string{}
	}
	e.setIndex++
}

// noTemplate is an effect PoB writes straight into its skill files: nothing of it is read from the game.
func (e *exporter) noTemplate(o *poe2.SkillOverlay) {
	for i := range o.StatSets {
		e.ge.StatSets = append(e.ge.StatSets, &StatSet{effect: e.ge, Overlay: &o.StatSets[i]})
	}
}

func concat[T any](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	out = append(out, a...)
	return append(out, b...)
}

// luaArr is a Lua array that may have holes: # is the length of its contiguous prefix.
type luaArr map[int]float64

func (a luaArr) border() int {
	n := 0
	for {
		if _, ok := a[n+1]; !ok {
			return n
		}
		n++
	}
}

// insertAt is table.insert(t, pos, v) as LuaJIT runs it.
func (a luaArr) insertAt(pos int, v float64) {
	for i := a.border() + 1; i > pos; i-- {
		if prev, ok := a[i-1]; ok {
			a[i] = prev
		} else {
			delete(a, i)
		}
	}
	a[pos] = v
}

// list is what ipairs reads.
func (a luaArr) list() []float64 {
	n := a.border()
	out := make([]float64, n)
	for i := range n {
		out[i] = a[i+1]
	}
	return out
}
