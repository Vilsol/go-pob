package calculator

import (
	"math"
	"testing"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

func phantasmBuffMod(env *Environment, name string) mod.Mod {
	for _, skill := range env.AuxSkillList {
		for _, buff := range skill.BuffList {
			for _, m := range buff.ModList.Mods() {
				if m.Name() == name && m.GetSource() == "Skill:SupportSummonPhantasm" {
					return m
				}
			}
		}
	}
	return nil
}

// TestBuffMergeCopiesSharedDestMod: a global-effect skill mod merging into a buff mod copies it first.
func TestBuffMergeCopiesSharedDestMod(t *testing.T) {
	requireGameData(t, "3.29")
	const build = "0221-reliquarian-ethereal-knives.xml"
	template := phantasmBuffMod(corpusEnvForLoadedData(t, build), "PhysicalMin")
	if template == nil {
		t.Fatalf("REFUSING: %s has no Phantasm buff PhysicalMin", build)
	}

	support, active := template.Clone().SetName("CowProbe"), template.Clone().SetName("CowProbe")
	support.Value().SetFloat(3)
	active.Value().SetFloat(5)
	saved := map[string][]mod.Mod{}
	for id, m := range map[string]mod.Mod{"SupportSummonPhantasm": support, "TriggeredSummonGhostOnKill": active} {
		saved[id] = raw2.SkillBaseMods[id]
		raw2.SkillBaseMods[id] = append(append([]mod.Mod{}, saved[id]...), m)
	}
	t.Cleanup(func() {
		for id, ms := range saved {
			raw2.SkillBaseMods[id] = ms
		}
	})

	for run := 1; run <= 2; run++ {
		merged := phantasmBuffMod(corpusEnvForLoadedData(t, build), "CowProbe")
		if merged == nil {
			t.Fatalf("run %d: the probe did not reach the Phantasm buff", run)
		}
		if got := merged.Value().Float(); got != 8 {
			t.Errorf("run %d: merged CowProbe = %v, want 8", run, got)
		}
	}
	if got := support.Value().Float(); got != 3 {
		t.Errorf("the shared SkillBaseMods mod was mutated to %v, want 3", got)
	}
}

// TestSkillInstanceModsSurviveACalculation: the cached stat-map mods (skillInstanceMods) are shared between calculations.
func TestSkillInstanceModsSurviveACalculation(t *testing.T) {
	requireGameData(t, "3.29")
	const build = "0221-reliquarian-ethereal-knives.xml"
	first := corpusEnvForLoadedData(t, build).Player.Output
	second := corpusEnvForLoadedData(t, build).Player.Output
	if phantasmBuffMod(corpusEnvForLoadedData(t, build), "PhysicalMin") == nil {
		t.Fatalf("REFUSING: %s has no Phantasm buff PhysicalMin", build)
	}
	n := 0
	for k, v := range first {
		if w := second[k]; math.Float64bits(v) != math.Float64bits(w) {
			n++
			if n <= 5 {
				t.Errorf("%s: %v, then %v", k, v, w)
			}
		}
	}
}
