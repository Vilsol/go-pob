package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

var socketSwapSlotBuilds = []string{
	"2006-ladder-kara8991_2821.NerotoxHS_TTV_Lightgirl.xml",
	"10983-ladder-Mike962222_4408.\u4f0a\u8389\u838e\u9ed1.xml",
	"13072-ladder-battlepat_6044.FDGO_Ultra.xml",
	"4185-ladder-Belvince_7684.Nicht_die_Hellste.xml",
	"5575-ladder-Narks__3615.NarksCondemnsSlavery.xml",
}

func TestSocketColourMatchesInWeaponSwapSlots(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawSwap := false
	for _, build := range socketSwapSlotBuilds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, present := rec.Output["GemQuality"]
		if !present || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero GemQuality for %s", build)
		}

		env := corpusEnvForLoadedData(t, build)
		ms := env.Player.MainSkill
		if ms == nil {
			t.Fatalf("REFUSING: %s has no main skill", build)
		}
		if sg, okg := ms.SocketGroup.(*pob.Skill); okg && sg != nil &&
			strings.HasSuffix(sg.Slot, " Swap") {
			sawSwap = true
		}

		if got := env.Player.Output["GemQuality"]; got != want {
			t.Errorf("%s: GemQuality = %v, PoB %v (difference %v; "+
				"data.MatchingSocketQualityBonus is %d)",
				build, got, want, want-got, data.MatchingSocketQualityBonus)
		}
	}

	if !sawSwap {
		t.Fatal("REFUSING: no fixture's main socket group sits in a \" Swap\" slot, so the arm " +
			"this test exists for is never reached")
	}
}
