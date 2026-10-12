package calculator

import (
	"os"
	"testing"
)

// TestMirageRunsAtTheRawGemLevel pins CalcActiveSkill.lua:171-176.
func TestMirageRunsAtTheRawGemLevel(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, name := range []string{
		"0573-pathfinder-caustic-arrow.xml",
		"1143-pathfinder-caustic-arrow-2.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob; this test has lost its subject", name)
		}
		want, ok := rec.Output["MirageDPS"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: %s reports no nonzero MirageDPS in the oracle; it no longer "+
				"exercises the mirage at all", name)
		}

		env := corpusEnvForLoadedData(t, name)
		skill := env.Player.MainSkill
		if skill == nil || skill.ActiveEffect == nil || skill.ActiveEffect.SrcInstance == nil {
			t.Fatalf("REFUSING: %s has no main skill with a source gem", name)
		}
		if skill.ActiveEffect.Level <= skill.ActiveEffect.SrcInstance.Level {
			t.Fatalf("PREMISE BROKEN on %s: the player's skill is at gem level %d and its source "+
				"gem at %d, so resetting the mirage to the source level changes nothing and this "+
				"build cannot distinguish a ported reset from an absent one",
				name, skill.ActiveEffect.Level, skill.ActiveEffect.SrcInstance.Level)
		}

		got := env.Player.Output["MirageDPS"]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s MirageDPS = %g, PoB %g (ratio %.4f); the mirage should run at the raw "+
				"gem level %d, not the player's boosted %d",
				name, got, want, got/want,
				skill.ActiveEffect.SrcInstance.Level, skill.ActiveEffect.Level)
		}
	}
}

// TestMirageRunsAtTheRawGemQuality is the quality half of the same reset (CalcActiveSkill.lua:173).
func TestMirageRunsAtTheRawGemQuality(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, name := range []string{
		"0135-deadeye-elemental-hit-of-the-spectrum.xml",
		"9346-ladder-ChefBoyardee437_2880.Oaks_OnTheSpectrum.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob; this test has lost its subject", name)
		}
		want, ok := rec.Output["MirageDPS"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: %s reports no nonzero MirageDPS in the oracle", name)
		}

		env := corpusEnvForLoadedData(t, name)
		skill := env.Player.MainSkill
		if skill == nil || skill.ActiveEffect == nil || skill.ActiveEffect.SrcInstance == nil {
			t.Fatalf("REFUSING: %s has no main skill with a source gem", name)
		}
		ae := skill.ActiveEffect
		if ae.Level != ae.SrcInstance.Level {
			t.Fatalf("PREMISE BROKEN on %s: gem level %d against source level %d, so the LEVEL "+
				"reset is live here and this build no longer isolates the quality half",
				name, ae.Level, ae.SrcInstance.Level)
		}
		if ae.Quality <= ae.SrcInstance.Quality {
			t.Fatalf("PREMISE BROKEN on %s: gem quality %d against source quality %d, so resetting "+
				"the mirage to the source quality changes nothing here", name, ae.Quality, ae.SrcInstance.Quality)
		}
		if playerDPS := env.Player.Output["TotalDPS"]; !oracleClose(playerDPS, rec.Output["TotalDPS"], 1e-9) {
			t.Fatalf("PREMISE BROKEN on %s: the player's own TotalDPS is %g against PoB %g, so a "+
				"MirageDPS divergence here need not be the mirage's",
				name, playerDPS, rec.Output["TotalDPS"])
		}

		got := env.Player.Output["MirageDPS"]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s MirageDPS = %g, PoB %g (ratio %.4f); the mirage should run at the raw "+
				"gem quality %d, not the player's socket-boosted %d",
				name, got, want, got/want, ae.SrcInstance.Quality, ae.Quality)
		}
	}
}
