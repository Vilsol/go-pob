package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCountAllowZeroInputBlocksPlaceholder pins ConfigTab.lua:1089-1092.
func TestCountAllowZeroInputBlocksPlaceholder(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0724-elementalist-lightning-tendrils-of-escalation-2.xml")
	if got := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:SigilOfPowerStage"); got != 0 {
		t.Errorf("Multiplier:SigilOfPowerStage = %v, want 0 (explicit input 0, countAllowZero)", got)
	}
}
