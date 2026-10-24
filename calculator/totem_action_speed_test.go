package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestTotemActionSpeedIsWritten pins CalcOffence.lua:1820-1826, the totem arm of the cast-rate block.
func TestTotemActionSpeedIsWritten(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.29" {
		t.Skipf("SKIP: this test pins 3.29 values but the process loaded %s game data", raw.LatestVersion)
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	cases := []struct {
		build    string
		mainHand bool
	}{
		{"1043-hierophant-winter-orb-2.xml", false},
		{"4104-ladder-AsidFlung_6035.ApusGotDownFoReal.xml", true},
	}

	for _, c := range cases {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", c.build)
		}
		pobTable, label := rec.Output, "TotemActionSpeed"
		if c.mainHand {
			pobTable, label = rec.MainHand, "MainHand.TotemActionSpeed"
		}
		want, ok := pobTable["TotemActionSpeed"]
		if !ok {
			t.Fatalf("REFUSING: PoB emits no %s for %s — wrong fixture", label, c.build)
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.build))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

		ourTable := env.Player.Output
		if c.mainHand {
			ourTable = env.Player.OutputTable[OutTableMainHand]
		}
		got, ok := ourTable["TotemActionSpeed"]
		switch {
		case !ok:
			t.Errorf("%s: we never write %s; PoB has %g", c.build, label, want)
		case !oracleClose(got, want, oracleDumpTolerance):
			t.Errorf("%s: %s = %g, PoB %g", c.build, label, got, want)
		}
	}
}
