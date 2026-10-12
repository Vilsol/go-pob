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

// TestTriggerBotsDoubleTheSimulatedTriggerRate pins CalcTriggers.lua:810-813.
func TestTriggerBotsDoubleTheSimulatedTriggerRate(t *testing.T) {
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
	controlSeen, controlAgreed := 0, 0
	var worst []string
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
		got := env.Player.Output["SkillTriggerRate"]
		if got == 0 {
			continue
		}
		want := rec.Output["SkillTriggerRate"]
		ok2 := math.Abs(got-want) <= 0.005*math.Abs(want)
		bots := env.ModDB.Flag(nil, "HaveTriggerBots") &&
			env.Player.MainSkill.SkillTypes[data.SkillTypeSpell]
		if !bots {
			controlSeen++
			if ok2 {
				controlAgreed++
			}
			continue
		}
		carriers++
		if ok2 {
			agreed++
		} else if len(worst) < 8 {
			worst = append(worst, name)
		}
	}

	if carriers < 5 {
		t.Fatalf("REFUSING: only %d builds carry HaveTriggerBots with a spell main skill and a "+
			"published SkillTriggerRate. The population has moved and this test is measuring "+
			"nothing; it has NOT told you the doubling is right.", carriers)
	}
	if controlSeen < 100 {
		t.Fatalf("REFUSING: the control population is %d builds, too small to say the carriers "+
			"are special", controlSeen)
	}

	const agreeFloor = 12
	if agreed < agreeFloor {
		t.Errorf("SkillTriggerRate agrees with PoB on %d of %d HaveTriggerBots builds, floor %d "+
			"(e.g. %v). CalcTriggers.lua:810-813 doubles the SIMULATED rate for a spell when the "+
			"player has Triggerbots; a regression here means that arm stopped firing.",
			agreed, carriers, agreeFloor, worst)
	}
	t.Logf("TRIGGERBOTS: carriers %d, agreed %d | control (no bots) %d seen, %d agreed",
		carriers, agreed, controlSeen, controlAgreed)
}
