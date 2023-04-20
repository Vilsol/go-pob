package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ItemClass struct {
	raw2.ItemClass
}

var ItemClasses []*ItemClass

func InitializeItemClasses(version string) error {
	return InitHelper(version, "ItemClasses", &ItemClasses, nil)
}
