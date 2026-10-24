package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSkillCostsAreFinite is the end-to-end pin on the cost divisor.
func TestSkillCostsAreFinite(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	const name = "15226-ladder-zelltran0303_7154.ADBZ_SungJinWo.xml"

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	want, ok := rec.Output["ManaCost"]
	if !ok {
		t.Fatalf("REFUSING: the blob has no ManaCost for %s", name)
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
		t.Fatal("REFUSING: the build did not calculate")
	}

	got := env.Player.Output["ManaCost"]
	if math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("ManaCost is %v — the cost divisor is zero", got)
	}
	if !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("ManaCost: got %v, PoB %v", got, want)
	}
}

// TestCostDivisorsMatchGameDataWhereItHasThem cross-checks the transcribed table against go-pob-data for every row where go-pob-data actually carries a divisor.
func TestCostDivisorsMatchGameDataWhereItHasThem(t *testing.T) {
	checked, empty := 0, 0
	for id, ct := range poe.CostTypesByID {
		ours, ok := costDivisors[id]
		if !ok {
			t.Errorf("%s: go-pob-data has a cost type that src/Data/Costs.lua does not; "+
				"costDivisors would divide by zero for it", id)
			continue
		}
		if ct.Divisor == 0 {
			empty++
			continue
		}
		checked++
		if float64(ct.Divisor) != ours {
			t.Errorf("%s: costDivisors has %v, go-pob-data %v", id, ours, ct.Divisor)
		}
	}
	for id, d := range costDivisors {
		if d <= 0 {
			t.Errorf("%s: costDivisors is %v — a non-positive divisor is the defect this "+
				"table exists to prevent", id, d)
		}
	}
	t.Logf("%s data: %d of %d cost types cross-checked against go-pob-data, %d left empty by "+
		"the export", raw.LatestVersion, checked, checked+empty, empty)
	if checked == 0 {
		t.Logf("⚠️ ZERO cross-checked: this run proves only that the table is self-consistent, " +
			"not that it is right. See the comment above for the control.")
	}
}
