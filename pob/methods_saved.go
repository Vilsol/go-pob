package pob

func (b *PathOfBuilding) WithMainSocketGroup(mainSocketGroup int) *PathOfBuilding {
	out := *b
	out.Build.MainSocketGroup = mainSocketGroup
	return &out
}

func (b *PathOfBuilding) SetMainSocketGroup(mainSocketGroup int) {
	b.Build.MainSocketGroup = mainSocketGroup
}

func (b *PathOfBuilding) SetSkillGroupName(skillSet int, socketGroup int, label string) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Label = label
}

func (b *PathOfBuilding) SetSocketGroupGems(skillSet int, socketGroup int, gems []Gem) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems = gems
}

func (b *PathOfBuilding) SetSortGemsByDPS(enabled bool) {
	b.Skills.SortGemsByDPS = enabled
}

func (b *PathOfBuilding) SetSortGemsByDPSField(field string) {
	b.Skills.SortGemsByDPSField = field
}

func (b *PathOfBuilding) SetMatchGemLevelToCharacterLevel(enabled bool) {
	b.Skills.MatchGemLevelToCharacterLevel = enabled
}

func (b *PathOfBuilding) SetDefaultGemLevel(gemLevel int) {
	b.Skills.DefaultGemLevel = &gemLevel
}

func (b *PathOfBuilding) SetDefaultGemQuality(gemQuality int) {
	b.Skills.DefaultGemQuality = &gemQuality
}

func (b *PathOfBuilding) SetShowSupportGemTypes(gemTypes string) {
	b.Skills.ShowSupportGemTypes = gemTypes
}

func (b *PathOfBuilding) SetShowAltQualityGems(enabled bool) {
	b.Skills.ShowAltQualityGems = enabled
}

func (b *PathOfBuilding) SetConfigOption(value Input) {
	for i, input := range b.Config.Inputs {
		if input.Name == value.Name {
			b.Config.Inputs[i] = value
			return
		}
	}
	b.Config.Inputs = append(b.Config.Inputs, value)
}

func (b *PathOfBuilding) RemoveConfigOption(name string) {
	toRemove := -1
	for i, input := range b.Config.Inputs {
		if input.Name == name {
			toRemove = i
			break
		}
	}
	if toRemove >= 0 {
		b.Config.Inputs = append(b.Config.Inputs[:toRemove], b.Config.Inputs[toRemove+1:]...)
	}
}

func (b *PathOfBuilding) GetStringOption(name string) string {
	for _, input := range b.Config.Inputs {
		if input.Name == name {
			if input.String == nil {
				return ""
			}

			return *input.String
		}
	}
	return ""
}

func (b *PathOfBuilding) AddNewSocketGroup() {
	b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills = append(b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills, Skill{
		Enabled: true,
	})
}

func (b *PathOfBuilding) DeleteSocketGroup(index int) {
	b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills = append(b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills[:index], b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills[index+1:]...)
}

func (b *PathOfBuilding) DeleteAllSocketGroups() {
	b.Skills.SkillSets[b.Skills.ActiveSkillSet-1].Skills = make([]Skill, 0)
}

func (b *PathOfBuilding) SetClass(clazz string) {
	b.Build.ClassName = clazz
}

func (b *PathOfBuilding) SetAscendancy(ascendancy string) {
	b.Build.AscendClassName = ascendancy
}

func (b *PathOfBuilding) SetLevel(level int) {
	b.Build.Level = level
}
