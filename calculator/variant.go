package calculator

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/pob"
)

type VariantEvaluator struct {
	build *pob.PathOfBuilding
	cache *EnvironmentCache
	mode  OutputMode
}

//pob:src/Modules/Calcs.lua:107-116
func NewVariantEvaluator(build *pob.PathOfBuilding, mode OutputMode) *VariantEvaluator {
	cache := NewEnvironmentCache()

	cache.items = make(map[string]*Item)

	return &VariantEvaluator{
		build: build,
		cache: cache,
		mode:  mode,
	}
}

type Variant struct {
	AllocateNodes   []int64
	DeallocateNodes []int64

	Gems []GemChange

	Items []ItemChange
}

type GemChange struct {
	Group   int
	Index   int
	Enabled *bool
	Level   *int
	Quality *int
	GemID   *string
	SkillID *string
}

type ItemChange struct {
	Slot string
	Text string
}

//pob:src/Modules/Calcs.lua:116-133
func (v *VariantEvaluator) Evaluate(variant Variant) *Environment {
	restore := v.apply(variant)
	defer restore()

	return BuildOutputWithCache(&Calculator{PoB: v.build}, v.mode, v.cache)
}

func (v *VariantEvaluator) apply(variant Variant) func() {
	undo := make([]func(), 0, 3)

	if len(variant.AllocateNodes) > 0 || len(variant.DeallocateNodes) > 0 {
		undo = append(undo, v.applyNodes(variant))
	}
	for _, change := range variant.Gems {
		if u := v.applyGem(change); u != nil {
			undo = append(undo, u)
		}
	}
	if len(variant.Items) > 0 {
		undo = append(undo, v.applyItems(variant.Items))
	}

	return func() {
		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}
	}
}

func (v *VariantEvaluator) applyNodes(variant Variant) func() {
	oldNodes := v.build.Build.PassiveNodes

	specIdx := v.activeSpecIndex()
	var oldAttr string
	if specIdx >= 0 {
		oldAttr = v.build.Tree.Specs[specIdx].NodesAttr
	}

	remove := make(map[int64]bool, len(variant.DeallocateNodes))
	for _, id := range variant.DeallocateNodes {
		remove[id] = true
	}
	present := make(map[int64]bool, len(oldNodes))

	updated := make([]int64, 0, len(oldNodes)+len(variant.AllocateNodes))
	for _, id := range oldNodes {
		if remove[id] || present[id] {
			continue
		}
		present[id] = true
		updated = append(updated, id)
	}
	for _, id := range variant.AllocateNodes {
		if present[id] || remove[id] {
			continue
		}
		present[id] = true
		updated = append(updated, id)
	}

	v.build.Build.PassiveNodes = updated
	if specIdx >= 0 {
		parts := make([]string, len(updated))
		for i, id := range updated {
			parts[i] = strconv.FormatInt(id, 10)
		}
		v.build.Tree.Specs[specIdx].NodesAttr = strings.Join(parts, ",")
	}

	return func() {
		v.build.Build.PassiveNodes = oldNodes
		if specIdx >= 0 {
			v.build.Tree.Specs[specIdx].NodesAttr = oldAttr
		}
	}
}

func (v *VariantEvaluator) activeSpecIndex() int {
	idx := v.build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(v.build.Tree.Specs) {
		if len(v.build.Tree.Specs) == 0 {
			return -1
		}
		return 0
	}
	return idx
}

func (v *VariantEvaluator) applyGem(change GemChange) func() {
	setIdx := activeSkillSetIndex(v.build.Skills)
	if setIdx < 0 {
		return nil
	}

	groups := v.build.Skills.SkillSets[setIdx].Skills
	if change.Group < 0 || change.Group >= len(groups) {
		return nil
	}
	gems := groups[change.Group].Gems
	if change.Index < 0 || change.Index >= len(gems) {
		return nil
	}

	target := &v.build.Skills.SkillSets[setIdx].Skills[change.Group].Gems[change.Index]
	old := *target

	if change.Enabled != nil {
		target.Enabled = *change.Enabled
	}
	if change.Level != nil {
		target.Level = *change.Level
	}
	if change.Quality != nil {
		target.Quality = *change.Quality
	}
	if change.GemID != nil {
		target.GemID = *change.GemID
	}
	if change.SkillID != nil {
		target.SkillID = *change.SkillID
	}

	return func() { *target = old }
}

func (v *VariantEvaluator) applyItems(changes []ItemChange) func() {
	if len(v.build.Items.ItemSets) == 0 {
		return func() {}
	}
	setIdx := activeItemSetIndex(v.build.Items)

	oldItems := v.build.Items.Items
	oldSlots := v.build.Items.ItemSets[setIdx].Slots

	items := make([]pob.Item, len(oldItems), len(oldItems)+len(changes))
	copy(items, oldItems)
	slots := make([]pob.Slot, len(oldSlots))
	copy(slots, oldSlots)

	nextID := 1
	for _, it := range items {
		if it.ID >= nextID {
			nextID = it.ID + 1
		}
	}

	for _, change := range changes {
		itemID := 0
		if change.Text != "" {
			items = append(items, pob.Item{ID: nextID, Text: change.Text})
			itemID = nextID
			nextID++
		}

		found := false
		for i := range slots {
			if slots[i].Name == change.Slot {
				slots[i].ItemID = itemID
				slots[i].Active = itemID != 0 && slots[i].Active
				found = true
				break
			}
		}
		if !found && itemID != 0 {
			slots = append(slots, pob.Slot{Name: change.Slot, ItemID: itemID})
		}
	}

	v.build.Items.Items = items
	v.build.Items.ItemSets[setIdx].Slots = slots

	return func() {
		v.build.Items.Items = oldItems
		v.build.Items.ItemSets[setIdx].Slots = oldSlots
	}
}
