package main

import (
	"fmt"
	"slices"
	"sort"
)

type poe2AlternatePassiveSkill struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	PassiveType []int  `json:"PassiveType"`
	Stats       []int  `json:"Stats"`
	Stat1Min    int    `json:"Stat1Min"`
	Stat1Max    int    `json:"Stat1Max"`
	Stat2Min    int    `json:"Stat2Min"`
	Stat2Max    int    `json:"Stat2Max"`
	Stat3Min    int    `json:"Stat3Min"`
	Stat3Max    int    `json:"Stat3Max"`
	Stat4Min    int    `json:"Stat4Min"`
	Stat4Max    int    `json:"Stat4Max"`
	Stat5Min    int    `json:"Stat5Min"`
	Stat5Max    int    `json:"Stat5Max"`
	Stat6Min    int    `json:"Stat6Min"`
	Stat6Max    int    `json:"Stat6Max"`
}

// poe2LegionNode is the part of a tree.legion node the PoE2 emitter reads.
type poe2LegionNode struct {
	ID  string   `json:"id"`
	Dn  string   `json:"dn"`
	Sd  []string `json:"sd"`
	Ks  bool     `json:"ks"`
	Not bool     `json:"not"`
	M   bool     `json:"m"`
}

//pob2:block:src/Export/Scripts/legionPassives.lua:77-108
func poe2LegionStats(describer *poe2Describer, stats []poe2Stat, id string, statKeys []int, ranges [][2]int) ([]string, error) {
	var sd []string
	descOrders := map[string]float64{}
	for idx, statKey := range statKeys {
		if statKey < 0 || statKey >= len(stats) || idx >= len(ranges) {
			return nil, fmt.Errorf("%s: stat %d (key %d) has no range", id, idx+1, statKey)
		}
		r := ranges[idx]
		statLines, orders, err := describer.describeStats(map[string]*edRange{stats[statKey].ID: {min: float64(r[0]), max: float64(r[1])}})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
		for i, line := range statLines {
			sd = append(sd, line)
			descOrders[line] = orders[i]
		}
	}
	// Have to re-sort since we described the stats earlier
	sort.SliceStable(sd, func(i, j int) bool { return descOrders[sd[i]] < descOrders[sd[j]] })
	return sd, nil
}

//pob2:block:src/Export/Scripts/legionPassives.lua:110-180
func dumpPoE2LegionPassives() (any, error) {
	stats, err := poe2Table[poe2Stat]("Stats")
	if err != nil {
		return nil, err
	}
	skills, err := poe2Table[poe2AlternatePassiveSkill]("AlternatePassiveSkills")
	if err != nil {
		return nil, err
	}
	describer, err := poe2LoadStatFile("passive_skill_stat_descriptions")
	if err != nil {
		return nil, err
	}
	nodes := make([]poe2LegionNode, 0, len(skills))
	for _, s := range skills {
		id := fixLegionDatErrors(s.ID, s.Name)
		node := poe2LegionNode{
			ID:  id,
			Ks:  slices.Contains(s.PassiveType, 4),
			Not: slices.Contains(s.PassiveType, 3),
			Dn:  s.Name,
		}
		node.Sd, err = poe2LegionStats(describer, stats, id, s.Stats, [][2]int{
			{s.Stat1Min, s.Stat1Max}, {s.Stat2Min, s.Stat2Max}, {s.Stat3Min, s.Stat3Max},
			{s.Stat4Min, s.Stat4Max}, {s.Stat5Min, s.Stat5Max}, {s.Stat6Min, s.Stat6Max},
		})
		if err != nil {
			return nil, err
		}
		if node.ID == "vaal_keystone_2_v2" { // Immortal Ambition needs to be manually added
			node.Sd = []string{
				"Energy Shield starts at zero",
				"Cannot Recharge or Regenerate Energy Shield",
				"Lose 5% of Energy Shield per second",
				"Life Leech effects are not removed when Unreserved Life is Filled",
				"Life Leech effects Recover Energy Shield instead while on Full Life",
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
