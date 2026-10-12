package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSpectralTigerCountAppliesFromThePlaceholder is the WOLF twin's missing half.
func TestSpectralTigerCountAppliesFromThePlaceholder(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0350-necromancer-bone-offering.xml"
	env := corpusEnvForLoadedData(t, build)

	for _, in := range env.Build.Config.Inputs {
		if in.Name == "configSpectralTigerCount" {
			t.Fatalf("premise broken: %s now writes a configSpectralTigerCount <Input>, so it no "+
				"longer exercises the placeholder path", build)
		}
	}

	if got := env.ConfigPlaceholder["configSpectralTigerCount"]; got != 5 {
		t.Errorf("ConfigPlaceholder[configSpectralTigerCount] = %g, want 5 "+
			"(ConfigOptions.lua:633 defaultPlaceholderState)", got)
	}
	if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:SpectralTigerConfig"); got != 5 {
		t.Errorf("Multiplier:SpectralTigerConfig BASE = %g, want 5; 0 means the handler never ran", got)
	}
}
