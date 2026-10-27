package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type poe2SoulCore struct {
	BaseItemType                   int  `json:"BaseItemType"`  // BaseItemTypes
	LevelReq                       int  `json:"RequiredLevel"` // LevelReq
	Limit                          *int `json:"Limit"`
	Type                           int  `json:"Type"`
	IsSocketBound                  bool `json:"IsSocketBound"`
	CanSocketInChakraSlots         bool `json:"CanSocketInMartialArtistSlots"` // CanSocketInChakraSlots
	CanSocketInUniqueItems         bool `json:"CanSocketInUniqueItems"`
	CanSocketInJewellery           bool `json:"CanSocketInJewellery"`
	CanSocketInCorruptedSanctified bool `json:"CanSocketInCorruptedSanctified"`
}

type poe2SoulCoreStat struct {
	SoulCore     int   `json:"SoulCore"`     // Id
	Category     int   `json:"StatCategory"` // Category
	Stats        []int `json:"Stats"`
	StatValue    []int `json:"StatsValues"` // StatValue
	BondedStats  []int `json:"BondedStats"`
	BondedValues []int `json:"BondedStatsValues"` // BondedValues
}

type poe2SoulCoreLimit struct {
	ID    string `json:"Id"`
	Limit int    `json:"Limit"`
}

type poe2SoulCoreID struct {
	ID string `json:"Id"`
}

type poe2HashedStat struct {
	ID          string `json:"Id"`
	Hash        uint32 `json:"HASH32"`        // Hash
	Local       bool   `json:"IsLocal"`       // Local
	WeaponLocal bool   `json:"IsWeaponLocal"` // WeaponLocal
}

//pob2:block:src/Export/Scripts/soulcores.lua:6-18
var poe2SoulCoreClassMap = map[string][]string{
	"Martial Weapon":                {"weapon"},
	"Caster Weapon":                 {"caster"},
	"Martial Or Caster Weapon":      {"weapon", "caster"},
	"Armour":                        {"armour"},
	"Wand or Staff":                 {"wand", "staff"},
	"Maces or Talisman":             {"one hand mace", "two hand mace", "talisman"},
	"One Hand Mace or Quarterstaff": {"one hand mace", "quarterstaff"},
	"Shield or Buckler":             {"shield", "buckler"},
	"All":                           {"weapon", "armour", "caster"},
	"Quarterstaff or Spear":         {"quarterstaff", "spear"},
	"Crossbow Bow or Spear":         {"crossbow", "bow", "spear"},
}

type poe2SoulCoresExport struct {
	describer      *poe2Describer
	baseItemTypes  []poe2BaseItemType
	stats          []poe2HashedStat
	soulCores      []poe2SoulCore
	soulCoreStats  []poe2SoulCoreStat
	soulCoreTypes  []poe2SoulCoreID
	soulCoreLimits []poe2SoulCoreLimit
	statCategories []poe2SoulCoreID
}

type poe2SoulCoreModLine struct {
	typ                                                  string
	canSocketInChakraSlots, canSocketInUniqueItems       bool
	canSocketInJewellery, canSocketInCorruptedSanctified bool
	limit                                                *int
	limitID                                              string
	localMod                                             bool
	slotType                                             string
	label                                                []string
	statOrder                                            []float64
	bondedLabel                                          []string
	bondedStatOrder                                      []float64
	levelReq                                             int
	tradeHashes                                          map[uint32][]string
	isSocketBound                                        bool
}

//pob2:src/Modules/Common.lua:350-357
func intToBytes(v uint32) []byte {
	return []byte{byte(v & 0xFF), byte(v >> 8 & 0xFF), byte(v >> 16 & 0xFF), byte(v >> 24 & 0xFF)} //nolint:gosec // masked to a byte
}

//pob2:src/Modules/Common.lua:359-390
func murmurHash2(key []byte, seed uint32) uint32 {
	mix := func(v uint32) uint32 { return v * 0x5BD1E995 }
	n := len(key)
	h := seed ^ uint32(n) //nolint:gosec // a key length
	o := 0
	for n >= 4 {
		k := uint32(key[o]) | uint32(key[o+1])<<8 | uint32(key[o+2])<<16 | uint32(key[o+3])<<24
		k = mix(k)
		k ^= k >> 24
		k = mix(k)
		h = mix(h)
		h ^= k
		o += 4
		n -= 4
	}
	if n > 0 {
		var k uint32
		for i := range n {
			k |= uint32(key[o+i]) << (8 * i)
		}
		h ^= k
		h = mix(h)
	}
	h ^= h >> 13
	h = mix(h)
	h ^= h >> 15
	return h
}

func newPoE2SoulCoresExport() (*poe2SoulCoresExport, error) {
	x := &poe2SoulCoresExport{}
	var err error
	poe2TableInto(&x.baseItemTypes, "BaseItemTypes", &err)
	poe2TableInto(&x.stats, "Stats", &err)
	poe2TableInto(&x.soulCores, "SoulCores", &err)
	poe2TableInto(&x.soulCoreStats, "SoulCoreStats", &err)
	poe2TableInto(&x.soulCoreTypes, "SoulCoreTypes", &err)
	poe2TableInto(&x.soulCoreLimits, "SoulCoreLimits", &err)
	poe2TableInto(&x.statCategories, "SoulCoreStatCategories", &err)
	if err != nil {
		return nil, err
	}
	x.describer, err = poe2LoadStatFile("stat_descriptions")
	return x, err
}

func (x *poe2SoulCoresExport) describe(stats []int, values []int) ([]string, []float64, error) {
	m := map[string]*edRange{}
	for i, s := range stats {
		if i >= len(values) {
			return nil, nil, fmt.Errorf("stat %s has no value", x.stats[s].ID)
		}
		m[x.stats[s].ID] = &edRange{min: float64(values[i]), max: float64(values[i])}
	}
	return x.describer.describeStats(m)
}

//pob2:block:src/Export/Scripts/soulcores.lua:35-182
func (x *poe2SoulCoresExport) base(args string, out *strings.Builder) error {
	w := func(parts ...string) {
		for _, p := range parts {
			_, _ = out.WriteString(p)
		}
	}
	baseTypeID, displayName := args, ""
	if m := baseDirectiveRe.FindStringSubmatch(args); m != nil {
		baseTypeID, displayName = m[1], m[2]
	}
	baseIdx := -1
	for i := range x.baseItemTypes {
		if x.baseItemTypes[i].ID == baseTypeID {
			baseIdx = i
			break
		}
	}
	if baseIdx < 0 {
		fmt.Fprintf(os.Stderr, "Invalid Id %s\n", baseTypeID)
		return nil
	}
	if displayName == "" {
		displayName = x.baseItemTypes[baseIdx].Name
	}
	if strings.Contains(displayName, "DNT") {
		return nil
	}
	displayName = strings.ReplaceAll(displayName, "ö", "o")
	displayName = strings.Trim(displayName, " \t\n\v\f\r") // trim spaces GGG might leave in by accident

	// Check for Standard Weapon, Armour, Caster Runes
	soulCores := poe2RowBy(x.soulCores, func(r *poe2SoulCore) int { return r.BaseItemType }, baseIdx)
	if soulCores == nil {
		return fmt.Errorf("%s has no SoulCores row", baseTypeID)
	}
	soulCoresIdx := 0
	for i := range x.soulCores {
		if &x.soulCores[i] == soulCores {
			soulCoresIdx = i
		}
	}
	w("\t[\"", displayName, "\"] = {\n")
	// Regular and Bonded stats may be separate data rows for the same slot. Keep an
	// ordered output list and a slot lookup so each Lua key is emitted exactly once.
	var modLines []*poe2SoulCoreModLine
	modLinesBySlot := map[string]*poe2SoulCoreModLine{}
	for i := range x.soulCoreStats {
		soulCoreStat := &x.soulCoreStats[i]
		if soulCoreStat.SoulCore != soulCoresIdx {
			continue
		}
		if len(soulCoreStat.Stats) == 0 && len(soulCoreStat.BondedStats) == 0 {
			continue
		}
		category := x.statCategories[soulCoreStat.Category].ID
		classes, ok := poe2SoulCoreClassMap[category]
		if !ok {
			classes = []string{strings.ToLower(category)}
		}
		for _, class := range classes {
			descStats, orders, err := x.describe(soulCoreStat.Stats, soulCoreStat.StatValue)
			if err != nil {
				return fmt.Errorf("%s: %w", displayName, err)
			}
			descBondedStats, bondedOrders, err := x.describe(soulCoreStat.BondedStats, soulCoreStat.BondedValues)
			if err != nil {
				return fmt.Errorf("%s: %w", displayName, err)
			}
			if len(orders) == 0 && len(bondedOrders) == 0 {
				continue
			}
			tradeHashes := map[uint32][]string{}
			localMod := true
			for modIdx := 0; modIdx < len(soulCoreStat.Stats); modIdx++ {
				stat := x.stats[soulCoreStat.Stats[modIdx]]
				if !(stat.Local || stat.WeaponLocal) {
					localMod = false
				}
				currentStats, currentValues := []int{soulCoreStat.Stats[modIdx]}, []int{soulCoreStat.StatValue[modIdx]}
				bytes := intToBytes(stat.Hash)
				// # to # stats consist of two different stats as the min and max have different ranges
				if strings.Contains(stat.ID, "minimum") && modIdx+1 < len(soulCoreStat.Stats) {
					if nextStat := x.stats[soulCoreStat.Stats[modIdx+1]]; strings.Contains(nextStat.ID, "maximum") {
						modIdx++
						bytes = append(bytes, intToBytes(nextStat.Hash)...)
						currentStats = append(currentStats, soulCoreStat.Stats[modIdx])
						currentValues = append(currentValues, soulCoreStat.StatValue[modIdx])
					}
				}
				description, _, err := x.describe(currentStats, currentValues)
				if err != nil {
					return fmt.Errorf("%s: %w", displayName, err)
				}
				tradeHashes[murmurHash2(bytes, 0x02312233)] = description
			}
			modLine := modLinesBySlot[class]
			if modLine == nil {
				var limit *int
				var limitID string
				if soulCores.Limit != nil {
					l := x.soulCoreLimits[*soulCores.Limit]
					limit, limitID = &l.Limit, l.ID
				}
				modLine = &poe2SoulCoreModLine{
					typ:                            x.soulCoreTypes[soulCores.Type].ID,
					canSocketInChakraSlots:         soulCores.CanSocketInChakraSlots,
					canSocketInUniqueItems:         soulCores.CanSocketInUniqueItems,
					canSocketInJewellery:           soulCores.CanSocketInJewellery,
					canSocketInCorruptedSanctified: soulCores.CanSocketInCorruptedSanctified,
					limit:                          limit,
					limitID:                        limitID,
					localMod:                       localMod,
					slotType:                       class,
					label:                          descStats,
					statOrder:                      orders,
					bondedLabel:                    descBondedStats,
					bondedStatOrder:                bondedOrders,
					levelReq:                       soulCores.LevelReq,
					tradeHashes:                    tradeHashes,
					isSocketBound:                  soulCores.IsSocketBound,
				}
				modLinesBySlot[class] = modLine
				modLines = append(modLines, modLine)
			} else {
				modLine.localMod = modLine.localMod && localMod
				modLine.label = append(modLine.label, descStats...)
				modLine.statOrder = append(modLine.statOrder, orders...)
				modLine.bondedLabel = append(modLine.bondedLabel, descBondedStats...)
				modLine.bondedStatOrder = append(modLine.bondedStatOrder, bondedOrders...)
				for hash, desc := range tradeHashes {
					modLine.tradeHashes[hash] = desc
				}
			}
		}
	}

	joinNums := func(v []float64) string {
		parts := make([]string, len(v))
		for i, f := range v {
			parts[i] = luaNumStr(f)
		}
		return strings.Join(parts, ", ")
	}
	for _, modLine := range modLines {
		w("\t\t[\"", modLine.slotType, "\"] = {\n")
		w("\t\t\t\ttype = \"", modLine.typ, "\",\n")
		if modLine.limit != nil {
			w("\t\t\t\tlimit = ", luaNumStr(float64(*modLine.limit)), ",\n")
			if modLine.limitID != "GenericLimit1" {
				w("\t\t\t\tlimitId = \"", modLine.limitID, "\",\n")
			}
		}
		w("\t\t\t\tlocalMod = ", fmt.Sprint(modLine.localMod), ",\n")
		if len(modLine.label) > 0 {
			w("\t\t\t\t\"", strings.Join(modLine.label, "\",\n\t\t\t\t\""), "\",\n")
			w("\t\t\t\tstatOrder = { ", joinNums(modLine.statOrder), " },\n")
		}
		// Lua writes these in pairs() order; ascending hash here
		w("\t\t\t\ttradeHashes = { ")
		hashes := make([]uint32, 0, len(modLine.tradeHashes))
		for h := range modLine.tradeHashes {
			hashes = append(hashes, h)
		}
		sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
		for _, h := range hashes {
			w(fmt.Sprintf("[%d] = { %s }, ", h, `"`+strings.Join(modLine.tradeHashes[h], `", "`)+`"`))
		}
		w(" },\n")
		if len(modLine.bondedLabel) > 0 {
			w("\t\t\t\tbonded = {\n")
			w("\t\t\t\t\t\"", strings.Join(modLine.bondedLabel, "\",\n\t\t\t\t\t\""), "\",\n")
			w("\t\t\t\t\tstatOrder = { ", joinNums(modLine.bondedStatOrder), " },\n")
			w("\t\t\t\t},\n")
		}
		for _, f := range []struct {
			name string
			on   bool
		}{{"isSocketBound", modLine.isSocketBound}, {"canSocketInChakraSlots", modLine.canSocketInChakraSlots},
			{"canSocketInUniqueItems", modLine.canSocketInUniqueItems}, {"canSocketInJewellery", modLine.canSocketInJewellery},
			{"canSocketInCorruptedSanctified", modLine.canSocketInCorruptedSanctified}} {
			if f.on {
				w("\t\t\t\t", f.name, " = true,\n")
			}
		}
		w("\t\t\t\tlevelReq = ", luaNumStr(float64(modLine.levelReq)), ",\n")
		w("\t\t},\n")
	}
	w("\t},\n")
	return nil
}

//pob2:block:src/Export/Scripts/soulcores.lua:184-200
func (x *poe2SoulCoresExport) baseMatch(argstr string, out *strings.Builder) error {
	args := strings.Fields(argstr)
	if len(args) == 0 {
		return nil
	}
	// Default to look at the Id column for matching
	key, value := "Id", args[0]
	// If column name is specified, use that
	if len(args) > 1 {
		key, value = args[0], args[1]
	}
	if key != "Id" {
		return fmt.Errorf("baseMatch column %s is not modelled", key)
	}
	re, err := luaPatternRe(value)
	if err != nil {
		return err
	}
	for _, t := range x.baseItemTypes {
		if re.MatchString(t.ID) {
			if err := x.base(t.ID, out); err != nil {
				return err
			}
		}
	}
	return nil
}

//pob2:block:src/Export/Scripts/soulcores.lua:202-220
func poe2ModRunesLua(pobDir string) (string, error) {
	x, err := newPoE2SoulCoresExport()
	if err != nil {
		return "", err
	}
	text, err := os.ReadFile(pobDir + "/src/Export/Bases/soulcore.txt") //nolint:gosec // the pinned PoB-PoE2 checkout
	if err != nil {
		return "", fmt.Errorf("template soulcore: %w", err)
	}
	var out strings.Builder
	out.WriteString("-- spell-checker: disable\n")
	out.WriteString("-- This file is automatically generated, do not edit!\n")
	out.WriteString("-- Item data (c) Grinding Gear Games\n\nreturn {\n")
	for _, line := range strings.Split(string(text), "\n") {
		m := templateLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch m[1] {
		case "type":
		case "base":
			err = x.base(m[2], &out)
		case "baseMatch":
			err = x.baseMatch(m[2], &out)
		default:
			fmt.Fprintf(os.Stderr, "Unknown directive '%s'\n", m[1])
		}
		if err != nil {
			return "", err
		}
	}
	out.WriteString("}")
	return out.String(), nil
}

func dumpPoE2ModRunes() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the soul core template is the pinned PoB-PoE2's Export/Bases/soulcore.txt")
	}
	text, err := poe2ModRunesLua(pobDir)
	if err != nil {
		return nil, err
	}
	return map[string]string{"Data/ModRunes.lua": text}, nil
}
