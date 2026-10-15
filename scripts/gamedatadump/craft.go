package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
)

type craftEssence struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Type int               `json:"type"`
	Tier int               `json:"tier"`
	Mods map[string]string `json:"mods"`
}

type craftCrucible struct {
	exportMod
	Tier         int   `json:"tier"`
	NodeLocation []int `json:"nodeLocation"`
}

var craftModTables = []struct {
	key, outName string
	cond         func(*poe.Mod) bool
}{
	{"Explicit", "../Data/ModExplicit.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainItem && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix) &&
			!strings.Contains(mod.ID, "Royale") && !strings.Contains(mod.ID, "Necropolis") && !strings.HasPrefix(mod.ID, "Synthesis") &&
			!delveNames[mod.Name] &&
			!(mod.GenerationType == genSearingExarch || mod.GenerationType == genEaterOfWorlds) && !modHasAuraFlags(mod)
	}},
	{"Corrupted", "../Data/ModCorrupted.lua", func(mod *poe.Mod) bool {
		return mod.GenerationType == genCorrupted && mod.Domain == domainItem
	}},
	{"Delve", "../Data/ModDelve.lua", func(mod *poe.Mod) bool { return mod.Domain == domainDelveFossil || delveNames[mod.Name] }},
	{"Synthesis", "../Data/ModSynthesis.lua", func(mod *poe.Mod) bool {
		return mod.GenerationType == genIntrinsic && mod.Domain == domainItem && strings.HasPrefix(mod.ID, "Synthesis")
	}},
	{"Scourge", "../Data/ModScourge.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainItem && (mod.GenerationType == genScourgeBenefit || mod.GenerationType == genScourgeDownside) &&
			!hellscapeMapRe.MatchString(mod.ID)
	}},
	{"Eldritch", "../Data/ModEldritch.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainItem && (mod.GenerationType == genSearingExarch || mod.GenerationType == genEaterOfWorlds)
	}},
	{"Flask", "../Data/ModFlask.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainFlask && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)
	}},
	{"Tincture", "../Data/ModTincture.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainTincture && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)
	}},
	{"Jewel", "../Data/ModJewel.lua", func(mod *poe.Mod) bool {
		return (mod.Domain == domainJewel || mod.Domain == domainDelveFossil) &&
			(mod.GenerationType == genPrefix || mod.GenerationType == genSuffix || mod.GenerationType == genCorrupted)
	}},
	{"JewelAbyss", "../Data/ModJewelAbyss.lua", func(mod *poe.Mod) bool {
		return (mod.Domain == domainAbyssJewel || mod.Domain == domainDelveFossil) &&
			(mod.GenerationType == genPrefix || mod.GenerationType == genSuffix || mod.GenerationType == genCorrupted)
	}},
	{"JewelCluster", "../Data/ModJewelCluster.lua", func(mod *poe.Mod) bool {
		return (mod.Domain == domainClusterJewel && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)) ||
			(mod.Domain == domainJewel && mod.GenerationType == genCorrupted)
	}},
	{"JewelCharm", "../Data/ModJewelCharm.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainCharm && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)
	}},
	{"ItemExclusive", "../Data/ModItemExclusive.lua", itemExclusive},
	{"Graft", "../Data/ModGraft.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainGraft && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix || mod.GenerationType == genCorrupted)
	}},
	{"Foulborn", "../Data/ModFoulborn.lua", func(mod *poe.Mod) bool {
		return (mod.Domain == domainItem || mod.Domain == domainJewel) && mod.GenerationType == genIntrinsic && strings.HasPrefix(mod.ID, "MutatedUnique")
	}},
	{"Mercenary", "../Data/ModMercenary.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainMercenary && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)
	}},
}

var hellscapeMapRe = regexp.MustCompile(`^Hellscape[UpDown]+sideMap`)

//pob:block:src/Export/Scripts/mods.lua:207-210
var delveNames = map[string]bool{"Subterranean": true, "of the Underground": true}

//pob:block:src/Export/Scripts/mods.lua:222
func modHasAuraFlags(mod *poe.Mod) bool { return len(mod.AuraFlags) > 0 }

var craftItemTables = []string{"Explicit", "ItemExclusive", "Corrupted", "Delve", "Synthesis", "Scourge", "Eldritch", "Mercenary"}

//pob:block:src/Modules/Data.lua:649-716
func dumpItemCraftMods() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the base and rare templates and Data.lua's lists are the pinned PoB's")
	}
	x, err := newModsExport()
	if err != nil {
		return nil, err
	}
	tables := map[string][]exportMod{}
	for _, t := range craftModTables {
		mods, err := x.writeMods(t.outName, t.cond)
		if err != nil {
			return nil, err
		}
		tables[t.key] = lastByID(mods)
	}
	var item []exportMod
	seen := map[string]int{}
	for _, key := range craftItemTables {
		for _, m := range tables[key] {
			if i, ok := seen[m.ID]; ok {
				item[i] = m
				continue
			}
			seen[m.ID] = len(item)
			item = append(item, m)
		}
	}
	tables["Item"] = item

	essences, err := dumpEssences()
	if err != nil {
		return nil, err
	}
	essenceMods := map[string]bool{}
	for _, e := range essences {
		for _, id := range e.Mods {
			essenceMods[id] = true
		}
	}
	for _, r := range data.Rares() {
		for _, m := range craftRareAffixRe.FindAllStringSubmatch(r.Raw, -1) {
			essenceMods[m[1]] = true
		}
	}
	spawns := func(m exportMod) bool {
		if essenceMods[m.ID] || strings.Contains(m.Affix, "Elevated") {
			return true
		}
		for _, w := range m.WeightVal {
			if w > 0 {
				return true
			}
		}
		return false
	}
	corruptTypes := map[string]bool{"Corrupted": true, "ScourgeUpside": true, "ScourgeDownside": true, "Exarch": true, "Eater": true, "DelveImplicit": true}
	affixTables := map[string][]exportMod{}
	for name, mods := range tables {
		list := []exportMod{}
		if craftSelectable[name] {
			for _, m := range mods {
				if corruptTypes[m.Type] || ((m.Type == "Prefix" || m.Type == "Suffix") && spawns(m)) {
					list = append(list, m)
				}
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
		affixTables[name] = list
	}

	byID := func(mods []exportMod) []exportMod {
		sort.Slice(mods, func(i, j int) bool { return mods[i].ID < mods[j].ID })
		return mods
	}
	veiled, err := x.writeMods("../Data/ModVeiled.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainUnveiled && (mod.GenerationType == genPrefix || mod.GenerationType == genSuffix)
	})
	if err != nil {
		return nil, err
	}
	necropolis, err := x.writeMods("../Data/ModNecropolis.lua", func(mod *poe.Mod) bool {
		return mod.Domain == domainItem && strings.HasPrefix(mod.ID, "NecropolisCrafting")
	})
	if err != nil {
		return nil, err
	}
	beast, err := x.writeMods("../Data/BeastCraft.lua", func(mod *poe.Mod) bool {
		return strings.Contains(mod.ID, "Aspect") && mod.GenerationType == genSuffix
	})
	if err != nil {
		return nil, err
	}
	crucible, err := dumpCrucible()
	if err != nil {
		return nil, err
	}
	masterMods, err := dumpMasterMods()
	if err != nil {
		return nil, err
	}
	//pob:block:src/Modules/Data.lua:667-673
	var vestigial []exportMod
	for _, m := range tables["ItemExclusive"] {
		if strings.HasPrefix(m.ID, "Divergent") {
			vestigial = append(vestigial, m)
		}
	}
	affixTables["Vestigial"] = byID(vestigial)
	//pob:block:src/Classes/ItemsTab.lua:3151-3155
	var specialCorruption []exportMod
	for _, m := range tables["ItemExclusive"] {
		if strings.HasPrefix(m.ID, "UniqueSpecialCorruption") {
			specialCorruption = append(specialCorruption, m)
		}
	}
	vestigialMappings, err := dumpVestigialModMappings(pobDir)
	if err != nil {
		return nil, err
	}
	dropRestricted := map[string][]craftDropRestricted{}
	for _, d := range []struct{ source, file, table string }{{"DELVE", "DelveDropOnly.lua", "Delve"}, {"INCURSION", "IncursionDropOnly.lua", "Explicit"}} {
		if dropRestricted[d.source], err = dumpDropRestricted(pobDir, d.file, tables[d.table]); err != nil {
			return nil, err
		}
	}
	literals := map[string]*luaTable{}
	for _, name := range []string{"data.enchantmentSource", "data.weaponTypeInfo", "data.casterTagCrucibleUniques", "data.minionTagCrucibleUniques", "data.vestigialUniqueBaseTypes"} {
		if literals[name], err = dataLuaLiteral(pobDir, name); err != nil {
			return nil, err
		}
	}
	sources := []map[string]any{}
	for _, s := range literals["data.enchantmentSource"].arr {
		if t, ok := s.(*luaTable); ok {
			sources = append(sources, map[string]any{"name": t.hash["name"], "label": t.hash["label"]})
		}
	}
	enchantments, err := dumpEnchantments(sortedKeys(literals["data.weaponTypeInfo"].hash))
	if err != nil {
		return nil, err
	}
	bases, err := dumpItemBases(pobDir)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pobVersion": "go-pob-data " + raw.LatestVersion, "affixTables": affixTables,
		"veiledMods": byID(lastByID(veiled)), "necropolisMods": byID(lastByID(necropolis)), "beastCraft": byID(lastByID(beast)), "crucible": crucible, "essences": essences, "masterMods": masterMods,
		"sources": sources, "enchantments": enchantments, "bases": bases,
		"casterTagCrucibleUniques": sortedKeys(literals["data.casterTagCrucibleUniques"].hash),
		"minionTagCrucibleUniques": sortedKeys(literals["data.minionTagCrucibleUniques"].hash),
		"vestigialUniqueBaseTypes": sortedKeys(literals["data.vestigialUniqueBaseTypes"].hash),
		"vestigialModMappings":     vestigialMappings, "specialCorruptionMods": byID(specialCorruption),
		"dropRestricted": dropRestricted,
	}, nil
}

type craftDropRestricted struct {
	exportMod
	Categories []string `json:"categories"`
}

//pob:block:src/Classes/ItemsTab.lua:3513-3514,src/Classes/ItemsTab.lua:3579-3602
func dumpDropRestricted(pobDir, file string, table []exportMod) ([]craftDropRestricted, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Data", file)) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	t, err := luaTableLiteral(string(b), "\nreturn ")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	byID := map[string]exportMod{}
	for _, m := range table {
		byID[m.ID] = m
	}
	var out []craftDropRestricted
	for _, id := range sortedKeys(t.hash) {
		mod, ok := byID[id]
		entry, isTable := t.hash[id].(*luaTable)
		if !ok || !isTable {
			continue
		}
		row := craftDropRestricted{exportMod: mod}
		if cats, ok := entry.hash["categories"].(*luaTable); ok {
			for _, c := range cats.arr {
				if s, ok := c.(string); ok {
					row.Categories = append(row.Categories, s)
				}
			}
		}
		out = append(out, row)
	}
	if len(out) < 10 {
		return nil, fmt.Errorf("%s: %d of its mods resolve", file, len(out))
	}
	return out, nil
}

//pob:block:src/Modules/Data.lua:866-872
func dumpVestigialModMappings(pobDir string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Data", "Vestigial.lua")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("Vestigial.lua: %w", err)
	}
	t, err := luaTableLiteral(string(b), "\nreturn ")
	if err != nil {
		return nil, fmt.Errorf("Vestigial.lua: %w", err)
	}
	out := map[string]string{}
	for id, v := range t.hash {
		if titles, ok := v.(*luaTable); ok && len(titles.arr) > 0 {
			if title, ok := titles.arr[0].(string); ok {
				out[id] = title
			}
		}
	}
	return out, nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dataLuaLiteral(pobDir, name string) (*luaTable, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Modules", "Data.lua")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("Data.lua: %w", err)
	}
	t, err := luaTableLiteral(string(b), "\n"+name+" = ")
	if err != nil {
		return nil, fmt.Errorf("Data.lua: %w", err)
	}
	return t, nil
}

func lastByID(mods []exportMod) []exportMod {
	at := map[string]int{}
	var out []exportMod
	for _, m := range mods {
		if i, ok := at[m.ID]; ok {
			out[i] = m
			continue
		}
		at[m.ID] = len(out)
		out = append(out, m)
	}
	return out
}

var craftRareAffixRe = regexp.MustCompile(`(?:[Pp]refix|[Ss]uffix): (\w+)`)

var craftSelectable = map[string]bool{"Item": true, "Explicit": true, "Flask": true, "Tincture": true, "Graft": true, "Jewel": true, "JewelAbyss": true, "JewelCluster": true, "JewelCharm": true}

//pob:block:src/Export/Scripts/crucible.lua:1-61
func dumpCrucible() ([]craftCrucible, error) {
	ctx, cache := gameDataContext(), assetCache()
	if err := poe.InitializeWeaponPassiveSkills(ctx, raw.LatestVersion, cache); err != nil {
		return nil, fmt.Errorf("WeaponPassiveSkills: %w", err)
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}
	x := &modsExport{describer: describer}
	at := map[string]int{}
	var out []craftCrucible
	for _, crucible := range poe.WeaponPassiveSkills {
		mod := poe.Mods[crucible.Mod]
		if strings.HasSuffix(mod.ID, "HardMode") {
			continue
		}
		stats, orders, err := x.describeMod(mod, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		if len(orders) == 0 {
			continue
		}
		row := craftCrucible{exportMod: exportMod{ID: mod.ID, Lines: stats, StatOrder: orders, Level: mod.Level, Group: poe.ModTypes[mod.ModTypeKey].Name,
			WeightKey: modTags(mod.SpawnWeightTagsKeys), WeightVal: mod.SpawnWeightValues, ModTags: modTags(mod.ImplicitTagsKeys)},
			Tier: crucible.Tier, NodeLocation: crucible.NodeSpawnLocation}
		switch mod.GenerationType {
		case 31:
			row.Type = "Spawn"
		case 32:
			row.Type = "MergeOnly"
		}
		if len(mod.GenerationWeightTagsKeys) > 0 {
			row.WeightMultiplierKey = modTags(mod.GenerationWeightTagsKeys)
			row.WeightMultiplierVal = mod.GenerationWeightValues
		}
		if i, ok := at[mod.ID]; ok {
			out[i] = row
			continue
		}
		at[mod.ID] = len(out)
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

var essenceModColumns = map[string]func(*poe.Essence) *int{
	"Amulet":                     func(e *poe.Essence) *int { return e.AmuletModsKey },
	"Quiver":                     func(e *poe.Essence) *int { return e.DisplayQuiverModsKey },
	"Ring":                       func(e *poe.Essence) *int { return e.RingModsKey },
	"Belt":                       func(e *poe.Essence) *int { return e.BeltModsKey },
	"Helmet":                     func(e *poe.Essence) *int { return e.HelmetModsKey },
	"Body Armour":                func(e *poe.Essence) *int { return e.BodyArmourModsKey },
	"Boots":                      func(e *poe.Essence) *int { return e.BootsModsKey },
	"Gloves":                     func(e *poe.Essence) *int { return e.GlovesModsKey },
	"Shield":                     func(e *poe.Essence) *int { return e.ShieldModsKey },
	"Bow":                        func(e *poe.Essence) *int { return e.BowModsKey },
	"Claw":                       func(e *poe.Essence) *int { return e.ClawModsKey },
	"Dagger":                     func(e *poe.Essence) *int { return e.DaggerModsKey },
	"Staff":                      func(e *poe.Essence) *int { return e.StaffModsKey },
	"Wand":                       func(e *poe.Essence) *int { return e.WandModsKey },
	"One Handed Axe":             func(e *poe.Essence) *int { return e.OneHandAxeModsKey },
	"One Handed Mace":            func(e *poe.Essence) *int { return e.OneHandMaceModsKey },
	"One Handed Sword":           func(e *poe.Essence) *int { return e.OneHandSwordModsKey },
	"Sceptre":                    func(e *poe.Essence) *int { return e.SceptreModsKey },
	"Thrusting One Handed Sword": func(e *poe.Essence) *int { return e.OneHandThrustingSwordModsKey },
	"Two Handed Axe":             func(e *poe.Essence) *int { return e.TwoHandAxeModsKey },
	"Two Handed Mace":            func(e *poe.Essence) *int { return e.TwoHandMaceModsKey },
	"Two Handed Sword":           func(e *poe.Essence) *int { return e.TwoHandSwordModsKey },
}

//pob:block:src/Export/Scripts/essence.lua:25-40
func dumpEssences() ([]craftEssence, error) {
	ctx, cache := gameDataContext(), assetCache()
	for name, init := range map[string]func() error{
		"BaseItemTypes": func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) },
		"Essences":      func() error { return poe.InitializeEssences(ctx, raw.LatestVersion, cache) },
	} {
		if err := init(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	at := map[string]int{}
	var out []craftEssence
	for _, essence := range poe.Essences {
		if essence.Level <= 0 {
			continue
		}
		base := poe.BaseItemTypes[essence.BaseItemTypesKey]
		e := craftEssence{ID: base.ID, Name: base.Name, Type: essence.EssenceTypeKey, Tier: essence.Level, Mods: map[string]string{}}
		for typ, col := range essenceModColumns {
			if key := col(essence); key != nil {
				e.Mods[typ] = poe.Mods[*key].ID
			}
		}
		if i, ok := at[e.ID]; ok {
			out[i] = e
			continue
		}
		at[e.ID] = len(out)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
