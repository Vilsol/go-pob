package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCrossLinkedSupportReachesSwapSlot pins CalcSetup.lua:1635/:1712-1731: the slot a support group is matched on is `group.slot:gsub(" Swap","")`.
func TestCrossLinkedSupportReachesSwapSlot(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	const name = "1615-juggernaut-kinetic-blast.xml"
	out := corpusEnvForLoadedData(t, name).Player.Output
	for _, stat := range []string{"ForkCountMax", "TotalDPS"} {
		want := oracle[name].Output[stat]
		if want == 0 {
			t.Fatalf("REFUSING: %s has no %s in the oracle", name, stat)
		}
		if got := out[stat]; math.Abs(got-want) > 1e-6*math.Abs(want) {
			t.Errorf("%s %s = %v, PoB %v", name, stat, got, want)
		}
	}
}
