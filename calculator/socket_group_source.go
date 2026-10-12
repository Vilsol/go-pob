package calculator

import (
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type GroupSourceName struct {
	Name   string
	Rarity string
}

//pob:block:src/Classes/SkillsTab.lua:363-379
func SocketGroupSourceNames(build *pob.PathOfBuilding, setIndex int, groupIndex int) []GroupSourceName {
	if setIndex < 0 || setIndex >= len(build.Skills.SkillSets) || groupIndex < 0 || groupIndex >= len(build.Skills.SkillSets[setIndex].Skills) {
		return nil
	}
	group := build.Skills.SkillSets[setIndex].Skills[groupIndex]
	source := group.Source
	if source == "" && group.SourceAttr != nil {
		source = *group.SourceAttr
	}
	if id, ok := strings.CutPrefix(source, "Tree:"); ok {
		return []GroupSourceName{{Name: treeNodeName(id), Rarity: "NORMAL"}}
	}
	if source != "Explode" {
		return nil
	}
	var out []GroupSourceName
	for _, gem := range group.Gems {
		key := gem.ExplodeSource()
		if key == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(key, "Item:"); ok {
			name := rest
			if _, after, found := strings.Cut(rest, ":"); found {
				name = after
			}
			out = append(out, GroupSourceName{Name: name, Rarity: buildItemRarity(build, name)})
		} else {
			out = append(out, GroupSourceName{Name: treeNodeName(key), Rarity: "NORMAL"})
		}
	}
	return out
}

func treeNodeName(id string) string {
	if node, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes[id]; ok && node.Name != nil {
		return *node.Name
	}
	return "???"
}

func buildItemRarity(build *pob.PathOfBuilding, name string) string {
	for _, it := range build.Items.Items {
		if item := parseItemGlobal(it.Text); item != nil && item.Name == name {
			return string(item.Rarity)
		}
	}
	return "NORMAL"
}
