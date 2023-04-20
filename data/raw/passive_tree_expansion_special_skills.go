package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type PassiveTreeExpansionSpecialSkill struct {
	raw2.PassiveTreeExpansionSpecialSkill
}

var PassiveTreeExpansionSpecialSkills []*PassiveTreeExpansionSpecialSkill

func InitializePassiveTreeExpansionSpecialSkills(version string) error {
	return InitHelper(version, "PassiveTreeExpansionSpecialSkills", &PassiveTreeExpansionSpecialSkills, nil)
}
