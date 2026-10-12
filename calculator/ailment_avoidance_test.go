package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestElementalAilmentAvoidanceReadsTheAggregateMods pins CalcDefence.lua:889-894, which this port collapsed into ONE loop over ONE mod name.
//
//	for _, ailment in ipairs(data.nonElementalAilmentTypeList) do
//	    output[ailment.."AvoidChance"] = m_min(modDB:Sum("BASE", nil, "Avoid"..ailment, "AvoidAilments"), 100)
//	for _, ailment in ipairs(data.elementalAilmentTypeList) do
//	    output[ailment.."AvoidChance"] = m_min(modDB:Sum("BASE", nil, "Avoid"..ailment, "AvoidAilments", "AvoidElementalAilments"), 100)
func TestElementalAilmentAvoidanceReadsTheAggregateMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.20")

	const name = "2412-guardian-herald-of-agony-3.xml"

	want, got := ailmentAvoidance(t, name)

	assertAvoidChances(t, name, []string{
		"IgniteAvoidChance", "ChillAvoidChance", "FreezeAvoidChance", "ShockAvoidChance",
		"ScorchAvoidChance", "BrittleAvoidChance", "SapAvoidChance",
	}, want, got)
}

func ailmentAvoidance(t *testing.T, name string) (map[string]float64, map[string]float64) {
	t.Helper()

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the 3.20 blob", name)
	}

	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env == nil {
		t.Fatalf("REFUSING: %s did not calculate", name)
	}
	return rec.Output, env.Player.Output
}

func assertAvoidChances(t *testing.T, name string, keys []string, want, got map[string]float64) {
	t.Helper()

	checked := 0
	for _, key := range keys {
		exp, ok := want[key]
		if !ok {
			continue
		}
		checked++
		if v := got[key]; !oracleClose(v, exp, oracleDumpTolerance) {
			t.Errorf("%s: got %v, PoB %v", key, v, exp)
		}
	}
	if checked < len(keys) {
		t.Fatalf("REFUSING: the blob has only %d of the %d avoid-chance keys for %s",
			checked, len(keys), name)
	}
}
