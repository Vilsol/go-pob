package calculator

import (
	"github.com/Vilsol/go-pob/mod"
)

//pob:src/Modules/ModParser.lua:2011-2013
func FLAG(modName string) mod.Mod {
	return MOD(modName, mod.TypeFlag, true)
}

//pob:src/Modules/ModTools.lua:20-46
func MOD(modName string, modType mod.Type, modVal any) mod.Mod {
	var realMod mod.Mod
	switch modType {
	case mod.TypeList:
		realMod = mod.NewList(modName, modVal)
	case mod.TypeFlag:
		realMod = mod.NewFlag(modName, modVal.(bool))
	default:
		switch modVal := modVal.(type) {
		case int:
			realMod = mod.NewFloat(modName, modType, float64(modVal))
		case *float64:
			realMod = mod.NewFloat(modName, modType, *modVal)
		default:
			realMod = mod.NewFloat(modName, modType, modVal.(float64))
		}
	}
	return realMod
}
