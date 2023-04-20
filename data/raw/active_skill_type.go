package raw

import (
	raw2 "github.com/Vilsol/go-pob-data/raw"
)

type ActiveSkillType struct {
	raw2.ActiveSkillType
}

var ActiveSkillTypes []*ActiveSkillType

func InitializeActiveSkillTypes(version string) error {
	return InitHelper(version, "ActiveSkillType", &ActiveSkillTypes, nil)
}
