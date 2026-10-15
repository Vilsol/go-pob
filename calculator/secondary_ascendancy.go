package calculator

import (
	"cmp"
	"slices"

	"github.com/Vilsol/go-pob/pob"
)

type SecondaryAscendOption struct {
	Label         string
	AscendClassID int
}

type SecondaryAscendancyDropdown struct {
	Entries  []SecondaryAscendOption
	Selected int
	Enabled  bool
}

//pob:src/Modules/Build.lua:1113-1158
func SecondaryAscendancyList(build *pob.PathOfBuilding) SecondaryAscendancyDropdown {
	out := SecondaryAscendancyDropdown{Entries: []SecondaryAscendOption{{Label: "None"}}}
	i := build.ActiveSpecIndex()
	if i < 0 {
		return out
	}
	if sec := build.Tree.Specs[i].SecondaryAscendClassID; sec != nil {
		out.Selected = *sec
	}
	var sortable []SecondaryAscendOption
	for k, a := range specTree().AlternateAscendancies {
		if a.ID != "" && !legacyAlternateAscendancies[string(a.ID)] {
			sortable = append(sortable, SecondaryAscendOption{Label: string(a.Name), AscendClassID: k + 1})
		}
	}
	slices.SortFunc(sortable, func(a, b SecondaryAscendOption) int { return cmp.Compare(a.Label, b.Label) })
	out.Entries = append(out.Entries, sortable...)
	out.Enabled = len(out.Entries) > 1
	return out
}
