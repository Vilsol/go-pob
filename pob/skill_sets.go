package pob

import "slices"

func (b *PathOfBuilding) nextSkillSetID() int {
	id := 1
	for slices.ContainsFunc(b.Skills.SkillSets, func(s SkillSet) bool { return s.ID == id }) {
		id++
	}
	return id
}

//pob:src/Classes/SkillsTab.lua:1452-1462
func (b *PathOfBuilding) NewSkillSet(title string) int {
	id := b.nextSkillSetID()
	b.Skills.SkillSets = append(b.Skills.SkillSets, SkillSet{ID: id, Title: &title, Skills: []Skill{}})
	return id
}

//pob:block:src/Classes/SkillSetListControl.lua:20-38,src/Classes/SkillSetListControl.lua:66-73
func (b *PathOfBuilding) CopySkillSet(skillSetID int, title string) int {
	i := slices.IndexFunc(b.Skills.SkillSets, func(s SkillSet) bool { return s.ID == skillSetID })
	if i < 0 {
		return 0
	}
	from := b.Skills.SkillSets[i]
	set := SkillSet{ID: b.nextSkillSetID(), Title: &title, Skills: make([]Skill, len(from.Skills))}
	for g, group := range from.Skills {
		group.Gems = slices.Clone(group.Gems)
		set.Skills[g] = group
	}
	b.Skills.SkillSets = append(b.Skills.SkillSets, set)
	return set.ID
}

//pob:block:src/Classes/SkillSetListControl.lua:66-67
func (b *PathOfBuilding) RenameSkillSet(skillSetID int, title string) {
	for i := range b.Skills.SkillSets {
		if b.Skills.SkillSets[i].ID == skillSetID {
			b.Skills.SkillSets[i].Title = &title
		}
	}
}

//pob:src/Classes/SkillSetListControl.lua:106-117
func (b *PathOfBuilding) DeleteSkillSet(skillSetID int) {
	if len(b.Skills.SkillSets) <= 1 {
		return
	}
	i := slices.IndexFunc(b.Skills.SkillSets, func(s SkillSet) bool { return s.ID == skillSetID })
	if i < 0 {
		return
	}
	b.Skills.SkillSets = slices.Delete(b.Skills.SkillSets, i, i+1)
	if skillSetID == b.Skills.ActiveSkillSet {
		b.SetActiveSkillSet(b.Skills.SkillSets[max(0, i-1)].ID)
	}
}

//pob:src/Classes/ImportTab.lua:1901-1911
func (b *PathOfBuilding) GuessMainSocketGroup() int {
	largestGroupSize, largestGroupIndex := 0, 1
	if len(b.Skills.SkillSets) == 0 {
		return largestGroupIndex
	}
	for i, group := range b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills {
		if len(group.Gems) > largestGroupSize {
			largestGroupSize = len(group.Gems)
			largestGroupIndex = i + 1
		}
	}
	return largestGroupIndex
}

//pob:block:src/Classes/SkillsTab.lua:322-350
func (b *PathOfBuilding) SetSocketGroupImbuedSupport(skillSet int, socketGroup int, name string) {
	group := &b.Skills.SkillSets[skillSet].Skills[socketGroup]
	if group.Slot == "" {
		return
	}
	if name == "" {
		group.ImbuedSupport = nil
		return
	}
	group.ImbuedSupport = &name
}

//pob:block:src/Classes/SkillsTab.lua:172-188
func (b *PathOfBuilding) SetSocketGroupSlot(skillSet int, socketGroup int, slot string) {
	groups := b.Skills.SkillSets[skillSet].Skills
	group := &groups[socketGroup]
	if group.Slot != "" && group.ImbuedSupport != nil {
		taken := false
		for i := range groups {
			if i != socketGroup && slot != "" && groups[i].Slot == slot && groups[i].ImbuedSupport != nil {
				taken = true
			}
		}
		if slot == "" || taken {
			group.ImbuedSupport = nil
		}
	}
	group.Slot = slot
}
