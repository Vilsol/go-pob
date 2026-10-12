package pob

import "slices"

//pob:block:src/Classes/ListControl.lua:428-436,src/Classes/PassiveSpecListControl.lua:82-84
func (b *PathOfBuilding) MoveSpec(from, to int) {
	n := len(b.Tree.Specs)
	if from < 1 || from > n || to < 1 || to > n || from == to {
		return
	}
	active := b.Tree.ActiveSpec
	spec := b.Tree.Specs[from-1]
	b.Tree.Specs = slices.Insert(slices.Delete(b.Tree.Specs, from-1, from), to-1, spec)
	switch {
	case active == from:
		b.Tree.ActiveSpec = to
	case from < active && active <= to:
		b.Tree.ActiveSpec = active - 1
	case to <= active && active < from:
		b.Tree.ActiveSpec = active + 1
	}
}

func FormatMasterySelections(sel []MasterySelection) string {
	return formatMasterySelections(sel)
}
