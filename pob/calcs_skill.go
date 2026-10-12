package pob

//pob:block:src/Classes/CalcsTab.lua:54-59
func (b *PathOfBuilding) SetMainActiveSkillCalcs(skillSet int, socketGroup int, mainActiveSkill int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].MainActiveSkillCalcs = mainActiveSkill
}

//pob:block:src/Classes/CalcsTab.lua:61-68
func (b *PathOfBuilding) SetGemSkillPartCalcs(skillSet int, socketGroup int, gem int, skillPart int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillPartCalcs = skillPart
}

//pob:block:src/Classes/CalcsTab.lua:69-76
func (b *PathOfBuilding) SetGemSkillStageCountCalcs(skillSet int, socketGroup int, gem int, stageCount int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillStageCountCalcs = stageCount
}

//pob:block:src/Classes/CalcsTab.lua:78-85
func (b *PathOfBuilding) SetGemSkillMineCountCalcs(skillSet int, socketGroup int, gem int, mineCount int) {
	var count *int
	if mineCount >= 0 {
		count = &mineCount
	}
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMineCountCalcs = count
}

//pob:block:src/Classes/CalcsTab.lua:93-104
func (b *PathOfBuilding) SetGemSkillMinionCalcs(skillSet int, socketGroup int, gem int, minionID string) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinionCalcs = &minionID
}

//pob:block:src/Classes/CalcsTab.lua:93-104
func (b *PathOfBuilding) SetGemSkillMinionItemSetCalcs(skillSet int, socketGroup int, gem int, itemSetID int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinionItemSetCalcs = &itemSetID
}

//pob:block:src/Classes/CalcsTab.lua:111-118
func (b *PathOfBuilding) SetGemSkillMinionSkillCalcs(skillSet int, socketGroup int, gem int, minionSkill int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinionSkillCalcs = &minionSkill
}

//pob:block:src/Classes/CalcsTab.lua:208-216
func (b *PathOfBuilding) SetCalcsSectionCollapsed(id string, subsection string, collapsed bool) {
	sections := make([]Section, 0, len(b.Calcs.Sections)+1)
	found := false
	for _, s := range b.Calcs.Sections {
		if s.ID == id && s.Subsection == subsection {
			s.Collapsed = collapsed
			found = true
		}
		sections = append(sections, s)
	}
	if !found {
		sections = append(sections, Section{ID: id, Subsection: subsection, Collapsed: collapsed})
	}
	b.Calcs.Sections = sections
}

//pob:block:src/Classes/CalcsTab.lua:174-189
func (b *PathOfBuilding) CalcsSectionCollapsed(id string, subsection string) (bool, bool) {
	for _, s := range b.Calcs.Sections {
		if s.ID == id && s.Subsection == subsection {
			return s.Collapsed, true
		}
	}
	return false, false
}
