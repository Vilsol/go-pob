package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCallToArmsSuppressesWarcryBuffs is the OTHER DIRECTION, and it exists because turning the branch on without it OVER-APPLIED.
func TestCallToArmsSuppressesWarcryBuffs(t *testing.T) {
	env := warcryFixtureEnv(t, "1899-ladder-Synetik_7053.Kurael_EnduCry.xml")

	if !env.ModDB.Flag(nil, "CannotGainWarcryBuffs") {
		t.Fatal("1899: CannotGainWarcryBuffs is not set; Call to Arms is either no longer " +
			"allocated on this build or the parser row is gone")
	}

	for _, probe := range []struct {
		stat, source string
		modType      mod.Type
	}{
		{"Armour", "Skill:SeismicCry", mod.TypeMore},
		{"StunThreshold", "Skill:SeismicCry", mod.TypeIncrease},
		{"LifeRegenPercent", "Skill:EnduringCry", mod.TypeBase},
	} {
		if v, ok := warcryMod(env, probe.stat, probe.source, probe.modType); ok {
			t.Errorf("1899: %s %s %s = %v applied through Call to Arms; PoB grants no warcry "+
				"buff on this build", probe.source, probe.modType, probe.stat, v)
		}
	}
}
