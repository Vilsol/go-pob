package main

import (
	"fmt"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

type pantheonSoul struct {
	Name string   `json:"name"`
	Mods []string `json:"mods"`
}

type pantheonGod struct {
	IsMajorGod bool           `json:"isMajorGod"`
	Souls      []pantheonSoul `json:"souls"`
}

//pob:block:src/Export/Scripts/pantheons.lua:9-50
func dumpPantheons() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	for _, init := range []func() error{
		func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePantheonPanelLayouts(ctx, raw.LatestVersion, cache) },
	} {
		if err := init(); err != nil {
			return nil, err
		}
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}

	gods := map[string]pantheonGod{}
	for _, p := range poe.PantheonPanelLayouts {
		if p.IsDisabled {
			continue
		}
		god := pantheonGod{IsMajorGod: p.IsMajorGod, Souls: []pantheonSoul{}}
		for _, g := range []struct {
			name     string
			statKeys []int
			values   []int
		}{
			{p.GodName1, p.Effect1StatsKeys, p.Effect1Values},
			{p.GodName2, p.Effect2StatsKeys, p.Effect2Values},
			{p.GodName3, p.Effect3StatsKeys, p.Effect3Values},
			{p.GodName4, p.Effect4StatsKeys, p.Effect4Values},
		} {
			if len(g.statKeys) == 0 {
				break
			}
			soul := pantheonSoul{Name: g.name, Mods: []string{}}
			for i, key := range g.statKeys {
				if key < 0 || key >= len(poe.Stats) || i >= len(g.values) {
					return nil, fmt.Errorf("%s: stat key %d / value %d out of range", p.ID, key, i)
				}
				value := float64(g.values[i])
				lines, _, err := describer.describeStats(map[string][2]float64{poe.Stats[key].ID: {value, value}})
				if err != nil {
					return nil, err
				}
				soul.Mods = append(soul.Mods, strings.Join(lines, " "))
			}
			god.Souls = append(god.Souls, soul)
		}
		gods[p.ID] = god
	}
	if len(gods) == 0 {
		return nil, fmt.Errorf("no enabled PantheonPanelLayout row")
	}
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "gods": gods}, nil
}
