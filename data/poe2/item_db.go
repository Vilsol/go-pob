package poe2

import "sort"

// RuneforgingBase is one data.runeforgingCrafts entry: the base and what runeforging or runemastering makes of it.
type RuneforgingBase struct {
	Base   string
	Crafts []RuneforgingCraft
}

type RuneforgingCraft struct {
	BaseTypeID string
	Name       string
}

// RuneforgingCraftsOf is data.runeforgingCrafts[baseName].
//
//pob2:block:src/Modules/ItemTools.lua:92-92
func RuneforgingCraftsOf(baseName string) []RuneforgingCraft {
	i := sort.Search(len(RuneforgingCrafts), func(i int) bool { return RuneforgingCrafts[i].Base >= baseName })
	if i < len(RuneforgingCrafts) && RuneforgingCrafts[i].Base == baseName {
		return RuneforgingCrafts[i].Crafts
	}
	return nil
}

// UniqueDBKind is one data.uniques entry: its raw uniques in order.
type UniqueDBKind struct {
	Kind string
	Raws []string
}
