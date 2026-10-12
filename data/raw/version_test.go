package raw

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestGameDataVersionOverride pins POE_VERSION as the way to point the whole process at a different game-data set.
func TestGameDataVersionOverride(t *testing.T) {
	fake := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	testza.AssertEqual(t, DefaultVersion, gameDataVersion(fake(nil)),
		"unset POE_VERSION must resolve to the pinned default — the oracle, every ratchet and "+
			"the whole 3.18 corpus are graded against it, so it can never depend on the environment")

	testza.AssertEqual(t, DefaultVersion, gameDataVersion(fake(map[string]string{"POE_VERSION": ""})),
		"an EMPTY POE_VERSION is unset, not a request for an empty version — an empty string "+
			"reaches the CDN as data.pob.gg/data//raw/... and 404s on every file")

	testza.AssertEqual(t, "3.20", gameDataVersion(fake(map[string]string{"POE_VERSION": "3.20"})),
		"POE_VERSION must select the game-data set")

	testza.AssertEqual(t, "3_20", gameDataVersion(fake(map[string]string{"POE_VERSION": "3_20"})),
		"the value is passed through verbatim — this asserts we do NOT silently rewrite it, so a "+
			"wrong spelling fails loudly at load rather than resolving to something unintended")
}

// TestDefaultVersionIsThePinnedOne guards the constant every published accuracy figure assumes.
func TestDefaultVersionIsThePinnedOne(t *testing.T) {
	testza.AssertEqual(t, "3.29", DefaultVersion,
		"DefaultVersion is the game data an unqualified run loads, and since 2026-08-11 that is "+
			"3.29 — the version the pinned PoB and the committed corpus and oracle all belong "+
			"to. It is the ONLY graded version: 3.18 grading was removed on 2026-08-09 and 3.20 "+
			"on 2026-08-11. The paths are deliberately not spelled out here; they derive from "+
			"this constant, and naming them would be a second place to edit at the next bump.")
}
