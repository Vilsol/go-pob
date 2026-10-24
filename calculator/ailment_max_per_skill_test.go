package calculator

import (
	"math"
	"os"
	"sync"
	"testing"
)

const ailmentMaxPerSkillStat = "MaximumShock"

var _ = registerCorpusCheck("TestAilmentMaximumIsMaxOverActiveSkills", func() corpusCheck {
	return &ailmentMaxPerSkillCheck{ours: map[float64]bool{}, theirs: map[float64]bool{}}
})

type ailmentMaxPerSkillCheck struct {
	mu           sync.Mutex
	seen, agreed int
	ours, theirs map[float64]bool
	log          corpusCheckLog
}

func (c *ailmentMaxPerSkillCheck) visit(name string, rec oracleRecord, _ bool, env *Environment) {
	const stat = ailmentMaxPerSkillStat
	if rec.Output[stat] == 0 {
		return
	}
	if env == nil || env.Player == nil {
		return
	}
	got, want := env.Player.Output[stat], rec.Output[stat]
	if math.IsNaN(got) || math.IsInf(got, 0) {
		c.log.fatalf(name, "%s: %s is %v", name, stat, got)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen++
	c.ours[math.Round(got*1e6)/1e6] = true
	c.theirs[math.Round(want*1e6)/1e6] = true
	if math.Abs(got-want) <= 0.005*math.Abs(want) {
		c.agreed++
	}
}

// TestAilmentMaximumIsMaxOverActiveSkills pins CalcPerform.lua:3532-3539.
func TestAilmentMaximumIsMaxOverActiveSkills(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	c := corpusWalkCheck[*ailmentMaxPerSkillCheck](t)
	c.log.report(t)
	const stat = ailmentMaxPerSkillStat
	seen, agreed, ours, theirs := c.seen, c.agreed, c.ours, c.theirs

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
