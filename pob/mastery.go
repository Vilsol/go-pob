package pob

import (
	"regexp"
	"strconv"
	"strings"
)

type MasterySelection struct {
	Node   int64
	Effect int64
}

var masterySelectionRe = regexp.MustCompile(`\{(\d+),(\d+)\}`)

//pob:block:src/Classes/PassiveSpec.lua:153-157
func SpecMasterySelections(spec Spec) []MasterySelection {
	var out []MasterySelection
	for _, m := range masterySelectionRe.FindAllStringSubmatch(spec.MasteryEffects, -1) {
		node, errNode := strconv.ParseInt(m[1], 10, 64)
		effect, errEffect := strconv.ParseInt(m[2], 10, 64)
		if errNode == nil && errEffect == nil {
			out = append(out, MasterySelection{Node: node, Effect: effect})
		}
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:202-205,src/Classes/PassiveSpec.lua:215
func formatMasterySelections(sel []MasterySelection) string {
	parts := make([]string, 0, len(sel))
	for _, s := range sel {
		parts = append(parts, "{"+strconv.FormatInt(s.Node, 10)+","+strconv.FormatInt(s.Effect, 10)+"}")
	}
	return strings.Join(parts, ",")
}

//pob:block:src/Classes/TreeTab.lua:1010
func (b *PathOfBuilding) SetMasteryEffect(nodeID int64, effectID int64) {
	spec := b.activeSpec()
	if spec == nil {
		return
	}
	sel := SpecMasterySelections(*spec)
	found := false
	for i := range sel {
		if sel[i].Node == nodeID {
			sel[i].Effect = effectID
			found = true
		}
	}
	if !found {
		sel = append(sel, MasterySelection{Node: nodeID, Effect: effectID})
	}
	spec.MasteryEffects = formatMasterySelections(sel)
}

//pob:block:src/Classes/PassiveSpec.lua:823-826
func (b *PathOfBuilding) dropMasterySelections(nodeIDs []int64) {
	spec := b.activeSpec()
	if spec == nil || spec.MasteryEffects == "" {
		return
	}
	gone := make(map[int64]bool, len(nodeIDs))
	for _, id := range nodeIDs {
		gone[id] = true
	}
	sel := SpecMasterySelections(*spec)
	kept := sel[:0]
	for _, s := range sel {
		if !gone[s.Node] {
			kept = append(kept, s)
		}
	}
	if len(kept) != len(sel) {
		spec.MasteryEffects = formatMasterySelections(kept)
	}
}
