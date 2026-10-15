package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMultiStageSkillsDefaultToMaximumStages pins CalcActiveSkill.lua:636-644.
//
//	activeSkill.activeStageCount = m_max(
//	    1 + skillModList:Sum("BASE", cfg, "Multiplier:"..name.."MinimumStage"))
func TestMultiStageSkillsDefaultToMaximumStages(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var decisive, unset, withStages int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil ||
			ms.SkillModList == nil || ms.ActiveEffect.SrcInstance == nil {
			continue
		}
		stageName := stageMultiplierName(ms.ActiveEffect.GrantedEffect)
		if stageName == "" {
			continue
		}
		//pob:src/Modules/CalcActiveSkill.lua:645-651
		if !ms.SkillFlags[SkillFlagMultiStage] {
			continue
		}
		limit := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:"+stageName+"MaxStages")
		if limit <= 0 {
			continue
		}
		decisive++

		saved := ms.ActiveEffect.SrcInstance.SkillStageCount
		if saved != 0 {
			continue
		}
		unset++
		if ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:"+stageName+"Stage") > 0 {
			withStages++
		}
		if unset >= 100 {
			break
		}
	}

	if decisive == 0 {
		t.Fatal("REFUSING: no graded build runs a multi-stage main skill (a live " +
			"Multiplier:<Name>MaxStages), so this test measures nothing")
	}
	if unset == 0 {
		t.Fatalf("REFUSING: all %d multi-stage builds saved an explicit skillStageCount, so the "+
			"fallback this test exists for is never taken", decisive)
	}
	if withStages != unset {
		t.Errorf("only %d of %d multi-stage builds WITHOUT a saved skillStageCount got a nonzero "+
			"Multiplier:<Name>Stage; PoB defaults all of them to stagesMax", withStages, unset)
	}
}
