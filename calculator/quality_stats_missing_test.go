package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
)

func findPartialQualityGem(t *testing.T) (*poe.GrantedEffect, int, int) {
	t.Helper()

	for _, effect := range poe.GrantedEffects {
		qualities := effect.GetEffectQualityStats()
		if len(qualities) == 0 || len(qualities) >= 4 {
			continue
		}

		present, missing := -1, -1
		for id := range 4 {
			if qualities[id] != nil && len(qualities[id].GetStats()) > 0 {
				present = id
			} else if qualities[id] == nil {
				missing = id
			}
		}
		if present >= 0 && missing >= 0 {
			return effect, present, missing
		}
	}

	t.Fatal("no granted effect with a partial quality-stat set in the data")
	return nil, 0, 0
}

var qualityIDNames = map[int]string{0: "Default", 1: "Alternate1", 2: "Alternate2", 3: "Alternate3"}

// TestBuildSkillInstanceStatsMissingAlternateQuality pins CalcBuildSkillInstanceStats against a gem set to an alternate quality it has no stat set for.
func TestBuildSkillInstanceStatsMissingAlternateQuality(t *testing.T) {
	rawEffect, present, missing := findPartialQualityGem(t)
	effect := &GrantedEffect{Raw: rawEffect}

	withPresent := CalcBuildSkillInstanceStats(&GemEffect{
		GrantedEffect: effect,
		Level:         1,
		Quality:       20,
		QualityID:     qualityIDNames[present],
	}, effect)

	withMissing := CalcBuildSkillInstanceStats(&GemEffect{
		GrantedEffect: effect,
		Level:         1,
		Quality:       20,
		QualityID:     qualityIDNames[missing],
	}, effect)

	unqualified := CalcBuildSkillInstanceStats(&GemEffect{
		GrantedEffect: effect,
		Level:         1,
		Quality:       0,
		QualityID:     qualityIDNames[present],
	}, effect)

	if len(withPresent) == 0 {
		t.Fatalf("%s: quality id %d produced no stats", rawEffect.ID, present)
	}
	differs := false
	for k, v := range withPresent {
		if unqualified[k] != v {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("%s: quality id %d contributed nothing", rawEffect.ID, present)
	}

	if len(withMissing) != len(unqualified) {
		t.Fatalf("%s: missing quality id %d produced %d stats, want %d", rawEffect.ID, missing, len(withMissing), len(unqualified))
	}
	for k, v := range unqualified {
		if withMissing[k] != v {
			t.Fatalf("%s: missing quality id %d changed %s: %v != %v", rawEffect.ID, missing, k, withMissing[k], v)
		}
	}
}
