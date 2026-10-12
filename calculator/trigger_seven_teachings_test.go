package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSevenTeachingsTriggerFullDPS pins CalcTriggers.lua's ["seven teachings"] entry (:1131-1134) on its corpus carrier.
func TestSevenTeachingsTriggerFullDPS(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0839-assassin-doryanis-touch.xml"
	rec, ok := oracle[name]
	if !ok || rec.Output["FullDPS"] == 0 {
		t.Fatalf("REFUSING: %s carries no FullDPS upstream; the fixture has moved", name)
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

	want, got := rec.Output["FullDPS"], env.Player.Output["FullDPS"]
	if !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("%s FullDPS = %v, PoB %v (x%.4f)", name, got, want, got/want)
	}
}
