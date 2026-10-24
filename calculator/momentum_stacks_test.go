package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

var _ = registerCorpusCheck("TestMomentumStacksMultiplier", func() corpusCheck { return &momentumStacksCheck{} })

type momentumStacksCheck struct {
	mu                                sync.Mutex
	carriers, wrote, clamped, control int
	firstBad                          corpusFirst
}

func (c *momentumStacksCheck) visit(name string, _ oracleRecord, _ bool, env *Environment) {
	if env == nil {
		return
	}
	as := env.Player.MainSkill
	if as == nil || as.SkillModList == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:MomentumStacks")
	if !as.SkillModList.Flag(nil, "SupportedByMomentum") {
		configured := env.Player.ModDB.Sum(mod.TypeBase, &moddb.ListCfg{Source: utils.Ptr(mod.Source("Config"))}, "Multiplier:MomentumStacks")
		if got-configured != 0 {
			c.control++
			c.firstBad.offer(name, fmt.Sprintf("%s has Multiplier:MomentumStacks=%v with no Momentum on the main skill", name, got))
		}
		return
	}
	c.carriers++
	maxStacks := as.SkillModList.Sum(mod.TypeBase, as.SkillCfg, "MomentumStacksMax")
	if maxStacks <= 0 {
		return
	}
	if got > 0 {
		c.wrote++
	}
	if got <= maxStacks {
		c.clamped++
	}
}

// TestMomentumStacksMultiplier: Momentum Support's stack count.
func TestMomentumStacksMultiplier(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*momentumStacksCheck](t)
	carriers, wrote, clamped, control, firstBad := c.carriers, c.wrote, c.clamped, c.control, c.firstBad.msg

	fmt.Printf("\nMomentum: %d builds have the MAIN skill supported by Momentum; %d got a stack "+
		"multiplier, %d within MomentumStacksMax\n", carriers, wrote, clamped)

	if carriers == 0 {
		t.Fatalf("REFUSING: no corpus build has its MAIN skill supported by Momentum, so this " +
			"fixture reaches nothing and a green result would mean the reader was never exercised")
	}
	if wrote == 0 {
		t.Errorf("Multiplier:MomentumStacks stayed 0 on all %d carriers — the reader is not firing", carriers)
	}
	if clamped != carriers {
		t.Errorf("%d of %d carriers exceed MomentumStacksMax — the clamp at CalcOffence.lua:1009 is missing",
			carriers-clamped, carriers)
	}
	if control > 0 {
		t.Errorf("%d builds got the multiplier with no Momentum on the main skill; first: %s", control, firstBad)
	}
}
