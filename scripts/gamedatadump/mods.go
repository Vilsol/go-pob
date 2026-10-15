package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

// https://www.poewiki.net/wiki/Modifier#Domain
const (
	domainItem         = 1
	domainFlask        = 2
	domainJewel        = 10
	domainAbyssJewel   = 13
	domainDelveFossil  = 16
	domainClusterJewel = 21
	domainUnveiled     = 28
	domainTincture     = 34
	domainCharm        = 35
	domainGraft        = 38
	domainMercenary    = 41
)

// https://www.poewiki.net/wiki/Modifier#Generation_type
const (
	genPrefix          = 1
	genSuffix          = 2
	genIntrinsic       = 3
	genCorrupted       = 5
	genScourgeBenefit  = 24
	genScourgeDownside = 25
	genSearingExarch   = 28
	genEaterOfWorlds   = 29
)

type exportMod struct {
	ID                  string    `json:"id"`
	Type                string    `json:"type"`
	Affix               string    `json:"affix"`
	Lines               []string  `json:"lines"`
	Level               int       `json:"level"`
	Group               string    `json:"group"`
	StatOrder           []float64 `json:"statOrder"`
	WeightKey           []string  `json:"weightKey"`
	WeightVal           []int     `json:"weightVal"`
	WeightMultiplierKey []string  `json:"weightMultiplierKey"`
	WeightMultiplierVal []int     `json:"weightMultiplierVal"`
	ModTags             []string  `json:"modTags"`
	Tags                []string  `json:"-"`
}

var modNumberRe = regexp.MustCompile(`\d+`)

type modsExport struct {
	describer *exportDescriber
}

//pob:block:src/Export/Scripts/mods.lua:1-4
func newModsExport() (*modsExport, error) {
	ctx, cache := gameDataContext(), assetCache()
	for name, init := range map[string]func() error{
		"Stats":         func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		"Tags":          func() error { return poe.InitializeTags(ctx, raw.LatestVersion, cache) },
		"ModType":       func() error { return poe.InitializeModTypes(ctx, raw.LatestVersion, cache) },
		"ModFamily":     func() error { return poe.InitializeModFamilies(ctx, raw.LatestVersion, cache) },
		"BuffTemplates": func() error { return poe.InitializeBuffTemplates(ctx, raw.LatestVersion, cache) },
		"Mods":          func() error { return poe.InitializeMods(ctx, raw.LatestVersion, cache) },
	} {
		if err := init(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	if len(poe.ModFamilies) == 0 {
		return nil, fmt.Errorf("ModFamily holds no rows")
	}
	describer, err := loadStatFiles("tincture_stat_descriptions", "graft_stat_descriptions")
	if err != nil {
		return nil, err
	}
	return &modsExport{describer: describer}, nil
}

func modFamily(m *poe.Mod, i int) string {
	if i >= len(m.Families) {
		return ""
	}
	return poe.ModFamilies[m.Families[i]].ID
}

func modTags(keys []int) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, poe.Tags[k].ID)
	}
	return out
}

//pob:src/Export/statdesc.lua:431-455
func (x *modsExport) describeMod(m *poe.Mod, stat2 *[2]int) ([]string, []float64, error) {
	buffTemplateStats := map[string]bool{}
	var templates []int
	if m.BuffTemplate != nil {
		templates = append(templates, *m.BuffTemplate)
	}
	templates = append(templates, m.BuffTemplate2...)
	for _, key := range templates {
		for _, s := range poe.BuffTemplates[key].StatsKey {
			buffTemplateStats[poe.Stats[s].ID] = true
		}
	}
	stats := map[string][2]float64{}
	for i, s := range m.Stats() {
		if buffTemplateStats[s.Stat.ID] {
			continue
		}
		v := [2]float64{float64(s.Min), float64(s.Max)}
		if i == 1 && stat2 != nil {
			v = [2]float64{float64(stat2[0]), float64(stat2[1])}
		}
		stats[s.Stat.ID] = v
	}
	return x.describer.describeStats(stats)
}

//pob:src/Export/Scripts/mods.lua:46-204
func (x *modsExport) writeMods(outName string, condFunc func(*poe.Mod) bool) ([]exportMod, error) {
	hasTag := func(m *poe.Mod, id string) bool {
		for _, k := range m.SpawnWeightTagsKeys {
			if poe.Tags[k].ID == id {
				return true
			}
		}
		return false
	}
	var out []exportMod
	for _, mod := range poe.Mods {
		if !condFunc(mod) {
			continue
		}
		if mod.Domain == domainDelveFossil && strings.Contains(outName, "Item") {
			tags := mod.SpawnWeightTagsKeys
			if len(tags) == 3 && poe.Tags[tags[0]].ID == "abyss_jewel" && poe.Tags[tags[1]].ID == "jewel" {
				continue
			}
		} else if mod.Domain == domainDelveFossil && strings.Contains(outName, "JewelAbyss") {
			if !hasTag(mod, "abyss_jewel") {
				continue
			}
		} else if mod.Domain == domainDelveFossil && strings.Contains(outName, "Jewel") {
			if !hasTag(mod, "jewel") {
				continue
			}
		}
		// game data has 0 and 0, which means no description is generated
		var stat2 *[2]int
		if mod.ID == "JewelExpansionPassiveNodes" {
			stat2 = &[2]int{2, 12}
		}
		stats, orders, err := x.describeMod(mod, stat2)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		if len(orders) == 0 || strings.HasPrefix(stats[0], "DNT") {
			continue
		}
		row := exportMod{ID: mod.ID, Affix: mod.Name, Level: mod.Level, Group: poe.ModTypes[mod.ModTypeKey].Name, WeightVal: mod.SpawnWeightValues, ModTags: modTags(mod.ImplicitTagsKeys)}
		switch mod.GenerationType {
		case genPrefix:
			row.Type = "Prefix"
		case genSuffix:
			row.Type = "Suffix"
		case genIntrinsic:
			if mod.Domain == domainItem {
				if strings.HasPrefix(mod.ID, "Synthesis") {
					row.Type = "Synthesis"
				} else if f2 := modFamily(mod, 1); strings.Contains(f2, "MatchedInfluencesTier") {
					row.Type = modNumberRe.FindString(f2) + strings.SplitN(modFamily(mod, 0), "Influence", 2)[0]
				}
			} else if mod.Domain == domainDelveFossil {
				row.Type = "DelveImplicit"
			}
		case genCorrupted:
			row.Type = "Corrupted"
		case genScourgeBenefit:
			row.Type = "ScourgeUpside"
		case genScourgeDownside:
			row.Type = "ScourgeDownside"
		case genSearingExarch:
			row.Type = "Exarch"
		case genEaterOfWorlds:
			row.Type = "Eater"
		}
		if strings.Contains(mod.ID, "EldritchImplicitUniquePresence") {
			for i, stat := range stats {
				stats[i] = "While a Unique Enemy is in your Presence, " + stat
			}
		}
		if strings.Contains(mod.ID, "EldritchImplicitPinnaclePresence") {
			for i, stat := range stats {
				stats[i] = "While a Pinnacle Atlas Boss is in your Presence, " + stat
			}
		}
		row.Lines, row.StatOrder = stats, orders
		row.WeightKey = modTags(mod.SpawnWeightTagsKeys)
		if len(mod.GenerationWeightTagsKeys) > 0 {
			// make large clusters only have 1 notable suffix
			if mod.GenerationType == genSuffix && len(mod.TagsKeys) > 0 && outName == "../Data/ModJewelCluster.lua" && poe.Tags[mod.TagsKeys[0]].ID == "has_affliction_notable" {
				row.WeightMultiplierKey = append([]string{"has_affliction_notable2"}, modTags(mod.GenerationWeightTagsKeys)...)
				row.WeightMultiplierVal = append([]int{0}, mod.GenerationWeightValues...)
				row.Tags = append([]string{"has_affliction_notable2"}, modTags(mod.TagsKeys)...)
			} else {
				row.WeightMultiplierKey = modTags(mod.GenerationWeightTagsKeys)
				row.WeightMultiplierVal = mod.GenerationWeightValues
				if len(mod.TagsKeys) > 0 {
					row.Tags = modTags(mod.TagsKeys)
				}
			}
		}
		out = append(out, row)
	}
	return out, nil
}

var itemExclusiveExcludeRe = regexp.MustCompile(`Royale|Cowards|Map|Ultimatum|UNUSED`)

//pob:block:src/Export/Scripts/mods.lua:282-292
func itemExclusive(mod *poe.Mod) bool {
	return (mod.Domain == domainItem || mod.Domain == domainFlask || mod.Domain == domainJewel || mod.Domain == domainClusterJewel || mod.Domain == domainTincture) &&
		mod.GenerationType == genIntrinsic &&
		len(mod.Families) > 0 && modFamily(mod, 0) != "AuraBonus" &&
		!strings.HasPrefix(mod.ID, "Synthesis") && !itemExclusiveExcludeRe.MatchString(mod.ID) &&
		!strings.HasPrefix(mod.ID, "MutatedUnique")
}

//pob:src/Classes/Item.lua:64-68
func normaliseModLine(line string) string {
	line = modLineNumberRe.ReplaceAllString(line, "#")
	line = modLineRangeRe.ReplaceAllString(line, "#")
	return strings.ReplaceAll(strings.ToLower(line), "\n", " ")
}

var (
	modLineNumberRe = regexp.MustCompile(`\d+\.?\d*`)
	modLineRangeRe  = regexp.MustCompile(`\(-?#-#\)`)
)

//pob:block:src/Classes/Item.lua:1361-1371
func dumpItemModStatOrder() (any, error) {
	x, err := newModsExport()
	if err != nil {
		return nil, err
	}
	mods, err := x.writeMods("../Data/ModItemExclusive.lua", itemExclusive)
	if err != nil {
		return nil, err
	}
	exact, normalised := map[string]float64{}, map[string]float64{}
	for _, mod := range mods {
		for index, line := range mod.Lines {
			order := mod.StatOrder[index]
			if order <= 0 {
				return nil, fmt.Errorf("%s line %d has statOrder %v <= 0", mod.ID, index+1, order)
			}
			exactLine := strings.ReplaceAll(strings.ToLower(line), "\n", " ")
			statLine := normaliseModLine(line)
			exact[exactLine] = math.Min(orDefault(exact, exactLine), order)
			normalised[statLine] = math.Min(orDefault(normalised, statLine), order)
		}
	}
	if len(exact) == 0 {
		return nil, fmt.Errorf("ModItemExclusive described no lines")
	}
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "exact": exact, "normalised": normalised}, nil
}

func orDefault(m map[string]float64, k string) float64 {
	if v, ok := m[k]; ok {
		return v
	}
	return math.Inf(1)
}
