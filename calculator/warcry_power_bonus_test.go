package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

func warcryFixtureEnv(t *testing.T, name string) *Environment {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
}

func warcryMod(env *Environment, stat, source string, modType mod.Type) (float64, bool) {
	for _, m := range env.Player.ModDB.Mods[stat] {
		if string(m.GetSource()) == source && m.Type() == modType {
			return ourModValue(m), true
		}
	}
	return 0, false
}

// TestWarcryPowerBonusScalesBuffMods pins CalcPerform.lua:2311-2351.
//
//	warcryBuff[1].warcryPowerBonus = m_floor((limit and m_min(warcryPower, limit) or warcryPower) / div)
func TestWarcryPowerBonusScalesBuffMods(t *testing.T) {
	env := warcryFixtureEnv(t, "0405-chieftain-tectonic-slam-2.xml")

	if got := env.Player.Output["WarcryPower"]; got != 999999 {
		t.Fatalf("REFUSING: WarcryPower = %v, want 999999; the fixture no longer runs "+
			"WarcryInfinitePower and cannot exercise the floor", got)
	}

	more, ok := warcryMod(env, "Armour", "Skill:SeismicCry", mod.TypeMore)
	if !ok {
		t.Fatalf("0405: no Skill:SeismicCry MORE Armour modifier; the warcry buff reached nothing")
	}
	if more != 40 {
		t.Errorf("0405: Skill:SeismicCry MORE Armour = %v, want 40 (32 means the WarcryPower "+
			"override is still written after the buff pass)", more)
	}

	stun, ok := warcryMod(env, "StunThreshold", "Skill:SeismicCry", mod.TypeIncrease)
	if !ok {
		t.Fatalf("0405: no Skill:SeismicCry INC StunThreshold modifier")
	}
	if stun != 120 {
		t.Errorf("0405: Skill:SeismicCry INC StunThreshold = %v, want 120", stun)
	}
}
