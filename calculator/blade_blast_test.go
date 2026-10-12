package calculator

import (
	"os"
	"testing"
)

// TestBladeBlastDpsBaseMultiplier pins act_dex.lua:1374-1379, a preDamageFunc this tree did not register, and the SkillData field it needs.
//
//	preDamageFunc = function(activeSkill, output)
//	    if activeSkill.skillPart == 2 then
//	    end
//	end
func TestBladeBlastDpsBaseMultiplier(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)

	var bladeBlast, withMultiplier, withDps int

	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil ||
			ms.ActiveEffect.GrantedEffect.Raw == nil || ms.SkillData == nil {
			continue
		}
		id := ms.ActiveEffect.GrantedEffect.Raw.ID
		if id != "BladeBlast" && id != "BladeBlastAltX" {
			continue
		}
		bladeBlast++
		if ms.SkillData.DpsBaseMultiplier > 0 {
			withMultiplier++
		}
		if ms.SkillData.DpsMultiplier > 0 {
			withDps++
		}
		if bladeBlast >= 40 {
			break
		}
	}

	if bladeBlast == 0 {
		t.Fatal("REFUSING: no graded build runs BladeBlast or BladeBlastAltX as its main skill, " +
			"so neither the field nor the callback decides anything here")
	}
	if withMultiplier == 0 {
		t.Errorf("across %d Blade Blast builds, NOT ONE carries a nonzero "+
			"SkillData.DpsBaseMultiplier — the re-emitted skill_base_mods rows are not reaching "+
			"SkillData, which is the state this fix closed", bladeBlast)
	}
	if withDps == 0 {
		t.Errorf("across %d Blade Blast builds, NOT ONE has a nonzero SkillData.DpsMultiplier — "+
			"the preDamageFunc is registered but is not running", bladeBlast)
	}
	t.Logf("MEASURED: %d Blade Blast builds, %d with DpsBaseMultiplier live, %d with DpsMultiplier set",
		bladeBlast, withMultiplier, withDps)
}
