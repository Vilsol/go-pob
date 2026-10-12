package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestExtraAuraAppliesToTotems pins CalcPerform.lua:3290-3299 — the totem arm of the ExtraAura loop.
func TestExtraAuraAppliesToTotems(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.20" {
		t.Skipf("SKIP: this test pins 3.20 values but the process loaded %s game data", raw.LatestVersion)
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	const build = "0226-deadeye-lightning-arrow.xml"

	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", build)
	}
	want, ok := rec.MainHand["TotemActionSpeed"]
	if !ok {
		t.Fatalf("REFUSING: PoB emits no MainHand.TotemActionSpeed for %s — wrong fixture", build)
	}
	if want == 1 {
		t.Fatalf("REFUSING: PoB's MainHand.TotemActionSpeed is 1 on %s, so this fixture cannot "+
			"distinguish the totem aura arm from the bare key being written", build)
	}

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	if got := env.Player.OutputTable[OutTableMainHand]["TotemActionSpeed"]; !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("%s: MainHand.TotemActionSpeed = %g, PoB %g", build, got, want)
	}
	if !env.ModDB.Flag(nil, "Condition:TotemTailwind") {
		t.Errorf("%s: Condition:TotemTailwind is not set; the ExtraAura name rewrite did not run", build)
	}
}
