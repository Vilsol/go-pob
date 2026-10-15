package calculator

import (
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// for baseName, jewel in pairs(data.clusterJewels.jewels) do
//
//	for skillId, skill in pairs(jewel.skills) do
//	end
//
// end
// for notable in pairs(data.clusterJewels.notableSortOrder) do
//
//	clusterJewelSkills["1 added passive skill is "..notable:lower()] = { mod("ClusterJewelNotable", "LIST", notable) }
//
// end
// for _, keystone in ipairs(data.clusterJewels.keystones) do
//
//	clusterJewelSkills["adds "..keystone:lower()] = { mod("JewelData", "LIST", { key = "clusterJewelKeystone", value = keystone }) }
//
// end
var clusterJewelSkills = sync.OnceValue(buildClusterJewelSkills)

//pob:block:src/Modules/ModParser.lua:6595-6607
func buildClusterJewelSkills() map[string][]mod.Mod {
	skills := make(map[string][]mod.Mod)
	bases := make([]string, 0, len(data.ClusterJewels.Jewels))
	for base := range data.ClusterJewels.Jewels {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	for _, base := range bases {
		jewel := data.ClusterJewels.Jewels[base]
		skillIDs := make([]string, 0, len(jewel.Skills))
		for skillID := range jewel.Skills {
			skillIDs = append(skillIDs, skillID)
		}
		sort.Strings(skillIDs)

		for _, skillID := range skillIDs {
			key := strings.ToLower(strings.Join(jewel.Skills[skillID].Enchant, " "))
			if _, taken := skills[key]; taken {
				continue
			}
			skills[key] = []mod.Mod{mod.NewList("JewelData", mod.JewelData{Key: "clusterJewelSkill", Value: skillID})}
		}
	}
	for notable := range data.ClusterJewels.NotableSortOrder {
		skills["1 added passive skill is "+strings.ToLower(notable)] = []mod.Mod{mod.NewList("ClusterJewelNotable", notable)}
	}
	for _, keystone := range data.ClusterJewels.Keystones {
		skills["adds "+strings.ToLower(keystone)] = []mod.Mod{mod.NewList("JewelData", mod.JewelData{Key: "clusterJewelKeystone", Value: keystone})}
	}
	return skills
}

//pob:block:src/Modules/ModParser.lua:6653-6656
func parseClusterJewelSkill(lineLower string) []mod.Mod {
	if mods, ok := clusterJewelSkills()[lineLower]; ok {
		return mod.CloneSlice(mods)
	}
	return nil
}
