package calculator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type MasteryOption struct {
	Effect   int64
	Label    string
	Selected bool
}

func activeMasterySelections(build *pob.PathOfBuilding) []pob.MasterySelection {
	if i := build.ActiveSpecIndex(); i >= 0 {
		return pob.SpecMasterySelections(build.Tree.Specs[i])
	}
	return nil
}

//pob:block:src/Classes/TreeTab.lua:1017-1023
func MasteryOptions(build *pob.PathOfBuilding, nodeID int64) []MasteryOption {
	node, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes[strconv.FormatInt(nodeID, 10)]
	if !ok || !isMasteryNode(node) {
		return nil
	}
	sel := activeMasterySelections(build)
	var out []MasteryOption
	for _, effect := range node.MasteryEffects {
		assigned := slices.IndexFunc(sel, func(s pob.MasterySelection) bool { return s.Effect == effect.Effect })
		if assigned >= 0 && sel[assigned].Node != nodeID {
			continue
		}
		out = append(out, MasteryOption{Effect: effect.Effect, Label: strings.Join(effect.Stats, " / "), Selected: assigned >= 0})
	}
	return out
}

//pob:block:src/Classes/TreeTab.lua:992-1009
func SelectMasteryEffect(build *pob.PathOfBuilding, nodeID int64, effectID int64, path []int64) error {
	if !slices.ContainsFunc(MasteryOptions(build, nodeID), func(o MasteryOption) bool { return o.Effect == effectID }) {
		return fmt.Errorf("effect %d is not offered for mastery %d", effectID, nodeID)
	}
	build.SetMasteryEffect(nodeID, effectID)
	if !slices.Contains(build.Build.PassiveNodes, nodeID) {
		add := make([]int64, 0, len(path)+1)
		for _, id := range path {
			if !slices.Contains(build.Build.PassiveNodes, id) && !slices.Contains(add, id) {
				add = append(add, id)
			}
		}
		if !slices.Contains(add, nodeID) {
			add = append(add, nodeID)
		}
		build.AllocateNodes(add)
	}
	return nil
}
