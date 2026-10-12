package pob

import "slices"

func (c *Config) storeActive() {
	if len(c.ConfigSets) == 0 {
		c.ConfigSets = []ConfigSet{{ID: 1}}
	}
	cur := c.activeConfigSetIndex()
	c.ConfigSets[cur].Inputs = slices.Clone(c.Inputs)
	c.ConfigSets[cur].Placeholders = slices.Clone(c.Placeholders)
	c.ConfigSets[cur].CustomModifierBlocks = slices.Clone(c.CustomModifierBlocks)
}

func (c *Config) nextConfigSetID() int {
	id := 1
	for slices.ContainsFunc(c.ConfigSets, func(s ConfigSet) bool { return s.ID == id }) {
		id++
	}
	return id
}

//pob:src/Classes/ConfigTab.lua:1224-1244
func (b *PathOfBuilding) NewConfigSet(title string) int {
	c := &b.Config
	if len(c.ConfigSets) == 0 {
		c.ConfigSets = []ConfigSet{{ID: 1}}
	}
	id := c.nextConfigSetID()
	c.ConfigSets = append(c.ConfigSets, ConfigSet{ID: id, Title: &title, CustomModifierBlocks: []CustomModifierBlock{defaultCustomModifierBlock("")}})
	return id
}

//pob:block:src/Classes/ConfigSetListControl.lua:13-22,src/Classes/ConfigSetListControl.lua:49-56
func (b *PathOfBuilding) CopyConfigSet(configSetID int, title string) int {
	c := &b.Config
	c.storeActive()
	i := slices.IndexFunc(c.ConfigSets, func(s ConfigSet) bool { return s.ID == configSetID })
	if i < 0 {
		return 0
	}
	from := c.ConfigSets[i]
	set := ConfigSet{
		ID: c.nextConfigSetID(), Title: &title,
		Inputs: slices.Clone(from.Inputs), Placeholders: slices.Clone(from.Placeholders), CustomModifierBlocks: slices.Clone(from.CustomModifierBlocks),
	}
	c.ConfigSets = append(c.ConfigSets, set)
	return set.ID
}

//pob:block:src/Classes/ConfigSetListControl.lua:49-50
func (b *PathOfBuilding) RenameConfigSet(configSetID int, title string) {
	for i := range b.Config.ConfigSets {
		if b.Config.ConfigSets[i].ID == configSetID {
			b.Config.ConfigSets[i].Title = &title
		}
	}
}

//pob:src/Classes/ConfigSetListControl.lua:85-100
func (b *PathOfBuilding) DeleteConfigSet(configSetID int) {
	c := &b.Config
	if len(c.ConfigSets) <= 1 {
		return
	}
	i := slices.IndexFunc(c.ConfigSets, func(s ConfigSet) bool { return s.ID == configSetID })
	if i < 0 {
		return
	}
	active := c.ConfigSets[c.activeConfigSetIndex()].ID
	c.storeActive()
	c.ConfigSets = slices.Delete(c.ConfigSets, i, i+1)
	if configSetID == active {
		next := c.ConfigSets[max(0, i-1)]
		c.ActiveConfigSet = next.ID
		c.Inputs = slices.Clone(next.Inputs)
		c.Placeholders = slices.Clone(next.Placeholders)
		c.CustomModifierBlocks = slices.Clone(next.CustomModifierBlocks)
	}
}

//pob:block:src/Classes/ListControl.lua:428-436
func (b *PathOfBuilding) MoveConfigSet(from int, to int) {
	sets := b.Config.ConfigSets
	if from == to || from < 0 || to < 0 || from >= len(sets) || to >= len(sets) {
		return
	}
	set := sets[from]
	sets = slices.Delete(sets, from, from+1)
	b.Config.ConfigSets = slices.Insert(sets, to, set)
}
