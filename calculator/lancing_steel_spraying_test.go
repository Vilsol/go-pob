package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestLancingSteelOfSprayingScalesByProjectileCount pins act_dex.lua:10334-10341.
func TestLancingSteelOfSprayingScalesByProjectileCount(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	type want struct {
		name string
		part int
	}
	cases := []want{
		{"1128-assassin-lancing-steel-of-spraying-6.xml", 2},
		{"3221-ladder-Rigbad_2058.WhiteCheekedGibbon.xml", 1},
	}
	checked := 0
	for _, c := range cases {
		raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.name))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		var ls *ActiveSkill
		for _, sk := range env.Player.ActiveSkillList {
			if sk.ActiveEffect != nil && sk.ActiveEffect.GrantedEffect != nil &&
				sk.ActiveEffect.GrantedEffect.ID() == "LancingSteelAltX" {
				ls = sk
			}
		}
		if ls == nil {
			t.Fatalf("REFUSING: %s no longer carries LancingSteelAltX; the fixture has moved and "+
				"this test is measuring nothing", c.name)
		}
		gi, si := ls.SocketGroupIndex, activeSkillIndexInGroup(env, ls)
		sub, _, _, _ := InitEnv(env.Build.WithMainSkill(gi, si), env.Cache, env.Mode)
		PerformCalc(sub)
		s := sub.Player.MainSkill
		if s.SkillPart != c.part {
			t.Fatalf("REFUSING: %s resolved LancingSteelAltX at skill part %d, expected %d",
				c.name, s.SkillPart, c.part)
		}
		pc := sub.Player.Output["ProjectileCount"]
		if pc < 2 {
			t.Fatalf("REFUSING: %s has ProjectileCount %v; the (n-1)/n term is degenerate and the "+
				"fixture cannot distinguish a right port from a wrong one", c.name, pc)
		}
		checked++

		got := s.SkillData.DpsMultiplier
		ours := 0
		for _, r := range s.SkillModList.Tabulate(mod.TypeMore, s.SkillCfg, "Damage") {
			if r.Mod.GetSource() == mod.Source("Skill:LancingSteelAltX") {
				ours++
			}
		}
		if c.part != 2 {
			if math.Abs(got-pc) <= 1e-9 || ours != 0 {
				t.Errorf("%s (part 1): dpsMultiplier %v equals ProjectileCount %v, or %d "+
					"Skill:LancingSteelAltX MORE mods landed — act_dex.lua:10335 gates the whole "+
					"body on skill part 2", c.name, got, pc, ours)
			}
			continue
		}
		if math.Abs(got-pc) > 1e-9 {
			t.Errorf("%s: dpsMultiplier %v, want ProjectileCount %v (act_dex.lua:10338)",
				c.name, got, pc)
		}
		if ours != 1 {
			t.Errorf("%s: %d Damage MORE mods sourced Skill:LancingSteelAltX, want 1. "+
				"act_dex.lua:10337-10339 adds (LancingSteelSubsequentDamage-1)*100*(n-1)/n; a "+
				"port that sets only the multiplier passes every other check and still loses the "+
				"subsequent-hit bonus", c.name, ours)
		}
	}
	if checked != len(cases) {
		t.Fatalf("REFUSING: checked %d of %d fixtures", checked, len(cases))
	}
}
