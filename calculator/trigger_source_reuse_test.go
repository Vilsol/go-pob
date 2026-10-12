package calculator

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

var triggerReuseBuilds = []string{
	"1591-necromancer-summon-holy-relic-of-conviction-15.xml",
	"0028-occultist-forbidden-rite-of-soul-sacrifice.xml",
	"0136-deadeye-elemental-hit-of-the-spectrum-2.xml",
	"0105-assassin-spellslinger.xml",
	"0001-necromancer-unknown.xml",
	"0454-occultist-flame-dash.xml",
}

// TestTriggerSourceReuseMatchesFresh: a what-if whose trigger sources are built on re-initialised envs must equal the same what-if built on a fresh calcs.initEnv per source.
func TestTriggerSourceReuseMatchesFresh(t *testing.T) {
	initBenchData(t)
	files := triggerReuseBuilds
	if list := os.Getenv("TRIGGER_REUSE_LIST"); list != "" {
		raw, err := os.ReadFile(list)
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		files = strings.Fields(string(raw))
	}

	reused := 0
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			build := loadCorpusBuild(t, file)
			override := &CalcOverride{RemoveNodes: build.Build.PassiveNodes[len(build.Build.PassiveNodes)-1:]}
			got := GetMiscCalculator(build).Calc(override)
			reuseTriggerSourceEnvs = false
			want := GetMiscCalculator(loadCorpusBuild(t, file)).Calc(override)
			reuseTriggerSourceEnvs = true

			if len(got.triggerCache.entries) != len(want.triggerCache.entries) {
				t.Fatalf("%d sources reused, %d fresh", len(got.triggerCache.entries), len(want.triggerCache.entries))
			}
			if len(got.triggerCache.entries) > 1 {
				reused += len(got.triggerCache.entries) - 1
			}
			for uuid, fresh := range want.triggerCache.entries {
				if !reflect.DeepEqual(got.triggerCache.entries[uuid], fresh) {
					t.Errorf("%s: reused %+v, fresh %+v", uuid, got.triggerCache.entries[uuid], fresh)
				}
			}
			if where, ok := outputsAgree(got.Player.Output, want.Player.Output); !ok {
				t.Errorf("player output: %s", where)
			}
		})
	}
	if reused == 0 {
		t.Fatal("no calculation built a second trigger source, so nothing was reused")
	}
}

// TestTriggerSourceReuseStaysInOneWhatIf: a what-if must equal the same what-if on a new calculator without reuse, whatever override ran before it.
func TestTriggerSourceReuseStaysInOneWhatIf(t *testing.T) {
	initBenchData(t)
	for _, file := range triggerReuseBuilds[:3] {
		t.Run(file, func(t *testing.T) {
			build := loadCorpusBuild(t, file)
			cases := whatIfCases(t, build, 3)
			if len(cases) < 2 {
				t.Fatalf("REFUSING: %d what-if cases", len(cases))
			}
			misc := GetMiscCalculator(build)
			for i, c := range cases {
				prev := cases[(i+1)%len(cases)]
				misc.Calc(&CalcOverride{AddNodes: prev.add, RemoveNodes: prev.remove})
				got := misc.Calc(&CalcOverride{AddNodes: c.add, RemoveNodes: c.remove})
				reuseTriggerSourceEnvs = false
				want := GetMiscCalculator(loadCorpusBuild(t, file)).Calc(&CalcOverride{AddNodes: c.add, RemoveNodes: c.remove})
				reuseTriggerSourceEnvs = true
				if where, ok := outputsAgree(got.Player.Output, want.Player.Output); !ok {
					t.Errorf("%s after %s: %s", c.name, prev.name, where)
				}
			}
		})
	}
}

// TestTriggerSourceIsTheSkillItNames: each trigger-cache entry must be the skill its uuid names.
func TestTriggerSourceIsTheSkillItNames(t *testing.T) {
	initBenchData(t)
	build := loadCorpusBuild(t, "0380-pathfinder-toxic-rain-13.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	checked := 0
	for i, skill := range env.Player.ActiveSkillList {
		cached, ok := env.triggerCache.entries[cacheSkillUUID(env, skill)]
		if !ok {
			continue
		}
		fresh, _, _, _ := InitEnvOverride(build, env.Cache, env.Mode, nil)
		fresh.Player.MainSkill = fresh.Player.ActiveSkillList[i]
		performCalc(fresh, true)
		if cached.ManaCost != fresh.Player.Output["ManaCost"] || cached.Speed != fresh.Player.Output["Speed"] {
			t.Errorf("%s: cached ManaCost %v Speed %v, the skill's own %v / %v", cacheSkillUUID(env, skill),
				cached.ManaCost, cached.Speed, fresh.Player.Output["ManaCost"], fresh.Player.Output["Speed"])
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("REFUSING: no trigger-cache entry to check")
	}
}
