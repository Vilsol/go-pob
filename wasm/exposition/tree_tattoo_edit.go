package exposition

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeTattooEditExports(r bind.Registry) {
	r.Type(TattooChoice{}, bind.Plain())
	r.Type(TattooArt{}, bind.Plain())

	r.Func(TattooEditKind)
	r.Func(GetTattooChoices)
	r.Func(AddTattoo)
	r.Func(TreeTattooArt)
}

type TattooChoice struct {
	DN           string
	Label        string
	Descriptions []string
	Legacy       bool
}

func tattooTreeNode(nodeID int64) (data.Node, bool) {
	n, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes[strconv.FormatInt(nodeID, 10)]
	return n, ok
}

//pob:block:src/Classes/PassiveTree.lua:336-384
func tattooNodeType(n data.Node) string {
	switch {
	case n.ClassStartIndex != nil:
		return "ClassStart"
	case n.IsAscendancyStart != nil && *n.IsAscendancyStart:
		return "AscendClassStart"
	case n.IsMastery != nil && *n.IsMastery:
		return "Mastery"
	case n.IsJewelSocket != nil && *n.IsJewelSocket:
		return "Socket"
	case n.IsKeystone != nil && *n.IsKeystone:
		return "Keystone"
	case n.IsNotable != nil && *n.IsNotable:
		return "Notable"
	}
	return "Normal"
}

//pob:block:src/Classes/PassiveTreeView.lua:1670-1680
func TattooEditKind(build *pob.PathOfBuilding, nodeID int64) string {
	n, ok := tattooTreeNode(nodeID)
	if !ok {
		return ""
	}
	name := ""
	if n.Name != nil {
		name = *n.Name
	}
	typ := tattooNodeType(n)
	editable := build.TattooOn(int(nodeID)) != "" ||
		(typ == "Normal" && (name == "Strength" || name == "Dexterity" || name == "Intelligence")) ||
		(typ == "Notable" && len(n.Stats) > 0 && (strings.Contains(n.Stats[0], "+30 to Dexterity") || strings.Contains(n.Stats[0], "+30 to Strength") || strings.Contains(n.Stats[0], "+30 to Intelligence"))) ||
		typ == "Keystone" || typ == "Mastery"
	switch {
	case !editable:
		return ""
	case typ == "Mastery":
		return "runegraft"
	}
	return "tattoo"
}

//pob:block:src/Classes/TreeTab.lua:852-884
func GetTattooChoices(build *pob.PathOfBuilding, nodeID int64, showLegacy bool) []TattooChoice {
	n, ok := tattooTreeNode(nodeID)
	if !ok || TattooEditKind(build, nodeID) == "" {
		return nil
	}
	nodeName, nodeValue := "", ""
	if n.Name != nil {
		nodeName = *n.Name
	}
	if len(n.Stats) > 0 {
		nodeValue = n.Stats[0]
	}
	numLinkedNodes := len(n.Out) + len(n.In)
	typ := tattooNodeType(n)
	var out []TattooChoice
	for dn, target := range data.TattooTargets {
		if (strings.Contains(nodeName, strings.TrimPrefix(target.TargetType, "Small ")) || (target.TargetValue != "" && strings.Contains(nodeValue, target.TargetValue)) ||
			(target.TargetType == "Small Attribute" && (nodeName == "Intelligence" || nodeName == "Strength" || nodeName == "Dexterity")) ||
			(target.TargetType == "Keystone" && typ == target.TargetType)) &&
			target.MinimumConnected <= numLinkedNodes && (!target.Legacy || showLegacy) {
			sd := data.TattooNodes[dn].Stats
			descriptions := slices.Clone(sd)
			if target.ReminderText != "" {
				descriptions = append(descriptions, target.ReminderText)
			}
			out = append(out, TattooChoice{
				DN:           dn,
				Label:        dn + "                                                " + strings.Join(sd, ","),
				Descriptions: descriptions,
				Legacy:       target.Legacy,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

//pob:block:src/Classes/TreeTab.lua:885-894
func AddTattoo(build *pob.PathOfBuilding, nodeID int64, dn string) error {
	if !slices.ContainsFunc(GetTattooChoices(build, nodeID, true), func(c TattooChoice) bool { return c.DN == dn }) {
		return fmt.Errorf("%s is not offered for node %d", dn, nodeID)
	}
	target := data.TattooTargets[dn]
	build.SetTattoo(pob.Override{NodeID: int(nodeID), DN: dn, Icon: &target.Icon, ActiveEffectImage: &target.ActiveEffectImage})
	return nil
}

type TattooArt struct {
	NodeID            int
	Icon              string
	ActiveEffectImage string
}

func TreeTattooArt(build *pob.PathOfBuilding) []TattooArt {
	var out []TattooArt
	if i := build.ActiveSpecIndex(); i >= 0 {
		for _, o := range build.Tree.Specs[i].Overrides {
			if t, ok := data.TattooTargets[o.DN]; ok {
				out = append(out, TattooArt{NodeID: o.NodeID, Icon: t.Icon, ActiveEffectImage: t.ActiveEffectImage})
			}
		}
	}
	return out
}
