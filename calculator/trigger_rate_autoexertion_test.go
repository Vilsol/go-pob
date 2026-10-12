package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAutoexertionIgnoresTickRateOnBothArms pins CalcTriggers.lua's ["autoexertion"] entry.
func TestAutoexertionIgnoresTickRateOnBothArms(t *testing.T) {
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
	tickRounded := 0
	distinct := map[float64]bool{}
	var diverged []string
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
		if !strings.Contains(string(raw), "Autoexertion") {
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
		isSelf := ms.ActiveEffect.GrantedEffect.Name() == "Autoexertion"
		triggeredBy := ms.TriggeredBy != nil && ms.TriggeredBy.GrantedEffect != nil &&
			ms.TriggeredBy.GrantedEffect.Name() == "Autoexertion"
		if !isSelf && !triggeredBy {
			continue
		}

		checked++
		if isSelf {
			selfArm++
		}
		distinct[math.Round(rec.Output["TriggerRateCap"]*1e6)/1e6] = true

		if got := env.Player.Output["TriggerRateCap"]; got > 0 {
			cd := 1 / got
			if math.Abs(cd/0.033-math.Round(cd/0.033)) < 1e-6 {
				tickRounded++
			}
		}

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

	if checked < 40 {
		t.Fatalf("REFUSING: only %d Autoexertion carriers dispatched; the fixture population has "+
			"moved (60 resolved to this configTable key when written)", checked)
	}
	if selfArm < 25 {
		t.Fatalf("REFUSING: only %d builds take the Autoexertion-is-the-main-skill arm; the arm "+
			"this test exists for is untested (27 when written)", selfArm)
	}
	if len(distinct) < 3 {
		t.Fatalf("REFUSING: only %d distinct caps published; a hardcoded constant would pass this",
			len(distinct))
	}

	t.Logf("autoexertion: %d dispatched (%d self-arm), %d agree (%d self-arm), %d distinct caps, %d tick-rounded",
		checked, selfArm, agreed, selfAgreed, len(distinct), tickRounded)

	if tickRounded > 0 {
		t.Errorf("%d of %d autoexertion carriers publish a tick-rounded cooldown; "+
			"CalcTriggers.lua:1264 sets skillData.ignoresTickRate on BOTH arms", tickRounded, checked)
	}

	const floor = 103
	const selfFloor = 46
	if agreed < floor {
		t.Errorf("agreement is %d of %d carriers, floor is %d; diverged sample: %v", agreed, checked, floor, diverged)
	}
	if selfAgreed < selfFloor {
		t.Errorf("self-arm agreement is %d of %d, floor is %d -- :1264 is outside the if/else",
			selfAgreed, selfArm, selfFloor)
	}
}
