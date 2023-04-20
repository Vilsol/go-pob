package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type PantheonPanelLayout struct {
	raw2.PantheonPanelLayout
}

var PantheonPanelLayouts []*PantheonPanelLayout

func InitializePantheonPanelLayouts(version string) error {
	return InitHelper(version, "PantheonPanelLayout", &PantheonPanelLayouts, nil)
}
