package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type PassiveTreeExpansionJewel struct {
	raw2.PassiveTreeExpansionJewel
}

var PassiveTreeExpansionJewels []*PassiveTreeExpansionJewel

func InitializePassiveTreeExpansionJewels(version string) error {
	return InitHelper(version, "PassiveTreeExpansionJewels", &PassiveTreeExpansionJewels, nil)
}
