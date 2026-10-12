package calculator

import (
	"os"
	"testing"
)

// TestCancellableReservationIsPublished pins CalcPerform.lua:544-545.
//
//	output[pool.."UncancellableReservation"] = m_min(uncancellableReservation, 0)
//	output[pool.."CancellableReservation"]   = 100 - uncancellableReservation
func TestCancellableReservationIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	for _, pool := range []string{"Life", "Mana"} {
		got, ok := env.Player.Output[pool+"CancellableReservation"]
		if !ok {
			t.Errorf("%sCancellableReservation absent; PoB publishes it for every pool with a "+
				"nonzero maximum", pool)
			continue
		}
		if got != 100 {
			t.Errorf("%sCancellableReservation = %v, want 100 (= 100 - uncancellableReservation, "+
				"which is 0 on this build)", pool, got)
		}
		if unc, ok := env.Player.Output[pool+"UncancellableReservation"]; !ok || unc != 0 {
			t.Errorf("%sUncancellableReservation = %v (present=%v), want 0", pool, unc, ok)
		}
	}
}
