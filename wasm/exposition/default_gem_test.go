package exposition

import (
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestDefaultGemSkillIDIsGrantedEffect pins the skillId PoB's loader matches against
// (SkillsTab.lua:472-479): a gem saved with any other skillId loads without gemData in PoB.
func TestDefaultGemSkillIDIsGrantedEffect(t *testing.T) {
	if len(raw.Gems) == 0 {
		t.Fatal("raw.Gems is empty")
	}
	bad := 0
	for id, row := range raw.Gems {
		if got := DefaultGem(id, row.Name, id, 20, 0).SkillID; got != row.GrantedEffectID {
			if bad < 5 {
				t.Errorf("DefaultGem(%q).SkillID = %q, PoB's grantedEffectId is %q", id, got, row.GrantedEffectID)
			}
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d gems carry a skillId PoB cannot match", bad, len(raw.Gems))
	}
}
