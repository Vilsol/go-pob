package calculator

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

var fullDPSShareBuilds = append([]string{
	"0001-necromancer-unknown.xml",
	"0280-elementalist-fireball.xml",
	"15355-necromancer-unknown.xml",
	"1575-necromancer-holy-flame-totem.xml",
}, fullDPSReuseBuilds...)

func forgetFullDPSReuse(env *Environment) {
	env.nodeModList = nil
	for _, activeSkill := range env.Player.ActiveSkillList {
		activeSkill.reusableMinionSkills, activeSkill.reusableMinionMain = nil, nil
	}
}

type fullDPSPassSnapshot struct {
	skill string
	outs  map[string]map[string]float64
}

func fullDPSPassSnapshots(build *pob.PathOfBuilding, beforeReinit func(*Environment)) []fullDPSPassSnapshot {
	cache := NewEnvironmentCache()
	mode := OutputModeCalculator
	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvWith(build, cache, mode, false, nil)
	fullEnv, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := initEnvSpec(build, cache, mode, false, nil,
		&calcSpecEnv{cachedPlayerDB: cachedPlayerDB, cachedEnemyDB: cachedEnemyDB, cachedMinionDB: cachedMinionDB})
	var snaps []fullDPSPassSnapshot
	for _, activeSkill := range fullEnv.Player.ActiveSkillList {
		group, ok := activeSkill.SocketGroup.(*pob.Skill)
		if !ok || group == nil || group.IncludeInFullDPS == nil || !*group.IncludeInFullDPS {
			continue
		}
		if _, enabled := getActiveSkillCount(activeSkill); !enabled {
			continue
		}
		fullEnv.Player.MainSkill = activeSkill
		performCalc(fullEnv, true)
		snaps = append(snaps, snapshotFullDPSPass(fullEnv))
		if beforeReinit != nil {
			beforeReinit(fullEnv)
		}
		fullEnv = reinitFullDPSEnv(fullEnv, cache, mode, cachedPlayerDB, cachedEnemyDB, cachedMinionDB)
	}
	return snaps
}

func snapshotFullDPSPass(env *Environment) fullDPSPassSnapshot {
	snap := fullDPSPassSnapshot{
		skill: env.Player.MainSkill.ActiveEffect.GrantedEffect.Name(),
		outs:  map[string]map[string]float64{},
	}
	add := func(name string, out map[string]float64) {
		if out != nil {
			snap.outs[name] = out
		}
	}
	add("player", env.Player.Output)
	add("enemy", env.Enemy.Output)
	for table, out := range env.Player.OutputTable {
		add(fmt.Sprintf("player.%v", table), out)
	}
	if env.Minion != nil {
		add("minion", env.Minion.Output)
	}
	if mirage := env.Player.MainSkill.Mirage; mirage != nil {
		add("mirage", mirage.Output)
	}
	return snap
}

func diffFullDPSPasses(got, want []fullDPSPassSnapshot) []string {
	var diffs []string
	if len(got) != len(want) {
		return []string{fmt.Sprintf("pass count: shared %d, fresh %d", len(got), len(want))}
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.skill != w.skill {
			diffs = append(diffs, fmt.Sprintf("pass %d: skill %q vs %q", i, g.skill, w.skill))
			continue
		}
		actors := make([]string, 0, len(w.outs))
		for actor := range w.outs {
			actors = append(actors, actor)
		}
		for actor := range g.outs {
			if _, ok := w.outs[actor]; !ok {
				actors = append(actors, actor)
			}
		}
		slices.Sort(actors)
		for _, actor := range slices.Compact(actors) {
			gOut, wOut := g.outs[actor], w.outs[actor]
			for k, wv := range wOut {
				gv, ok := gOut[k]
				if !ok || math.Float64bits(gv) != math.Float64bits(wv) {
					diffs = append(diffs, fmt.Sprintf("pass %d %s %s.%s: shared %v (set %v), fresh %v", i, g.skill, actor, k, gv, ok, wv))
				}
			}
			for k, gv := range gOut {
				if _, ok := wOut[k]; !ok {
					diffs = append(diffs, fmt.Sprintf("pass %d %s %s.%s: shared %v, fresh unset", i, g.skill, actor, k, gv))
				}
			}
		}
	}
	return diffs
}

// TestFullDPSSharedPassesMatchFresh: Every Full DPS pass that reuses work from the previous pass must equal.
func TestFullDPSSharedPassesMatchFresh(t *testing.T) {
	initBenchData(t)
	names := fullDPSShareBuilds
	parallel := false
	if list := os.Getenv("FULLDPS_SHARE_LIST"); list != "" {
		raw, err := os.ReadFile(list)
		if err != nil {
			t.Fatal(err)
		}
		names = strings.Fields(string(raw))
		parallel = true
	}
	passes := 0
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if parallel {
				t.Parallel()
			}
			build := loadHeavyBuild(t, name)
			shared := fullDPSPassSnapshots(build, nil)
			fresh := fullDPSPassSnapshots(build, forgetFullDPSReuse)
			diffs := diffFullDPSPasses(shared, fresh)
			t.Logf("%d passes compared", len(shared))
			for i, d := range diffs {
				if i == 20 {
					t.Errorf("... %d more", len(diffs)-i)
					break
				}
				t.Error(d)
			}
			if !parallel {
				passes += len(shared)
			}
		})
	}
	if !parallel && passes < 2*len(names) {
		t.Fatalf("CONTROL IS VACUOUS: %d passes over %d builds", passes, len(names))
	}
}

// TestAllyLifeNeedsARedirect: CalcPerform.lua:3596-3599: without a redirect to an ally's Life.
func TestAllyLifeNeedsARedirect(t *testing.T) {
	initBenchData(t)
	build := loadHeavyBuild(t, "0280-elementalist-fireball.xml")
	env, _, _, _ := initEnvWith(build, NewEnvironmentCache(), OutputModeCalculator, false, nil)
	PerformCalc(env)
	if env.ModDB.Sum(mod.TypeBase, nil, allyLifeRedirects...) != 0 {
		t.Fatal("REFUSING: the build redirects damage to an ally's Life")
	}
	allies := 0
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.Minion == nil || activeSkill.Minion == env.Minion || activeSkill.SkillFlags[SkillFlagDisable] {
			continue
		}
		allies++
		if life, ok := activeSkill.Minion.Output["Life"]; ok {
			t.Errorf("%s: ally minion Life computed (%v) with no redirect", activeSkill.ActiveEffect.GrantedEffect.Name(), life)
		}
	}
	if allies == 0 {
		t.Fatal("CONTROL IS VACUOUS: no ally minion")
	}
}
