package data

type ItemBaseEnchant struct {
	Enchant         string
	EnchantModTypes [][]string
}

func BaseEnchant(name string) (ItemBaseEnchant, bool) {
	e, ok := itemBaseEnchants[name]
	return e, ok
}
