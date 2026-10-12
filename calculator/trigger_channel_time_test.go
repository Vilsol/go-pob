package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCastWhileChannellingPublishesChannelTime pins CalcTriggers.lua:379's `output.ChannelTimeToTrigger = triggerInterval`.
func TestCastWhileChannellingPublishesChannelTime(t *testing.T) {
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

	const stat = "ChannelTimeToTrigger"
	const tol = 0.005

	checked, agreed := 0, 0
	distinct := map[float64]bool{}
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output[stat] == 0 {
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
		if env == nil || env.Player == nil {
			continue
		}

		checked++
		exp := rec.Output[stat]
		distinct[math.Round(exp*1e6)/1e6] = true
		got := env.Player.Output[stat]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, stat, got)
		}
		if math.Abs(got-exp) <= tol*math.Abs(exp) {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	if checked < 60 {
		t.Fatalf("REFUSING: only %d carriers publish %s; the fixture population has moved", checked, stat)
	}
	if len(distinct) < 3 {
		t.Fatalf("REFUSING: only %d distinct values across %d carriers; the sweep no longer exercises the source's triggerTime", len(distinct), checked)
	}

	const floor = 76
	if agreed < floor {
		shown := diverged
		if len(shown) > 5 {
			shown = shown[:5]
		}
		t.Errorf("agreement REGRESSED: %d of %d carriers agree, floor is %d; %d diverged, first: %v",
			agreed, checked, floor, len(diverged), shown)
	}
}
