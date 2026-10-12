package calculator

import (
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

type CompareModRow struct {
	Key        string
	Value      float64
	Display    string
	SourceType string
	SourceName string
	ModName    string
}

var compareItemSourceRe = regexp.MustCompile(`^(Item):-?\d+:`)

//pob:block:src/Classes/CompareCalcsHelpers.lua:77-138
func (env *Environment) CompareModRows(sd ModSectionData, minion bool) []CompareModRow {
	if env == nil || env.Player == nil {
		return nil
	}
	actor := env.Player
	if minion && env.Minion != nil && env.Minion.Actor != nil {
		actor = env.Minion.Actor
	}
	cfg := &moddb.ListCfg{}
	if sd.Cfg != "" && actor.MainSkill != nil {
		if c := actor.MainSkill.breakdownCfg(sd.Cfg); c != nil {
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

	type store interface {
		Tabulate(modType mod.Type, cfg *moddb.ListCfg, modNames ...string) []moddb.ModResult
	}
	var modStore store
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
	rowList := modStore.Tabulate(mod.Type(sd.ModType), cfg, sd.ModName...)
	rows := make([]CompareModRow, 0, len(rowList))
	for _, r := range rowList {
		src := string(r.Mod.GetSource())
		row := CompareModRow{
			Key:   compareItemSourceRe.ReplaceAllString(src, "$1:") + "|" + r.Mod.Name() + "|" + string(r.Mod.Type()),
			Value: r.Value,
		}
		if sd.ModType == "" {
			row.Display = breakdownModValue(r.Value, r.Mod.Type())
		} else {
			row.Display = luaNumber(luaRound(r.Value, 2))
		}
		row.SourceType, _, _ = strings.Cut(src, ":")
		if src == "" {
			row.SourceType = "?"
		}
		if row.SourceType != "Custom" {
			_, row.SourceName, _, _ = env.modSourceName(src)
		}
		if sd.Multi {
			row.ModName = "  " + formatModName(r.Mod.Name())
		}
		rows = append(rows, row)
	}
	return rows
}
