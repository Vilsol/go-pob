package raw

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestCalculatedStatOrderIsDeterministic pins the granted-effect stat index against the order its level rows arrive in.
func TestCalculatedStatOrderIsDeterministic(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	ids := []string{"ExileFireball", "GladiatorBossFireball", "Cyclone", "LightningStrike", "EtherealKnives"}
	varying := 0
	for _, id := range ids {
		raw := poe.GrantedEffectByID(id)
		if raw == nil {
			t.Fatalf("granted effect %s not found -- a fixture has been deleted from the game "+
				"data; re-point it rather than dropping it, or this test stops exercising the "+
				"varying-stat-count shape entirely", id)
		}

		first := (&CalculatedGrantedEffect{GrantedEffect: raw})
		first.calculate()

		for i := 0; i < 60; i++ {
			again := (&CalculatedGrantedEffect{GrantedEffect: raw})
			again.calculate()
			testza.AssertEqual(t, first.calculatedStats, again.calculatedStats, id+": stat order must not depend on level-map iteration order")
		}

		lo, hi := 1<<30, 0
		for _, level := range first.GetCalculatedLevels() {
			if n := len(level.Values); n < lo {
				lo = n
			} else if n > hi {
				hi = n
			}
		}
		if len(first.GetCalculatedLevels()) > 1 && hi > lo {
			varying++
		}
	}
	if varying == 0 {
		t.Fatal("REFUSING: not one fixture has levels declaring different numbers of float stats, " +
			"which is the only shape whose stat index can shift. This test would pass vacuously.")
	}
}
