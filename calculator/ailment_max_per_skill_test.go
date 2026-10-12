package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAilmentMaximumIsMaxOverActiveSkills pins CalcPerform.lua:3532-3539.
func TestAilmentMaximumIsMaxOverActiveSkills(t *testing.T) {
	t.Parallel()

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

	const stat = "MaximumShock"
	seen, agreed := 0, 0
	ours := map[float64]bool{}
	theirs := map[float64]bool{}

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
		got, want := env.Player.Output[stat], rec.Output[stat]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: %s is %v", name, stat, got)
		}
		seen++
		ours[math.Round(got*1e6)/1e6] = true
		theirs[math.Round(want*1e6)/1e6] = true
		if math.Abs(got-want) <= 0.005*math.Abs(want) {
			agreed++
		}
	}

	if seen < 1000 {
		t.Fatalf("REFUSING: only %d builds publish %s; the population has moved", seen, stat)
	}
	const agreeFloor = 8490
	if agreed < agreeFloor {
		t.Errorf("%s agreement REGRESSED to %d of %d, floor %d", stat, agreed, seen, agreeFloor)
	}
	const distinctFloor = 10
	if len(ours) < distinctFloor {
		t.Errorf("%s takes only %d distinct values across %d builds, floor %d (PoB takes %d) -- it is pinned again, which is what reading a single skill's mods looks like",
			stat, len(ours), seen, distinctFloor, len(theirs))
	}
	t.Logf("%s: agreed %d of %d, ours %d distinct, PoB %d distinct", stat, agreed, seen, len(ours), len(theirs))
}
