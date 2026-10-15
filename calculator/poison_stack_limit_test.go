package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestPoisonStackLimitCapsStacks pins CalcOffence.lua:4602 and :4610-4618 on a corpus poisoner given ModParser.lua:3938's line as a custom mod.
func TestPoisonStackLimitCapsStacks(t *testing.T) {
	const name = "0059-occultist-viper-strike.xml"
	base := (&Calculator{PoB: loadCorpusBuild(t, name)}).BuildOutput(OutputModeMain)
	uncapped := base.Player.Output["PoisonStacks"]
	if math.Abs(uncapped-118.65335625) > 1e-6 {
		t.Fatalf("REFUSING: uncapped PoisonStacks %v moved off PoB's 118.65335625", uncapped)
	}

	build := loadCorpusBuild(t, name)
	title, enabled := "Probe", "true"
	build.SetCustomModifierBlocks(append(build.GetCustomModifierBlocks(), pob.CustomModifierBlock{
		Title: &title, Enabled: &enabled, Text: "Cannot Poison Enemies with at least 10 Poisons on them",
	}))
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	ms := env.Player.MainSkill
	if limit, ok := ms.SkillModList.Min(ms.SkillCfg, "PoisonStackLimit"); !ok || limit != 10 {
		t.Fatalf("REFUSING: PoisonStackLimit MIN = %v, want 10", limit)
	}
	additional := 1 + min(ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "AdditionalPoisonChance")/100, 1) +
		ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "AdditionalPoisonStacks")
	want := min(uncapped, math.Ceil(10/additional)*additional)
	if got := env.Player.Output["PoisonStacks"]; math.Abs(got-want) > 1e-9 {
		t.Errorf("PoisonStacks = %v, want %v (m_ceil(limit / additional) * additional)", got, want)
	}
}
