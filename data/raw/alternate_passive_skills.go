package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type AlternatePassiveSkill struct {
	raw2.AlternatePassiveSkill
}

var AlternatePassiveSkills []*AlternatePassiveSkill

func InitializeAlternatePassiveSkills(version string) error {
	return InitHelper(version, "AlternatePassiveSkills", &AlternatePassiveSkills, nil)
}
