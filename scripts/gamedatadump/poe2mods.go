package main

import (
	"fmt"
	"sort"
	"strings"
)

// poe2ModRow is a Mods row under the names PoB-PoE2's Export/spec.lua gives its columns.
type poe2ModRow struct {
	ID                     string `json:"Id"`
	Type                   *int   `json:"ModType"`
	Level                  int    `json:"Level"`
	Stat1                  *int   `json:"Stat1"`
	Stat2                  *int   `json:"Stat2"`
	Stat3                  *int   `json:"Stat3"`
	Stat4                  *int   `json:"Stat4"`
	Stat5                  *int   `json:"Stat5"`
	Stat6                  *int   `json:"Stat6"`
	Stat1ValueMin          int    `json:"Stat1ValueMin"`
	Stat1ValueMax          int    `json:"Stat1ValueMax"`
	Stat2ValueMin          int    `json:"Stat2ValueMin"`
	Stat2ValueMax          int    `json:"Stat2ValueMax"`
	Stat3ValueMin          int    `json:"Stat3ValueMin"`
	Stat3ValueMax          int    `json:"Stat3ValueMax"`
	Stat4ValueMin          int    `json:"Stat4ValueMin"`
	Stat4ValueMax          int    `json:"Stat4ValueMax"`
	Stat5ValueMin          int    `json:"Stat5ValueMin"`
	Stat5ValueMax          int    `json:"Stat5ValueMax"`
	Stat6ValueMin          int    `json:"Stat6ValueMin"`
	Stat6ValueMax          int    `json:"Stat6ValueMax"`
	Domain                 int    `json:"Domain"`
	Name                   string `json:"Name"`
	GenerationType         int    `json:"GenerationType"`
	Family                 []int  `json:"Families"`
	SpawnTags              []int  `json:"SpawnWeight_Tags"`
	SpawnWeight            []int  `json:"SpawnWeight_Values"`
	Tags                   []int  `json:"Tags"`
	GenerationWeightTags   []int  `json:"GenerationWeight_Tags"`
	GenerationWeightValues []int  `json:"GenerationWeight_Values"`
	ImplicitTags           []int  `json:"ImplicitTags"`
	NodeType               int    `json:"RadiusJewelType"`
	// spec.lua's IsUnscalable is the column go-pob-data's schema calls IsEssenceOnlyModifier
	IsUnscalable bool `json:"IsEssenceOnlyModifier"`
}

type poe2ModFamily struct {
	ID string `json:"Id"`
}

type poe2ModRowStat struct {
	stat     *int
	min, max int
}

func (m *poe2ModRow) stats() [6]poe2ModRowStat {
	return [6]poe2ModRowStat{
		{m.Stat1, m.Stat1ValueMin, m.Stat1ValueMax}, {m.Stat2, m.Stat2ValueMin, m.Stat2ValueMax},
		{m.Stat3, m.Stat3ValueMin, m.Stat3ValueMax}, {m.Stat4, m.Stat4ValueMin, m.Stat4ValueMax},
		{m.Stat5, m.Stat5ValueMin, m.Stat5ValueMax}, {m.Stat6, m.Stat6ValueMin, m.Stat6ValueMax},
	}
}

//pob2:block:src/Export/Scripts/mods.lua:8-24
const (
	poe2DomainGenericMod    = 1
	poe2DomainFlaskCharm    = 2
	poe2DomainJewel         = 11
	poe2DomainUniqueJewel   = 22
	poe2DomainVeiled        = 28
	poe2DomainIncursionLimb = 37

	poe2GenPrefix     = 1
	poe2GenSuffix     = 2
	poe2GenIntrinsic  = 3
	poe2GenCorruption = 5
)

type poe2ModsExport struct {
	describer *poe2Describer
	mods      []poe2ModRow
	stats     []poe2Stat
	modTypes  []poe2ModType
	families  []poe2ModFamily
	tags      []poe2Tag
}

func newPoE2ModsExport() (*poe2ModsExport, error) {
	x := &poe2ModsExport{}
	var err error
	poe2TableInto(&x.mods, "Mods", &err)
	poe2TableInto(&x.stats, "Stats", &err)
	poe2TableInto(&x.modTypes, "ModType", &err)
	poe2TableInto(&x.families, "ModFamily", &err)
	poe2TableInto(&x.tags, "Tags", &err)
	if err != nil {
		return nil, err
	}
	x.describer, err = poe2LoadStatFile("stat_descriptions")
	return x, err
}

//pob2:src/Export/statdesc.lua:420-433
func (x *poe2ModsExport) describeMod(mod *poe2ModRow) ([]string, []float64, string, error) {
	stats := map[string]*edRange{}
	for _, s := range mod.stats() {
		if s.stat != nil {
			stats[x.stats[*s.stat].ID] = &edRange{min: float64(s.min), max: float64(s.max)}
		}
	}
	out, orders, err := x.describer.describeStats(stats)
	return out, orders, x.describeModTags(mod.ImplicitTags), err
}

//pob2:src/Export/statdesc.lua:140-155
func (x *poe2ModsExport) describeModTags(modTags []int) string {
	modTagsText := ""
	for _, t := range modTags {
		if len(modTagsText) > 0 {
			modTagsText += ", "
		}
		modTagsText += `"` + x.tags[t].ID + `"`
	}
	return modTagsText
}

//pob2:src/Export/Scripts/mods.lua:39-44
func (x *poe2ModsExport) getSpawnWeightTag(mod *poe2ModRow, tag int) string {
	if mod.Domain == poe2DomainJewel && x.tags[tag].ID == "default" {
		return "jewel"
	}
	return x.tags[tag].ID
}

//pob2:src/Modules/Common.lua:1107-1118
func hashStats(stats []string, extraStat string) uint32 {
	if extraStat != "" {
		stats = append(append([]string{}, stats...), extraStat)
	}
	statHashes := make([]byte, 0, 4*len(stats))
	for _, statName := range stats {
		statHashes = append(statHashes, intToBytes(murmurHash2([]byte(statName), 0xC58F1A7B))...)
	}
	return murmurHash2(statHashes, 0x02312233)
}

func luaInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = fmt.Sprint(n)
	}
	return strings.Join(parts, ", ")
}

//pob2:block:src/Export/Scripts/mods.lua:46-211
//nolint:gocyclo // Mirrors the Lua body.
func (x *poe2ModsExport) writeMods(condFunc func(mod *poe2ModRow) bool) (string, error) {
	var out strings.Builder
	w := func(parts ...string) {
		for _, p := range parts {
			out.WriteString(p)
		}
	}
	w("-- This file is automatically generated, do not edit!\n")
	w("-- Item data (c) Grinding Gear Games\n\nreturn {\n")
	for i := range x.mods {
		mod := &x.mods[i]
		if !condFunc(mod) {
			continue
		}
		stats, orders, modTags, err := x.describeMod(mod)
		if err != nil {
			return "", fmt.Errorf("%s: %w", mod.ID, err)
		}
		if len(orders) == 0 {
			continue
		}
		w("\t[\"", mod.ID, "\"] = { ")
		switch mod.GenerationType {
		case poe2GenPrefix:
			w(`type = "Prefix", `)
		case poe2GenSuffix:
			w(`type = "Suffix", `)
		case poe2GenIntrinsic:
			if strings.Contains(mod.ID, "SpecialCorruption") {
				w(`type = "SpecialCorrupted", `)
			}
		case poe2GenCorruption:
			w(`type = "Corrupted", `)
		}
		w(`affix = "`, mod.Name, `", `)
		for index, value := range mod.Family {
			if strings.Contains(x.families[value].ID, "LocalDisplayNearbyEnemy") && len(stats) > index+1 && len(orders) > index+1 {
				stats = append(stats[:index:index], stats[index+1:]...)
				orders = append(orders[:index:index], orders[index+1:]...)
				break
			}
		}
		w(`"`, strings.Join(stats, `", "`), `", `)
		orderStrs := make([]string, len(orders))
		for i, o := range orders {
			orderStrs[i] = luaNumStr(o)
		}
		w("statOrder = { ", strings.Join(orderStrs, ", "), " }, ")
		if mod.Type == nil {
			return "", fmt.Errorf("%s has no mod type", mod.ID)
		}
		w("level = ", fmt.Sprint(mod.Level), `, group = "`, x.modTypes[*mod.Type].Name, `", `)
		w("weightKey = { ")
		for _, tag := range mod.SpawnTags {
			w(`"`, x.getSpawnWeightTag(mod, tag), `", `)
		}
		w("}, ")
		w("weightVal = { ", luaInts(mod.SpawnWeight), " }, ")
		if len(mod.GenerationWeightTags) > 0 {
			w("weightMultiplierKey = { ")
			for _, tag := range mod.GenerationWeightTags {
				w(`"`, x.tags[tag].ID, `", `)
			}
			w("}, ")
			w("weightMultiplierVal = { ", luaInts(mod.GenerationWeightValues), " }, ")
		}
		if len(mod.Tags) > 0 {
			w("tags = { ")
			for _, tag := range mod.Tags {
				w(`"`, x.tags[tag].ID, `", `)
			}
			w("}, ")
		}
		w("modTags = { ", modTags, " }, ")
		if mod.IsUnscalable {
			w("unscalable = true, ")
		}
		if mod.NodeType != 3 {
			w("nodeType = ", fmt.Sprint(mod.NodeType), ", ")
		}

		tradeHashes := map[uint32][]string{}
		modStats := mod.stats()
		for statIdx := range 6 {
			if modStats[statIdx].stat == nil {
				break
			}
			stat := x.stats[*modStats[statIdx].stat]
			statEntry := x.describer.descriptors[stat.ID]
			// skip stats that are missing fields. these are most likely
			// hidden stats or e.g. map stats
			if statEntry == nil || statEntry.stats == nil {
				continue
			}
			currentStats := map[string]*edRange{}
			for _, statID := range statEntry.stats {
				for _, ms := range modStats {
					if ms.stat != nil && x.stats[*ms.stat].ID == statID {
						currentStats[statID] = &edRange{min: float64(ms.min), max: float64(ms.max)}
					}
				}
			}
			var prefix, extraStat string
			switch mod.NodeType {
			case 2:
				extraStat = "local_jewel_mod_stats_added_to_notable_passives"
				prefix = "Notable Passive Skills in Radius also grant "
			case 1:
				extraStat = "local_jewel_mod_stats_added_to_small_passives"
				prefix = "Small Passive Skills in Radius also grant "
			}
			description, _, err := x.describer.describeStats(currentStats)
			if err != nil {
				return "", fmt.Errorf("%s: %w", mod.ID, err)
			}
			if prefix != "" {
				for i, v := range description {
					description[i] = prefix + v
				}
			}
			tradeHashes[hashStats(statEntry.stats, extraStat)] = description
		}
		// Lua writes these in pairs() order; ascending hash here
		w("tradeHashes = { ")
		hashes := make([]uint32, 0, len(tradeHashes))
		for h := range tradeHashes {
			hashes = append(hashes, h)
		}
		sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
		for _, h := range hashes {
			w(fmt.Sprintf("[%d] = { %s }, ", h, `"`+strings.Join(tradeHashes[h], `", "`)+`"`))
		}
		w("} ")
		w("},\n")
	}
	w("}")
	return out.String(), nil
}

func (x *poe2ModsExport) family1(mod *poe2ModRow) string {
	if len(mod.Family) == 0 {
		return ""
	}
	return x.families[mod.Family[0]].ID
}

// poe2ModFiles are mods.lua's writeMods calls, by output file.
//
//pob2:block:src/Export/Scripts/mods.lua:213-249
func (x *poe2ModsExport) poe2ModFiles() []struct {
	name string
	cond func(mod *poe2ModRow) bool
} {
	notAura := func(mod *poe2ModRow) bool { return len(mod.Family) == 0 || x.family1(mod) != "AuraBonus" }
	affix := func(mod *poe2ModRow) bool {
		return mod.GenerationType == poe2GenPrefix || mod.GenerationType == poe2GenSuffix
	}
	return []struct {
		name string
		cond func(mod *poe2ModRow) bool
	}{
		{"Data/ModItem.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainGenericMod && affix(mod) && notAura(mod) &&
				!strings.Contains(mod.ID, "Cowards") && !strings.Contains(mod.ID, "Master")
		}},
		{"Data/ModCorrupted.lua", func(mod *poe2ModRow) bool {
			return (mod.Domain == poe2DomainJewel || mod.Domain == poe2DomainGenericMod) &&
				(mod.GenerationType == poe2GenIntrinsic && strings.Contains(mod.ID, "SpecialCorruption") || mod.GenerationType == poe2GenCorruption)
		}},
		{"Data/ModFlask.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainFlaskCharm && affix(mod) && strings.HasPrefix(mod.ID, "Flask")
		}},
		{"Data/ModCharm.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainFlaskCharm && ((mod.GenerationType == poe2GenPrefix && strings.HasPrefix(mod.ID, "Charm")) ||
				(mod.GenerationType == poe2GenSuffix && !strings.Contains(mod.ID, "Immunity")))
		}},
		// Domains.FlaskCharm1 is nil in the Lua, so its intrinsic half never matches
		{"Data/ModJewel.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainJewel && affix(mod)
		}},
		{"Data/ModIncursionLimb.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainIncursionLimb && mod.GenerationType == poe2GenIntrinsic
		}},
		// contains primarily uniques and items implicits but also other mods only available on a single base or unique.
		{"Data/ModItemExclusive.lua", func(mod *poe2ModRow) bool {
			return (mod.Domain == poe2DomainGenericMod || mod.Domain == poe2DomainFlaskCharm || mod.Domain == poe2DomainJewel || mod.Domain == poe2DomainUniqueJewel) &&
				mod.GenerationType == poe2GenIntrinsic && notAura(mod) &&
				!strings.HasPrefix(mod.ID, "Synthesis") && !strings.Contains(mod.ID, "Royale") && !strings.Contains(mod.ID, "Cowards") &&
				!strings.Contains(mod.ID, "Map") && !strings.Contains(mod.ID, "Ultimatum") && !strings.Contains(mod.ID, "SpecialCorruption")
		}},
		{"Data/ModVeiled.lua", func(mod *poe2ModRow) bool {
			return mod.Domain == poe2DomainVeiled && !strings.Contains(mod.ID, "Map")
		}},
	}
}

func poe2ModsLua() (map[string]string, error) {
	x, err := newPoE2ModsExport()
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for _, f := range x.poe2ModFiles() {
		text, err := x.writeMods(f.cond)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.name, err)
		}
		files[f.name] = text
	}
	return files, nil
}

func dumpPoE2Mods() (any, error) {
	files, err := poe2ModsLua()
	if err != nil {
		return nil, err
	}
	if len(files["Data/ModItem.lua"]) < 100000 {
		return nil, fmt.Errorf("Data/ModItem.lua holds %d bytes; the Mods table did not load", len(files["Data/ModItem.lua"]))
	}
	return files, nil
}
