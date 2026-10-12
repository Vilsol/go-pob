package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHeraldOfTheBreachHitTimeInputsArrive: other.lua:2130-2132 is the eighteenth member of the `hitTimeOverride = repeatFrequency / (1 + Sum("INC", cfg, <name>) / 100)` family.
func TestHeraldOfTheBreachHitTimeInputsArrive(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	dir, names := corpusNamesForLoadedData(t)

	const sample = 60
	var carriers, active, withRepeatFrequency, asMain int
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		if !strings.Contains(string(raw), `skillId="HeraldOfTheBreach"`) {
			continue
		}
		carriers++
		if carriers > sample {
			break
		}
		env := corpusEnvForLoadedData(t, name)
		for _, as := range env.Player.ActiveSkillList {
			if as == nil || as.ActiveEffect == nil || as.ActiveEffect.GrantedEffect == nil ||
				as.ActiveEffect.GrantedEffect.ID() != "HeraldOfTheBreach" {
				continue
			}
			active++
			if as.SkillData.RepeatFrequency != 0 {
				withRepeatFrequency++
			}
			if env.Player.MainSkill == as {
				asMain++
			}
			break
		}
	}

	if active == 0 {
		t.Fatalf("REFUSING: none of %d Herald of the Breach carriers builds the skill at all, so "+
			"this test measures nothing", carriers)
	}
	if withRepeatFrequency != active {
		t.Errorf("only %d of %d built Herald of the Breach skills carry a nonzero "+
			"SkillData.RepeatFrequency; data/raw/skill_stat_maps.go:2699-2702 emits it for all of "+
			"them, and a starved input would make the closure's zero mean something else entirely",
			withRepeatFrequency, active)
	}
	if asMain != 0 {
		t.Errorf("%d build(s) now run Herald of the Breach as the MAIN skill, so "+
			"other.lua:2130-2132's hitTimeOverride decides numbers and needs a graded fixture; "+
			"this test recorded the arm as inert", asMain)
	}
	t.Logf("carriers=%d built=%d withRepeatFrequency=%d asMainSkill=%d", carriers, active, withRepeatFrequency, asMain)
}
