package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
)

// TestExertionSkillTypesResolveInGameData answers a question that CANNOT be answered by grepping this repository, and that I got wrong once by trying.
func TestExertionSkillTypesResolveInGameData(t *testing.T) {
	requireGameData(t, "3.29")

	if len(poe.ActiveSkillTypesByID) == 0 {
		t.Fatalf("REFUSING: ActiveSkillTypesByID is empty; the game data did not load and every " +
			"answer below would read as 'absent'")
	}
	if _, ok := poe.ActiveSkillTypesByID["ZZZNotASkillType"]; ok {
		t.Fatalf("REFUSING: a nonsense id resolved, so this lookup cannot distinguish present " +
			"from absent")
	}
	for _, known := range []string{"Warcry", "Brand", "Vaal", "Channel"} {
		if _, ok := poe.ActiveSkillTypesByID[known]; !ok {
			t.Fatalf("REFUSING: %q does not resolve, so the lookup is not the right index for "+
				"skill type ids and no conclusion below is safe", known)
		}
	}

	for _, id := range []string{"NeverExertable", "Triggered", "OtherThingUsesSkill", "Retaliation", "SummonsTotem"} {
		if _, ok := poe.ActiveSkillTypesByID[id]; !ok {
			t.Errorf("%s does not resolve in game data, so the exertion guard term written "+
				"against it is a silent no-op", id)
		}
	}
}

// TestNeverExertableMainSkillGetsNoWarcryUptime: 1785 is Flicker Strike with Multistrike, which adds NeverExertable, and Intimidating Cry.
func TestNeverExertableMainSkillGetsNoWarcryUptime(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1785-ladder-Haunterz_7064.SSJ_Degoshi.xml")
	if !env.Player.MainSkill.SkillTypes[data.SkillTypeNeverExertable] {
		t.Fatalf("fixture premise moved: the main skill no longer carries NeverExertable")
	}
	if got := env.Player.Output["GlobalWarcryUptimeRatio"]; got != 0 {
		t.Errorf("GlobalWarcryUptimeRatio = %v, want 0 (PoB)", got)
	}
}
