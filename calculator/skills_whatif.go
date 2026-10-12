package calculator

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/pob"
)

func (m *MiscCalculator) CalcGem(socketGroup int, gemIndex int, gem pob.Gem) *Environment {
	return m.over(func(b *pob.PathOfBuilding) {
		b.Skills.SkillSets = slices.Clone(b.Skills.SkillSets)
		set := &b.Skills.SkillSets[b.ActiveSkillSetIndex()]
		set.Skills = slices.Clone(set.Skills)
		if socketGroup < 0 || socketGroup >= len(set.Skills) {
			return
		}
		group := &set.Skills[socketGroup]
		group.Gems = slices.Clone(group.Gems)
		switch {
		case gemIndex >= 0 && gemIndex < len(group.Gems):
			group.Gems[gemIndex] = gem
		case gemIndex == len(group.Gems):
			group.Gems = append(group.Gems, gem)
		}
	}).Calc(&CalcOverride{})
}

func (m *MiscCalculator) CalcConfig(input pob.Input, remove bool) *Environment {
	return m.over(func(b *pob.PathOfBuilding) {
		b.Config.Inputs = slices.Clone(b.Config.Inputs)
		if remove {
			b.RemoveConfigOption(input.Name)
		} else {
			b.SetConfigOption(input)
		}
	}).Calc(&CalcOverride{})
}

//pob:block:src/Classes/ConfigTab.lua:53-68
func (m *MiscCalculator) CalcCustomModGroup(i int) *Environment {
	return m.over(func(b *pob.PathOfBuilding) {
		b.Config.Inputs = slices.Clone(b.Config.Inputs)
		blocks := b.GetCustomModifierBlocks()
		if i >= 0 && i < len(blocks) {
			enabled := strconv.FormatBool(!blocks[i].IsEnabled())
			blocks[i].Enabled = &enabled
		}
		b.SetCustomModifierBlocks(blocks)
	}).Calc(&CalcOverride{})
}
