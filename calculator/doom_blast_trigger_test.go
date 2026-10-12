package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestDoomBlastTriggerEntry pins CalcTriggers.lua:1383-1392.
func TestDoomBlastTriggerEntry(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for build, want := range map[string]float64{
		"1307-occultist-doom-blast-2.xml": 10.933333333333,
		"0271-occultist-doom-blast.xml":   6.6666666666667,
	} {
		env := corpusEnvForLoadedData(t, build)
		if got := env.Player.Output["TriggerRateCap"]; math.Abs(got-want) > want*1e-6 {
			t.Errorf("%s TriggerRateCap = %v, PoB has %v", build, got, want)
		}
	}
}

// TestDoomBlastVixenArm pins :700-713 with doomBlastSource set EXPLICITLY.
func TestDoomBlastVixenArm(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, c := range []struct {
		build, key string
		want       float64
	}{
		{"1307-occultist-doom-blast-2.xml", "EffectiveSourceRate", 4.88488},
		{"1307-occultist-doom-blast-2.xml", "Speed", 4.88488},
		{"1318-occultist-doom-blast-3.xml", "Speed", 4.86},
		{"0271-occultist-doom-blast.xml", "EffectiveSourceRate", 2.4},
	} {
		raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.build))
		if err != nil {
			t.Fatal(err)
		}
		b, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		if b.GetStringOption("doomBlastSource") != "" {
			t.Fatalf("REFUSING: %s already saves doomBlastSource", c.build)
		}
		b.Config.Inputs = append(b.Config.Inputs, pob.Input{Name: "doomBlastSource", String: utils.Ptr("vixen")})
		env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
		if got := env.Player.Output[c.key]; math.Abs(got-c.want) > c.want*1e-6 {
			t.Errorf("%s %s = %v, PoB has %v", c.build, c.key, got, c.want)
		}
	}
}
