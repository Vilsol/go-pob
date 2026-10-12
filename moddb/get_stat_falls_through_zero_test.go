package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

type zeroOutputActor map[string]float64

func (a zeroOutputActor) GetOutput(key string) (float64, bool) {
	v, ok := a[key]
	return v, ok
}

// TestGetStatFallsThroughAZeroOutputToSkillStats: ModStore.lua:316 is one `or` chain.
func TestGetStatFallsThroughAZeroOutputToSkillStats(t *testing.T) {
	db := NewModDB()
	db.Actor = zeroOutputActor{"Life": 0}
	cfg := &ListCfg{SkillStats: map[string]float64{"Life": 500}}

	testza.AssertEqual(t, float64(500), db.getStat("Life", cfg))

	db.Actor = zeroOutputActor{"Life": 7}
	testza.AssertEqual(t, float64(7), db.getStat("Life", cfg))

	db.Actor = zeroOutputActor{"Life": 0}
	testza.AssertEqual(t, float64(0), db.getStat("Life", nil))
}
