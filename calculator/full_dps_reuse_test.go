package calculator

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

var fullDPSReadKeys = []string{
	"TotalDPS", "BleedDPS", "CorruptingBloodDPS", "IgniteDPS", "BurningGroundDPS", "PoisonDPS",
	"TotalPoisonDPS", "CausticGroundDPS", "ImpaleDPS", "DecayDPS", "TotalDot", "CullMultiplier",
}

func compareFullDPSKeys(t *testing.T, where string, got, want map[string]float64) int {
	t.Helper()
	bad := 0
	for _, k := range fullDPSReadKeys {
		g, gok := got[k]
		w, wok := want[k]
		if gok != wok || (g != w && !(math.IsNaN(g) && math.IsNaN(w))) {
			t.Errorf("%s %s: reused %v (set %v), first pass %v (set %v)", where, k, g, gok, w, wok)
			bad++
		}
	}
	return bad
}

var fullDPSReuseBuilds = []string{
	"1591-necromancer-summon-holy-relic-of-conviction-15.xml",
	"0008-necromancer-vaal-summon-skeletons.xml",
	"0012-occultist-unknown.xml",
	"0019-champion-herald-of-agony.xml",
	"0002-pathfinder-unknown.xml",
	"0461-hierophant-blade-blast.xml",
	"0597-warden-split-arrow.xml",
	"0188-champion-generals-cry.xml",
	"0299-pathfinder-toxic-rain-8.xml",
	"0353-reliquarian-blade-blast-3.xml",
	"0444-reliquarian-static-strike-2.xml",
}

func fullDPSPasses(build *pob.PathOfBuilding, mode OutputMode, visit func(index int, fullEnv *Environment)) {
	cache := NewEnvironmentCache()
	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(build, cache, mode, false, nil)
	fullEnv, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(build, cache, mode, false, nil,
		&calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	for i, activeSkill := range fullEnv.Player.ActiveSkillList {
		group, ok := activeSkill.SocketGroup.(*pob.Skill)
		if !ok || group == nil || group.IncludeInFullDPS == nil || !*group.IncludeInFullDPS {
			continue
		}
		if _, enabled := getActiveSkillCount(activeSkill); !enabled {
			continue
		}
		fullEnv.Player.MainSkill = activeSkill
		performCalc(fullEnv, true)
		visit(i, fullEnv)
		fullEnv = reinitFullDPSEnv(fullEnv, cache, mode, cachedPlayerDB, cachedEnemyDB, cachedMinionDB)
	}
}

func firstFullDPSPass(build *pob.PathOfBuilding, mode OutputMode, index int) *Environment {
	cache := NewEnvironmentCache()
	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(build, cache, mode, false, nil)
	env, _, _, _ := initEnvSpec(build, cache, mode, false, nil,
		&calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	env.Player.MainSkill = env.Player.ActiveSkillList[index]
	performCalc(env, true)
	return env
}

// TestFullDPSPerSkillPassMatchesUncached: A reused FullDPS pass must equal the same skill's first pass.
func TestFullDPSPerSkillPassMatchesUncached(t *testing.T) {
	initBenchData(t)
	names := fullDPSReuseBuilds
	if list := os.Getenv("FULLDPS_REUSE_LIST"); list != "" {
		raw, err := os.ReadFile(list)
		if err != nil {
			t.Fatal(err)
		}
		names = strings.Fields(string(raw))
	}
	for _, name := range names {
		t.Run(name[:4], func(t *testing.T) {
			build := loadHeavyBuild(t, name)
			compared, minions, triggered, totems, mirages := 0, 0, 0, 0, 0
			fullDPSPasses(build, OutputModeCalculator, func(i int, got *Environment) {
				want := firstFullDPSPass(build, OutputModeCalculator, i)

				where := got.Player.MainSkill.ActiveEffect.GrantedEffect.Name()
				if !got.skipEHP {
					t.Errorf("%s: the per-skill pass ran buildDefenceEstimations; Calcs.lua:207 passes skipEHP", where)
				}
				compareFullDPSKeys(t, where, got.Player.Output, want.Player.Output)
				if (got.Minion == nil) != (want.Minion == nil) {
					t.Errorf("%s: minion presence differs", where)
				} else if got.Minion != nil {
					minions++
					compareFullDPSKeys(t, where+" (minion)", got.Minion.Output, want.Minion.Output)
				}
				if ms, ws := got.Player.MainSkill, want.Player.MainSkill; (ms.Mirage == nil) != (ws.Mirage == nil) {
					t.Errorf("%s: mirage presence differs", where)
				} else if ms.Mirage != nil {
					mirages++
					compareFullDPSKeys(t, where+" (mirage)", ms.Mirage.Output, ws.Mirage.Output)
				}
				if got.Player.MainSkill.SkillData != nil && got.Player.MainSkill.SkillData.Triggered {
					triggered++
				}
				if got.Player.MainSkill.SkillFlags[SkillFlagTotem] {
					totems++
				}
				compared++
			})
			if compared < 2 {
				t.Fatalf("CONTROL IS VACUOUS: only %d FullDPS passes compared", compared)
			}
			t.Logf("%d passes compared (%d minion, %d triggered, %d totem, %d mirage)", compared, minions, triggered, totems, mirages)
		})
	}
}

// TestAcceleratedEnvKeepsSpecNodeCounts: A reused env reads the spec's node counts.
func TestAcceleratedEnvKeepsSpecNodeCounts(t *testing.T) {
	initBenchData(t)
	build := loadHeavyBuild(t, "0280-elementalist-fireball.xml")
	base, _, _, _ := initEnvWith(build, NewEnvironmentCache(), OutputModeCalculator, false, nil)
	var notable int64
	for id, node := range base.AllocatedNodes {
		if data.IsTreeNotable(node) {
			notable, _ = strconv.ParseInt(id, 10, 64)
			break
		}
	}
	if notable == 0 {
		t.Fatal("REFUSING: the build allocates no notable")
	}
	allocated := func(env *Environment) float64 {
		return env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:AllocatedNotable")
	}
	specCount := allocated(base)

	cache := NewEnvironmentCache()
	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(build, cache, OutputModeCalculator, false, nil)
	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(build, cache, OutputModeCalculator, false,
		&CalcOverride{RemoveNodes: []int64{notable}},
		&calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	if got := allocated(env); got != specCount-1 {
		t.Fatalf("CONTROL: the first initEnv counts %v notables, want the override's %v", got, specCount-1)
	}
	env = reinitFullDPSEnv(env, cache, OutputModeCalculator, cachedPlayerDB, cachedEnemyDB, cachedMinionDB)
	if got := allocated(env); got != specCount {
		t.Errorf("the reused env counts %v notables, want the spec's %v", got, specCount)
	}
}

// TestFullDPSReinitKeepsEnergyBladeCondition: initEnv's Energy Blade restart appends to the override table itself.
func TestFullDPSReinitKeepsEnergyBladeCondition(t *testing.T) {
	initBenchData(t)
	build := loadHeavyBuild(t, "1252-juggernaut-bladefall-of-trarthus-2.xml")
	cache := NewEnvironmentCache()
	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(build, cache, OutputModeCalculator, false, nil)
	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(build, cache, OutputModeCalculator, false, nil,
		&calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
		t.Fatalf("CONTROL: the first initEnv did not set %s", conditionAffectedByEnergyBlade)
	}
	if env.ModDB.Parent != cachedPlayerDB {
		t.Errorf("the restarted initEnv dropped specEnv: its modDB is not parent-linked to the cached DB (CalcSetup.lua:1819)")
	}
	env = reinitFullDPSEnv(env, cache, OutputModeCalculator, cachedPlayerDB, cachedEnemyDB, cachedMinionDB)
	if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
		t.Errorf("the reused env lost %s", conditionAffectedByEnergyBlade)
	}
}

// TestCalculatorPassHasNoBreakdown: CALCULATOR passes (calcFullDPS, the what-if) build no breakdown (CalcPerform.lua:1479-1487).
func TestCalculatorPassHasNoBreakdown(t *testing.T) {
	initBenchData(t)
	build := loadHeavyBuild(t, "1591-necromancer-summon-holy-relic-of-conviction-15.xml")
	main := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if main.Player.Breakdown == nil {
		t.Fatal("CONTROL: the MAIN env has no breakdown; the UI reads it")
	}
	passes := 0
	fullDPSPasses(build, OutputModeCalculator, func(_ int, env *Environment) {
		passes++
		if env.Player.Breakdown != nil || (env.Minion != nil && env.Minion.Breakdown != nil) {
			t.Errorf("pass %d (%s) built a breakdown", passes, env.Player.MainSkill.ActiveEffect.GrantedEffect.Name())
		}
	})
	if passes == 0 {
		t.Fatal("CONTROL IS VACUOUS: no FullDPS pass")
	}
}
