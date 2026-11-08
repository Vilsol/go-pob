package calculator2

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/tree2"
)

// MiscCalculator is calcs.getMiscCalculator over the last main calculation: its base pass and a calculation per
// override, each with the Full DPS of the groups included in it when the base pass found any.
type MiscCalculator struct {
	main *Environment
	// Base is the base pass, in the "CALCULATOR" mode every override runs in.
	Base *Environment
	// usedFullDPS is set when the base pass's Full DPS has a skill.
	usedFullDPS bool
}

// ItemOverride is the item half of initEnv's override.
type ItemOverride struct {
	RepSlotName string
	RepItem     *Item
	ToggleFlask *Item
	ToggleCharm *Item
}

// The base pass's Full DPS result cache (fullDPSCache) is not ported.
//
//pob2:block:src/Modules/Calcs.lua:92-105
func NewMiscCalculator(main *Environment) *MiscCalculator {
	m := &MiscCalculator{main: main}
	m.Base = m.calc(calcOverride{}, false)
	fullDPS := calcFullDPS(main, "CALCULATOR", calcOverride{})
	m.usedFullDPS = len(fullDPS.Skills) > 0
	if m.usedFullDPS {
		setFullDPS(m.Base, fullDPS)
	}
	return m
}

// calc is the calculator function's full pass for one override; useFullDPS false is the caller passing false
// outside the tree view.
//
//pob2:block:src/Modules/Calcs.lua:132-145
func (m *MiscCalculator) calc(override calcOverride, useFullDPS bool) *Environment {
	env := newWeaponSetEnv(m.main, "CALCULATOR", override)
	initEnv(env)
	perform(env)
	if useFullDPS && m.usedFullDPS {
		setFullDPS(env, calcFullDPS(m.main, "CALCULATOR", override))
	}
	return env
}

// Calc is the calculator function for one override.
func (m *MiscCalculator) Calc(o ItemOverride) *Environment {
	return m.calc(calcOverride{RepSlotName: o.RepSlotName, RepItem: o.RepItem, ToggleFlask: o.ToggleFlask, ToggleCharm: o.ToggleCharm}, true)
}

// CalcFullDPSOnly is the calculator function with fastCalcOptions.fullDPSOnly: the Full DPS roll-up alone when the
// base pass found any, else the full pass without it. The accelerated pass is a full one here.
//
//pob2:block:src/Modules/Calcs.lua:108-130
func (m *MiscCalculator) CalcFullDPSOnly(useFullDPS bool) *Environment {
	if useFullDPS && m.usedFullDPS {
		fullDPS := calcFullDPS(m.main, "CALCULATOR", calcOverride{})
		env := &Environment{Player: &Actor{Output: map[string]float64{}}}
		setFullDPS(env, fullDPS)
		return env
	}
	return m.calc(calcOverride{}, false)
}

// NodeOverride is the passive half of initEnv's override: removeNodes keyed by node, and keyed by id for a granted
// passive.
type NodeOverride struct {
	AddNodes      map[*tree2.SpecNode]bool
	RemoveNodes   map[*tree2.SpecNode]bool
	RemoveNodeIDs map[int]bool
}

// CalcNodes is the calculator function for a passive override. The tree view always takes Full DPS.
func (m *MiscCalculator) CalcNodes(o NodeOverride) *Environment {
	return m.calc(calcOverride{AddNodes: o.AddNodes, RemoveNodes: o.RemoveNodes, RemoveNodeIDs: o.RemoveNodeIDs}, true)
}

// ItemCompare is one block of an item tooltip's stat differences: the calculation with the item in Slot (or toggled
// for a flask or charm) under Header.
type ItemCompare struct {
	Header string
	// Slot is the slot the item is compared in; "" for a flask or charm toggle.
	Slot string
	Env  *Environment
}

type compareSlotEntry struct {
	selItem      *Item
	env          *Environment
	slot         *ItemSlot
	label        string
	isSameUnique bool
}

func (tab *ItemsTab) slotLabel(slot *ItemSlot) string {
	switch {
	case slot.NodeID != 0:
		return "Socket"
	case slot.ParentSlot != nil:
		return "Jewel #" + strconv.Itoa(slot.SlotNum)
	}
	return strings.TrimSuffix(slot.SlotName, " Swap")
}

// ItemCompares is the stat difference blocks AddItemTooltip shows for the item: activating or deactivating a flask
// or charm, else equipping it in, or removing it from, each slot it fits in the weapon set in use. Headers are without
// their colour codes.
//
//pob2:block:src/Classes/ItemsTab.lua:4382-4389
//pob2:block:src/Classes/ItemsTab.lua:4417-4424
//pob2:block:src/Classes/ItemsTab.lua:4426-4538
//nolint:gocyclo // mirrors the Lua
func (tab *ItemsTab) ItemCompares(item *Item, m *MiscCalculator, additionalRingSlot bool) []ItemCompare {
	main := m.main
	switch {
	case item.Base.Flask != nil:
		header := "Activating this flask will give you:"
		if main.Flasks[item] {
			header = "Deactivating this flask will give you:"
		}
		return []ItemCompare{{Header: header, Env: m.Calc(ItemOverride{ToggleFlask: item})}}
	case item.Base.Charm != nil:
		header := "Activating this charm will give you:"
		if main.Charms[item] {
			header = "Deactivating this charm will give you:"
		}
		return []ItemCompare{{Header: header, Env: m.Calc(ItemOverride{ToggleCharm: item})}}
	}
	weaponSet := main.WeaponSet
	var compareSlots []*ItemSlot
	for _, slot := range tab.OrderedSlots {
		if tab.IsItemValidForSlot(item, slot.SlotName, nil) && !tab.slotInactive(slot) && (slot.WeaponSet == 0 || slot.WeaponSet == weaponSet) &&
			(slot.WeaponSet != 0 || tab.SlotShown(slot, additionalRingSlot)) {
			compareSlots = append(compareSlots, slot)
		}
	}
	isUnique := item.Rarity == "UNIQUE" || item.Rarity == "RELIC"
	var slots []compareSlotEntry
	currentSameUniqueCount := 0.0
	for _, compareSlot := range compareSlots {
		selItem := tab.Items[compareSlot.SelItemID]
		o := ItemOverride{RepSlotName: compareSlot.SlotName}
		if item != selItem {
			o.RepItem = item
		}
		isSameUnique := isUnique && selItem != nil && item.Name == selItem.Name
		if isSameUnique && item.Limit != nil {
			currentSameUniqueCount++
		}
		slots = append(slots, compareSlotEntry{selItem: selItem, env: m.Calc(o), slot: compareSlot, label: tab.slotLabel(compareSlot), isSameUnique: isSameUnique})
	}
	block := func(e compareSlotEntry) ItemCompare {
		var header string
		if item == e.selItem {
			header = "Removing this item from " + e.label + " will give you:"
		} else {
			header = "Equipping this item in " + e.label + " will give you:"
			if e.selItem != nil {
				header += "\n(replacing " + e.selItem.Name + ")"
			}
		}
		return ItemCompare{Header: header, Slot: e.slot.SlotName, Env: e.env}
	}
	var out []ItemCompare
	if item.Limit != nil && currentSameUniqueCount == *item.Limit {
		for _, e := range slots {
			if e.isSameUnique {
				out = append(out, block(e))
			}
		}
		return out
	}
	similar := func(compareItem *Item, sameUnique bool) float64 {
		if compareItem == nil {
			return 0
		}
		sameBaseType := !isUnique && compareItem.Rarity != "UNIQUE" && compareItem.Rarity != "RELIC" &&
			item.Base.Type == compareItem.Base.Type && item.Base.SubType == compareItem.Base.SubType
		if sameBaseType || sameUnique {
			return 1
		}
		return 0
	}
	emptyOf := func(e compareSlotEntry) float64 {
		if e.slot.SelItemID == 0 {
			return 1
		}
		return 0
	}
	// a nil parameter is skipped
	sort.SliceStable(slots, func(i, j int) bool {
		a, b := slots[i], slots[j]
		params := [][2]float64{{emptyOf(a), emptyOf(b)}, {similar(a.selItem, a.isSameUnique), similar(b.selItem, b.isSameUnique)}}
		for _, stat := range []string{"FullDPS", "CombinedDPS", "TotalEHP"} {
			av, aok := a.env.Player.Output[stat]
			bv, bok := b.env.Player.Output[stat]
			if aok && bok {
				params = append(params, [2]float64{av, bv})
			}
		}
		for _, p := range params {
			if p[0] > p[1] {
				return true
			}
			if p[0] < p[1] {
				return false
			}
		}
		if a.label != b.label {
			return a.label > b.label
		}
		return a.slot.SlotName > b.slot.SlotName
	})
	for _, e := range slots {
		out = append(out, block(e))
	}
	return out
}
