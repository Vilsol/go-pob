package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFlaskChargeOnCritChanceParses pins ModParser.lua:3539.
func TestFlaskChargeOnCritChanceParses(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("15% chance to gain a Flask Charge when you deal a Critical Strike", false)
	if e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("extra %q, %d mods", e.Extra, len(e.ModList))
	}
	m := e.ModList[0]
	if m.Name() != "FlaskChargeOnCritChance" || m.Type() != mod.TypeBase || len(m.Tags()) != 0 ||
		m.Value().Float() != 15 {
		t.Fatalf("got %s %s %v tags=%d", m.Name(), m.Type(), m.Value().Float(), len(m.Tags()))
	}

	n := ParseMod("Gain a Flask Charge when you deal a Critical Strike while affected by Precision", false)
	for _, nm := range n.ModList {
		if nm.Name() == "FlaskChargeOnCritChance" && len(nm.Tags()) == 0 {
			t.Fatalf("neighbour swallowed into an UNTAGGED FlaskChargeOnCritChance (extra %q)", n.Extra)
		}
	}
}

// TestMedicFlaskKeepsItsOwnMergeKey is the end-to-end half.
func TestMedicFlaskKeepsItsOwnMergeKey(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, b := range []string{"0024-trickster-frostbite.xml", "0057-trickster-spark-of-the-nova-2.xml", "1458-trickster-spark-of-the-nova-2.xml"} {
		env := corpusEnvForLoadedData(t, b)
		for key, want := range map[string]float64{"ActionSpeedMod": 1, "SelfChillEffect": -37, "SelfFreezeDuration": 18} {
			if got := env.Player.Output[key]; math.Abs(got-want) > math.Abs(want)*1e-6 {
				t.Errorf("%s %s = %v, PoB has %v", b, key, got, want)
			}
		}
	}
}
