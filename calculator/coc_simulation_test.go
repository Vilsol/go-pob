package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCastOnCritSkillTriggerRate pins the 3.29 rotation simulation, CalcTriggers.lua:90-133, against the oracle for the Cast-on-Crit population.
func TestCastOnCritSkillTriggerRate(t *testing.T) {
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

	carriers, agreed := 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["SkillTriggerRate"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil || !strings.Contains(string(raw), "Cast On Critical Strike") {
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
		got := env.Player.Output["SkillTriggerRate"]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: SkillTriggerRate is %v -- it cannot be graded and the NaN ceiling only moves down", name, got)
		}
		carriers++
		want := rec.Output["SkillTriggerRate"]
		if math.Abs(got-want) <= 0.005*math.Abs(want) {
			agreed++
		}
	}

	if carriers < 50 {
		t.Fatalf("REFUSING: only %d Cast-on-Crit carriers publish SkillTriggerRate; the population has moved", carriers)
	}
	const agreeFloor = 46
	if agreed < agreeFloor {
		t.Errorf("SkillTriggerRate agreement REGRESSED to %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	t.Logf("Cast-on-Crit SkillTriggerRate: agreed %d of %d", agreed, carriers)
}
