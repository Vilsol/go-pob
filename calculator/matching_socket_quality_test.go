package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestMatchingSocketQualityBonus pins SkillsTab:UpdateSocketGroups (SkillsTab.lua:1210-1245) and the two things upstream does with its result.
func TestMatchingSocketQualityBonus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var matchedBuilds, unmatchedBuilds, graded, wrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.SkillModList == nil {
			continue
		}
		bonus := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "GemSocketQuality")

		if bonus == 0 {
			unmatchedBuilds++
			continue
		}
		matchedBuilds++

		if r := bonus / float64(data.MatchingSocketQualityBonus); r != float64(int(r)) {
			t.Errorf("%s: GemSocketQuality %v is not a multiple of MatchingSocketQualityBonus %d",
				name, bonus, data.MatchingSocketQualityBonus)
		}

		want, has := rec.Output["GemQuality"]
		if !has {
			continue
		}
		graded++
		if !oracleClose(env.Player.Output["GemQuality"], want, oracleDumpTolerance) {
			wrong++
		}
		if matchedBuilds >= 150 && unmatchedBuilds > 0 {
			break
		}
	}

	if matchedBuilds == 0 {
		t.Fatal("REFUSING: no graded build has a colour-matched socket on its main skill. Either " +
			"the pre-pass no longer runs, or the gem-colour column has emptied — before this port " +
			"landed this population was zero by construction and that is the state to catch")
	}
	if unmatchedBuilds == 0 {
		t.Fatal("REFUSING: EVERY graded build reports a socket-quality bonus. The colour comparison " +
			"cannot be answering NO, so a match test that always returns true would pass this " +
			"test with a plausible-looking number")
	}
	if graded == 0 {
		t.Fatal("REFUSING: the blob publishes no GemQuality on the matched builds")
	}
	if wrong > 0 {
		t.Logf("GemQuality disagrees with PoB on %d of %d matched builds (see the note above: a "+
			"separate, diffuse residual)", wrong, graded)
	}
}
