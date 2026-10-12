package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestArcaneSurgeBuffApplies pins CalcPerform.lua:686-704, a buff block that was never ported.
func TestArcaneSurgeBuffApplies(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0020-assassin-storm-burst.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	if rec.Output["Speed"] == 0 {
		t.Fatalf("REFUSING: %s publishes no Speed; the fixture has moved", name)
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	if !env.ModDB.Conditions["AffectedByArcaneSurge"] {
		t.Fatalf("%s: AffectedByArcaneSurge not set -- the block did not run", name)
	}
	castFlag := mod.MFlagCast
	var found float64
	for _, r := range env.ModDB.Tabulate(mod.TypeIncrease, &moddb.ListCfg{Flags: &castFlag}, "Speed") {
		if r.Mod.GetSource() == "Arcane Surge" {
			found = r.Mod.Value().Float()
		}
	}
	if found != 20 {
		t.Errorf("%s: Arcane Surge contributes Speed INC %v, want 20 (PoB's dump row is "+
			"`Other:Arcane Surge INC 20`)", name, found)
	}
}
