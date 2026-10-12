package calculator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

type TreeJewelRing struct {
	NodeID int64
	Outer  float64
	Inner  float64
	Art    string
}

type TreeSocketRadius struct {
	Inner float64
	Outer float64
	Col   string
	Nodes []int64
}

var timelessRingArt = []string{"Brutal Restraint", "Elegant Hubris", "Glorious Vanity", "Lethal Pride", "Militant Faith", "Heroic Tragedy"}

func socketedJewel(build *pob.PathOfBuilding, nodeID int64) (*Item, bool) {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return nil, false
	}
	for _, s := range build.Tree.Specs[i].Sockets {
		if int64(s.NodeID) != nodeID || s.ItemID == 0 {
			continue
		}
		for _, it := range build.Items.Items {
			if it.ID == s.ItemID {
				return PeekItem(it.Text), true
			}
		}
	}
	return nil, false
}

func isAbyssConqueredItem(item *Item) bool {
	c, ok := itemConqueredBy(item)
	return ok && strings.HasPrefix(c.Conqueror.Type, "abyss_")
}

func treeSockets(tree *data.Tree) []int64 {
	var out []int64
	for key, n := range tree.Nodes {
		if n.IsJewelSocket == nil || !*n.IsJewelSocket || (n.Name != nil && *n.Name == "Charm Socket") ||
			(n.ExpansionJewel != nil && n.ExpansionJewel.Size != 2) {
			continue
		}
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

//pob:block:src/Classes/PassiveTreeView.lua:1157-1204,src/Classes/PassiveTreeView.lua:1235-1246
func TreeJewelRings(build *pob.PathOfBuilding, compared []int64) []TreeJewelRing {
	tree := specTree()
	alloc := allocSet(build, tree)
	for _, id := range compared {
		alloc[id] = true
	}
	var out []TreeJewelRing
	for _, socket := range treeSockets(tree) {
		if !alloc[socket] {
			continue
		}
		item, ok := socketedJewel(build, socket)
		if !ok {
			continue
		}
		index, ok := itemJewelRadiusIndex(item)
		if !ok || index < 0 || index >= len(data.JewelRadius) || isAbyssConqueredItem(item) {
			continue
		}
		r := data.JewelRadius[index]
		title := item.Title
		if strings.Contains(title, "Impossible Escape") {
			for _, m := range item.Mods {
				v, ok := m.Value().List().(mod.JewelData)
				if m.Name() != "JewelData" || !ok || v.Key != "impossibleEscapeKeystone" {
					continue
				}
				name, _ := v.Value.(string)
				for key, n := range tree.Nodes {
					if n.IsKeystone != nil && *n.IsKeystone && n.Name != nil && strings.EqualFold(*n.Name, name) {
						if id, err := strconv.ParseInt(key, 10, 64); err == nil {
							out = append(out, TreeJewelRing{NodeID: id, Outer: r.Outer, Inner: 150})
						}
					}
				}
			}
			continue
		}
		ring := TreeJewelRing{NodeID: socket, Outer: r.Outer, Inner: r.Inner * 1.06}
		for _, art := range timelessRingArt {
			if strings.HasPrefix(title, art) {
				ring.Art = art
			}
		}
		out = append(out, ring)
	}
	return out
}

//pob:block:src/Classes/PassiveTreeView.lua:1213-1234,src/Classes/PassiveTreeView.lua:977-1003
func TreeSocketRadii(build *pob.PathOfBuilding, nodeID int64) []TreeSocketRadius {
	tree := specTree()
	if !slices.Contains(treeSockets(tree), nodeID) {
		return nil
	}
	variable := false
	if item, ok := socketedJewel(build, nodeID); ok {
		variable = item.JewelRadiusLabel == "Variable"
	}
	perRadius := tree.JewelRadiusNodes()[nodeID]
	var out []TreeSocketRadius
	for i, r := range data.JewelRadius {
		if (r.Inner != 0) != variable {
			continue
		}
		var nodes []int64
		if i < len(perRadius) {
			nodes = slices.Clone(perRadius[i])
			slices.Sort(nodes)
		}
		out = append(out, TreeSocketRadius{Inner: r.Inner, Outer: r.Outer, Col: r.Col, Nodes: nodes})
	}
	return out
}

type TreeJewelOverlay struct {
	NodeID int64
	Art    string
}

//pob:src/Classes/PassiveTreeView.lua:126-155
func jewelSocketOverlay(jewel *Item, isExpansion bool) string {
	alt := func(name string) string {
		if isExpansion {
			return name + "Alt"
		}
		return name
	}
	subType := ""
	if jewel.Base != nil {
		subType = jewel.Base.SubType
	}
	switch {
	case jewel.BaseName == "Crimson Jewel":
		return alt("JewelSocketActiveRed")
	case jewel.BaseName == "Viridian Jewel":
		return alt("JewelSocketActiveGreen")
	case jewel.BaseName == "Cobalt Jewel":
		return alt("JewelSocketActiveBlue")
	case jewel.BaseName == "Prismatic Jewel":
		return alt("JewelSocketActivePrismatic")
	case subType == "Abyss":
		return alt("JewelSocketActiveAbyss")
	case subType == "Charm":
		switch jewel.BaseName {
		case "Ursine Charm":
			return "CharmSocketActiveStr"
		case "Corvine Charm":
			return "CharmSocketActiveInt"
		case "Lupine Charm":
			return "CharmSocketActiveDex"
		}
		return ""
	case jewel.BaseName == "Timeless Jewel":
		return alt("JewelSocketActiveLegion")
	case jewel.BaseName == "Large Cluster Jewel":
		return "JewelSocketActiveAltPurple"
	case jewel.BaseName == "Medium Cluster Jewel":
		return "JewelSocketActiveAltBlue"
	case jewel.BaseName == "Small Cluster Jewel":
		return "JewelSocketActiveAltRed"
	}
	return ""
}

//pob:block:src/Classes/PassiveTreeView.lua:840-846
func TreeJewelOverlays(build *pob.PathOfBuilding) []TreeJewelOverlay {
	tree := specTree()
	alloc := allocSet(build, tree)
	i := build.ActiveSpecIndex()
	if i < 0 {
		return nil
	}
	var out []TreeJewelOverlay
	for _, s := range build.Tree.Specs[i].Sockets {
		id := int64(s.NodeID)
		node, ok := tree.Nodes[nodeKey(id)]
		if !ok || !alloc[id] {
			continue
		}
		item, ok := socketedJewel(build, id)
		if !ok {
			continue
		}
		if art := jewelSocketOverlay(item, node.ExpansionJewel != nil); art != "" {
			out = append(out, TreeJewelOverlay{NodeID: id, Art: art})
		}
	}
	slices.SortFunc(out, func(a, b TreeJewelOverlay) int { return int(a.NodeID - b.NodeID) })
	return out
}
