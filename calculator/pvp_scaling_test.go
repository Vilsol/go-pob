package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestPvpScalingOnCarrier pins CalcDefence.lua:3073-3086 and CalcOffence.lua:3778-3826 on the corpus build that sets PvpScaling.
func TestPvpScalingOnCarrier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "15349-juggernaut-static-strike-of-gathering-lightning.xml"
	rec, ok := oracle[name]
	if !ok || rec.Output["PvPTotalTakenHit"] == 0 || rec.Output["PvpTotalDPS"] == 0 {
		t.Fatalf("REFUSING: %s carries no PvP output upstream; the fixture has moved", name)
	}
	raw, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env == nil || env.Player == nil {
		t.Fatalf("REFUSING: %s produced no player environment", name)
	}

	want, got := rec.Output["PvPTotalTakenHit"], env.Player.Output["PvPTotalTakenHit"]
	if !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("PvPTotalTakenHit = %v, PoB %v", got, want)
	}

	for _, key := range []string{"PvpAverageDamage", "PvpTotalDPS"} {
		want, got := rec.Output[key], env.Player.Output[key]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, PoB %v", key, got, want)
		}
	}
}
