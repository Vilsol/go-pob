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
