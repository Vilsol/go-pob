package calculator

import (
	"os"
	"sync"
	"testing"
)

// TestTriggeredSkillWithZeroCastTimeKeepsItsRate: CalcOffence.lua:2205 opens the attack/cast speed ladder with.
//
//	and not skillData.triggered then
func TestTriggeredSkillWithZeroCastTimeKeepsItsRate(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0549-guardian-tempest-shield",
		"0156-assassin-steelskin",
		"0379-luminary-convocation",
		"0578-saboteur-brand-recall",
		"0687-luminary-phase-run",
	} {
		t.Run(build, func(t *testing.T) {
			t.Parallel()

			rec, ok := oracle[build+".xml"]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob — an absent build cannot fail "+
					"this test, so it must not pass it either", build)
			}
			want, has := rec.Output["Speed"]
			if !has || want == 0 {
				t.Fatalf("REFUSING: the oracle carries no nonzero Speed for %s, so this case "+
					"asserts nothing", build)
			}

			env := corpusEnvForLoadedData(t, build)
			if env == nil || env.Player == nil {
				t.Fatalf("REFUSING: %s produced no player environment", build)
			}
			got := env.Player.Output["Speed"]
			if got == 0 {
				t.Errorf("Speed = 0, want %g — the triggered skill took the castTime==0 arm and "+
					"never reached its trigger rate", want)
				return
			}
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("Speed = %g, want %g (ratio %.4f)", got, want, got/want)
			}
		})
	}
}

var _ = registerCorpusCheck("TestSpeedIsNotStarvedToZero", func() corpusCheck { return &speedStarvedCheck{} })

type speedStarvedCheck struct {
	mu              sync.Mutex
	graded, starved int
}

func (c *speedStarvedCheck) visit(_ string, rec oracleRecord, _ bool, env *Environment) {
	want, has := rec.Output["Speed"]
	if !has || want == 0 {
		return
	}
	if env == nil || env.Player == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.graded++
	if env.Player.Output["Speed"] == 0 {
		c.starved++
	}
}

// TestSpeedIsNotStarvedToZero is the sweep behind the five named builds.
func TestSpeedIsNotStarvedToZero(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*speedStarvedCheck](t)
	graded, starved := c.graded, c.starved

	if graded == 0 {
		t.Fatal("REFUSING: no build carries a nonzero Speed in the oracle, so this sweep is not " +
			"reaching the speed ladder at all and its count means nothing")
	}

	t.Logf("Speed: %d graded, %d starved to exactly 0 (%.2f%%)",
		graded, starved, 100*float64(starved)/float64(graded))

	const ceiling = 1
	if starved > ceiling {
		t.Errorf("Speed is starved to exactly 0 on %d of %d builds, ceiling %d — a triggered "+
			"skill reaching the castTime==0 arm loses its rate and every consumer of Speed with "+
			"it", starved, graded, ceiling)
	}
}
