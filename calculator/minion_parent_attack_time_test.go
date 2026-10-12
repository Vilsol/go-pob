package calculator

import (
	"os"
	"testing"
)

// TestNonSpectreMinionsUseParentMainHandAttackTime covers Replica Maata's Teaching end to end.
func TestNonSpectreMinionsUseParentMainHandAttackTime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, name := range []string{
		"1438-necromancer-shield-charge-5.xml",
		"0911-gladiator-herald-of-agony-3.xml",
	} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			env := corpusEnvForLoadedData(t, name)
			if env.Minion == nil {
				t.Fatalf("REFUSING: %s builds no minion", name)
			}
			if !env.ModDB.Flag(nil, "NonSpectreMinionsUseParentMainHandAttackTime") {
				t.Errorf("%s: the player carries no NonSpectreMinionsUseParentMainHandAttackTime flag", name)
			}
			for _, key := range []string{"Speed"} {
				want := rec.Minion[key]
				if want == 0 {
					t.Fatalf("REFUSING: PoB reports 0 for %s on %s", key, name)
				}
				got := env.Minion.Output[key]
				if !oracleClose(got, want, oracleDumpTolerance) {
					t.Errorf("%s Minion.%s = %g, PoB %g (ratio %.4f)", name, key, got, want, got/want)
				}
			}
		})
	}
}
