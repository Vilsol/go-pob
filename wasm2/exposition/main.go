//go:build js || bindings2

// The manifest is hidden from the PoE1 generator, which loads every package of the module natively;
// `mise run generate2` runs crystalline with the bindings2 tag.
package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/storage"
)

//go:generate go tool crystalline -app go-pob2 -dir ../.. -js-out ../../frontend/src/lib/types2/index.js -ts-out ../../frontend/src/lib/types2/index.d.ts -banner "/* eslint-disable */" ./wasm2/exposition

//crystalline:exports
func Exports(r bind.Registry) {
	r.Type(Calculator{}, bind.AsPromise("Calculate"))
	r.Type(Outputs{}, bind.Plain())
	r.Type(calculator2.SocketGroupView{}, bind.Plain())
	r.Type(calculator2.ConfigValue{}, bind.Plain())
	r.Type(calculator2.ConfigRelevance{}, bind.Plain())
	r.Type(storage.DirEntry{}, bind.Plain())
	r.Type(TreeState{}, bind.Plain())
	r.Type(TreeNodeText{}, bind.Plain())
	r.Type(TreeHover{}, bind.Plain())
	r.Type(SavedStat{}, bind.Plain())
	r.Type(SavedFullDPS{}, bind.Plain())
	r.Type(CalcsModQuery{}, bind.Plain())
	r.Type(CalcsModEntry{}, bind.Plain())
	r.Type(CalcsOutputs{}, bind.Plain())
	r.Type(SkillGroupView{}, bind.Plain())
	r.Type(SkillGemView{}, bind.Plain())
	r.Type(ItemsView{}, bind.Plain())
	r.Type(ItemView{}, bind.Plain())
	r.Type(ItemsPageView{}, bind.Plain())
	r.Type(ItemSlotView{}, bind.Plain())
	r.Type(ItemSetView{}, bind.Plain())
	r.Type(RuneSlotView{}, bind.Plain())
	r.Type(RuneChoiceView{}, bind.Plain())
	r.Type(DisplayItemView{}, bind.Plain())
	r.Type(calculator2.GemOptions{}, bind.Plain())
	r.Type(calculator2.GemListEntry{}, bind.Plain())
	r.Type(CraftBaseGroup{}, bind.Plain())
	r.Type(ItemDBInfo{}, bind.Plain())
	r.Type(ItemDBRow{}, bind.Plain())
	r.Type(calculator2.ItemDBFilter{}, bind.Plain())
	r.Type(CompareOutputs{}, bind.Plain())
	r.Type(ItemCompareBlock{}, bind.Plain())
	r.Type(ItemCompareView{}, bind.Plain())

	r.Func(ParseBuild)
	r.Func(Preload, bind.AsPromise())
	r.Func(PassiveTreeJSON, bind.AsPromise())
	r.Func(TreeClasses, bind.AsPromise())
	r.Func(GemList, bind.AsPromise())
	r.Value("TreeVersion", TreeVersion)
	r.Func(CraftBases, bind.AsPromise())
	r.Func(CraftItemPreview, bind.AsPromise())
	r.Func(ItemDBLists, bind.AsPromise())
	r.Func(ItemDBTooltip, bind.AsPromise())
	r.Func(ItemDBRaw, bind.AsPromise())

	r.Func(config.InitLogging)
	r.Func(storage.InitializeStorage)
}
