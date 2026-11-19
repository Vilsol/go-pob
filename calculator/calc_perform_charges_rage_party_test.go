package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

func corpusEnvWithReplacements(t *testing.T, file string, replacements map[string]string) *Environment {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), file+".xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	s := string(raw)
	for old, replacement := range replacements {
		if n := strings.Count(s, old); n != 1 {
			t.Fatalf("REFUSING: %q occurs %d times", old, n)
		}
		s = strings.Replace(s, old, replacement, 1)
	}
	b, err := builds.ParseBuild([]byte(s))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
}

// TestMinimumChargesEqualMaximum pins CalcPerform.lua:1039-1047 on an edited build PoB dumped.
func TestMinimumChargesEqualMaximum(t *testing.T) {
	env := corpusEnvWithReplacements(t, "0059-occultist-viper-strike", map[string]string{
		"{crafted}+22% to Fire and Lightning Resistances\n": "{crafted}+22% to Fire and Lightning Resistances\nMinimum Frenzy Charges equal to Maximum while stationary\n" +
			"Minimum Endurance Charges equal to Maximum while stationary\nMinimum Power Charges equal to Maximum while stationary\n",
		"<ConfigSet id=\"1\" title=\"Default\">\n": "<ConfigSet id=\"1\" title=\"Default\">\n\t\t\t<Input number=\"1\" name=\"conditionStationary\"/>\n",
	})
	// PoB's dump of this edit.
	for stat, want := range map[string]float64{"FrenzyChargesMin": 3, "EnduranceChargesMin": 3, "PowerChargesMin": 3, "TotalCharges": 9} {
		if got := env.Player.Output[stat]; got != want {
			t.Errorf("%s = %v, PoB %v", stat, got, want)
		}
	}
}

// TestMinimumRageSetsRageMultiplier pins CalcPerform.lua:651-654 on an edited build PoB dumped.
func TestMinimumRageSetsRageMultiplier(t *testing.T) {
	env := corpusEnvWithReplacements(t, "0161-assassin-viper-strike-of-the-mamba", map[string]string{
		"{crafted}+28 to maximum Energy Shield\n": "{crafted}+28 to maximum Energy Shield\n+10 to Minimum Rage\n",
	})
	// PoB's dump of this edit.
	if got := env.ModDB.Multipliers["Rage"]; got != 10 {
		t.Errorf("Rage multiplier = %v, PoB 10", got)
	}
}

// TestPartyMemberChargesAndFortifyEqualToYours pins CalcPerform.lua:655 and 1003 on an edited build PoB dumped: the
// imported party member's maximum endurance charges and fortification replace the player's.
func TestPartyMemberChargesAndFortifyEqualToYours(t *testing.T) {
	env := corpusEnvWithReplacements(t, "0003-guardian-unknown", map[string]string{
		`<Party destination="All" ShowAdvanceTools="false" append="false"/>`: `<Party destination="All" ShowAdvanceTools="false" append="false">` + "\n\t\t<ImportedBuffs name=\"PartyMemberStats\">\n" +
			"EnduranceChargesMax=7\nFortificationStacks=20\nPartyMemberMaximumEnduranceChargesEqualToYours\n" +
			"PartyMemberFortifyEqualToYours\n\t\t</ImportedBuffs>\n\t</Party>",
	})
	// PoB's dump of this edit.
	if got := env.Player.Output["EnduranceChargesMax"]; got != 7 {
		t.Errorf("EnduranceChargesMax = %v, PoB 7", got)
	}
	if !env.ModDB.Conditions["Fortified"] {
		t.Errorf("Fortified not set; PoB sets it")
	}
}
