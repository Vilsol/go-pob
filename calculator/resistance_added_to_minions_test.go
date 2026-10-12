package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestResistanceAddedToMinions pins ModParser.lua:4374 and CalcDefence.lua:648-653 together.
func TestResistanceAddedToMinions(t *testing.T) {
	t.Run("the line parses", func(t *testing.T) {
		const line = "Minions gain added Resistances equal to 50% of your Resistances"
		e := ParseMod(line, false)
		if e == nil || e.Extra != "" {
			t.Fatalf("%q left %v unparsed — an unparsed remainder discards the whole line", line, e)
		}
		if len(e.ModList) != 1 {
			t.Fatalf("%q produced %d mods, want 1", line, len(e.ModList))
		}
		m := e.ModList[0]
		if m.Name() != "ResistanceAddedToMinions" || m.Type() != mod.TypeBase || m.Value().Float() != 50 {
			t.Fatalf("mod = %s/%s/%v, want ResistanceAddedToMinions/BASE/50",
				m.Name(), m.Type(), m.Value().Float())
		}
	})

	t.Run("the minion receives it", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.SkipNow()
		}
		requireGameData(t, "3.29")

		env := corpusEnvForLoadedData(t, "0664-necromancer-raise-spectre-4.xml")
		if env.Minion == nil {
			t.Fatal("REFUSING: this fixture must have a minion")
		}

		if got, want := env.Minion.Actor.Output["ChaosResist"], 13.0; math.Abs(got-want) > 0.005 {
			t.Errorf("Minion.ChaosResist = %g, want %g (0 base + floor(26 × 50%%))", got, want)
		}
		if got, want := env.Player.Output["ChaosResist"], 26.0; math.Abs(got-want) > 0.005 {
			t.Errorf("Player.ChaosResist = %g, want %g (the grant's input, unchanged)", got, want)
		}
	})
}
