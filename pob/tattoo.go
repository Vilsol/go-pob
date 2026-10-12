package pob

import (
	"slices"
	"strings"
)

func (b *PathOfBuilding) TattooOn(nodeID int) string {
	if spec := b.activeSpec(); spec != nil {
		for _, o := range spec.Overrides {
			if o.NodeID == nodeID {
				return o.DN
			}
		}
	}
	return ""
}

//pob:src/Classes/TreeTab.lua:346-351
func (b *PathOfBuilding) RemoveTattoo(nodeID int) bool {
	spec := b.activeSpec()
	if spec == nil {
		return false
	}
	n := len(spec.Overrides)
	spec.Overrides = slices.DeleteFunc(spec.Overrides, func(o Override) bool { return o.NodeID == nodeID })
	return len(spec.Overrides) != n
}

//pob:block:src/Classes/TreeTab.lua:953-969
func (b *PathOfBuilding) TattooCount() int {
	count := 0
	if spec := b.activeSpec(); spec != nil {
		for _, o := range spec.Overrides {
			if !strings.Contains(o.DN, "Runegraft") {
				count++
			}
		}
	}
	return count
}

//pob:src/Classes/TreeTab.lua:879-885
func (b *PathOfBuilding) SetTattoo(o Override) {
	spec := b.activeSpec()
	if spec == nil {
		return
	}
	if i := slices.IndexFunc(spec.Overrides, func(x Override) bool { return x.NodeID == o.NodeID }); i >= 0 {
		spec.Overrides[i] = o
		return
	}
	spec.Overrides = append(spec.Overrides, o)
}
