package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCurseKeywordFollowsAppliesCurse pins CalcActiveSkill.lua:379-381: KeywordFlag.Curse comes from SkillType.AppliesCurse, not from Hex or Mark.
func TestCurseKeywordFollowsAppliesCurse(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for build, wantCurse := range map[string]bool{
		"0373-occultist-bane-of-condemnation.xml": false,
		"0792-luminary-snipers-mark.xml":          true,
	} {
		env := corpusEnvForLoadedData(t, build)
		cfg := env.Player.MainSkill.SkillCfg
		got := cfg.KeywordFlags != nil && *cfg.KeywordFlags&mod.KeywordFlagCurse != 0
		if got != wantCurse {
			t.Errorf("%s (%s): KeywordFlag.Curse = %v, want %v", build, env.Player.MainSkill.ActiveEffect.GrantedEffect.Name(), got, wantCurse)
		}
	}
	env := corpusEnvForLoadedData(t, "0373-occultist-bane-of-condemnation.xml")
	if got := env.Player.Output["Speed"]; got < 1.5625*0.995 || got > 1.5625*1.005 {
		t.Errorf("0373 Speed = %v, PoB's oracle has 1.5625", got)
	}
}
