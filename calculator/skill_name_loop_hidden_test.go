package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	raw "github.com/Vilsol/go-pob/data/raw"
)

// TestSkillNameLoopHasNoHiddenGems is the retiring condition for the one conjunct of ModParser.lua:6044 that initializeSkillNameList does not test.
func TestSkillNameLoopHasNoHiddenGems(t *testing.T) {
	requireGameData(t, "3.29")

	reached := 0
	var hidden []string

	entries := make([]raw.PoBGem, 0, len(poe.SkillGems)+len(raw.GemVariants))
	for _, gemData := range poe.SkillGems {
		if pobGemEntry, inPoB := raw.Gems[gemBaseID(gemData)]; inPoB {
			entries = append(entries, pobGemEntry)
		}
	}
	for _, variant := range raw.GemVariants {
		entries = append(entries, variant)
	}
	for _, pobGemEntry := range entries {
		grantedEffect := poe.GrantedEffectByID(pobGemEntry.GrantedEffectID)
		if grantedEffect == nil || grantedEffect.IsSupport || grantedEffect.GetActiveSkill() == nil {
			continue
		}
		reached++
		if raw.PoBSkills[grantedEffect.ID].Hidden {
			hidden = append(hidden, grantedEffect.ID)
		}
	}

	if reached == 0 {
		t.Fatal("REFUSING: no gem reaches initializeSkillNameList's body, so this test measures " +
			"nothing. Either the loop's guards changed or the gem data did.")
	}
	if len(hidden) != 0 {
		t.Errorf("%d of %d gems reaching initializeSkillNameList are grantedEffect.hidden: %v\n"+
			"ModParser.lua:6044 SKIPS those and go-pob does not, so each one now contributes an "+
			"infix and four-plus prefix patterns PoB never builds. Port the hidden conjunct.",
			len(hidden), reached, hidden)
	}
	t.Logf("gems reaching the loop: %d, of which hidden: %d", reached, len(hidden))
}
