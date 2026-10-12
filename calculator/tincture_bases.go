package calculator

import "github.com/Vilsol/go-pob/data/raw"

type tinctureBase struct {
	ManaBurn float64
	Cooldown float64
}

//pob:block:src/Export/Scripts/bases.lua:272-275
func tinctureBaseOf(name string) tinctureBase {
	for _, t := range raw.Tinctures() {
		if base := t.GetBaseItem(); base != nil && base.Name == name {
			return tinctureBase{ManaBurn: float64(t.DebuffInterval) / 1000, Cooldown: float64(t.Cooldown) / 1000}
		}
	}
	return tinctureBase{}
}
