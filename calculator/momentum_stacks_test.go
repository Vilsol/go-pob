package calculator

import (
	"fmt"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMomentumStacksMultiplier: Momentum Support's stack count.
func TestMomentumStacksMultiplier(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)

	var carriers, wrote, clamped, control int
	var firstBad string
	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		as := env.Player.MainSkill
		if as == nil || as.SkillModList == nil {
			continue
		}
		got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:MomentumStacks")
		if !as.SkillModList.Flag(nil, "SupportedByMomentum") {
			if got != 0 {
				control++
				if firstBad == "" {
					firstBad = fmt.Sprintf("%s has Multiplier:MomentumStacks=%v with no Momentum on the main skill", name, got)
				}
			}
			continue
		}
		carriers++
		maxStacks := as.SkillModList.Sum(mod.TypeBase, as.SkillCfg, "MomentumStacksMax")
		if maxStacks <= 0 {
			continue
		}
		if got > 0 {
			wrote++
		}
		if got <= maxStacks {
			clamped++
		}
	}

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
