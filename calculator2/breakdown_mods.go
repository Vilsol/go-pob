package calculator2

import (
	"math"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// ModSectionData is the part of a CalcSections.lua cell entry AddModSection reads; Multi is a table modName.
type ModSectionData struct {
	ModName   []string
	Multi     bool
	ModType   string
	Cfg       string
	ModSource string
	Enemy     bool
	Actor     string
}

// ModSectionRow is one row of a mod table as CalcBreakdownControl draws it. Totals is the source tooltip's lines,
// nil where the row has none; ItemID and NodeID are the item or passive node the source names.
type ModSectionRow struct {
	Value      string
	Name       string
	Flags      string
	Tags       string
	Source     string
	SourceName string
	Totals     []string
	ItemID     int
	NodeID     int
}

var (
	modNameCaseRe   = regexp.MustCompile(`([a-z0-9]:?)([A-Z])`)
	modNameDigitRe  = regexp.MustCompile(`([a-z])([0-9])`)
	modNameSpacedRe = regexp.MustCompile(`([a-z])([A-Z])`)
	itemSourceRe    = regexp.MustCompile(`^Item:(\d+):.`)
	treeSourceRe    = regexp.MustCompile(`^Tree:(\d+)`)
)

//pob2:src/Classes/CalcBreakdownControl.lua:550-552
func formatModName(modName string) string {
	return modNameDigitRe.ReplaceAllString(modNameCaseRe.ReplaceAllString(modName, "$1 $2"), "$1 $2")
}

//pob2:src/Classes/CalcBreakdownControl.lua:554-556
func formatVarNameOrList(varList []string) string {
	return formatModName(strings.Join(varList, " / "))
}

//pob2:src/Classes/CalcBreakdownControl.lua:558-560
func formatModBase(m mod.Mod, base float64) string {
	if m.Type() == mod.TypeBase {
		return sFormat("%+g", math.Abs(base))
	}
	return luaToString(math.Abs(base)) + "%"
}

// formatModValue takes a LIST value as list; a FLAG value is true.
//
//pob2:src/Classes/CalcBreakdownControl.lua:562-590
func formatModValue(value float64, list any, modType mod.Type) string {
	switch modType {
	case mod.TypeBase:
		return sFormat("%+g base", value)
	case mod.TypeIncrease:
		if value >= 0 {
			return luaToString(value) + "% increased"
		}
		return luaToString(-value) + "% reduced"
	case mod.TypeMore:
		if value >= 0 {
			return luaToString(value) + "% more"
		}
		return luaToString(-value) + "% less"
	case mod.TypeOverride:
		return "Override: " + luaToString(value)
	case mod.TypeFlag:
		return "True"
	case mod.TypeList:
		if m := listValueMod(list); m != nil {
			return "Modifier: " + formatModName(m.Name())
		}
		return "?"
	}
	return luaToString(value)
}

// listValueMod is value.mod of a LIST value.
func listValueMod(list any) mod.Mod {
	v := reflect.ValueOf(list)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	f := v.FieldByName("Mod")
	if !f.IsValid() || !f.CanInterface() {
		return nil
	}
	m, _ := f.Interface().(mod.Mod)
	return m
}

// mainSkillCfg is actor.mainSkill[cfg.."Cfg"].
func (a *ActiveSkill) mainSkillCfg(name string) *moddb.ListCfg {
	switch name {
	case "skill":
		return a.SkillCfg
	case "weapon1":
		return a.Weapon1Cfg
	case "weapon2":
		return a.Weapon2Cfg
	case "dot":
		return a.DotCfg
	case "decay":
		return a.DecayCfg
	case "dotPhysical", "dotLightning", "dotCold", "dotFire", "dotChaos":
		return a.DotTypeCfg[name[len("dot"):]]
	}
	return a.AilmentCfg[name+"Cfg"]
}

func modIsNumeric(m mod.Mod) bool {
	return m.Type() != mod.TypeFlag && m.Type() != mod.TypeList
}

//pob2:block:src/Classes/CalcBreakdownControl.lua:471-483
func modFlagText(m mod.Mod) string {
	flags, keywordFlags := int64(m.Flags()), int64(m.KeywordFlags())
	if flags == 0 && keywordFlags == 0 {
		return ""
	}
	var names []string
	add := func(bits int64, keyword bool) {
		if keyword {
			for name, val := range mod.KeywordFlagsByName {
				if bits&int64(val) == int64(val) {
					names = append(names, name)
				}
			}
			return
		}
		for name, val := range modFlagByName {
			if bits&int64(val) == int64(val) {
				names = append(names, name)
			}
		}
	}
	// { [flags] = ModFlag, [keywordFlags] = KeywordFlag } holds one entry where the two are equal
	if flags != keywordFlags {
		add(flags, false)
	}
	add(keywordFlags, true)
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func conditionText(neg bool, vars []string) string {
	if len(vars) == 0 {
		return ""
	}
	if neg {
		return ": Not " + formatVarNameOrList(vars)
	}
	return ": " + formatVarNameOrList(vars)
}

func notText(neg bool) string {
	if neg {
		return "Not "
	}
	return ""
}

// modTagText is the row's Notes cell.
//
//pob2:block:src/Classes/CalcBreakdownControl.lua:484-546
//nolint:gocyclo // Mirrors the Lua elseif chain.
func (env *Environment) modTagText(m mod.Mod) string {
	tags := m.Tags()
	if len(tags) == 0 {
		return ""
	}
	value, numeric := 0.0, false
	if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
		value, numeric = v.Float(), true
	}
	baseVal := ""
	if numeric {
		baseVal = formatModBase(m, value) + " "
	}
	var out []string
	for _, tag := range tags {
		var desc string
		switch t := tag.(type) {
		case *mod.ConditionTag:
			desc = "Condition" + conditionText(t.Negative, t.VarList)
		case *mod.ActorConditionTag:
			actor := ""
			if t.Actor != nil && *t.Actor != "" {
				actor = strings.ToUpper((*t.Actor)[:1]) + (*t.Actor)[1:] + " "
			}
			desc = actor + "Condition" + conditionText(t.Negative, t.VariableList)
		case *mod.MultiplierTag:
			base := baseVal
			if t.TagBase != 0 {
				base = formatModBase(m, t.TagBase) + " + " + luaToString(math.Abs(value)) + " "
			}
			div := ""
			if t.DivisionSet {
				div = luaToString(t.Division) + " "
			}
			desc = base + "per " + div + formatVarNameOrList(t.VariableList)
			baseVal = ""
		case *mod.PerStatTag:
			base := baseVal
			if t.Base != 0 {
				base = formatModBase(m, t.Base) + " + " + luaToString(math.Abs(value)) + " "
			}
			div := t.Divide
			if div == 0 {
				div = 1
			}
			actor := ""
			if t.TagActor != "" {
				actor = t.TagActor + " "
			}
			desc = base + "per " + luaToString(div) + " " + actor + formatVarNameOrList(t.StatList)
			baseVal = ""
		case *mod.PercentStatTag:
			percent := t.Percent
			if percent == 0 {
				percent = 1
			}
			finalPercent := (value * (percent / 100)) * 100
			actor := ""
			if t.TagActor != "" {
				actor = t.TagActor + " "
			}
			stats := t.StatList
			if t.PercentVar != "" {
				stats = []string{t.PercentVar}
			} else if t.Stat != "" {
				stats = []string{t.Stat}
			}
			desc = formatModBase(m, finalPercent) + "% of " + actor + formatVarNameOrList(stats)
			baseVal = ""
		case *mod.MultiplierThresholdTag:
			vars := t.VariableList
			if t.Variable != "" {
				vars = []string{t.Variable}
			}
			threshold := ""
			switch {
			case t.TagThreshold != nil:
				threshold = luaToString(*t.TagThreshold)
			case t.ThresholdVariable != nil:
				threshold = formatModName(*t.ThresholdVariable)
			}
			desc = "If " + formatVarNameOrList(vars) + utilsTernary(t.TagUpper, " <= ", " >= ") + threshold
		case *mod.StatThresholdTag:
			stats := t.StatList
			if t.Stat != "" {
				stats = []string{t.Stat}
			}
			pct := ""
			if t.ThresholdPercent != 0 {
				pct = luaToString(t.ThresholdPercent) + "% "
			}
			threshold := luaToString(t.Threshold)
			if t.TagThresholdStat != "" {
				threshold = formatModName(t.TagThresholdStat)
			}
			desc = "If " + formatVarNameOrList(stats) + utilsTernary(t.TagUpper, " <= ", " >= ") + pct + threshold
		case *mod.SkillNameTag:
			desc = "Skill: " + strings.Join(t.SkillNameList, " / ")
		case *mod.SkillIDTag:
			name := t.IDTag
			if ge := env.loadData().effect(t.IDTag); ge != nil {
				name = ge.Name
			}
			desc = "Skill: " + name
		case *mod.SkillTypeTag:
			names := make([]string, len(t.SkillTypeList))
			for i, st := range t.SkillTypeList {
				names[i] = formatModName(skillTypeNameOf(st))
			}
			desc = "Skill type: " + notText(t.Negative) + strings.Join(names, " / ")
		case *mod.GemTagTag:
			desc = "Gem tag: " + notText(t.Negative) + formatVarNameOrList(t.GemTagList)
		case *mod.BaseFlagTag:
			desc = "Base flag: " + notText(t.Negative) + formatModName(t.BaseFlag)
		case *mod.SlotNumberTag:
			desc = "When in slot #" + strconv.Itoa(t.N)
		case *mod.GlobalEffectTag:
			if len(t.GlobalEffectList) > 0 {
				desc = formatModName(t.GlobalEffectList[0])
			}
		case *mod.LimitTag:
			switch {
			case t.Negative && t.LimitVar != "":
				desc = "Limited to -" + formatModName(t.LimitVar)
			case t.Negative:
				desc = "Limited to " + formatModBase(m, -t.TagLimit)
			case t.LimitVar != "":
				desc = "Limited to " + formatModName(t.LimitVar)
			default:
				desc = "Limited to " + formatModBase(m, t.TagLimit)
			}
		case *mod.MonsterTagTag:
			desc = "Monster Tag: " + strings.Join(t.MonsterTagList, "/")
		default:
			desc = formatModName(string(tag.Type()))
		}
		if desc != "" {
			out = append(out, desc)
		}
	}
	return strings.Join(out, ", ")
}

// skillTypeNameOf is SkillTypeName[skillType] or tostring(skillType) of a skill type a tag holds by name or by id.
func skillTypeNameOf(st string) string {
	if id, err := strconv.Atoi(st); err == nil {
		if name, ok := poe2.SkillTypeName(id); ok {
			return name
		}
	}
	return st
}

func utilsTernary(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

// modSourceName is the row's source type, source name and the item or node it names.
//
//pob2:block:src/Classes/CalcBreakdownControl.lua:420-469
func (env *Environment) modSourceName(source string) (string, string, int, int) {
	sourceType, rest, _ := strings.Cut(source, ":")
	switch sourceType {
	case "Item":
		if m := itemSourceRe.FindStringSubmatch(source); m != nil && env.items != nil {
			id, _ := strconv.Atoi(m[1])
			if item := env.items.Items[id]; item != nil {
				return sourceType, rarityColors[item.Rarity] + item.Name, id, 0
			}
		}
	case "Tree":
		if m := treeSourceRe.FindStringSubmatch(source); m != nil && env.Spec != nil {
			id, _ := strconv.Atoi(m[1])
			if node := env.Spec.Nodes[id]; node != nil {
				return sourceType, node.Dn, 0, id
			}
			if node := env.Spec.SwitchableNodes[id]; node != nil {
				return sourceType, node.Dn, 0, id
			}
			if env.Spec.Tree != nil {
				if node := env.Spec.Tree.Nodes[id]; node != nil {
					return sourceType, node.Dn, 0, id
				}
			}
		}
	case "Skill":
		if ge := env.loadData().effect(rest); ge != nil {
			return sourceType, ge.Name, 0, 0
		}
	case "Many Sources", "Pantheon", "Spectre", "Quest", "Custom", "Rune":
		return sourceType, rest, 0, 0
	}
	return sourceType, "", 0, 0
}

type modSectionStore interface {
	Tabulate(modType mod.Type, cfg *moddb.ListCfg, modNames ...string) []moddb.ModResult
	Combine(modType mod.Type, cfg *moddb.ListCfg, modNames ...string) *mod.ModValueMulti
}

// ModSection is the mod table AddModSection builds for one cell entry over the player; nil where it has no rows.
//
//pob2:src/Classes/CalcBreakdownControl.lua:300-548
//nolint:gocyclo // Mirrors the Lua.
func (env *Environment) ModSection(sd ModSectionData) []ModSectionRow {
	if env == nil || env.Player == nil {
		return nil
	}
	actor := env.Player
	cfg := &moddb.ListCfg{}
	if sd.Cfg != "" && actor.MainSkill != nil {
		if c := actor.MainSkill.mainSkillCfg(sd.Cfg); c != nil {
			copied := *c
			cfg = &copied
		}
	}
	cfg.Source = nil
	if sd.ModSource != "" {
		src := mod.Source(sd.ModSource)
		cfg.Source = &src
	}
	cfg.IgnoreSourceInCheckConditions = true
	cfg.Actor = sd.Actor

	var modStore modSectionStore
	switch {
	case sd.Enemy:
		if actor.Enemy == nil || actor.Enemy.ModDB == nil {
			return nil
		}
		modStore = actor.Enemy.ModDB
	case sd.Cfg != "":
		if actor.MainSkill == nil || actor.MainSkill.SkillModList == nil {
			return nil
		}
		modStore = actor.MainSkill.SkillModList
	default:
		modStore = actor.ModDB
	}
	modType := mod.Type(sd.ModType)
	var rowList []moddb.ModResult
	if sd.Multi {
		for _, name := range sd.ModName {
			rowList = append(rowList, modStore.Tabulate(modType, cfg, name)...)
		}
	} else {
		rowList = modStore.Tabulate(modType, cfg, sd.ModName...)
	}
	if len(rowList) == 0 {
		return nil
	}

	byValue := func(a, b moddb.ModResult) bool {
		if a.Mod.Name() != b.Mod.Name() {
			return a.Mod.Name() > b.Mod.Name()
		}
		return modIsNumeric(a.Mod) && modIsNumeric(b.Mod) && a.Value > b.Value
	}
	if sd.ModType == "" {
		luaSort(rowList, func(a, b moddb.ModResult) bool {
			if a.Mod.Type() == b.Mod.Type() {
				return byValue(a, b)
			}
			return a.Mod.Type() < b.Mod.Type()
		})
	} else {
		luaSort(rowList, byValue)
	}

	var sourceTotals map[string][]string
	if sd.ModSource == "" {
		sourceTotals = map[string][]string{}
		var typeList []mod.Type
		for _, row := range rowList {
			if !slices.Contains(typeList, row.Mod.Type()) {
				typeList = append(typeList, row.Mod.Type())
			}
			sourceType, _, _ := strings.Cut(string(row.Mod.GetSource()), ":")
			sourceTotals[sourceType] = nil
		}
		total := func(modType mod.Type, names ...string) (string, bool) {
			v := modStore.Combine(modType, cfg, names...)
			if v == nil {
				return "", false
			}
			switch v.Type() {
			case mod.ModValueMultiTypeFlag:
				return formatModValue(0, nil, modType), v.Flag()
			case mod.ModValueMultiTypeList:
				return formatModValue(0, v.List(), modType), true
			case mod.ModValueMultiTypeFloat:
			default:
				return "", false
			}
			t := v.Float()
			if modType == mod.TypeMore {
				t = round((t - 1) * 100)
			}
			return formatModValue(t, nil, modType), t != 0
		}
		for sourceType := range sourceTotals {
			src := mod.Source(sourceType)
			cfg.Source = &src
			var lines []string
			for _, modType := range typeList {
				if sd.Multi {
					for _, modName := range sd.ModName {
						if t, ok := total(modType, modName); ok {
							lines = append(lines, t+" "+modNameSpacedRe.ReplaceAllString(modName, "$1 $2"))
						}
					}
				} else if t, ok := total(modType, sd.ModName...); ok {
					lines = append(lines, t)
				}
			}
			sourceTotals[sourceType] = lines
		}
	}

	rows := make([]ModSectionRow, 0, len(rowList))
	for _, r := range rowList {
		var row ModSectionRow
		if sd.ModType == "" {
			var list any
			if v := r.Mod.Value(); v != nil && v.Type() == mod.ModValueMultiTypeList {
				list = v.List()
			}
			row.Value = formatModValue(r.Value, list, r.Mod.Type())
		} else {
			row.Value = luaToString(round(r.Value, 2))
		}
		if sd.Multi {
			row.Name = formatModName(r.Mod.Name())
		}
		row.Source, row.SourceName, row.ItemID, row.NodeID = env.modSourceName(string(r.Mod.GetSource()))
		if sourceTotals != nil {
			row.Totals = append([]string{"Total from " + row.Source + ":"}, sourceTotals[row.Source]...)
		}
		row.Flags = modFlagText(r.Mod)
		row.Tags = env.modTagText(r.Mod)
		rows = append(rows, row)
	}
	return rows
}

// CalcsBreakdownModList is the mod table AddModSection builds from breakdown name's modList over the player, or the
// minion; nil where it has no rows.
//
//pob2:block:src/Classes/CalcBreakdownControl.lua:293-296,src/Classes/CalcBreakdownControl.lua:311-312,src/Classes/CalcBreakdownControl.lua:355-358,src/Classes/CalcBreakdownControl.lua:407-421
func (env *Environment) CalcsBreakdownModList(name string, minion bool) []ModSectionRow {
	if env == nil || env.Player == nil {
		return nil
	}
	actor := env.Player
	if minion && env.Minion != nil {
		actor = &env.Minion.Actor
	}
	if actor.Breakdown == nil || actor.Breakdown.Entries[name] == nil {
		return nil
	}
	rowList := slices.Clone(actor.Breakdown.Entries[name].ModList)
	if len(rowList) == 0 {
		return nil
	}
	luaSort(rowList, func(a, b BreakdownMod) bool {
		if a.Mod.Name() != b.Mod.Name() {
			return a.Mod.Name() > b.Mod.Name()
		}
		return a.List == nil && b.List == nil && a.Mod.Type() != mod.TypeFlag && b.Mod.Type() != mod.TypeFlag && a.Value > b.Value
	})
	rows := make([]ModSectionRow, 0, len(rowList))
	for _, r := range rowList {
		row := ModSectionRow{Value: formatModValue(r.Value, r.List, r.Mod.Type()), Name: formatModName(r.Mod.Name())}
		row.Source, row.SourceName, row.ItemID, row.NodeID = env.modSourceName(string(r.Mod.GetSource()))
		row.Flags = modFlagText(r.Mod)
		row.Tags = env.modTagText(r.Mod)
		rows = append(rows, row)
	}
	return rows
}
