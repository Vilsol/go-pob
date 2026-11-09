package calculator2

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

var modRowItemSourceRe = regexp.MustCompile(`^(Item):\d+:`)

// tabulateMods is TabulateMods over the actor; the names are tabulated eight at a time, as unpack takes them.
//
//pob2:src/Classes/CompareCalcsHelpers.lua:79-116
func tabulateMods(sd ModSectionData, actor *Actor) []moddb.ModResult {
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
	cfg.Actor = sd.Actor
	var modStore modSectionStore
	switch {
	case sd.Enemy && actor.Enemy != nil:
		modStore = actor.Enemy.ModDB
	case sd.Cfg != "" && actor.MainSkill != nil:
		modStore = actor.MainSkill.SkillModList
	default:
		modStore = actor.ModDB
	}
	if modStore == nil {
		return nil
	}
	modType := mod.Type(sd.ModType)
	if !sd.Multi {
		return modStore.Tabulate(modType, cfg, sd.ModName...)
	}
	var rowList []moddb.ModResult
	for index := 0; index < len(sd.ModName); index += 8 {
		rowList = append(rowList, modStore.Tabulate(modType, cfg, sd.ModName[index:min(index+8, len(sd.ModName))]...)...)
	}
	return rowList
}

//pob2:src/Classes/CompareCalcsHelpers.lua:119-127
func modRowKey(row moddb.ModResult) string {
	normalizedSrc := modRowItemSourceRe.ReplaceAllString(string(row.Mod.GetSource()), "$1:")
	return normalizedSrc + "|" + row.Mod.Name() + "|" + string(row.Mod.Type())
}

// modRowValue is tostring(row.value): a list's table prints its address, which no other row shares.
func modRowValue(row moddb.ModResult, n int) string {
	switch row.Mod.Type() {
	case mod.TypeFlag:
		return "true"
	case mod.TypeList:
		return "table:" + strconv.Itoa(n)
	}
	return strconv.FormatFloat(row.Value, 'g', 14, 64)
}

// CompareModRowsMatch is whether a sectionData's mod rows match between the two calculations' players: the same rows
// by source (an item's id left out), name, type and value, in any order.
//
//pob2:block:src/Classes/CompareTab.lua:4546-4561
func CompareModRowsMatch(primary, compare *Environment, sd ModSectionData) bool {
	primaryRows := tabulateMods(sd, primary.Player)
	compareRows := tabulateMods(sd, compare.Player)
	if len(primaryRows) != len(compareRows) {
		return false
	}
	counts := map[string]int{}
	for i, row := range primaryRows {
		counts[modRowKey(row)+"|"+modRowValue(row, i)]++
	}
	for i, row := range compareRows {
		key := modRowKey(row) + "|" + modRowValue(row, len(primaryRows)+i)
		count, ok := counts[key]
		if !ok {
			return false
		}
		if count > 1 {
			counts[key] = count - 1
		} else {
			delete(counts, key)
		}
	}
	return true
}

// CompareModRow is one TabulateMods row as the Calcs view's tooltip lists it: its ModRowKey, value, FormatModRow's
// display value, source type, source name and (for a list of names) spaced mod name.
type CompareModRow struct {
	Key        string
	Value      float64
	Display    string
	SourceType string
	SourceName string
	ModName    string
}

// cFormatG is string.format("%+g", v).
func cFormatG(v float64) string {
	s := strconv.FormatFloat(v, 'g', 6, 64)
	if v >= 0 && !math.Signbit(v) {
		s = "+" + s
	}
	return s
}

//pob2:src/Classes/CompareCalcsHelpers.lua:13-35
func formatCalcModValue(row moddb.ModResult) string {
	value := row.Value
	switch row.Mod.Type() {
	case mod.TypeBase:
		return cFormatG(value) + " base"
	case mod.TypeIncrease:
		if value >= 0 {
			return luaNum(value) + "% increased"
		}
		return luaNum(-value) + "% reduced"
	case mod.TypeMore:
		if value >= 0 {
			return luaNum(value) + "% more"
		}
		return luaNum(-value) + "% less"
	case mod.TypeOverride:
		return "Override: " + luaNum(value)
	case mod.TypeFlag:
		return "True"
	}
	return luaNum(value)
}

// resolveSourceName is ResolveSourceName over the calculation's items, spec and skills.
//
//pob2:src/Classes/CompareCalcsHelpers.lua:45-76
func (env *Environment) resolveSourceName(source string) string {
	sourceType, rest, _ := strings.Cut(source, ":")
	switch sourceType {
	case "Item", "Tree", "Skill":
		_, name, _, _ := env.modSourceName(source)
		return name
	case "Pantheon", "Spectre":
		return rest
	}
	return ""
}

// CompareModRows is the player's TabulateMods rows for sd, each formatted as FormatModRow formats it.
//
//pob2:src/Classes/CompareCalcsHelpers.lua:130-146
func (env *Environment) CompareModRows(sd ModSectionData) []CompareModRow {
	if env == nil || env.Player == nil {
		return nil
	}
	rowList := tabulateMods(sd, env.Player)
	rows := make([]CompareModRow, 0, len(rowList))
	for _, row := range rowList {
		src := string(row.Mod.GetSource())
		out := CompareModRow{Key: modRowKey(row), Value: row.Value}
		if sd.ModType == "" {
			out.Display = formatCalcModValue(row)
		} else {
			out.Display = luaNum(math.Floor(float64(row.Value*100)+0.5) / 100)
		}
		out.SourceType = "?"
		if src != "" {
			out.SourceType, _, _ = strings.Cut(src, ":")
		}
		out.SourceName = env.resolveSourceName(src)
		if sd.Multi {
			out.ModName = "  " + formatModName(row.Mod.Name())
		}
		rows = append(rows, out)
	}
	return rows
}

// CompareSkillBreakdownRow is one row of the Calcs view header's SkillBuffs or SkillDebuffs panel.
type CompareSkillBreakdownRow struct {
	Value      string
	Name       string
	Source     string
	SourceName string
}

//pob2:src/Classes/CompareCalcsHelpers.lua:332-366
func (env *Environment) resolveModSource(source string) (string, string) {
	if source == "" {
		return "?", ""
	}
	sourceType, rest, _ := strings.Cut(source, ":")
	switch sourceType {
	case "Item", "Skill":
		_, name, _, _ := env.modSourceName(source)
		return sourceType, name
	case "Tree":
		if m := treeSourceRe.FindStringSubmatch(source); m != nil && env.Spec != nil {
			id, _ := strconv.Atoi(m[1])
			if node := env.Spec.Nodes[id]; node != nil {
				return sourceType, node.Dn
			}
			if env.Spec.Tree != nil {
				if node := env.Spec.Tree.Nodes[id]; node != nil {
					return sourceType, node.Dn
				}
			}
		}
	case "Pantheon", "Spectre":
		return sourceType, rest
	}
	return sourceType, ""
}

// CompareSkillBreakdown is the panel DrawSkillBreakdownPanel draws for the player's SkillBuffs or SkillDebuffs
// breakdown: its mods by name, then value, descending.
//
//pob2:block:src/Classes/CompareCalcsHelpers.lua:370-409
func (env *Environment) CompareSkillBreakdown(breakdownKey string) []CompareSkillBreakdownRow {
	if env == nil || env.Player == nil || env.Player.Breakdown == nil {
		return nil
	}
	breakdown := env.Player.Breakdown.Entries[breakdownKey]
	if breakdown == nil || len(breakdown.ModList) == 0 {
		return nil
	}
	rowList := slices.Clone(breakdown.ModList)
	isNumber := func(e BreakdownMod) bool { return e.List == nil && e.Mod.Type() != mod.TypeFlag }
	luaSort(rowList, func(a, b BreakdownMod) bool {
		return a.Mod.Name() > b.Mod.Name() || (a.Mod.Name() == b.Mod.Name() && isNumber(a) && isNumber(b) && a.Value > b.Value)
	})
	rows := make([]CompareSkillBreakdownRow, 0, len(rowList))
	for _, entry := range rowList {
		row := CompareSkillBreakdownRow{
			Value: formatCalcModValue(moddb.ModResult{Value: entry.Value, Mod: entry.Mod}),
			Name:  formatModName(entry.Mod.Name()),
		}
		row.Source, row.SourceName = env.resolveModSource(string(entry.Mod.GetSource()))
		rows = append(rows, row)
	}
	return rows
}
