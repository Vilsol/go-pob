package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ActiveSkill struct {
	raw2.ActiveSkill
}

var ActiveSkills []*ActiveSkill

func InitializeActiveSkills(version string) error {
	return InitHelper(version, "ActiveSkills", &ActiveSkills, nil)
}

func (g *ActiveSkill) GetActiveSkillTypes() []*ActiveSkillType {
	if g.ActiveSkillTypes == nil {
		return nil
	}

	out := make([]*ActiveSkillType, len(g.ActiveSkillTypes))
	for i, skillType := range g.ActiveSkillTypes {
		out[i] = ActiveSkillTypes[skillType]
	}

	return out
}

func (g *ActiveSkill) GetWeaponRestrictions() []*ItemClass {
	if g.WeaponRestrictionItemClassesKeys == nil {
		return nil
	}

	out := make([]*ItemClass, len(g.WeaponRestrictionItemClassesKeys))
	for i, skillType := range g.WeaponRestrictionItemClassesKeys {
		out[i] = ItemClasses[skillType]
	}

	return out
}
