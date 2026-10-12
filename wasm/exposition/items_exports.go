package exposition

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

//crystalline:exports
func Items30Exports(r bind.Registry) {
	r.Type(ItemSetTip{}, bind.Plain())

	r.Func(ItemSetTooltips, bind.AsPromise())
	r.Func(DropItemOnSlot, bind.AsPromise())
	r.Func(EquipItemInSet, bind.AsPromise())
	r.Func(ItemListLoadoutFilters, bind.AsPromise())
	r.Func(ItemListFiltered, bind.AsPromise())
	r.Type(SharedItemSetRow{}, bind.Plain())
	r.Func(SharedItemSets, bind.AsPromise())
	r.Func(ShareItemSet, bind.AsPromise())
	r.Func(ImportSharedItemSet, bind.AsPromise())
	r.Func(DeleteSharedItemSet, bind.AsPromise())
	r.Func(RenameSharedItemSet, bind.AsPromise())
	r.Func(SharedItems, bind.AsPromise())
	r.Func(ShareItem, bind.AsPromise())
	r.Func(DeleteSharedItem, bind.AsPromise())
	r.Func(AddSharedItem, bind.AsPromise())
	r.Func(NormaliseDisplayItem, bind.AsPromise())
	r.Func(SetShowFlavourText, bind.AsPromise())
}

type ItemSetTip struct {
	ID    string
	Lines []string
}

func ItemSetTooltips(build *pob.PathOfBuilding) []ItemSetTip {
	if build == nil {
		return nil
	}
	out := make([]ItemSetTip, 0, len(build.Items.ItemSets))
	for _, set := range build.Items.ItemSets {
		out = append(out, ItemSetTip{ID: set.ID, Lines: calculator.ItemSetTooltip(build, set)})
	}
	return out
}

func (m *MiscCalculator) ConditionalSlots() []string {
	return m.calc.ShownConditionalSlots()
}

func DropItemOnSlot(build *pob.PathOfBuilding, slotName string, id int, kind string, name string) bool {
	return calculator.DropItemOnSlot(build, slotName, id, calculator.ItemDBKind(kind), name)
}

func EquipItemInSet(build *pob.PathOfBuilding, setID int, id int, kind string, name string, second bool) bool {
	return calculator.EquipItemInSet(build, setID, id, calculator.ItemDBKind(kind), name, second)
}

func ItemListLoadoutFilters(build *pob.PathOfBuilding) []string {
	return calculator.ItemListLoadoutFilters(build)
}

func ItemListFiltered(build *pob.PathOfBuilding, filter string) []int {
	return calculator.ItemListFiltered(build, filter)
}

const sharedItemSetsKey = "shared-item-sets.json"

func loadSharedItemSets() []calculator.SharedItemSet {
	var out []calculator.SharedItemSet
	if raw, err := storage.Get().GetSetting(sharedItemSetsKey); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func saveSharedItemSets(list []calculator.SharedItemSet) error {
	raw, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("shared item sets: %w", err)
	}
	return storage.Get().SetSetting(sharedItemSetsKey, raw) //nolint:wrapcheck // the storage's own error, passed on
}

type SharedItemSetRow struct {
	Title string
	Tip   []string
}

func SharedItemSets() []SharedItemSetRow {
	list := loadSharedItemSets()
	out := make([]SharedItemSetRow, len(list))
	for k, s := range list {
		out[k] = SharedItemSetRow{Title: s.Title, Tip: calculator.SharedItemSetTooltip(s)}
	}
	return out
}

func ShareItemSet(build *pob.PathOfBuilding, setID int, index int) bool {
	shared, ok := calculator.ShareItemSet(build, setID)
	if !ok {
		return false
	}
	list := loadSharedItemSets()
	list = slices.Insert(list, min(max(index, 0), len(list)), shared)
	return saveSharedItemSets(list) == nil
}

func ImportSharedItemSet(build *pob.PathOfBuilding, index int) int {
	list := loadSharedItemSets()
	if index < 0 || index >= len(list) {
		return 0
	}
	return calculator.ImportSharedItemSet(build, list[index])
}

func DeleteSharedItemSet(index int) bool {
	list := loadSharedItemSets()
	if index < 0 || index >= len(list) {
		return false
	}
	return saveSharedItemSets(slices.Delete(list, index, index+1)) == nil
}

func RenameSharedItemSet(index int, title string) bool {
	list := loadSharedItemSets()
	if index < 0 || index >= len(list) {
		return false
	}
	list[index].Title = title
	return saveSharedItemSets(list) == nil
}

const sharedItemsKey = "shared-items.json"

func loadSharedItems() []string {
	var out []string
	if raw, err := storage.Get().GetSetting(sharedItemsKey); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func saveSharedItems(list []string) error {
	raw, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("shared items: %w", err)
	}
	return storage.Get().SetSetting(sharedItemsKey, raw) //nolint:wrapcheck // the storage's own error, passed on
}

func SharedItems() []string {
	return loadSharedItems()
}

func ShareItem(build *pob.PathOfBuilding, id int, kind string, name string, index int) bool {
	text, ok := calculator.ShareItem(build, id, calculator.ItemDBKind(kind), name)
	if !ok {
		return false
	}
	list := loadSharedItems()
	return saveSharedItems(slices.Insert(list, min(max(index, 0), len(list)), text)) == nil
}

func DeleteSharedItem(index int) bool {
	list := loadSharedItems()
	if index < 0 || index >= len(list) {
		return false
	}
	return saveSharedItems(slices.Delete(list, index, index+1)) == nil
}

func AddSharedItem(build *pob.PathOfBuilding, index int, slotName string) int {
	list := loadSharedItems()
	if build == nil || index < 0 || index >= len(list) {
		return 0
	}
	id, _ := calculator.AddSharedItem(build, list[index], slotName)
	return id
}

func NormaliseDisplayItem(build *pob.PathOfBuilding, text string, copyEldritchImplicits bool) string {
	if build == nil {
		return text
	}
	return calculator.NormaliseDisplayItem(build, text, copyEldritchImplicits)
}

var showFlavourText = true

func SetShowFlavourText(on bool) {
	showFlavourText = on
}
