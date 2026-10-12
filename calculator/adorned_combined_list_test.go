package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAdornedScalesTheCombinedList pins CalcSetup.lua:1135-1141.
func TestAdornedScalesTheCombinedList(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, name := range []string{
		"1252-juggernaut-bladefall-of-trarthus-2.xml",
		"1404-juggernaut-bladefall-of-trarthus-3.xml",
	} {
		t.Run(name, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, name)
			var values []float64
			for _, r := range env.ModDB.Tabulate(mod.TypeIncrease, nil, "ManaCost") {
				if strings.Contains(string(r.Mod.GetSource()), "Synthesised Warding Cobalt Jewel") {
					values = append(values, r.Value)
				}
			}
			if len(values) != 1 || values[0] != -17 {
				t.Errorf("jewel ManaCost INC = %v, want one mod at -17", values)
			}
		})
	}
}
