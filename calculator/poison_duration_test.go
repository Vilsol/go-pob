package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestPoisonDurationIsPublishedOutsideTheBaseValGuard pins CalcOffence.lua:4584-4588 against :4722.
func TestPoisonDurationIsPublishedOutsideTheBaseValGuard(t *testing.T) {
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

	const agreeFloor = 642
	const starvedCeiling = 11

	carriers, agreed, starved := 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, published := rec.Output["PoisonDuration"]
		if !published || want == 0 {
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
		carriers++
		got := env.Player.Output["PoisonDuration"]
		switch {
		case math.Abs(got-want) <= math.Abs(want)*oracleDumpTolerance:
			agreed++
		case got == 0:
			starved++
		}
	}

	if carriers < 500 {
		t.Fatalf("REFUSING: only %d builds publish PoisonDuration; the population has moved", carriers)
	}
	if agreed < agreeFloor {
		t.Errorf("PoisonDuration agreement is %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	if starved > starvedCeiling {
		t.Errorf("PoisonDuration starved (ours exactly 0) on %d builds, ceiling %d", starved, starvedCeiling)
	}
	t.Logf("PoisonDuration: agreed %d of %d, starved %d", agreed, carriers, starved)
}
