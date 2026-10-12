package main

import (
	"fmt"
	"sort"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

type craftMasterMod struct {
	exportMod
	Types []string `json:"types"`
}

//pob:block:src/Export/Scripts/masters.lua:6-39
var masterItemClassMap = map[string]string{
	"LifeFlask":                "Flask",
	"ManaFlask":                "Flask",
	"HybridFlask":              "Flask",
	"Amulet":                   "Amulet",
	"Ring":                     "Ring",
	"Claw":                     "Claw",
	"Dagger":                   "Dagger",
	"Rune Dagger":              "Dagger",
	"Wand":                     "Wand",
	"One Hand Sword":           "One Handed Sword",
	"Thrusting One Hand Sword": "Thrusting One Handed Sword",
	"One Hand Axe":             "One Handed Axe",
	"One Hand Mace":            "One Handed Mace",
	"Bow":                      "Bow",
	"Fishing Rod":              "Fishing Rod",
	"Staff":                    "Staff",
	"Warstaff":                 "Staff",
	"Two Hand Sword":           "Two Handed Sword",
	"Two Hand Axe":             "Two Handed Axe",
	"Two Hand Mace":            "Two Handed Mace",
	"Quiver":                   "Quiver",
	"Belt":                     "Belt",
	"Gloves":                   "Gloves",
	"Boots":                    "Boots",
	"Body Armour":              "Body Armour",
	"Helmet":                   "Helmet",
	"Shield":                   "Shield",
	"Sceptre":                  "Sceptre",
	"UtilityFlask":             "Flask",
	"UtilityFlaskCritical":     "Flask",
	"Map":                      "Map",
	"Jewel":                    "Jewel",
}

//pob:block:src/Export/Scripts/masters.lua:40-72
func dumpMasterMods() ([]craftMasterMod, error) {
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"ItemClasses", func() error { return poe.InitializeItemClasses(ctx, raw.LatestVersion, cache) }},
		{"CraftingItemClassCategories", func() error { return poe.InitializeCraftingItemClassCategories(ctx, raw.LatestVersion, cache) }},
		{"CraftingBenchOptions", func() error { return poe.InitializeCraftingBenchOptions(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}
	x := &modsExport{describer: describer}
	out := []craftMasterMod{}
	for _, craft := range poe.CraftingBenchOptions {
		if craft.IsDisabled || craft.AddMod == nil {
			continue
		}
		mod := poe.Mods[*craft.AddMod]
		row := craftMasterMod{exportMod: exportMod{Affix: mod.Name, Level: mod.Level, Group: poe.ModTypes[mod.ModTypeKey].Name, ModTags: modTags(mod.ImplicitTagsKeys),
			WeightKey: []string{}, WeightVal: []int{}, WeightMultiplierKey: []string{}, WeightMultiplierVal: []int{}}}
		switch mod.GenerationType {
		case genPrefix:
			row.Type = "Prefix"
		case genSuffix:
			row.Type = "Suffix"
		}
		row.Lines, row.StatOrder, err = x.describeMod(mod, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		if len(row.Lines) == 0 {
			row.Lines = []string{""}
		}
		if row.StatOrder == nil {
			row.StatOrder = []float64{}
		}
		uniqueTypes := map[string]bool{}
		for _, category := range craft.CraftingItemClassCategories {
			for _, itemClass := range poe.CraftingItemClassCategories[category].ItemClasses {
				if t, ok := masterItemClassMap[poe.ItemClasses[itemClass].ID]; ok {
					uniqueTypes[t] = true
				}
			}
		}
		row.Types = make([]string, 0, len(uniqueTypes))
		for t := range uniqueTypes {
			row.Types = append(row.Types, t)
		}
		sort.Strings(row.Types)
		out = append(out, row)
	}
	return out, nil
}
