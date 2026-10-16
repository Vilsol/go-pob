package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestEvalMultiplierThresholdEquals: ModStore.lua:478 — `tag.equals and mult ~= threshold` drops the mod above the threshold too.
func TestEvalMultiplierThresholdEquals(t *testing.T) {
	for _, tc := range []struct {
		gems, want float64
	}{{0, 0}, {1, 6}, {2, 0}} {
		db := NewModDB()
		db.Multipliers["SocketedGemsInHelmet"] = tc.gems
		db.AddMod(mod.NewFloat("GemLevel", mod.TypeBase, 6).Tag(mod.MultiplierThreshold("SocketedGemsInHelmet").Threshold(1).Equals(true)))
		testza.AssertEqual(t, tc.want, db.Sum(mod.TypeBase, nil, "GemLevel"), "gems=%v", tc.gems)
	}
}
