package calculator

import (
	"os"
	"testing"
)

// TestBlockEffectMatchesThePinsBranch closes one of two recorded latent divergences.
//
//	output.BlockEffect = 100 - modDB:Sum("BASE", nil, "BlockEffect")
//	if output.BlockEffect ~= 0 then
//	end
func TestBlockEffectMatchesThePinsBranch(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, build := range []string{
		"0001-necromancer-unknown.xml",
		"0003-guardian-unknown.xml",
	} {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			out := env.Player.Output

			if out["BlockEffect"] != 100 {
				t.Fatalf("REFUSING: %s has BlockEffect %v, not 100; the divergent arm is the "+
					"BlockEffect == 100 one and this fixture cannot reach it", build, out["BlockEffect"])
			}
			if out["ShowBlockEffect"] != 1 {
				t.Errorf("ShowBlockEffect = %v, want 1 — CalcDefence.lua:769's branch is `~= 0` and "+
					"100 is not 0", out["ShowBlockEffect"])
			}
			if _, ok := out["DamageTakenOnBlock"]; !ok {
				t.Errorf("DamageTakenOnBlock is absent; the pin writes 100 - BlockEffect = 0 here, " +
					"and an absent key scores ok against PoB's 0 on every build")
			}
		})
	}
}
