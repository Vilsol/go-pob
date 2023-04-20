package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type PassiveTreeExpansionSkill struct {
	raw2.PassiveTreeExpansionSkill
}

var PassiveTreeExpansionSkills []*PassiveTreeExpansionSkill

func InitializePassiveTreeExpansionSkills(version string) error {
	return InitHelper(version, "PassiveTreeExpansionSkills", &PassiveTreeExpansionSkills, nil)
}
