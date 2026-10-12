package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/utils"
)

type skillsExport struct {
	skillTypeMap   []string
	skillStatScope map[string]string
	gems           map[string]bool
	trueGemNames   map[string]string
	perLevel       map[int][]*poe.GrantedEffectsPerLevel
	statsPerLevel  map[int][]*poe.GrantedEffectStatSetsPerLevel
	qualityStats   map[int][]*poe.GrantedEffectQualityStat
	unknown        []string
}

type skillLevel struct {
	level             int
	values            []string
	extra             map[string]string
	statInterpolation []string
	cost              map[string]string
}

type skillData struct {
	baseFlags           []string
	mods                []string
	levels              []*skillLevel
	stats               []string
	cannotGrantToMinion []string
	constantStats       [][2]string
	qualityStats        [][2]string
	addSkillTypes       []string
	isSupport           bool
}

type skillState struct {
	noGem         bool
	addSkillTypes []string
	skill         *skillData
}

var (
	skillStatScopeRe = regexp.MustCompile(`([\pL\pN_]+) "Metadata/StatDescriptions/([\pL\pN_]+)\.txt"`)
	skillStatCopyRe  = regexp.MustCompile(`copy ([\pL\pN_]+) ([\pL\pN_]+)`)
	luaAlphaRunRe    = regexp.MustCompile(`[A-Za-z]+`)
)

func (e *skillsExport) mapAST(key int) string {
	if key >= 0 && key < len(e.skillTypeMap) {
		return "SkillType." + e.skillTypeMap[key]
	}
	return "SkillType.Unknown" + luaNumStr(float64(key+1))
}

func convertUTF16to8(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		u := make([]uint16, 0, len(b)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
		}
		return string(utf16.Decode(u))
	}
	return strings.TrimPrefix(string(b), "\uFEFF")
}

//pob:block:src/Export/Scripts/skills.lua:1-4
//pob:block:src/Export/Scripts/skills.lua:33-42
func newSkillsExport() (*skillsExport, error) {
	e := &skillsExport{gems: map[string]bool{}, trueGemNames: map[string]string{}, skillStatScope: map[string]string{}}
	for _, row := range poe.ActiveSkillTypes {
		e.skillTypeMap = append(e.skillTypeMap, row.ID)
	}
	b, err := loader.LoadFile(gameDataContext(), raw.LatestVersion, "Metadata/StatDescriptions/skillpopup_stat_filters.txt", assetCache())
	if err != nil {
		return nil, fmt.Errorf("skillpopup_stat_filters.txt: %w", err)
	}
	text := convertUTF16to8(b)
	for _, m := range skillStatScopeRe.FindAllStringSubmatch(text, -1) {
		e.skillStatScope[m[1]] = m[2]
	}
	for _, m := range skillStatCopyRe.FindAllStringSubmatch(text, -1) {
		if scope, ok := e.skillStatScope[m[2]]; ok {
			e.skillStatScope[m[1]] = scope
		} else {
			delete(e.skillStatScope, m[1])
		}
	}
	e.perLevel = map[int][]*poe.GrantedEffectsPerLevel{}
	for _, row := range poe.GrantedEffectsPerLevels {
		e.perLevel[row.GrantedEffect] = append(e.perLevel[row.GrantedEffect], row)
	}
	e.statsPerLevel = map[int][]*poe.GrantedEffectStatSetsPerLevel{}
	for _, row := range poe.GrantedEffectStatSetsPerLevels {
		e.statsPerLevel[row.StatSet] = append(e.statsPerLevel[row.StatSet], row)
	}
	e.qualityStats = map[int][]*poe.GrantedEffectQualityStat{}
	for _, row := range poe.GrantedEffectQualityStats {
		e.qualityStats[row.GrantedEffectsKey] = append(e.qualityStats[row.GrantedEffectsKey], row)
	}
	return e, nil
}

// Same as in Flavour Text exporter.
//
//pob:block:src/Export/Scripts/skills.lua:44-59
func skillCleanAndSplit(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	lines := []string{}
	for _, line := range strings.Split(str, "\n") {
		line = strings.Trim(line, luaSpace)
		if line != "" {
			lines = append(lines, strings.ReplaceAll(line, `"`, `\"`))
		}
	}
	return lines
}

func skillEscapeText(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", `\n`)
	return utils.EscapeGGGString(s)
}

// #skill <GrantedEffectId> [<Display name>]
// Initialises the skill data and emits the skill header
//
//pob:block:src/Export/Scripts/skills.lua:80-438
func (e *skillsExport) skill(st *skillState, args string, out *strings.Builder) {
	grantedID, displayName := args, args
	if m := skillDirectiveRe.FindStringSubmatch(args); m != nil {
		grantedID, displayName = m[1], m[2]
	}
	out.WriteString(`skills["` + grantedID + `"] = {` + "\n")
	granted := poe.GrantedEffectByID(grantedID)
	if granted == nil {
		e.unknown = append(e.unknown, grantedID)
		return
	}
	var gemEffect *poe.GemEffect
	secondaryEffect := false
	for _, g := range poe.GemEffects {
		if g.GrantedEffect == granted.Key {
			gemEffect = g
			break
		}
	}
	if gemEffect == nil {
		for _, g := range poe.GemEffects {
			if g.GrantedEffect2 != nil && *g.GrantedEffect2 == granted.Key {
				gemEffect, secondaryEffect = g, true
				break
			}
		}
	}
	var skillGem *poe.SkillGem
	if gemEffect != nil {
	gems:
		for _, gem := range poe.SkillGems {
			for _, variant := range gem.GetGemVariants() {
				if gemEffect.ID == variant.ID {
					skillGem = gem
					for _, g := range poe.GemEffects {
						if g.ID == gemEffect.ID {
							if g.Name != "" {
								e.trueGemNames[gemEffect.ID] = g.Name
							}
							break
						}
					}
					break gems
				}
			}
		}
	}
	active := granted.GetActiveSkill()
	skill := &skillData{}
	st.skill = skill
	if skillGem != nil && !st.noGem {
		base := poe.BaseItemTypes[skillGem.BaseItemType]
		e.gems[gemEffect.ID] = true
		if granted.IsSupport {
			name := strings.ReplaceAll(base.Name, " Support", "")
			if fullNameGems[base.ID] {
				name = base.Name
			}
			out.WriteString("\tname = \"" + name + "\",\n")
			if len(gemEffect.SupportText) > 0 {
				out.WriteString("\tdescription = \"" + skillEscapeText(gemEffect.SupportText) + "\",\n")
			}
		} else {
			name := active.DisplayedName
			if !secondaryEffect {
				if n, ok := e.trueGemNames[gemEffect.ID]; ok {
					name = n
				}
			}
			out.WriteString("\tname = \"" + name + "\",\n")
			// Hybrid gems (e.g. Vaal gems) use the display name of the active skill e.g. Vaal Summon Skeletons of Sorcery
			out.WriteString("\tbaseTypeName = \"" + active.DisplayedName + "\",\n")
		}
	} else {
		if displayName == args && !granted.IsSupport {
			n, ok := "", false
			if gemEffect != nil {
				n, ok = e.trueGemNames[gemEffect.ID]
			}
			if ok {
				displayName = n
			} else {
				displayName = active.DisplayedName
			}
		}
		out.WriteString("\tname = \"" + displayName + "\",\n")
		out.WriteString("\thidden = true,\n")
	}
	if skillGem != nil {
		if base := poe.BaseItemTypes[skillGem.BaseItemType]; base.FlavourTextKey != nil {
			out.WriteString("\tflavourText = {")
			for _, line := range skillCleanAndSplit(poe.FlavourTexts[*base.FlavourTextKey].Text) {
				out.WriteString(`"` + line + `", `)
			}
			out.WriteString("},\n")
		}
	}
	st.noGem = false
	skill.addSkillTypes = st.addSkillTypes
	st.addSkillTypes = nil
	out.WriteString("\tcolor = " + luaNumStr(float64(granted.Attribute)) + ",\n")
	statSet := (&raw.CalculatedGrantedEffect{GrantedEffect: granted}).GetStatSet()
	if f := float64(float32(statSet.BaseEffectiveness)); f != 1 {
		out.WriteString("\tbaseEffectiveness = " + luaNumStr(f) + ",\n")
	}
	if f := float64(float32(statSet.IncrementalEffectiveness)); f != 0 {
		out.WriteString("\tincrementalEffectiveness = " + luaNumStr(f) + ",\n")
	}
	if granted.IsSupport {
		skill.isSupport = true
		out.WriteString("\tsupport = true,\n")
		out.WriteString("\trequireSkillTypes = { ")
		for _, t := range granted.SupportTypes {
			out.WriteString(e.mapAST(t) + ", ")
		}
		out.WriteString("},\n")
		out.WriteString("\taddSkillTypes = { ")
		isTrigger := false
		for _, t := range granted.AddTypes {
			typeString := e.mapAST(t)
			if typeString == "SkillType.Triggered" {
				isTrigger = true
			}
			out.WriteString(typeString + ", ")
		}
		out.WriteString("},\n")
		out.WriteString("\texcludeSkillTypes = { ")
		for _, t := range granted.ExcludeTypes {
			out.WriteString(e.mapAST(t) + ", ")
		}
		out.WriteString("},\n")
		if isTrigger {
			out.WriteString("\tisTrigger = true,\n")
		}
		if granted.SupportsGemsOnly {
			out.WriteString("\tsupportGemsOnly = true,\n")
		}
		if granted.IgnoreMinionTypes {
			out.WriteString("\tignoreMinionTypes = true,\n")
		}
		if granted.PlusVersionOf != nil {
			out.WriteString("\tplusVersionOf = \"" + poe.GrantedEffects[*granted.PlusVersionOf].ID + "\",\n")
		}
		writeWeaponTypes(granted.WeaponRestrictions, out)
		out.WriteString("\tstatDescriptionScope = \"gem_stat_descriptions\",\n")
	} else {
		if len(active.Description) > 0 {
			out.WriteString("\tdescription = \"" + skillEscapeText(active.Description) + "\",\n")
		}
		out.WriteString("\tskillTypes = { ")
		for _, t := range active.ActiveSkillTypes {
			out.WriteString("[" + e.mapAST(t) + "] = true, ")
		}
		for _, t := range skill.addSkillTypes {
			out.WriteString("[SkillType." + t + "] = true, ")
		}
		out.WriteString("},\n")
		if len(active.MinionActiveSkillTypes) > 0 {
			out.WriteString("\tminionSkillTypes = { ")
			for _, t := range active.MinionActiveSkillTypes {
				out.WriteString("[" + e.mapAST(t) + "] = true, ")
			}
			out.WriteString("},\n")
		}
		writeWeaponTypes(active.WeaponRestrictionItemClassesKeys, out)
		scope, ok := e.skillStatScope[active.ID]
		if !ok {
			scope = "skill_stat_descriptions"
		}
		out.WriteString("\tstatDescriptionScope = \"" + scope + "\",\n")
		if active.SkillTotemID <= 21 {
			out.WriteString("\tskillTotemId = " + luaNumStr(float64(active.SkillTotemID)) + ",\n")
		}
		out.WriteString("\tcastTime = " + luaNumStr(float64(granted.CastTime)/1000) + ",\n")
		if granted.CannotBeSupported {
			out.WriteString("\tcannotBeSupported = true,\n")
		}
	}
	e.levels(granted, statSet, skill)
}

//pob:block:src/Export/Scripts/skills.lua:199-211
func writeWeaponTypes(classes []int, out *strings.Builder) {
	weaponTypes := map[string]bool{}
	for _, class := range classes {
		if t, ok := weaponClassMap[poe.ItemClasses[class].ID]; ok {
			weaponTypes[t] = true
		}
	}
	if len(weaponTypes) == 0 {
		return
	}
	types := make([]string, 0, len(weaponTypes))
	for t := range weaponTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	out.WriteString("\tweaponTypes = {\n")
	for _, t := range types {
		out.WriteString("\t\t[\"" + t + "\"] = true,\n")
	}
	out.WriteString("\t},\n")
}

//pob:block:src/Export/Scripts/skills.lua:256-437
func (e *skillsExport) levels(granted *poe.GrantedEffect, statSet *poe.GrantedEffectStatSet, skill *skillData) {
	statsPerLevel := e.statsPerLevel[statSet.Key]
	perLevel := e.perLevel[granted.Key]
	var statMapOrder []string
	statMap := map[string]int{}
	if len(perLevel) != len(statsPerLevel) && len(perLevel) > 1 && len(statsPerLevel) > 1 {
		fmt.Fprintf(os.Stderr, "UNKNOWN CASE of Level to Stat rows for '%s'\n", granted.ID)
	}
	num := func(f float64) string { return luaNumStr(f) }
	for indx := 1; indx <= max(len(perLevel), len(statsPerLevel)); indx++ {
		levelRow := perLevel[0]
		if indx <= len(perLevel) {
			levelRow = perLevel[indx-1]
		}
		statRow := statsPerLevel[0]
		if indx <= len(statsPerLevel) {
			statRow = statsPerLevel[indx-1]
		}
		level := &skillLevel{extra: map[string]string{}, cost: map[string]string{}}
		if len(perLevel) == 1 {
			level.level = statRow.GemLevel
			level.extra["levelRequirement"] = num(float64(statRow.PlayerLevelReq))
		} else {
			level.level = levelRow.Level
			level.extra["levelRequirement"] = num(float64(levelRow.PlayerLevelReq))
		}
		for i, cost := range levelRow.GetCostTypes() {
			level.cost[cost.ID] = num(float64(levelRow.CostAmounts[i]))
		}
		if levelRow.ManaReservationFlat != 0 {
			level.extra["manaReservationFlat"] = num(float64(levelRow.ManaReservationFlat))
		}
		if levelRow.ManaReservationPercent != 0 {
			level.extra["manaReservationPercent"] = num(float64(levelRow.ManaReservationPercent) / 100)
		}
		if levelRow.LifeReservationFlat != 0 {
			level.extra["lifeReservationFlat"] = num(float64(levelRow.LifeReservationFlat))
		}
		if levelRow.LifeReservationPercent != 0 {
			level.extra["lifeReservationPercent"] = num(float64(levelRow.LifeReservationPercent) / 100)
		}
		if levelRow.CostMultiplier != 100 {
			level.extra["manaMultiplier"] = num(float64(levelRow.CostMultiplier - 100))
		}
		if levelRow.AttackSpeedMultiplier != 0 {
			level.extra["attackSpeedMultiplier"] = num(float64(levelRow.AttackSpeedMultiplier))
		}
		if levelRow.AttackTime != 0 {
			level.extra["attackTime"] = num(float64(levelRow.AttackTime))
		}
		if levelRow.Cooldown != 0 {
			level.extra["cooldown"] = num(float64(levelRow.Cooldown) / 1000)
		}
		if levelRow.PvPDamageMultiplier != 0 {
			level.extra["PvPDamageMultiplier"] = num(float64(levelRow.PvPDamageMultiplier))
		}
		if levelRow.StoredUses != 0 {
			level.extra["storedUses"] = num(float64(levelRow.StoredUses))
		}
		if levelRow.VaalSouls != 0 {
			level.cost["Soul"] = num(float64(levelRow.VaalSouls))
		}
		if levelRow.VaalStoredUses != 0 {
			level.extra["vaalStoredUses"] = num(float64(levelRow.VaalStoredUses))
		}
		if levelRow.SoulGainPreventionDuration != 0 {
			level.extra["soulPreventionDuration"] = num(float64(levelRow.SoulGainPreventionDuration) / 1000)
		}
		// stat based level info
		if statRow.DamageEffectiveness != 0 {
			level.extra["damageEffectiveness"] = num(float64(statRow.DamageEffectiveness)/10000 + 1)
		}
		if statRow.AttackCritChance != 0 {
			level.extra["critChance"] = num(float64(statRow.AttackCritChance) / 100)
		}
		if statRow.OffhandCritChance != 0 {
			level.extra["critChance"] = num(float64(statRow.OffhandCritChance) / 100)
		}
		if statRow.BaseMultiplier != 0 {
			level.extra["baseMultiplier"] = num(float64(statRow.BaseMultiplier)/10000 + 1)
		}
		for _, v := range statRow.StatInterpolations {
			level.statInterpolation = append(level.statInterpolation, num(float64(v)))
		}
		statMapOrderIndex := 1
		addMissing := func(id string) {
			for statMapOrderIndex < len(statMapOrder) && statMapOrder[statMapOrderIndex-1] != id {
				level.values = append(level.values, "0")
				if len(level.statInterpolation) < len(statMapOrder) {
					at := min(statMapOrderIndex-1, len(level.statInterpolation))
					level.statInterpolation = append(level.statInterpolation[:at], append([]string{"0"}, level.statInterpolation[at:]...)...)
				}
				statMapOrderIndex++
			}
		}
		orderAt := func(i int) string {
			if i >= 1 && i <= len(statMapOrder) {
				return statMapOrder[i-1]
			}
			return ""
		}
		for i, stat := range statRow.GetFloatStats() {
			if _, ok := statMap[stat.ID]; !ok || indx == 1 {
				statMap[stat.ID] = len(skill.stats) + 1
				skill.stats = append(skill.stats, stat.ID)
				if indx == 1 {
					statMapOrder = append(statMapOrder, stat.ID)
					if stat.CannotGrantToMinion {
						skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, stat.ID)
					}
				} else {
					fmt.Fprintf(os.Stderr, "%s: stat missing from earlier levels: %s\n", granted.ID, stat.ID)
				}
			} else if orderAt(statMapOrderIndex) != stat.ID {
				// add missing stats
				addMissing(stat.ID)
			}
			statMapOrderIndex++
			base := float64(float32(raw.InterpolationBaseValue(statRow.InterpolationBases[i])))
			level.values = append(level.values, num(float64(float32(statRow.FloatStatsValues[i]))/max(base, 0.00001)))
		}
		for i, stat := range statRow.GetAdditionalStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(skill.stats) + 1
				skill.stats = append(skill.stats, stat.ID)
				if indx == 1 {
					statMapOrder = append(statMapOrder, stat.ID)
					if stat.CannotGrantToMinion {
						skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, stat.ID)
					}
				} else {
					fmt.Fprintf(os.Stderr, "%s: stat missing from earlier levels: %s\n", granted.ID, stat.ID)
				}
			} else if orderAt(statMapOrderIndex) != stat.ID {
				// add missing stats
				addMissing(stat.ID)
			}
			statMapOrderIndex++
			level.values = append(level.values, num(float64(statRow.AdditionalStatsValues[i])))
		}
		for _, stat := range statRow.GetAdditionalBooleanStats() {
			if _, ok := statMap[stat.ID]; !ok {
				statMap[stat.ID] = len(skill.stats) + 1
				skill.stats = append(skill.stats, stat.ID)
				if stat.CannotGrantToMinion {
					skill.cannotGrantToMinion = append(skill.cannotGrantToMinion, stat.ID)
				}
			}
		}
		skill.levels = append(skill.levels, level)
	}
	for _, key := range statSet.ImplicitStats {
		id := poe.Stats[key].ID
		if _, ok := statMap[id]; !ok {
			statMap[id] = len(skill.stats) + 1
			skill.stats = append(skill.stats, id)
		}
	}
	for i, key := range statSet.ConstantStats {
		skill.constantStats = append(skill.constantStats, [2]string{poe.Stats[key].ID, num(float64(statSet.ConstantStatsValues[i]))})
	}
	for _, row := range e.qualityStats[granted.Key] {
		for j, key := range row.StatsKeys {
			if id := poe.Stats[key].ID; id != "dummy_stat_display_nothing" {
				skill.qualityStats = append(skill.qualityStats, [2]string{id, num(float64(row.StatsValuesPermille[j]) / 1000)})
			}
		}
	}
}

// Emits the skill modifiers
//
//pob:block:src/Export/Scripts/skills.lua:456-540
func (e *skillsExport) mods(st *skillState, args string, out *strings.Builder) {
	skill := st.skill
	if !strings.Contains(args, "noBaseFlags") && !skill.isSupport {
		out.WriteString("\tbaseFlags = {\n")
		for _, flag := range skill.baseFlags {
			out.WriteString("\t\t" + flag + " = true,\n")
		}
		out.WriteString("\t},\n")
	}
	if !strings.Contains(args, "noBaseMods") && len(skill.mods) > 0 {
		out.WriteString("\tbaseMods = {\n")
		for _, m := range skill.mods {
			out.WriteString("\t\t" + m + ",\n")
		}
		out.WriteString("\t},\n")
	}
	if !strings.Contains(args, "noQualityStats") && len(skill.qualityStats) > 0 {
		out.WriteString("\tqualityStats = {\n")
		for _, s := range skill.qualityStats {
			out.WriteString("\t\t{ \"" + s[0] + "\", " + s[1] + " },\n")
		}
		out.WriteString("\t},\n")
	}
	if !strings.Contains(args, "noStats") {
		if len(skill.constantStats) > 0 {
			// write out constant stats that don't change per level
			out.WriteString("\tconstantStats = {\n")
			for _, s := range skill.constantStats {
				out.WriteString("\t\t{ \"" + s[0] + "\", " + s[1] + " },\n")
			}
			out.WriteString("\t},\n")
		}
		// write out per level stats
		out.WriteString("\tstats = {\n")
		for _, s := range skill.stats {
			out.WriteString("\t\t\"" + s + "\",\n")
		}
		out.WriteString("\t},\n")
		if len(skill.cannotGrantToMinion) > 0 {
			out.WriteString("\tnotMinionStat = {\n")
			for _, s := range skill.cannotGrantToMinion {
				out.WriteString("\t\t\"" + s + "\",\n")
			}
			out.WriteString("\t},\n")
		}
	}
	if !strings.Contains(args, "noLevels") {
		out.WriteString("\tlevels = {\n")
		for _, level := range skill.levels {
			out.WriteString("\t\t[" + luaNumStr(float64(level.level)) + "] = { ")
			for _, v := range level.values {
				out.WriteString(v + ", ")
			}
			keys := make([]string, 0, len(level.extra))
			for k := range level.extra {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				out.WriteString(k + " = " + level.extra[k] + ", ")
			}
			if len(level.statInterpolation) > 0 {
				out.WriteString("statInterpolation = { ")
				for _, t := range level.statInterpolation {
					out.WriteString(t + ", ")
				}
				out.WriteString("}, ")
			}
			if len(level.cost) > 0 {
				out.WriteString("cost = { ")
				keys = keys[:0]
				for k := range level.cost {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					out.WriteString(k + " = " + level.cost[k] + ", ")
				}
				out.WriteString("}, ")
			}
			out.WriteString("},\n")
		}
		out.WriteString("\t},\n")
	}
	out.WriteString("}")
	st.skill = nil
}

//pob:block:src/Export/Main.lua:112-129
//pob:block:src/Export/Scripts/skills.lua:63-78
//pob:block:src/Export/Scripts/skills.lua:440-454
func (e *skillsExport) processTemplate(pobDir, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Export", "Skills", name+".txt")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return "", fmt.Errorf("template %s: %w", name, err)
	}
	var out strings.Builder
	out.WriteString("-- This file is automatically generated, do not edit!\n")
	st := &skillState{}
	text := strings.TrimSuffix(string(b), "\n")
	for _, line := range strings.Split(text, "\n") {
		m := templateLineRe.FindStringSubmatch(line)
		if m == nil {
			out.WriteString(line + "\n")
			continue
		}
		args := m[2]
		switch m[1] {
		case "noGem":
			st.noGem = true
		case "addSkillTypes":
			st.addSkillTypes = luaAlphaRunRe.FindAllString(args, -1)
			if st.addSkillTypes == nil {
				st.addSkillTypes = []string{}
			}
		case "skill":
			e.skill(st, args, &out)
		case "flags":
			if st.skill == nil {
				return "", fmt.Errorf("template %s: #flags outside a skill", name)
			}
			st.skill.baseFlags = append(st.skill.baseFlags, luaAlphaRunRe.FindAllString(args, -1)...)
		case "baseMod":
			if st.skill == nil {
				return "", fmt.Errorf("template %s: #baseMod outside a skill", name)
			}
			st.skill.mods = append(st.skill.mods, args)
		case "mods":
			if st.skill == nil {
				return "", fmt.Errorf("template %s: #mods outside a skill", name)
			}
			e.mods(st, args, &out)
		default:
			fmt.Fprintf(os.Stderr, "Unknown directive '%s'\n", m[1])
		}
	}
	return out.String(), nil
}

//pob:block:src/Export/Scripts/skills.lua:546-592
func (e *skillsExport) gemsLua() string {
	var out strings.Builder
	out.WriteString("-- This file is automatically generated, do not edit!\n")
	out.WriteString("-- Gem data (c) Grinding Gear Games\n\nreturn {\n")
	for _, skillGem := range poe.SkillGems {
		base := poe.BaseItemTypes[skillGem.BaseItemType]
		for _, gemEffect := range skillGem.GetGemVariants() {
			if !e.gems[gemEffect.ID] {
				continue
			}
			// Some skills have an additional old version that exists in the game files and messes up transfigured gem matching
			delete(e.gems, gemEffect.ID)
			out.WriteString("\t[\"Metadata/Items/Gems/SkillGem" + gemEffect.ID + "\"] = {\n")
			name := strings.ReplaceAll(base.Name, " Support", "")
			if fullNameGems[base.ID] {
				name = base.Name
			} else if n, ok := e.trueGemNames[gemEffect.ID]; ok {
				name = n
			}
			out.WriteString("\t\tname = \"" + name + "\",\n")
			granted := poe.GrantedEffects[gemEffect.GrantedEffect]
			// Hybrid gems (e.g. Vaal gems) use the display name of the active skill e.g. Vaal Summon Skeletons of Sorcery
			if !skillGem.IsSupport {
				out.WriteString("\t\tbaseTypeName = \"" + granted.GetActiveSkill().DisplayedName + "\",\n")
			}
			out.WriteString("\t\tgameId = \"" + base.ID + "\",\n")
			out.WriteString("\t\tvariantId = \"" + gemEffect.ID + "\",\n")
			out.WriteString("\t\tgrantedEffectId = \"" + granted.ID + "\",\n")
			if gemEffect.GrantedEffect2 != nil {
				out.WriteString("\t\tsecondaryGrantedEffectId = \"" + poe.GrantedEffects[*gemEffect.GrantedEffect2].ID + "\",\n")
			}
			if len(gemEffect.SupportName) > 0 {
				out.WriteString("\t\tsecondaryEffectName = \"" + gemEffect.SupportName + "\",\n")
			}
			if skillGem.IsVaalGem {
				out.WriteString("\t\tvaalGem = true,\n")
			}
			var tagNames []string
			out.WriteString("\t\ttags = {\n")
			for _, tag := range gemEffect.GetGemTags() {
				//Remove the words in brackets e.g. [DurationSkill|Duration] -> Duration
				name := utils.EscapeGGGString(tag.Tag)
				out.WriteString("\t\t\t" + tag.ID + " = true,\n")
				if len(name) > 0 {
					tagNames = append(tagNames, name)
				}
			}
			out.WriteString("\t\t},\n")
			out.WriteString("\t\ttagString = \"" + strings.Join(tagNames, ", ") + "\",\n")
			out.WriteString("\t\treqStr = " + luaNumStr(float64(skillGem.Str)) + ",\n")
			out.WriteString("\t\treqDex = " + luaNumStr(float64(skillGem.Dex)) + ",\n")
			out.WriteString("\t\treqInt = " + luaNumStr(float64(skillGem.Int)) + ",\n")
			naturalMaxLevel := len(skillGem.GetLevelProgression())
			if naturalMaxLevel <= 0 {
				naturalMaxLevel = 1
			}
			out.WriteString("\t\tnaturalMaxLevel = " + luaNumStr(float64(naturalMaxLevel)) + ",\n")
			out.WriteString("\t},\n")
		}
	}
	out.WriteString("}")
	return out.String()
}

func dumpSkillsData() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the skill set is the pinned PoB's Export/Skills templates")
	}
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"BaseItemTypes", func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) }},
		{"FlavourText", func() error { return poe.InitializeFlavourTexts(ctx, raw.LatestVersion, cache) }},
		{"ItemClasses", func() error { return poe.InitializeItemClasses(ctx, raw.LatestVersion, cache) }},
		{"ActiveSkillType", func() error { return poe.InitializeActiveSkillTypes(ctx, raw.LatestVersion, cache) }},
		{"ActiveSkills", func() error { return poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache) }},
		{"Stats", func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) }},
		{"CostTypes", func() error { return poe.InitializeCostTypes(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffects", func() error { return poe.InitializeGrantedEffects(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffectsPerLevel", func() error { return poe.InitializeGrantedEffectsPerLevels(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffectStatSets", func() error { return poe.InitializeGrantedEffectStatSets(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffectStatSetsPerLevel", func() error { return poe.InitializeGrantedEffectStatSetsPerLevels(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffectQualityStats", func() error { return poe.InitializeGrantedEffectQualityStats(ctx, raw.LatestVersion, cache) }},
		{"GemTags", func() error { return poe.InitializeGemTags(ctx, raw.LatestVersion, cache) }},
		{"GemEffects", func() error { return poe.InitializeGemEffects(ctx, raw.LatestVersion, cache) }},
		{"SkillGems", func() error { return poe.InitializeSkillGems(ctx, raw.LatestVersion, cache) }},
		{"ItemExperiencePerLevel", func() error { return poe.InitializeItemExperiencePerLevels(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	e, err := newSkillsExport()
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for _, name := range skillTemplates {
		text, err := e.processTemplate(pobDir, name)
		if err != nil {
			return nil, err
		}
		files["Data/Skills/"+name+".lua"] = text
	}
	if len(e.unknown) > 0 {
		return nil, fmt.Errorf("the templates name granted effects go-pob-data does not carry: %s", strings.Join(e.unknown, ", "))
	}
	files["Data/Gems.lua"] = e.gemsLua()
	constants, err := dumpCoreConstantsData()
	if err != nil {
		return nil, err
	}
	files["Data/Misc.lua"] = constants.(map[string]string)["Data/Misc.lua"]
	if dir := os.Getenv("SKILL_LUA_DIR"); dir != "" {
		for path, text := range files {
			p := filepath.Join(dir, path)
			if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil { //nolint:gosec // the caller's directory
				return nil, fmt.Errorf("SKILL_LUA_DIR: %w", err)
			}
			if err := os.WriteFile(p, []byte(text), 0o644); err != nil { //nolint:gosec // the caller's directory
				return nil, fmt.Errorf("SKILL_LUA_DIR: %w", err)
			}
		}
	}
	return files, nil
}
