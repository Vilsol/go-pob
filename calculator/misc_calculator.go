package calculator

import (
	"runtime"
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

//pob:src/Modules/CalcSetup.lua:386
type CalcOverride struct {
	Conditions     []string
	AddNodes       []int64
	RemoveNodes    []int64
	RepSlotName    string
	RepItem        *Item
	ToggleFlask    *Item
	ToggleTincture *Item
	SocketNodeID   int
	SocketItem     *Item
	AddMods        []mod.Mod
}

func (o *CalcOverride) removesNode(id int64) bool {
	if o == nil {
		return false
	}
	for _, r := range o.RemoveNodes {
		if r == id {
			return true
		}
	}
	return false
}

func (o *CalcOverride) removesNodeKey(id string) bool {
	if o == nil || len(o.RemoveNodes) == 0 {
		return false
	}
	n, err := strconv.ParseInt(id, 10, 64)
	return err == nil && o.removesNode(n)
}

type MiscCalculator struct {
	build *pob.PathOfBuilding
	cache *EnvironmentCache
	base  *Environment
}

//pob:src/Modules/Calcs.lua:123-133
func GetMiscCalculator(build *pob.PathOfBuilding) *MiscCalculator {
	m := &MiscCalculator{build: build, cache: envCache}
	m.base = m.run(nil)
	return m
}

func (m *MiscCalculator) Base() *Environment {
	return m.base
}

func MiscCalculatorOver(build *pob.PathOfBuilding, c *Calculator) *MiscCalculator {
	if build == nil || c == nil || c.main == nil {
		return nil
	}
	return &MiscCalculator{build: build, cache: envCache, base: c.main}
}

//pob:src/Modules/Calcs.lua:134-147
func (m *MiscCalculator) Calc(override *CalcOverride) *Environment {
	if override != nil && override.SocketNodeID != 0 {
		if orphans := socketOrphans(m.build, override.SocketNodeID, override.SocketItem); len(orphans) > 0 {
			o := *override
			o.RemoveNodes = append(slices.Clone(o.RemoveNodes), orphans...)
			override = &o
		}
	}
	return m.run(override)
}

func (m *MiscCalculator) run(override *CalcOverride) *Environment {
	runtime.Gosched()
	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnvOverride(m.build, m.cache, OutputModeCalculator, override)
	PerformCalc(env)
	fullDPS, fullDotDPS, _ := calcFullDPS(&Calculator{PoB: m.build}, env, OutputModeCalculator, m.cache, env.override, &calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	env.Player.Output["FullDPS"] = fullDPS
	env.Player.Output["FullDotDPS"] = fullDotDPS
	return env
}
