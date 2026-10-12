package calculator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/pob"
)

func NewCompareJewelPowerBuilder(primary, compare *pob.PathOfBuilding, stats []string) *ComparePowerBuilder {
	b := NewComparePowerBuilder(primary, compare, stats, ComparePowerCategories{}, nil)
	b.jewelTasks()
	return b
}

//pob:block:src/Classes/CompareTab.lua:2788-2868
func (b *ComparePowerBuilder) jewelTasks() {
	allocated := ActiveTreeSockets(b.primary)
	sockets := treeSocketIDs()
	for _, j := range CompareJewelSlots(b.primary, b.compare) {
		if j.CText == "" || (j.PText != "" && PeekItem(j.PText).Name == PeekItem(j.CText).Name) {
			continue
		}
		b.tasks = append(b.tasks, func() {
			cItem := PeekItem(j.CText)
			newItem := ParseItem(j.CText)
			newItem.NormaliseQuality()
			label := j.Label
			var best []float64
			if j.PAllocated {
				best = b.impactOf(b.calc(&CalcOverride{SocketNodeID: j.NodeID, SocketItem: newItem}))
			} else {
				for _, id := range sockets {
					o := &CalcOverride{SocketNodeID: id, SocketItem: newItem}
					if !slices.Contains(allocated, id) {
						o.AddNodes = []int64{int64(id)}
					}
					impact := b.impactOf(b.calc(o))
					if best == nil {
						best = impact
					} else {
						for k, v := range impact {
							best[k] = max(best[k], v)
						}
					}
					label = j.Label + " (best socket)"
				}
			}
			if best != nil {
				b.results = append(b.results, ComparePowerEntry{Category: "Item", Name: cItem.Name + ", " + label, Rarity: string(cItem.Rarity), Impact: best})
			}
		})
	}
}

//pob:block:src/Classes/CompareTab.lua:2706-2735,src/Classes/CompareTab.lua:2742-2750
func (b *ComparePowerBuilder) equipCompareAbyssJewels(slotName string, newItem *Item) (int, func()) {
	if newItem.AbyssalSocketCount <= 0 {
		return 0, func() {}
	}
	cSet, cByID := setItems(b.compare)
	pSet, _ := setItems(b.primary)
	oldEquipped := map[string]int{}
	var copies []int
	for idx := 1; idx <= int(newItem.AbyssalSocketCount); idx++ {
		abyssSlotName := fmt.Sprintf("%s Abyssal Socket %d", slotName, idx)
		oldEquipped[abyssSlotName] = setSlotItemID(pSet, abyssSlotName)
		b.primary.SetSlotItem(abyssSlotName, 0)
		if text, ok := cByID[setSlotItemID(cSet, abyssSlotName)]; ok && strings.TrimSpace(text) != "" {
			id := b.primary.AddItem(text)
			copies = append(copies, id)
			b.primary.SetSlotItem(abyssSlotName, id)
		}
	}
	return len(copies), func() {
		for name, id := range oldEquipped {
			b.primary.SetSlotItem(name, id)
		}
		for _, id := range copies {
			b.primary.DeleteItem(id)
		}
	}
}

//pob:block:src/Classes/CompareTab.lua:2757-2758
func abyssJewelText(n int) string {
	switch {
	case n == 0:
		return ""
	case n > 1:
		return fmt.Sprintf(", %d Jewels", n)
	}
	return fmt.Sprintf(", %d Jewel", n)
}
