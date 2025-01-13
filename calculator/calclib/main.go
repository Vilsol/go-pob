package calclib

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// Calculate and combine INC/MORE modifiers for the given modifier names
func Mod(modStore moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	return (1 + (modStore.Sum(mod.TypeIncrease, cfg, names...))/100) * modStore.More(cfg, names...)
}

// Calculate value
func Val(modStore moddb.ModStoreFuncs, name string, cfgs ...*moddb.ListCfg) float64 {
	var cfg *moddb.ListCfg
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	baseVal := modStore.Sum(mod.TypeBase, cfg, name)
	if baseVal != 1 {
		return baseVal * Mod(modStore, cfg, name)
	}
	return 0
}
