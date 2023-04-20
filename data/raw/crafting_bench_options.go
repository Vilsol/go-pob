package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type CraftingBenchOption struct {
	raw2.CraftingBenchOption
}

var CraftingBenchOptions []*CraftingBenchOption

func InitializeCraftingBenchOptions(version string) error {
	return InitHelper(version, "CraftingBenchOptions", &CraftingBenchOptions, nil)
}
