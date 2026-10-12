package calculator

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/moddb"
)

type CompareSkillBreakdownRow struct {
	Value      string
	Name       string
	Source     string
	SourceName string
}

//pob:src/Classes/CompareCalcsHelpers.lua:355-394
func (env *Environment) CompareSkillBreakdown(name string) []CompareSkillBreakdownRow {
	if env == nil || env.Player == nil {
		return nil
	}
	rowList := slices.Clone(env.Player.breakdownModLists[name])
	slices.SortStableFunc(rowList, func(a, b moddb.ModResult) int {
		if c := cmp.Compare(b.Mod.Name(), a.Mod.Name()); c != 0 {
			return c
		}
		return cmp.Compare(b.Value, a.Value)
	})
	rows := make([]CompareSkillBreakdownRow, 0, len(rowList))
	for _, entry := range rowList {
		src := string(entry.Mod.GetSource())
		row := CompareSkillBreakdownRow{Value: breakdownModValue(entry.Value, entry.Mod.Type()), Name: formatModName(entry.Mod.Name())}
		row.Source, _, _ = strings.Cut(src, ":")
		if src == "" {
			row.Source = "?"
		}
		_, row.SourceName, _, _ = env.modSourceName(src)
		rows = append(rows, row)
	}
	return rows
}
