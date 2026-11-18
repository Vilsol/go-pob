package main

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
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

//pob:src/Export/statdesc.lua:571-595
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
	return x.describer.describeModStats(m, stats)
}

var grantedEffectNames map[string]string

// getGrantedEffectNames maps granted effect ids to gem names.
// Secondary effects such as Snipe Support have their own name instead of the gem's
//
//pob:src/Export/statdesc.lua:471-491
func getGrantedEffectNames() (map[string]string, error) {
	if grantedEffectNames != nil {
		return grantedEffectNames, nil
	}
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"ClientStrings", func() error { return poe.InitializeClientStrings(ctx, raw.LatestVersion, cache) }},
		{"BaseItemTypes", func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) }},
		{"ActiveSkills", func() error { return poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffects", func() error { return poe.InitializeGrantedEffects(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffectsPerLevel", func() error { return poe.InitializeGrantedEffectsPerLevels(ctx, raw.LatestVersion, cache) }},
		{"GemEffects", func() error { return poe.InitializeGemEffects(ctx, raw.LatestVersion, cache) }},
		{"SkillGems", func() error { return poe.InitializeSkillGems(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	names := map[string]string{}
	for _, gem := range poe.SkillGems {
		for _, gemEffect := range gem.GetGemVariants() {
			if id := poe.GrantedEffects[gemEffect.GrantedEffect].ID; names[id] == "" {
				names[id] = poe.BaseItemTypes[gem.BaseItemType].Name
			}
			if gemEffect.GrantedEffect2 != nil {
				name := gemEffect.SupportName
				if name == "" {
					name = poe.BaseItemTypes[gem.BaseItemType].Name
				}
				if id := poe.GrantedEffects[*gemEffect.GrantedEffect2].ID; names[id] == "" {
					names[id] = name
				}
			}
		}
	}
	grantedEffectNames = names
	return names, nil
}

// describeGrantedEffects: 3.30 moved the gem and level of most "Supported by" and "Grants" mods from their display
// stats, which are now 0, into a GrantedEffect column on the mod row. The display stat may still hold a level when
// describing legacy values. These mods might also have no stats at all, in which case the line is only shown if the
// mod is flagged to show it. Trigger mods are not
//
//pob:src/Export/statdesc.lua:498-552
func (e *exportDescriber) describeGrantedEffects(m *poe.Mod, stats map[string]*edRange) ([]string, []float64, error) {
	var hasDisplayStat bool
	var level *edRange
	// Only handle display stats that GGG zeroed. If a legacy value is given for one that still has a description, describeStats handles it
	for _, s := range m.Stats() {
		if strings.HasPrefix(s.Stat.ID, "local_display_") && s.Min == 0 && s.Max == 0 {
			if value := stats[s.Stat.ID]; value != nil && (value.min != 0 || value.max != 0) {
				if _, ok := e.descriptors[s.Stat.ID]; ok {
					return nil, nil, nil
				}
				level = value
			}
			hasDisplayStat = true
		}
	}
	if !hasDisplayStat && (m.StatsKey1 != nil || !m.ShowGrantedEffects) {
		return nil, nil, nil
	}
	names, err := getGrantedEffectNames()
	if err != nil {
		return nil, nil, err
	}
	var out []string
	var orders []float64
	seen := map[string]bool{}
	for _, key := range m.GrantedEffectsPerLevelKeys {
		levelRow := poe.GrantedEffectsPerLevels[key]
		grantedEffect := poe.GrantedEffects[levelRow.GrantedEffect]
		// Supports are named after their gem and actives after their skill. The stat order of a remaining descriptor of the same kind is borrowed
		var name, template, orderStat string
		if grantedEffect.IsSupport {
			name, template, orderStat = names[grantedEffect.ID], "ModGrantsSupportEffect", "local_random_support_gem_level"
		} else {
			if grantedEffect.ActiveSkill != nil {
				name = poe.ActiveSkills[*grantedEffect.ActiveSkill].DisplayedName
			}
			template, orderStat = "ModGrantsEffect", "local_display_grants_skill_frostbite_level"
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var levelText string
		hasLevel := false
		if level != nil {
			levelText, hasLevel = luaNumber(level.min), true
			if level.min != level.max {
				levelText = "(" + luaNumber(level.min) + "-" + luaNumber(level.max) + ")"
			}
			// Skills that cannot be supported, such as Harbinger skills, are shown without a level
		} else if grantedEffect.IsSupport || !grantedEffect.CannotBeSupported {
			levelText, hasLevel = strconv.Itoa(levelRow.Level), true
		}
		if hasLevel {
			template += "WithLevel"
		}
		// The same templates the client uses, e.g. "Socketed Gems are Supported by Level {1} {0}"
		row, ok := poe.ClientStringsByID[template]
		if !ok {
			return nil, nil, fmt.Errorf("ClientStrings has no %s", template)
		}
		order, ok := e.descriptors[orderStat]
		if !ok {
			return nil, nil, fmt.Errorf("no stat descriptor for %s", orderStat)
		}
		out = append(out, strings.ReplaceAll(strings.ReplaceAll(row.Text, "{0}", name), "{1}", levelText))
		orders = append(orders, float64(order.order))
	}
	return out, orders, nil
}

// describeModStats is describeStats with the granted effect lines of the mod inserted by stat order
//
//pob:src/Export/statdesc.lua:554-570
func (e *exportDescriber) describeModStats(m *poe.Mod, stats map[string][2]float64) ([]string, []float64, error) {
	ranges := make(map[string]*edRange, len(stats))
	for s, v := range stats {
		ranges[s] = &edRange{min: v[0], max: v[1]}
	}
	grantedOut, grantedOrders, err := e.describeGrantedEffects(m, ranges)
	if err != nil {
		return nil, nil, err
	}
	out, orders, err := e.describeRanges(ranges)
	if err != nil {
		return nil, nil, err
	}
	for i, line := range grantedOut {
		index := len(orders)
		for j, order := range orders {
			if order > grantedOrders[i] {
				index = j
				break
			}
		}
		out = slices.Insert(out, index, line)
		orders = slices.Insert(orders, index, grantedOrders[i])
	}
	return out, orders, nil
}

func luaNumber(v float64) string {
	return strconv.FormatFloat(v, 'g', 14, 64)
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

//pob:block:src/Export/Scripts/mods.lua:282-293
func itemExclusive(mod *poe.Mod) bool {
	// mods.lua:291 drops VisceraMod rows, but the pin's shipped ModItemExclusive.lua still carries them; follow the data.
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

//pob:block:src/Classes/Item.lua:1373-1383
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
