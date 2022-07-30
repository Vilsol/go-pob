package calculator

import "go-pob/calculator/mod"

func CalcMod(list ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	return (1 + list.Sum(mod.TypeIncrease, cfg, names...)/100) * list.More(cfg, names...)
}

func CalcVal(modStore ModStoreFuncs, name string, cfg *ListCfg) float64 {
	baseVal := modStore.Sum(mod.TypeBase, cfg, name)
	if baseVal != 0 {
		return baseVal * CalcMod(modStore, cfg, name)
	}
	return 0
}
