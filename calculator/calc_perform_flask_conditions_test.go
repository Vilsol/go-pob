package calculator

import (
	"os"
	"testing"
)

func corpusOnlyEnv(t *testing.T, file string) *Environment {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	return corpusEnvForLoadedData(t, file)
}

// TestFlaskUsingBaseConditions pins CalcPerform.lua:1797-1804: every active flask sets Using<BaseName>.
func TestFlaskUsingBaseConditions(t *testing.T) {
	env := corpusOnlyEnv(t, "0011-saboteur-cyclone")
	// PoB's conditions dump for this build.
	for _, c := range []string{"UsingFlask", "UsingGraniteFlask", "UsingJadeFlask", "UsingQuicksilverFlask", "UsingSilverFlask"} {
		if !env.ModDB.Conditions[c] {
			t.Errorf("condition %s not set; PoB sets it", c)
		}
	}
	if env.ModDB.Conditions["UsingLifeFlask"] {
		t.Errorf("UsingLifeFlask set; PoB leaves it unset")
	}
}

// TestIronFlaskUnbrokenWard pins CalcPerform.lua:1814-1823 on an edited build PoB dumped: 50 charges on ward break
// beat the Iron Flask's charges per use, 10 do not.
func TestIronFlaskUnbrokenWard(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	const old = "Magic Utility Flask Effects cannot be removed\n"
	for _, c := range []struct {
		charges string
		want    bool
	}{{"50", true}, {"10", false}} {
		env := corpusEnvWithEdit(t, "0754-reliquarian-icestorm", old,
			old+"Iron Flasks gain "+c.charges+" Charges when your Ward Breaks\n")
		if !env.ModDB.Conditions["UsingIronFlask"] {
			t.Fatalf("PREMISE BROKEN: the Iron Flask is not active")
		}
		if got := env.ModDB.Conditions["UnbrokenWard"]; got != c.want {
			t.Errorf("%s charges: UnbrokenWard = %v, PoB %v", c.charges, got, c.want)
		}
	}
}
