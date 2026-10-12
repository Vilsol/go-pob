package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data/raw"
)

//pob:src/Export/Scripts/statdesc.lua:118-141
var statFileList = []string{
	"active_skill_gem_stat_descriptions",
	"aura_skill_stat_descriptions",
	"banner_aura_skill_stat_descriptions",
	"beam_skill_stat_descriptions",
	"brand_skill_stat_descriptions",
	"curse_skill_stat_descriptions",
	"debuff_skill_stat_descriptions",
	"secondary_debuff_skill_stat_descriptions",
	"gem_stat_descriptions",
	"minion_attack_skill_stat_descriptions",
	"minion_skill_stat_descriptions",
	"minion_spell_skill_stat_descriptions",
	"minion_spell_damage_skill_stat_descriptions",
	"single_minion_spell_skill_stat_descriptions",
	"monster_stat_descriptions",
	"offering_skill_stat_descriptions",
	"skill_stat_descriptions",
	"stat_descriptions",
	"variable_duration_skill_stat_descriptions",
	"buff_skill_stat_descriptions",
	"tincture_stat_descriptions",
	"graft_stat_descriptions",
}

var (
	gggMarkupRe = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggLinkRe   = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggAliasRe  = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
	includeRe   = regexp.MustCompile(`^Metadata/StatDescriptions/(.+)\.txt$`)
)

//pob:src/Modules/Common.lua:1048-1054
func escapeGGGString(text string) string {
	text = gggMarkupRe.ReplaceAllString(text, "$1")
	text = gggLinkRe.ReplaceAllString(text, "$1")
	return gggAliasRe.ReplaceAllString(text, "$1")
}

type sdSpec struct {
	K string `json:"k"`
	V any    `json:"v"`
}

type sdVariant struct {
	Limit      [][2]any `json:"limit"`
	Text       string   `json:"text"`
	Specs      []sdSpec `json:"specs"`
	GemQuality bool     `json:"gemQuality,omitempty"`
}

type sdDescriptor struct {
	Stats    []string    `json:"stats"`
	Variants []sdVariant `json:"variants,omitempty"`
}

type sdScope struct {
	Parent      string                  `json:"parent,omitempty"`
	Stats       map[string]int          `json:"stats"`
	Descriptors map[string]sdDescriptor `json:"descriptors"`
}

//pob:block:src/Export/Scripts/statdesc.lua:48-69
func sdLimit(c rawdata.Condition) [2]any {
	end := func(p *int) any {
		if p == nil {
			return "#"
		}
		return *p
	}
	if c.Negated && c.Min != nil && c.Max != nil && *c.Min == *c.Max {
		return [2]any{"!", *c.Min}
	}
	return [2]any{end(c.Min), end(c.Max)}
}

//pob:block:src/Export/Scripts/statdesc.lua:70-97
func sdSpecs(handlers []rawdata.IndexHandler) []sdSpec {
	var tokens []string
	for _, h := range handlers {
		tokens = append(tokens, h.Name)
		if h.Arg != "" {
			tokens = append(tokens, h.Arg)
		}
	}
	specs := []sdSpec{}
	for i := 0; i < len(tokens); {
		if tokens[i] == "canonical_line" {
			specs = append(specs, sdSpec{K: "canonical_line", V: true})
			i++
			continue
		}
		if i+1 < len(tokens) {
			var v any = tokens[i+1]
			if n, err := strconv.ParseFloat(tokens[i+1], 64); err == nil {
				v = n
			}
			specs = append(specs, sdSpec{K: tokens[i], V: v})
			i += 2
		} else {
			i++
		}
	}
	return specs
}

func sdUniverse() map[string]bool {
	universe := map[string]bool{}
	add := func(stats []*poe.Stat) {
		for _, s := range stats {
			if s != nil {
				universe[s.ID] = true
			}
		}
	}
	for _, set := range poe.GrantedEffectStatSets {
		add(set.GetImplicitStats())
		add(set.GetConstantStats())
	}
	for _, level := range poe.GrantedEffectStatSetsPerLevels {
		add(level.GetAdditionalStats())
		add(level.GetFloatStats())
		add(level.GetAdditionalBooleanStats())
	}
	for _, q := range poe.GrantedEffectQualityStats {
		add(q.GetStats())
	}
	return universe
}

//pob:src/Export/Scripts/statdesc.lua:3-116
func sdScopeOf(file *rawdata.TranslationFile, universe map[string]bool) (sdScope, error) {
	scope := sdScope{Stats: map[string]int{}, Descriptors: map[string]sdDescriptor{}}
	for _, inc := range file.Includes {
		m := includeRe.FindStringSubmatch(inc)
		if m == nil {
			return scope, fmt.Errorf("include %q is not a stat description file", inc)
		}
		scope.Parent = m[1]
	}
	index := 0
	for _, stat := range file.NoDescription {
		index++
		if universe[stat] {
			scope.Stats[stat] = index
		}
	}
	first := index
	for i, d := range file.Descriptors {
		for _, stat := range d.IDs {
			if universe[stat] {
				scope.Stats[stat] = first + i + 1
			}
		}
	}
	for _, at := range scope.Stats {
		key := strconv.Itoa(at)
		if _, ok := scope.Descriptors[key]; ok {
			continue
		}
		if at <= first {
			scope.Descriptors[key] = sdDescriptor{Stats: []string{file.NoDescription[at-1]}}
			continue
		}
		d := file.Descriptors[at-first-1]
		desc := sdDescriptor{Stats: d.IDs, Variants: []sdVariant{}}
		for _, line := range d.List {
			if line.Qualifier == "table_only" {
				continue
			}
			v := sdVariant{Text: strings.ReplaceAll(escapeGGGString(line.String), `\n`, "\n"), Specs: sdSpecs(line.Handlers), GemQuality: strings.Contains(line.Qualifier, "gem_quality")}
			for _, c := range line.Conditions {
				v.Limit = append(v.Limit, sdLimit(c))
			}
			desc.Variants = append(desc.Variants, v)
		}
		scope.Descriptors[key] = desc
	}
	for _, stat := range file.NoDescription {
		for _, d := range file.Descriptors {
			for _, s := range d.IDs {
				if s == stat && universe[s] {
					return scope, fmt.Errorf("%s is both described and no_description; the export's index order is unknown", stat)
				}
			}
		}
	}
	return scope, nil
}

func dumpStatDescriptions() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	for _, init := range []func() error{
		func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeGrantedEffectStatSets(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeGrantedEffectStatSetsPerLevels(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeGrantedEffectQualityStats(ctx, raw.LatestVersion, cache) },
	} {
		if err := init(); err != nil {
			return nil, err
		}
	}
	universe := sdUniverse()
	if len(universe) < 1000 {
		return nil, fmt.Errorf("only %d stats reach a granted effect", len(universe))
	}

	scopes := map[string]sdScope{}
	var load func(name string) error
	load = func(name string) error {
		if _, ok := scopes[name]; ok {
			return nil
		}
		file, err := loader.LoadTranslation(ctx, raw.LatestVersion, "en", name, cache)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		scope, err := sdScopeOf(file, universe)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		scopes[name] = scope
		if scope.Parent != "" {
			return load(scope.Parent)
		}
		return nil
	}
	for _, name := range statFileList {
		if err := load(name); err != nil {
			return nil, err
		}
	}
	return map[string]any{"version": raw.LatestVersion, "scopes": scopes}, nil
}
