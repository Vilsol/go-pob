package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestForbiddenRiteOfSoulSacrificeAddsEnergyShield pins act_int.lua:7549-7589 — the preDamageFunc of ForbiddenRiteAltX (Forbidden Rite of Soul Sacrifice).
func TestForbiddenRiteOfSoulSacrificeAddsEnergyShield(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const agreeFloor = 25

	carriers, agreed := 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, published := rec.Output["ChaosMinBase"]
		if !published || want == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "ForbiddenRiteAltX") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		if env.Player.MainSkill.ActiveEffect == nil || env.Player.MainSkill.ActiveEffect.GrantedEffect == nil ||
			env.Player.MainSkill.ActiveEffect.GrantedEffect.Raw == nil ||
			env.Player.MainSkill.ActiveEffect.GrantedEffect.Raw.ID != "ForbiddenRiteAltX" {
			continue
		}
		carriers++
		got := env.Player.Output["ChaosMinBase"]
		if math.Abs(got-want) <= math.Abs(want)*oracleDumpTolerance {
			agreed++
		}
	}

	if carriers < 15 {
		t.Fatalf("REFUSING: only %d builds run ForbiddenRiteAltX as the main skill; the population has moved", carriers)
	}
	if agreed < agreeFloor {
		t.Errorf("ChaosMinBase agreement on ForbiddenRiteAltX carriers is %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	t.Logf("ChaosMinBase on ForbiddenRiteAltX carriers: agreed %d of %d", agreed, carriers)
}
