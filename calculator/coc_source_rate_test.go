package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestCastOnCritEffectiveSourceRateIsTheSourceRate pins CalcTriggers.lua:715 against :809.
func TestCastOnCritEffectiveSourceRateIsTheSourceRate(t *testing.T) {
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

	carriers, agreed, distinctFromSim := 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["EffectiveSourceRate"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
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
		ms := env.Player.MainSkill
		if ms.TriggeredBy == nil || ms.TriggeredBy.GrantedEffect == nil ||
			ms.TriggeredBy.GrantedEffect.ID() != "SupportCastOnCritTriggered" {
			continue
		}
		got := env.Player.Output["EffectiveSourceRate"]
		if got == 0 {
			continue
		}
		carriers++
		if math.Abs(got-rec.Output["EffectiveSourceRate"]) <= 0.005*math.Abs(rec.Output["EffectiveSourceRate"]) {
			agreed++
		}
		if math.Abs(got-env.Player.Output["SkillTriggerRate"]) > 1e-9 {
			distinctFromSim++
		}
		_ = data.SkillTypeAttack
	}

	if carriers < 50 {
		t.Fatalf("REFUSING: only %d Cast-on-Crit carriers publish EffectiveSourceRate; the population has moved", carriers)
	}
	const agreeFloor, distinctFloor = 56, 30
	if agreed < agreeFloor {
		t.Errorf("EffectiveSourceRate agreement REGRESSED to %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	if distinctFromSim < distinctFloor {
		t.Errorf("EffectiveSourceRate equals SkillTriggerRate on all but %d of %d carriers, floor %d -- the two keys have been conflated again",
			distinctFromSim, carriers, distinctFloor)
	}
	t.Logf("CoC carriers %d, agreed %d, distinct from simulated %d", carriers, agreed, distinctFromSim)
}
