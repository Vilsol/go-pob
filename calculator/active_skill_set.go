package calculator

import "github.com/Vilsol/go-pob/pob"

//pob:block:src/Classes/SkillsTab.lua:1489-1491
func activeSkillSetIndex(skills pob.Skills) int {
	if len(skills.SkillSets) == 0 {
		return -1
	}
	for i, set := range skills.SkillSets {
		if set.ID == skills.ActiveSkillSet {
			return i
		}
	}
	return 0
}
