package calculator

import (
	"slices"
	"strings"
)

type activeTreeSocket struct {
	id     int
	itemID int
	item   *Item
}

//pob:block:src/Classes/ItemsTab.lua:1659-1677
func (m *MiscCalculator) activeTreeSockets() []activeTreeSocket {
	i := m.build.ActiveSpecIndex()
	if i < 0 {
		return nil
	}
	spec := m.build.Tree.Specs[i]
	alloc := specAllocated(spec)
	var out []activeTreeSocket
	for _, id := range treeSocketIDs() {
		if !alloc[id] {
			continue
		}
		s := activeTreeSocket{id: id}
		for _, sock := range spec.Sockets {
			if sock.NodeID == id && sock.ItemID != 0 {
				s.itemID = sock.ItemID
				for _, it := range m.build.Items.Items {
					if it.ID == sock.ItemID {
						s.item = ParseItem(it.Text)
					}
				}
			}
		}
		out = append(out, s)
	}
	return out
}

func sameItemText(a, b string) bool {
	lines := func(s string) []string {
		var out []string
		for _, l := range strings.Split(s, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				out = append(out, l)
			}
		}
		return out
	}
	return slices.Equal(lines(a), lines(b))
}
