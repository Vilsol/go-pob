package main

import (
	"fmt"
	"sort"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

//pob:src/Export/Scripts/skills.lua:15-31
var weaponClassMap = map[string]string{
	"Claw":                     "Claw",
	"Dagger":                   "Dagger",
	"Wand":                     "Wand",
	"One Hand Sword":           "One Handed Sword",
	"Thrusting One Hand Sword": "Thrusting One Handed Sword",
	"One Hand Axe":             "One Handed Axe",
	"One Hand Mace":            "One Handed Mace",
	"Bow":                      "Bow",
	"FishingRod":               "Fishing Rod",
	"Staff":                    "Staff",
	"Two Hand Sword":           "Two Handed Sword",
	"Two Hand Axe":             "Two Handed Axe",
	"Two Hand Mace":            "Two Handed Mace",
	"Sceptre":                  "Sceptre",
	"Unarmed":                  "None",
}

//pob:block:src/Export/Scripts/skills.lua:199-211
//pob:block:src/Export/Scripts/skills.lua:234-245
func dumpSkillWeaponTypes() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	for _, init := range []func() error{
		func() error { return poe.InitializeItemClasses(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeGrantedEffects(ctx, raw.LatestVersion, cache) },
	} {
		if err := init(); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
	}
	skills := map[string]any{}
	for _, granted := range poe.GrantedEffects {
		classes := granted.WeaponRestrictions
		if !granted.IsSupport {
			active := granted.GetActiveSkill()
			if active == nil {
				continue
			}
			classes = active.WeaponRestrictionItemClassesKeys
		}
		set := map[string]bool{}
		for _, class := range classes {
			if t, ok := weaponClassMap[poe.ItemClasses[class].ID]; ok {
				set[t] = true
			}
		}
		if len(set) == 0 {
			continue
		}
		types := make([]string, 0, len(set))
		for t := range set {
			types = append(types, t)
		}
		sort.Strings(types)
		skills[granted.ID] = map[string]any{"weaponTypes": types, "support": granted.IsSupport}
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("no granted effect carries a weapon restriction")
	}
	return map[string]any{"skills": skills}, nil
}
