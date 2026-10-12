package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSiegebreakerGroundRowsCarryThePercent pins ModParser.lua:2326-2333.
func TestSiegebreakerGroundRowsCarryThePercent(t *testing.T) {
	cases := []struct {
		line    string
		skillID string
		mult    string
		percent float64
	}{
		{"Your Minions spread Caustic Ground on Death, dealing 20% of their maximum Life as Chaos Damage per second", "SiegebreakerCausticGround", "Multiplier:SiegebreakerCausticGroundPercent", 20},
		{"Your Minions spread Caustic Ground on Death, dealing 24% of their maximum Life as Chaos Damage per second", "SiegebreakerCausticGround", "Multiplier:SiegebreakerCausticGroundPercent", 24},
		{"Your Minions spread Burning Ground on Death, dealing 20% of their maximum Life as Fire Damage per second", "ReplicaSiegebreakerBurningGround", "Multiplier:SiegebreakerBurningGroundPercent", 20},
	}

	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e == nil || e.Extra != "" {
				t.Fatalf("line did not parse whole: %v", e)
			}
			if len(e.ModList) != 2 {
				t.Fatalf("got %d mods, want 2 (ExtraMinionSkill + MinionModifier)", len(e.ModList))
			}
			ems, ok := e.ModList[0].Value().List().(mod.ExtraMinionSkill)
			if !ok || ems.SkillID != c.skillID {
				t.Errorf("mod 0 = %v, want ExtraMinionSkill %s", e.ModList[0].Value().List(), c.skillID)
			}
			mm, ok := e.ModList[1].Value().List().(mod.MinionModifier)
			if !ok {
				t.Fatalf("mod 1 = %T, want mod.MinionModifier", e.ModList[1].Value().List())
			}
			if mm.Mod.Name() != c.mult || mm.Mod.Type() != mod.TypeBase {
				t.Fatalf("nested mod = %s %s, want %s BASE", mm.Mod.Name(), mm.Mod.Type(), c.mult)
			}
			if got := mm.Mod.Value().Float(); got != c.percent {
				t.Errorf("percent = %v, want %v", got, c.percent)
			}
		})
	}
}
