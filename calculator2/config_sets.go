package calculator2

import (
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/pob2"
)

// ConfigSetView is one entry of the config set list, in configSetOrderList's order.
type ConfigSetView struct {
	ID    int
	Title string
}

// configSets is SetActiveConfigSet's first step; a build saved before config sets holds its options directly under
// Config, which Load reads as set 1.
//
//pob2:block:src/Classes/ConfigTab.lua:1409-1412
func configSets(build *pob2.PathOfBuilding2) []pob2.ConfigSet {
	c := &build.Config
	if len(c.ConfigSets) == 0 {
		c.ConfigSets = []pob2.ConfigSet{{ID: 1, Inputs: c.Inputs, Placeholders: c.Placeholders, CustomModifierBlocks: c.CustomModifierBlocks}}
		c.Inputs, c.Placeholders, c.CustomModifierBlocks = nil, nil, nil
	}
	return c.ConfigSets
}

func configSetIndex(build *pob2.PathOfBuilding2, id int) int {
	return slices.IndexFunc(configSets(build), func(s pob2.ConfigSet) bool { return s.ID == id })
}

// ConfigSetList is the set selector's list and the active set's id.
//
//pob2:block:src/Classes/ConfigSetListControl.lua:93-99
func ConfigSetList(build *pob2.PathOfBuilding2) ([]ConfigSetView, int) {
	sets := configSets(build)
	out := make([]ConfigSetView, len(sets))
	for i, s := range sets {
		out[i] = ConfigSetView{ID: s.ID, Title: "Default"}
		if s.Title != nil {
			out[i].Title = *s.Title
		}
	}
	return out, activeConfigSetID(build)
}

//pob2:src/Classes/ConfigTab.lua:1407-1434
func SetActiveConfigSet(build *pob2.PathOfBuilding2, id int) {
	sets := configSets(build)
	if configSetIndex(build, id) < 0 {
		id = sets[0].ID
	}
	build.Config.ActiveConfigSet = id
}

// Go must differ: a free id in place of #self.configSets + 1, whose border is undefined once a set is deleted.
func nextConfigSetID(build *pob2.PathOfBuilding2) int {
	id := 1
	for configSetIndex(build, id) >= 0 {
		id++
	}
	return id
}

// NewConfigSet is ConfigSetService:NewConfigSet: a set with the options' defaults and one empty custom modifier block,
// made active. The defaults are absent rather than written, as Save drops an input at its default state.
//
//pob2:block:src/Classes/ConfigSetService.lua:17-22
//pob2:block:src/Classes/ConfigTab.lua:1317-1342
func NewConfigSet(build *pob2.PathOfBuilding2, title string) int {
	id := nextConfigSetID(build)
	enabled := true
	def := "Default"
	build.Config.ConfigSets = append(configSets(build), pob2.ConfigSet{ID: id, Title: &title, CustomModifierBlocks: []pob2.CustomModifierBlock{{Title: &def, Enabled: &enabled}}})
	SetActiveConfigSet(build, id)
	return id
}

//pob2:block:src/Classes/ConfigSetService.lua:24-29
//pob2:block:src/Classes/ConfigTab.lua:1344-1353
func CopyConfigSet(build *pob2.PathOfBuilding2, id int, title string) int {
	i := configSetIndex(build, id)
	if i < 0 {
		return 0
	}
	from := build.Config.ConfigSets[i]
	set := pob2.ConfigSet{
		ID: nextConfigSetID(build), Title: &title, Inputs: slices.Clone(from.Inputs), Placeholders: slices.Clone(from.Placeholders),
		CustomModifierBlocks: slices.Clone(from.CustomModifierBlocks),
	}
	build.Config.ConfigSets = append(build.Config.ConfigSets, set)
	SetActiveConfigSet(build, set.ID)
	return set.ID
}

//pob2:block:src/Classes/ConfigTab.lua:1355-1364
func RenameConfigSet(build *pob2.PathOfBuilding2, id int, title string) {
	if i := configSetIndex(build, id); i >= 0 {
		build.Config.ConfigSets[i].Title = &title
	}
}

//pob2:block:src/Classes/ConfigSetService.lua:37-46
//pob2:block:src/Classes/ConfigTab.lua:1367-1371
func DeleteConfigSet(build *pob2.PathOfBuilding2, id int) {
	sets := configSets(build)
	i := configSetIndex(build, id)
	if len(sets) <= 1 || i < 0 {
		return
	}
	build.Config.ConfigSets = slices.Delete(sets, i, i+1)
	if id == activeConfigSetID(build) {
		SetActiveConfigSet(build, build.Config.ConfigSets[max(0, i-1)].ID)
	}
}

// MoveConfigSet is the list's drag reorder of configSetOrderList, from one index to another.
//
//pob2:block:src/Classes/ListControl.lua:441-449
func MoveConfigSet(build *pob2.PathOfBuilding2, from, to int) {
	sets := configSets(build)
	if from == to || from < 0 || to < 0 || from >= len(sets) || to >= len(sets) {
		return
	}
	set := sets[from]
	build.Config.ConfigSets = slices.Insert(slices.Delete(sets, from, from+1), to, set)
}

// CustomModifierBlocks is the active config set's customModsList.
func CustomModifierBlocks(build *pob2.PathOfBuilding2) []pob2.CustomModifierBlock {
	return customModifierBlocks(build)
}

// SetCustomModifierBlocks replaces the active config set's customModsList, as the blocks' controls edit it.
//
//pob2:block:src/Classes/ConfigTab.lua:32-98
func SetCustomModifierBlocks(build *pob2.PathOfBuilding2, blocks []pob2.CustomModifierBlock) {
	configSets(build)
	if i := configSetIndex(build, activeConfigSetID(build)); i >= 0 {
		build.Config.ConfigSets[i].CustomModifierBlocks = blocks
	}
}

// CustomModLine is a custom modifier line as BuildModList hands it to parseMod.
//
//pob2:block:src/Classes/ConfigTab.lua:1208
func CustomModLine(line string) string {
	return strings.Trim(stripEscapes(line), luaSpace)
}
