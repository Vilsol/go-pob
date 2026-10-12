package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestTraumaStacksSeededFromSustainable pins CalcOffence.lua:2329-2331.
func TestTraumaStacksSeededFromSustainable(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true — this reads the graded corpus")
	}
	requireGameData(t, "3.29")

	stacks := func(name string) (float64, float64, bool) {
		t.Helper()
		file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name+".xml"))
		if err != nil {
			t.Fatalf("REFUSING: corpus xml missing: %s: %v", name, err)
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("REFUSING: build unparseable: %s: %v", name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env.Player == nil || env.Player.MainSkill == nil || env.Player.MainSkill.SkillModList == nil {
			t.Fatalf("REFUSING: no main skill for %s", name)
		}
		list := env.Player.MainSkill.SkillModList
		cfg := env.Player.MainSkill.SkillCfg
		return list.Sum(mod.TypeBase, cfg, "Multiplier:TraumaStacks"),
			list.Sum(mod.TypeBase, cfg, "Multiplier:SustainableTraumaStacks"),
			list.Flag(nil, "HasTrauma")
	}

	const noTrauma = "0113-chieftain-ice-crash"
	if _, _, has := stacks(noTrauma); has {
		t.Fatalf("REFUSING: control %s carries HasTrauma; it cannot separate the zero arm", noTrauma)
	}

	const build = "2507-ladder-Lanlin_0179.AVL_TraumaRex"
	trauma, sustainable, has := stacks(build)
	if !has {
		t.Fatalf("REFUSING: %s does not carry HasTrauma; the fixture no longer exercises the copy", build)
	}
	if sustainable <= 0 {
		t.Fatalf("REFUSING: %s solved SustainableTraumaStacks = %v; the SOLVER is broken, not the copy",
			build, sustainable)
	}
	if math.Abs(trauma-sustainable) > 1e-9 {
		t.Errorf("Multiplier:TraumaStacks = %v, want the sustainable count %v (CalcOffence.lua:2329-2331)",
			trauma, sustainable)
	}
}
