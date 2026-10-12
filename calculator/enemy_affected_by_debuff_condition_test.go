package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnemyCarriesAffectedByDebuffCondition pins CalcPerform.lua:2548 — the ENEMY-side half of the AffectedBy<Debuff> condition pair.
func TestEnemyCarriesAffectedByDebuffCondition(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const name = "1518-slayer-sunder-of-earthbreaking-3.xml"
	const key = "AffectedByPride"

	corpusDir, _ := corpusNamesForLoadedData(t)
	raw, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("fixture build is gone from the corpus: %v", err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env == nil || env.Player == nil || env.EnemyModDB == nil {
		t.Fatal("no environment")
	}

	if !env.ModDB.Conditions[key] {
		t.Fatalf("REFUSING: %s does not set the PLAYER's %s, so it no longer reaches :2548 at all", name, key)
	}
	if !env.EnemyModDB.Conditions[key] {
		t.Errorf("enemy modDB is missing Condition:%s; CalcPerform.lua:2548 sets it on the enemy too", key)
	}
}
