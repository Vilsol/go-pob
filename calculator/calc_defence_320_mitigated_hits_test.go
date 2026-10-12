package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

// TestMitigatedHitsEqualUnmitigatedAtFullDamageChance: CalcDefence.lua:1991 does NOT always run the hit search.
//
//	output["NumberOfMitigatedDamagingHits"] = (output["ConfiguredDamageChance"] ~= 100
func TestMitigatedHitsEqualUnmitigatedAtFullDamageChance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true to run")
	}
	requireGameData(t, "3.20")

	corpusDir, names := corpusNamesForLoadedData(t)

	checked, violations := 0, 0
	for _, file := range names {
		rawXML, err := os.ReadFile(filepath.Join(corpusDir, file))
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(rawXML)
		if err != nil {
			continue
		}
		env := buildOutputOrNil(build)
		if env == nil {
			continue
		}
		out := env.Player.Output
		if out["ConfiguredDamageChance"] != 100 {
			continue
		}
		checked++
		if out["NumberOfMitigatedDamagingHits"] != out["NumberOfDamagingHits"] {
			violations++
			if violations <= 5 {
				t.Errorf("%s: ConfiguredDamageChance is exactly 100 but NumberOfMitigatedDamagingHits "+
					"= %.10g while NumberOfDamagingHits = %.10g", file,
					out["NumberOfMitigatedDamagingHits"], out["NumberOfDamagingHits"])
			}
		}
	}

	if checked == 0 {
		t.Fatalf("REFUSING: no corpus build reached ConfiguredDamageChance == 100, so this test " +
			"proves nothing. The population moved; re-derive it before trusting a pass.")
	}
	t.Logf("%d of %d builds have ConfiguredDamageChance exactly 100; %d violate the fallback",
		checked, len(names), violations)
	if violations > 0 {
		t.Fatalf("%d builds take the search where CalcDefence.lua:1991 reuses NumberOfDamagingHits", violations)
	}
}

func buildOutputOrNil(build *pob.PathOfBuilding) *Environment {
	var env *Environment
	func() {
		defer func() { _ = recover() }()
		env = (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	}()
	return env
}
