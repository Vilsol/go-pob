package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestDualWieldSkillCfgUsesMainHandFlagsOnly pins CalcActiveSkill.lua:455.
func TestDualWieldSkillCfgUsesMainHandFlagsOnly(t *testing.T) {
	requireGameData(t, "3.29")
	const build = "0016-gladiator-dual-strike-of-ambidexterity.xml"

	corpus := corpusDirForLoadedData(t)
	xml, err := os.ReadFile(filepath.Join(corpus, build))
	if err != nil {
		t.Fatalf("REFUSING: cannot read fixture %s: %v", build, err)
	}
	b, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatalf("REFUSING: cannot parse fixture %s: %v", build, err)
	}
	env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
	if env == nil || env.Player == nil || env.Player.MainSkill == nil {
		t.Fatalf("REFUSING: fixture %s built no main skill", build)
	}
	as := env.Player.MainSkill

	w1, w2 := as.Weapon1Flags, as.Weapon2Flags
	offHandOnly := w2 &^ w1
	if w1 == 0 || w2 == 0 || offHandOnly == 0 {
		t.Fatalf("REFUSING: %s is no longer a dual wield with DIFFERENT weapon types "+
			"(weapon1Flags 0x%x, weapon2Flags 0x%x) — the assertion below cannot fail on such a "+
			"build, so a pass would say nothing. Pick another fixture.", build, w1, w2)
	}
	if as.SkillCfg == nil {
		t.Fatalf("REFUSING: %s has no SkillCfg", build)
	}

	got := as.SkillCfg.Flags.Get()
	if got&offHandOnly != 0 {
		t.Errorf("skillCfg carries OFF-HAND-only weapon flags 0x%x (cfg 0x%x, weapon1 0x%x, "+
			"weapon2 0x%x); CalcActiveSkill.lua:455 takes weapon1Flags alone when both are set",
			got&offHandOnly, got, w1, w2)
	}
	if got&w1 != w1 {
		t.Errorf("skillCfg 0x%x is missing MAIN-HAND flags 0x%x", got, w1)
	}
	if as.Weapon2Cfg != nil && as.Weapon2Cfg.Flags.Get()&w2 != w2 {
		t.Errorf("weapon2Cfg 0x%x lost its own hand's flags 0x%x", as.Weapon2Cfg.Flags.Get(), w2)
	}
	if as.Weapon1Cfg != nil && as.Weapon1Cfg.Flags.Get()&offHandOnly != 0 {
		t.Errorf("weapon1Cfg 0x%x carries off-hand-only flags 0x%x",
			as.Weapon1Cfg.Flags.Get(), offHandOnly)
	}
}
