package calculator2

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//pob2:src/Modules/ModParser.lua:2182-2184
func FLAG(modName string) mod.Mod {
	return MOD(modName, mod.TypeFlag, true)
}

//pob2:src/Modules/ModTools.lua:57-91
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

type radiusJewelData struct {
	modSource mod.Source

	stats map[string]float64

	total   float64
	totalOK bool
	modList *moddb.ModList

	funcData map[int]*radiusJewelData
}

type radiusJewelNode struct {
	id          int
	nodeType    string
	isAttribute bool
	modList     *moddb.ModList

	conquered bool

	isTattoo bool
}

type radiusJewelFunc func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData)

type jewelFuncEntry struct {
	Type string
	Func radiusJewelFunc
}

func (s jewelFuncEntry) Clone(pointer bool) mod.Cloneable {
	obj := jewelFuncEntry{Type: s.Type, Func: s.Func}
	if pointer {
		return &obj
	}
	return obj
}

type nodeModifier struct {
	Mod mod.Mod
}

func (s nodeModifier) Clone(pointer bool) mod.Cloneable {
	obj := nodeModifier{Mod: s.Mod}
	if pointer {
		return &obj
	}
	return obj
}

//pob2:src/Classes/ModList.lua:75-88
func mergeMod(list *moddb.ModList, m mod.Mod) {
	if t := m.Type(); t == mod.TypeBase || t == mod.TypeIncrease || t == mod.TypeMore {
		mods := list.Mods()
		for i, existing := range mods {
			if !moddb.CompareModParams(existing, m) {
				continue
			}
			ev, nv := existing.Value(), m.Value()
			if ev == nil || nv == nil ||
				ev.Type() != mod.ModValueMultiTypeFloat || nv.Type() != mod.ModValueMultiTypeFloat {
				continue
			}
			merged := existing.Clone()
			merged.Value().SetFloat(ev.Float() + nv.Float())
			mods[i] = merged
			list.SetMods(mods)
			return
		}
	}
	list.AddMod(m)
}

//pob2:src/Classes/ModList.lua:98-100
func mergeNewMod(list *moddb.ModList, name string, modType mod.Type, value float64, proto mod.Mod) {
	m := mod.NewFloat(name, modType, value).
		Source(proto.GetSource()).
		Flag(proto.Flags()).
		KeywordFlag(proto.KeywordFlags())
	if tags := proto.Tags(); len(tags) > 0 {
		m.Tag(tags...)
	}
	mergeMod(list, m)
}

const conditionAffectedByEnergyBlade = "AffectedByEnergyBlade"

func newRadiusJewelData(source mod.Source) *radiusJewelData {
	return &radiusJewelData{modSource: source, stats: make(map[string]float64)}
}

//pob2:block:src/Modules/Data.lua:411-418
var (
	dataNonDamagingAilmentTypeList = []string{"Chill", "Freeze", "Shock"}
	dataNonDamagingAilmentDefault  = map[string]float64{
		"Chill": 30,
		"Shock": poe2.GameConstants["BaseShockMagnitude"],
	}
)

// LuaTable carries a LIST value whose Lua table has no typed Go counterpart, keyed by its Lua field names.
type LuaTable map[string]any

func (l LuaTable) Clone(pointer bool) mod.Cloneable {
	out := make(LuaTable, len(l))
	for k, v := range l {
		switch v := v.(type) {
		case mod.Mod:
			out[k] = v.Clone()
		case []any:
			out[k] = append([]any(nil), v...)
		default:
			out[k] = v
		}
	}
	if pointer {
		return &out
	}
	return out
}

// tonumber is Lua's tonumber for a capture: 0 where Lua gives nil, without the warning utils.Float logs.
func tonumber(s string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return n
}
