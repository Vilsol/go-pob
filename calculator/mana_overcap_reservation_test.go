package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestManaOvercapLightningResReservationMatchesPoB pins CalcPerform.lua:2040 and :3328-3343 — the reservation re-run that Foulborn Choir of the Storm needs.
func TestManaOvercapLightningResReservationMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	dir := corpusDirForLoadedData(t)

	builds29 := []string{
		"1193-juggernaut-bladefall-of-trarthus",
		"1252-juggernaut-bladefall-of-trarthus-2",
		"1396-chieftain-bladefall-of-trarthus",
		"1404-juggernaut-bladefall-of-trarthus-3",
	}
	for _, name := range builds29 {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name+".xml"]
			if !ok {
				t.Fatalf("build not in the oracle blob")
			}
			file, err := os.ReadFile(filepath.Join(dir, name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			pob, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)

			if oc := env.Player.Output["LightningResistOverCap"]; oc <= 0 {
				t.Fatalf("LightningResistOverCap = %v: the fixture no longer exercises the mechanic", oc)
			}
			for _, stat := range []string{"Mana", "ManaUnreserved"} {
				want, ok := rec.Output[stat]
				if !ok {
					t.Fatalf("%s absent from the oracle record", stat)
				}
				got := env.Player.Output[stat]
				if want == 0 || math.Abs(got-want)/math.Abs(want) > oracleDumpTolerance {
					t.Errorf("%s = %v, PoB %v", stat, got, want)
				}
			}
		})
	}
}
