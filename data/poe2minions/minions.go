// Package poe2minions is PoB-PoE2's data.minions: go-pob-data's monster rows joined with PoB's minion
// and spectre templates.
package poe2minions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Vilsol/go-pob-data/loader"
	game "github.com/Vilsol/go-pob-data/poe2"

	"github.com/Vilsol/go-pob/data/poe2"
)

// Mod is a minion modList entry as ModCache-style JSON: name, type, value, flags, keywordFlags, tags,
// source.
type Mod = map[string]any

// Minion is one data.minions entry.
type Minion struct {
	game.Minion
	// Key is the entry's key in data.minions.
	Key string
	// Limit is the entry's limit variable; "" is none.
	Limit      string
	ExtraFlags map[string]bool
	// SkillList is the variety's granted effects, then the template's extra skills.
	SkillList []string
	// ModList is the variety's stats SkillStatMap maps, then the template's #mod lines.
	ModList []Mod
	// Accuracy is the exporter's constant accuracy.
	Accuracy float64
	// Spectre marks an entry of Spectres.lua.
	Spectre bool
}

// ErrNoVariety is a template's variety the game does not have.
var ErrNoVariety = errors.New("no such monster variety")

// Minions is data.minions by key.
type Minions map[string]*Minion

// Load builds data.minions from version's monster tables and .ot files.
func Load(ctx context.Context, version string, cache loader.AssetCache) (Minions, error) {
	tables, err := game.LoadMinionTables(ctx, version, cache)
	if err != nil {
		return nil, fmt.Errorf("minion tables: %w", err)
	}
	manifest, err := loader.LoadManifest(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	var readErr error
	readOT := func(path string) (string, bool) {
		if !manifest.HasFile(path) {
			return "", false
		}
		b, err := loader.LoadFile(ctx, version, path, cache)
		if err != nil {
			readErr = err
			return "", false
		}
		return string(b), true
	}
	out, err := Build(func(variety string) (*game.Minion, error) {
		if tables.VarietyByID(variety) == nil {
			return nil, fmt.Errorf("%s: %w", variety, ErrNoVariety)
		}
		m, err := tables.Minion(variety, readOT)
		if err == nil && readErr != nil {
			err = readErr
		}
		return m, err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Build is Data.lua's load of Minions.lua and Spectres.lua over the templates, with variety giving an
// entry's game half, or ErrNoVariety for one the game does not have. The exporter skips such an entry
// before it clears #flags, so they carry to the file's next entry.
//
//pob2:block:src/Modules/Data.lua:1083-1094
func Build(variety func(id string) (*game.Minion, error)) (Minions, error) {
	minions := Minions{}
	spectres := Minions{}
	var carried []string
	file := ""
	for i := range poe2.MinionTemplates {
		t := &poe2.MinionTemplates[i]
		if t.File != file {
			file, carried = t.File, nil
		}
		row, err := variety(t.Variety)
		flags := append(carried, t.Flags...)
		if errors.Is(err, ErrNoVariety) {
			carried = flags
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.Key, err)
		}
		carried = nil
		m, err := emit(t, row, flags)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.Key, err)
		}
		if t.File == "Spectres" {
			spectres[t.Key] = m
		} else {
			minions[t.Key] = m
		}
	}
	for _, name := range sortedKeys(spectres) {
		spectre := spectres[name]
		spectre.Limit = "ActiveSpectreLimit"
		spectre.Spectre = true
		minions[name] = spectre
	}
	for _, minion := range minions {
		for _, m := range minion.ModList {
			m["source"] = "Minion:" + minion.Name
		}
	}
	return minions, nil
}

func sortedKeys(m Minions) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// emit is the template half of #emit: the limit, flags, extra skills and mods, and the stats the
// exporter maps through SkillStatMap's first mod.
//
//pob2:block:src/Export/Scripts/minions.lua:257-376
func emit(t *poe2.MinionTemplate, row *game.Minion, flags []string) (*Minion, error) {
	m := &Minion{Minion: *row, Key: t.Key, Limit: t.Limit, Accuracy: 1}
	if len(flags) > 0 {
		m.ExtraFlags = make(map[string]bool, len(flags))
		for _, f := range flags {
			m.ExtraFlags[f] = true
		}
	}
	m.SkillList = append(append([]string{}, row.SkillList...), t.Skills...)
	for _, stat := range row.ModList {
		entry, ok := poe2.SkillStatMapEntry(stat.StatID)
		if !ok || len(entry.Items) == 0 {
			continue
		}
		item := entry.Items[0]
		if item.Group != nil {
			return nil, fmt.Errorf("stat %s maps to a group, which the exporter cannot write", stat.StatID)
		}
		var newMod Mod
		if err := json.Unmarshal([]byte(item.Mod), &newMod); err != nil {
			return nil, fmt.Errorf("stat %s: %w", stat.StatID, err)
		}
		switch v := newMod["value"].(type) {
		case nil, bool:
			value := stat.Value
			if entry.HasValue {
				value = entry.Value
			} else {
				mult, div := 1.0, 1.0
				if entry.HasMult {
					mult = entry.Mult
				}
				if entry.HasDiv {
					div = entry.Div
				}
				value = value * mult / div
			}
			newMod["value"] = luaNumber(value)
		case map[string]any:
		default:
			return nil, fmt.Errorf("stat %s maps to a mod with a %T value, which the exporter cannot write", stat.StatID, v)
		}
		m.ModList = append(m.ModList, newMod)
	}
	for _, s := range t.Mods {
		var extra Mod
		if err := json.Unmarshal([]byte(s), &extra); err != nil {
			return nil, fmt.Errorf("a #mod: %w", err)
		}
		m.ModList = append(m.ModList, extra)
	}
	return m, nil
}

// luaNumber is a number as the exporter's file reads back: written with Lua's %.14g.
func luaNumber(v float64) float64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 14, 64), 64)
	return f
}
