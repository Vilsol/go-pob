package raw

type CraftingBenchOption struct {
	AddEnchantment              *int   `json:"AddEnchantment"`
	AddMod                      *int   `json:"AddMod"`
	CostBaseItemTypes           []int  `json:"Cost_BaseItemTypes"`
	CostValues                  []int  `json:"Cost_Values"`
	CraftingBenchCustomAction   int    `json:"CraftingBenchCustomAction"`
	CraftingItemClassCategories []int  `json:"CraftingItemClassCategories"`
	Description                 string `json:"Description"`
	HideoutNPCSKey              int    `json:"HideoutNPCsKey"`
	IsAreaOption                bool   `json:"IsAreaOption"`
	IsDisabled                  bool   `json:"IsDisabled"`
	ItemClasses                 []int  `json:"ItemClasses"`
	ItemQuantity                int    `json:"ItemQuantity"`
	Links                       int    `json:"Links"`
	Name                        string `json:"Name"`
	Order                       int    `json:"Order"`
	RecipeIDS                   []int  `json:"RecipeIds"`
	RequiredLevel               int    `json:"RequiredLevel"`
	SocketColours               string `json:"SocketColours"`
	Sockets                     int    `json:"Sockets"`
	SortCategory                int    `json:"SortCategory"`
	Tier                        int    `json:"Tier"`
	UnlockCategory              *int   `json:"UnlockCategory"`
	UnveilsRequired             int    `json:"UnveilsRequired"`
	UnveilsRequired2            int    `json:"UnveilsRequired2"`
	Key                         int    `json:"_key"`
}

var CraftingBenchOptions []*CraftingBenchOption
var CraftingBenchOptionsMap map[int]*CraftingBenchOption

func InitializeCraftingBenchOptions(version string) error {
	if err := InitHelper(version, "CraftingBenchOptions", &CraftingBenchOptions); err != nil {
		return err
	}

	CraftingBenchOptionsMap = make(map[int]*CraftingBenchOption)
	for _, i := range CraftingBenchOptions {
		CraftingBenchOptionsMap[i.Key] = i
	}

	return nil
}
