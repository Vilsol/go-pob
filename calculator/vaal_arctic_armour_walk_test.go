package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	vaalArcticWalkAgreeFloor    = 76
	vaalArcticControlAgreeFloor = 7948
)

var _ = registerCorpusCheck("TestVaalArcticArmourWalkAgrees", func() corpusCheck { return &vaalArcticArmourWalkCheck{} })

type vaalArcticArmourWalkCheck struct {
	mu                                             sync.Mutex
	carrierSeen, carrierOK, controlSeen, controlOK int
}

func (c *vaalArcticArmourWalkCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	want, has := rec.Output["NumberOfDamagingHits"]
	if !has {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agrees := oracleClose(env.Player.Output["NumberOfDamagingHits"], want, oracleDumpTolerance)
	if env.Player.Output["VaalArcticArmourLife"] > 0 {
		c.carrierSeen++
		if agrees {
			c.carrierOK++
		}
	} else {
		c.controlSeen++
		if agrees {
			c.controlOK++
		}
	}
}

// TestVaalArcticArmourWalkAgrees pins CalcDefence.lua:2627-2630 and :2640-2644.
func TestVaalArcticArmourWalkAgrees(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*vaalArcticArmourWalkCheck](t)
	carrierSeen, carrierOK, controlSeen, controlOK := c.carrierSeen, c.carrierOK, c.controlSeen, c.controlOK

	if carrierSeen == 0 || controlSeen == 0 {
		t.Fatalf("REFUSING: carrier population %d, control population %d — a floor over an empty "+
			"set asserts nothing", carrierSeen, controlSeen)
	}
	fmt.Printf("=== VAAL ARCTIC ARMOUR WALK === carriers %d/%d agree · control %d/%d agree\n",
		carrierOK, carrierSeen, controlOK, controlSeen)

	if carrierOK < vaalArcticWalkAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with VaalArcticArmourLife > 0, "+
			"floor %d — the CalcDefence.lua:2640-2641 multiplier regressed", carrierOK, carrierSeen,
			vaalArcticWalkAgreeFloor)
	}
	if controlOK < vaalArcticControlAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with NO Vaal Arctic Armour, floor "+
			"%d — the multiplier is not inert where it should be", controlOK, controlSeen,
			vaalArcticControlAgreeFloor)
	}
}
