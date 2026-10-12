package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWitheredStackCountSelfAppliesFromThePlaceholder covers the second general-placeholder row with a live consumer here.
func TestWitheredStackCountSelfAppliesFromThePlaceholder(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:WitheredStack"); math.Abs(got-15) > 1e-9 {
		t.Errorf("player Multiplier:WitheredStack BASE = %g, want 15 "+
			"(ConfigOptions.lua:1220, defaultPlaceholderState = 15)", got)
	}
}
