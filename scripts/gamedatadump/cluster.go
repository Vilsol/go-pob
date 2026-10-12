package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data/raw"
)

type clusterSkill struct {
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	MasteryIcon string   `json:"masteryIcon,omitempty"`
	Tag         string   `json:"tag"`
	Stats       []string `json:"stats"`
	Enchant     []string `json:"enchant"`
}

type clusterJewel struct {
	Size            string                  `json:"size"`
	SizeIndex       int                     `json:"sizeIndex"`
	MinNodes        int                     `json:"minNodes"`
	MaxNodes        int                     `json:"maxNodes"`
	TotalIndicies   int                     `json:"totalIndicies"`
	SmallIndicies   []int                   `json:"smallIndicies"`
	NotableIndicies []int                   `json:"notableIndicies"`
	SocketIndicies  []int                   `json:"socketIndicies"`
	Skills          map[string]clusterSkill `json:"skills"`
}

func passiveSkillStats(node *rawdata.PassiveSkill) map[string][2]float64 {
	values := []int64{node.Stat1, node.Stat2, node.Stat3, node.Stat4, node.Stat5}
	stats := map[string][2]float64{}
	for i, key := range node.Stats {
		if i < len(values) {
			stats[poe.Stats[key].ID] = [2]float64{float64(values[i]), float64(values[i])}
		}
	}
	return stats
}

//pob:block:src/Export/Scripts/cluster.lua:1-91
func dumpClusterJewels() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	var passives []*rawdata.PassiveSkill
	for _, init := range []func() error{
		func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeTags(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveTreeExpansionJewelSizes(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveTreeExpansionJewels(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveTreeExpansionSkills(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveTreeExpansionSpecialSkills(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveJewelSlots(ctx, raw.LatestVersion, cache) },
		func() error { return loader.InitHelper(ctx, raw.LatestVersion, "PassiveSkills", &passives, nil, cache) },
	} {
		if err := init(); err != nil {
			return nil, err
		}
	}
	describer, err := loadStatFile("passive_skill_stat_descriptions")
	if err != nil {
		return nil, err
	}

	jewels := map[string]clusterJewel{}
	for _, jewel := range poe.PassiveTreeExpansionJewels {
		size := poe.PassiveTreeExpansionJewelSizes[jewel.PassiveTreeExpansionJewelSizesKey]
		cj := clusterJewel{
			Size:            size.Name,
			SizeIndex:       size.Key,
			MinNodes:        jewel.MinNodes,
			MaxNodes:        jewel.MaxNodes,
			TotalIndicies:   jewel.TotalIndices,
			SmallIndicies:   jewel.SmallIndices,
			NotableIndicies: jewel.NotableIndices,
			SocketIndicies:  jewel.SocketIndices,
			Skills:          map[string]clusterSkill{},
		}
		for _, skill := range poe.PassiveTreeExpansionSkills {
			if skill.PassiveTreeExpansionJewelSizesKey != size.Key {
				continue
			}
			node := passives[skill.PassiveSkillsKey]
			tag := poe.Tags[skill.TagsKey].ID
			cs := clusterSkill{Name: node.Name, Icon: strings.TrimSuffix(node.Icon, "dds") + "png", Tag: tag, Enchant: []string{}}
			if !strings.HasSuffix(node.Icon, "dds") {
				cs.Icon = node.Icon
			}
			if strings.Contains(tag, "old_do_not_use") {
				cs.Name = node.Name + " (Legacy)"
			}
			if skill.MasteryPassiveSkillsKey != nil {
				icon := passives[*skill.MasteryPassiveSkillsKey].Icon
				if strings.HasSuffix(icon, "dds") {
					icon = strings.TrimSuffix(icon, "dds") + "png"
				}
				cs.MasteryIcon = icon
			}
			desc, _, err := describer.describeStats(passiveSkillStats(node))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", node.ID, err)
			}
			cs.Stats = append([]string{}, desc...)
			for _, line := range desc {
				cs.Enchant = append(cs.Enchant, "Added Small Passive Skills grant: "+line)
			}
			cj.Skills[node.ID] = cs
		}
		jewels[poe.BaseItemTypes[jewel.BaseItemTypesKey].Name] = cj
	}

	notableSortOrder := map[string]int{}
	keystones := []string{}
	for _, skill := range poe.PassiveTreeExpansionSpecialSkills {
		node := passives[skill.PassiveSkillsKey]
		if node.Notable {
			notableSortOrder[node.Name] = skill.StatsKey + 1
		}
	}
	for _, skill := range poe.PassiveTreeExpansionSpecialSkills {
		if node := passives[skill.PassiveSkillsKey]; node.Keystone {
			keystones = append(keystones, node.Name)
		}
	}
	orbitOffsets := map[string]map[string]int{}
	for _, slot := range poe.PassiveJewelSlots {
		if slot.ClusterJewelSize == nil {
			continue
		}
		if slot.ProxySlot == nil {
			return nil, fmt.Errorf("jewel slot %d has a cluster size but no proxy", slot.Key)
		}
		row := map[string]int{}
		for i, start := range slot.StartIndices {
			if i < 3 {
				row[strconv.Itoa(i)] = start
			}
		}
		orbitOffsets[strconv.FormatInt(passives[*slot.ProxySlot].Hash, 10)] = row
	}
	if len(jewels) == 0 {
		return nil, fmt.Errorf("no PassiveTreeExpansionJewels row")
	}
	return map[string]any{
		"pobVersion":       "go-pob-data " + raw.LatestVersion,
		"jewels":           jewels,
		"keystones":        keystones,
		"notableSortOrder": notableSortOrder,
		"orbitOffsets":     orbitOffsets,
	}, nil
}
