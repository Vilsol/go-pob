package data

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestMapLevelLifeMultIsSparseAndStartsAt66 pins the SHAPE of src/Data/Misc.lua:470.
func TestMapLevelLifeMultIsSparseAndStartsAt66(t *testing.T) {
	testza.AssertEqual(t, 25, len(MapLevelLifeMult),
		"Misc.lua:470 has 25 entries, 66 through 90")

	for level := 66; level <= 90; level++ {
		_, ok := MapLevelLifeMult[level]
		testza.AssertTrue(t, ok, "level %d must be present", level)
	}

	for _, level := range []int{0, 1, 65, 91, 100} {
		_, ok := MapLevelLifeMult[level]
		testza.AssertFalse(t, ok, "level %d must be ABSENT — `or 1` is the reader's fallback", level)
	}

	prev := 1.0
	for level := 66; level <= 90; level++ {
		v := MapLevelLifeMult[level]
		testza.AssertTrue(t, v >= prev, "level %d (%v) must not scale below level %d (%v)",
			level, v, level-1, prev)
		prev = v
	}
	testza.AssertEqual(t, 1.01, MapLevelLifeMult[66])
	testza.AssertEqual(t, 2.66, MapLevelLifeMult[90])
}
