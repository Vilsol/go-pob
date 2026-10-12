package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestLeechingOnFullLifeDoesNotWriteTheConfig: CalcPerform.lua:873-880 sets the Leeching conditions on the modDB only.
func TestLeechingOnFullLifeDoesNotWriteTheConfig(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"Life Leech effects are not removed at Full Life",
		"Energy Shield Leech effects are not removed when Energy Shield is Filled",
	} {
		t.Run(line, func(t *testing.T) {
			build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
			for _, in := range build.Config.Inputs {
				if in.Name == "conditionLeeching" {
					t.Fatalf("REFUSING: the build already sets %s", in.Name)
				}
			}
			build.SetConfigOption(pob.Input{Name: "customMods", String: utils.Ptr(line)})
			before := len(build.Config.Inputs)

			misc := GetMiscCalculator(build)
			if !misc.Base().Player.ModDB.Conditions["Leeching"] {
				t.Fatalf("REFUSING: %q did not make the build leech, so it cannot test the write", line)
			}
			for _, in := range build.Config.Inputs {
				if in.Name == "conditionLeeching" {
					t.Fatalf("the calculation wrote conditionLeeching into the build's config")
				}
			}
			if len(build.Config.Inputs) != before {
				t.Fatalf("the calculation changed the build's config inputs: %d -> %d", before, len(build.Config.Inputs))
			}
		})
	}
}
