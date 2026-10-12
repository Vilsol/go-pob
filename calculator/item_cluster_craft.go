package calculator

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
)

type ClusterSkillChoice struct {
	ID    string
	Label string
}

type ClusterInfo struct {
	Skills    []ClusterSkillChoice
	Skill     string
	NodeCount int
	Min       int
	Max       int
}

func clusterJewelSkillOf(raw string) string {
	for _, l := range splitRaw(raw) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "Cluster Jewel Skill: "); ok {
			return v
		}
	}
	return ""
}

func clusterNodeCountOf(raw string, cj data.ClusterJewel) (int, bool) {
	for _, l := range splitRaw(raw) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "Cluster Jewel Node Count: "); ok {
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				n = cj.MaxNodes
			}
			return min(max(n, cj.MinNodes), cj.MaxNodes), true
		}
	}
	return 0, false
}

//pob:src/Classes/ItemsTab.lua:2059-2084
func itemClusterInfo(raw string, item *Item) (ClusterInfo, bool) {
	cj, ok := data.ClusterJewels.Jewels[item.BaseName]
	if !ok || !item.Crafted {
		return ClusterInfo{}, false
	}
	unavailable := map[string]bool{"affliction_strength": true, "affliction_dexterity": true, "affliction_intelligence": true}
	out := ClusterInfo{Min: cj.MinNodes, Max: cj.MaxNodes}
	for id, skill := range cj.Skills {
		if !unavailable[id] {
			out.Skills = append(out.Skills, ClusterSkillChoice{ID: id, Label: skill.Name})
		}
	}
	slices.SortFunc(out.Skills, func(a, b ClusterSkillChoice) int {
		if c := strings.Compare(a.Label, b.Label); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	out.Skill = clusterJewelSkillOf(raw)
	if _, ok := cj.Skills[out.Skill]; !ok && len(out.Skills) > 0 {
		out.Skill = out.Skills[0].ID
	}
	out.NodeCount = cj.MinNodes
	if n, ok := clusterNodeCountOf(raw, cj); ok {
		out.NodeCount = n
	}
	return out, true
}

func clusterNodeCount(val float64, cj data.ClusterJewel) int {
	return int(math.Floor(val*float64(cj.MaxNodes-cj.MinNodes) + float64(cj.MinNodes) + 0.5))
}

//pob:block:src/Classes/ItemsTab.lua:700-715,src/Classes/ItemsTab.lua:2086-2105
func craftClusterText(raw string, skillID string, nodeCount int) (string, bool) {
	item := ParseItem(raw)
	cj, ok := data.ClusterJewels.Jewels[item.BaseName]
	info, ok2 := itemClusterInfo(raw, item)
	if !ok || !ok2 {
		return raw, false
	}
	if _, ok := cj.Skills[skillID]; !ok {
		skillID = info.Skill
	}
	nodeCount = min(max(nodeCount, cj.MinNodes), cj.MaxNodes)
	b := craftBlock(raw, item)
	var before []string
	for _, l := range b.before {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "Cluster Jewel Skill: ") || strings.HasPrefix(t, "Cluster Jewel Node Count: ") {
			continue
		}
		before = append(before, l)
	}
	at := len(before)
	for k, l := range before {
		t := strings.TrimSpace(l)
		if t == "Crafted: true" || affixSpecRe.MatchString(t) || strings.HasPrefix(t, "Catalyst: ") || strings.HasPrefix(t, "CatalystQuality: ") {
			at = k + 1
		}
	}
	b.before = slices.Insert(before, at, "Cluster Jewel Skill: "+skillID, "Cluster Jewel Node Count: "+strconv.Itoa(nodeCount))
	b.enchant = []string{"{crafted}Adds " + strconv.Itoa(nodeCount) + " Passive Skills"}
	switch cj.Size {
	case "Large":
		b.enchant = append(b.enchant, "{crafted}2 Added Passive Skills are Jewel Sockets")
	case "Medium":
		b.enchant = append(b.enchant, "{crafted}1 Added Passive Skill is a Jewel Socket")
	}
	b.enchant = append(b.enchant, "{crafted}"+strings.Join(cj.Skills[skillID].Enchant, "\n"))
	text := b.join()

	item = ParseItem(text)
	st, ok := itemAffixState(text, item)
	controls, ok2 := ItemAffixControls(text)
	if !ok || !ok2 {
		return text, true
	}
	for i := range st.limit {
		a, _, _ := st.slot(i)
		if a.modID == "None" {
			continue
		}
		if c := controls.Controls[i]; c.Sel == 0 {
			*a = affixSlot{modID: "None", fractured: a.fractured}
		}
	}
	return craftText(text, item, st), true
}
