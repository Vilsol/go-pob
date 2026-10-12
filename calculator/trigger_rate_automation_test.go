package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAutomationPublishesTriggerRates pins CalcTriggers.lua's ["automation"] configTable entry.
func TestAutomationPublishesTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const tol = 0.005

	checked, agreed := 0, 0
	selfArm, selfAgreed := 0, 0
	distinct := map[float64]bool{}
	var diverged []string
	highSide := map[string]bool{}
	var highNames []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["TriggerRateCap"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Automation") {
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
		if ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil {
			continue
		}
		isSelf := ms.ActiveEffect.GrantedEffect.Name() == "Automation"
		triggeredByAutomation := ms.TriggeredBy != nil && ms.TriggeredBy.GrantedEffect != nil &&
			ms.TriggeredBy.GrantedEffect.Name() == "Automation"
		if !isSelf && !triggeredByAutomation {
			continue
		}

		checked++
		if isSelf {
			selfArm++
		}
		distinct[math.Round(rec.Output["TriggerRateCap"]*1e6)/1e6] = true

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[key]
			exp, has := rec.Output[key]
			if !has {
				continue
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
				if got > exp && !highSide[name] {
					highSide[name] = true
					highNames = append(highNames, name)
				}
			}
		}
		if agrees {
			agreed++
			if isSelf {
				selfAgreed++
			}
		} else if len(diverged) < 12 {
			diverged = append(diverged, name)
		}
	}

	if checked < 180 {
		t.Fatalf("REFUSING: only %d Automation carriers dispatched; the fixture population has moved "+
			"(221 resolved to this configTable key when written)", checked)
	}
	if selfArm < 60 {
		t.Fatalf("REFUSING: only %d builds take the Automation-is-the-main-skill arm; the "+
			"skipEffectiveRate half of this entry is untested (93 when written)", selfArm)
	}
	if len(distinct) < 5 {
		t.Fatalf("REFUSING: only %d distinct caps published; the sweep no longer exercises the "+
			"cooldown expression and a hardcoded constant would pass it", len(distinct))
	}

	t.Logf("automation: %d dispatched (%d self-arm), %d agree (%d self-arm), %d distinct caps",
		checked, selfArm, agreed, selfAgreed, len(distinct))

	const floor = 199
	const selfFloor = 93
	const knownHigh = "8483-ladder-wowhunter_1060.CurseoftheAllSpark.xml"
	if len(highNames) > 1 || (len(highNames) == 1 && highNames[0] != knownHigh) {
		t.Errorf("ours-HIGH on %v -- the known remainder is a LOW-side story and this is not it; "+
			"the only recorded over-application is %s", highNames, knownHigh)
	}
	if agreed < floor {
		t.Errorf("agreement is %d of %d carriers, floor is %d; diverged sample: %v", agreed, checked, floor, diverged)
	}
	if selfAgreed < selfFloor {
		t.Errorf("self-arm agreement is %d of %d, floor is %d -- skipEffectiveRate is what makes "+
			"SkillTriggerRate 0 on this arm", selfAgreed, selfArm, selfFloor)
	}
}
