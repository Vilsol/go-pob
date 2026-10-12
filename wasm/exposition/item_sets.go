package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func ItemSetExports(r bind.Registry) {
	r.Type(ItemsTabSlotList{}, bind.Plain())
	r.Type(SocketSlot{}, bind.Plain())
	r.Type(AddedModCompare{}, bind.Plain())
	r.Func(ItemsTabSlots, bind.AsPromise())
	r.Func(SetWeaponSet, bind.AsPromise())
	r.Func(NewItemSet, bind.AsPromise())
	r.Func(CopyItemSet, bind.AsPromise())
	r.Func(RenameItemSet, bind.AsPromise())
	r.Func(DeleteItemSet, bind.AsPromise())
	r.Func(MoveItemSet, bind.AsPromise())
	r.Func(DeleteItemPrompt, bind.AsPromise())
	r.Func(DeleteItem, bind.AsPromise())
	r.Func(DeleteUnusedItems, bind.AsPromise())
	r.Func(DeleteAllItems, bind.AsPromise())
	r.Func(SortItems, bind.AsPromise())
	r.Func(ToggleEquipItem, bind.AsPromise())
}

func SetWeaponSet(build *pob.PathOfBuilding, second bool) bool {
	return build.SetUseSecondWeaponSet(second)
}

func NewItemSet(build *pob.PathOfBuilding, title string) int {
	return build.NewItemSet(title)
}

func CopyItemSet(build *pob.PathOfBuilding, itemSetID int, title string) int {
	return build.CopyItemSet(itemSetID, title)
}

func RenameItemSet(build *pob.PathOfBuilding, itemSetID int, title string) {
	build.RenameItemSet(itemSetID, title)
}

func DeleteItemSet(build *pob.PathOfBuilding, itemSetID int) {
	if build.DeleteItemSet(itemSetID) {
		calculator.PopulateSlots(build)
	}
}

func MoveItemSet(build *pob.PathOfBuilding, from, to int) {
	build.MoveItemSet(from, to)
}

func DeleteItemPrompt(build *pob.PathOfBuilding, itemID int) string {
	return calculator.DeleteItemPrompt(build, itemID)
}

func DeleteItem(build *pob.PathOfBuilding, itemID int) bool {
	return calculator.DeleteItems(build, []int{itemID})
}

func DeleteUnusedItems(build *pob.PathOfBuilding) bool {
	return calculator.DeleteItems(build, calculator.UnusedItemIDs(build))
}

func DeleteAllItems(build *pob.PathOfBuilding) {
	calculator.DeleteAllItems(build)
}

func SortItems(build *pob.PathOfBuilding) {
	calculator.SortItems(build)
}

func ToggleEquipItem(build *pob.PathOfBuilding, itemID int, second bool) bool {
	return calculator.ToggleEquipItem(build, itemID, second)
}

type SocketSlot struct {
	NodeID  int
	ItemID  int
	Choices []int
}

type ItemsTabSlotList struct {
	Abyssal []string
	Sockets []SocketSlot
}

func ItemsTabSlots(build *pob.PathOfBuilding) ItemsTabSlotList {
	out := ItemsTabSlotList{Abyssal: calculator.ShownAbyssalSlots(build)}
	for _, id := range calculator.ActiveTreeSockets(build) {
		out.Sockets = append(out.Sockets, SocketSlot{NodeID: id, ItemID: build.SocketedJewel(id), Choices: calculator.TreeSocketItemIDs(build, id)})
	}
	return out
}

func (m *MiscCalculator) SocketJewelTooltip(nodeID int, stats calculator.ItemTooltipStats) (calculator.ItemTooltip, bool) {
	build := m.calc.Build()
	ctx := calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, Stats: stats, CharacterLevel: build.Build.Level, ClassName: string(build.Build.ClassName)}
	return m.calc.SocketJewelTooltip(nodeID, ctx)
}

type AddedModCompare struct {
	Base  StatOutputs
	Added StatOutputs
}

func (m *MiscCalculator) CompareAddedMod(text string, lines []string, modType string) (AddedModCompare, bool) {
	base, added := m.calc.CompareAddedMod(text, lines, modType)
	if base == nil || added == nil {
		return AddedModCompare{}, false
	}
	return AddedModCompare{Base: statOutputs(base), Added: statOutputs(added)}, true
}
