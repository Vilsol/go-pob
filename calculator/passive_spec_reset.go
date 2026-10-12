package calculator

import (
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//pob:src/Classes/PassiveSpec.lua:756-765
func ResetTree(build *pob.PathOfBuilding) {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return
	}
	tree := specTree()
	var drop []int64
	for _, id := range build.Build.PassiveNodes {
		if n, ok := tree.Nodes[nodeKey(id)]; ok && (isClassStartNode(n) || isAscendClassStartNode(n)) {
			continue
		}
		drop = append(drop, id)
	}
	if len(drop) > 0 {
		build.DeallocateNodes(drop)
	}
	build.Tree.Specs[i].MasteryEffects = ""
}

//pob:block:src/Classes/TreeTab.lua:140-149
func RemoveAllTattoos(build *pob.PathOfBuilding) {
	if i := build.ActiveSpecIndex(); i >= 0 {
		build.Tree.Specs[i].Overrides = []pob.Override{}
	}
}

type TattooCountLine struct {
	Line  string
	Count int
}

//pob:block:src/Classes/TreeTab.lua:946-973
func TattooCountLines(build *pob.PathOfBuilding) []TattooCountLine {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return nil
	}
	counts := map[string]int{}
	for _, o := range build.Tree.Specs[i].Overrides {
		if strings.Contains(o.DN, "Runegraft") {
			continue
		}
		combined := ""
		for _, line := range data.TattooNodes[o.DN].Stats {
			if !strings.Contains(line, "Limited") && !strings.Contains(line, "Requires") {
				combined += " " + line
			}
		}
		counts[combined]++
	}
	out := make([]TattooCountLine, 0, len(counts))
	for line, n := range counts {
		out = append(out, TattooCountLine{Line: line, Count: n})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Line < out[b].Line })
	return out
}
