package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTraumaStacksConfigIsDispatched pins the config var name.
func TestTraumaStacksConfigIsDispatched(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0194-juggernaut-boneshatter-of-complex-trauma-2.xml")
	if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:TraumaStacks"); got == 0 {
		t.Errorf("Multiplier:TraumaStacks = 0 — the build's <Input name=\"traumaStacks\"> was never " +
			"dispatched, because configOrder listed the name `boneshatterTraumaStacks` instead " +
			"(ConfigOptions.lua:704 calls it `traumaStacks`)")
	}
}
