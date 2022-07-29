package calculator

import "go-pob/calculator/mod"

func CalcMod(list *ModList, cfg *ListCfg, names ...string) float64 {
	return (1 + list.Sum(mod.TypeIncrease, cfg, names...)/100) * list.More(cfg, names...)
}
